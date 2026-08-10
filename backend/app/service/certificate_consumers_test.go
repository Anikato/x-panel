package service

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"xpanel/app/model"
	"xpanel/global"

	"github.com/glebarez/sqlite"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func installFakeNginx(t *testing.T, running bool) string {
	t.Helper()
	root := t.TempDir()
	callsPath := filepath.Join(root, "calls.log")
	for _, dir := range []string{filepath.Join(root, "sbin"), filepath.Join(root, "conf"), filepath.Join(root, "logs")} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("create fake nginx dir: %v", err)
		}
	}
	script := fmt.Sprintf("#!/bin/sh\nprintf '%%s\\n' \"$*\" >> %q\n", callsPath)
	if err := os.WriteFile(filepath.Join(root, "sbin", "nginx"), []byte(script), 0o755); err != nil {
		t.Fatalf("write fake nginx: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "conf", "nginx.conf"), []byte("events {}\n"), 0o644); err != nil {
		t.Fatalf("write fake nginx config: %v", err)
	}
	if running {
		if err := os.WriteFile(filepath.Join(root, "logs", "nginx.pid"), []byte(fmt.Sprintf("%d", os.Getpid())), 0o644); err != nil {
			t.Fatalf("write fake nginx pid: %v", err)
		}
	}
	previousConf, previousLog := global.CONF, global.LOG
	global.CONF.Nginx = global.NginxConfig{InstallDir: root, Mode: "prefix"}
	global.CONF.Nginx.DetectNginx()
	global.LOG = logrus.New()
	t.Cleanup(func() {
		global.CONF = previousConf
		global.LOG = previousLog
	})
	return callsPath
}

func installCertificateConsumerDatabase(t *testing.T) {
	t.Helper()
	database, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := database.AutoMigrate(&model.Website{}, &model.HAProxyLB{}, &model.GostService{}); err != nil {
		t.Fatalf("migrate consumers: %v", err)
	}
	previous := global.DB
	global.DB = database
	t.Cleanup(func() { global.DB = previous })
}

func TestReloadNginxGlobalTestsConfigBeforeReload(t *testing.T) {
	callsPath := installFakeNginx(t, true)
	if err := reloadNginxGlobal(); err != nil {
		t.Fatalf("reload nginx: %v", err)
	}
	content, err := os.ReadFile(callsPath)
	if err != nil {
		t.Fatalf("read fake nginx calls: %v", err)
	}
	lines := strings.FieldsFunc(string(content), func(r rune) bool { return r == '\n' })
	if len(lines) != 2 || !strings.Contains(lines[0], "-t") || !strings.Contains(lines[1], "-s reload") {
		t.Fatalf("nginx calls = %#v, want config test before reload", lines)
	}
}

func TestFindCertificateConsumerTargetsSelectsRunningNginxWithoutWebsiteRows(t *testing.T) {
	installCertificateConsumerDatabase(t)
	installFakeNginx(t, true)
	targets, err := findCertificateConsumerTargets([]uint{7})
	if err != nil {
		t.Fatalf("find consumers: %v", err)
	}
	if !targets.Nginx {
		t.Fatalf("targets = %#v, want running nginx", targets)
	}
}

func TestFindCertificateConsumerTargetsSkipsStoppedNginx(t *testing.T) {
	installCertificateConsumerDatabase(t)
	installFakeNginx(t, false)
	targets, err := findCertificateConsumerTargets([]uint{7})
	if err != nil {
		t.Fatalf("find consumers: %v", err)
	}
	if targets.Nginx {
		t.Fatalf("targets = %#v, stopped nginx must not be selected", targets)
	}
}

func TestRefreshCertificateConsumersRunsRemainingTargetsAfterFailure(t *testing.T) {
	var calls []string
	err := refreshCertificateConsumers(certificateConsumerTargets{
		Nginx:   true,
		HAProxy: true,
		GOST:    true,
	}, certificateConsumerRefreshActions{
		ReloadNginx: func() error {
			calls = append(calls, "nginx")
			return nil
		},
		ReloadHAProxy: func() error {
			calls = append(calls, "haproxy")
			return errors.New("reload failed")
		},
		ReloadGOST: func() error {
			calls = append(calls, "gost")
			return nil
		},
	})
	if err == nil {
		t.Fatal("expected a consumer refresh error")
	}
	if want := []string{"nginx", "haproxy", "gost"}; !reflect.DeepEqual(calls, want) {
		t.Fatalf("expected all selected consumers to be attempted, got %v", calls)
	}
}

func TestRefreshCertificateConsumersVerifiesAfterReloadAndContinuesAfterVerifyFailure(t *testing.T) {
	var calls []string
	err := refreshCertificateConsumers(certificateConsumerTargets{Nginx: true, HAProxy: true, GOST: true}, certificateConsumerRefreshActions{
		ReloadNginx: func() error {
			calls = append(calls, "reload")
			return nil
		},
		VerifyNginx: func() error {
			calls = append(calls, "verify")
			return errors.New("fingerprint mismatch")
		},
		ReloadHAProxy: func() error {
			calls = append(calls, "haproxy")
			return nil
		},
		ReloadGOST: func() error {
			calls = append(calls, "gost")
			return nil
		},
	})
	if err == nil || !strings.Contains(err.Error(), "fingerprint mismatch") {
		t.Fatalf("refresh error = %v", err)
	}
	want := []string{"reload", "verify", "haproxy", "gost"}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("calls = %#v, want %#v", calls, want)
	}
}

func TestCertificateSyncPostActionsRefreshBeforeCustomCommand(t *testing.T) {
	var calls []string
	refreshErr := errors.New("refresh failed")
	err := runCertificateSyncPostActions([]uint{7}, "custom command",
		func(ids []uint) error {
			if !reflect.DeepEqual(ids, []uint{7}) {
				t.Fatalf("unexpected certificate IDs: %v", ids)
			}
			calls = append(calls, "refresh")
			return refreshErr
		},
		func(command string) error {
			if command != "custom command" {
				t.Fatalf("unexpected command: %s", command)
			}
			calls = append(calls, "command")
			return nil
		},
	)
	if !errors.Is(err, refreshErr) {
		t.Fatalf("expected refresh error, got %v", err)
	}
	if want := []string{"refresh", "command"}; !reflect.DeepEqual(calls, want) {
		t.Fatalf("expected refresh before command, got %v", calls)
	}
}

func TestCertificateSyncStatusKeepsRefreshFailureRetryable(t *testing.T) {
	status := certificateSyncStatus(1, 0, 0, errors.New("consumer refresh failed"))
	if status != "warning" {
		t.Fatalf("expected refresh failure to remain retryable as warning, got %q", status)
	}
}
