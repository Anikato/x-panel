package credentials

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"xpanel/global"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestRegistryContainsApprovedCredentialFields(t *testing.T) {
	expected := []string{
		"acme_accounts.eab_hmac_key",
		"acme_accounts.private_key",
		"backup_accounts.access_key",
		"backup_accounts.credential",
		"cert_sources.token",
		"certificates.private_key",
		"cronjobs.encrypt_password",
		"database_instances.password",
		"database_servers.password",
		"dns_accounts.authorization",
		"gost_chains.hops",
		"gost_services.auth_pass",
		"ha_proxy_config_versions.content",
		"hosts.pass_phrase",
		"hosts.password",
		"hosts.private_key",
		"nodes.ssh_password",
		"nodes.token",
		"websites.basic_password",
	}
	actual := make([]string, 0, len(FieldSpecs))
	for _, spec := range FieldSpecs {
		actual = append(actual, spec.Table+"."+spec.Column)
	}
	sort.Strings(actual)
	if strings.Join(actual, "\n") != strings.Join(expected, "\n") {
		t.Fatalf("registered fields:\n%s\nwant:\n%s", strings.Join(actual, "\n"), strings.Join(expected, "\n"))
	}

	for _, key := range []string{
		"MFASecret", "GitHubToken", "AgentToken", "CertServerToken",
		"FleetInstanceToken", "FleetEnrollmentToken", "HAProxyStatsPass",
		"GostAPIPass", "ProxyAddress", "SecurityEntrance",
	} {
		if !IsSecretSetting(key) {
			t.Errorf("setting %q is not registered as secret", key)
		}
	}
	for _, key := range []string{"Password", "PanelName", "FleetEndpoint"} {
		if IsSecretSetting(key) {
			t.Errorf("setting %q must not be registered as reversible secret", key)
		}
	}
}

func TestDatabaseMigrationEncryptsLegacyValuesAndRemovesSnapshot(t *testing.T) {
	db, dbPath := openCredentialFixture(t)
	sentinels := seedLegacyCredentials(t, db)
	manager := newTestManager(t)
	backupDir := filepath.Join(t.TempDir(), "migration-backups")

	state, err := ScanDatabase(db)
	if err != nil {
		t.Fatalf("scan legacy database: %v", err)
	}
	if !state.HasPlaintext || state.HasEncrypted {
		t.Fatalf("legacy state = %+v, want plaintext only", state)
	}

	if err := MigrateDatabase(db, manager, backupDir); err != nil {
		t.Fatalf("migrate legacy database: %v", err)
	}
	state, err = ScanDatabase(db)
	if err != nil {
		t.Fatalf("scan migrated database: %v", err)
	}
	if state.HasPlaintext || !state.HasEncrypted {
		t.Fatalf("migrated state = %+v, want encrypted only", state)
	}
	if err := ValidateDatabase(db, manager); err != nil {
		t.Fatalf("validate migrated database: %v", err)
	}
	assertCredentialValues(t, db, manager, sentinels)
	assertDirectoryEmpty(t, backupDir)

	if err := MigrateDatabase(db, manager, backupDir); err != nil {
		t.Fatalf("second migration must be idempotent: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql database: %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close database: %v", err)
	}
	databaseBytes, err := os.ReadFile(dbPath)
	if err != nil {
		t.Fatalf("read migrated database: %v", err)
	}
	for _, sentinel := range sentinels {
		if strings.Contains(string(databaseBytes), sentinel) {
			t.Fatalf("database file still contains plaintext sentinel %q", sentinel)
		}
	}
}

