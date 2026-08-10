package model

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestCertSourceManualSyncIntervalPersistsAsZero(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	if err := db.AutoMigrate(&CertSource{}); err != nil {
		t.Fatalf("migrate cert source: %v", err)
	}

	source := CertSource{
		Name:         "manual",
		ServerAddr:   "https://primary.example.test",
		Token:        "token",
		SyncInterval: 0,
		Enabled:      true,
	}
	if err := db.Create(&source).Error; err != nil {
		t.Fatalf("create cert source: %v", err)
	}

	var stored CertSource
	if err := db.First(&stored, source.ID).Error; err != nil {
		t.Fatalf("read cert source: %v", err)
	}
	if stored.SyncInterval != 0 {
		t.Fatalf("manual sync interval = %d, want 0", stored.SyncInterval)
	}
}
