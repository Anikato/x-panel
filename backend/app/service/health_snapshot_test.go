package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"xpanel/app/model"
	"xpanel/constant"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func openHealthSnapshotDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&model.Website{},
		&model.Certificate{},
		&model.BackupRecord{},
		&model.Cronjob{},
		&model.CronjobRecord{},
	); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestBuildHealthSnapshotSummarizesWithoutSecrets(t *testing.T) {
	db := openHealthSnapshotDB(t)
	now := time.Date(2026, 10, 29, 8, 0, 0, 0, time.UTC)
	secret := "SENSITIVE-HEALTH-SNAPSHOT-SENTINEL"

	mustCreate := func(value any) {
		t.Helper()
		if err := db.Create(value).Error; err != nil {
			t.Fatal(err)
		}
	}
	mustCreate(&model.Website{PrimaryDomain: "running.example", Alias: "running", Status: "running", SSLEnable: true, BasicPassword: secret, CustomNginx: secret})
	mustCreate(&model.Website{PrimaryDomain: "stopped.example", Alias: "stopped", Status: "stopped"})
	mustCreate(&model.Certificate{PrimaryDomain: "expired.example", Provider: "dns", Status: "ready", ExpireDate: now.Add(-time.Hour), Pem: secret, PrivateKey: secret})
	mustCreate(&model.Certificate{PrimaryDomain: "soon.example", Provider: "dns", Status: "ready", ExpireDate: now.Add(6 * 24 * time.Hour), Pem: secret, PrivateKey: secret})
	mustCreate(&model.Certificate{PrimaryDomain: "later.example", Provider: "dns", Status: "error", Message: secret, ExpireDate: now.Add(12 * 24 * time.Hour)})
	mustCreate(&model.Certificate{PrimaryDomain: "zero.example", Provider: "dns", Status: "ready"})

	mustCreate(&model.BackupRecord{BaseModel: model.BaseModel{CreatedAt: now.Add(-2 * time.Hour)}, Type: "website", Name: "ok-site", Status: constant.StatusSuccess, Message: secret})
	mustCreate(&model.BackupRecord{BaseModel: model.BaseModel{CreatedAt: now.Add(-time.Hour)}, Type: "directory", Name: "failed-dir", Status: constant.StatusFailed, Message: secret})

	job := model.Cronjob{Name: "nightly", Type: "shell", Spec: "0 0 * * *", Status: constant.StatusEnable, Script: secret, EncryptPassword: secret}
	mustCreate(&job)
	mustCreate(&model.CronjobRecord{BaseModel: model.BaseModel{CreatedAt: now.Add(-30 * time.Minute)}, CronjobID: job.ID, Status: constant.StatusFailed, Message: secret})
	mustCreate(&model.CronjobRecord{BaseModel: model.BaseModel{CreatedAt: now.Add(-time.Hour)}, CronjobID: job.ID, Status: constant.StatusFailed, Message: secret})
	mustCreate(&model.CronjobRecord{BaseModel: model.BaseModel{CreatedAt: now.Add(-2 * time.Hour)}, CronjobID: job.ID, Status: constant.StatusSuccess, Message: secret})

	data, err := BuildHealthSnapshot(db, now)
	if err != nil {
		t.Fatal(err)
	}
	if data.SchemaVersion != 1 || !data.Panel.DatabaseReadable {
		t.Fatalf("panel=%+v schema=%d", data.Panel, data.SchemaVersion)
	}
	if data.Websites.Total != 2 || data.Websites.Running != 1 || data.Websites.Stopped != 1 || data.Websites.SSLEnabled != 1 {
		t.Fatalf("websites=%+v", data.Websites)
	}
	if data.Certificates.Total != 4 || data.Certificates.Expired != 1 || data.Certificates.Expiring7Days != 1 || data.Certificates.Expiring15Days != 2 || data.Certificates.Error != 1 {
		t.Fatalf("certificates=%+v", data.Certificates)
	}
	if data.Backups.Success48Hours != 1 || data.Backups.Failed48Hours != 1 || len(data.Backups.RecentFailures) != 1 {
		t.Fatalf("backups=%+v", data.Backups)
	}
	if data.Backups.LatestSuccessAt == nil || data.Backups.LatestFailureAt == nil {
		t.Fatalf("backup times=%+v", data.Backups)
	}
	if data.Cronjobs.Total != 1 || data.Cronjobs.Enabled != 1 || data.Cronjobs.Failed24Hours != 2 || len(data.Cronjobs.ConsecutiveFailure) != 1 {
		t.Fatalf("cronjobs=%+v", data.Cronjobs)
	}
	if got := data.Cronjobs.ConsecutiveFailure[0]; got.ID != job.ID || got.Name != "nightly" || got.ConsecutiveFailures != 2 {
		t.Fatalf("failure=%+v", got)
	}

	raw, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{secret, "BEGIN CERTIFICATE", "PRIVATE KEY", "encrypt_password", "basic_password", "custom_nginx", "message"} {
		if bytes.Contains(raw, []byte(forbidden)) {
			t.Fatalf("snapshot leaked %q: %s", forbidden, raw)
		}
	}
}

func TestBuildHealthSnapshotLimitsFailuresAndRecords(t *testing.T) {
	db := openHealthSnapshotDB(t)
	now := time.Date(2026, 10, 29, 8, 0, 0, 0, time.UTC)
	for index := 0; index < HealthSnapshotFailureLimit+1; index++ {
		if err := db.Create(&model.BackupRecord{
			BaseModel: model.BaseModel{CreatedAt: now.Add(-time.Duration(index) * time.Minute)},
			Type:      "directory",
			Name:      fmt.Sprintf("failed-%d", index),
			Status:    constant.StatusFailed,
		}).Error; err != nil {
			t.Fatal(err)
		}
	}
	job := model.Cronjob{Name: "always-fails", Type: "shell", Spec: "* * * * *", Status: constant.StatusEnable}
	if err := db.Create(&job).Error; err != nil {
		t.Fatal(err)
	}
	records := make([]model.CronjobRecord, HealthSnapshotRecordScanLimit+1)
	for index := range records {
		records[index] = model.CronjobRecord{
			BaseModel: model.BaseModel{CreatedAt: now.Add(-time.Duration(index) * time.Second)},
			CronjobID: job.ID,
			Status:    constant.StatusFailed,
		}
	}
	if err := db.CreateInBatches(records, 100).Error; err != nil {
		t.Fatal(err)
	}

	data, err := BuildHealthSnapshot(db, now)
	if err != nil {
		t.Fatal(err)
	}
	if !data.Backups.Truncated || len(data.Backups.RecentFailures) != HealthSnapshotFailureLimit {
		t.Fatalf("backups=%+v", data.Backups)
	}
	if !data.Cronjobs.RecordsTruncated || len(data.Cronjobs.ConsecutiveFailure) != 1 || data.Cronjobs.ConsecutiveFailure[0].ConsecutiveFailures != HealthSnapshotRecordScanLimit {
		t.Fatalf("cronjobs=%+v", data.Cronjobs)
	}
}
