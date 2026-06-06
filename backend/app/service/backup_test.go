package service

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"xpanel/global"
)

func TestBackupTempDirUsesConfiguredDataDir(t *testing.T) {
	original := global.CONF.System.DataDir
	t.Cleanup(func() {
		global.CONF.System.DataDir = original
	})

	dataDir := t.TempDir()
	global.CONF.System.DataDir = dataDir

	got := backupTempDir()
	want := filepath.Join(dataDir, "tmp", "xpanel-backup")

	if got != want {
		t.Fatalf("backupTempDir() = %q, want %q", got, want)
	}
}

func TestEstimateDirectorySize(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("12345"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "nested"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "nested", "b.txt"), []byte("123"), 0644); err != nil {
		t.Fatal(err)
	}

	size, err := estimateDirectorySize(dir)
	if err != nil {
		t.Fatal(err)
	}
	if size != 8 {
		t.Fatalf("estimateDirectorySize() = %d, want 8", size)
	}
}

func TestCleanBackupTempDirRemovesOldArchivesOnly(t *testing.T) {
	original := global.CONF.System.DataDir
	t.Cleanup(func() {
		global.CONF.System.DataDir = original
	})

	dataDir := t.TempDir()
	global.CONF.System.DataDir = dataDir
	tmpDir := backupTempDir()
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		t.Fatal(err)
	}

	oldFile := filepath.Join(tmpDir, "old.tar.gz")
	newFile := filepath.Join(tmpDir, "new.tar.gz")
	otherFile := filepath.Join(tmpDir, "note.txt")
	for _, file := range []string{oldFile, newFile, otherFile} {
		if err := os.WriteFile(file, []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	oldTime := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(oldFile, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}

	CleanBackupTempDir(24 * time.Hour)

	if _, err := os.Stat(oldFile); !os.IsNotExist(err) {
		t.Fatalf("old archive should be removed, stat err: %v", err)
	}
	if _, err := os.Stat(newFile); err != nil {
		t.Fatalf("new archive should remain: %v", err)
	}
	if _, err := os.Stat(otherFile); err != nil {
		t.Fatalf("non archive should remain: %v", err)
	}
}
