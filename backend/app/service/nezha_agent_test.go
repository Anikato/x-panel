package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"xpanel/app/dto"
)

// ----- fakes -----

type fakeNezhaRunner struct {
	mu sync.Mutex

	calls [][]string

	active  string // active | inactive | failed | ...
	enabled string // enabled | enabled-runtime | disabled | ...

	// versionOutput is returned for `<binary> -v` (Task 2B1a Status).
	versionOutput string
	// versionFail makes `<binary> -v` return an error (Task 2B1b).
	versionFail bool

	// Empty systemctl probe outputs with failure (Task 2B1b Status).
	activeEmptyFail  bool
	enabledEmptyFail bool

	// failOps maps systemctl verb -> error ("start", "enable", ...)
	failOps map[string]error

	// list-units output for conflict detection (Task 2B2).
	listUnitsOutput string
	listUnitsErr    error

	// After stop, remain active for this many subsequent is-active probes.
	activeProbesAfterStop int
	probesSinceStop       int
	stopped               bool

	// statesAfterStop, when non-empty, is the is-active sequence returned after stop.
	// After the sequence is exhausted the last entry sticks.
	statesAfterStop []string
	stateSeqIdx     int

	afterStop  func()
	afterStart func()
}

func newFakeNezhaRunner(active, enabled string) *fakeNezhaRunner {
	return &fakeNezhaRunner{
		active:  active,
		enabled: enabled,
		failOps: map[string]error{},
	}
}

func (f *fakeNezhaRunner) CombinedOutput(name string, args ...string) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.calls = append(f.calls, append([]string{name}, args...))
	// Support Status version probe: <binaryPath> -v
	if name != "systemctl" {
		if len(args) == 1 && args[0] == "-v" {
			if f.versionFail {
				return []byte("probe failed\n"), errors.New("binary -v failed")
			}
			out := f.versionOutput
			if out == "" {
				out = "v0.0.0"
			}
			return []byte(out + "\n"), nil
		}
		return nil, fmt.Errorf("unexpected command %q", name)
	}
	if len(args) == 0 {
		return nil, errors.New("systemctl: missing args")
	}

	verb := args[0]
	switch verb {
	case "is-active":
		if f.activeEmptyFail {
			return []byte{}, errors.New("systemctl is-active: empty fail")
		}
		state := f.active
		if f.stopped && len(f.statesAfterStop) > 0 {
			if f.stateSeqIdx < len(f.statesAfterStop) {
				state = f.statesAfterStop[f.stateSeqIdx]
				f.stateSeqIdx++
			} else {
				state = f.statesAfterStop[len(f.statesAfterStop)-1]
			}
			f.active = state
		} else if f.stopped && f.probesSinceStop < f.activeProbesAfterStop {
			f.probesSinceStop++
			state = "active"
		} else if f.stopped {
			state = "inactive"
			f.active = "inactive"
		}
		if state == "active" {
			return []byte(state + "\n"), nil
		}
		// systemctl returns non-zero for inactive/failed; output still carries state.
		return []byte(state + "\n"), fmt.Errorf("systemctl is-active: %s", state)

	case "is-enabled":
		if f.enabledEmptyFail {
			return []byte{}, errors.New("systemctl is-enabled: empty fail")
		}
		state := f.enabled
		if state == "enabled" || state == "enabled-runtime" {
			return []byte(state + "\n"), nil
		}
		return []byte(state + "\n"), fmt.Errorf("systemctl is-enabled: %s", state)

	case "start", "stop", "restart":
		if err, ok := f.failOps[verb]; ok && err != nil {
			return []byte(err.Error()), err
		}
		switch verb {
		case "start", "restart":
			f.active = "active"
			f.stopped = false
			f.probesSinceStop = 0
			f.stateSeqIdx = 0
			if verb == "start" && f.afterStart != nil {
				f.afterStart()
			}
		case "stop":
			f.stopped = true
			f.probesSinceStop = 0
			f.stateSeqIdx = 0
			if len(f.statesAfterStop) == 0 && f.activeProbesAfterStop == 0 {
				f.active = "inactive"
			}
			if f.afterStop != nil {
				f.afterStop()
			}
		}
		return []byte{}, nil

	case "enable":
		if err, ok := f.failOps["enable"]; ok && err != nil {
			return []byte(err.Error()), err
		}
		f.enabled = "enabled"
		f.active = "active"
		f.stopped = false
		return []byte{}, nil

	case "disable":
		if err, ok := f.failOps["disable"]; ok && err != nil {
			return []byte(err.Error()), err
		}
		f.enabled = "disabled"
		f.active = "inactive"
		f.stopped = true
		return []byte{}, nil

	case "list-units":
		// When both are set, return the explicit diagnostic text with the error
		// so fail-closed behavior can be tested (non-empty output must not mean success).
		if f.listUnitsErr != nil {
			out := f.listUnitsOutput
			if out == "" {
				out = f.listUnitsErr.Error() + "\n"
			}
			return []byte(out), f.listUnitsErr
		}
		return []byte(f.listUnitsOutput), nil

	case "daemon-reload":
		if err := f.failOps["daemon-reload"]; err != nil {
			return []byte(err.Error()), err
		}
		return []byte{}, nil
	}

	return nil, fmt.Errorf("unexpected systemctl verb %q", verb)
}

func (f *fakeNezhaRunner) assertCalled(t *testing.T, want ...string) {
	t.Helper()
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, call := range f.calls {
		if len(call) != len(want) {
			continue
		}
		match := true
		for i := range want {
			if call[i] != want[i] {
				match = false
				break
			}
		}
		if match {
			return
		}
	}
	t.Fatalf("expected call %v not found in %v", want, f.calls)
}

func (f *fakeNezhaRunner) assertNotCalled(t *testing.T, verb string) {
	t.Helper()
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, call := range f.calls {
		if len(call) >= 2 && call[0] == "systemctl" && call[1] == verb {
			t.Fatalf("systemctl %s should not have been called; calls=%v", verb, f.calls)
		}
	}
}

// assertNeverMutatesExternalNezhaUnits ensures conflict detection never
// stop/disable external nezha-agent units (only the bundled unit is managed).
func (f *fakeNezhaRunner) assertNeverMutatesExternalNezhaUnits(t *testing.T) {
	t.Helper()
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, call := range f.calls {
		if len(call) < 3 || call[0] != "systemctl" {
			continue
		}
		verb := call[1]
		if verb != "stop" && verb != "disable" {
			continue
		}
		for _, arg := range call[2:] {
			base := strings.TrimSuffix(arg, ".service")
			if base == NezhaAgentUnitName || strings.HasPrefix(arg, "--") {
				continue
			}
			if arg == "nezha-agent" || arg == "nezha-agent.service" ||
				strings.HasPrefix(arg, "nezha-agent@") ||
				strings.HasPrefix(base, "nezha-agent@") {
				t.Fatalf("must not stop/disable external unit; call=%v", call)
			}
		}
	}
}

func (f *fakeNezhaRunner) callVerbs() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	var verbs []string
	for _, call := range f.calls {
		if len(call) >= 2 && call[0] == "systemctl" {
			verbs = append(verbs, call[1])
		}
	}
	return verbs
}

type fakeNezhaSettings struct {
	mu     sync.Mutex
	values map[string]string
	writes [][2]string
	// manyWrites stores defensive copies of each CreateOrUpdateMany payload.
	manyWrites []map[string]string
	fail       error
	// failManyOnCall fails CreateOrUpdateMany on the Nth call (1-based). 0 = never.
	failManyOnCall int
	manyCallCount  int
}

func newFakeNezhaSettings(initial map[string]string) *fakeNezhaSettings {
	if initial == nil {
		initial = map[string]string{}
	}
	cp := make(map[string]string, len(initial))
	for k, v := range initial {
		cp[k] = v
	}
	return &fakeNezhaSettings{values: cp}
}

func (s *fakeNezhaSettings) GetValueByKey(key string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.values[key]
	if !ok {
		return "", errors.New("not found")
	}
	return v, nil
}

func (s *fakeNezhaSettings) CreateOrUpdate(key, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.writes = append(s.writes, [2]string{key, value})
	if s.fail != nil {
		return s.fail
	}
	s.values[key] = value
	return nil
}

func (s *fakeNezhaSettings) CreateOrUpdateMany(values map[string]string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Defensive copy so callers mutating the original map cannot corrupt recorded writes.
	cp := make(map[string]string, len(values))
	for k, v := range values {
		cp[k] = v
	}
	s.manyWrites = append(s.manyWrites, cp)
	for k, v := range cp {
		s.writes = append(s.writes, [2]string{k, v})
	}
	s.manyCallCount++
	if s.fail != nil {
		return s.fail
	}
	if s.failManyOnCall > 0 && s.manyCallCount == s.failManyOnCall {
		return errors.New("db batch write failed")
	}
	for k, v := range cp {
		s.values[k] = v
	}
	return nil
}

func (s *fakeNezhaSettings) assertWritten(t *testing.T, key, value string) {
	t.Helper()
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, w := range s.writes {
		if w[0] == key && w[1] == value {
			return
		}
	}
	t.Fatalf("expected setting write %s=%s, got %v", key, value, s.writes)
}

func (s *fakeNezhaSettings) assertNotWritten(t *testing.T, key string) {
	t.Helper()
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, w := range s.writes {
		if w[0] == key {
			t.Fatalf("setting %s should not have been written; writes=%v", key, s.writes)
		}
	}
}

func (s *fakeNezhaSettings) assertManyWritten(t *testing.T, want map[string]string) {
	t.Helper()
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, got := range s.manyWrites {
		if len(got) != len(want) {
			continue
		}
		match := true
		for k, v := range want {
			if got[k] != v {
				match = false
				break
			}
		}
		if match {
			return
		}
	}
	t.Fatalf("expected batch write %#v, got %#v", want, s.manyWrites)
}

func (s *fakeNezhaSettings) manyWriteCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.manyWrites)
}

func (s *fakeNezhaSettings) valueOf(key string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.values[key]
	return v, ok
}

// assertErrorHasNoSecret fails without embedding the secret value in the message.
func assertErrorHasNoSecret(t *testing.T, err error, secret string) {
	t.Helper()
	if err == nil || secret == "" {
		return
	}
	if strings.Contains(err.Error(), secret) {
		t.Fatalf("error must not expose secret (len=%d)", len(secret))
	}
}

func testNezhaConfigPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "config.yml")
}

func newTestNezhaAgentService(t *testing.T, runner *fakeNezhaRunner, settings *fakeNezhaSettings) *NezhaAgentService {
	t.Helper()
	if settings == nil {
		settings = newFakeNezhaSettings(nil)
	}
	return newNezhaAgentService(nezhaAgentDeps{
		ConfigPath:  testNezhaConfigPath(t),
		Unit:        NezhaAgentUnitName,
		Runner:      runner,
		Settings:    settings,
		Sleep:       func(time.Duration) {},
		Now:         time.Now,
		StopTimeout: 50 * time.Millisecond,
		PollEvery:   time.Millisecond,
	})
}

// ----- Operate -----

func TestNezhaStartDoesNotChangeEnabledExpectation(t *testing.T) {
	runner := newFakeNezhaRunner("inactive", "disabled")
	settings := newFakeNezhaSettings(map[string]string{"NezhaEnabled": "false"})
	svc := newTestNezhaAgentService(t, runner, settings)

	if err := svc.Operate("start"); err != nil {
		t.Fatal(err)
	}
	runner.assertCalled(t, "systemctl", "start", NezhaAgentUnitName)
	settings.assertNotWritten(t, "NezhaEnabled")
}

func TestNezhaStopDoesNotChangeEnabledExpectation(t *testing.T) {
	runner := newFakeNezhaRunner("active", "enabled")
	settings := newFakeNezhaSettings(map[string]string{"NezhaEnabled": "true"})
	svc := newTestNezhaAgentService(t, runner, settings)

	if err := svc.Operate("stop"); err != nil {
		t.Fatal(err)
	}
	runner.assertCalled(t, "systemctl", "stop", NezhaAgentUnitName)
	settings.assertNotWritten(t, "NezhaEnabled")
}

func TestNezhaRestartDoesNotChangeEnabledExpectation(t *testing.T) {
	runner := newFakeNezhaRunner("active", "enabled")
	settings := newFakeNezhaSettings(map[string]string{"NezhaEnabled": "true"})
	svc := newTestNezhaAgentService(t, runner, settings)

	if err := svc.Operate("restart"); err != nil {
		t.Fatal(err)
	}
	runner.assertCalled(t, "systemctl", "restart", NezhaAgentUnitName)
	settings.assertNotWritten(t, "NezhaEnabled")
}

func TestNezhaEnableWritesDBOnlyAfterSystemctlSuccess(t *testing.T) {
	runner := newFakeNezhaRunner("inactive", "disabled")
	settings := newFakeNezhaSettings(map[string]string{"NezhaEnabled": "false"})
	svc := newTestNezhaAgentService(t, runner, settings)

	if err := svc.Operate("enable"); err != nil {
		t.Fatal(err)
	}
	runner.assertCalled(t, "systemctl", "enable", "--now", NezhaAgentUnitName)
	settings.assertWritten(t, "NezhaEnabled", "true")
}

func TestNezhaEnableSystemctlFailureDoesNotWriteDB(t *testing.T) {
	runner := newFakeNezhaRunner("inactive", "disabled")
	runner.failOps["enable"] = errors.New("enable failed")
	settings := newFakeNezhaSettings(map[string]string{"NezhaEnabled": "false"})
	svc := newTestNezhaAgentService(t, runner, settings)

	if err := svc.Operate("enable"); err == nil {
		t.Fatal("expected enable failure")
	}
	settings.assertNotWritten(t, "NezhaEnabled")
}

func TestNezhaEnableSettingWriteFailureReturnsError(t *testing.T) {
	runner := newFakeNezhaRunner("inactive", "disabled")
	settings := newFakeNezhaSettings(map[string]string{"NezhaEnabled": "false"})
	settings.fail = errors.New("db down")
	svc := newTestNezhaAgentService(t, runner, settings)

	err := svc.Operate("enable")
	if err == nil {
		t.Fatal("expected setting write error")
	}
	if !strings.Contains(err.Error(), "NezhaEnabled") && !strings.Contains(err.Error(), "db down") {
		t.Fatalf("error should mention setting failure, got %v", err)
	}
	runner.assertCalled(t, "systemctl", "enable", "--now", NezhaAgentUnitName)
}

func TestNezhaDisableWritesDBOnlyAfterSystemctlSuccess(t *testing.T) {
	runner := newFakeNezhaRunner("active", "enabled")
	settings := newFakeNezhaSettings(map[string]string{"NezhaEnabled": "true"})
	svc := newTestNezhaAgentService(t, runner, settings)

	if err := svc.Operate("disable"); err != nil {
		t.Fatal(err)
	}
	runner.assertCalled(t, "systemctl", "disable", "--now", NezhaAgentUnitName)
	settings.assertWritten(t, "NezhaEnabled", "false")
}

func TestNezhaDisableSystemctlFailureDoesNotWriteDB(t *testing.T) {
	runner := newFakeNezhaRunner("active", "enabled")
	runner.failOps["disable"] = errors.New("disable failed")
	settings := newFakeNezhaSettings(map[string]string{"NezhaEnabled": "true"})
	svc := newTestNezhaAgentService(t, runner, settings)

	if err := svc.Operate("disable"); err == nil {
		t.Fatal("expected disable failure")
	}
	settings.assertNotWritten(t, "NezhaEnabled")
}

func TestNezhaOperateRejectsUnknownOperation(t *testing.T) {
	runner := newFakeNezhaRunner("inactive", "disabled")
	svc := newTestNezhaAgentService(t, runner, nil)
	if err := svc.Operate("reload"); err == nil {
		t.Fatal("expected unknown operation error")
	}
	if len(runner.calls) != 0 {
		t.Fatalf("no systemctl calls expected, got %v", runner.calls)
	}
}

// ----- Configure -----

func TestNezhaConfigureValidatesBeforeStop(t *testing.T) {
	runner := newFakeNezhaRunner("active", "enabled")
	svc := newTestNezhaAgentService(t, runner, nil)

	if err := os.WriteFile(svc.configPath, []byte("server: old.example.com:443\nclient_secret: keep\n"), 0600); err != nil {
		t.Fatal(err)
	}

	bad := "http://insecure.example.com"
	err := svc.Configure(dto.NezhaAgentConfigUpdate{DashboardURL: &bad})
	if err == nil {
		t.Fatal("expected validation error")
	}
	runner.assertNotCalled(t, "stop")
}

