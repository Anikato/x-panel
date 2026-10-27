package ssl

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"xpanel/global"
)

func TestHTTP01WebRootStaysOutsidePrivateDataDir(t *testing.T) {
	previous := global.CONF.System.DataDir
	t.Cleanup(func() { global.CONF.System.DataDir = previous })
	global.CONF.System.DataDir = filepath.Join("/opt/xpanel", "data")

	got := HTTP01WebRoot()
	want := filepath.Join("/opt/xpanel", "acme-http")
	if got != want {
		t.Fatalf("HTTP01WebRoot() = %s, want %s", got, want)
	}
	if filepath.Dir(got) != filepath.Dir(global.CONF.System.DataDir) {
		t.Fatalf("webroot %s is not beside the data directory", got)
	}
}

func TestWriteHTTP01KeyAuthRemainsReadableWhenProcessUmaskIsPrivate(t *testing.T) {
	previousUmask := syscall.Umask(0o077)
	t.Cleanup(func() { syscall.Umask(previousUmask) })

	root := t.TempDir()
	if err := os.Chmod(root, 0o755); err != nil {
		t.Fatal(err)
	}
	dataDir := filepath.Join(root, "data")
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		t.Fatal(err)
	}
	previous := global.CONF.System.DataDir
	t.Cleanup(func() { global.CONF.System.DataDir = previous })
	global.CONF.System.DataDir = dataDir

	const token = "challenge-token"
	if err := WriteHTTP01KeyAuth(token, "key-auth"); err != nil {
		t.Fatal(err)
	}

	webRoot := HTTP01WebRoot()
	if filepath.Dir(webRoot) != root {
		t.Fatalf("webroot %s is inside the private data directory", webRoot)
	}
	assertModeAtLeast(t, webRoot, 0o755)
	assertModeAtLeast(t, HTTP01ChallengeDir(), 0o755)
	assertModeAtLeast(t, HTTP01ChallengeFile(token), 0o644)
	body, err := os.ReadFile(HTTP01ChallengeFile(token))
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "key-auth" {
		t.Fatalf("challenge body = %q", body)
	}
}

func assertModeAtLeast(t *testing.T, path string, want os.FileMode) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&want != want {
		t.Fatalf("%s mode = %o, want at least %o", path, info.Mode().Perm(), want)
	}
}
