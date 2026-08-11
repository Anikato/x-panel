package service

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
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

func uploadChunkChecksum(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func uploadChunkRequest(root, relativePath, uploadID string, index, count int, totalSize int64, data []byte) dto.FileUploadChunkReq {
	return dto.FileUploadChunkReq{
		TargetPath:   root,
		RelativePath: relativePath,
		UploadID:     uploadID,
		ChunkIndex:   index,
		ChunkCount:   count,
		TotalSize:    totalSize,
		Checksum:     uploadChunkChecksum(data),
	}
}

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

func TestChunkedUploadCompletesAtomically(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "archive.bin")
	if err := os.WriteFile(target, []byte("old"), 0o640); err != nil {
		t.Fatal(err)
	}

	first := bytes.Repeat([]byte("a"), uploadChunkSize)
	last := []byte("tail")
	totalSize := int64(len(first) + len(last))
	uploadID := "0123456789abcdef0123456789abcdef"
	svc := NewIFileService()

	if err := svc.SaveUploadChunk(
		uploadChunkRequest(root, "archive.bin", uploadID, 0, 2, totalSize, first),
		bytes.NewReader(first),
	); err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(target); err != nil || string(data) != "old" {
		t.Fatalf("target changed before completion: data=%q err=%v", data, err)
	}
	if err := svc.SaveUploadChunk(
		uploadChunkRequest(root, "archive.bin", uploadID, 1, 2, totalSize, last),
		bytes.NewReader(last),
	); err != nil {
		t.Fatal(err)
	}

	savedPath, err := svc.CompleteUploadChunks(dto.FileUploadChunkCompleteReq{
		TargetPath: root, RelativePath: "archive.bin", UploadID: uploadID,
		TotalSize: totalSize, Overwrite: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if savedPath != "archive.bin" {
		t.Fatalf("savedPath=%q", savedPath)
	}
	want := append(append([]byte{}, first...), last...)
	data, err := os.ReadFile(target)
	if err != nil || !bytes.Equal(data, want) {
		t.Fatalf("completed data length=%d err=%v", len(data), err)
	}
	info, err := os.Stat(target)
	if err != nil || info.Mode().Perm() != 0o640 {
		t.Fatalf("mode=%v err=%v", info.Mode().Perm(), err)
	}
}

func TestSaveUploadChunkChecksumFailureRollsBack(t *testing.T) {
	root := t.TempDir()
	first := bytes.Repeat([]byte("a"), uploadChunkSize)
	last := []byte("tail")
	totalSize := int64(len(first) + len(last))
	uploadID := "11111111111111111111111111111111"
	svc := NewIFileService()

	if err := svc.SaveUploadChunk(
		uploadChunkRequest(root, "archive.bin", uploadID, 0, 2, totalSize, first),
		bytes.NewReader(first),
	); err != nil {
		t.Fatal(err)
	}
	req := uploadChunkRequest(root, "archive.bin", uploadID, 1, 2, totalSize, last)
	req.Checksum = uploadChunkChecksum([]byte("fail"))
	if err := svc.SaveUploadChunk(req, bytes.NewReader(last)); !errors.Is(err, ErrUploadChecksum) {
		t.Fatalf("err=%v", err)
	}

	tempPath, err := uploadChunkTempPath("archive.bin", uploadID, totalSize)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(root, filepath.FromSlash(tempPath)))
	if err != nil || info.Size() != int64(uploadChunkSize) {
		t.Fatalf("temp size=%d err=%v", info.Size(), err)
	}
}

func TestSaveUploadChunkRejectsOutOfOrderChunk(t *testing.T) {
	root := t.TempDir()
	last := []byte("tail")
	totalSize := int64(uploadChunkSize + len(last))
	req := uploadChunkRequest(root, "archive.bin", "22222222222222222222222222222222", 1, 2, totalSize, last)

	err := NewIFileService().SaveUploadChunk(req, bytes.NewReader(last))
	if !errors.Is(err, ErrUploadChunkOrder) {
		t.Fatalf("err=%v", err)
	}
}

func TestSaveUploadChunkRejectsTotalSizeChangeForUploadID(t *testing.T) {
	root := t.TempDir()
	first := bytes.Repeat([]byte("a"), uploadChunkSize)
	second := bytes.Repeat([]byte("b"), uploadChunkSize)
	uploadID := "99999999999999999999999999999999"
	svc := NewIFileService()
	if err := svc.SaveUploadChunk(
		uploadChunkRequest(root, "archive.bin", uploadID, 0, 1, int64(len(first)), first),
		bytes.NewReader(first),
	); err != nil {
		t.Fatal(err)
	}

	err := svc.SaveUploadChunk(
		uploadChunkRequest(root, "archive.bin", uploadID, 1, 2, int64(len(first)+len(second)), second),
		bytes.NewReader(second),
	)
	if !errors.Is(err, ErrUploadChunkOrder) {
		t.Fatalf("err=%v", err)
	}
}

func TestVerifyUploadTempFileRejectsPublishedInode(t *testing.T) {
	rootPath := t.TempDir()
	root, err := os.OpenRoot(rootPath)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	tempPath := ".xpanel-upload-test.part"
	if err := root.WriteFile(tempPath, []byte("partial"), 0o600); err != nil {
		t.Fatal(err)
	}
	file, err := root.Open(tempPath)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	openedInfo, err := file.Stat()
	if err != nil {
		t.Fatal(err)
	}
	if err := root.Rename(tempPath, "published.bin"); err != nil {
		t.Fatal(err)
	}
	if err := verifyUploadTempFile(root, tempPath, openedInfo); !errors.Is(err, ErrUploadChunkOrder) {
		t.Fatalf("err=%v", err)
	}
}

func TestCompleteUploadChunksRejectsIncompleteFileAndPreservesTarget(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "archive.bin")
	if err := os.WriteFile(target, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	first := bytes.Repeat([]byte("a"), uploadChunkSize)
	totalSize := int64(uploadChunkSize + 4)
	uploadID := "33333333333333333333333333333333"
	svc := NewIFileService()
	if err := svc.SaveUploadChunk(
		uploadChunkRequest(root, "archive.bin", uploadID, 0, 2, totalSize, first),
		bytes.NewReader(first),
	); err != nil {
		t.Fatal(err)
	}

	_, err := svc.CompleteUploadChunks(dto.FileUploadChunkCompleteReq{
		TargetPath: root, RelativePath: "archive.bin", UploadID: uploadID,
		TotalSize: totalSize, Overwrite: true,
	})
	if !errors.Is(err, ErrUploadSizeMismatch) {
		t.Fatalf("err=%v", err)
	}
	data, readErr := os.ReadFile(target)
	if readErr != nil || string(data) != "old" {
		t.Fatalf("target data=%q err=%v", data, readErr)
	}
}

func TestCompleteUploadChunksWithoutOverwritePreservesConflict(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "archive.bin")
	if err := os.WriteFile(target, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	data := []byte("new")
	uploadID := "44444444444444444444444444444444"
	svc := NewIFileService()
	if err := svc.SaveUploadChunk(
		uploadChunkRequest(root, "archive.bin", uploadID, 0, 1, int64(len(data)), data),
		bytes.NewReader(data),
	); err != nil {
		t.Fatal(err)
	}

	_, err := svc.CompleteUploadChunks(dto.FileUploadChunkCompleteReq{
		TargetPath: root, RelativePath: "archive.bin", UploadID: uploadID,
		TotalSize: int64(len(data)), Overwrite: false,
	})
	if !errors.Is(err, ErrUploadConflict) {
		t.Fatalf("err=%v", err)
	}
	got, readErr := os.ReadFile(target)
	if readErr != nil || string(got) != "old" {
		t.Fatalf("target data=%q err=%v", got, readErr)
	}
}

func TestAbortUploadChunksRemovesTemporaryFile(t *testing.T) {
	root := t.TempDir()
	data := []byte("new")
	uploadID := "55555555555555555555555555555555"
	svc := NewIFileService()
	if err := svc.SaveUploadChunk(
		uploadChunkRequest(root, "archive.bin", uploadID, 0, 1, int64(len(data)), data),
		bytes.NewReader(data),
	); err != nil {
		t.Fatal(err)
	}
	if err := svc.AbortUploadChunks(dto.FileUploadChunkAbortReq{
		TargetPath: root, RelativePath: "archive.bin", UploadID: uploadID, TotalSize: int64(len(data)),
	}); err != nil {
		t.Fatal(err)
	}
	tempPath, err := uploadChunkTempPath("archive.bin", uploadID, int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(tempPath))); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("temp stat err=%v", err)
	}
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) {
	return 0, errors.New("forced read failure")
}
