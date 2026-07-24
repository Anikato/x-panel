package credential

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"xpanel/app/model"
	"xpanel/global"
	securityCredentials "xpanel/security/credentials"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestInitializeCreatesKeyAndMigratesLegacyDatabase(t *testing.T) {
	db := openCredentialDatabase(t)
	const sentinel = "SENTINEL_LEGACY_HOST_PASSWORD"
	if err := db.Create(&model.Host{Name: "legacy", Addr: "127.0.0.1", Password: sentinel}).Error; err != nil {
		t.Fatalf("seed legacy host: %v", err)
	}
	root := t.TempDir()
	keyPath := filepath.Join(root, "secrets", "credential-keyring.json")

	manager, err := Initialize(db, keyPath, filepath.Join(root, "recovery"))
	if err != nil {
		t.Fatalf("Initialize legacy database: %v", err)
	}
	if manager == nil || global.CREDENTIALS != manager {
		t.Fatal("credential manager was not installed globally")
	}
	if _, err := os.Stat(keyPath); err != nil {
		t.Fatalf("keyring was not created: %v", err)
	}
	var raw string
	if err := db.Table("hosts").Select("password").Scan(&raw).Error; err != nil {
		t.Fatalf("read migrated password: %v", err)
	}
	if strings.Contains(raw, sentinel) || !manager.IsEncrypted(raw) {
		t.Fatalf("legacy password was not encrypted: %q", raw)
	}
}

func TestInitializeLoadsExistingKeyForEncryptedDatabase(t *testing.T) {
	db := openCredentialDatabase(t)
	root := t.TempDir()
	keyPath := filepath.Join(root, "secrets", "credential-keyring.json")
	original, _, err := securityCredentials.LoadOrCreate(keyPath, true)
	if err != nil {
		t.Fatalf("create original keyring: %v", err)
	}
	value, err := original.Protect("nodes.token", "node-secret")
	if err != nil {
		t.Fatalf("protect node token: %v", err)
	}
	if err := db.Create(&model.Node{Name: "node", Token: value}).Error; err != nil {
		t.Fatalf("seed encrypted node: %v", err)
	}

	loaded, err := Initialize(db, keyPath, filepath.Join(root, "recovery"))
	if err != nil {
		t.Fatalf("Initialize encrypted database: %v", err)
	}
	if loaded.ActiveKeyID() != original.ActiveKeyID() {
		t.Fatalf("active key changed from %q to %q", original.ActiveKeyID(), loaded.ActiveKeyID())
	}
}

func TestInitializeRejectsEncryptedDatabaseWithoutMatchingKey(t *testing.T) {
	source := newCredentialManager(t)
	value, err := source.Protect("nodes.token", "node-secret")
	if err != nil {
		t.Fatalf("protect node token: %v", err)
	}

	t.Run("missing keyring", func(t *testing.T) {
		db := openCredentialDatabase(t)
		if err := db.Create(&model.Node{Name: "node", Token: value}).Error; err != nil {
			t.Fatalf("seed encrypted node: %v", err)
		}
		root := t.TempDir()
		_, err := Initialize(
			db,
			filepath.Join(root, "secrets", "credential-keyring.json"),
			filepath.Join(root, "recovery"),
		)
		if err == nil || !strings.Contains(err.Error(), "does not exist") {
			t.Fatalf("missing keyring error = %v", err)
		}
	})

	t.Run("wrong keyring", func(t *testing.T) {
		db := openCredentialDatabase(t)
		if err := db.Create(&model.Node{Name: "node", Token: value}).Error; err != nil {
			t.Fatalf("seed encrypted node: %v", err)
		}
		root := t.TempDir()
		wrongPath := filepath.Join(root, "secrets", "credential-keyring.json")
		if _, _, err := securityCredentials.LoadOrCreate(wrongPath, true); err != nil {
			t.Fatalf("create wrong keyring: %v", err)
		}
		_, err := Initialize(db, wrongPath, filepath.Join(root, "recovery"))
		if err == nil || !strings.Contains(err.Error(), "unavailable") {
			t.Fatalf("wrong keyring error = %v", err)
		}
	})
}

func TestInstallRootDoesNotPromoteTopLevelDataDirectory(t *testing.T) {
	if got := installRoot("/data"); got != "/data" {
		t.Fatalf("installRoot(/data) = %q, want /data", got)
	}
	if got := installRoot("/srv/xpanel/data"); got != "/srv/xpanel" {
		t.Fatalf("installRoot(/srv/xpanel/data) = %q, want /srv/xpanel", got)
	}
}

func openCredentialDatabase(t *testing.T) *gorm.DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "xpanel.db")
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		t.Fatalf("open credential database: %v", err)
	}
	if err := db.AutoMigrate(
		&model.Setting{},
		&model.Host{},
		&model.Node{},
		&model.BackupAccount{},
		&model.DatabaseServer{},
		&model.DatabaseInstance{},
		&model.AcmeAccount{},
		&model.DnsAccount{},
		&model.Certificate{},
		&model.CertSource{},
		&model.Website{},
		&model.GostService{},
		&model.GostChain{},
		&model.Cronjob{},
		&model.HAProxyConfigVersion{},
	); err != nil {
		t.Fatalf("migrate credential database: %v", err)
	}
	t.Cleanup(func() {
		global.CREDENTIALS = nil
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

func newCredentialManager(t *testing.T) *securityCredentials.Manager {
	t.Helper()
	manager, _, err := securityCredentials.LoadOrCreate(
		filepath.Join(t.TempDir(), "secrets", "credential-keyring.json"),
		true,
	)
	if err != nil {
		t.Fatalf("create credential manager: %v", err)
	}
	return manager
}
