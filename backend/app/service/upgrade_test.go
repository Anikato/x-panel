package service

import (
	"errors"
	"path/filepath"
	"testing"

	"xpanel/app/dto"
)

// TestDoUpgradeRequiresChecksumURL is a standalone component-upgrade contract:
// missing checksum URL must be rejected synchronously before any background work.
func TestDoUpgradeRequiresChecksumURL(t *testing.T) {
	upgradeMu.Lock()
	previousUpgrading := upgrading
	upgrading = false
	upgradeMu.Unlock()
	t.Cleanup(func() {
		upgradeMu.Lock()
		upgrading = previousUpgrading
		upgradeMu.Unlock()
	})

	err := (&UpgradeService{}).DoUpgrade(dto.UpgradeReq{
		Version:     "v9.9.9",
		DownloadURL: "https://updates.example.com/xpanel-v9.9.9-linux-amd64.tar.gz",
		// ChecksumURL intentionally omitted — component packages require it.
	})
	if err == nil {
		t.Fatal("DoUpgrade() error = nil, want checksum URL required rejection")
	}
	upgradeMu.Lock()
	stillUpgrading := upgrading
	upgradeMu.Unlock()
	if stillUpgrading {
		t.Fatal("upgrading = true after checksum URL rejection")
	}
}

func TestUpgradeFileLockRejectsSecondHolder(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".upgrade.lock")
	first, err := acquireUpgradeFileLock(path)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Release()

	if _, err := acquireUpgradeFileLock(path); err == nil {
		t.Fatal("second lock acquisition succeeded")
	}
	if err := first.Release(); err != nil {
		t.Fatal(err)
	}
	second, err := acquireUpgradeFileLock(path)
	if err != nil {
		t.Fatalf("lock after release: %v", err)
	}
	defer second.Release()
}

func TestDoUpgradeSyncReturnsCoreFailureAndReleasesLock(t *testing.T) {
	want := errors.New("apply failed")
	path := filepath.Join(t.TempDir(), ".upgrade.lock")
	svc := &UpgradeService{
		lockPath: path,
		logPath:  filepath.Join(t.TempDir(), "upgrade.log"),
		runUpgradeFn: func(req dto.UpgradeReq, _ func(string, ...any)) error {
			if req.Version != "v9.9.9" {
				t.Fatalf("version = %q", req.Version)
			}
			return want
		},
	}
	req := dto.UpgradeReq{Version: "v9.9.9", DownloadURL: "https://updates.example/pkg", ChecksumURL: "https://updates.example/pkg.sha256"}
	if err := svc.DoUpgradeSync(req); !errors.Is(err, want) {
		t.Fatalf("DoUpgradeSync error = %v, want %v", err, want)
	}
	lock, err := acquireUpgradeFileLock(path)
	if err != nil {
		t.Fatalf("lock not released: %v", err)
	}
	defer lock.Release()
}

func TestUpgradeLatestUsesResolvedReleaseAndNoopsWhenCurrent(t *testing.T) {
	called := 0
	svc := &UpgradeService{
		lockPath: filepath.Join(t.TempDir(), ".upgrade.lock"),
		checkUpdateFn: func(dto.UpgradeCheckReq) (*dto.UpgradeInfo, error) {
			return &dto.UpgradeInfo{HasUpdate: false, LatestVersion: "v1.0.0"}, nil
		},
		runUpgradeFn: func(dto.UpgradeReq, func(string, ...any)) error {
			called++
			return nil
		},
	}
	if err := svc.UpgradeLatest(); err != nil {
		t.Fatal(err)
	}
	if called != 0 {
		t.Fatalf("upgrade core called %d times for current version", called)
	}
}

func TestUpgradeLatestPassesResolvedAssetToSyncCore(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".upgrade.lock")
	logPath := filepath.Join(t.TempDir(), "upgrade.log")
	called := 0
	svc := &UpgradeService{
		lockPath: path,
		logPath:  logPath,
		checkUpdateFn: func(dto.UpgradeCheckReq) (*dto.UpgradeInfo, error) {
			return &dto.UpgradeInfo{
				HasUpdate:     true,
				LatestVersion: "v1.2.3.1",
				DownloadURL:   "https://updates.example/xpanel.tar.gz",
				ChecksumURL:   "https://updates.example/xpanel.tar.gz.sha256",
			}, nil
		},
		runUpgradeFn: func(req dto.UpgradeReq, _ func(string, ...any)) error {
			called++
			if req.Version != "v1.2.3.1" || req.DownloadURL != "https://updates.example/xpanel.tar.gz" || req.ChecksumURL != "https://updates.example/xpanel.tar.gz.sha256" {
				t.Fatalf("upgrade request = %#v", req)
			}
			return nil
		},
	}
	if err := svc.UpgradeLatest(); err != nil {
		t.Fatal(err)
	}
	if called != 1 {
		t.Fatalf("upgrade core called %d times", called)
	}
}
