package viper

import (
	"os"
	"path/filepath"
	"testing"

	spf "github.com/spf13/viper"
)

func TestConfigSearchPrefersExecutableDirOverCwd(t *testing.T) {
	root := t.TempDir()
	execDir := filepath.Join(root, "xpanel")
	cwd := filepath.Join(root, "nezha-agent")
	if err := os.MkdirAll(execDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(cwd, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(execDir, "config.yaml"), []byte("system:\n  db_path: /from-exec\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cwd, "config.yml"), []byte("system:\n  db_path: /from-agent\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(cwd)

	v := spf.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	addConfigSearchPaths(v, execDir)
	if err := v.ReadInConfig(); err != nil {
		t.Fatal(err)
	}
	used := v.ConfigFileUsed()
	if used != filepath.Join(execDir, "config.yaml") {
		t.Fatalf("used %s, want panel config.yaml", used)
	}
	if got := v.GetString("system.db_path"); got != "/from-exec" {
		t.Fatalf("db_path=%q", got)
	}
}
