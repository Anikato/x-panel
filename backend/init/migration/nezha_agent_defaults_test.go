package migration

import (
	"path/filepath"
	"testing"

	"xpanel/app/model"
	"xpanel/global"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestNezhaAgentDefaultSettingsInitIdempotent(t *testing.T) {
	db := openNezhaAgentDefaultsDB(t)

	initDefaultSettings()

	assertRawSettingValue(t, db, "NezhaEnabled", "false")
	assertRawSettingValue(t, db, "NezhaServer", "")
	assertRawSettingValue(t, db, "NezhaClientSecret", "")

	// Existing user values must survive a second init (create-if-missing only).
	if err := db.Model(&model.Setting{}).Where("`key` = ?", "NezhaEnabled").Update("value", "true").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.Setting{}).Where("`key` = ?", "NezhaServer").Update("value", "dashboard.example.com:443").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.Setting{}).Where("`key` = ?", "NezhaClientSecret").Update("value", "user-preserved").Error; err != nil {
		t.Fatal(err)
	}

	initDefaultSettings()

	assertRawSettingValue(t, db, "NezhaEnabled", "true")
	assertRawSettingValue(t, db, "NezhaServer", "dashboard.example.com:443")
	assertRawSettingValue(t, db, "NezhaClientSecret", "user-preserved")
}

func openNezhaAgentDefaultsDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "nezha-defaults.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Setting{}); err != nil {
		t.Fatal(err)
	}
	previousDB := global.DB
	global.DB = db
	t.Cleanup(func() {
		global.DB = previousDB
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}
