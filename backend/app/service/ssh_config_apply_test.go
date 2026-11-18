package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSSHDConfigApplyAndRollback(t *testing.T) {
	for _, tc := range []struct {
		mode, want, wantError string
		restarts              int
	}{
		{"success", "new", "", 1},
		{"invalid", "old", "config test failed", 0},
		{"restart", "old", "original config restored and service restarted", 2},
		{"recovery", "old", "recovery restart failed", 2},
	} {
		t.Run(tc.mode, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "config")
			link := filepath.Join(dir, "sshd_config")
			if err := os.WriteFile(path, []byte("old"), 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(path, link); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", dir+":"+os.Getenv("PATH"))
			t.Setenv("CONFIG", path)
			t.Setenv("MODE", tc.mode)
			calls := filepath.Join(dir, "calls")
			t.Setenv("CALLS", calls)
			for name, script := range map[string]string{
				"sshd": `#!/bin/sh
test "$1" = -t && test "$2" = -f && test "$(cat "$3")" = new && test "$(cat "$CONFIG")" = old && test "$MODE" != invalid
`,
				"systemctl": `#!/bin/sh
if [ "$1" = show ] && [ ! -e "$CALLS" ]; then echo loaded; exit 0; fi
test "$1" = restart || exit 2
cat "$CONFIG" >> "$CALLS"
if [ "$MODE" = recovery ]; then exit 1; fi
if [ "$MODE" = restart ] && [ "$(cat "$CONFIG")" = new ]; then exit 1; fi
`,
			} {
				if err := os.WriteFile(filepath.Join(dir, name), []byte(script), 0700); err != nil {
					t.Fatal(err)
				}
			}
			err := editSSHDConfig(link, func(string) string { return "new" })
			if tc.wantError == "" && err != nil || tc.wantError != "" && (err == nil || !strings.Contains(err.Error(), tc.wantError)) {
				t.Fatalf("error = %v", err)
			}
			if data, err := os.ReadFile(path); err != nil || string(data) != tc.want {
				t.Fatalf("config = %q %v", data, err)
			}
			info, err := os.Stat(path)
			if err != nil || info.Mode().Perm() != 0600 {
				t.Fatalf("config mode: %v", err)
			}
			if _, err := os.Readlink(link); err != nil {
				t.Fatal("config symlink was replaced")
			}
			called, _ := os.ReadFile(calls)
			wantCalls := []string{"", "new", "newold"}[tc.restarts]
			if string(called) != wantCalls {
				t.Fatalf("restart observed %q want %q", called, wantCalls)
			}
		})
	}
}
