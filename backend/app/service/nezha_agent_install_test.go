package service

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestExactAgentPackageURLs(t *testing.T) {
	downloadURL, checksumURL := exactAgentPackageURLs("https://updates.example.com/", "v0.7.79", "amd64")
	want := "https://updates.example.com/releases/v0.7.79/xpanel-v0.7.79-linux-amd64.tar.gz"
	if downloadURL != want || checksumURL != want+".sha256" {
		t.Fatalf("urls = %q, %q; want %q, %q", downloadURL, checksumURL, want, want+".sha256")
	}
}

func TestExtractNezhaAgentBundleOnlyRequiredAssets(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "release.tar.gz")
	agentBody := elfWithMarker(t, runtime.GOARCH, "agent")
	unitBody := []byte("[Unit]\nDescription=test\n")
	writeGzipTar(t, archive, []tarEntry{
		{Name: "xpanel", Body: []byte("must-not-extract"), Mode: 0o755},
		{Name: "nezha-agent/nezha-agent", Body: agentBody, Mode: 0o755},
		{Name: "xpanel-nezha-agent.service", Body: unitBody, Mode: 0o644},
	})

	agentPath, unitPath, err := extractNezhaAgentBundle(archive, filepath.Join(dir, "extract"))
	if err != nil {
		t.Fatalf("extractNezhaAgentBundle: %v", err)
	}
	if got, _ := os.ReadFile(agentPath); string(got) != string(agentBody) {
		t.Fatal("agent content mismatch")
	}
	if got, _ := os.ReadFile(unitPath); string(got) != string(unitBody) {
		t.Fatal("unit content mismatch")
	}
	if _, err := os.Stat(filepath.Join(dir, "extract", "xpanel")); !os.IsNotExist(err) {
		t.Fatal("xpanel binary must not be extracted")
	}
}

func TestExtractNezhaAgentBundleRejectsTraversal(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "release.tar.gz")
	writeGzipTar(t, archive, []tarEntry{
		{Name: "../escape", Body: []byte("bad"), Mode: 0o644},
		{Name: "nezha-agent/nezha-agent", Body: elfWithMarker(t, runtime.GOARCH, "agent"), Mode: 0o755},
		{Name: "xpanel-nezha-agent.service", Body: []byte("unit"), Mode: 0o644},
	})
	if _, _, err := extractNezhaAgentBundle(archive, filepath.Join(dir, "extract")); err == nil {
		t.Fatal("expected traversal rejection")
	}
}

func TestApplyNezhaAgentBundleInstallsModesWithoutStarting(t *testing.T) {
	root := t.TempDir()
	archive := filepath.Join(root, "release.tar.gz")
	writeGzipTar(t, archive, []tarEntry{
		{Name: "nezha-agent/nezha-agent", Body: elfWithMarker(t, runtime.GOARCH, "agent"), Mode: 0o755},
		{Name: "xpanel-nezha-agent.service", Body: []byte("[Unit]\n"), Mode: 0o644},
	})
	runner := newFakeNezhaRunner("inactive", "not-found")
	agentPath := filepath.Join(root, "opt", "xpanel", "nezha-agent", "nezha-agent")
	unitPath := filepath.Join(root, "systemd", "xpanel-nezha-agent.service")
	if err := applyNezhaAgentBundle(agentBundleInstallDeps{
		AgentPath: agentPath,
		UnitPath:  unitPath,
		Runner:    runner,
	}, archive); err != nil {
		t.Fatalf("applyNezhaAgentBundle: %v", err)
	}

	assertMode := func(path string, want os.FileMode) {
		t.Helper()
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != want {
			t.Fatalf("%s mode = %o, want %o", path, info.Mode().Perm(), want)
		}
	}
	assertMode(filepath.Dir(agentPath), 0o700)
	assertMode(agentPath, 0o755)
	assertMode(unitPath, 0o644)
	runner.assertCalled(t, "systemctl", "daemon-reload")
	runner.assertNotCalled(t, "start")
	runner.assertNotCalled(t, "enable")
}

