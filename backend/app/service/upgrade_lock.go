package service

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"xpanel/buserr"
	"xpanel/constant"
)

type upgradeFileLock struct {
	file *os.File
}

func acquireUpgradeFileLock(path string) (*upgradeFileLock, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open upgrade lock: %w", err)
	}
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = file.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN) {
			return nil, buserr.New(constant.ErrUpgradeInProgress)
		}
		return nil, fmt.Errorf("lock upgrade file: %w", err)
	}
	return &upgradeFileLock{file: file}, nil
}

func (l *upgradeFileLock) Release() error {
	if l == nil || l.file == nil {
		return nil
	}
	file := l.file
	l.file = nil
	unlockErr := syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
	closeErr := file.Close()
	if unlockErr != nil {
		return unlockErr
	}
	return closeErr
}

func defaultUpgradeLockPath() (string, error) {
	execPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("resolve executable for upgrade lock: %w", err)
	}
	if resolved, err := filepath.EvalSymlinks(execPath); err == nil {
		execPath = resolved
	}
	return filepath.Join(filepath.Dir(execPath), ".upgrade.lock"), nil
}
