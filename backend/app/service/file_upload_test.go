package service

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"xpanel/app/dto"
)

func TestPreflightUploadReportsNestedConflicts(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "site", "assets"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "site", "index.html"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := NewIFileService().PreflightUpload(dto.FileUploadPreflightReq{
		TargetPath:    root,
		RelativePaths: []string{"site/index.html", "site/assets/app.js"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.Conflicts, []string{"site/index.html"}) {
		t.Fatalf("conflicts = %#v", got.Conflicts)
	}
	if len(got.Blocked) != 0 {
		t.Fatalf("blocked = %#v", got.Blocked)
	}
}

func TestPreflightUploadBlocksTraversalAndDirectoryTargets(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "cache"), 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := NewIFileService().PreflightUpload(dto.FileUploadPreflightReq{
		TargetPath:    root,
		RelativePaths: []string{"../escape.txt", "site/../collapsed.txt", "cache"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Blocked) != 3 {
		t.Fatalf("blocked = %#v", got.Blocked)
	}
}

func TestSaveUploadCreatesNestedDirectories(t *testing.T) {
	root := t.TempDir()
	_, err := NewIFileService().SaveUpload(root, "site/assets/app.js", false, strings.NewReader("new"))
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, "site", "assets", "app.js"))
	if err != nil || string(data) != "new" {
		t.Fatalf("data=%q err=%v", data, err)
	}
}

func TestSaveUploadSkipPreservesExistingFile(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "config.json")
	if err := os.WriteFile(target, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := NewIFileService().SaveUpload(root, "config.json", false, strings.NewReader("new"))
	if !errors.Is(err, ErrUploadConflict) {
		t.Fatalf("err=%v", err)
	}
	data, readErr := os.ReadFile(target)
	if readErr != nil || string(data) != "old" {
		t.Fatalf("data=%q err=%v", data, readErr)
	}
}

func TestSaveUploadOverwriteReplacesExistingFile(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "config.json")
	if err := os.WriteFile(target, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := NewIFileService().SaveUpload(root, "config.json", true, strings.NewReader("new")); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(target)
	if err != nil || string(data) != "new" {
		t.Fatalf("data=%q err=%v", data, err)
	}
	info, err := os.Stat(target)
	if err != nil || info.Mode().Perm() != 0o644 {
		t.Fatalf("mode=%v err=%v", info.Mode().Perm(), err)
	}
}

func TestSaveUploadRejectsSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "escape")); err != nil {
		t.Fatal(err)
	}

	_, err := NewIFileService().SaveUpload(root, "escape/payload.txt", false, strings.NewReader("data"))
	if err == nil {
		t.Fatal("expected symlink escape error")
	}
	if _, statErr := os.Stat(filepath.Join(outside, "payload.txt")); !errors.Is(statErr, fs.ErrNotExist) {
		t.Fatalf("outside file stat err=%v", statErr)
	}
}

func TestSaveUploadFailedOverwritePreservesExistingFile(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "config.json")
	if err := os.WriteFile(target, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := NewIFileService().SaveUpload(
		root,
		"config.json",
		true,
		io.MultiReader(strings.NewReader("partial"), errReader{}),
	)
	if err == nil {
		t.Fatal("expected write error")
	}
	data, readErr := os.ReadFile(target)
	if readErr != nil || string(data) != "old" {
		t.Fatalf("data=%q err=%v", data, readErr)
	}
}

func TestPreflightUploadBlocksSymlinkTarget(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target.txt")
	link := filepath.Join(root, "link.txt")
	if err := os.WriteFile(target, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("target.txt", link); err != nil {
		t.Fatal(err)
	}

	got, err := NewIFileService().PreflightUpload(dto.FileUploadPreflightReq{
		TargetPath:    root,
		RelativePaths: []string{"link.txt"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Conflicts) != 0 || len(got.Blocked) != 1 {
		t.Fatalf("preflight = %#v", got)
	}
}

func TestSaveUploadOverwriteRejectsSymlinkTarget(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target.txt")
	link := filepath.Join(root, "link.txt")
	if err := os.WriteFile(target, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("target.txt", link); err != nil {
		t.Fatal(err)
	}

	_, err := NewIFileService().SaveUpload(root, "link.txt", true, strings.NewReader("new"))
	if !errors.Is(err, ErrInvalidUploadPath) {
		t.Fatalf("err=%v", err)
	}
	data, readErr := os.ReadFile(target)
	if readErr != nil || string(data) != "old" {
		t.Fatalf("target data=%q err=%v", data, readErr)
	}
	info, lstatErr := os.Lstat(link)
	if lstatErr != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("link mode=%v err=%v", info.Mode(), lstatErr)
	}
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) {
	return 0, errors.New("forced read failure")
}
