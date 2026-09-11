package service

import (
	"os"
	"path/filepath"
	"testing"

	"xpanel/app/dto"
)

func TestOverwriteCopyKeepsDestinationWhenNestedSourceIsUnreadable(t *testing.T) {
	installFileServiceLog(t)
	if os.Geteuid() == 0 {
		t.Skip("root can read mode 000 files")
	}
	root := t.TempDir()
	srcDir := filepath.Join(root, "payload")
	blocked := filepath.Join(srcDir, "blocked.bin")
	writeFile(t, blocked, "new-content", 0o644)
	if err := os.Chmod(blocked, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(blocked, 0o644) })

	dstDir := filepath.Join(root, "dst")
	dst := filepath.Join(dstDir, "payload")
	writeFile(t, dst, "keep-original", 0o644)

	err := NewIFileService().Move(dto.FileMoveReq{
		SrcPaths:       []string{srcDir},
		DstPath:        dstDir,
		IsCopy:         true,
		ConflictPolicy: "overwrite",
	})
	if err == nil {
		t.Fatal("expected copy failure")
	}
	info, statErr := os.Lstat(dst)
	if statErr != nil {
		t.Fatalf("destination missing: %v", statErr)
	}
	if info.IsDir() {
		t.Fatal("destination was replaced by a partial directory")
	}
	if got := readFile(t, dst); got != "keep-original" {
		t.Fatalf("destination = %q", got)
	}
}

func TestMergeOverwriteKeepsDestinationWhenSourceCannotBeCopied(t *testing.T) {
	installFileServiceLog(t)
	if os.Geteuid() == 0 {
		t.Skip("root can read mode 000 files")
	}
	root := t.TempDir()
	dstRoot := filepath.Join(root, "destination")
	existing := filepath.Join(dstRoot, "keep.txt")
	writeFile(t, existing, "original", 0o644)

	src := filepath.Join(root, "extracted", "keep.txt")
	writeFile(t, src, "from-archive", 0o644)
	if err := os.Chmod(src, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(src, 0o644) })

	err := mergeExtractedPath(src, existing, dstRoot, "overwrite")
	if err == nil {
		t.Fatal("expected merge copy failure")
	}
	if got := readFile(t, existing); got != "original" {
		t.Fatalf("destination = %q", got)
	}
}
