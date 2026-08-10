package service

import (
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