func TestApplyNezhaAgentBundleRollsBackWhenDaemonReloadFails(t *testing.T) {
	root := t.TempDir()
	archive := filepath.Join(root, "release.tar.gz")
	writeGzipTar(t, archive, []tarEntry{
		{Name: "nezha-agent/nezha-agent", Body: elfWithMarker(t, runtime.GOARCH, "new"), Mode: 0o755},
		{Name: "xpanel-nezha-agent.service", Body: []byte("new-unit"), Mode: 0o644},
	})
	agentPath := filepath.Join(root, "agent", "nezha-agent")
	unitPath := filepath.Join(root, "systemd", "xpanel-nezha-agent.service")
	if err := os.MkdirAll(filepath.Dir(agentPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(unitPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(agentPath, []byte("old-agent"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(unitPath, []byte("old-unit"), 0o644); err != nil {
		t.Fatal(err)
	}
	runner := newFakeNezhaRunner("inactive", "disabled")
	runner.failOps["daemon-reload"] = errors.New("reload failed")
	if err := applyNezhaAgentBundle(agentBundleInstallDeps{AgentPath: agentPath, UnitPath: unitPath, Runner: runner}, archive); err == nil {
		t.Fatal("expected daemon-reload failure")
	}
	if got, _ := os.ReadFile(agentPath); string(got) != "old-agent" {
		t.Fatalf("agent after rollback = %q", got)
	}
	if got, _ := os.ReadFile(unitPath); string(got) != "old-unit" {
		t.Fatalf("unit after rollback = %q", got)
	}
}

func TestInstallRejectsExternalConflictBeforeDownload(t *testing.T) {
	runner := newFakeNezhaRunner("inactive", "not-found")
	runner.listUnitsOutput = "nezha-agent.service loaded active running\n"
	downloadCalled := false
	svc := newNezhaAgentService(nezhaAgentDeps{
		BinaryPath: filepath.Join(t.TempDir(), "nezha-agent"),
		UnitPath:   filepath.Join(t.TempDir(), "xpanel-nezha-agent.service"),
		Runner:     runner,
		Settings:   newFakeNezhaSettings(nil),
		InstallBundle: func() (string, func(), error) {
			downloadCalled = true
			return "", func() {}, nil
		},
	})
	if err := svc.Install(); err == nil || !strings.Contains(err.Error(), "conflict") {
		t.Fatalf("Install error = %v, want conflict", err)
	}
	if downloadCalled {
		t.Fatal("release download must not run before conflict rejection")
	}
}

func TestInstallPropagatesBundleFailure(t *testing.T) {
	runner := newFakeNezhaRunner("inactive", "not-found")
	want := errors.New("download failed")
	svc := newNezhaAgentService(nezhaAgentDeps{
		BinaryPath: filepath.Join(t.TempDir(), "nezha-agent"),
		UnitPath:   filepath.Join(t.TempDir(), "xpanel-nezha-agent.service"),
		Runner:     runner,
		Settings:   newFakeNezhaSettings(nil),
		InstallBundle: func() (string, func(), error) {
			return "", func() {}, want
		},
	})
	if err := svc.Install(); !errors.Is(err, want) {
		t.Fatalf("Install error = %v, want %v", err, want)
	}
}

func TestInstallRepairsRunningAgentAndRestoresRuntimeState(t *testing.T) {
	root := t.TempDir()
	archive := filepath.Join(root, "release.tar.gz")
	writeGzipTar(t, archive, []tarEntry{
		{Name: "nezha-agent/nezha-agent", Body: elfWithMarker(t, runtime.GOARCH, "new-agent"), Mode: 0o755},
		{Name: "xpanel-nezha-agent.service", Body: []byte("[Unit]\nDescription=new\n"), Mode: 0o644},
	})
	agentPath := filepath.Join(root, "agent", "nezha-agent")
	unitPath := filepath.Join(root, "systemd", "xpanel-nezha-agent.service")
	configPath := filepath.Join(root, "agent", "config.yml")
	if err := os.MkdirAll(filepath.Dir(agentPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(unitPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(agentPath, elfWithMarker(t, runtime.GOARCH, "old-agent"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(unitPath, []byte("[Unit]\nDescription=old\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, []byte("uuid: keep-me\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	runner := newFakeNezhaRunner("active", "enabled")
	svc := newNezhaAgentService(nezhaAgentDeps{
		ConfigPath: configPath,
		BinaryPath: agentPath,
		UnitPath:   unitPath,
		Runner:     runner,
		Settings:   newFakeNezhaSettings(nil),
		Sleep:      func(time.Duration) {},
		Now:        time.Now,
		InstallBundle: func() (string, func(), error) {
			return archive, func() {}, nil
		},
	})

	if err := svc.Install(); err != nil {
		t.Fatalf("Install: %v", err)
	}
	runner.assertCalled(t, "systemctl", "stop", NezhaAgentUnitName)
	runner.assertCalled(t, "systemctl", "start", NezhaAgentUnitName)
	if runner.active != "active" || runner.enabled != "enabled" {
		t.Fatalf("state after repair = active:%s enabled:%s", runner.active, runner.enabled)
	}
	if got, err := os.ReadFile(configPath); err != nil || string(got) != "uuid: keep-me\n" {
		t.Fatalf("config after repair = %q, %v", got, err)
	}
}

func TestStatusTreatsMissingUnitAsComponentMissingWithoutNotFoundFault(t *testing.T) {
	configPath, binaryPath := nezhaStatusFixture(t, 0o700, 0o755, 0o600, nil)
	unitPath := filepath.Join(filepath.Dir(binaryPath), "xpanel-nezha-agent.service")
	if err := os.Remove(unitPath); err != nil {
		t.Fatal(err)
	}
	svc := newStatusNezhaService(t, configPath, binaryPath, newFakeNezhaRunner("inactive", "not-found"), nil)
	status, err := svc.Status()
	if err != nil {
		t.Fatal(err)
	}
	if status.ComponentAvailable {
		t.Fatal("component must be unavailable when unit is missing")
	}
	if strings.Contains(strings.ToLower(status.ServiceError), "not-found") {
		t.Fatalf("duplicate not-found service fault: %q", status.ServiceError)
	}
}
