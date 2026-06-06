package v1

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveLocalTerminalDirUsesValidAbsoluteDirectory(t *testing.T) {
	home := t.TempDir()
	target := filepath.Join(home, "etc", "apparmor.d")
	if err := os.MkdirAll(target, 0755); err != nil {
		t.Fatalf("create target dir: %v", err)
	}
	t.Setenv("HOME", home)

	got := resolveLocalTerminalDir(" " + target + " ")

	if got != target {
		t.Fatalf("resolveLocalTerminalDir() = %q, want %q", got, target)
	}
}

func TestResolveLocalTerminalDirFallsBackToHome(t *testing.T) {
	home := t.TempDir()
	filePath := filepath.Join(home, "not-a-dir")
	if err := os.WriteFile(filePath, []byte("x"), 0644); err != nil {
		t.Fatalf("create file: %v", err)
	}
	t.Setenv("HOME", home)

	tests := []string{
		"",
		"relative/path",
		filePath,
		filepath.Join(home, "missing"),
	}

	for _, tt := range tests {
		if got := resolveLocalTerminalDir(tt); got != home {
			t.Fatalf("resolveLocalTerminalDir(%q) = %q, want %q", tt, got, home)
		}
	}
}
