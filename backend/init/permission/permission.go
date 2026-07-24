package permission

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

const (
	privateDirectoryMode = 0700
	privateFileMode      = 0600
)

type Paths struct {
	Directories []string
	Files       []string
}

func SetProcessUmask() {
	syscall.Umask(0077)
}

func Harden(paths Paths) error {
	for _, path := range paths.Directories {
		if path == "" {
			continue
		}
		if err := EnsurePrivateDirectory(path); err != nil {
			return err
		}
	}
	for _, path := range paths.Files {
		if path == "" {
			continue
		}
		if err := HardenPrivateFileIfExists(path); err != nil {
			return err
		}
	}
	return nil
}

// HardenOwned limits permission changes to paths owned by X-Panel. Both the
// configured path and its resolved parent chain must stay below managedRoot.
func HardenOwned(managedRoot string, paths Paths) error {
	if strings.TrimSpace(managedRoot) == "" {
		return errors.New("managed root is empty")
	}
	root, err := filepath.Abs(filepath.Clean(managedRoot))
	if err != nil {
		return fmt.Errorf("resolve managed root %s: %w", managedRoot, err)
	}
	resolvedRoot, err := resolvePathWithExistingAncestor(root, false)
	if err != nil {
		return fmt.Errorf("resolve managed root %s: %w", root, err)
	}

	validated := Paths{
		Directories: make([]string, 0, len(paths.Directories)),
		Files:       make([]string, 0, len(paths.Files)),
	}
	for _, path := range paths.Directories {
		if path == "" {
			continue
		}
		if err := validateOwnedPath(root, resolvedRoot, path); err != nil {
			return err
		}
		validated.Directories = append(validated.Directories, path)
	}
	for _, path := range paths.Files {
		if path == "" {
			continue
		}
		if err := validateOwnedPath(root, resolvedRoot, path); err != nil {
			return err
		}
		validated.Files = append(validated.Files, path)
	}
	return Harden(validated)
}

func HardenRuntime(dataDir, dbPath, logPath, keyPath, configPath string) error {
	var paths Paths
	installRoot := ResolveInstallRoot(dataDir)
	if dataDir != "" {
		paths.Directories = append(paths.Directories, dataDir, filepath.Join(installRoot, "backups"))
	}
	if dbPath != "" {
		paths.Directories = append(paths.Directories, filepath.Dir(dbPath))
		paths.Files = append(paths.Files, dbPath, dbPath+"-wal", dbPath+"-shm")
	}
	if logPath != "" {
		paths.Directories = append(paths.Directories, logPath)
		paths.Files = append(paths.Files, filepath.Join(logPath, "xpanel.log"))
	}
	if keyPath != "" {
		paths.Directories = append(paths.Directories, filepath.Dir(keyPath))
		paths.Files = append(paths.Files, keyPath)
	}
	if configPath != "" {
		paths.Files = append(paths.Files, configPath)
	}
	return HardenOwned(installRoot, paths)
}

// ResolveInstallRoot preserves the legacy <install>/data layout without ever
// promoting a top-level /data directory to the filesystem root.
func ResolveInstallRoot(dataDir string) string {
	root := filepath.Clean(dataDir)
	if filepath.Base(root) != "data" {
		return root
	}
	parent := filepath.Dir(root)
	if filepath.IsAbs(parent) && filepath.Dir(parent) == parent {
		return root
	}
	return parent
}

func validateOwnedPath(root, resolvedRoot, path string) error {
	absolute, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return fmt.Errorf("resolve managed path %s: %w", path, err)
	}
	if !pathWithin(root, absolute) {
		return fmt.Errorf("path is outside managed root %s: %s", root, absolute)
	}

	info, err := os.Lstat(absolute)
	if err == nil && info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("managed path must not be a symlink: %s", absolute)
	}
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect managed path %s: %w", absolute, err)
	}

	resolved, err := resolvePathWithExistingAncestor(absolute, true)
	if err != nil {
		return fmt.Errorf("resolve managed path %s: %w", absolute, err)
	}
	if !pathWithin(resolvedRoot, resolved) {
		return fmt.Errorf("path escapes managed root %s through a symlink: %s", root, absolute)
	}
	return nil
}

func resolvePathWithExistingAncestor(path string, rejectFinalSymlink bool) (string, error) {
	current := filepath.Clean(path)
	for {
		info, err := os.Lstat(current)
		if err == nil {
			if rejectFinalSymlink && current == filepath.Clean(path) && info.Mode()&os.ModeSymlink != 0 {
				return "", fmt.Errorf("path must not be a symlink: %s", path)
			}
			resolved, err := filepath.EvalSymlinks(current)
			if err != nil {
				return "", err
			}
			relative, err := filepath.Rel(current, filepath.Clean(path))
			if err != nil {
				return "", err
			}
			return filepath.Clean(filepath.Join(resolved, relative)), nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", fmt.Errorf("no existing ancestor for %s", path)
		}
		current = parent
	}
}

func pathWithin(root, path string) bool {
	relative, err := filepath.Rel(root, path)
	if err != nil || filepath.IsAbs(relative) {
		return false
	}
	return relative != ".." && !strings.HasPrefix(relative, ".."+string(os.PathSeparator))
}

func EnsurePrivateDirectory(path string) error {
	if err := os.MkdirAll(path, privateDirectoryMode); err != nil {
		return fmt.Errorf("create private directory %s: %w", path, err)
	}
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("inspect private directory %s: %w", path, err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("private directory has unsafe type: %s", path)
	}
	if err := os.Chmod(path, privateDirectoryMode); err != nil {
		return fmt.Errorf("harden private directory %s: %w", path, err)
	}
	return nil
}

func HardenPrivateFileIfExists(path string) error {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect private file %s: %w", path, err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return fmt.Errorf("private file has unsafe type: %s", path)
	}
	if err := os.Chmod(path, privateFileMode); err != nil {
		return fmt.Errorf("harden private file %s: %w", path, err)
	}
	return nil
}
