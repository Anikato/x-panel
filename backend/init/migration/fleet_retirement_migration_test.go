package migration

import (
	"path/filepath"
	"testing"

	"xpanel/app/model"
	"xpanel/global"
	"xpanel/security/credentials"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// Every known Fleet setting key from the retirement design, plus one unexpected
// future key so the migration must use a prefix delete rather than a fixed list.
var fleetRetirementKnownKeys = []string{
	"FleetEnabled",
	"FleetEndpoint",
	"FleetInstanceID",
	"FleetEnrollmentToken",
	"FleetRecoveryToken",
	"FleetEnrollmentMode",
	"FleetNodePrivateSeed",
	"FleetPendingRecoverySeed",
	"FleetNodeCertificate",
	"FleetNodeCAPublicKey",
	"FleetManualRecoveryRequired",
	"FleetHeartbeatIntervalSeconds",
	"FleetInstanceToken",
	"FleetLegacyInstanceToken",
	"FleetLegacyTokenExpiresAt",
	"FleetProtocolMode",
	"FleetAutoUpgrade",
	"FleetAutoUpgradeReleaseURL",
	"FleetTaskPollIntervalSeconds",
	"FleetUnexpectedKey",
}

func TestFleetRetirementMigrationDeletesEveryFleetSettingIdempotently(t *testing.T) {
	db := openFleetRetirementMigrationDB(t)
	manager := global.CREDENTIALS

	githubToken, err := manager.Protect(credentials.SettingScope("GitHubToken"), "ghp_preserved-secret")
	if err != nil {
		t.Fatal(err)
	}
	nezhaSecret, err := manager.Protect(credentials.SettingScope("NezhaClientSecret"), "nezha-preserved-secret")
	if err != nil {
		t.Fatal(err)
	}

	// Seed every known Fleet key plus an unexpected future Fleet* key.
	for _, key := range fleetRetirementKnownKeys {
		value := "retire-" + key
		if err := db.Create(&model.Setting{Key: key, Value: value}).Error; err != nil {
			t.Fatal(err)
		}
	}

	// Non-Fleet settings that must survive raw deletion of Fleet* rows.
	retained := map[string]string{
		"NezhaEnabled":      "true",
		"NezhaServer":       "dashboard.example.com:443",
		"NezhaClientSecret": nezhaSecret,
		"GitHubToken":       githubToken,
		"PanelName":         "keep-me",
		"Language":          "zh",
		"AutoUpgrade":       "disable",
	}
	for key, value := range retained {
		if err := db.Create(&model.Setting{Key: key, Value: value}).Error; err != nil {
			t.Fatal(err)
		}
	}

	if err := migrateFleetRetirementSettings(); err != nil {
		t.Fatal(err)
	}

	assertFleetPrefixCount(t, db, 0)
	assertRawSettingValue(t, db, fleetRetirementMigrationKey, "done")
	for key, want := range retained {
		assertRawSettingValue(t, db, key, want)
	}

	// Second run must succeed and leave retained raw values untouched.
	if err := db.Model(&model.Setting{}).
		Where("`key` = ?", "PanelName").
		Update("value", "still-must-survive").Error; err != nil {
		t.Fatal(err)
	}
	retained["PanelName"] = "still-must-survive"

	if err := migrateFleetRetirementSettings(); err != nil {
		t.Fatal(err)
	}

	assertFleetPrefixCount(t, db, 0)
	assertRawSettingValue(t, db, fleetRetirementMigrationKey, "done")
	for key, want := range retained {
		assertRawSettingValue(t, db, key, want)
	}
}

func TestFleetRetirementDefaultSettingsOmitFleetKeys(t *testing.T) {
	db := openFleetRetirementMigrationDB(t)

	initDefaultSettings()

	assertFleetPrefixCount(t, db, 0)
	assertRawSettingValue(t, db, "NezhaEnabled", "false")
	assertRawSettingValue(t, db, "NezhaServer", "")
	assertRawSettingValue(t, db, "NezhaClientSecret", "")
}

func openFleetRetirementMigrationDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "fleet-retirement.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Setting{}); err != nil {
		t.Fatal(err)
	}
	manager, _, err := credentials.LoadOrCreate(filepath.Join(t.TempDir(), "secrets", "credential-keyring.json"), true)
	if err != nil {
		t.Fatal(err)
	}
	previousDB, previousCredentials := global.DB, global.CREDENTIALS
	global.DB, global.CREDENTIALS = db, manager
	t.Cleanup(func() {
		global.DB, global.CREDENTIALS = previousDB, previousCredentials
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

func assertFleetPrefixCount(t *testing.T, db *gorm.DB, want int64) {
	t.Helper()
	var count int64
	if err := db.Model(&model.Setting{}).Where("`key` LIKE ?", "Fleet%").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != want {
		t.Fatalf("Fleet%% setting count = %d, want %d", count, want)
	}
}

func assertRawSettingValue(t *testing.T, db *gorm.DB, key, want string) {
	t.Helper()
	var setting model.Setting
	if err := db.Where("`key` = ?", key).First(&setting).Error; err != nil {
		t.Fatalf("read %s: %v", key, err)
	}
	if setting.Value != want {
		t.Fatalf("%s = %q, want %q", key, setting.Value, want)
	}
}
