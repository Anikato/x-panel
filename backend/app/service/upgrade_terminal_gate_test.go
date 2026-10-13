package service

import (
	"errors"
	"os/exec"
	"path/filepath"
	"testing"

	"xpanel/app/dto"

	"github.com/creack/pty"
)

func TestDoUpgradeRefusesWhileBackgroundTerminalIsRunning(t *testing.T) {
	resetTerminalSessionsForTest(t)
	cmd := exec.Command("sleep", "60")
	ptmx, err := pty.Start(cmd)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := AdoptTerminalSession(ptmx, cmd, "编译", "/src"); err != nil {
		t.Fatal(err)
	}

	upgradeMu.Lock()
	previous := upgrading
	upgrading = false
	upgradeMu.Unlock()
	t.Cleanup(func() {
		upgradeMu.Lock()
		upgrading = previous
		upgradeMu.Unlock()
	})

	called := 0
	svc := &UpgradeService{
		lockPath: filepath.Join(t.TempDir(), ".upgrade.lock"),
		logPath:  filepath.Join(t.TempDir(), "upgrade.log"),
		runUpgradeFn: func(dto.UpgradeReq, func(string, ...any)) error {
			called++
			return nil
		},
	}
	req := dto.UpgradeReq{
		Version:     "v9.9.9",
		DownloadURL: "https://updates.example/pkg",
		ChecksumURL: "https://updates.example/pkg.sha256",
	}
	err = svc.DoUpgrade(req)
	var blocked *BackgroundTerminalsError
	if !errors.As(err, &blocked) {
		t.Fatalf("DoUpgrade error = %v", err)
	}
	if called != 0 {
		t.Fatal("upgrade core ran while a background terminal was alive")
	}
	upgradeMu.Lock()
	still := upgrading
	upgradeMu.Unlock()
	if still {
		t.Fatal("upgrading flag stayed set after refusal")
	}

	if err := svc.DoUpgradeSync(req); !errors.As(err, &blocked) {
		t.Fatalf("DoUpgradeSync error = %v", err)
	}
	if called != 0 {
		t.Fatal("sync upgrade core ran while a background terminal was alive")
	}
}