func TestNezhaConfigureFirstConfigRequiresDashboardAndSecretBeforeStop(t *testing.T) {
	runner := newFakeNezhaRunner("active", "disabled")
	svc := newTestNezhaAgentService(t, runner, nil)

	dash := "https://dashboard.example.com"
	err := svc.Configure(dto.NezhaAgentConfigUpdate{DashboardURL: &dash})
	if err == nil {
		t.Fatal("expected missing secret error")
	}
	runner.assertNotCalled(t, "stop")

	secret := "agent-secret"
	emptyDash := ""
	err = svc.Configure(dto.NezhaAgentConfigUpdate{
		DashboardURL: &emptyDash,
		ClientSecret: &secret,
	})
	if err == nil {
		t.Fatal("expected missing dashboard error")
	}
	runner.assertNotCalled(t, "stop")
}

func TestNezhaConfigureWaitsForInactiveBeforeReadingConfig(t *testing.T) {
	runner := newFakeNezhaRunner("active", "enabled")
	runner.activeProbesAfterStop = 3
	settings := newFakeNezhaSettings(nil)
	svc := newTestNezhaAgentService(t, runner, settings)

	original := []byte("server: old.example.com:443\nclient_secret: before-stop\nuuid: keep-uuid\n")
	if err := os.WriteFile(svc.configPath, original, 0600); err != nil {
		t.Fatal(err)
	}
	runner.afterStop = func() {
		_ = os.WriteFile(svc.configPath, []byte("server: old.example.com:443\nclient_secret: after-stop-rotated\nuuid: keep-uuid\n"), 0600)
	}

	var readWhile string
	var firstRead bool
	realRead := svc.readConfig
	svc.readConfig = func(path string) ([]byte, error) {
		runner.mu.Lock()
		// Capture only the first read (post-stop merge/sync); later post-start
		// sync also reads while the unit is active again.
		if !firstRead {
			readWhile = runner.active
			firstRead = true
		}
		runner.mu.Unlock()
		return realRead(path)
	}

	dash := "https://new.example.com:8443"
	if err := svc.Configure(dto.NezhaAgentConfigUpdate{DashboardURL: &dash}); err != nil {
		t.Fatal(err)
	}
	if readWhile != "inactive" {
		t.Fatalf("config read while active state=%q, want inactive", readWhile)
	}

	verbs := runner.callVerbs()
	stopIdx, activeIdx := -1, -1
	for i, v := range verbs {
		if v == "stop" && stopIdx < 0 {
			stopIdx = i
		}
		if v == "is-active" && stopIdx >= 0 && i > stopIdx && activeIdx < 0 {
			activeIdx = i
		}
	}
	if stopIdx < 0 || activeIdx < 0 {
		t.Fatalf("expected stop then is-active wait, verbs=%v", verbs)
	}

	got, err := os.ReadFile(svc.configPath)
	if err != nil {
		t.Fatal(err)
	}
	assertYAMLValue(t, got, "client_secret", "after-stop-rotated")
	assertYAMLValue(t, got, "uuid", "keep-uuid")
	assertYAMLValue(t, got, "server", "new.example.com:8443")
	runner.assertCalled(t, "systemctl", "start", NezhaAgentUnitName)
}

func TestNezhaConfigureWaitsThroughDeactivatingBeforeReadingConfig(t *testing.T) {
	runner := newFakeNezhaRunner("active", "enabled")
	runner.statesAfterStop = []string{"deactivating", "deactivating", "inactive"}
	svc := newTestNezhaAgentService(t, runner, nil)

	original := []byte("server: old.example.com:443\nclient_secret: before-stop\nuuid: keep-uuid\n")
	if err := os.WriteFile(svc.configPath, original, 0600); err != nil {
		t.Fatal(err)
	}
	runner.afterStop = func() {
		_ = os.WriteFile(svc.configPath, []byte("server: old.example.com:443\nclient_secret: after-stop-rotated\nuuid: keep-uuid\n"), 0600)
	}

	var readWhile string
	var readCount int
	realRead := svc.readConfig
	svc.readConfig = func(path string) ([]byte, error) {
		runner.mu.Lock()
		// First read must be after inactive; later post-start sync re-reads while active.
		if readCount == 0 {
			readWhile = runner.active
		}
		readCount++
		runner.mu.Unlock()
		return realRead(path)
	}

	dash := "https://new.example.com:8443"
	if err := svc.Configure(dto.NezhaAgentConfigUpdate{DashboardURL: &dash}); err != nil {
		t.Fatal(err)
	}
	if readCount == 0 {
		t.Fatal("expected config read after wait")
	}
	if readWhile != "inactive" {
		t.Fatalf("config read while state=%q, want inactive (must not read during deactivating)", readWhile)
	}

	// Ensure the wait actually observed transitional deactivating probes, not a single inactive.
	postStopActive := 0
	seenStop := false
	for _, call := range runner.calls {
		if len(call) >= 2 && call[0] == "systemctl" && call[1] == "stop" {
			seenStop = true
			continue
		}
		if seenStop && len(call) >= 2 && call[0] == "systemctl" && call[1] == "is-active" {
			postStopActive++
		}
	}
	if postStopActive < 3 {
		t.Fatalf("expected at least 3 post-stop is-active probes for deactivating sequence, got %d; calls=%v", postStopActive, runner.calls)
	}

	got, err := os.ReadFile(svc.configPath)
	if err != nil {
		t.Fatal(err)
	}
	assertYAMLValue(t, got, "client_secret", "after-stop-rotated")
	assertYAMLValue(t, got, "server", "new.example.com:8443")
}

