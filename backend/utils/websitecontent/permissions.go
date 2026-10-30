package websitecontent

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

const nginxGroup = "www-data"

type Policy struct {
	UID           int
	GID           int
	DirectoryMode os.FileMode
	FileMode      os.FileMode
}

func DefaultPolicy() Policy {
	policy := Policy{
		UID:           os.Geteuid(),
		GID:           os.Getegid(),
		DirectoryMode: 0o750,
		FileMode:      0o640,
	}
	if policy.UID != 0 {
		return policy
	}
	group, err := user.LookupGroup(nginxGroup)
	if err != nil {
		policy.DirectoryMode = 0o755
		policy.FileMode = 0o644
		return policy
	}
	gid, err := strconv.Atoi(group.Gid)
	if err != nil {
		policy.DirectoryMode = 0o755
		policy.FileMode = 0o644
		return policy
	}
	policy.GID = gid
	return policy
}

func EnsureDirectory(path string, policy Policy) error {
	clean := filepath.Clean(path)
	missing := make([]string, 0)
	for current := clean; ; current = filepath.Dir(current) {
		info, err := os.Lstat(current)
		if err == nil {
			if !info.IsDir() {
				return fmt.Errorf("website path is not a directory: %s", current)
			}
			break
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		missing = append(missing, current)
		parent := filepath.Dir(current)
		if parent == current {
			return fmt.Errorf("website path has no existing parent: %s", clean)
		}
	}
	if err := os.MkdirAll(clean, policy.DirectoryMode); err != nil {
		return err
	}
	for index := len(missing) - 1; index >= 0; index-- {
		if err := ApplyDirectory(missing[index], policy); err != nil {
			return err
		}
	}
	return nil
}

func CreateFileIfMissing(path string, data []byte, policy Policy) (bool, error) {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL|syscall.O_NOFOLLOW, policy.FileMode)
	if errors.Is(err, fs.ErrExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	remove := true
	defer func() {
		_ = file.Close()
		if remove {
			_ = os.Remove(path)
		}
	}()
	if _, err := file.Write(data); err != nil {
		return false, err
	}
	if err := file.Chown(policy.UID, policy.GID); err != nil {
		return false, err
	}
	if err := file.Chmod(policy.FileMode); err != nil {
		return false, err
	}
	if err := file.Close(); err != nil {
		return false, err
	}
	remove = false
	return true, nil
}

func ApplyDirectory(path string, policy Policy) error {
	if err := os.Chown(path, policy.UID, policy.GID); err != nil {
		return err
	}
	return os.Chmod(path, policy.DirectoryMode)
}

func ApplyFile(path string, policy Policy) error {
	if err := os.Chown(path, policy.UID, policy.GID); err != nil {
		return err
	}
	return os.Chmod(path, policy.FileMode)
}

func Contains(root, candidate string) bool {
	relative, err := filepath.Rel(filepath.Clean(root), filepath.Clean(candidate))
	return err == nil && relative != ".." && !filepath.IsAbs(relative) && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}
