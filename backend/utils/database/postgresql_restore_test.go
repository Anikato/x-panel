package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPostgresCustomRestoreUsesSingleTransaction(t *testing.T) {
	args := postgresCustomRestoreArgs("127.0.0.1", 5432, "postgres", "appdb", "/tmp/restore.dump")
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "--single-transaction") {
		t.Fatalf("custom restore must wrap in a transaction so a failed restore does not keep partial objects: %v", args)
	}
	if !strings.Contains(joined, "--exit-on-error") && !strings.Contains(joined, " --single-transaction") {
		t.Fatalf("custom restore args = %v", args)
	}
}

func TestPostgresSQLRestoreEnablesOnErrorStop(t *testing.T) {
	args := postgresSQLRestoreArgs("127.0.0.1", 5432, "postgres", "appdb", "/tmp/restore.sql")
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "ON_ERROR_STOP=1") {
		t.Fatalf("psql args missing ON_ERROR_STOP: %v", args)
	}
	if strings.Contains(joined, " -1 ") || strings.HasSuffix(joined, " -1") {
		t.Fatalf("must not force a single transaction: %v", args)
	}
}

func TestPostgresSQLRestoreFailsOnSQLError(t *testing.T) {
	if os.Getenv("XPANEL_PG_INTEGRATION") == "" {
		t.Skip("set XPANEL_PG_INTEGRATION=1 with a throwaway Postgres")
	}

	host := getenvDefault("PGHOST", "127.0.0.1")
	user := getenvDefault("PGUSER", "postgres")
	pass := getenvDefault("PGPASSWORD", "xpanel-it")
	port := uint(5432)
	if raw := os.Getenv("PGPORT"); raw != "" {
		var parsed uint
		if _, err := fmt.Sscanf(raw, "%d", &parsed); err == nil {
			port = parsed
		}
	}

	admin, err := NewPostgresClient(host, port, user, pass)
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	defer admin.Close()

	dbName := fmt.Sprintf("xpanel_restore_it_%d", time.Now().UnixNano())
	if err := admin.CreateDatabase(dbName, user); err != nil {
		t.Fatalf("create database: %v", err)
	}
	defer func() { _ = admin.DeleteDatabase(dbName) }()

	dir := t.TempDir()
	goodFile := filepath.Join(dir, "good.sql")
	badFile := filepath.Join(dir, "bad.sql")
	if err := os.WriteFile(goodFile, []byte("CREATE TABLE restore_probe(id int);\nINSERT INTO restore_probe VALUES (1);\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(badFile, []byte(`CREATE TABLE restore_probe(id int);
INSERT INTO restore_probe VALUES (1);
SELECT * FROM does_not_exist;
INSERT INTO restore_probe VALUES (2);
`), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := admin.Restore(dbName, goodFile); err != nil {
		t.Fatalf("good SQL restore failed: %v", err)
	}
	if got := queryRestoreProbeMax(t, host, port, user, pass, dbName); got != 1 {
		t.Fatalf("good restore max(id)=%d, want 1", got)
	}

	if err := admin.DeleteDatabase(dbName); err != nil {
		t.Fatal(err)
	}
	if err := admin.CreateDatabase(dbName, user); err != nil {
		t.Fatal(err)
	}

	err = admin.Restore(dbName, badFile)
	if err == nil {
		t.Fatal("restore with SQL error returned success")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "does_not_exist") &&
		!strings.Contains(strings.ToLower(err.Error()), "error") {
		t.Fatalf("restore error missing SQL diagnostics: %v", err)
	}
	if got := queryRestoreProbeMax(t, host, port, user, pass, dbName); got == 2 {
		t.Fatalf("partial import was reported as complete; max(id)=2")
	}
}

func queryRestoreProbeMax(t *testing.T, host string, port uint, user, pass, dbName string) int {
	t.Helper()
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", host, port, user, pass, dbName)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var maxID int
	err = db.QueryRow("SELECT COALESCE(MAX(id), 0) FROM restore_probe").Scan(&maxID)
	if err != nil {
		return 0
	}
	return maxID
}

func getenvDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
