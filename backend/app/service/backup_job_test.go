package service

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestBackupDirectoryExcludesConfiguredPatterns(t *testing.T) {
	src := t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "keep.txt"), []byte("keep"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "debug.log"), []byte("log"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(src, "cache"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "cache", "x"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	outDir := t.TempDir()
	svc := &BackupService{}
	localFile, _, err := svc.backupDirectory(src, "app", outDir, "20260101120000", BackupJobOptions{
		ExclusionRules: "*.log\ncache",
	})
	if err != nil {
		t.Fatal(err)
	}

	list := tarList(t, localFile)
	if !strings.Contains(list, "keep.txt") {
		t.Fatalf("archive missing keep.txt: %s", list)
	}
	if strings.Contains(list, "debug.log") {
		t.Fatalf("archive should exclude *.log: %s", list)
	}
	if strings.Contains(list, "cache/") || strings.Contains(list, "/cache/") {
		t.Fatalf("archive should exclude cache dir: %s", list)
	}
}

func TestFinalizeUploadedBackupDeletesLocalWhenRequested(t *testing.T) {
	localFile := filepath.Join(t.TempDir(), "db.sql")
	if err := os.WriteFile(localFile, []byte("sql"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := finalizeUploadedBackup(localFile, "sftp", true); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(localFile); !os.IsNotExist(err) {
		t.Fatalf("local backup should be deleted, stat err=%v", err)
	}
}

func TestFinalizeUploadedBackupKeepsLocalWhenDisabled(t *testing.T) {
	localFile := filepath.Join(t.TempDir(), "db.sql")
	if err := os.WriteFile(localFile, []byte("sql"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := finalizeUploadedBackup(localFile, "sftp", false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(localFile); err != nil {
		t.Fatalf("local backup should remain: %v", err)
	}
}

func TestFinalizeUploadedBackupNeverDeletesLocalAccountCopy(t *testing.T) {
	localFile := filepath.Join(t.TempDir(), "db.sql")
	if err := os.WriteFile(localFile, []byte("sql"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := finalizeUploadedBackup(localFile, "local", true); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(localFile); err != nil {
		t.Fatalf("local account backup is the destination and must remain: %v", err)
	}
}

func tarList(t *testing.T, archive string) string {
	t.Helper()
	cmd := exec.Command("tar", "-tf", archive)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("tar -tf %s: %v\n%s", archive, err, out)
	}
	return string(out)
}
