package migration

import (
	"os"
	"path/filepath"
	"testing"

	"xpanel/app/model"
	"xpanel/utils/websitecontent"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestRepairManagedWebsitePermissions(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "test.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Website{}); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	managedRoot := filepath.Join(root, "managed")
	sourceRoot := filepath.Join(root, "source")
	for _, path := range []string{managedRoot, sourceRoot} {
		if err := os.Mkdir(path, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(path, "index.html"), []byte("ok"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	sites := []model.Website{
		{PrimaryDomain: "managed.example", Alias: "managed", Type: "static", ConfigMode: "managed", SiteDir: managedRoot},
		{PrimaryDomain: "source.example", Alias: "source", Type: "static", ConfigMode: "source", SiteDir: sourceRoot},
	}
	if err := db.Create(&sites).Error; err != nil {
		t.Fatal(err)
	}
	policy := websitecontent.Policy{UID: os.Geteuid(), GID: os.Getegid(), DirectoryMode: 0o750, FileMode: 0o640}
	if err := repairManagedWebsitePermissions(db, policy); err != nil {
		t.Fatal(err)
	}
	assertMode := func(path string, want os.FileMode) {
		t.Helper()
		info, err := os.Stat(path)
		if err != nil || info.Mode().Perm() != want {
			t.Fatalf("path=%s mode=%v want=%v err=%v", path, info.Mode().Perm(), want, err)
		}
	}
	assertMode(managedRoot, policy.DirectoryMode)
	assertMode(filepath.Join(managedRoot, "index.html"), policy.FileMode)
	assertMode(sourceRoot, 0o700)
	assertMode(filepath.Join(sourceRoot, "index.html"), 0o600)
}
