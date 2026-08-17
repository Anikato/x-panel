package service

import (
	"errors"
	"testing"

	"xpanel/app/dto"
	"xpanel/app/model"
	"xpanel/global"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func installNotificationDB(t *testing.T) {
	t.Helper()
	database, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.AutoMigrate(&model.Setting{}, &model.Notification{}); err != nil {
		t.Fatal(err)
	}
	previous := global.DB
	global.DB = database
	t.Cleanup(func() { global.DB = previous })
}

func TestCreateNotificationUsesLowercaseEventPreference(t *testing.T) {
	installNotificationDB(t)
	svc := NewINotificationService()
	if err := svc.Create(dto.NotificationCreate{
		Type:   "error",
		Event:  "cronjob.Failed",
		Title:  "job failed",
		Source: "cronjob",
	}); err != nil {
		t.Fatal(err)
	}
	items, err := svc.Recent(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("items=%d", len(items))
	}
	if items[0].Event != "cronjob.failed" {
		t.Fatalf("event=%q", items[0].Event)
	}
	if !items[0].Popup || !items[0].ShowBadge {
		t.Fatalf("popup=%v badge=%v", items[0].Popup, items[0].ShowBadge)
	}
}

func TestPreferenceKeepsAllFalseDefaults(t *testing.T) {
	installNotificationDB(t)
	svc := NewINotificationService()
	if err := svc.UpdatePreference(dto.NotificationPreference{
		Defaults: dto.NotificationPreferenceRule{},
		Events: map[string]dto.NotificationPreferenceRule{
			"file.upload.completed": {Center: true},
		},
	}); err != nil {
		t.Fatal(err)
	}
	got, err := svc.GetPreference()
	if err != nil {
		t.Fatal(err)
	}
	if got.Defaults.Center || got.Defaults.Badge || got.Defaults.Popup {
		t.Fatalf("defaults=%+v", got.Defaults)
	}
}

func TestDefaultNotificationCatalog(t *testing.T) {
	pref := defaultNotificationPreference()
	if _, ok := pref.Events["operation.failed"]; ok {
		t.Fatal("operation.failed must not be a default event")
	}
	if _, ok := pref.Events["system.log.error"]; ok {
		t.Fatal("system.log.error must not be a default event")
	}
	for _, event := range []string{
		"cronjob.failed", "cronjob.success", "ssl.renew.failed", "security.login.failed",
		"file.task.success", "database.task.success",
	} {
		if _, ok := pref.Events[event]; !ok {
			t.Fatalf("missing %s", event)
		}
	}
	success := pref.Events["cronjob.success"]
	if !success.Center || success.Badge || success.Popup {
		t.Fatalf("cronjob.success=%+v", success)
	}
}

func TestNotifyJobResultWritesLowercaseEvents(t *testing.T) {
	installNotificationDB(t)
	svc := &CronjobService{}
	svc.notifyJobResult(&model.Cronjob{Name: "backup"}, "Failed", "disk full")
	svc.notifyJobResult(&model.Cronjob{Name: "backup"}, "Success", "ok")

	items, err := NewINotificationService().Recent(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("items=%d", len(items))
	}
	got := map[string]dto.NotificationInfo{}
	for _, item := range items {
		got[item.Event] = item
	}
	failed, ok := got["cronjob.failed"]
	if !ok || !failed.Popup {
		t.Fatalf("failed=%+v", failed)
	}
	success, ok := got["cronjob.success"]
	if !ok || success.ShowBadge || success.Popup {
		t.Fatalf("success=%+v", success)
	}
}

func TestSaveLoginLogCreatesSecurityNotification(t *testing.T) {
	installNotificationDB(t)
	if err := global.DB.AutoMigrate(&model.LoginLog{}); err != nil {
		t.Fatal(err)
	}
	SaveLoginLog("1.2.3.4", "test-agent", errors.New("invalid password"))
	items, err := NewINotificationService().Recent(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("items=%d", len(items))
	}
	if items[0].Event != "security.login.failed" || items[0].Source != "security" {
		t.Fatalf("item=%+v", items[0])
	}
	if items[0].TargetURL != "/log/login" || !items[0].Popup {
		t.Fatalf("item=%+v", items[0])
	}
}

func TestNotifySSLRenewFailed(t *testing.T) {
	installNotificationDB(t)
	notifySSLRenewFailed("a.example.com", errors.New("acme timeout"))
	items, err := NewINotificationService().Recent(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("items=%d", len(items))
	}
	if items[0].Event != "ssl.renew.failed" || items[0].TargetURL != "/website/ssl" || !items[0].Popup {
		t.Fatalf("item=%+v", items[0])
	}
}
