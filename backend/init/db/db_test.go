package db

import (
	"path/filepath"
	"testing"

	"gorm.io/gorm/logger"
)

func TestOpenSQLiteSetsBusyTimeoutWALAndSingleConnection(t *testing.T) {
	database, err := openSQLite(filepath.Join(t.TempDir(), "panel.db"), logger.Silent)
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := database.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	if sqlDB.Stats().MaxOpenConnections != 1 {
		t.Fatalf("max open connections = %d, want 1", sqlDB.Stats().MaxOpenConnections)
	}
	var timeout int
	if err := database.Raw("PRAGMA busy_timeout").Scan(&timeout).Error; err != nil {
		t.Fatal(err)
	}
	if timeout != 5000 {
		t.Fatalf("busy_timeout = %d, want 5000", timeout)
	}
	var mode string
	if err := database.Raw("PRAGMA journal_mode").Scan(&mode).Error; err != nil {
		t.Fatal(err)
	}
	if mode != "wal" {
		t.Fatalf("journal_mode = %q, want wal", mode)
	}
}
