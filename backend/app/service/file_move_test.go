package service

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"xpanel/app/dto"
	"xpanel/global"

	"github.com/sirupsen/logrus"
)

func installFileServiceLog(t *testing.T) {
	t.Helper()
	previous := global.LOG
	global.LOG = logrus.New()
	t.Cleanup(func() { global.LOG = previous })
}

func writeFile(t *testing.T, path, content string, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestMoveSkipKeepsExistingDestination(t *testing.T) {
	installFileServiceLog(t)
	root := t.TempDir()
	src := filepath.Join(root, "src", "important.txt")
	dstDir := filepath.Join(root, "dst")
	dst := filepath.Join(dstDir, "important.txt")
	writeFile(t, src, "source", 0o644)
	writeFile(t, dst, "destination", 0o644)

	err := NewIFileService().Move(dto.FileMoveReq{
		SrcPaths:       []string{src},
		DstPath:        dstDir,
		ConflictPolicy: "skip",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, src); got != "source" {
		t.Fatalf("source = %q", got)
	}
	if got := readFile(t, dst); got != "destination" {
		t.Fatalf("destination = %q", got)
	}
}

func TestMoveOverwriteReplacesDestination(t *testing.T) {
	installFileServiceLog(t)
	root := t.TempDir()
	src := filepath.Join(root, "src", "important.txt")
	dstDir := filepath.Join(root, "dst")
	dst := filepath.Join(dstDir, "important.txt")
	writeFile(t, src, "source", 0o644)
	writeFile(t, dst, "destination", 0o644)

	err := NewIFileService().Move(dto.FileMoveReq{
		SrcPaths:       []string{src},
		DstPath:        dstDir,
		ConflictPolicy: "overwrite",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Fatalf("source still exists: %v", err)
	}
	if got := readFile(t, dst); got != "source" {
		t.Fatalf("destination = %q", got)
	}
}

func TestCopyToSameDirectoryDoesNotDeleteSource(t *testing.T) {
	installFileServiceLog(t)
	root := t.TempDir()
	src := filepath.Join(root, "a.txt")
	writeFile(t, src, "keep-me", 0o644)

	err := NewIFileService().Move(dto.FileMoveReq{
		SrcPaths:       []string{src},
		DstPath:        root,
		IsCopy:         true,
		ConflictPolicy: "overwrite",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, src); got != "keep-me" {
		t.Fatalf("source = %q", got)
	}
}

func TestCopyThroughDestinationSymlinkAliasDoesNotDeleteSource(t *testing.T) {
	installFileServiceLog(t)
	root := t.TempDir()
	dataDir := filepath.Join(root, "data")
	src := filepath.Join(dataDir, "a.txt")
	writeFile(t, src, "keep-me", 0o644)
	alias := filepath.Join(root, "alias")
	if err := os.Symlink(dataDir, alias); err != nil {
		t.Fatal(err)
	}

	err := NewIFileService().Move(dto.FileMoveReq{
		SrcPaths:       []string{src},
		DstPath:        alias,
		IsCopy:         true,
		ConflictPolicy: "overwrite",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, src); got != "keep-me" {
		t.Fatalf("source = %q", got)
	}
}

func TestOverwriteMissingSourcePreservesDestination(t *testing.T) {
	installFileServiceLog(t)
	root := t.TempDir()
	dstDir := filepath.Join(root, "dst")
	dst := filepath.Join(dstDir, "a.txt")
	writeFile(t, dst, "keep-dest", 0o644)

	err := NewIFileService().Move(dto.FileMoveReq{
		SrcPaths:       []string{filepath.Join(root, "a.txt")},
		DstPath:        dstDir,
		IsCopy:         true,
		ConflictPolicy: "overwrite",
	})
	if err == nil {
		t.Fatal("expected missing source error")
	}
	if got := readFile(t, dst); got != "keep-dest" {
		t.Fatalf("destination = %q", got)
	}
}

func TestCopyDirectoryPreservesSymlink(t *testing.T) {
	installFileServiceLog(t)
	root := t.TempDir()
	srcDir := filepath.Join(root, "src")
	dstDir := filepath.Join(root, "parent")
	writeFile(t, filepath.Join(srcDir, "real.txt"), "hello", 0o644)
	if err := os.Symlink("real.txt", filepath.Join(srcDir, "link.txt")); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(dstDir, 0o755); err != nil {
		t.Fatal(err)
	}

	err := NewIFileService().Move(dto.FileMoveReq{
		SrcPaths: []string{srcDir},
		DstPath:  dstDir,
		IsCopy:   true,
	})
	if err != nil {
		t.Fatal(err)
	}

	copiedLink := filepath.Join(dstDir, "src", "link.txt")
	info, err := os.Lstat(copiedLink)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("copied link is not a symlink: mode=%v", info.Mode())
	}
	target, err := os.Readlink(copiedLink)
	if err != nil || target != "real.txt" {
		t.Fatalf("symlink target = %q err=%v", target, err)
	}
	if got := readFile(t, copiedLink); got != "hello" {
		t.Fatalf("followed content = %q", got)
	}
}

func TestCopyDirectoryPreservesPrivateMode(t *testing.T) {
	installFileServiceLog(t)
	root := t.TempDir()
	srcDir := filepath.Join(root, "private")
	dstParent := filepath.Join(root, "dst")
	if err := os.Mkdir(srcDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(srcDir, 0o700); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(srcDir, "secret.txt"), "secret", 0o600)
	if err := os.Mkdir(dstParent, 0o755); err != nil {
		t.Fatal(err)
	}

	err := NewIFileService().Move(dto.FileMoveReq{
		SrcPaths: []string{srcDir},
		DstPath:  dstParent,
		IsCopy:   true,
	})
	if err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(filepath.Join(dstParent, "private"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o700 {
		t.Fatalf("dir mode = %o", info.Mode().Perm())
	}
}

func TestCopyPreservesDanglingAndDirectorySymlinks(t *testing.T) {
	installFileServiceLog(t)
	root := t.TempDir()
	srcDir := filepath.Join(root, "src")
	dstParent := filepath.Join(root, "dst")
	if err := os.MkdirAll(filepath.Join(srcDir, "realdir"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(srcDir, "realdir", "file.txt"), "ok", 0o644)
	if err := os.Symlink("missing.txt", filepath.Join(srcDir, "broken")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("realdir", filepath.Join(srcDir, "linkdir")); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(dstParent, 0o755); err != nil {
		t.Fatal(err)
	}

	err := NewIFileService().Move(dto.FileMoveReq{
		SrcPaths: []string{srcDir},
		DstPath:  dstParent,
		IsCopy:   true,
	})
	if err != nil {
		t.Fatal(err)
	}

	copied := filepath.Join(dstParent, "src")
	broken, err := os.Lstat(filepath.Join(copied, "broken"))
	if err != nil || broken.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("dangling symlink missing: err=%v", err)
	}
	linkdir, err := os.Lstat(filepath.Join(copied, "linkdir"))
	if err != nil || linkdir.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("directory symlink became a directory: err=%v mode=%v", err, linkdir)
	}
	if _, err := os.Stat(filepath.Join(copied, "linkdir", "file.txt")); err != nil {
		t.Fatalf("directory symlink target not reachable: %v", err)
	}
}

func TestCopyRejectsFifo(t *testing.T) {
	installFileServiceLog(t)
	root := t.TempDir()
	srcDir := filepath.Join(root, "src")
	dstParent := filepath.Join(root, "dst")
	if err := os.Mkdir(srcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := syscall.Mkfifo(filepath.Join(srcDir, "pipe"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(dstParent, 0o755); err != nil {
		t.Fatal(err)
	}

	done := make(chan error, 1)
	go func() {
		done <- NewIFileService().Move(dto.FileMoveReq{
			SrcPaths: []string{srcDir},
			DstPath:  dstParent,
			IsCopy:   true,
		})
	}()
	var err error
	select {
	case err = <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("copy hung on fifo")
	}
	if err == nil {
		t.Fatal("expected fifo copy to fail")
	}
	if _, statErr := os.Lstat(filepath.Join(dstParent, "src", "pipe")); !os.IsNotExist(statErr) {
		t.Fatalf("fifo should not be copied: %v", statErr)
	}
}

func TestCopyModeDoesNotWidenDirectory(t *testing.T) {
	installFileServiceLog(t)
	root := t.TempDir()
	srcDir := filepath.Join(root, "locked")
	dstParent := filepath.Join(root, "dst")
	if err := os.Mkdir(srcDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(srcDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(dstParent, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := NewIFileService().Move(dto.FileMoveReq{
		SrcPaths: []string{srcDir},
		DstPath:  dstParent,
		IsCopy:   true,
	}); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(filepath.Join(dstParent, "locked"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("copied dir widened permissions: %o", info.Mode().Perm())
	}
}

func TestMoveSameParentIsSafe(t *testing.T) {
	installFileServiceLog(t)
	root := t.TempDir()
	src := filepath.Join(root, "a.txt")
	writeFile(t, src, "keep-me", 0o644)

	err := NewIFileService().Move(dto.FileMoveReq{
		SrcPaths:       []string{src},
		DstPath:        root,
		ConflictPolicy: "overwrite",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, src); got != "keep-me" {
		t.Fatalf("source = %q", got)
	}
}
