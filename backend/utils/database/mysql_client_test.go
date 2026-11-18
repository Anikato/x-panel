package database

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBackupAndRestoreProtectClientPassword(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("TMPDIR", tmp)
	t.Setenv("PATH", tmp+":"+os.Getenv("PATH"))
	t.Setenv("MYSQL_PWD", "")
	// Inspect only synthetic credentials, while the client still owns the options file.
	const script = `#!/bin/sh
set -eu
case "$1" in --defaults-extra-file=*) file=${1#*=} ;; *) exit 12 ;; esac
printf '%s\n' "$@" > "$TMPDIR/args"
test -z "$MYSQL_PWD"
test "$(stat -c %a "$file")" = 600
cp "$file" "$TMPDIR/captured"
test -z "$FAIL_CLIENT"
`
	for _, name := range []string{"mysql", "mysqldump"} {
		if err := os.WriteFile(filepath.Join(tmp, name), []byte(script), 0700); err != nil {
			t.Fatal(err)
		}
	}
	const password = "a #;\\s\"x\nr\r\tt"
	client := MysqlClient{Address: "localhost", Port: 3307, Username: "backup", Password: password}
	sql := filepath.Join(tmp, "fixture.sql")
	if err := os.WriteFile(sql, []byte("SELECT 1;"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		run  func() error
		fail string
	}{
		{"backup", func() error { return client.Backup("app", filepath.Join(tmp, "out.sql")) }, ""},
		{"restore failure", func() error { return client.Restore("app", sql) }, "1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("FAIL_CLIENT", tc.fail)
			err := tc.run()
			if (err != nil) != (tc.fail != "") {
				t.Fatalf("client error = %v", err)
			}
			data, err := os.ReadFile(filepath.Join(tmp, "captured"))
			if err != nil {
				t.Fatal(err)
			}
			if string(data) != "[client]\npassword=\"a #;\\\\s\\\"x\\nr\\r\\tt\"\n" {
				t.Fatal("incorrect password encoding")
			}
			args, err := os.ReadFile(filepath.Join(tmp, "args"))
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(args), password) || strings.Contains(string(args), "\n-p") || !strings.Contains(string(args), "--protocol=TCP") {
				t.Fatal("unsafe or incorrect arguments")
			}
			file := strings.TrimPrefix(strings.SplitN(string(args), "\n", 2)[0], "--defaults-extra-file=")
			if _, err := os.Stat(file); !os.IsNotExist(err) {
				t.Fatalf("credential file remains: %v", err)
			}
		})
	}
}
