package service

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"strings"
	"syscall"

	"xpanel/app/dto"
)

const maxUploadFiles = 1000

var ErrUploadConflict = errors.New("upload target already exists")
var ErrInvalidUploadPath = errors.New("invalid upload path")

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
