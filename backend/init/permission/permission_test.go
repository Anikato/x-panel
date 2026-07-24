package permission

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHardenRestrictsOnlyRegisteredOwnedPaths(t *testing.T) {
	root := t.TempDir()
	privateDir := filepath.Join(root, "data")
	if err := os.Mkdir(privateDir, 0755); err != nil {
		t.Fatalf("create private directory: %v", err)
	}
	privateFile := filepath.Join(privateDir, "xpanel.db")
	if err := os.WriteFile(privateFile, []byte("db"), 0644); err != nil {
		t.Fatalf("create private file: %v", err)
	}
	externalCertificate := filepath.Join(root, "external-certificate.pem")
	if err := os.WriteFile(externalCertificate, []byte("certificate"), 0644); err != nil {
		t.Fatalf("create external certificate: %v", err)
	}

	if err := Harden(Paths{
		Directories: []string{privateDir},
		Files:       []string{privateFile},
	}); err != nil {
		t.Fatalf("harden owned paths: %v", err)
	}

	assertMode(t, privateDir, 0700)
	assertMode(t, privateFile, 0600)
	assertMode(t, externalCertificate, 0644)
}

func TestHardenRejectsSymlinkTargets(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target")
	if err := os.WriteFile(target, []byte("target"), 0600); err != nil {
		t.Fatalf("create symlink target: %v", err)
	}
	link := filepath.Join(root, "database")
	if err := os.Symlink(target, link); err != nil {
		t.Fatalf("create symlink: %v", err)
	}

	if err := Harden(Paths{Files: []string{link}}); err == nil {
		t.Fatalf("symlink target should be rejected")
	}
}

func TestHardenIgnoresRegisteredFilesThatDoNotExistYet(t *testing.T) {
	if err := Harden(Paths{Files: []string{filepath.Join(t.TempDir(), "future.db-wal")}}); err != nil {
		t.Fatalf("missing runtime sidecar should be ignored: %v", err)
	}
}

func TestHardenOwnedRejectsPathsOutsideManagedRoot(t *testing.T) {
	root := t.TempDir()
	external := filepath.Join(t.TempDir(), "external.db")
	if err := os.WriteFile(external, []byte("db"), 0644); err != nil {
		t.Fatalf("create external file: %v", err)
	}

	err := HardenOwned(root, Paths{Files: []string{external}})
	if err == nil || !strings.Contains(err.Error(), "outside managed root") {
		t.Fatalf("outside path error = %v", err)
	}
	assertMode(t, external, 0644)
}

func TestHardenOwnedRejectsParentSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	external := t.TempDir()
	link := filepath.Join(root, "linked")
	if err := os.Symlink(external, link); err != nil {
		t.Fatalf("create parent symlink: %v", err)
	}

	escaped := filepath.Join(link, "private")
	err := HardenOwned(root, Paths{Directories: []string{escaped}})
	if err == nil || !strings.Contains(err.Error(), "escapes managed root") {
		t.Fatalf("parent symlink error = %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(external, "private")); !os.IsNotExist(statErr) {
		t.Fatalf("escaped directory should not be created, stat error = %v", statErr)
	}
}

func TestResolveInstallRootDoesNotPromoteTopLevelDataDirectory(t *testing.T) {
	if got := ResolveInstallRoot("/data"); got != "/data" {
		t.Fatalf("ResolveInstallRoot(/data) = %q, want /data", got)
	}
	if got := ResolveInstallRoot("/srv/xpanel/data"); got != "/srv/xpanel" {
		t.Fatalf("ResolveInstallRoot(/srv/xpanel/data) = %q, want /srv/xpanel", got)
	}
	if got := ResolveInstallRoot("data"); got != "." {
		t.Fatalf("ResolveInstallRoot(data) = %q, want .", got)
	}
}

func assertMode(t *testing.T, path string, want os.FileMode) {
	t.Helper()
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatalf("inspect %s: %v", path, err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Fatalf("%s mode = %#o, want %#o", path, got, want)
	}
}
