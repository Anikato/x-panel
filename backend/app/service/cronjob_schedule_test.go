package service

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"xpanel/app/dto"
	"xpanel/app/model"
	"xpanel/app/repo"
	"xpanel/constant"
	"xpanel/global"

	"github.com/glebarez/sqlite"
	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
)

func setupCronjobTest(t *testing.T) *CronjobService {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "cronjob.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Cronjob{}, &model.CronjobRecord{}); err != nil {
		t.Fatal(err)
	}
	previousDB, previousCron := global.DB, global.CRON
	global.DB = db
	scheduler := cron.New()
	scheduler.Start()
	global.CRON = scheduler
	resetCronLiveEntries()
	t.Cleanup(func() {
		scheduler.Stop()
		resetCronLiveEntries()
		global.DB = previousDB
		global.CRON = previousCron
	})
	return NewICronjobService().(*CronjobService)
}

func countCronEntries(t *testing.T) int {
	t.Helper()
	return len(global.CRON.Entries())
}

func TestUpdateInvalidSpecKeepsOriginalSchedule(t *testing.T) {
	svc := setupCronjobTest(t)
	if err := svc.Create(dto.CronjobCreate{
		Name:   "backup",
		Type:   "shell",
		Spec:   "@every 1h",
		Script: "echo ok",
	}); err != nil {
		t.Fatal(err)
	}
	if countCronEntries(t) != 1 {
		t.Fatalf("entries = %d, want 1", countCronEntries(t))
	}

	jobs, err := svc.cronjobRepo.List()
	if err != nil || len(jobs) != 1 {
		t.Fatalf("jobs = %d err=%v", len(jobs), err)
	}
	err = svc.Update(dto.CronjobUpdate{
		ID:     jobs[0].ID,
		Name:   "backup",
		Type:   "shell",
		Spec:   "not a cron spec",
		Script: "echo ok",
	})
	if err == nil {
		t.Fatal("expected invalid spec error")
	}
	if countCronEntries(t) != 1 {
		t.Fatalf("schedule lost: entries = %d", countCronEntries(t))
	}
	stored, getErr := svc.cronjobRepo.Get(jobs[0].ID)
	if getErr != nil {
		t.Fatal(getErr)
	}
	if stored.Spec != "@every 1h" {
		t.Fatalf("spec = %q, want original", stored.Spec)
	}

	before := time.Now()
	time.Sleep(20 * time.Millisecond)
	if countCronEntries(t) != 1 {
		t.Fatal("schedule disappeared after failed update")
	}
	_ = before
}

func TestEnableTwiceDoesNotDuplicateSchedule(t *testing.T) {
	svc := setupCronjobTest(t)
	if err := svc.Create(dto.CronjobCreate{
		Name:   "once",
		Type:   "shell",
		Spec:   "@every 2h",
		Script: "true",
	}); err != nil {
		t.Fatal(err)
	}
	jobs, _ := svc.cronjobRepo.List()
	if err := svc.UpdateStatus(jobs[0].ID, constant.StatusEnable); err != nil {
		t.Fatal(err)
	}
	if countCronEntries(t) != 1 {
		t.Fatalf("entries = %d, want 1 after second enable", countCronEntries(t))
	}
}

type cronRepoHook struct {
	repo.ICronjobRepo
	failFields bool
	failEntry  bool
}

func (c cronRepoHook) Update(id uint, fields map[string]interface{}) error {
	if _, ok := fields["entry_id"]; ok {
		if c.failEntry {
			return fmt.Errorf("persist entry id")
		}
	} else if c.failFields {
		return fmt.Errorf("persist fields")
	}
	return c.ICronjobRepo.Update(id, fields)
}

