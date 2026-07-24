package service

import (
	"errors"
	"fmt"
	"path/filepath"
	"testing"

	"xpanel/global"
	"xpanel/security/credentials"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestRotateCredentialKeyReencryptsDatabaseAndRetainsOldKey(t *testing.T) {
	db, manager := openCredentialRotationDatabase(t)
	oldID := manager.ActiveKeyID()
	seedCredentialSetting(t, db, manager, "AgentToken", "agent-secret")

	result, err := newCredentialSecurityService(db, manager, credentials.ReencryptDatabase).
		RotateCredentialKey()
	if err != nil {
		t.Fatalf("rotate credential key: %v", err)
	}
	if result.KeyID == "" || result.KeyID == oldID {
		t.Fatalf("new key ID = %q, old = %q", result.KeyID, oldID)
	}
	if err := credentials.ValidateDatabase(db, manager); err != nil {
		t.Fatalf("validate rotated database: %v", err)
	}
	state, err := credentials.ScanDatabase(db)
	if err != nil {
		t.Fatalf("scan rotated database: %v", err)
	}
	if len(state.KeyIDs) != 1 {
		t.Fatalf("rotated database key IDs = %#v", state.KeyIDs)
	}
	if _, ok := state.KeyIDs[result.KeyID]; !ok {
		t.Fatalf("rotated database does not use active key %q: %#v", result.KeyID, state.KeyIDs)
	}
	if !containsString(manager.KeyIDs(), oldID) {
		t.Fatalf("old key %q was removed: %#v", oldID, manager.KeyIDs())
	}
}

func TestRotateCredentialKeyFailureKeepsOldCiphertextDecryptableAndRetryable(t *testing.T) {
	db, manager := openCredentialRotationDatabase(t)
	oldID := manager.ActiveKeyID()
	seedCredentialSetting(t, db, manager, "AgentToken", "agent-secret")

	service := newCredentialSecurityService(
		db,
		manager,
		func(*gorm.DB, global.CredentialProtector) error {
			return errors.New("injected re-encryption failure")
		},
	)
	if _, err := service.RotateCredentialKey(); err == nil {
		t.Fatalf("injected rotation failure should be returned")
	}
	firstNewID := manager.ActiveKeyID()
	if firstNewID == oldID {
		t.Fatalf("failed rotation did not persist a recoverable new key")
	}
	if !containsString(manager.KeyIDs(), oldID) || !containsString(manager.KeyIDs(), firstNewID) {
		t.Fatalf("failed rotation did not retain both keys: %#v", manager.KeyIDs())
	}
	if err := credentials.ValidateDatabase(db, manager); err != nil {
		t.Fatalf("old ciphertext became unreadable: %v", err)
	}

	service.reencrypt = credentials.ReencryptDatabase
	result, err := service.RotateCredentialKey()
	if err != nil {
		t.Fatalf("retry credential rotation: %v", err)
	}
	if result.KeyID == firstNewID {
		t.Fatalf("retry did not activate a fresh key")
	}
	if err := credentials.ValidateDatabase(db, manager); err != nil {
		t.Fatalf("validate retried rotation: %v", err)
	}
}

func openCredentialRotationDatabase(t *testing.T) (*gorm.DB, *credentials.Manager) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "rotation.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open rotation database: %v", err)
	}
	columnsByTable := make(map[string][]string)
	for _, spec := range credentials.FieldSpecs {
		columnsByTable[spec.Table] = append(columnsByTable[spec.Table], spec.Column)
	}
	for table, columns := range columnsByTable {
		statement := fmt.Sprintf(`CREATE TABLE "%s" (id INTEGER PRIMARY KEY`, table)
		for _, column := range columns {
			statement += fmt.Sprintf(`, "%s" TEXT`, column)
		}
		statement += ")"
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("create table %s: %v", table, err)
		}
	}
	if err := db.Exec(`CREATE TABLE settings (id INTEGER PRIMARY KEY, key TEXT UNIQUE, value TEXT)`).Error; err != nil {
		t.Fatalf("create settings table: %v", err)
	}
	manager, _, err := credentials.LoadOrCreate(
		filepath.Join(t.TempDir(), "secrets", "credential-keyring.json"),
		true,
	)
	if err != nil {
		t.Fatalf("create rotation keyring: %v", err)
	}
	return db, manager
}

func seedCredentialSetting(
	t *testing.T,
	db *gorm.DB,
	manager *credentials.Manager,
	key string,
	value string,
) {
	t.Helper()
	protected, err := manager.Protect(credentials.SettingScope(key), value)
	if err != nil {
		t.Fatalf("protect seed credential: %v", err)
	}
	if err := db.Exec(`INSERT INTO settings (key, value) VALUES (?, ?)`, key, protected).Error; err != nil {
		t.Fatalf("seed credential setting: %v", err)
	}
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