func TestDatabaseMigrationHandlesMixedState(t *testing.T) {
	db, _ := openCredentialFixture(t)
	manager := newTestManager(t)
	encrypted, err := manager.Protect("hosts.password", "already-encrypted")
	if err != nil {
		t.Fatalf("protect existing value: %v", err)
	}
	if err := db.Exec(
		`INSERT INTO hosts (password, private_key, pass_phrase) VALUES (?, ?, '')`,
		encrypted,
		"legacy-private-key",
	).Error; err != nil {
		t.Fatalf("seed mixed state: %v", err)
	}

	state, err := ScanDatabase(db)
	if err != nil {
		t.Fatalf("scan mixed database: %v", err)
	}
	if !state.HasPlaintext || !state.HasEncrypted {
		t.Fatalf("mixed state = %+v", state)
	}
	if err := MigrateDatabase(db, manager, filepath.Join(t.TempDir(), "backups")); err != nil {
		t.Fatalf("migrate mixed database: %v", err)
	}

	var row struct {
		Password   string
		PrivateKey string
	}
	if err := db.Table("hosts").Select("password, private_key").First(&row).Error; err != nil {
		t.Fatalf("read migrated host: %v", err)
	}
	if row.Password != encrypted {
		t.Fatal("existing envelope using the active KEK must not be re-encrypted")
	}
	privateKey, err := manager.Reveal("hosts.private_key", row.PrivateKey)
	if err != nil {
		t.Fatalf("reveal migrated private key: %v", err)
	}
	if privateKey != "legacy-private-key" {
		t.Fatalf("private key = %q", privateKey)
	}
}

func TestDatabaseValidationRejectsMissingKeyAndTampering(t *testing.T) {
	sourceManager := newTestManager(t)
	value, err := sourceManager.Protect("nodes.token", "node-secret")
	if err != nil {
		t.Fatalf("protect node token: %v", err)
	}

	t.Run("missing key", func(t *testing.T) {
		db, _ := openCredentialFixture(t)
		if err := db.Exec(`INSERT INTO nodes (token, ssh_password) VALUES (?, '')`, value).Error; err != nil {
			t.Fatalf("insert encrypted token: %v", err)
		}
		if err := ValidateDatabase(db, newTestManager(t)); err == nil {
			t.Fatal("validation with a different keyring must fail")
		}
	})

	t.Run("tampered envelope", func(t *testing.T) {
		db, _ := openCredentialFixture(t)
		parts := strings.Split(value, ":")
		ciphertext, err := base64.RawURLEncoding.DecodeString(parts[5])
		if err != nil {
			t.Fatalf("decode ciphertext for tampering: %v", err)
		}
		ciphertext[len(ciphertext)-1] ^= 0x01
		parts[5] = base64.RawURLEncoding.EncodeToString(ciphertext)
		tampered := strings.Join(parts, ":")
		if err := db.Exec(`INSERT INTO nodes (token, ssh_password) VALUES (?, '')`, tampered).Error; err != nil {
			t.Fatalf("insert tampered token: %v", err)
		}
		if err := ValidateDatabase(db, sourceManager); err == nil {
			t.Fatal("validation of tampered ciphertext must fail")
		}
	})
}

func TestDatabaseMigrationRollsBackAndRetainsSnapshotOnFailure(t *testing.T) {
	db, _ := openCredentialFixture(t)
	manager := newTestManager(t)
	backupDir := filepath.Join(t.TempDir(), "migration-backups")
	if err := db.Exec(
		`INSERT INTO hosts (password, private_key, pass_phrase) VALUES (?, ?, '')`,
		"legacy-password",
		"xpanel:enc:v1:missing:bad:bad",
	).Error; err != nil {
		t.Fatalf("seed invalid mixed state: %v", err)
	}

	if err := MigrateDatabase(db, manager, backupDir); err == nil {
		t.Fatal("migration with malformed ciphertext must fail")
	}
	var password string
	if err := db.Table("hosts").Select("password").Scan(&password).Error; err != nil {
		t.Fatalf("read rolled back password: %v", err)
	}
	if password != "legacy-password" {
		t.Fatalf("password after rollback = %q, want legacy-password", password)
	}
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		t.Fatalf("read backup directory: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("retained snapshots = %d, want 1", len(entries))
	}
	info, err := entries[0].Info()
	if err != nil {
		t.Fatalf("stat retained snapshot: %v", err)
	}
	if got := info.Mode().Perm(); got != 0600 {
		t.Fatalf("retained snapshot mode = %04o, want 0600", got)
	}
}