func TestUpdateDoesNotRegisterDisabledJobWhenDatabaseWriteFails(t *testing.T) {
	svc := setupCronjobTest(t)
	if err := svc.Create(dto.CronjobCreate{Name: "off", Type: "shell", Spec: "@every 3h", Script: "true"}); err != nil {
		t.Fatal(err)
	}
	jobs, _ := svc.cronjobRepo.List()
	if err := svc.UpdateStatus(jobs[0].ID, constant.StatusDisable); err != nil {
		t.Fatal(err)
	}
	if countCronEntries(t) != 0 {
		t.Fatalf("disabled job still scheduled: %d", countCronEntries(t))
	}
	svc.cronjobRepo = cronRepoHook{ICronjobRepo: svc.cronjobRepo, failFields: true}
	err := svc.Update(dto.CronjobUpdate{
		ID: jobs[0].ID, Name: "off", Type: "shell", Spec: "@every 4h", Script: "true",
	})
	if err == nil {
		t.Fatal("expected database write failure")
	}
	if countCronEntries(t) != 0 {
		t.Fatalf("disabled job was registered after failed update: %d", countCronEntries(t))
	}
}

func TestStatusPersistFailureRestoresScheduler(t *testing.T) {
	svc := setupCronjobTest(t)
	if err := svc.Create(dto.CronjobCreate{Name: "job", Type: "shell", Spec: "@every 1h", Script: "true"}); err != nil {
		t.Fatal(err)
	}
	if countCronEntries(t) != 1 {
		t.Fatalf("entries = %d, want 1", countCronEntries(t))
	}
	jobs, err := svc.cronjobRepo.List()
	if err != nil || len(jobs) != 1 {
		t.Fatalf("jobs = %d err=%v", len(jobs), err)
	}
	base := svc.cronjobRepo
	svc.cronjobRepo = cronRepoHook{ICronjobRepo: base, failFields: true}
	if err := svc.UpdateStatus(jobs[0].ID, constant.StatusDisable); err == nil {
		t.Fatal("expected status persist failure")
	}
	if countCronEntries(t) != 1 {
		t.Fatalf("disable persist failure left %d schedules", countCronEntries(t))
	}
	stored, err := base.Get(jobs[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Status != constant.StatusEnable {
		t.Fatalf("status = %q, want enable", stored.Status)
	}

	svc.cronjobRepo = base
	if err := svc.UpdateStatus(jobs[0].ID, constant.StatusDisable); err != nil {
		t.Fatal(err)
	}
	svc.cronjobRepo = cronRepoHook{ICronjobRepo: base, failFields: true}
	if err := svc.UpdateStatus(jobs[0].ID, constant.StatusEnable); err == nil {
		t.Fatal("expected enable persist failure")
	}
	if countCronEntries(t) != 0 {
		t.Fatalf("enable persist failure left %d schedules", countCronEntries(t))
	}
	stored, err = base.Get(jobs[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Status != constant.StatusDisable {
		t.Fatalf("status = %q, want disable", stored.Status)
	}
}

func TestAddCronJobRemovesScheduleWhenEntryPersistFails(t *testing.T) {
	svc := setupCronjobTest(t)
	if err := svc.Create(dto.CronjobCreate{Name: "on", Type: "shell", Spec: "@every 5h", Script: "true"}); err != nil {
		t.Fatal(err)
	}
	if countCronEntries(t) != 1 {
		t.Fatalf("entries = %d", countCronEntries(t))
	}
	jobs, _ := svc.cronjobRepo.List()
	baseRepo := svc.cronjobRepo
	svc.cronjobRepo = cronRepoHook{ICronjobRepo: baseRepo, failEntry: true}
	err := svc.Update(dto.CronjobUpdate{
		ID: jobs[0].ID, Name: "on", Type: "shell", Spec: "@every 6h", Script: "true",
	})
	if err == nil {
		t.Fatal("expected entry persist failure")
	}
	if countCronEntries(t) != 1 {
		t.Fatalf("schedule count = %d, want original restored without leftover", countCronEntries(t))
	}
	svc.cronjobRepo = baseRepo
	if err := svc.UpdateStatus(jobs[0].ID, constant.StatusDisable); err != nil {
		t.Fatal(err)
	}
	if countCronEntries(t) != 0 {
		t.Fatalf("disable after restore left %d schedules", countCronEntries(t))
	}
	if err := svc.UpdateStatus(jobs[0].ID, constant.StatusEnable); err != nil {
		t.Fatal(err)
	}
	if countCronEntries(t) != 1 {
		t.Fatalf("enable after restore left %d schedules", countCronEntries(t))
	}
}