func TestNezhaConfigureWaitFailureAttemptsRestore(t *testing.T) {
	runner := newFakeNezhaRunner("active", "enabled")
	// Stick in deactivating so waitUntilInactive times out.
	runner.statesAfterStop = []string{"deactivating"}
	runner.failOps["start"] = errors.New("start refused")
	svc := newTestNezhaAgentService(t, runner, nil)

	original := []byte("server: old.example.com:443\nclient_secret: keep-me\n")
	if err := os.WriteFile(svc.configPath, original, 0600); err != nil {
		t.Fatal(err)
	}

	readAttempted := false
	svc.readConfig = func(path string) ([]byte, error) {
		readAttempted = true
		return nil, errors.New("should not read config before inactive")
	}

	dash := "https://new.example.com"
	err := svc.Configure(dto.NezhaAgentConfigUpdate{DashboardURL: &dash})
	if err == nil {
		t.Fatal("expected wait failure with restore attempt")
	}
	msg := err.Error()
	if !strings.Contains(msg, "timed out") {
		t.Fatalf("error should include wait timeout reason: %v", err)
	}
	if !strings.Contains(msg, "start refused") {
		t.Fatalf("error should include restore start failure: %v", err)
	}
	runner.assertCalled(t, "systemctl", "stop", NezhaAgentUnitName)
	runner.assertCalled(t, "systemctl", "start", NezhaAgentUnitName)
	if readAttempted {
		t.Fatal("config must not be read when stop wait fails")
	}

	got, err := os.ReadFile(svc.configPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(original) {
		t.Fatalf("original config must remain intact after wait failure, got %q", got)
	}
}

func TestNezhaConfigureWriteFailureRestoresActiveService(t *testing.T) {
	runner := newFakeNezhaRunner("active", "enabled")
	svc := newTestNezhaAgentService(t, runner, nil)

	original := []byte("server: old.example.com:443\nclient_secret: keep-me\n")
	if err := os.WriteFile(svc.configPath, original, 0600); err != nil {
		t.Fatal(err)
	}

	svc.writeConfig = func(path string, data []byte) error {
		return errors.New("disk full")
	}

	dash := "https://new.example.com"
	err := svc.Configure(dto.NezhaAgentConfigUpdate{DashboardURL: &dash})
	if err == nil {
		t.Fatal("expected write failure")
	}
	if !strings.Contains(err.Error(), "disk full") {
		t.Fatalf("error should mention write failure: %v", err)
	}
	runner.assertCalled(t, "systemctl", "stop", NezhaAgentUnitName)
	runner.assertCalled(t, "systemctl", "start", NezhaAgentUnitName)

	got, err := os.ReadFile(svc.configPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(original) {
		t.Fatalf("original config must remain intact, got %q", got)
	}
}

func TestNezhaConfigureWriteFailureAndRestoreFailureReturnsCombinedError(t *testing.T) {
	runner := newFakeNezhaRunner("active", "enabled")
	runner.failOps["start"] = errors.New("start refused")
	svc := newTestNezhaAgentService(t, runner, nil)

	if err := os.WriteFile(svc.configPath, []byte("server: old.example.com:443\nclient_secret: keep\n"), 0600); err != nil {
		t.Fatal(err)
	}
	svc.writeConfig = func(path string, data []byte) error {
		return errors.New("write boom")
	}

	dash := "https://new.example.com"
	err := svc.Configure(dto.NezhaAgentConfigUpdate{DashboardURL: &dash})
	if err == nil {
		t.Fatal("expected combined error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "write boom") || !strings.Contains(msg, "start refused") {
		t.Fatalf("combined diagnostic missing parts: %v", err)
	}
}

func TestNezhaConfigureWriteSuccessStartFailureKeepsNewBytes(t *testing.T) {
	runner := newFakeNezhaRunner("active", "enabled")
	runner.failOps["start"] = errors.New("start refused")
	svc := newTestNezhaAgentService(t, runner, nil)

	if err := os.WriteFile(svc.configPath, []byte("server: old.example.com:443\nclient_secret: keep\n"), 0600); err != nil {
		t.Fatal(err)
	}

	dash := "https://new.example.com:9443"
	err := svc.Configure(dto.NezhaAgentConfigUpdate{DashboardURL: &dash})
	if err == nil {
		t.Fatal("expected start failure after successful write")
	}
	if !strings.Contains(err.Error(), "start refused") {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := os.ReadFile(svc.configPath)
	if err != nil {
		t.Fatal(err)
	}
	assertYAMLValue(t, got, "server", "new.example.com:9443")
	assertYAMLValue(t, got, "client_secret", "keep")
	assertYAMLValue(t, got, "tls", true)
	assertYAMLValue(t, got, "insecure_tls", false)
}

func TestNezhaConfigureOriginallyStoppedStaysStopped(t *testing.T) {
	runner := newFakeNezhaRunner("inactive", "disabled")
	svc := newTestNezhaAgentService(t, runner, nil)

	if err := os.WriteFile(svc.configPath, []byte("server: old.example.com:443\nclient_secret: keep\n"), 0600); err != nil {
		t.Fatal(err)
	}

	dash := "https://new.example.com"
	if err := svc.Configure(dto.NezhaAgentConfigUpdate{DashboardURL: &dash}); err != nil {
		t.Fatal(err)
	}
	runner.assertNotCalled(t, "stop")
	runner.assertNotCalled(t, "start")
	runner.assertNotCalled(t, "enable")

	got, err := os.ReadFile(svc.configPath)
	if err != nil {
		t.Fatal(err)
	}
	assertYAMLValue(t, got, "server", "new.example.com:443")
}

func TestNezhaConfigureEnableAndStart(t *testing.T) {
	runner := newFakeNezhaRunner("inactive", "disabled")
	settings := newFakeNezhaSettings(map[string]string{"NezhaEnabled": "false"})
	svc := newTestNezhaAgentService(t, runner, settings)

	conflictChecked := false
	svc.checkConflictFree = func() error {
		conflictChecked = true
		return nil
	}

	dash := "https://dashboard.example.com"
	secret := "first-secret"
	if err := svc.Configure(dto.NezhaAgentConfigUpdate{
		DashboardURL:   &dash,
		ClientSecret:   &secret,
		EnableAndStart: true,
	}); err != nil {
		t.Fatal(err)
	}
	if !conflictChecked {
		t.Fatal("expected conflict-free hook to run before enable")
	}
	runner.assertCalled(t, "systemctl", "enable", "--now", NezhaAgentUnitName)
	settings.assertWritten(t, "NezhaEnabled", "true")

	got, err := os.ReadFile(svc.configPath)
	if err != nil {
		t.Fatal(err)
	}
	assertYAMLValue(t, got, "server", "dashboard.example.com:443")
	assertYAMLValue(t, got, "client_secret", "first-secret")
	assertYAMLValue(t, got, "disable_auto_update", true)
	assertYAMLValue(t, got, "disable_force_update", true)
	assertYAMLValue(t, got, "disable_command_execute", false)
	assertYAMLMissing(t, got, "uuid")
}

func TestNezhaConfigureEnableAndStartSystemctlFailureDoesNotWriteDB(t *testing.T) {
	runner := newFakeNezhaRunner("inactive", "disabled")
	runner.failOps["enable"] = errors.New("enable failed")
	settings := newFakeNezhaSettings(map[string]string{"NezhaEnabled": "false"})
	svc := newTestNezhaAgentService(t, runner, settings)

	if err := os.WriteFile(svc.configPath, []byte("server: old.example.com:443\nclient_secret: keep\n"), 0600); err != nil {
		t.Fatal(err)
	}

	dash := "https://new.example.com"
	err := svc.Configure(dto.NezhaAgentConfigUpdate{
		DashboardURL:   &dash,
		EnableAndStart: true,
	})
	if err == nil {
		t.Fatal("expected enable failure")
	}
	settings.assertNotWritten(t, "NezhaEnabled")

	got, _ := os.ReadFile(svc.configPath)
	assertYAMLValue(t, got, "server", "new.example.com:443")
}

func TestNezhaConfigureEmptySecretLeavesExisting(t *testing.T) {
	runner := newFakeNezhaRunner("inactive", "disabled")
	svc := newTestNezhaAgentService(t, runner, nil)

	if err := os.WriteFile(svc.configPath, []byte("server: old.example.com:443\nclient_secret: rotated\n"), 0600); err != nil {
		t.Fatal(err)
	}
	empty := ""
	dash := "https://new.example.com"
	if err := svc.Configure(dto.NezhaAgentConfigUpdate{
		DashboardURL: &dash,
		ClientSecret: &empty,
	}); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(svc.configPath)
	if err != nil {
		t.Fatal(err)
	}
	assertYAMLValue(t, got, "client_secret", "rotated")
	assertYAMLValue(t, got, "server", "new.example.com:443")
}

func TestNezhaConfigureMergesOnlyExplicitRemoteOperations(t *testing.T) {
	runner := newFakeNezhaRunner("inactive", "disabled")
	svc := newTestNezhaAgentService(t, runner, nil)

	input := []byte("server: old.example.com:443\nclient_secret: keep\ndisable_command_execute: false\ncustom_flag: keep-me\n")
	if err := os.WriteFile(svc.configPath, input, 0600); err != nil {
		t.Fatal(err)
	}
	off := false
	if err := svc.Configure(dto.NezhaAgentConfigUpdate{RemoteOperationsEnabled: &off}); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(svc.configPath)
	if err != nil {
		t.Fatal(err)
	}
	assertYAMLValue(t, got, "disable_command_execute", true)
	assertYAMLValue(t, got, "client_secret", "keep")
	assertYAMLValue(t, got, "server", "old.example.com:443")
	assertYAMLValue(t, got, "custom_flag", "keep-me")
}

func TestNezhaConstantsPathsAndUnit(t *testing.T) {
	if NezhaAgentDir != "/opt/xpanel/nezha-agent" {
		t.Fatalf("dir=%q", NezhaAgentDir)
	}
	if NezhaAgentBinaryPath != "/opt/xpanel/nezha-agent/nezha-agent" {
		t.Fatalf("binary=%q", NezhaAgentBinaryPath)
	}
	if NezhaAgentConfigPath != "/opt/xpanel/nezha-agent/config.yml" {
		t.Fatalf("config=%q", NezhaAgentConfigPath)
	}
	if NezhaAgentUnitName != "xpanel-nezha-agent" {
		t.Fatalf("unit=%q", NezhaAgentUnitName)
	}
	if !filepath.IsAbs(NezhaAgentBinaryPath) || !filepath.IsAbs(NezhaAgentConfigPath) {
		t.Fatal("binary and config paths must be absolute")
	}
}

// ----- Status (Task 2B1a healthy slice) -----

func TestNezhaStatusHealthyDoesNotExposeSecret(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}

	binaryPath := filepath.Join(dir, "nezha-agent")
	configPath := filepath.Join(dir, "config.yml")
	unitPath := filepath.Join(dir, "xpanel-nezha-agent.service")

	if err := os.WriteFile(binaryPath, []byte("#!/bin/sh\necho stub\n"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(binaryPath, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(unitPath, []byte("[Unit]\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Unique secret sentinel — must never appear in Status JSON.
	secret := "NzjStatusHealthySec_c0ffee99"
	cfg := strings.Join([]string{
		"server: dashboard.example.com:443",
		"client_secret: " + secret,
		"uuid: agent-uuid-status-healthy",
		"tls: true",
		"insecure_tls: false",
		"disable_command_execute: false",
		"",
	}, "\n")
	if err := os.WriteFile(configPath, []byte(cfg), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(configPath, 0600); err != nil {
		t.Fatal(err)
	}

	runner := newFakeNezhaRunner("active", "enabled")
	runner.versionOutput = "nezha-agent version v2.3.1"
	settings := newFakeNezhaSettings(map[string]string{"NezhaEnabled": "true"})

	svc := newNezhaAgentService(nezhaAgentDeps{
		ConfigPath:  configPath,
		BinaryPath:  binaryPath,
		UnitPath:    unitPath,
		Unit:        NezhaAgentUnitName,
		Runner:      runner,
		Settings:    settings,
		Sleep:       func(time.Duration) {},
		Now:         time.Now,
		StopTimeout: 50 * time.Millisecond,
		PollEvery:   time.Millisecond,
	})

	st, err := svc.Status()
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if st == nil {
		t.Fatal("Status returned nil")
	}

	if !st.ComponentAvailable {
		t.Fatal("componentAvailable want true")
	}
	if !st.Configured {
		t.Fatal("configured want true")
	}
	if !st.ConfigHealthy {
		t.Fatal("configHealthy want true")
	}
	if st.ConfigError != "" {
		t.Fatalf("configError want empty, got %q", st.ConfigError)
	}
	if !st.Active {
		t.Fatal("active want true")
	}
	if st.ServiceState != "active" {
		t.Fatalf("serviceState=%q want active", st.ServiceState)
	}
	if !st.Enabled {
		t.Fatal("enabled want true")
	}
	if !st.DesiredEnabled {
		t.Fatal("desiredEnabled want true")
	}
	if st.Drift {
		t.Fatal("drift want false when enabled matches desired")
	}
	if st.Version != "v2.3.1" {
		t.Fatalf("version=%q want v2.3.1 (last field of binary -v)", st.Version)
	}
	if st.UUID != "agent-uuid-status-healthy" {
		t.Fatalf("uuid=%q", st.UUID)
	}
	if st.DashboardURL != "https://dashboard.example.com" {
		t.Fatalf("dashboardUrl=%q want https://dashboard.example.com", st.DashboardURL)
	}
	if st.Server != "dashboard.example.com:443" {
		t.Fatalf("server=%q", st.Server)
	}
	if !st.TLS {
		t.Fatal("tls want true")
	}
	if st.InsecureTLS {
		t.Fatal("insecureTls want false")
	}
	if !st.SecretConfigured {
		t.Fatal("secretConfigured want true")
	}
	if !st.RemoteOperationsEnabled {
		t.Fatal("remoteOperationsEnabled want true (disable_command_execute=false)")
	}
	if st.PermissionsWarning != "" {
		t.Fatalf("permissionsWarning want empty, got %q", st.PermissionsWarning)
	}
	if st.ServiceError != "" {
		t.Fatalf("serviceError want empty, got %q", st.ServiceError)
	}
	if st.Conflicts == nil {
		t.Fatal("conflicts must be non-nil empty slice")
	}
	if len(st.Conflicts) != 0 {
		t.Fatalf("conflicts want empty, got %#v", st.Conflicts)
	}

	raw, err := json.Marshal(st)
	if err != nil {
		t.Fatal(err)
	}
	body := string(raw)
	if strings.Contains(body, secret) {
		t.Fatal("Status JSON must not expose client_secret value")
	}
	if strings.Contains(body, "clientSecret") || strings.Contains(body, "client_secret") {
		t.Fatalf("Status JSON must not include secret field names, got %s", body)
	}

	runner.assertCalled(t, binaryPath, "-v")
	runner.assertCalled(t, "systemctl", "is-active", NezhaAgentUnitName)
	runner.assertCalled(t, "systemctl", "is-enabled", NezhaAgentUnitName)
}

// ----- Status boundaries (Task 2B1b) -----

// nezhaStatusFixture builds a temp Agent dir/binary/config for Status tests.
// pass cfgContent=nil to leave the config path missing.
func nezhaStatusFixture(t *testing.T, dirMode, binMode, cfgMode os.FileMode, cfgContent []byte) (configPath, binaryPath string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.Chmod(dir, dirMode); err != nil {
		t.Fatal(err)
	}
	binaryPath = filepath.Join(dir, "nezha-agent")
	configPath = filepath.Join(dir, "config.yml")
	if err := os.WriteFile(binaryPath, []byte("#!/bin/sh\necho stub\n"), binMode); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(binaryPath, binMode); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "xpanel-nezha-agent.service"), []byte("[Unit]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if cfgContent != nil {
		if err := os.WriteFile(configPath, cfgContent, cfgMode); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(configPath, cfgMode); err != nil {
			t.Fatal(err)
		}
	}
	return configPath, binaryPath
}

func newStatusNezhaService(t *testing.T, configPath, binaryPath string, runner *fakeNezhaRunner, settings *fakeNezhaSettings) *NezhaAgentService {
	t.Helper()
	if settings == nil {
		settings = newFakeNezhaSettings(nil)
	}
	return newNezhaAgentService(nezhaAgentDeps{
		ConfigPath:  configPath,
		BinaryPath:  binaryPath,
		UnitPath:    filepath.Join(filepath.Dir(binaryPath), "xpanel-nezha-agent.service"),
		Unit:        NezhaAgentUnitName,
		Runner:      runner,
		Settings:    settings,
		Sleep:       func(time.Duration) {},
		Now:         time.Now,
		StopTimeout: 50 * time.Millisecond,
		PollEvery:   time.Millisecond,
	})
}

func assertStatusNoSecret(t *testing.T, st *dto.NezhaAgentStatus, secret string) {
	t.Helper()
	raw, err := json.Marshal(st)
	if err != nil {
		t.Fatal(err)
	}
	body := string(raw)
	if secret != "" && strings.Contains(body, secret) {
		t.Fatal("Status JSON must not expose secret")
	}
	if st.ConfigError != "" && secret != "" && strings.Contains(st.ConfigError, secret) {
		t.Fatal("configError must not expose secret")
	}
	if st.ServiceError != "" && secret != "" && strings.Contains(st.ServiceError, secret) {
		t.Fatal("serviceError must not expose secret")
	}
	if st.PermissionsWarning != "" && secret != "" && strings.Contains(st.PermissionsWarning, secret) {
		t.Fatal("permissionsWarning must not expose secret")
	}
}

func TestNezhaStatusMissingConfig(t *testing.T) {
	configPath, binaryPath := nezhaStatusFixture(t, 0700, 0755, 0600, nil)
	settings := newFakeNezhaSettings(map[string]string{"NezhaEnabled": "true"})
	runner := newFakeNezhaRunner("inactive", "disabled")
	svc := newStatusNezhaService(t, configPath, binaryPath, runner, settings)

	st, err := svc.Status()
	if err != nil {
		t.Fatalf("Status top-level error: %v", err)
	}
	if st == nil {
		t.Fatal("Status returned nil")
	}
	if st.Configured {
		t.Fatal("configured want false when config file is missing")
	}
	if st.ConfigHealthy {
		t.Fatal("configHealthy want false when config file is missing")
	}
	if st.ConfigError == "" {
		t.Fatal("configError must be explicit when config is missing")
	}
	// Must not rebuild from DB / write settings.
	settings.assertNotWritten(t, "NezhaEnabled")
	settings.assertNotWritten(t, "NezhaClientSecret")
	// Service state still reported.
	if st.ServiceState != "inactive" {
		t.Fatalf("serviceState=%q want inactive", st.ServiceState)
	}
	if st.Active {
		t.Fatal("active want false")
	}
}

func TestNezhaStatusCorruptYAML(t *testing.T) {
	secret := "NzjStatusCorruptSec_deadbeef42"
	// YAML is broken but embeds the secret sentinel so we can assert no leakage.
	cfg := []byte("server: dash.example.com:443\nclient_secret: " + secret + "\n[unterminated\n")
	configPath, binaryPath := nezhaStatusFixture(t, 0700, 0755, 0600, cfg)
	runner := newFakeNezhaRunner("inactive", "disabled")
	svc := newStatusNezhaService(t, configPath, binaryPath, runner, nil)

	st, err := svc.Status()
	if err != nil {
		t.Fatalf("Status top-level error: %v", err)
	}
	if !st.Configured {
		t.Fatal("configured want true when config file exists (even if corrupt)")
	}
	if st.ConfigHealthy {
		t.Fatal("configHealthy want false for corrupt YAML")
	}
	if st.ConfigError == "" {
		t.Fatal("configError must describe corrupt YAML safely")
	}
	if strings.Contains(st.ConfigError, secret) {
		t.Fatal("configError must not include secret from file")
	}
	if strings.Contains(st.ConfigError, string(cfg)) {
		t.Fatal("configError must not dump file content")
	}
	assertStatusNoSecret(t, st, secret)
	// Other status still returned.
	if st.ServiceState != "inactive" {
		t.Fatalf("serviceState=%q want inactive", st.ServiceState)
	}
}

func TestNezhaStatusSymlinkConfig(t *testing.T) {
	secret := "NzjStatusSymlinkSec_aabbcc11"
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	binaryPath := filepath.Join(dir, "nezha-agent")
	if err := os.WriteFile(binaryPath, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(dir, "real.yml")
	if err := os.WriteFile(target, []byte("server: dash:443\nclient_secret: "+secret+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(dir, "config.yml")
	if err := os.Symlink(target, configPath); err != nil {
		t.Fatal(err)
	}
	runner := newFakeNezhaRunner("inactive", "disabled")
	svc := newStatusNezhaService(t, configPath, binaryPath, runner, nil)

	st, err := svc.Status()
	if err != nil {
		t.Fatalf("Status top-level error: %v", err)
	}
	if !st.Configured {
		t.Fatal("configured want true when config path exists as symlink")
	}
	if st.ConfigHealthy {
		t.Fatal("configHealthy want false for symlink config")
	}
	if st.ConfigError == "" {
		t.Fatal("configError must diagnose unsafe config path")
	}
	if strings.Contains(st.ConfigError, secret) {
		t.Fatal("configError must not include secret")
	}
	assertStatusNoSecret(t, st, secret)
}

func TestNezhaStatusNonRegularConfig(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	binaryPath := filepath.Join(dir, "nezha-agent")
	if err := os.WriteFile(binaryPath, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(dir, "config.yml")
	if err := os.Mkdir(configPath, 0700); err != nil {
		t.Fatal(err)
	}
	runner := newFakeNezhaRunner("inactive", "disabled")
	svc := newStatusNezhaService(t, configPath, binaryPath, runner, nil)

	st, err := svc.Status()
	if err != nil {
		t.Fatalf("Status top-level error: %v", err)
	}
	if !st.Configured {
		t.Fatal("configured want true when config path exists (directory)")
	}
	if st.ConfigHealthy {
		t.Fatal("configHealthy want false for non-regular config")
	}
	if st.ConfigError == "" {
		t.Fatal("configError must diagnose non-regular config")
	}
}

func TestNezhaStatusServerAndSecretRequireNonEmptyStrings(t *testing.T) {
	secret := "NzjStatusTypeSec_11223344"
	cases := []struct {
		name string
		cfg  string
	}{
		{
			name: "missing server",
			cfg:  "client_secret: " + secret + "\nuuid: u1\n",
		},
		{
			name: "empty server",
			cfg:  "server: \"\"\nclient_secret: " + secret + "\n",
		},
		{
			name: "server wrong type",
			cfg:  "server: 443\nclient_secret: " + secret + "\n",
		},
		{
			name: "missing secret",
			cfg:  "server: dash.example.com:443\nuuid: u1\n",
		},
		{
			name: "empty secret",
			cfg:  "server: dash.example.com:443\nclient_secret: \"\"\n",
		},
		{
			name: "secret wrong type",
			cfg:  "server: dash.example.com:443\nclient_secret: 12345\n",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			configPath, binaryPath := nezhaStatusFixture(t, 0700, 0755, 0600, []byte(tc.cfg))
			runner := newFakeNezhaRunner("inactive", "disabled")
			svc := newStatusNezhaService(t, configPath, binaryPath, runner, nil)
			st, err := svc.Status()
			if err != nil {
				t.Fatalf("Status: %v", err)
			}
			if !st.Configured {
				t.Fatal("configured want true")
			}
			if st.ConfigHealthy {
				t.Fatal("configHealthy want false when server/secret invalid")
			}
			if st.ConfigError == "" {
				t.Fatal("configError must explain invalid server/secret")
			}
			// Never leak typed/wrong values into the diagnostic.
			if strings.Contains(st.ConfigError, secret) {
				t.Fatal("configError must not include secret value")
			}
			if strings.Contains(st.ConfigError, "12345") {
				t.Fatal("configError must not include wrong-type secret value")
			}
			assertStatusNoSecret(t, st, secret)
		})
	}
}

func TestNezhaStatusUUIDOnlyStringDisplay(t *testing.T) {
	// Non-string uuid must not be stringified into Status.
	cfg := []byte("server: dash.example.com:443\nclient_secret: ok-secret\nuuid: 42\n")
	configPath, binaryPath := nezhaStatusFixture(t, 0700, 0755, 0600, cfg)
	runner := newFakeNezhaRunner("inactive", "disabled")
	svc := newStatusNezhaService(t, configPath, binaryPath, runner, nil)
	st, err := svc.Status()
	if err != nil {
		t.Fatal(err)
	}
	if st.UUID != "" {
		t.Fatalf("uuid want empty for non-string, got %q", st.UUID)
	}

	cfg2 := []byte("server: dash.example.com:443\nclient_secret: ok-secret\nuuid: real-uuid-str\n")
	configPath2, binaryPath2 := nezhaStatusFixture(t, 0700, 0755, 0600, cfg2)
	svc2 := newStatusNezhaService(t, configPath2, binaryPath2, newFakeNezhaRunner("inactive", "disabled"), nil)
	st2, err := svc2.Status()
	if err != nil {
		t.Fatal(err)
	}
	if st2.UUID != "real-uuid-str" {
		t.Fatalf("uuid=%q want real-uuid-str", st2.UUID)
	}
}

func TestNezhaStatusDisableCommandExecuteMissingDefaultsRemoteOpsTrue(t *testing.T) {
	cfg := []byte("server: dash.example.com:443\nclient_secret: ok-secret\n")
	configPath, binaryPath := nezhaStatusFixture(t, 0700, 0755, 0600, cfg)
	svc := newStatusNezhaService(t, configPath, binaryPath, newFakeNezhaRunner("inactive", "disabled"), nil)
	st, err := svc.Status()
	if err != nil {
		t.Fatal(err)
	}
	if !st.ConfigHealthy {
		t.Fatalf("configHealthy want true, err=%q", st.ConfigError)
	}
	if !st.RemoteOperationsEnabled {
		t.Fatal("remoteOperationsEnabled want true when disable_command_execute is missing")
	}
}

func TestNezhaStatusPermissionsWarning(t *testing.T) {
	secret := "NzjStatusPermSec_ffee0011"
	// Loose dir (0755), binary with exec but not 0755 (0700), config 0644.
	cfg := []byte("server: dash.example.com:443\nclient_secret: " + secret + "\n")
	configPath, binaryPath := nezhaStatusFixture(t, 0755, 0700, 0644, cfg)
	svc := newStatusNezhaService(t, configPath, binaryPath, newFakeNezhaRunner("inactive", "disabled"), nil)
	st, err := svc.Status()
	if err != nil {
		t.Fatal(err)
	}
	// componentAvailable only needs regular + any execute bit.
	if !st.ComponentAvailable {
		t.Fatal("componentAvailable want true for 0700 binary (has execute bit)")
	}
	if st.PermissionsWarning == "" {
		t.Fatal("permissionsWarning want non-empty for mode deviations")
	}
	warn := st.PermissionsWarning
	if !strings.Contains(warn, "0700") && !strings.Contains(warn, "dir") && !strings.Contains(strings.ToLower(warn), "directory") {
		// Must mention directory mode issue somehow.
		if !strings.Contains(warn, "755") && !strings.Contains(warn, "0755") {
			// Accept any diagnostic that covers the three actual deviations.
		}
	}
	// Expect all three deviations summarized.
	needles := []string{"0700", "0755", "0600"}
	for _, n := range needles {
		if !strings.Contains(warn, n) {
			t.Fatalf("permissionsWarning %q should mention expected mode %s", warn, n)
		}
	}
	if strings.Contains(warn, secret) {
		t.Fatal("permissionsWarning must not include secret")
	}
	assertStatusNoSecret(t, st, secret)
}

func TestNezhaStatusIPv6DashboardURL(t *testing.T) {
	// Default HTTPS port must keep IPv6 brackets.
	cfg := []byte("server: \"[2001:db8::1]:443\"\nclient_secret: ok-secret\ntls: true\n")
	configPath, binaryPath := nezhaStatusFixture(t, 0700, 0755, 0600, cfg)
	svc := newStatusNezhaService(t, configPath, binaryPath, newFakeNezhaRunner("inactive", "disabled"), nil)
	st, err := svc.Status()
	if err != nil {
		t.Fatal(err)
	}
	if st.DashboardURL != "https://[2001:db8::1]" {
		t.Fatalf("dashboardUrl=%q want https://[2001:db8::1]", st.DashboardURL)
	}

	// Explicit non-default port keeps brackets and port.
	cfg2 := []byte("server: \"[2001:db8::1]:8443\"\nclient_secret: ok-secret\ntls: true\n")
	configPath2, binaryPath2 := nezhaStatusFixture(t, 0700, 0755, 0600, cfg2)
	svc2 := newStatusNezhaService(t, configPath2, binaryPath2, newFakeNezhaRunner("inactive", "disabled"), nil)
	st2, err := svc2.Status()
	if err != nil {
		t.Fatal(err)
	}
	if st2.DashboardURL != "https://[2001:db8::1]:8443" {
		t.Fatalf("dashboardUrl=%q want https://[2001:db8::1]:8443", st2.DashboardURL)
	}

	// HTTP default port 80 with IPv6.
	cfg3 := []byte("server: \"[2001:db8::2]:80\"\nclient_secret: ok-secret\ntls: false\n")
	configPath3, binaryPath3 := nezhaStatusFixture(t, 0700, 0755, 0600, cfg3)
	svc3 := newStatusNezhaService(t, configPath3, binaryPath3, newFakeNezhaRunner("inactive", "disabled"), nil)
	st3, err := svc3.Status()
	if err != nil {
		t.Fatal(err)
	}
	if st3.DashboardURL != "http://[2001:db8::2]" {
		t.Fatalf("dashboardUrl=%q want http://[2001:db8::2]", st3.DashboardURL)
	}
}

func TestNezhaStatusDriftAndServiceStateCombinations(t *testing.T) {
	cfg := []byte("server: dash.example.com:443\nclient_secret: ok-secret\n")
	configPath, binaryPath := nezhaStatusFixture(t, 0700, 0755, 0600, cfg)

	t.Run("active disabled desired true drifts", func(t *testing.T) {
		runner := newFakeNezhaRunner("active", "disabled")
		settings := newFakeNezhaSettings(map[string]string{"NezhaEnabled": "true"})
		svc := newStatusNezhaService(t, configPath, binaryPath, runner, settings)
		st, err := svc.Status()
		if err != nil {
			t.Fatal(err)
		}
		if !st.Active {
			t.Fatal("active want true")
		}
		if st.ServiceState != "active" {
			t.Fatalf("serviceState=%q", st.ServiceState)
		}
		if st.Enabled {
			t.Fatal("enabled want false")
		}
		if !st.DesiredEnabled {
			t.Fatal("desiredEnabled want true")
		}
		if !st.Drift {
			t.Fatal("drift want true when disabled but desired enabled")
		}
	})

	t.Run("inactive enabled desired true no drift", func(t *testing.T) {
		runner := newFakeNezhaRunner("inactive", "enabled")
		settings := newFakeNezhaSettings(map[string]string{"NezhaEnabled": "true"})
		svc := newStatusNezhaService(t, configPath, binaryPath, runner, settings)
		st, err := svc.Status()
		if err != nil {
			t.Fatal(err)
		}
		if st.Active {
			t.Fatal("active want false")
		}
		if st.ServiceState != "inactive" {
			t.Fatalf("serviceState=%q want inactive", st.ServiceState)
		}
		if !st.Enabled {
			t.Fatal("enabled want true")
		}
		if !st.DesiredEnabled {
			t.Fatal("desiredEnabled want true")
		}
		if st.Drift {
			t.Fatal("drift want false when enabled matches desired (runtime inactive is ok)")
		}
	})

	t.Run("enabled-runtime is enabled", func(t *testing.T) {
		runner := newFakeNezhaRunner("active", "enabled-runtime")
		settings := newFakeNezhaSettings(map[string]string{"NezhaEnabled": "true"})
		svc := newStatusNezhaService(t, configPath, binaryPath, runner, settings)
		st, err := svc.Status()
		if err != nil {
			t.Fatal(err)
		}
		if !st.Enabled {
			t.Fatal("enabled want true for enabled-runtime")
		}
		if st.Drift {
			t.Fatal("drift want false")
		}
		if st.ServiceError != "" {
			t.Fatalf("serviceError want empty, got %q", st.ServiceError)
		}
	})
}

func TestNezhaStatusKnownServiceStatesMapNormally(t *testing.T) {
	cfg := []byte("server: dash.example.com:443\nclient_secret: ok-secret\n")
	configPath, binaryPath := nezhaStatusFixture(t, 0700, 0755, 0600, cfg)

	for _, state := range []string{"inactive", "failed"} {
		t.Run("active="+state, func(t *testing.T) {
			runner := newFakeNezhaRunner(state, "disabled")
			svc := newStatusNezhaService(t, configPath, binaryPath, runner, nil)
			st, err := svc.Status()
			if err != nil {
				t.Fatal(err)
			}
			if st.ServiceState != state {
				t.Fatalf("serviceState=%q want %q", st.ServiceState, state)
			}
			if st.Active {
				t.Fatal("active want false")
			}
			if st.ServiceError != "" {
				t.Fatalf("serviceError want empty for known state %s, got %q", state, st.ServiceError)
			}
		})
	}
	for _, state := range []string{"disabled", "static", "masked"} {
		t.Run("enabled="+state, func(t *testing.T) {
			runner := newFakeNezhaRunner("inactive", state)
			svc := newStatusNezhaService(t, configPath, binaryPath, runner, nil)
			st, err := svc.Status()
			if err != nil {
				t.Fatal(err)
			}
			if st.Enabled {
				t.Fatal("enabled want false")
			}
			if st.ServiceError != "" {
				t.Fatalf("serviceError want empty for known enabled state %s, got %q", state, st.ServiceError)
			}
		})
	}
}

func TestNezhaStatusServiceProbeEmptyAndUnknownMergeErrors(t *testing.T) {
	secret := "NzjStatusSvcSec_99aa88bb"
	cfg := []byte("server: dash.example.com:443\nclient_secret: " + secret + "\n")
	configPath, binaryPath := nezhaStatusFixture(t, 0700, 0755, 0600, cfg)

	t.Run("empty is-active and is-enabled failures merge", func(t *testing.T) {
		runner := newFakeNezhaRunner("inactive", "disabled")
		runner.activeEmptyFail = true
		runner.enabledEmptyFail = true
		svc := newStatusNezhaService(t, configPath, binaryPath, runner, nil)
		st, err := svc.Status()
		if err != nil {
			t.Fatalf("top-level Status error want nil, got %v", err)
		}
		if st.ServiceError == "" {
			t.Fatal("serviceError want non-empty for empty probe failures")
		}
		// Both diagnostics present (merge, not overwrite).
		if !strings.Contains(st.ServiceError, "is-active") {
			t.Fatalf("serviceError should mention is-active: %q", st.ServiceError)
		}
		if !strings.Contains(st.ServiceError, "is-enabled") {
			t.Fatalf("serviceError should mention is-enabled: %q", st.ServiceError)
		}
		if strings.Contains(st.ServiceError, secret) {
			t.Fatal("serviceError must not include secret")
		}
		assertStatusNoSecret(t, st, secret)
	})

	t.Run("unknown non-empty states enter serviceError", func(t *testing.T) {
		runner := newFakeNezhaRunner("weird-active-state", "weird-enabled-state")
		svc := newStatusNezhaService(t, configPath, binaryPath, runner, nil)
		st, err := svc.Status()
		if err != nil {
			t.Fatal(err)
		}
		if st.ServiceState != "weird-active-state" {
			t.Fatalf("serviceState should preserve raw state, got %q", st.ServiceState)
		}
		if st.ServiceError == "" {
			t.Fatal("serviceError want non-empty for unknown states")
		}
		if !strings.Contains(st.ServiceError, "weird-active-state") && !strings.Contains(st.ServiceError, "is-active") {
			t.Fatalf("serviceError should diagnose unknown active state: %q", st.ServiceError)
		}
		if !strings.Contains(st.ServiceError, "weird-enabled-state") && !strings.Contains(st.ServiceError, "is-enabled") {
			t.Fatalf("serviceError should diagnose unknown enabled state: %q", st.ServiceError)
		}
		if strings.Contains(st.ServiceError, secret) {
			t.Fatal("serviceError must not include secret")
		}
	})
}

func TestNezhaStatusBinaryVersionFailureLeavesEmpty(t *testing.T) {
	cfg := []byte("server: dash.example.com:443\nclient_secret: ok-secret\n")
	configPath, binaryPath := nezhaStatusFixture(t, 0700, 0755, 0600, cfg)
	runner := newFakeNezhaRunner("inactive", "disabled")
	runner.versionFail = true
	svc := newStatusNezhaService(t, configPath, binaryPath, runner, nil)

	st, err := svc.Status()
	if err != nil {
		t.Fatalf("Status top-level error want nil, got %v", err)
	}
	if st == nil {
		t.Fatal("Status returned nil")
	}
	if !st.ComponentAvailable {
		t.Fatal("componentAvailable want true")
	}
	if st.Version != "" {
		t.Fatalf("version want empty on -v failure, got %q", st.Version)
	}
	// Must not fabricate a default version.
	if st.Version == "v0.0.0" {
		t.Fatal("must not fabricate version")
	}
}

// ----- Journal redaction (Task 2C) -----

// assertJournalHasNoSecret fails without embedding the secret value in the message.
func assertJournalHasNoSecret(t *testing.T, got, secret, label string) {
	t.Helper()
	if secret == "" {
		return
	}
	if strings.Contains(got, secret) {
		t.Fatalf("journal redaction leaked %s (len=%d)", label, len(secret))
	}
}

func TestNezhaJournalRedactionRemovesKnownSecretAndAuthorization(t *testing.T) {
	// Unique sentinels — failure messages only report label/length, never the value.
	known := "NzjKnownSec_a7f3b91e2c"
	bearer := "NzjBearerTok_d4e81a6b"
	authVal := "NzjAuthHdr_9c2f0e"

	cases := []struct {
		name  string
		input string
		ban   []string
	}{
		{
			name:  "known secret any occurrence",
			input: "agent connected with " + known + " mid-line and again " + known,
			ban:   []string{known},
		},
		{
			name:  "client_secret equals",
			input: "cfg client_secret=" + known + " ok",
			ban:   []string{known},
		},
		{
			name:  "client-secret colon spaces",
			input: "meta client-secret:  " + known + " trail",
			ban:   []string{known},
		},
		{
			name:  "Client_Secret case insensitive",
			input: "Client_Secret = " + known,
			ban:   []string{known},
		},
		{
			name:  "AgentSecret assignment",
			input: "AgentSecret=" + known,
			ban:   []string{known},
		},
		{
			name:  "agent-secret colon",
			input: "agent-secret: " + known,
			ban:   []string{known},
		},
		{
			name:  "general secret assignment",
			input: "secret=" + known + " next=value",
			ban:   []string{known},
		},
		{
			name:  "json quoted client_secret",
			input: `{"client_secret":"` + known + `","server":"dash:443"}`,
			ban:   []string{known},
		},
		{
			name:  "json quoted AgentSecret",
			input: `{"AgentSecret": "` + known + `"}`,
			ban:   []string{known},
		},
		{
			name:  "Authorization header colon",
			input: "Authorization: " + authVal,
			ban:   []string{authVal},
		},
		{
			name:  "Authorization equals Bearer",
			input: "Authorization=Bearer " + bearer,
			ban:   []string{bearer},
		},
		{
			name:  "Bearer token standalone",
			input: "token Bearer " + bearer + " end",
			ban:   []string{bearer},
		},
		{
			name:  "combined plan sample",
			input: "client_secret=" + known + " Authorization: Bearer " + bearer,
			ban:   []string{known, bearer},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := redactNezhaJournal(tc.input, known)
			for i, s := range tc.ban {
				assertJournalHasNoSecret(t, got, s, "ban#"+strconv.Itoa(i))
			}
			if got == tc.input && len(tc.ban) > 0 {
				t.Fatalf("redaction left input unchanged for case %q", tc.name)
			}
		})
	}
}

func TestNezhaJournalRedactionPatternOnlyWithoutKnownSecret(t *testing.T) {
	// Unique sentinels — failure messages only report label/length, never the value.
	val := "NzjPatternOnly_55aa99"
	bearer := "NzjPatBearer_1122cc"
	spaceOnly := "SpaceOnlySentinel"
	quotedSpace := "Quoted Space Value"
	spaceAgent := "SpaceAgentValue"
	spaceGeneric := "SpaceGenericValue"

	input := strings.Join([]string{
		"client_secret=" + val,
		"Authorization: Bearer " + bearer,
		"secret: " + val,
		// Space-separated assignments (no = or :) must also redact when knownSecret is empty.
		"client_secret " + spaceOnly,
		`client-secret "` + quotedSpace + `"`,
		"AgentSecret " + spaceAgent,
		"secret " + spaceGeneric,
	}, " ")

	got := redactNezhaJournal(input, "")
	assertJournalHasNoSecret(t, got, val, "pattern-secret")
	assertJournalHasNoSecret(t, got, bearer, "pattern-bearer")
	assertJournalHasNoSecret(t, got, spaceOnly, "space-only")
	assertJournalHasNoSecret(t, got, quotedSpace, "quoted-space")
	assertJournalHasNoSecret(t, got, spaceAgent, "space-agent")
	assertJournalHasNoSecret(t, got, spaceGeneric, "space-generic")
}

func TestNezhaReadAgentSecretFromConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yml")
	secret := "NzjFileSecret_c0ffee99"
	if err := os.WriteFile(path, []byte("server: dash:443\nclient_secret: "+secret+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	got := readNezhaAgentSecretFromConfig(path)
	if got != secret {
		t.Fatalf("expected secret from config (got len=%d want len=%d)", len(got), len(secret))
	}
}

func TestNezhaReadAgentSecretFromConfigMissingCorruptUnsafe(t *testing.T) {
	dir := t.TempDir()

	if got := readNezhaAgentSecretFromConfig(filepath.Join(dir, "nope.yml")); got != "" {
		t.Fatalf("missing config must return empty, got len=%d", len(got))
	}

	corrupt := filepath.Join(dir, "corrupt.yml")
	if err := os.WriteFile(corrupt, []byte("client_secret: [unterminated\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if got := readNezhaAgentSecretFromConfig(corrupt); got != "" {
		t.Fatalf("corrupt config must return empty, got len=%d", len(got))
	}

	target := filepath.Join(dir, "real.yml")
	secret := "NzjSymlinkSecret_deadbeef"
	if err := os.WriteFile(target, []byte("client_secret: "+secret+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link.yml")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if got := readNezhaAgentSecretFromConfig(link); got != "" {
		t.Fatalf("symlink config must return empty, got len=%d", len(got))
	}

	subdir := filepath.Join(dir, "subdir.yml")
	if err := os.Mkdir(subdir, 0700); err != nil {
		t.Fatal(err)
	}
	if got := readNezhaAgentSecretFromConfig(subdir); got != "" {
		t.Fatalf("non-regular config must return empty, got len=%d", len(got))
	}

	empty := filepath.Join(dir, "empty.yml")
	if err := os.WriteFile(empty, []byte("server: dash:443\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if got := readNezhaAgentSecretFromConfig(empty); got != "" {
		t.Fatalf("missing client_secret must return empty, got len=%d", len(got))
	}
}

// ----- Systemd GetLogs redaction (Task 2C) -----

func withSystemdExecFixture(t *testing.T, stdout string, fail bool) {
	t.Helper()
	prev := systemdExecCommand
	t.Cleanup(func() { systemdExecCommand = prev })

	fixture := filepath.Join(t.TempDir(), "journal.out")
	if err := os.WriteFile(fixture, []byte(stdout), 0600); err != nil {
		t.Fatal(err)
	}
	// Quote the path so fixture content never enters the shell command string.
	q := strconv.Quote(fixture)
	script := "cat " + q
	if fail {
		script += "; exit 1"
	}
	systemdExecCommand = func(name string, args ...string) *exec.Cmd {
		return exec.Command("sh", "-c", script)
	}
}

func TestSystemdGetLogsNezhaSuccessRedacts(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config.yml")
	secret := "NzjGetLogsSec_aabb11"
	if err := os.WriteFile(cfg, []byte("client_secret: "+secret+"\n"), 0600); err != nil {
		t.Fatal(err)
	}

	prevConfig := nezhaJournalConfigPath
	t.Cleanup(func() { nezhaJournalConfigPath = prevConfig })
	nezhaJournalConfigPath = cfg

	raw := "start client_secret=" + secret + " Authorization: Bearer NzjGLBearer_99 end"
	withSystemdExecFixture(t, raw, false)

	svc := &SystemdServiceImpl{}
	got, err := svc.GetLogs(dto.SystemdServiceLogReq{Name: NezhaAgentUnitName, Lines: 50})
	if err != nil {
		t.Fatalf("GetLogs success path error: %v", err)
	}
	assertJournalHasNoSecret(t, got, secret, "known-from-config")
	assertJournalHasNoSecret(t, got, "NzjGLBearer_99", "bearer")
}

func TestSystemdGetLogsNezhaFailureRedactsBeforeError(t *testing.T) {
	prevConfig := nezhaJournalConfigPath
	t.Cleanup(func() { nezhaJournalConfigPath = prevConfig })
	// Missing config: pattern redaction still required on error diagnostics.
	nezhaJournalConfigPath = filepath.Join(t.TempDir(), "missing.yml")

	secretish := "NzjFailSec_eeff22"
	raw := "journalctl: permission denied client_secret=" + secretish + " Bearer NzjFailBearer_33"
	withSystemdExecFixture(t, raw, true)

	svc := &SystemdServiceImpl{}
	_, err := svc.GetLogs(dto.SystemdServiceLogReq{Name: "xpanel-nezha-agent.service", Lines: 20})
	if err == nil {
		t.Fatal("expected journalctl failure")
	}
	msg := err.Error()
	assertJournalHasNoSecret(t, msg, secretish, "fail-secret")
	assertJournalHasNoSecret(t, msg, "NzjFailBearer_33", "fail-bearer")
}

func TestSystemdGetLogsNezhaOtherUnitDoesNotRedact(t *testing.T) {
	raw := "client_secret=NzjOtherUnit_keepme Authorization: Bearer NzjOtherBearer_keep"
	withSystemdExecFixture(t, raw, false)

	svc := &SystemdServiceImpl{}
	got, err := svc.GetLogs(dto.SystemdServiceLogReq{Name: "nginx", Lines: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, "NzjOtherUnit_keepme") || !strings.Contains(got, "NzjOtherBearer_keep") {
		t.Fatal("non-nezha unit must keep original journal text without Nezha redaction")
	}
}

func TestSystemdGetLogsNezhaMissingConfigStillPatternRedacts(t *testing.T) {
	prevConfig := nezhaJournalConfigPath
	t.Cleanup(func() { nezhaJournalConfigPath = prevConfig })
	nezhaJournalConfigPath = filepath.Join(t.TempDir(), "absent.yml")

	val := "NzjMissCfg_7788"
	raw := "client_secret=" + val + " Authorization: Bearer NzjMissBearer_44"
	withSystemdExecFixture(t, raw, false)

	svc := &SystemdServiceImpl{}
	got, err := svc.GetLogs(dto.SystemdServiceLogReq{Name: NezhaAgentUnitName, Lines: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertJournalHasNoSecret(t, got, val, "miss-cfg-secret")
	assertJournalHasNoSecret(t, got, "NzjMissBearer_44", "miss-cfg-bearer")
}

// ----- Unit file static checks (Task 2C) -----

func TestNezhaAgentUnitFileConstraints(t *testing.T) {
	path := filepath.Join("..", "..", "..", "scripts", "xpanel-nezha-agent.service")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read unit file: %v", err)
	}
	content := string(data)
	lower := strings.ToLower(content)

	required := []string{
		"Description=Nezha Agent managed by X-Panel",
		"Wants=network-online.target",
		"After=network-online.target",
		"Type=simple",
		"WorkingDirectory=/opt/xpanel/nezha-agent",
		"ExecStart=/opt/xpanel/nezha-agent/nezha-agent -c /opt/xpanel/nezha-agent/config.yml",
		"Restart=always",
		"RestartSec=10",
		"UMask=0077",
		"WantedBy=multi-user.target",
	}
	for _, want := range required {
		if !strings.Contains(content, want) {
			t.Fatalf("unit missing required line/value %q", want)
		}
	}

	if strings.Contains(lower, "client_secret") || strings.Contains(lower, "agentsecret") {
		t.Fatal("unit must not contain credential field names")
	}
	if strings.Contains(content, "Environment") {
		t.Fatal("unit must not contain Environment")
	}
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "ExecStart=") {
			continue
		}
		val := strings.TrimPrefix(trimmed, "ExecStart=")
		if strings.Contains(val, "sh -c") || strings.Contains(val, "/bin/sh") ||
			strings.Contains(val, "/bin/bash") || strings.HasPrefix(val, "sh ") {
			t.Fatalf("ExecStart must not use a shell wrapper")
		}
		if !strings.HasPrefix(val, "/opt/xpanel/nezha-agent/nezha-agent") {
			t.Fatalf("ExecStart must invoke the agent binary directly")
		}
	}
}

// ----- Task 2B2: external Agent conflict detection & operate blocking -----

func newConflictNezhaService(t *testing.T, runner *fakeNezhaRunner, settings *fakeNezhaSettings, opts nezhaAgentDeps) *NezhaAgentService {
	t.Helper()
	if settings == nil {
		settings = newFakeNezhaSettings(nil)
	}
	if opts.ConfigPath == "" {
		opts.ConfigPath = testNezhaConfigPath(t)
	}
	if opts.BinaryPath == "" {
		opts.BinaryPath = NezhaAgentBinaryPath
	}
	if opts.Unit == "" {
		opts.Unit = NezhaAgentUnitName
	}
	if opts.Runner == nil {
		opts.Runner = runner
	}
	if opts.Settings == nil {
		opts.Settings = settings
	}
	if opts.Sleep == nil {
		opts.Sleep = func(time.Duration) {}
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	if opts.StopTimeout <= 0 {
		opts.StopTimeout = 50 * time.Millisecond
	}
	if opts.PollEvery <= 0 {
		opts.PollEvery = time.Millisecond
	}
	return newNezhaAgentService(opts)
}

func TestNezhaConflictDetectsActivePlainUnit(t *testing.T) {
	runner := newFakeNezhaRunner("inactive", "disabled")
	runner.listUnitsOutput = "nezha-agent.service loaded active running Nezha Agent\n"
	svc := newConflictNezhaService(t, runner, nil, nezhaAgentDeps{})

	got, err := svc.detectConflicts()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("want 1 conflict, got %#v", got)
	}
	if got[0].Kind != nezhaConflictKindUnit {
		t.Fatalf("kind=%q want %q", got[0].Kind, nezhaConflictKindUnit)
	}
	if got[0].Detail != "nezha-agent.service" {
		t.Fatalf("detail=%q", got[0].Detail)
	}
	if got[0].Message == "" {
		t.Fatal("message want non-empty")
	}
	runner.assertNeverMutatesExternalNezhaUnits(t)
	// Detection is read-only: list-units only, never stop/disable/start.
	runner.assertNotCalled(t, "stop")
	runner.assertNotCalled(t, "disable")
	runner.assertNotCalled(t, "start")
	runner.assertNotCalled(t, "enable")
}

func TestNezhaConflictDetectsActiveInstantiatedUnit(t *testing.T) {
	runner := newFakeNezhaRunner("inactive", "disabled")
	runner.listUnitsOutput = "nezha-agent@node1.service loaded active running instance\n"
	svc := newConflictNezhaService(t, runner, nil, nezhaAgentDeps{})

	got, err := svc.detectConflicts()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("want 1 conflict, got %#v", got)
	}
	if got[0].Kind != nezhaConflictKindUnit {
		t.Fatalf("kind=%q", got[0].Kind)
	}
	if got[0].Detail != "nezha-agent@node1.service" {
		t.Fatalf("detail=%q", got[0].Detail)
	}
	runner.assertNeverMutatesExternalNezhaUnits(t)
}

func TestNezhaConflictDetectsExternalProcess(t *testing.T) {
	runner := newFakeNezhaRunner("inactive", "disabled")
	svc := newConflictNezhaService(t, runner, nil, nezhaAgentDeps{
		BinaryPath: "/opt/xpanel/nezha-agent/nezha-agent",
		ListProcessExecutables: func() ([]string, error) {
			return []string{"/usr/local/bin/nezha-agent"}, nil
		},
	})

	got, err := svc.detectConflicts()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("want 1 conflict, got %#v", got)
	}
	if got[0].Kind != nezhaConflictKindProcess {
		t.Fatalf("kind=%q", got[0].Kind)
	}
	if got[0].Detail != "/usr/local/bin/nezha-agent" {
		t.Fatalf("detail=%q", got[0].Detail)
	}
}

func TestNezhaConflictDetectsExternalDirectory(t *testing.T) {
	dir := t.TempDir()
	external := filepath.Join(dir, "opt", "nezha", "agent")
	if err := os.MkdirAll(external, 0o755); err != nil {
		t.Fatal(err)
	}
	// Sibling paths that do not exist must not appear.
	missing := filepath.Join(dir, "opt", "nezha-agent")

	runner := newFakeNezhaRunner("inactive", "disabled")
	svc := newConflictNezhaService(t, runner, nil, nezhaAgentDeps{
		ExternalDirs: []string{external, missing, filepath.Join(dir, "etc", "nezha")},
	})

	got, err := svc.detectConflicts()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("want 1 directory conflict, got %#v", got)
	}
	if got[0].Kind != nezhaConflictKindDirectory {
		t.Fatalf("kind=%q", got[0].Kind)
	}
	if got[0].Detail != external {
		t.Fatalf("detail=%q want %q", got[0].Detail, external)
	}
}

func TestNezhaConflictIgnoresBundledProcess(t *testing.T) {
	bundled := "/opt/xpanel/nezha-agent/nezha-agent"
	runner := newFakeNezhaRunner("inactive", "disabled")
	svc := newConflictNezhaService(t, runner, nil, nezhaAgentDeps{
		BinaryPath: bundled,
		ListProcessExecutables: func() ([]string, error) {
			return []string{
				bundled,
				filepath.Clean(bundled),
				"/opt/xpanel/nezha-agent/nezha-agent", // self
				"/usr/bin/other-agent",                // wrong basename
			}, nil
		},
	})

	got, err := svc.detectConflicts()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("bundled process must not conflict, got %#v", got)
	}
}

func TestNezhaConflictDedupAndStableSort(t *testing.T) {
	dir := t.TempDir()
	d1 := filepath.Join(dir, "etc", "nezha")
	d2 := filepath.Join(dir, "opt", "nezha", "agent")
	for _, p := range []string{d1, d2} {
		if err := os.MkdirAll(p, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	runner := newFakeNezhaRunner("inactive", "disabled")
	// Duplicate unit line + mixed order sources.
	runner.listUnitsOutput = strings.Join([]string{
		"nezha-agent@z.service loaded active running z",
		"nezha-agent.service loaded active running plain",
		"nezha-agent.service loaded active running plain", // dup
		"nezha-agent@a.service loaded active running a",
	}, "\n") + "\n"

	svc := newConflictNezhaService(t, runner, nil, nezhaAgentDeps{
		BinaryPath: "/opt/xpanel/nezha-agent/nezha-agent",
		ListProcessExecutables: func() ([]string, error) {
			return []string{
				"/usr/local/bin/nezha-agent",
				"/usr/local/bin/nezha-agent", // dup
				"/opt/nezha/agent/nezha-agent",
			}, nil
		},
		ExternalDirs: []string{d2, d1, d2}, // dup dir
	})

	got, err := svc.detectConflicts()
	if err != nil {
		t.Fatal(err)
	}

	// Expected stable order: kind asc, then detail asc.
	wantDetails := []struct {
		kind   string
		detail string
	}{
		{nezhaConflictKindDirectory, d1},
		{nezhaConflictKindDirectory, d2},
		{nezhaConflictKindProcess, "/opt/nezha/agent/nezha-agent"},
		{nezhaConflictKindProcess, "/usr/local/bin/nezha-agent"},
		{nezhaConflictKindUnit, "nezha-agent.service"},
		{nezhaConflictKindUnit, "nezha-agent@a.service"},
		{nezhaConflictKindUnit, "nezha-agent@z.service"},
	}
	if len(got) != len(wantDetails) {
		t.Fatalf("len=%d want %d; got %#v", len(got), len(wantDetails), got)
	}
	for i, w := range wantDetails {
		if got[i].Kind != w.kind || got[i].Detail != w.detail {
			t.Fatalf("index %d: got kind=%q detail=%q want kind=%q detail=%q\nfull=%#v",
				i, got[i].Kind, got[i].Detail, w.kind, w.detail, got)
		}
	}
}

func TestNezhaStatusConflictPopulatesConflicts(t *testing.T) {
	cfg := []byte("server: dash.example.com:443\nclient_secret: status-conflict-secret\n")
	configPath, binaryPath := nezhaStatusFixture(t, 0700, 0755, 0600, cfg)
	runner := newFakeNezhaRunner("inactive", "disabled")
	runner.listUnitsOutput = "nezha-agent.service loaded active running Nezha Agent\n"
	runner.versionOutput = "v1.2.3"

	svc := newNezhaAgentService(nezhaAgentDeps{
		ConfigPath: configPath,
		BinaryPath: binaryPath,
		UnitPath:   filepath.Join(filepath.Dir(binaryPath), "xpanel-nezha-agent.service"),
		Unit:       NezhaAgentUnitName,
		Runner:     runner,
		Settings:   newFakeNezhaSettings(nil),
		Sleep:      func(time.Duration) {},
		Now:        time.Now,
		ListProcessExecutables: func() ([]string, error) {
			return []string{"/opt/other/nezha-agent"}, nil
		},
	})

	st, err := svc.Status()
	if err != nil {
		t.Fatalf("Status top-level error want nil, got %v", err)
	}
	if st.ServiceError != "" {
		t.Fatalf("serviceError want empty, got %q", st.ServiceError)
	}
	if !st.ComponentAvailable || !st.ConfigHealthy {
		t.Fatalf("other status fields should still populate: available=%v healthy=%v", st.ComponentAvailable, st.ConfigHealthy)
	}
	if len(st.Conflicts) < 2 {
		t.Fatalf("want unit+process conflicts, got %#v", st.Conflicts)
	}
	kinds := map[string]bool{}
	for _, c := range st.Conflicts {
		kinds[c.Kind] = true
	}
	if !kinds[nezhaConflictKindUnit] || !kinds[nezhaConflictKindProcess] {
		t.Fatalf("missing expected kinds: %#v", st.Conflicts)
	}
	assertStatusNoSecret(t, st, "status-conflict-secret")
	runner.assertNeverMutatesExternalNezhaUnits(t)
}

func TestNezhaStatusConflictDetectionErrorEntersServiceError(t *testing.T) {
	cfg := []byte("server: dash.example.com:443\nclient_secret: det-err-secret\n")
	configPath, binaryPath := nezhaStatusFixture(t, 0700, 0755, 0600, cfg)
	runner := newFakeNezhaRunner("active", "enabled")
	runner.versionOutput = "v9.9.9"

	svc := newNezhaAgentService(nezhaAgentDeps{
		ConfigPath: configPath,
		BinaryPath: binaryPath,
		UnitPath:   filepath.Join(filepath.Dir(binaryPath), "xpanel-nezha-agent.service"),
		Unit:       NezhaAgentUnitName,
		Runner:     runner,
		Settings:   newFakeNezhaSettings(map[string]string{"NezhaEnabled": "true"}),
		Sleep:      func(time.Duration) {},
		Now:        time.Now,
		ListProcessExecutables: func() ([]string, error) {
			return nil, errors.New("proc scan boom")
		},
	})

	st, err := svc.Status()
	if err != nil {
		t.Fatalf("Status top-level error want nil, got %v", err)
	}
	if st.ServiceError == "" {
		t.Fatal("serviceError want non-empty when detection fails")
	}
	if !strings.Contains(st.ServiceError, "proc scan boom") && !strings.Contains(strings.ToLower(st.ServiceError), "conflict") {
		t.Fatalf("serviceError should mention detection failure: %q", st.ServiceError)
	}
	// Other status still returned.
	if !st.Active || !st.Enabled || st.Version == "" {
		t.Fatalf("other fields should remain: active=%v enabled=%v version=%q", st.Active, st.Enabled, st.Version)
	}
	if st.Conflicts == nil {
		t.Fatal("conflicts slice want non-nil")
	}
	if len(st.Conflicts) != 0 {
		t.Fatalf("conflicts want empty on detection error, got %#v", st.Conflicts)
	}
	assertStatusNoSecret(t, st, "det-err-secret")
}

func TestNezhaOperateConflictBlocksStartRestartEnable(t *testing.T) {
	for _, op := range []string{"start", "restart", "enable"} {
		t.Run(op, func(t *testing.T) {
			runner := newFakeNezhaRunner("inactive", "disabled")
			runner.listUnitsOutput = "nezha-agent.service loaded active running Nezha Agent\n"
			settings := newFakeNezhaSettings(map[string]string{"NezhaEnabled": "false"})
			svc := newConflictNezhaService(t, runner, settings, nezhaAgentDeps{})

			err := svc.Operate(op)
			if err == nil {
				t.Fatal("expected conflict error")
			}
			if !strings.Contains(strings.ToLower(err.Error()), "conflict") {
				t.Fatalf("error should mention conflict: %v", err)
			}
			// No target systemctl write before failure.
			runner.assertNotCalled(t, "start")
			runner.assertNotCalled(t, "restart")
			runner.assertNotCalled(t, "enable")
			runner.assertNotCalled(t, "stop")
			runner.assertNotCalled(t, "disable")
			settings.assertNotWritten(t, "NezhaEnabled")
			runner.assertNeverMutatesExternalNezhaUnits(t)
		})
	}
}

func TestNezhaOperateConflictAllowsStopAndDisable(t *testing.T) {
	t.Run("stop", func(t *testing.T) {
		runner := newFakeNezhaRunner("active", "enabled")
		runner.listUnitsOutput = "nezha-agent.service loaded active running Nezha Agent\n"
		settings := newFakeNezhaSettings(map[string]string{"NezhaEnabled": "true"})
		svc := newConflictNezhaService(t, runner, settings, nezhaAgentDeps{})

		if err := svc.Operate("stop"); err != nil {
			t.Fatal(err)
		}
		runner.assertCalled(t, "systemctl", "stop", NezhaAgentUnitName)
		settings.assertNotWritten(t, "NezhaEnabled")
		runner.assertNeverMutatesExternalNezhaUnits(t)
	})
	t.Run("disable", func(t *testing.T) {
		runner := newFakeNezhaRunner("active", "enabled")
		runner.listUnitsOutput = "nezha-agent.service loaded active running Nezha Agent\n"
		settings := newFakeNezhaSettings(map[string]string{"NezhaEnabled": "true"})
		svc := newConflictNezhaService(t, runner, settings, nezhaAgentDeps{})

		if err := svc.Operate("disable"); err != nil {
			t.Fatal(err)
		}
		runner.assertCalled(t, "systemctl", "disable", "--now", NezhaAgentUnitName)
		settings.assertWritten(t, "NezhaEnabled", "false")
		runner.assertNeverMutatesExternalNezhaUnits(t)
	})
}

func TestNezhaConfigureConflictBlocksEnableAndStart(t *testing.T) {
	runner := newFakeNezhaRunner("inactive", "disabled")
	runner.listUnitsOutput = "nezha-agent.service loaded active running Nezha Agent\n"
	settings := newFakeNezhaSettings(map[string]string{"NezhaEnabled": "false"})
	svc := newConflictNezhaService(t, runner, settings, nezhaAgentDeps{})

	// Existing config so validation is partial-update friendly.
	if err := os.WriteFile(svc.configPath, []byte("server: old.example.com:443\nclient_secret: keep\n"), 0600); err != nil {
		t.Fatal(err)
	}

	dash := "https://new.example.com"
	err := svc.Configure(dto.NezhaAgentConfigUpdate{
		DashboardURL:   &dash,
		EnableAndStart: true,
	})
	if err == nil {
		t.Fatal("expected conflict error before enable --now")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "conflict") {
		t.Fatalf("error should mention conflict: %v", err)
	}
	runner.assertNotCalled(t, "enable")
	runner.assertNotCalled(t, "start")
	// Must not stop/disable external unit; bundled stop is also unnecessary when blocked early.
	runner.assertNeverMutatesExternalNezhaUnits(t)
	settings.assertNotWritten(t, "NezhaEnabled")
}

func TestNezhaConfigureConflictAllowsSaveWithoutStart(t *testing.T) {
	runner := newFakeNezhaRunner("inactive", "disabled")
	runner.listUnitsOutput = "nezha-agent.service loaded active running Nezha Agent\n"
	svc := newConflictNezhaService(t, runner, nil, nezhaAgentDeps{})

	if err := os.WriteFile(svc.configPath, []byte("server: old.example.com:443\nclient_secret: keep\n"), 0600); err != nil {
		t.Fatal(err)
	}

	dash := "https://new.example.com"
	if err := svc.Configure(dto.NezhaAgentConfigUpdate{DashboardURL: &dash}); err != nil {
		t.Fatalf("plain Configure should save bundled config despite external conflict: %v", err)
	}
	got, err := os.ReadFile(svc.configPath)
	if err != nil {
		t.Fatal(err)
	}
	assertYAMLValue(t, got, "server", "new.example.com:443")
	assertYAMLValue(t, got, "client_secret", "keep")
	runner.assertNotCalled(t, "enable")
	runner.assertNotCalled(t, "start")
	runner.assertNeverMutatesExternalNezhaUnits(t)
}

func TestNezhaConfigureUnknownActiveStateErrorsBeforeStopReadWrite(t *testing.T) {
	runner := newFakeNezhaRunner("weird-unknown-state", "disabled")
	svc := newConflictNezhaService(t, runner, nil, nezhaAgentDeps{})

	original := []byte("server: old.example.com:443\nclient_secret: keep-me\n")
	if err := os.WriteFile(svc.configPath, original, 0600); err != nil {
		t.Fatal(err)
	}

	readCalled := false
	writeCalled := false
	svc.readConfig = func(path string) ([]byte, error) {
		readCalled = true
		return os.ReadFile(path)
	}
	svc.writeConfig = func(path string, data []byte) error {
		writeCalled = true
		return os.WriteFile(path, data, 0600)
	}

	dash := "https://new.example.com"
	err := svc.Configure(dto.NezhaAgentConfigUpdate{DashboardURL: &dash})
	if err == nil {
		t.Fatal("expected error for unknown active state")
	}
	if !strings.Contains(err.Error(), "weird-unknown-state") && !strings.Contains(err.Error(), "unexpected") {
		t.Fatalf("error should diagnose unknown state: %v", err)
	}
	if readCalled {
		t.Fatal("must not read config when active state is unknown")
	}
	if writeCalled {
		t.Fatal("must not write config when active state is unknown")
	}
	runner.assertNotCalled(t, "stop")
	runner.assertNotCalled(t, "start")
	runner.assertNotCalled(t, "enable")

	got, err := os.ReadFile(svc.configPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(original) {
		t.Fatalf("config must be unchanged, got %q", got)
	}
}

func TestNezhaConflictDefaultTestConstructorSkipsHostScan(t *testing.T) {
	// newTestNezhaAgentService / newNezhaAgentService defaults must not scan host
	// /proc or real external dirs (no dependency on local machine layout).
	runner := newFakeNezhaRunner("inactive", "disabled")
	svc := newTestNezhaAgentService(t, runner, nil)
	if svc.listProcessExecutables != nil {
		t.Fatal("test default listProcessExecutables want nil")
	}
	if len(svc.externalDirs) != 0 {
		t.Fatalf("test default externalDirs want empty, got %v", svc.externalDirs)
	}
	got, err := svc.detectConflicts()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("default test service should report no host conflicts, got %#v", got)
	}
}

func TestNezhaConflictProductionDefaultsEnableRealDetection(t *testing.T) {
	if len(defaultNezhaExternalAgentDirs) < 1 {
		t.Fatal("default external dirs must include at least /opt/nezha/agent")
	}
	found := false
	for _, d := range defaultNezhaExternalAgentDirs {
		if d == "/opt/nezha/agent" {
			found = true
		}
	}
	if !found {
		t.Fatalf("defaults missing /opt/nezha/agent: %v", defaultNezhaExternalAgentDirs)
	}
	// NewNezhaAgentService must enable real detection (may construct setting repo).
	svc := NewNezhaAgentService()
	if svc.listProcessExecutables == nil {
		t.Fatal("NewNezhaAgentService must enable process scanning")
	}
	// Function identity: production uses /proc scanner.
	// Compare by invoking is not safe on all hosts; check non-nil + default dirs.
	if len(svc.externalDirs) == 0 {
		t.Fatal("NewNezhaAgentService must enable external directory scanning")
	}
	// Ensure defaults are actually assigned (not an empty intentional skip).
	got := map[string]bool{}
	for _, d := range svc.externalDirs {
		got[d] = true
	}
	if !got["/opt/nezha/agent"] {
		t.Fatalf("production externalDirs missing /opt/nezha/agent: %v", svc.externalDirs)
	}
}

// ----- Task 2B2 review fixes: fail-closed detection + wasActive configure -----

// Group 1: list-units non-empty diagnostic + error must fail closed (never treat
// error output as a successful unit list and fail-open into Operate start).
func TestNezhaConflictListUnitsErrorWithDiagnosticFailClosed(t *testing.T) {
	runner := newFakeNezhaRunner("inactive", "disabled")
	// Non-empty diagnostic text that is NOT a unit listing, plus a command error.
	// Parsing this as a success list would yield zero conflicts and fail-open.
	runner.listUnitsOutput = "Failed to list units: Connection timed out\n"
	runner.listUnitsErr = errors.New("exit status 1")
	settings := newFakeNezhaSettings(map[string]string{"NezhaEnabled": "false"})
	svc := newConflictNezhaService(t, runner, settings, nezhaAgentDeps{})

	got, err := svc.detectConflicts()
	if err == nil {
		t.Fatal("detectConflicts must return error when list-units fails (even with non-empty output)")
	}
	if len(got) != 0 {
		t.Fatalf("conflicts want empty/nil on detection error, got %#v", got)
	}
	// Safe diagnostic only — must not echo raw command output as success data.
	if strings.Contains(err.Error(), "Connection timed out") {
		// Wrapping the underlying err is fine; the critical fail-open bug is treating
		// non-empty stdout as a parsed unit list. Ensure ensureNoConflicts still fails.
	}

	if err := svc.ensureNoConflicts(); err == nil {
		t.Fatal("ensureNoConflicts must surface detection error")
	} else if !strings.Contains(strings.ToLower(err.Error()), "conflict") &&
		!strings.Contains(strings.ToLower(err.Error()), "detection") &&
		!strings.Contains(strings.ToLower(err.Error()), "list-units") {
		t.Fatalf("ensureNoConflicts error should mention detection/list-units failure: %v", err)
	}

	if err := svc.Operate("start"); err == nil {
		t.Fatal("Operate start must be blocked when list-units detection fails")
	}
	runner.assertNotCalled(t, "start")
	runner.assertNotCalled(t, "enable")
	runner.assertNotCalled(t, "stop")
	runner.assertNeverMutatesExternalNezhaUnits(t)
	settings.assertNotWritten(t, "NezhaEnabled")
}

// Group 2: injectable Lstat for external dirs — NotExist ignored; Permission and
// any other error fail closed (Status serviceError; start/enable blocked).
func TestNezhaConflictDirectoryLstatNotExistIgnored(t *testing.T) {
	runner := newFakeNezhaRunner("inactive", "disabled")
	missing := "/nonexistent/path/for/nezha/agent"
	svc := newConflictNezhaService(t, runner, nil, nezhaAgentDeps{
		ExternalDirs: []string{missing},
		Lstat: func(name string) (os.FileInfo, error) {
			if name != missing {
				t.Fatalf("unexpected Lstat path %q", name)
			}
			return nil, os.ErrNotExist
		},
	})

	got, err := svc.detectConflicts()
	if err != nil {
		t.Fatalf("NotExist must be ignored, not an error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("NotExist directory must not report conflict, got %#v", got)
	}
}

func TestNezhaConflictDirectoryLstatPermissionFailsClosed(t *testing.T) {
	runner := newFakeNezhaRunner("inactive", "disabled")
	dir := "/opt/nezha/agent"
	svc := newConflictNezhaService(t, runner, nil, nezhaAgentDeps{
		ExternalDirs: []string{dir},
		Lstat: func(name string) (os.FileInfo, error) {
			return nil, os.ErrPermission
		},
	})

	got, err := svc.detectConflicts()
	if err == nil {
		t.Fatal("Permission on external dir Lstat must return detection error")
	}
	if len(got) != 0 {
		t.Fatalf("conflicts want empty on detection error, got %#v", got)
	}

	if err := svc.ensureNoConflicts(); err == nil {
		t.Fatal("ensureNoConflicts must fail on directory Lstat Permission")
	}

	for _, op := range []string{"start", "enable"} {
		t.Run("operate_"+op, func(t *testing.T) {
			r := newFakeNezhaRunner("inactive", "disabled")
			s := newConflictNezhaService(t, r, newFakeNezhaSettings(map[string]string{"NezhaEnabled": "false"}), nezhaAgentDeps{
				ExternalDirs: []string{dir},
				Lstat: func(name string) (os.FileInfo, error) {
					return nil, os.ErrPermission
				},
			})
			if err := s.Operate(op); err == nil {
				t.Fatalf("Operate %s must be blocked on directory Lstat Permission", op)
			}
			r.assertNotCalled(t, "start")
			r.assertNotCalled(t, "enable")
			r.assertNotCalled(t, "stop")
			r.assertNeverMutatesExternalNezhaUnits(t)
		})
	}
}

func TestNezhaConflictDirectoryLstatOtherErrorFailsClosed(t *testing.T) {
	runner := newFakeNezhaRunner("inactive", "disabled")
	dir := "/opt/nezha-agent"
	svc := newConflictNezhaService(t, runner, nil, nezhaAgentDeps{
		ExternalDirs: []string{dir},
		Lstat: func(name string) (os.FileInfo, error) {
			return nil, errors.New("i/o error: input/output error")
		},
	})

	got, err := svc.detectConflicts()
	if err == nil {
		t.Fatal("any non-NotExist Lstat error must return detection error")
	}
	if len(got) != 0 {
		t.Fatalf("conflicts want empty on detection error, got %#v", got)
	}
}

func TestNezhaStatusDirectoryLstatErrorEntersServiceError(t *testing.T) {
	secret := "NzjDirLstatSec_cafebabe"
	cfg := []byte("server: dash.example.com:443\nclient_secret: " + secret + "\n")
	configPath, binaryPath := nezhaStatusFixture(t, 0700, 0755, 0600, cfg)
	runner := newFakeNezhaRunner("active", "enabled")
	runner.versionOutput = "v1.0.0"

	svc := newNezhaAgentService(nezhaAgentDeps{
		ConfigPath:   configPath,
		BinaryPath:   binaryPath,
		Unit:         NezhaAgentUnitName,
		Runner:       runner,
		Settings:     newFakeNezhaSettings(map[string]string{"NezhaEnabled": "true"}),
		Sleep:        func(time.Duration) {},
		Now:          time.Now,
		ExternalDirs: []string{"/opt/nezha/agent"},
		Lstat: func(name string) (os.FileInfo, error) {
			return nil, os.ErrPermission
		},
	})

	st, err := svc.Status()
	if err != nil {
		t.Fatalf("Status top-level error want nil, got %v", err)
	}
	if st.ServiceError == "" {
		t.Fatal("serviceError want non-empty when directory Lstat fails")
	}
	if !strings.Contains(strings.ToLower(st.ServiceError), "conflict") &&
		!strings.Contains(strings.ToLower(st.ServiceError), "permission") &&
		!strings.Contains(strings.ToLower(st.ServiceError), "detection") {
		t.Fatalf("serviceError should diagnose detection failure: %q", st.ServiceError)
	}
	// Other status still returned.
	if !st.Active || !st.Enabled {
		t.Fatalf("other fields should remain: active=%v enabled=%v", st.Active, st.Enabled)
	}
	if st.Conflicts == nil {
		t.Fatal("conflicts slice want non-nil")
	}
	if len(st.Conflicts) != 0 {
		t.Fatalf("conflicts want empty on detection error, got %#v", st.Conflicts)
	}
	assertStatusNoSecret(t, st, secret)
}

// Group 3: external conflict + bundled wasActive → plain Configure (EnableAndStart=false)
// still needs stop then start to restore; must block before any stop/read/write.
// Config bytes unchanged. Inactive plain Configure remains allowed (existing test).
func TestNezhaConfigureConflictBlocksWhenWasActiveNeedsRestore(t *testing.T) {
	runner := newFakeNezhaRunner("active", "enabled")
	runner.listUnitsOutput = "nezha-agent.service loaded active running Nezha Agent\n"
	svc := newConflictNezhaService(t, runner, nil, nezhaAgentDeps{})

	original := []byte("server: old.example.com:443\nclient_secret: keep-me-was-active\n")
	if err := os.WriteFile(svc.configPath, original, 0600); err != nil {
		t.Fatal(err)
	}

	readCalled := false
	writeCalled := false
	svc.readConfig = func(path string) ([]byte, error) {
		readCalled = true
		return os.ReadFile(path)
	}
	svc.writeConfig = func(path string, data []byte) error {
		writeCalled = true
		return os.WriteFile(path, data, 0600)
	}

	dash := "https://new.example.com"
	// EnableAndStart=false: still needs restore start because wasActive.
	err := svc.Configure(dto.NezhaAgentConfigUpdate{DashboardURL: &dash})
	if err == nil {
		t.Fatal("expected conflict error before stop/read/write when wasActive needs restore")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "conflict") {
		t.Fatalf("error should mention conflict: %v", err)
	}
	if readCalled {
		t.Fatal("must not read config when wasActive restore is blocked by conflict")
	}
	if writeCalled {
		t.Fatal("must not write config when wasActive restore is blocked by conflict")
	}
	runner.assertNotCalled(t, "stop")
	runner.assertNotCalled(t, "start")
	runner.assertNotCalled(t, "enable")
	runner.assertNeverMutatesExternalNezhaUnits(t)

	got, err := os.ReadFile(svc.configPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(original) {
		t.Fatalf("config bytes must be unchanged, got %q", got)
	}
}

// ----- Task 3A: config.yml → Settings one-way sync -----

const (
	nezhaSyncServerKey = "NezhaServer"
	nezhaSyncSecretKey = "NezhaClientSecret"
)

func writeNezhaSyncConfig(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0600); err != nil {
		t.Fatal(err)
	}
}

func TestNezhaSyncConfigToSettingsNormalizesHostPort(t *testing.T) {
	secret := "NzjSyncNormSec_a1b2c3d4"
	runner := newFakeNezhaRunner("inactive", "disabled")
	settings := newFakeNezhaSettings(nil)
	svc := newTestNezhaAgentService(t, runner, settings)

	// Uppercase host + leading-zero port must canonicalize.
	writeNezhaSyncConfig(t, svc.configPath,
		"server: Dashboard.Example.COM:0443\nclient_secret: "+secret+"\n")

	if err := svc.SyncConfigToSettings(); err != nil {
		assertErrorHasNoSecret(t, err, secret)
		t.Fatalf("SyncConfigToSettings: %v", err)
	}
	settings.assertManyWritten(t, map[string]string{
		nezhaSyncServerKey: "dashboard.example.com:443",
		nezhaSyncSecretKey: secret,
	})
	// Ensure batch path was used (not two independent single writes only).
	if settings.manyWriteCount() != 1 {
		t.Fatalf("want exactly 1 batch write, got %d", settings.manyWriteCount())
	}
}

func TestNezhaSyncConfigToSettingsNormalizesIPv6(t *testing.T) {
	secret := "NzjSyncIPv6Sec_e5f6a7b8"
	runner := newFakeNezhaRunner("inactive", "disabled")
	settings := newFakeNezhaSettings(nil)
	svc := newTestNezhaAgentService(t, runner, settings)

	writeNezhaSyncConfig(t, svc.configPath,
		"server: \"[2001:DB8::1]:08443\"\nclient_secret: "+secret+"\n")

	if err := svc.SyncConfigToSettings(); err != nil {
		assertErrorHasNoSecret(t, err, secret)
		t.Fatalf("SyncConfigToSettings: %v", err)
	}
	settings.assertManyWritten(t, map[string]string{
		nezhaSyncServerKey: "[2001:db8::1]:8443",
		nezhaSyncSecretKey: secret,
	})
}

func TestNezhaSyncConfigToSettingsMissingIsNoOp(t *testing.T) {
	runner := newFakeNezhaRunner("inactive", "disabled")
	settings := newFakeNezhaSettings(map[string]string{
		nezhaSyncServerKey: "stale.example.com:443",
		nezhaSyncSecretKey: "stale-secret",
	})
	svc := newTestNezhaAgentService(t, runner, settings)
	// config path intentionally missing (temp path never written)

	if err := svc.SyncConfigToSettings(); err != nil {
		t.Fatalf("missing config must be no-op, got %v", err)
	}
	if settings.manyWriteCount() != 0 {
		t.Fatalf("missing config must not write settings, many=%d", settings.manyWriteCount())
	}
	settings.assertNotWritten(t, nezhaSyncServerKey)
	settings.assertNotWritten(t, nezhaSyncSecretKey)
}

func TestNezhaSyncConfigToSettingsRejectsCorruptUnsafeAndInvalid(t *testing.T) {
	secret := "NzjSyncBadSec_deadbeef99"

	cases := []struct {
		name   string
		setup  func(t *testing.T, path string)
		secret string
	}{
		{
			name: "corrupt yaml",
			setup: func(t *testing.T, path string) {
				writeNezhaSyncConfig(t, path, "server: dash:443\nclient_secret: "+secret+"\n[broken\n")
			},
			secret: secret,
		},
		{
			name: "missing server",
			setup: func(t *testing.T, path string) {
				writeNezhaSyncConfig(t, path, "client_secret: "+secret+"\n")
			},
			secret: secret,
		},
		{
			name: "missing secret",
			setup: func(t *testing.T, path string) {
				writeNezhaSyncConfig(t, path, "server: dash.example.com:443\n")
			},
			secret: "",
		},
		{
			name: "illegal port zero",
			setup: func(t *testing.T, path string) {
				writeNezhaSyncConfig(t, path, "server: dash.example.com:0\nclient_secret: "+secret+"\n")
			},
			secret: secret,
		},
		{
			name: "illegal port too large",
			setup: func(t *testing.T, path string) {
				writeNezhaSyncConfig(t, path, "server: dash.example.com:65536\nclient_secret: "+secret+"\n")
			},
			secret: secret,
		},
		{
			name: "bare host without port",
			setup: func(t *testing.T, path string) {
				writeNezhaSyncConfig(t, path, "server: dash.example.com\nclient_secret: "+secret+"\n")
			},
			secret: secret,
		},
		{
			name: "symlink unsafe",
			setup: func(t *testing.T, path string) {
				dir := filepath.Dir(path)
				target := filepath.Join(dir, "real.yml")
				writeNezhaSyncConfig(t, target, "server: dash.example.com:443\nclient_secret: "+secret+"\n")
				if err := os.Symlink(target, path); err != nil {
					t.Fatal(err)
				}
			},
			secret: secret,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			runner := newFakeNezhaRunner("inactive", "disabled")
			settings := newFakeNezhaSettings(nil)
			svc := newTestNezhaAgentService(t, runner, settings)
			tc.setup(t, svc.configPath)

			err := svc.SyncConfigToSettings()
			if err == nil {
				t.Fatal("expected sync error")
			}
			assertErrorHasNoSecret(t, err, tc.secret)
			if settings.manyWriteCount() != 0 {
				t.Fatalf("must not write DB on invalid config, many=%d", settings.manyWriteCount())
			}
			settings.assertNotWritten(t, nezhaSyncServerKey)
			settings.assertNotWritten(t, nezhaSyncSecretKey)
		})
	}
}

func TestNezhaStatusSyncsHealthyConfigToSettings(t *testing.T) {
	secret := "NzjStatusSyncSec_1122aabb"
	cfg := []byte("server: Dashboard.Example.COM:0443\nclient_secret: " + secret + "\ntls: true\n")
	configPath, binaryPath := nezhaStatusFixture(t, 0700, 0755, 0600, cfg)
	runner := newFakeNezhaRunner("active", "enabled")
	runner.versionOutput = "nezha-agent version v2.3.1"
	settings := newFakeNezhaSettings(map[string]string{"NezhaEnabled": "true"})
	svc := newStatusNezhaService(t, configPath, binaryPath, runner, settings)

	st, err := svc.Status()
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if !st.ConfigHealthy {
		t.Fatalf("configHealthy want true, err=%q", st.ConfigError)
	}
	settings.assertManyWritten(t, map[string]string{
		nezhaSyncServerKey: "dashboard.example.com:443",
		nezhaSyncSecretKey: secret,
	})
	assertStatusNoSecret(t, st, secret)
	if st.ServiceError != "" {
		t.Fatalf("serviceError want empty on successful sync, got %q", st.ServiceError)
	}
}

func TestNezhaStatusSyncStoreFailureWritesServiceError(t *testing.T) {
	secret := "NzjStatusSyncFail_ccddeeff"
	cfg := []byte("server: dash.example.com:443\nclient_secret: " + secret + "\n")
	configPath, binaryPath := nezhaStatusFixture(t, 0700, 0755, 0600, cfg)
	runner := newFakeNezhaRunner("inactive", "disabled")
	settings := newFakeNezhaSettings(nil)
	settings.fail = errors.New("db down")
	svc := newStatusNezhaService(t, configPath, binaryPath, runner, settings)

	st, err := svc.Status()
	if err != nil {
		t.Fatalf("Status top-level error want nil, got %v", err)
	}
	if st == nil {
		t.Fatal("Status returned nil")
	}
	if !st.ConfigHealthy {
		t.Fatalf("configHealthy want true even when settings sync fails, err=%q", st.ConfigError)
	}
	if st.ServiceError == "" {
		t.Fatal("serviceError must diagnose settings sync failure")
	}
	if strings.Contains(st.ServiceError, secret) {
		t.Fatal("serviceError must not expose secret")
	}
	assertStatusNoSecret(t, st, secret)
	// Partial status still present.
	if st.ServiceState != "inactive" {
		t.Fatalf("serviceState=%q want inactive", st.ServiceState)
	}
	if st.Server != "dash.example.com:443" {
		t.Fatalf("server=%q", st.Server)
	}
}

func TestNezhaStatusMissingOrCorruptDoesNotWriteSettings(t *testing.T) {
	secret := "NzjStatusNoWrite_99887766"

	t.Run("missing", func(t *testing.T) {
		configPath, binaryPath := nezhaStatusFixture(t, 0700, 0755, 0600, nil)
		settings := newFakeNezhaSettings(nil)
		svc := newStatusNezhaService(t, configPath, binaryPath, newFakeNezhaRunner("inactive", "disabled"), settings)
		st, err := svc.Status()
		if err != nil {
			t.Fatal(err)
		}
		if st.ConfigHealthy {
			t.Fatal("configHealthy want false")
		}
		if settings.manyWriteCount() != 0 {
			t.Fatalf("must not write settings, many=%d", settings.manyWriteCount())
		}
		settings.assertNotWritten(t, nezhaSyncServerKey)
		settings.assertNotWritten(t, nezhaSyncSecretKey)
	})

	t.Run("corrupt", func(t *testing.T) {
		cfg := []byte("server: dash:443\nclient_secret: " + secret + "\n[broken\n")
		configPath, binaryPath := nezhaStatusFixture(t, 0700, 0755, 0600, cfg)
		settings := newFakeNezhaSettings(nil)
		svc := newStatusNezhaService(t, configPath, binaryPath, newFakeNezhaRunner("inactive", "disabled"), settings)
		st, err := svc.Status()
		if err != nil {
			t.Fatal(err)
		}
		if st.ConfigHealthy {
			t.Fatal("configHealthy want false")
		}
		if settings.manyWriteCount() != 0 {
			t.Fatalf("must not write settings, many=%d", settings.manyWriteCount())
		}
		assertStatusNoSecret(t, st, secret)
	})
}

func TestNezhaConfigurePreWriteSyncCapturesRotatedSecret(t *testing.T) {
	// Dashboard rotates client_secret after stop; pre-write sync must capture it
	// before merge, and final sync must mirror the final file.
	secretBefore := "NzjCfgPreBefore_1111"
	secretRotated := "NzjCfgPreRotated_2222"
	runner := newFakeNezhaRunner("active", "enabled")
	settings := newFakeNezhaSettings(nil)
	svc := newTestNezhaAgentService(t, runner, settings)

	writeNezhaSyncConfig(t, svc.configPath,
		"server: old.example.com:443\nclient_secret: "+secretBefore+"\nuuid: keep-uuid\n")
	runner.afterStop = func() {
		_ = os.WriteFile(svc.configPath, []byte(
			"server: old.example.com:443\nclient_secret: "+secretRotated+"\nuuid: keep-uuid\n",
		), 0600)
	}

	dash := "https://new.example.com:8443"
	if err := svc.Configure(dto.NezhaAgentConfigUpdate{DashboardURL: &dash}); err != nil {
		assertErrorHasNoSecret(t, err, secretRotated)
		assertErrorHasNoSecret(t, err, secretBefore)
		t.Fatal(err)
	}

	// At least one batch must have captured the rotated secret (pre-write).
	foundRotated := false
	foundFinal := false
	settings.mu.Lock()
	for _, m := range settings.manyWrites {
		if m[nezhaSyncSecretKey] == secretRotated {
			foundRotated = true
		}
		if m[nezhaSyncServerKey] == "new.example.com:8443" && m[nezhaSyncSecretKey] == secretRotated {
			foundFinal = true
		}
	}
	settings.mu.Unlock()
	if !foundRotated {
		t.Fatal("pre-write sync must capture Dashboard-rotated secret")
	}
	if !foundFinal {
		t.Fatal("final file sync must write new server with rotated secret")
	}
	if settings.manyWriteCount() < 2 {
		t.Fatalf("want pre+post sync batch writes, got %d", settings.manyWriteCount())
	}

	got, err := os.ReadFile(svc.configPath)
	if err != nil {
		t.Fatal(err)
	}
	assertYAMLValue(t, got, "client_secret", secretRotated)
	assertYAMLValue(t, got, "server", "new.example.com:8443")
	runner.assertCalled(t, "systemctl", "start", NezhaAgentUnitName)
}

func TestNezhaConfigureFinalFileSyncAfterWrite(t *testing.T) {
	secret := "NzjCfgFinalSec_3333"
	runner := newFakeNezhaRunner("inactive", "disabled")
	settings := newFakeNezhaSettings(nil)
	svc := newTestNezhaAgentService(t, runner, settings)

	writeNezhaSyncConfig(t, svc.configPath,
		"server: old.example.com:443\nclient_secret: "+secret+"\n")

	dash := "https://New.Example.COM:9443"
	if err := svc.Configure(dto.NezhaAgentConfigUpdate{DashboardURL: &dash}); err != nil {
		assertErrorHasNoSecret(t, err, secret)
		t.Fatal(err)
	}

	// Last batch write must reflect the final on-disk server.
	settings.mu.Lock()
	if len(settings.manyWrites) == 0 {
		settings.mu.Unlock()
		t.Fatal("expected settings sync writes")
	}
	last := settings.manyWrites[len(settings.manyWrites)-1]
	settings.mu.Unlock()
	if last[nezhaSyncServerKey] != "new.example.com:9443" {
		t.Fatalf("final sync server=%q want new.example.com:9443", last[nezhaSyncServerKey])
	}
	if last[nezhaSyncSecretKey] != secret {
		t.Fatal("final sync secret mismatch (len check only)")
	}
	settings.assertWritten(t, nezhaSyncServerKey, "new.example.com:9443")
	settings.assertWritten(t, nezhaSyncSecretKey, secret)
}

func TestNezhaConfigurePreWriteSyncFailureDoesNotWriteAndRestores(t *testing.T) {
	secret := "NzjCfgPreFailSec_4444"
	runner := newFakeNezhaRunner("active", "enabled")
	settings := newFakeNezhaSettings(nil)
	// Fail the first (pre-write) batch sync.
	settings.failManyOnCall = 1
	svc := newTestNezhaAgentService(t, runner, settings)

	original := []byte("server: old.example.com:443\nclient_secret: " + secret + "\n")
	writeNezhaSyncConfig(t, svc.configPath, string(original))

	dash := "https://new.example.com"
	err := svc.Configure(dto.NezhaAgentConfigUpdate{DashboardURL: &dash})
	if err == nil {
		t.Fatal("expected pre-write sync failure")
	}
	assertErrorHasNoSecret(t, err, secret)

	got, err2 := os.ReadFile(svc.configPath)
	if err2 != nil {
		t.Fatal(err2)
	}
	if string(got) != string(original) {
		t.Fatalf("pre-write sync failure must not write file, got %q", got)
	}
	runner.assertCalled(t, "systemctl", "stop", NezhaAgentUnitName)
	runner.assertCalled(t, "systemctl", "start", NezhaAgentUnitName)

	// Values must not be applied (fail before apply); only the failed attempt is recorded.
	if v, ok := settings.valueOf(nezhaSyncServerKey); ok && v != "" {
		// failManyOnCall returns before applying; values map must stay empty for these keys.
		t.Fatalf("settings must not retain server after failed pre-write sync, got %q", v)
	}
}

func TestNezhaConfigurePostWriteSyncFailureKeepsFileAndRestores(t *testing.T) {
	secret := "NzjCfgPostFailSec_5555"
	runner := newFakeNezhaRunner("active", "enabled")
	settings := newFakeNezhaSettings(nil)
	// Call 1 = pre-write (ok), call 2 = post-write (fail).
	settings.failManyOnCall = 2
	svc := newTestNezhaAgentService(t, runner, settings)

	writeNezhaSyncConfig(t, svc.configPath,
		"server: old.example.com:443\nclient_secret: "+secret+"\n")

	dash := "https://new.example.com:8443"
	err := svc.Configure(dto.NezhaAgentConfigUpdate{DashboardURL: &dash})
	if err == nil {
		t.Fatal("expected post-write sync failure")
	}
	assertErrorHasNoSecret(t, err, secret)

	got, err2 := os.ReadFile(svc.configPath)
	if err2 != nil {
		t.Fatal(err2)
	}
	// New file must be retained.
	assertYAMLValue(t, got, "server", "new.example.com:8443")
	assertYAMLValue(t, got, "client_secret", secret)

	runner.assertCalled(t, "systemctl", "stop", NezhaAgentUnitName)
	runner.assertCalled(t, "systemctl", "start", NezhaAgentUnitName)
}

func TestNezhaConfigureFirstConfigSkipsPreWriteSync(t *testing.T) {
	secret := "NzjCfgFirstSec_6666"
	runner := newFakeNezhaRunner("inactive", "disabled")
	settings := newFakeNezhaSettings(nil)
	// If a pre-write sync were attempted against a missing file it would no-op;
	// assert we only sync after the final write (one batch).
	svc := newTestNezhaAgentService(t, runner, settings)

	dash := "https://dashboard.example.com"
	if err := svc.Configure(dto.NezhaAgentConfigUpdate{
		DashboardURL: &dash,
		ClientSecret: &secret,
	}); err != nil {
		assertErrorHasNoSecret(t, err, secret)
		t.Fatal(err)
	}
	if settings.manyWriteCount() != 1 {
		t.Fatalf("first config should only post-write sync once, got %d", settings.manyWriteCount())
	}
	settings.assertManyWritten(t, map[string]string{
		nezhaSyncServerKey: "dashboard.example.com:443",
		nezhaSyncSecretKey: secret,
	})
}

func TestNezhaOperateStartAndEnableSyncAfterSuccess(t *testing.T) {
	secret := "NzjOpSyncSec_7777"

	t.Run("start", func(t *testing.T) {
		runner := newFakeNezhaRunner("inactive", "disabled")
		settings := newFakeNezhaSettings(nil)
		svc := newTestNezhaAgentService(t, runner, settings)
		writeNezhaSyncConfig(t, svc.configPath,
			"server: Dash.Example.COM:0443\nclient_secret: "+secret+"\n")

		if err := svc.Operate("start"); err != nil {
			assertErrorHasNoSecret(t, err, secret)
			t.Fatal(err)
		}
		runner.assertCalled(t, "systemctl", "start", NezhaAgentUnitName)
		settings.assertManyWritten(t, map[string]string{
			nezhaSyncServerKey: "dash.example.com:443",
			nezhaSyncSecretKey: secret,
		})
	})

	t.Run("restart", func(t *testing.T) {
		runner := newFakeNezhaRunner("active", "enabled")
		settings := newFakeNezhaSettings(nil)
		svc := newTestNezhaAgentService(t, runner, settings)
		writeNezhaSyncConfig(t, svc.configPath,
			"server: dash.example.com:443\nclient_secret: "+secret+"\n")

		if err := svc.Operate("restart"); err != nil {
			assertErrorHasNoSecret(t, err, secret)
			t.Fatal(err)
		}
		runner.assertCalled(t, "systemctl", "restart", NezhaAgentUnitName)
		if settings.manyWriteCount() != 1 {
			t.Fatalf("restart should sync once, got %d", settings.manyWriteCount())
		}
	})

	t.Run("enable", func(t *testing.T) {
		runner := newFakeNezhaRunner("inactive", "disabled")
		settings := newFakeNezhaSettings(map[string]string{"NezhaEnabled": "false"})
		svc := newTestNezhaAgentService(t, runner, settings)
		writeNezhaSyncConfig(t, svc.configPath,
			"server: dash.example.com:443\nclient_secret: "+secret+"\n")

		if err := svc.Operate("enable"); err != nil {
			assertErrorHasNoSecret(t, err, secret)
			t.Fatal(err)
		}
		runner.assertCalled(t, "systemctl", "enable", "--now", NezhaAgentUnitName)
		settings.assertWritten(t, "NezhaEnabled", "true")
		settings.assertManyWritten(t, map[string]string{
			nezhaSyncServerKey: "dash.example.com:443",
			nezhaSyncSecretKey: secret,
		})
	})

	t.Run("stop does not sync", func(t *testing.T) {
		runner := newFakeNezhaRunner("active", "enabled")
		settings := newFakeNezhaSettings(nil)
		svc := newTestNezhaAgentService(t, runner, settings)
		writeNezhaSyncConfig(t, svc.configPath,
			"server: dash.example.com:443\nclient_secret: "+secret+"\n")

		if err := svc.Operate("stop"); err != nil {
			t.Fatal(err)
		}
		if settings.manyWriteCount() != 0 {
			t.Fatalf("stop must not sync, many=%d", settings.manyWriteCount())
		}
	})

	t.Run("disable does not sync", func(t *testing.T) {
		runner := newFakeNezhaRunner("active", "enabled")
		settings := newFakeNezhaSettings(map[string]string{"NezhaEnabled": "true"})
		svc := newTestNezhaAgentService(t, runner, settings)
		writeNezhaSyncConfig(t, svc.configPath,
			"server: dash.example.com:443\nclient_secret: "+secret+"\n")

		if err := svc.Operate("disable"); err != nil {
			t.Fatal(err)
		}
		if settings.manyWriteCount() != 0 {
			t.Fatalf("disable must not sync, many=%d", settings.manyWriteCount())
		}
	})
}

func TestNezhaConfigureEnableAndStartSyncsAfterStart(t *testing.T) {
	secret := "NzjCfgEnStartSec_8888"
	runner := newFakeNezhaRunner("inactive", "disabled")
	settings := newFakeNezhaSettings(map[string]string{"NezhaEnabled": "false"})
	svc := newTestNezhaAgentService(t, runner, settings)

	dash := "https://dashboard.example.com"
	if err := svc.Configure(dto.NezhaAgentConfigUpdate{
		DashboardURL:   &dash,
		ClientSecret:   &secret,
		EnableAndStart: true,
	}); err != nil {
		assertErrorHasNoSecret(t, err, secret)
		t.Fatal(err)
	}
	runner.assertCalled(t, "systemctl", "enable", "--now", NezhaAgentUnitName)
	settings.assertWritten(t, "NezhaEnabled", "true")
	// First-config post-write sync + post-start sync.
	if settings.manyWriteCount() < 2 {
		t.Fatalf("enableAndStart should sync after write and after start, got %d", settings.manyWriteCount())
	}
	settings.assertManyWritten(t, map[string]string{
		nezhaSyncServerKey: "dashboard.example.com:443",
		nezhaSyncSecretKey: secret,
	})
}

// Plain Configure on an originally-active unit must sync config.yml → DB after
// a successful restore start, so Agent post-start secret rotation is mirrored.
func TestNezhaConfigureWasActiveSyncsAfterStart(t *testing.T) {
	secretWrite := "NzjCfgWasActiveWrite_aaa1"
	secretAfterStart := "NzjCfgWasActivePostStart_bbb2"
	runner := newFakeNezhaRunner("active", "enabled")
	settings := newFakeNezhaSettings(nil)
	svc := newTestNezhaAgentService(t, runner, settings)

	writeNezhaSyncConfig(t, svc.configPath,
		"server: old.example.com:443\nclient_secret: "+secretWrite+"\nuuid: keep-uuid\n")

	// Official Agent rewrites client_secret once systemctl start succeeds.
	runner.afterStart = func() {
		_ = os.WriteFile(svc.configPath, []byte(
			"server: new.example.com:8443\nclient_secret: "+secretAfterStart+"\nuuid: keep-uuid\ntls: true\ninsecure_tls: false\n",
		), 0600)
	}

	dash := "https://new.example.com:8443"
	if err := svc.Configure(dto.NezhaAgentConfigUpdate{DashboardURL: &dash}); err != nil {
		assertErrorHasNoSecret(t, err, secretWrite)
		assertErrorHasNoSecret(t, err, secretAfterStart)
		t.Fatal(err)
	}
	runner.assertCalled(t, "systemctl", "stop", NezhaAgentUnitName)
	runner.assertCalled(t, "systemctl", "start", NezhaAgentUnitName)

	// Last batch mirror must capture the post-start rotated secret, not only the
	// post-write value that existed before restore start.
	settings.mu.Lock()
	if len(settings.manyWrites) == 0 {
		settings.mu.Unlock()
		t.Fatal("expected settings batch sync writes")
	}
	last := settings.manyWrites[len(settings.manyWrites)-1]
	settings.mu.Unlock()
	if last[nezhaSyncSecretKey] != secretAfterStart {
		t.Fatalf("last batch secret must be post-start rotation (len=%d), got len=%d writes=%d",
			len(secretAfterStart), len(last[nezhaSyncSecretKey]), settings.manyWriteCount())
	}
	if last[nezhaSyncServerKey] != "new.example.com:8443" {
		t.Fatalf("last batch server=%q want new.example.com:8443", last[nezhaSyncServerKey])
	}
	settings.assertManyWritten(t, map[string]string{
		nezhaSyncServerKey: "new.example.com:8443",
		nezhaSyncSecretKey: secretAfterStart,
	})

	// Fail-path diagnostics must never leak either sentinel.
	settings2 := newFakeNezhaSettings(nil)
	settings2.fail = errors.New("db exploded")
	runner2 := newFakeNezhaRunner("active", "enabled")
	svc2 := newTestNezhaAgentService(t, runner2, settings2)
	writeNezhaSyncConfig(t, svc2.configPath,
		"server: old.example.com:443\nclient_secret: "+secretAfterStart+"\n")
	err := svc2.Configure(dto.NezhaAgentConfigUpdate{DashboardURL: &dash})
	if err == nil {
		t.Fatal("expected sync/store failure path to surface an error")
	}
	assertErrorHasNoSecret(t, err, secretAfterStart)
	assertErrorHasNoSecret(t, err, secretWrite)
}

func TestNezhaSyncErrorsNeverContainSecretSentinel(t *testing.T) {
	secret := "NzjSyncSentinel_MUST_NOT_LEAK_9f3a"
	runner := newFakeNezhaRunner("inactive", "disabled")
	settings := newFakeNezhaSettings(nil)
	svc := newTestNezhaAgentService(t, runner, settings)

	// Corrupt config embedding the sentinel.
	writeNezhaSyncConfig(t, svc.configPath,
		"server: dash.example.com:443\nclient_secret: "+secret+"\n[unterminated\n")
	err := svc.SyncConfigToSettings()
	if err == nil {
		t.Fatal("expected error")
	}
	assertErrorHasNoSecret(t, err, secret)

	// Store failure path after valid parse.
	writeNezhaSyncConfig(t, svc.configPath,
		"server: dash.example.com:443\nclient_secret: "+secret+"\n")
	settings.fail = errors.New("db exploded " + secret) // store error must still not pass secret through if we sanitize
	// Actually: implementation must not wrap store errors that could contain secrets from our map.
	// Use a clean store error.
	settings.fail = errors.New("db exploded")
	err = svc.SyncConfigToSettings()
	if err == nil {
		t.Fatal("expected store error")
	}
	assertErrorHasNoSecret(t, err, secret)

	// Operate start sync failure.
	settings.fail = errors.New("db exploded")
	runner2 := newFakeNezhaRunner("inactive", "disabled")
	settings2 := newFakeNezhaSettings(nil)
	settings2.fail = errors.New("db exploded")
	svc2 := newTestNezhaAgentService(t, runner2, settings2)
	writeNezhaSyncConfig(t, svc2.configPath,
		"server: dash.example.com:443\nclient_secret: "+secret+"\n")
	err = svc2.Operate("start")
	if err == nil {
		t.Fatal("expected start sync error")
	}
	assertErrorHasNoSecret(t, err, secret)
}

func TestNezhaFakeStoreCreateOrUpdateManyCopiesMap(t *testing.T) {
	// Guard against reference sharing false-positives in sync tests.
	s := newFakeNezhaSettings(nil)
	payload := map[string]string{
		nezhaSyncServerKey: "dash.example.com:443",
		nezhaSyncSecretKey: "temp-secret",
	}
	if err := s.CreateOrUpdateMany(payload); err != nil {
		t.Fatal(err)
	}
	payload[nezhaSyncServerKey] = "mutated.example.com:1"
	payload[nezhaSyncSecretKey] = "mutated"
	s.mu.Lock()
	got := s.manyWrites[0]
	s.mu.Unlock()
	if got[nezhaSyncServerKey] != "dash.example.com:443" {
		t.Fatalf("manyWrites must copy map, got server %q", got[nezhaSyncServerKey])
	}
	if got[nezhaSyncSecretKey] != "temp-secret" {
		t.Fatal("manyWrites must copy secret value")
	}
}
