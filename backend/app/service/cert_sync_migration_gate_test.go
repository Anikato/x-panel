package service

import (
	"testing"

	"xpanel/app/model"
	"xpanel/global"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestCertificateSyncMigrationGateFailsClosedUntilMarkerExists(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	previous := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = previous })
	if err := db.AutoMigrate(&model.Setting{}); err != nil {
		t.Fatal(err)
	}

	if err := certificateSyncMigrationReady(); err == nil {
		t.Fatal("missing certificate migration marker must block synchronization")
	}
	if err := db.Create(&model.Setting{Key: model.CertificateLineageMigrationKey, Value: "done"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := certificateSyncMigrationReady(); err != nil {
		t.Fatalf("expected marker to permit synchronization: %v", err)
	}
}
