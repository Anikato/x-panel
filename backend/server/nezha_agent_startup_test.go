package server

import (
	"bytes"
	"errors"
	"strings"
	"sync/atomic"
	"testing"

	"xpanel/global"

	"github.com/sirupsen/logrus"
)

func TestNezhaAgentStartupSyncCallsOnceAndSucceedsSilently(t *testing.T) {
	restoreNezhaAgentStartupSeam(t)

	var calls atomic.Int32
	syncNezhaAgentSettingsOnStartup = func() error {
		calls.Add(1)
		return nil
	}

	var buf bytes.Buffer
	logger := logrus.New()
	logger.SetOutput(&buf)
	logger.SetLevel(logrus.WarnLevel)
	previousLOG := global.LOG
	global.LOG = logger
	t.Cleanup(func() { global.LOG = previousLOG })

	runNezhaAgentStartupSync()

	if got := calls.Load(); got != 1 {
		t.Fatalf("startup sync calls = %d, want 1", got)
	}
	if strings.Contains(strings.ToLower(buf.String()), "warn") {
		t.Fatalf("successful sync must not warn: %q", buf.String())
	}
}

func TestNezhaAgentStartupSyncFailureWarnsWithoutPanicOrSecret(t *testing.T) {
	restoreNezhaAgentStartupSeam(t)

	const secret = "SENTINEL_NEZHA_STARTUP_SECRET_7b91"
	var calls atomic.Int32
	syncNezhaAgentSettingsOnStartup = func() error {
		calls.Add(1)
		return errors.New("sync failed: " + secret)
	}

	var buf bytes.Buffer
	logger := logrus.New()
	logger.SetOutput(&buf)
	logger.SetLevel(logrus.WarnLevel)
	previousLOG := global.LOG
	global.LOG = logger
	t.Cleanup(func() { global.LOG = previousLOG })

	// Must not panic; failure only emits a safe warning.
	runNezhaAgentStartupSync()

	if got := calls.Load(); got != 1 {
		t.Fatalf("startup sync calls = %d, want 1", got)
	}
	logged := buf.String()
	if logged == "" {
		t.Fatal("expected a warning log on sync failure")
	}
	if !strings.Contains(strings.ToLower(logged), "warn") &&
		!strings.Contains(strings.ToLower(logged), "nezha") {
		t.Fatalf("expected nezha warning log, got %q", logged)
	}
	if strings.Contains(logged, secret) {
		t.Fatalf("startup warning leaked secret material: %q", logged)
	}
	// Callback seam must not itself start/enable the agent; production callback
	// only SyncConfigToSettings. Failure path must not invent Operate/systemctl.
	if strings.Contains(logged, "systemctl") || strings.Contains(logged, "Operate") {
		t.Fatalf("failure path must not mention agent lifecycle ops: %q", logged)
	}
}

func restoreNezhaAgentStartupSeam(t *testing.T) {
	t.Helper()
	previous := syncNezhaAgentSettingsOnStartup
	t.Cleanup(func() {
		syncNezhaAgentSettingsOnStartup = previous
	})
}
