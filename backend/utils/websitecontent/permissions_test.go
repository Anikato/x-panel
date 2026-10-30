package websitecontent

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestEnsureDirectoryAndFileIgnoreRestrictiveUmask(t *testing.T) {
	previous := syscall.Umask(0o077)
	defer syscall.Umask(previous)

	policy := Policy{
		UID:           os.Geteuid(),
		GID:           os.Getegid(),
		DirectoryMode: 0o750,
		FileMode:      0o640,
	}
	root := t.TempDir()
	nested := filepath.Join(root, "site", "public")
	if err := EnsureDirectory(nested, policy); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{filepath.Join(root, "site"), nested} {
		info, err := os.Stat(path)
		if err != nil || info.Mode().Perm() != policy.DirectoryMode {
			t.Fatalf("directory %s mode=%v err=%v", path, info.Mode().Perm(), err)
		}
	}
	indexPath := filepath.Join(nested, "index.html")
	created, err := CreateFileIfMissing(indexPath, []byte("ok"), policy)
	if err != nil || !created {
		t.Fatalf("created=%v err=%v", created, err)
	}
	info, err := os.Stat(indexPath)
	if err != nil || info.Mode().Perm() != policy.FileMode {
		t.Fatalf("file mode=%v err=%v", info.Mode().Perm(), err)
	}
}

func TestEnsureDirectoryPreservesExistingDirectoryMode(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "existing")
	if err := os.Mkdir(path, 0o711); err != nil {
		t.Fatal(err)
	}
	if err := EnsureDirectory(path, Policy{UID: os.Geteuid(), GID: os.Getegid(), DirectoryMode: 0o750, FileMode: 0o640}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o711 {
		t.Fatalf("mode=%v err=%v", info.Mode().Perm(), err)
	}
}
