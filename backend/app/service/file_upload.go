package service

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"strings"
	"syscall"
	"time"

	"xpanel/app/dto"
)

const (
	maxUploadFiles       = 1000
	uploadChunkSize      = 5 * 1024 * 1024
	uploadChunkMaxAge    = 24 * time.Hour
	uploadChunkTempStart = ".xpanel-upload-"
	uploadChunkTempEnd   = ".part"
)

var ErrUploadConflict = errors.New("upload target already exists")
var ErrInvalidUploadPath = errors.New("invalid upload path")
var ErrUploadChecksum = errors.New("upload chunk checksum mismatch")
var ErrUploadChunkOrder = errors.New("upload chunk is out of order")
var ErrUploadSizeMismatch = errors.New("upload size mismatch")
var ErrInvalidUploadMetadata = errors.New("invalid upload metadata")

func normalizeUploadRelativePath(value string) (string, error) {
	value = strings.ReplaceAll(value, `\`, "/")
	clean := path.Clean(value)
	hasParentSegment := false
	for _, segment := range strings.Split(value, "/") {
		if segment == ".." {
			hasParentSegment = true
			break
		}
	}
	if clean == "." || path.IsAbs(clean) || hasParentSegment ||
		strings.ContainsRune(clean, 0) || strings.ContainsAny(clean, "\r\n") {
		return "", fmt.Errorf("%w %q", ErrInvalidUploadPath, value)
	}
	return clean, nil
}

func (s *FileService) PreflightUpload(req dto.FileUploadPreflightReq) (*dto.FileUploadPreflightResp, error) {
	if len(req.RelativePaths) == 0 || len(req.RelativePaths) > maxUploadFiles {
		return nil, fmt.Errorf("upload file count must be between 1 and %d", maxUploadFiles)
	}

	root, err := os.OpenRoot(req.TargetPath)
	if err != nil {
		return nil, err
	}
	defer root.Close()

	result := &dto.FileUploadPreflightResp{
		Conflicts: []string{},
		Blocked:   []dto.FileUploadBlocked{},
	}
	seen := make(map[string]struct{}, len(req.RelativePaths))

	for _, raw := range req.RelativePaths {
		relativePath, pathErr := normalizeUploadRelativePath(raw)
		if pathErr != nil {
			result.Blocked = append(result.Blocked, dto.FileUploadBlocked{RelativePath: raw, Reason: "非法路径"})
			continue
		}
		if _, ok := seen[relativePath]; ok {
			result.Blocked = append(result.Blocked, dto.FileUploadBlocked{RelativePath: relativePath, Reason: "重复路径"})
			continue
		}
		seen[relativePath] = struct{}{}

		info, statErr := root.Lstat(relativePath)
		if errors.Is(statErr, fs.ErrNotExist) {
			continue
		}
		if statErr != nil {
			result.Blocked = append(result.Blocked, dto.FileUploadBlocked{RelativePath: relativePath, Reason: "目标路径不可访问"})
			continue
		}
		if !info.Mode().IsRegular() {
			result.Blocked = append(result.Blocked, dto.FileUploadBlocked{RelativePath: relativePath, Reason: "目标位置不是普通文件"})
			continue
		}
		result.Conflicts = append(result.Conflicts, relativePath)
	}

	return result, nil
}

func (s *FileService) SaveUpload(targetPath, relativePath string, overwrite bool, src io.Reader) (string, error) {
	relativePath, err := normalizeUploadRelativePath(relativePath)
	if err != nil {
		return "", err
	}

	root, err := os.OpenRoot(targetPath)
	if err != nil {
		return "", err
	}
	defer root.Close()

	parent := path.Dir(relativePath)
	if err := root.MkdirAll(parent, 0o755); err != nil {
		return "", err
	}

	if !overwrite {
		file, openErr := root.OpenFile(relativePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if errors.Is(openErr, fs.ErrExist) {
			return "", ErrUploadConflict
		}
		if openErr != nil {
			return "", openErr
		}
		if writeErr := copyAndCloseUpload(file, src); writeErr != nil {
			_ = root.Remove(relativePath)
			return "", writeErr
		}
		return relativePath, nil
	}

	mode, uid, gid, preserveOwner, err := uploadTargetMetadata(root, relativePath)
	if err != nil {
		return "", err
	}
	tempPath, file, err := createUploadTemp(root, parent, mode)
	if err != nil {
		return "", err
	}
	if writeErr := copyAndCloseUpload(file, src); writeErr != nil {
		_ = root.Remove(tempPath)
		return "", writeErr
	}
	if err := root.Chmod(tempPath, mode); err != nil {
		_ = root.Remove(tempPath)
		return "", err
	}
	if preserveOwner {
		if err := root.Chown(tempPath, uid, gid); err != nil {
			_ = root.Remove(tempPath)
			return "", err
		}
	}
	if err := root.Rename(tempPath, relativePath); err != nil {
		_ = root.Remove(tempPath)
		return "", err
	}
	return relativePath, nil
}

func (s *FileService) SaveUploadChunk(req dto.FileUploadChunkReq, src io.Reader) error {
	relativePath, expectedOffset, expectedSize, err := validateUploadChunk(req)
	if err != nil {
		return err
	}

	root, err := os.OpenRoot(req.TargetPath)
	if err != nil {
		return err
	}
	defer root.Close()

	parent := path.Dir(relativePath)
	if err := root.MkdirAll(parent, 0o755); err != nil {
		return err
	}
	if req.ChunkIndex == 0 {
		cleanupStaleUploadChunks(root, parent, time.Now().Add(-uploadChunkMaxAge))
	}
	tempPath, err := uploadChunkTempPath(relativePath, req.UploadID, req.TotalSize)
	if err != nil {
		return err
	}

	flags := os.O_WRONLY | syscall.O_NOFOLLOW
	if req.ChunkIndex == 0 {
		flags |= os.O_CREATE | os.O_EXCL
	}
	file, err := root.OpenFile(tempPath, flags, 0o600)
	if err != nil {
		if (req.ChunkIndex == 0 && errors.Is(err, fs.ErrExist)) ||
			(req.ChunkIndex > 0 && errors.Is(err, fs.ErrNotExist)) {
			return ErrUploadChunkOrder
		}
		return err
	}
	defer file.Close()
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX); err != nil {
		return err
	}
	defer syscall.Flock(int(file.Fd()), syscall.LOCK_UN)

	info, err := file.Stat()
	if err != nil {
		return err
	}
	if err := verifyUploadTempFile(root, tempPath, info); err != nil {
		return err
	}
	if !info.Mode().IsRegular() || info.Size() != expectedOffset {
		return ErrUploadChunkOrder
	}
	if _, err := file.Seek(expectedOffset, io.SeekStart); err != nil {
		return err
	}

	hasher := sha256.New()
	limited := &io.LimitedReader{R: src, N: expectedSize + 1}
	written, copyErr := io.Copy(io.MultiWriter(file, hasher), limited)
	rollback := func() {
		_ = file.Truncate(expectedOffset)
		_, _ = file.Seek(expectedOffset, io.SeekStart)
	}
	if copyErr != nil {
		rollback()
		return copyErr
	}
	if written != expectedSize {
		rollback()
		return ErrUploadSizeMismatch
	}
	want, _ := hex.DecodeString(req.Checksum)
	if subtle.ConstantTimeCompare(hasher.Sum(nil), want) != 1 {
		rollback()
		return ErrUploadChecksum
	}
	if err := file.Sync(); err != nil {
		rollback()
		return err
	}
	return nil
}

func (s *FileService) CompleteUploadChunks(req dto.FileUploadChunkCompleteReq) (string, error) {
	relativePath, err := validateChunkUploadTarget(req.RelativePath, req.UploadID, req.TotalSize)
	if err != nil {
		return "", err
	}
	tempPath, err := uploadChunkTempPath(relativePath, req.UploadID, req.TotalSize)
	if err != nil {
		return "", err
	}
	root, err := os.OpenRoot(req.TargetPath)
	if err != nil {
		return "", err
	}
	defer root.Close()

	file, err := root.OpenFile(tempPath, os.O_RDWR|syscall.O_NOFOLLOW, 0)
	if errors.Is(err, fs.ErrNotExist) {
		return "", ErrUploadChunkOrder
	}
	if err != nil {
		return "", err
	}
	removeTemp := true
	defer func() {
		_ = syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
		_ = file.Close()
		if removeTemp {
			_ = root.Remove(tempPath)
		}
	}()
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX); err != nil {
		return "", err
	}
	info, err := file.Stat()
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() || info.Size() != req.TotalSize {
		return "", ErrUploadSizeMismatch
	}

	mode, uid, gid, preserveOwner := os.FileMode(0o644), 0, 0, false
	if req.Overwrite {
		mode, uid, gid, preserveOwner, err = uploadTargetMetadata(root, relativePath)
		if err != nil {
			return "", err
		}
	} else {
		targetInfo, statErr := root.Lstat(relativePath)
		if statErr == nil {
			if !targetInfo.Mode().IsRegular() {
				return "", fmt.Errorf("%w %q: target is not a regular file", ErrInvalidUploadPath, relativePath)
			}
			return "", ErrUploadConflict
		}
		if !errors.Is(statErr, fs.ErrNotExist) {
			return "", statErr
		}
	}
	if err := file.Chmod(mode); err != nil {
		return "", err
	}
	if preserveOwner {
		if err := file.Chown(uid, gid); err != nil {
			return "", err
		}
	}
	if err := file.Sync(); err != nil {
		return "", err
	}

	if req.Overwrite {
		if err := root.Rename(tempPath, relativePath); err != nil {
			return "", err
		}
	} else {
		if err := root.Link(tempPath, relativePath); err != nil {
			if errors.Is(err, fs.ErrExist) {
				return "", ErrUploadConflict
			}
			return "", err
		}
		if err := root.Remove(tempPath); err != nil {
			_ = root.Remove(relativePath)
			return "", err
		}
	}
	removeTemp = false
	return relativePath, nil
}

func (s *FileService) AbortUploadChunks(req dto.FileUploadChunkAbortReq) error {
	relativePath, err := validateChunkUploadTarget(req.RelativePath, req.UploadID, req.TotalSize)
	if err != nil {
		return err
	}
	tempPath, err := uploadChunkTempPath(relativePath, req.UploadID, req.TotalSize)
	if err != nil {
		return err
	}
	root, err := os.OpenRoot(req.TargetPath)
	if err != nil {
		return err
	}
	defer root.Close()
	if err := root.Remove(tempPath); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}

func validateUploadChunk(req dto.FileUploadChunkReq) (string, int64, int64, error) {
	relativePath, err := validateChunkUploadTarget(req.RelativePath, req.UploadID, req.TotalSize)
	if err != nil {
		return "", 0, 0, err
	}
	expectedCount := int((req.TotalSize-1)/int64(uploadChunkSize) + 1)
	if req.ChunkCount != expectedCount || req.ChunkIndex < 0 || req.ChunkIndex >= req.ChunkCount {
		return "", 0, 0, ErrUploadChunkOrder
	}
	if err := validateLowerHex(req.Checksum, sha256.Size*2); err != nil {
		return "", 0, 0, fmt.Errorf("%w: invalid upload checksum: %v", ErrInvalidUploadMetadata, err)
	}
	offset := int64(req.ChunkIndex) * int64(uploadChunkSize)
	size := min(int64(uploadChunkSize), req.TotalSize-offset)
	return relativePath, offset, size, nil
}

func validateChunkUploadTarget(relativePath, uploadID string, totalSize int64) (string, error) {
	relativePath, err := normalizeUploadRelativePath(relativePath)
	if err != nil {
		return "", err
	}
	if err := validateUploadID(uploadID); err != nil {
		return "", err
	}
	if totalSize <= 0 {
		return "", ErrUploadSizeMismatch
	}
	return relativePath, nil
}

func validateUploadID(uploadID string) error {
	if err := validateLowerHex(uploadID, 32); err != nil {
		return fmt.Errorf("%w: invalid upload id: %v", ErrInvalidUploadMetadata, err)
	}
	return nil
}

func validateLowerHex(value string, length int) error {
	if len(value) != length {
		return errors.New("invalid hexadecimal length")
	}
	for _, char := range value {
		if (char < '0' || char > '9') && (char < 'a' || char > 'f') {
			return errors.New("invalid lowercase hexadecimal value")
		}
	}
	return nil
}

func uploadChunkTempPath(relativePath, uploadID string, totalSize int64) (string, error) {
	relativePath, err := normalizeUploadRelativePath(relativePath)
	if err != nil {
		return "", err
	}
	if err := validateUploadID(uploadID); err != nil {
		return "", err
	}
	if totalSize <= 0 {
		return "", ErrUploadSizeMismatch
	}
	digest := sha256.Sum256([]byte(fmt.Sprintf("%s\x00%s\x00%d", relativePath, uploadID, totalSize)))
	name := uploadChunkTempStart + hex.EncodeToString(digest[:16]) + uploadChunkTempEnd
	if parent := path.Dir(relativePath); parent != "." {
		name = path.Join(parent, name)
	}
	return name, nil
}

func verifyUploadTempFile(root *os.Root, tempPath string, openedInfo os.FileInfo) error {
	currentInfo, err := root.Lstat(tempPath)
	if err != nil || !currentInfo.Mode().IsRegular() || !os.SameFile(openedInfo, currentInfo) {
		return ErrUploadChunkOrder
	}
	return nil
}

func cleanupStaleUploadChunks(root *os.Root, parent string, before time.Time) {
	directory, err := root.Open(parent)
	if err != nil {
		return
	}
	defer directory.Close()
	entries, err := directory.ReadDir(-1)
	if err != nil {
		return
	}
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(name, uploadChunkTempStart) || !strings.HasSuffix(name, uploadChunkTempEnd) {
			continue
		}
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() || !info.ModTime().Before(before) {
			continue
		}
		if parent != "." {
			name = path.Join(parent, name)
		}
		_ = root.Remove(name)
	}
}

func uploadTargetMetadata(root *os.Root, relativePath string) (os.FileMode, int, int, bool, error) {
	info, err := root.Lstat(relativePath)
	if errors.Is(err, fs.ErrNotExist) {
		return 0o644, 0, 0, false, nil
	}
	if err != nil {
		return 0, 0, 0, false, err
	}
	if !info.Mode().IsRegular() {
		return 0, 0, 0, false, fmt.Errorf("%w %q: target is not a regular file", ErrInvalidUploadPath, relativePath)
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return info.Mode().Perm(), 0, 0, false, nil
	}
	return info.Mode().Perm(), int(stat.Uid), int(stat.Gid), true, nil
}

func createUploadTemp(root *os.Root, parent string, mode os.FileMode) (string, *os.File, error) {
	for range 10 {
		var suffix [8]byte
		if _, err := rand.Read(suffix[:]); err != nil {
			return "", nil, err
		}
		name := fmt.Sprintf(".xpanel-upload-%x.tmp", suffix)
		if parent != "." {
			name = path.Join(parent, name)
		}
		file, err := root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
		if errors.Is(err, fs.ErrExist) {
			continue
		}
		return name, file, err
	}
	return "", nil, errors.New("failed to allocate upload temporary file")
}

func copyAndCloseUpload(dst *os.File, src io.Reader) error {
	buffer := make([]byte, 32*1024)
	_, copyErr := io.CopyBuffer(dst, src, buffer)
	closeErr := dst.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}
