package main

import (
	"fmt"
	"path/filepath"
	"testing"

	"xpanel/security/credentials"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestParseCredentialVerifyArgs(t *testing.T) {
	path, err := parseCredentialVerifyArgs([]string{"--db", "/srv/backup.db"})
	if err != nil {
		t.Fatalf("parse credential verify args: %v", err)
	}
	if path != "/srv/backup.db" {
		t.Fatalf("database path = %q", path)
	}
	for _, args := range [][]string{
		nil,
		{"--db", ""},
		{"--db", "/srv/backup.db", "extra"},
	} {
		if _, err := parseCredentialVerifyArgs(args); err == nil {
			t.Fatalf("args %#v should fail", args)
		}
	}
}

func TestVerifyCredentialDatabaseAcceptsMatchingKeyring(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "candidate.db")
	keyPath := filepath.Join(t.TempDir(), "secrets", "credential-keyring.json")
	db := openCredentialVerifyFixture(t, dbPath)
	manager, _, err := credentials.LoadOrCreate(keyPath, true)
	if err != nil {
		t.Fatalf("create matching keyring: %v", err)
	}
	protected, err := manager.Protect("settings.AgentToken", "agent-secret")
	if err != nil {
		t.Fatalf("protect candidate credential: %v", err)
	}
	if err := db.Exec(`INSERT INTO settings (key, value) VALUES ('AgentToken', ?)`, protected).Error; err != nil {
		t.Fatalf("seed candidate credential: %v", err)
	}
	closeCredentialVerifyFixture(t, db)

	if err := verifyCredentialDatabase(dbPath, keyPath); err != nil {
		t.Fatalf("verify matching candidate: %v", err)
	}
}

func TestVerifyCredentialDatabaseRejectsWrongKeyring(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "candidate.db")
	db := openCredentialVerifyFixture(t, dbPath)
	sourceManager, _, err := credentials.LoadOrCreate(
		filepath.Join(t.TempDir(), "source", "credential-keyring.json"),
		true,
	)
	if err != nil {
		t.Fatalf("create source keyring: %v", err)
	}
	protected, err := sourceManager.Protect("settings.AgentToken", "agent-secret")
	if err != nil {
		t.Fatalf("protect candidate credential: %v", err)
	}
	if err := db.Exec(`INSERT INTO settings (key, value) VALUES ('AgentToken', ?)`, protected).Error; err != nil {
		t.Fatalf("seed candidate credential: %v", err)
	}
	closeCredentialVerifyFixture(t, db)

	wrongKeyPath := filepath.Join(t.TempDir(), "wrong", "credential-keyring.json")
	if _, _, err := credentials.LoadOrCreate(wrongKeyPath, true); err != nil {
		t.Fatalf("create wrong keyring: %v", err)
	}
	if err := verifyCredentialDatabase(dbPath, wrongKeyPath); err == nil {
		t.Fatalf("wrong keyring should be rejected")
	}
}

func openCredentialVerifyFixture(t *testing.T, path string) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		t.Fatalf("open credential fixture: %v", err)
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
	return db
}

func closeCredentialVerifyFixture(t *testing.T, db *gorm.DB) {
	t.Helper()
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get fixture connection: %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close fixture database: %v", err)
	}
}
