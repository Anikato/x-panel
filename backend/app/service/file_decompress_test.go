package service

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"xpanel/app/dto"
)

func TestIsSafeArchiveEntryAllowsTarRootDot(t *testing.T) {
	safe := []string{"./", ".", "./sub/", "./sub/file.txt", "sub/file.txt"}
	for _, entry := range safe {
		if !isSafeArchiveEntry(entry) {
			t.Errorf("isSafeArchiveEntry(%q) = false, want true", entry)
		}
	}

	unsafe := []string{"../escape", "/etc/passwd", "..", "foo/../../etc/passwd", "abs\x00path"}
	for _, entry := range unsafe {
		if isSafeArchiveEntry(entry) {
			t.Errorf("isSafeArchiveEntry(%q) = true, want false", entry)
		}
	}
}

func TestDecompressRejectsWritingThroughDestinationSymlink(t *testing.T) {
	installFileServiceLog(t)
	root := t.TempDir()
	destination := filepath.Join(root, "destination")
	outside := filepath.Join(root, "outside")
	if err := os.MkdirAll(destination, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(outside, "important.txt"), "original", 0o644)
	if err := os.Symlink(filepath.Join("..", "outside"), filepath.Join(destination, "sub")); err != nil {
		t.Fatal(err)
	}

	content := filepath.Join(root, "content", "sub")
	writeFile(t, filepath.Join(content, "important.txt"), "replaced", 0o644)
	archive := filepath.Join(root, "input.tar")
	cmd := exec.Command("tar", "-cf", archive, "-C", filepath.Join(root, "content"), "sub")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("create tar: %v\n%s", err, output)
	}

	err := NewIFileService().Decompress(dto.FileDecompressReq{
		Path:           archive,
		Dst:            destination,
		ConflictPolicy: "overwrite",
	})
	if got := readFile(t, filepath.Join(outside, "important.txt")); got != "original" {
		t.Fatalf("outside file mutated: %q", got)
	}
	if err == nil {
		t.Fatal("expected symlink escape to fail")
	}
}

func TestDecompressSkipAndRenameDoNotEscapeSymlink(t *testing.T) {
	installFileServiceLog(t)
	for _, policy := range []string{"skip", "rename"} {
		t.Run(policy, func(t *testing.T) {
			root := t.TempDir()
			destination := filepath.Join(root, "destination")
			outside := filepath.Join(root, "outside")
			if err := os.MkdirAll(destination, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(outside, 0o755); err != nil {
				t.Fatal(err)
			}
			writeFile(t, filepath.Join(outside, "important.txt"), "original", 0o644)
			if err := os.Symlink(filepath.Join("..", "outside"), filepath.Join(destination, "sub")); err != nil {
				t.Fatal(err)
			}

			content := filepath.Join(root, "content", "sub")
			writeFile(t, filepath.Join(content, "important.txt"), "replaced", 0o644)
			archive := filepath.Join(root, "input.tar")
			cmd := exec.Command("tar", "-cf", archive, "-C", filepath.Join(root, "content"), "sub")
			if output, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("create tar: %v\n%s", err, output)
			}

			err := NewIFileService().Decompress(dto.FileDecompressReq{
				Path:           archive,
				Dst:            destination,
				ConflictPolicy: policy,
			})
			if got := readFile(t, filepath.Join(outside, "important.txt")); got != "original" {
				t.Fatalf("outside file mutated: %q", got)
			}
			if err == nil {
				t.Fatal("expected symlink escape to fail")
			}
		})
	}
}

func TestDecompressSkipDoesNotCreateThroughEmptySymlink(t *testing.T) {
	installFileServiceLog(t)
	root := t.TempDir()
	destination := filepath.Join(root, "destination")
	outside := filepath.Join(root, "outside")
	if err := os.MkdirAll(destination, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join("..", "outside"), filepath.Join(destination, "sub")); err != nil {
		t.Fatal(err)
	}

	content := filepath.Join(root, "content", "sub")
	writeFile(t, filepath.Join(content, "important.txt"), "replaced", 0o644)
	archive := filepath.Join(root, "input.tar")
	cmd := exec.Command("tar", "-cf", archive, "-C", filepath.Join(root, "content"), "sub")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("create tar: %v\n%s", err, output)
	}

	err := NewIFileService().Decompress(dto.FileDecompressReq{
		Path:           archive,
		Dst:            destination,
		ConflictPolicy: "skip",
	})
	if _, statErr := os.Stat(filepath.Join(outside, "important.txt")); !os.IsNotExist(statErr) {
		t.Fatalf("outside file created through symlink: %v", statErr)
	}
	if err == nil {
		t.Fatal("expected symlink escape to fail")
	}
}

func TestDecompressNormalNestedArchive(t *testing.T) {
	installFileServiceLog(t)
	root := t.TempDir()
	destination := filepath.Join(root, "destination")
	if err := os.MkdirAll(destination, 0o755); err != nil {
		t.Fatal(err)
	}
	content := filepath.Join(root, "content", "nested", "dir")
	writeFile(t, filepath.Join(content, "file.txt"), "ok", 0o644)
	archive := filepath.Join(root, "input.tar")
	cmd := exec.Command("tar", "-cf", archive, "-C", filepath.Join(root, "content"), "nested")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("create tar: %v\n%s", err, output)
	}

	if err := NewIFileService().Decompress(dto.FileDecompressReq{
		Path: archive,
		Dst:  destination,
	}); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, filepath.Join(destination, "nested", "dir", "file.txt")); got != "ok" {
		t.Fatalf("extracted = %q", got)
	}
}

func TestDecompressAllowsTarRootDotEntries(t *testing.T) {
	installFileServiceLog(t)
	root := t.TempDir()
	destination := filepath.Join(root, "destination")
	if err := os.MkdirAll(destination, 0o755); err != nil {
		t.Fatal(err)
	}
	content := filepath.Join(root, "content", "sub")
	writeFile(t, filepath.Join(content, "file.txt"), "from-dot", 0o644)
	archive := filepath.Join(root, "input.tar")
	cmd := exec.Command("tar", "-cf", archive, "-C", filepath.Join(root, "content"), ".")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("create tar: %v\n%s", err, output)
	}

	listed, err := exec.Command("tar", "-tf", archive).CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(listed), "./") {
		t.Fatalf("fixture tar missing ./ entry: %s", listed)
	}

	if err := NewIFileService().Decompress(dto.FileDecompressReq{
		Path: archive,
		Dst:  destination,
	}); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, filepath.Join(destination, "sub", "file.txt")); got != "from-dot" {
		t.Fatalf("extracted = %q", got)
	}
}

func TestDecompressRejectsPathTraversalEntries(t *testing.T) {
	installFileServiceLog(t)
	root := t.TempDir()
	destination := filepath.Join(root, "destination")
	if err := os.MkdirAll(destination, 0o755); err != nil {
		t.Fatal(err)
	}
	content := filepath.Join(root, "content")
	if err := os.MkdirAll(content, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(content, "ok.txt"), "x", 0o644)
	archive := filepath.Join(root, "evil.tar")
	cmd := exec.Command("tar", "-cf", archive, "-C", content, "ok.txt")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("create tar: %v\n%s", err, output)
	}

	if isSafeArchiveEntry("../outside.txt") {
		t.Fatal("traversal entry must stay rejected")
	}
}