func TestDatabaseMigrationRollsBackWhenPostTransformValidationFails(t *testing.T) {
	db, _ := openCredentialFixture(t)
	manager := newTestManager(t)
	if err := db.Exec(
		`INSERT INTO hosts (password, private_key, pass_phrase) VALUES (?, '', '')`,
		"legacy-password",
	).Error; err != nil {
		t.Fatalf("seed legacy credential: %v", err)
	}

	protector := invalidProtectResult{CredentialProtector: manager}
	if err := MigrateDatabase(db, protector, filepath.Join(t.TempDir(), "backups")); err == nil {
		t.Fatalf("invalid transformed envelope should fail validation")
	}
	var password string
	if err := db.Table("hosts").Select("password").Scan(&password).Error; err != nil {
		t.Fatalf("read credential after failed validation: %v", err)
	}
	if password != "legacy-password" {
		t.Fatalf("failed validation committed transformed value: %q", password)
	}
}

func TestDatabaseMigrationRetriesScrubBeforeMarkingAndRemovesRetainedSnapshot(t *testing.T) {
	db, _ := openCredentialFixture(t)
	manager := newTestManager(t)
	backupDir := filepath.Join(t.TempDir(), "migration-backups")
	if err := db.Exec(
		`INSERT INTO hosts (password, private_key, pass_phrase) VALUES (?, '', '')`,
		"legacy-password",
	).Error; err != nil {
		t.Fatalf("seed legacy credential: %v", err)
	}

	scrubErr := fmt.Errorf("injected scrub failure")
	if err := migrateDatabase(db, manager, backupDir, func(*gorm.DB) error {
		return scrubErr
	}); err == nil || !strings.Contains(err.Error(), scrubErr.Error()) {
		t.Fatalf("scrub failure error = %v", err)
	}
	marked, err := migrationMarked(db)
	if err != nil {
		t.Fatalf("read marker after scrub failure: %v", err)
	}
	if marked {
		t.Fatal("migration was marked complete before SQLite scrub succeeded")
	}
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		t.Fatalf("read retained snapshot directory: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("retained snapshots after scrub failure = %d, want 1", len(entries))
	}

	if err := MigrateDatabase(db, manager, backupDir); err != nil {
		t.Fatalf("retry migration after scrub failure: %v", err)
	}
	marked, err = migrationMarked(db)
	if err != nil {
		t.Fatalf("read marker after retry: %v", err)
	}
	if !marked {
		t.Fatal("successful retry did not write migration marker")
	}
	assertDirectoryEmpty(t, backupDir)
}

func TestDatabaseReencryptionRollsBackWhenValidationFails(t *testing.T) {
	db, _ := openCredentialFixture(t)
	manager := newTestManager(t)
	original, err := manager.Protect("hosts.password", "host-password")
	if err != nil {
		t.Fatalf("protect original credential: %v", err)
	}
	if err := db.Exec(
		`INSERT INTO hosts (password, private_key, pass_phrase) VALUES (?, '', '')`,
		original,
	).Error; err != nil {
		t.Fatalf("seed encrypted credential: %v", err)
	}

	protector := invalidProtectResult{CredentialProtector: manager}
	if err := ReencryptDatabase(db, protector); err == nil {
		t.Fatalf("invalid re-encrypted envelope should fail validation")
	}
	var password string
	if err := db.Table("hosts").Select("password").Scan(&password).Error; err != nil {
		t.Fatalf("read credential after failed re-encryption: %v", err)
	}
	if password != original {
		t.Fatalf("failed re-encryption validation committed transformed value: %q", password)
	}
}

