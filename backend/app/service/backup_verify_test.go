package service

import (
	"os"
	"path/filepath"
	"testing"

	"xpanel/app/model"
	"xpanel/utils/checksum"
)

func TestVerifyBackupRecordRejectsTamperingBeforeRestore(t *testing.T) {
	path := filepath.Join(t.TempDir(), "copy.sql")
	if err := os.WriteFile(path, []byte("dump"), 0600); err != nil {
		t.Fatal(err)
	}
	hash, err := checksum.FileSHA256(path)
	if err != nil {
		t.Fatal(err)
	}
	record := &model.BackupRecord{Size: 4, SHA256: hash}
	if err := verifyBackupRecordFile(path, record); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("junk"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := verifyBackupRecordFile(path, record); err == nil {
		t.Fatal("same-size tampered backup passed verification")
	}
	if err := verifyBackupRecordFile(path, &model.BackupRecord{Size: 4}); err != nil {
		t.Fatalf("legacy backup record should still work: %v", err)
	}
}