func TestDatabaseMigrationCompletesInterruptedKeyRotationOnRestart(t *testing.T) {
	db, _ := openCredentialFixture(t)
	manager := newTestManager(t)
	original, err := manager.Protect("hosts.password", "host-password")
	if err != nil {
		t.Fatalf("protect original credential: %v", err)
	}
	if err := db.Exec(
		`INSERT INTO hosts (password, private_key, pass_phrase) VALUES (?, '', '')`,
		original,
	).Error; err != nil {
		t.Fatalf("seed encrypted credential: %v", err)
	}
	if err := setMigrationMarker(db); err != nil {
		t.Fatalf("seed migration marker: %v", err)
	}
	oldKeyID := manager.ActiveKeyID()
	if _, err := manager.AddActiveKey(); err != nil {
		t.Fatalf("activate replacement key: %v", err)
	}

	protector := invalidProtectResult{CredentialProtector: manager}
	if err := ReencryptDatabase(db, protector); err == nil {
		t.Fatal("injected re-encryption failure should be returned")
	}
	marked, err := migrationMarked(db)
	if err != nil {
		t.Fatalf("read marker after interrupted rotation: %v", err)
	}
	if marked {
		t.Fatal("interrupted rotation must leave startup recovery marker clear")
	}

	if err := MigrateDatabase(db, manager, filepath.Join(t.TempDir(), "backups")); err != nil {
		t.Fatalf("restart recovery migration: %v", err)
	}
	state, err := ScanDatabase(db)
	if err != nil {
		t.Fatalf("scan recovered database: %v", err)
	}
	if _, exists := state.KeyIDs[oldKeyID]; exists {
		t.Fatalf("restart recovery retained inactive key %q: %#v", oldKeyID, state.KeyIDs)
	}
	if _, exists := state.KeyIDs[manager.ActiveKeyID()]; !exists {
		t.Fatalf("restart recovery did not use active key %q: %#v", manager.ActiveKeyID(), state.KeyIDs)
	}
}

type invalidProtectResult struct {
	global.CredentialProtector
}

func (invalidProtectResult) Protect(string, string) (string, error) {
	return "xpanel:enc:v1:invalid:invalid:invalid", nil
}

func openCredentialFixture(t *testing.T) (*gorm.DB, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "xpanel.db")
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		t.Fatalf("open fixture database: %v", err)
	}

	columnsByTable := make(map[string][]string)
	for _, spec := range FieldSpecs {
		columnsByTable[spec.Table] = append(columnsByTable[spec.Table], spec.Column)
	}
	for table, columns := range columnsByTable {
		definitions := make([]string, 0, len(columns)+1)
		definitions = append(definitions, "id INTEGER PRIMARY KEY AUTOINCREMENT")
		for _, column := range columns {
			definitions = append(definitions, fmt.Sprintf("%s TEXT", quoteTestIdentifier(column)))
		}
		query := fmt.Sprintf(
			"CREATE TABLE %s (%s)",
			quoteTestIdentifier(table),
			strings.Join(definitions, ", "),
		)
		if err := db.Exec(query).Error; err != nil {
			t.Fatalf("create table %s: %v", table, err)
		}
	}
	if err := db.Exec(
		`CREATE TABLE settings (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			key TEXT NOT NULL UNIQUE,
			value TEXT
		)`,
	).Error; err != nil {
		t.Fatalf("create settings table: %v", err)
	}
	return db, path
}

func seedLegacyCredentials(t *testing.T, db *gorm.DB) map[string]string {
	t.Helper()
	sentinels := make(map[string]string)
	columnsByTable := make(map[string][]FieldSpec)
	for _, spec := range FieldSpecs {
		columnsByTable[spec.Table] = append(columnsByTable[spec.Table], spec)
	}
	for table, specs := range columnsByTable {
		columns := make([]string, 0, len(specs))
		placeholders := make([]string, 0, len(specs))
		values := make([]any, 0, len(specs))
		for _, spec := range specs {
			sentinel := "SENTINEL_" + strings.ToUpper(strings.ReplaceAll(spec.Scope, ".", "_"))
			sentinels[spec.Scope] = sentinel
			columns = append(columns, quoteTestIdentifier(spec.Column))
			placeholders = append(placeholders, "?")
			values = append(values, sentinel)
		}
		query := fmt.Sprintf(
			"INSERT INTO %s (%s) VALUES (%s)",
			quoteTestIdentifier(table),
			strings.Join(columns, ", "),
			strings.Join(placeholders, ", "),
		)
		if err := db.Exec(query, values...).Error; err != nil {
			t.Fatalf("seed table %s: %v", table, err)
		}
	}
	for _, key := range SecretSettingKeyList() {
		sentinel := "SENTINEL_SETTINGS_" + strings.ToUpper(key)
		sentinels[SettingScope(key)] = sentinel
		if err := db.Exec(`INSERT INTO settings (key, value) VALUES (?, ?)`, key, sentinel).Error; err != nil {
			t.Fatalf("seed setting %s: %v", key, err)
		}
	}
	for key, value := range map[string]string{
		"Password":  "$2a$10$already-hashed",
		"PanelName": "X-Panel",
	} {
		if err := db.Exec(`INSERT INTO settings (key, value) VALUES (?, ?)`, key, value).Error; err != nil {
			t.Fatalf("seed ordinary setting %s: %v", key, err)
		}
	}
	return sentinels
}

func assertCredentialValues(t *testing.T, db *gorm.DB, manager *Manager, sentinels map[string]string) {
	t.Helper()
	for _, spec := range FieldSpecs {
		var value string
		query := fmt.Sprintf("SELECT %s FROM %s LIMIT 1", quoteTestIdentifier(spec.Column), quoteTestIdentifier(spec.Table))
		if err := db.Raw(query).Scan(&value).Error; err != nil {
			t.Fatalf("read %s: %v", spec.Scope, err)
		}
		if strings.Contains(value, sentinels[spec.Scope]) || !manager.IsEncrypted(value) {
			t.Fatalf("%s was not encrypted: %q", spec.Scope, value)
		}
		plaintext, err := manager.Reveal(spec.Scope, value)
		if err != nil {
			t.Fatalf("reveal %s: %v", spec.Scope, err)
		}
		if plaintext != sentinels[spec.Scope] {
			t.Fatalf("%s plaintext = %q, want %q", spec.Scope, plaintext, sentinels[spec.Scope])
		}
	}
	for _, key := range SecretSettingKeyList() {
		var value string
		if err := db.Raw(`SELECT value FROM settings WHERE key = ?`, key).Scan(&value).Error; err != nil {
			t.Fatalf("read setting %s: %v", key, err)
		}
		scope := SettingScope(key)
		plaintext, err := manager.Reveal(scope, value)
		if err != nil {
			t.Fatalf("reveal setting %s: %v", key, err)
		}
		if plaintext != sentinels[scope] {
			t.Fatalf("setting %s plaintext = %q, want %q", key, plaintext, sentinels[scope])
		}
	}
	var ordinary []struct {
		Key   string
		Value string
	}
	if err := db.Raw(`SELECT key, value FROM settings WHERE key IN ('Password', 'PanelName') ORDER BY key`).Scan(&ordinary).Error; err != nil {
		t.Fatalf("read ordinary settings: %v", err)
	}
	for _, setting := range ordinary {
		if manager.IsEncrypted(setting.Value) {
			t.Fatalf("ordinary setting %s was encrypted", setting.Key)
		}
	}
}

func newTestManager(t *testing.T) *Manager {
	t.Helper()
	manager, _, err := LoadOrCreate(filepath.Join(t.TempDir(), "secrets", "credential-keyring.json"), true)
	if err != nil {
		t.Fatalf("create test keyring: %v", err)
	}
	return manager
}

func assertDirectoryEmpty(t *testing.T, path string) {
	t.Helper()
	entries, err := os.ReadDir(path)
	if err != nil {
		t.Fatalf("read directory %s: %v", path, err)
	}
	if len(entries) != 0 {
		t.Fatalf("directory %s contains %d entries, want 0", path, len(entries))
	}
}

func quoteTestIdentifier(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}
