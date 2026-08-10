package service

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

// ---------------------------------------------------------------------------
// Task 6 RED: component-package upgrade contract
//
// These tests define the public/package-local API expected from
// component_upgrade.go. Production is wired through dependency injection
// (paths + systemd runner + optional replace hook) so tests never need
// root, real systemd, network, or /opt.
//
// Expected production symbols (not implemented in this RED step):
//   - extractComponentArchive(archivePath, destDir string) (xpanelPath, agentPath string, err error)
//   - validateComponentELF(path string) error
//   - componentUpgradeDeps { XPanelPath, AgentPath, ConfigPath, AgentUnit, Runner, RestartXPanel, ReplaceBinary }
//   - applyComponentPackage(deps componentUpgradeDeps, archivePath string) error
// ---------------------------------------------------------------------------

const (
	componentArchiveXPanelName = "xpanel"
	componentArchiveAgentName  = "nezha-agent/nezha-agent"

	// Distinct markers so live vs staged content is obvious in assertions.
	liveXPanelMarker  = "LIVE-XPANEL-BINARY-v1"
	liveAgentMarker   = "LIVE-AGENT-BINARY-v1"
	newXPanelMarker   = "NEW-XPANEL-BINARY-v2"
	newAgentMarker    = "NEW-AGENT-BINARY-v2"
	configFixtureBody = "client_secret: keep-me-byte-for-byte\nuuid: fixed-uuid-0001\ncustom_field: preserved\n"
)

// ----- ELF / archive fixtures (stdlib only; no go build, no network) -----

// minimalELF returns a parseable ELF64 header for the given Go arch.
// debug/elf only needs a valid FileHeader for architecture preflight.
func minimalELF(t *testing.T, goarch string) []byte {
	t.Helper()
	var machine uint16
	switch goarch {
	case "amd64":
		machine = 0x3e // EM_X86_64
	case "arm64":
		machine = 0xb7 // EM_AARCH64
	default:
		t.Fatalf("unsupported GOARCH %q for ELF fixture", goarch)
	}
	hdr := make([]byte, 64)
	copy(hdr[0:], []byte{0x7f, 'E', 'L', 'F'})
	hdr[4] = 2                                 // ELFCLASS64
	hdr[5] = 1                                 // ELFDATA2LSB
	hdr[6] = 1                                 // EV_CURRENT
	binary.LittleEndian.PutUint16(hdr[16:], 2) // ET_EXEC
	binary.LittleEndian.PutUint16(hdr[18:], machine)
	binary.LittleEndian.PutUint32(hdr[20:], 1)  // e_version
	binary.LittleEndian.PutUint16(hdr[52:], 64) // e_ehsize
	// Append a marker payload so live/new content remains distinguishable.
	return hdr
}

func elfWithMarker(t *testing.T, goarch, marker string) []byte {
	t.Helper()
	base := minimalELF(t, goarch)
	return append(base, []byte(marker)...)
}

func wrongArchForRuntime(t *testing.T) string {
	t.Helper()
	switch runtime.GOARCH {
	case "amd64":
		return "arm64"
	case "arm64":
		return "amd64"
	default:
		t.Fatalf("test host GOARCH %q has no opposite fixture", runtime.GOARCH)
		return ""
	}
}

type tarEntry struct {
	Name     string
	Body     []byte
	Typeflag byte
	Linkname string
	Mode     int64
}

func writeGzipTar(t *testing.T, path string, entries []tarEntry) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)
	for _, e := range entries {
		mode := e.Mode
		if mode == 0 {
			mode = 0755
		}
		hdr := &tar.Header{
			Name:     e.Name,
			Mode:     mode,
			Typeflag: e.Typeflag,
			Linkname: e.Linkname,
		}
		if e.Typeflag == 0 || e.Typeflag == tar.TypeReg {
			hdr.Typeflag = tar.TypeReg
			hdr.Size = int64(len(e.Body))
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("write header %s: %v", e.Name, err)
		}
		if hdr.Typeflag == tar.TypeReg && len(e.Body) > 0 {
			if _, err := tw.Write(e.Body); err != nil {
				t.Fatalf("write body %s: %v", e.Name, err)
			}
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gw.Close(); err != nil {
		t.Fatal(err)
	}
}

func validComponentArchive(t *testing.T, dir string, xpanelBody, agentBody []byte) string {
	t.Helper()
	path := filepath.Join(dir, "component.tar.gz")
	writeGzipTar(t, path, []tarEntry{
		{Name: componentArchiveXPanelName, Body: xpanelBody},
		{Name: componentArchiveAgentName, Body: agentBody},
	})
	return path
}

// ----- fakes / harness -----

// componentReplaceCall records one staged→live binary replacement.
type componentReplaceCall struct {
	Src string
	Dst string
}

// fakeComponentRunner records systemctl calls without talking to a real host.
// Only the Agent unit is managed by the upgrade transaction.
type fakeComponentRunner struct {
	mu      sync.Mutex
	calls   [][]string
	active  string
	enabled string
	failOps map[string]error
	// After a successful stop, active becomes inactive.
	// After a successful start/restart, active becomes active.
}

func newFakeComponentRunner(active, enabled string) *fakeComponentRunner {
	return &fakeComponentRunner{
		active:  active,
		enabled: enabled,
		failOps: map[string]error{},
	}
}

func (f *fakeComponentRunner) CombinedOutput(name string, args ...string) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, append([]string{name}, args...))
	if name != "systemctl" {
		return nil, fmt.Errorf("unexpected command %q", name)
	}
	if len(args) == 0 {
		return nil, errors.New("systemctl: missing args")
	}
	verb := args[0]
	switch verb {
	case "is-active":
		if err := f.failOps[verb]; err != nil {
			return []byte(err.Error()), err
		}
		state := f.active
		if state == "active" {
			return []byte(state + "\n"), nil
		}
		return []byte(state + "\n"), fmt.Errorf("systemctl is-active: %s", state)
	case "is-enabled":
		if err := f.failOps[verb]; err != nil {
			return []byte(err.Error()), err
		}
		state := f.enabled
		if state == "enabled" || state == "enabled-runtime" {
			return []byte(state + "\n"), nil
		}
		return []byte(state + "\n"), fmt.Errorf("systemctl is-enabled: %s", state)
	case "stop":
		if err := f.failOps["stop"]; err != nil {
			return []byte(err.Error()), err
		}
		f.active = "inactive"
		return []byte{}, nil
	case "start", "restart":
		if err := f.failOps[verb]; err != nil {
			return []byte(err.Error()), err
		}
		f.active = "active"
		return []byte{}, nil
	case "enable", "disable":
		// Upgrade must never mutate the enabled dimension.
		if err := f.failOps[verb]; err != nil {
			return []byte(err.Error()), err
		}
		if verb == "enable" {
			f.enabled = "enabled"
		} else {
			f.enabled = "disabled"
		}
		return []byte{}, nil
	default:
		return nil, fmt.Errorf("unexpected systemctl verb %q", verb)
	}
}

func (f *fakeComponentRunner) systemctlVerbs() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	var verbs []string
	for _, c := range f.calls {
		if len(c) >= 2 && c[0] == "systemctl" {
			verbs = append(verbs, c[1])
		}
	}
	return verbs
}

func (f *fakeComponentRunner) assertNoEnableDisable(t *testing.T) {
	t.Helper()
	for _, v := range f.systemctlVerbs() {
		if v == "enable" || v == "disable" {
			t.Fatalf("upgrade must not change enabled state; verbs=%v", f.systemctlVerbs())
		}
	}
}

func (f *fakeComponentRunner) assertNoMutatingSystemctl(t *testing.T) {
	t.Helper()
	for _, v := range f.systemctlVerbs() {
		switch v {
		case "stop", "start", "restart", "enable", "disable":
			t.Fatalf("preflight failure must not mutate systemd; verbs=%v", f.systemctlVerbs())
		}
	}
}

// componentUpgradeHarness is a temp-dir live install layout for applyComponentPackage.
type componentUpgradeHarness struct {
	Root       string
	XPanelPath string
	AgentPath  string
	ConfigPath string
	AgentUnit  string
	Runner     *fakeComponentRunner
	// ReplaceCalls records production ReplaceBinary invocations when the
	// default FS path is wrapped.
	ReplaceCalls []componentReplaceCall
	// RestartCalls counts RestartXPanel invocations.
	RestartCalls int
	// Timeline merges systemd verbs and replace/restart events for order asserts.
	Timeline []string
	// failReplaceDst, when non-empty, makes the injected ReplaceBinary fail
	// for that live destination path.
	failReplaceDst string
	// failRestart makes RestartXPanel return an error.
	failRestart bool
}

func newComponentUpgradeHarness(t *testing.T, active, enabled string) *componentUpgradeHarness {
	t.Helper()
	root := t.TempDir()
	agentDir := filepath.Join(root, "nezha-agent")
	if err := os.MkdirAll(agentDir, 0o700); err != nil {
		t.Fatal(err)
	}
	h := &componentUpgradeHarness{
		Root:       root,
		XPanelPath: filepath.Join(root, "xpanel"),
		AgentPath:  filepath.Join(agentDir, "nezha-agent"),
		ConfigPath: filepath.Join(agentDir, "config.yml"),
		AgentUnit:  NezhaAgentUnitName,
		Runner:     newFakeComponentRunner(active, enabled),
	}
	if err := os.WriteFile(h.XPanelPath, elfWithMarker(t, runtime.GOARCH, liveXPanelMarker), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(h.AgentPath, elfWithMarker(t, runtime.GOARCH, liveAgentMarker), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(h.ConfigPath, []byte(configFixtureBody), 0o600); err != nil {
		t.Fatal(err)
	}
	return h
}

// recordingComponentRunner wraps fakeComponentRunner and appends verbs to the harness timeline.
type recordingComponentRunner struct {
	inner *fakeComponentRunner
	h     *componentUpgradeHarness
}

func (r *recordingComponentRunner) CombinedOutput(name string, args ...string) ([]byte, error) {
	if name == "systemctl" && len(args) > 0 {
		r.h.Timeline = append(r.h.Timeline, "systemctl:"+args[0])
	}
	return r.inner.CombinedOutput(name, args...)
}

func (h *componentUpgradeHarness) deps() componentUpgradeDeps {
	hRef := h
	return componentUpgradeDeps{
		XPanelPath: h.XPanelPath,
		AgentPath:  h.AgentPath,
		ConfigPath: h.ConfigPath,
		AgentUnit:  h.AgentUnit,
		Runner:     &recordingComponentRunner{inner: h.Runner, h: hRef},
		RestartXPanel: func() error {
			hRef.RestartCalls++
			hRef.Timeline = append(hRef.Timeline, "restart-xpanel")
			if hRef.failRestart {
				return errors.New("injected xpanel restart failure")
			}
			return nil
		},
		ReplaceBinary: func(src, dst string) error {
			hRef.ReplaceCalls = append(hRef.ReplaceCalls, componentReplaceCall{Src: src, Dst: dst})
			switch dst {
			case hRef.AgentPath:
				hRef.Timeline = append(hRef.Timeline, "replace:agent")
			case hRef.XPanelPath:
				hRef.Timeline = append(hRef.Timeline, "replace:xpanel")
			default:
				hRef.Timeline = append(hRef.Timeline, "replace:other")
			}
			if hRef.failReplaceDst != "" && dst == hRef.failReplaceDst {
				return fmt.Errorf("injected replace failure for %s", dst)
			}
			// Production contract: stage as dst+".new" then rename over dst.
			// Mirror that so failure-cleanup tests can observe .new files.
			staging := dst + ".new"
			data, err := os.ReadFile(src)
			if err != nil {
				return err
			}
			if err := os.WriteFile(staging, data, 0o755); err != nil {
				return err
			}
			if err := os.Rename(staging, dst); err != nil {
				_ = os.Remove(staging)
				return err
			}
			return nil
		},
	}
}

func (h *componentUpgradeHarness) newArchive(t *testing.T) string {
	t.Helper()
	return validComponentArchive(t, h.Root,
		elfWithMarker(t, runtime.GOARCH, newXPanelMarker),
		elfWithMarker(t, runtime.GOARCH, newAgentMarker),
	)
}

func (h *componentUpgradeHarness) read(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func (h *componentUpgradeHarness) assertContains(t *testing.T, path, marker string) {
	t.Helper()
	if !bytes.Contains(h.read(t, path), []byte(marker)) {
		t.Fatalf("%s does not contain marker %q", path, marker)
	}
}

func (h *componentUpgradeHarness) assertConfigUnchanged(t *testing.T) {
	t.Helper()
	got := string(h.read(t, h.ConfigPath))
	if got != configFixtureBody {
		t.Fatalf("config.yml mutated:\n got: %q\nwant: %q", got, configFixtureBody)
	}
}

func (h *componentUpgradeHarness) assertNoDotNew(t *testing.T) {
	t.Helper()
	for _, p := range []string{h.XPanelPath + ".new", h.AgentPath + ".new"} {
		if _, err := os.Lstat(p); err == nil {
			t.Fatalf("leftover staging file %s", p)
		} else if !os.IsNotExist(err) {
			t.Fatalf("stat %s: %v", p, err)
		}
	}
}

func (h *componentUpgradeHarness) assertNoFixedBackups(t *testing.T) {
	t.Helper()
	for _, p := range []string{h.XPanelPath + ".bak", h.AgentPath + ".bak"} {
		if _, err := os.Lstat(p); err == nil {
			t.Fatalf("leftover or clobbered fixed backup %s", p)
		} else if !os.IsNotExist(err) {
			t.Fatalf("stat %s: %v", p, err)
		}
	}
}

func (h *componentUpgradeHarness) assertLiveUntouched(t *testing.T) {
	t.Helper()
	h.assertContains(t, h.XPanelPath, liveXPanelMarker)
	h.assertContains(t, h.AgentPath, liveAgentMarker)
	h.assertConfigUnchanged(t)
	h.assertNoDotNew(t)
}

// ----- extract / preflight -----

func TestComponentUpgradeExtractAllowsOnlyXPanelAndAgent(t *testing.T) {
	dir := t.TempDir()
	archive := validComponentArchive(t, dir,
		elfWithMarker(t, runtime.GOARCH, newXPanelMarker),
		elfWithMarker(t, runtime.GOARCH, newAgentMarker),
	)
	dest := filepath.Join(dir, "out")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatal(err)
	}

	xpanelPath, agentPath, err := extractComponentArchive(archive, dest)
	if err != nil {
		t.Fatalf("extractComponentArchive() error = %v", err)
	}
	if !bytes.Contains(mustRead(t, xpanelPath), []byte(newXPanelMarker)) {
		t.Fatalf("extracted xpanel missing marker at %s", xpanelPath)
	}
	if !bytes.Contains(mustRead(t, agentPath), []byte(newAgentMarker)) {
		t.Fatalf("extracted agent missing marker at %s", agentPath)
	}

	// Only the two required assets may land on disk (plus any empty parents).
	var found []string
	_ = filepath.Walk(dest, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(dest, path)
		found = append(found, filepath.ToSlash(rel))
		return nil
	})
	want := map[string]bool{
		componentArchiveXPanelName: true,
		componentArchiveAgentName:  true,
	}
	for _, f := range found {
		if !want[f] {
			t.Fatalf("unexpected extracted file %q (found=%v)", f, found)
		}
	}
	if len(found) != 2 {
		t.Fatalf("extracted files = %v, want exactly the two component assets", found)
	}
}

func TestComponentUpgradeExtractRejectsPathTraversal(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "bad.tar.gz")
	writeGzipTar(t, archive, []tarEntry{
		{Name: "../escape-xpanel", Body: elfWithMarker(t, runtime.GOARCH, newXPanelMarker)},
		{Name: componentArchiveAgentName, Body: elfWithMarker(t, runtime.GOARCH, newAgentMarker)},
	})
	dest := filepath.Join(dir, "out")
	_ = os.MkdirAll(dest, 0o755)

	if _, _, err := extractComponentArchive(archive, dest); err == nil {
		t.Fatal("extractComponentArchive() error = nil, want path-traversal rejection")
	}
	assertDirHasNoRegularFiles(t, dest)
}

func TestComponentUpgradeExtractRejectsSymlinkAsset(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "symlink.tar.gz")
	writeGzipTar(t, archive, []tarEntry{
		{Name: componentArchiveXPanelName, Body: elfWithMarker(t, runtime.GOARCH, newXPanelMarker)},
		{Name: componentArchiveAgentName, Typeflag: tar.TypeSymlink, Linkname: "/etc/passwd"},
	})
	dest := filepath.Join(dir, "out")
	_ = os.MkdirAll(dest, 0o755)

	if _, _, err := extractComponentArchive(archive, dest); err == nil {
		t.Fatal("extractComponentArchive() error = nil, want symlink rejection")
	}
	assertDirHasNoRegularFiles(t, dest)
}

func TestComponentUpgradeExtractRejectsNonRegularAsset(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "dir-asset.tar.gz")
	writeGzipTar(t, archive, []tarEntry{
		{Name: componentArchiveXPanelName, Typeflag: tar.TypeDir, Mode: 0755},
		{Name: componentArchiveAgentName, Body: elfWithMarker(t, runtime.GOARCH, newAgentMarker)},
	})
	dest := filepath.Join(dir, "out")
	_ = os.MkdirAll(dest, 0o755)

	if _, _, err := extractComponentArchive(archive, dest); err == nil {
		t.Fatal("extractComponentArchive() error = nil, want non-regular rejection")
	}
	assertDirHasNoRegularFiles(t, dest)
}

func TestComponentUpgradeExtractRejectsDuplicateTarget(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "dup.tar.gz")
	body := elfWithMarker(t, runtime.GOARCH, newXPanelMarker)
	writeGzipTar(t, archive, []tarEntry{
		{Name: componentArchiveXPanelName, Body: body},
		{Name: componentArchiveXPanelName, Body: body}, // duplicate xpanel
		{Name: componentArchiveAgentName, Body: elfWithMarker(t, runtime.GOARCH, newAgentMarker)},
	})
	dest := filepath.Join(dir, "out")
	_ = os.MkdirAll(dest, 0o755)

	if _, _, err := extractComponentArchive(archive, dest); err == nil {
		t.Fatal("extractComponentArchive() error = nil, want duplicate-target rejection")
	}
}

func TestComponentUpgradeExtractRejectsMissingXPanel(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "no-xpanel.tar.gz")
	writeGzipTar(t, archive, []tarEntry{
		{Name: componentArchiveAgentName, Body: elfWithMarker(t, runtime.GOARCH, newAgentMarker)},
	})
	dest := filepath.Join(dir, "out")
	_ = os.MkdirAll(dest, 0o755)

	if _, _, err := extractComponentArchive(archive, dest); err == nil {
		t.Fatal("extractComponentArchive() error = nil, want missing xpanel rejection")
	}
}

func TestComponentUpgradeExtractRejectsMissingAgent(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "no-agent.tar.gz")
	writeGzipTar(t, archive, []tarEntry{
		{Name: componentArchiveXPanelName, Body: elfWithMarker(t, runtime.GOARCH, newXPanelMarker)},
	})
	dest := filepath.Join(dir, "out")
	_ = os.MkdirAll(dest, 0o755)

	if _, _, err := extractComponentArchive(archive, dest); err == nil {
		t.Fatal("extractComponentArchive() error = nil, want missing agent rejection")
	}
}

func TestComponentUpgradeValidateELFAcceptsRuntimeArch(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bin")
	if err := os.WriteFile(path, elfWithMarker(t, runtime.GOARCH, "ok"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := validateComponentELF(path); err != nil {
		t.Fatalf("validateComponentELF() error = %v, want nil for runtime GOARCH", err)
	}
}

func TestComponentUpgradeValidateELFRejectsWrongArch(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bin")
	if err := os.WriteFile(path, elfWithMarker(t, wrongArchForRuntime(t), "bad"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := validateComponentELF(path); err == nil {
		t.Fatal("validateComponentELF() error = nil, want wrong-arch rejection")
	}
}

func TestComponentUpgradeWrongArchFailsBeforeLiveOrSystemd(t *testing.T) {
	h := newComponentUpgradeHarness(t, "active", "enabled")
	archive := validComponentArchive(t, h.Root,
		elfWithMarker(t, wrongArchForRuntime(t), newXPanelMarker),
		elfWithMarker(t, runtime.GOARCH, newAgentMarker),
	)

	err := applyComponentPackage(h.deps(), archive)
	if err == nil {
		t.Fatal("applyComponentPackage() error = nil, want wrong-arch preflight failure")
	}
	h.assertLiveUntouched(t)
	h.Runner.assertNoMutatingSystemctl(t)
	if len(h.ReplaceCalls) != 0 {
		t.Fatalf("ReplaceBinary must not run on preflight failure; calls=%v", h.ReplaceCalls)
	}
	if h.RestartCalls != 0 {
		t.Fatalf("RestartXPanel must not run on preflight failure; calls=%d", h.RestartCalls)
	}
}

func TestComponentUpgradeMissingAssetFailsBeforeLiveOrSystemd(t *testing.T) {
	h := newComponentUpgradeHarness(t, "active", "enabled")
	archive := filepath.Join(h.Root, "missing-agent.tar.gz")
	writeGzipTar(t, archive, []tarEntry{
		{Name: componentArchiveXPanelName, Body: elfWithMarker(t, runtime.GOARCH, newXPanelMarker)},
	})

	err := applyComponentPackage(h.deps(), archive)
	if err == nil {
		t.Fatal("applyComponentPackage() error = nil, want missing-asset preflight failure")
	}
	h.assertLiveUntouched(t)
	h.Runner.assertNoMutatingSystemctl(t)
	if len(h.ReplaceCalls) != 0 {
		t.Fatalf("ReplaceBinary must not run on preflight failure; calls=%v", h.ReplaceCalls)
	}
}

func TestComponentUpgradeSymlinkFailsBeforeLiveOrSystemd(t *testing.T) {
	h := newComponentUpgradeHarness(t, "inactive", "disabled")
	archive := filepath.Join(h.Root, "symlink.tar.gz")
	writeGzipTar(t, archive, []tarEntry{
		{Name: componentArchiveXPanelName, Body: elfWithMarker(t, runtime.GOARCH, newXPanelMarker)},
		{Name: componentArchiveAgentName, Typeflag: tar.TypeSymlink, Linkname: "xpanel"},
	})

	err := applyComponentPackage(h.deps(), archive)
	if err == nil {
		t.Fatal("applyComponentPackage() error = nil, want symlink preflight failure")
	}
	h.assertLiveUntouched(t)
	h.Runner.assertNoMutatingSystemctl(t)
}

func TestComponentUpgradeSystemdStateErrorsFailClosedBeforeMutation(t *testing.T) {
	for _, verb := range []string{"is-active", "is-enabled"} {
		t.Run(verb, func(t *testing.T) {
			h := newComponentUpgradeHarness(t, "inactive", "disabled")
			h.Runner.failOps[verb] = errors.New("injected systemd state failure")

			err := applyComponentPackage(h.deps(), h.newArchive(t))
			if err == nil {
				t.Fatalf("applyComponentPackage() error = nil for %s failure", verb)
			}
			h.assertLiveUntouched(t)
			h.Runner.assertNoMutatingSystemctl(t)
			if len(h.ReplaceCalls) != 0 {
				t.Fatalf("ReplaceBinary ran after %s failure: %#v", verb, h.ReplaceCalls)
			}
		})
	}
}

// ----- transaction / state preservation -----

func TestComponentUpgradeActiveEnabledStopReplaceRestoreOrder(t *testing.T) {
	h := newComponentUpgradeHarness(t, "active", "enabled")
	archive := h.newArchive(t)

	if err := applyComponentPackage(h.deps(), archive); err != nil {
		t.Fatalf("applyComponentPackage() error = %v", err)
	}

	// Binaries upgraded.
	h.assertContains(t, h.XPanelPath, newXPanelMarker)
	h.assertContains(t, h.AgentPath, newAgentMarker)
	h.assertConfigUnchanged(t)
	h.assertNoDotNew(t)
	h.assertNoFixedBackups(t)

	// Order: stop → Agent replace → XPanel replace → Agent restore → XPanel restart.
	stopIdx := indexOf(h.Timeline, "systemctl:stop")
	agentIdx := indexOf(h.Timeline, "replace:agent")
	xpanelIdx := indexOf(h.Timeline, "replace:xpanel")
	restoreIdx := indexOfAny(h.Timeline, "systemctl:start", "systemctl:restart")
	restartPanelIdx := indexOf(h.Timeline, "restart-xpanel")
	if stopIdx < 0 || agentIdx < 0 || xpanelIdx < 0 || restoreIdx < 0 || restartPanelIdx < 0 {
		t.Fatalf("missing required timeline events: %v", h.Timeline)
	}
	if !(stopIdx < agentIdx && agentIdx < xpanelIdx && xpanelIdx < restoreIdx && restoreIdx < restartPanelIdx) {
		t.Fatalf("timeline order want stop→agent→xpanel→restore→restart-xpanel; got %v", h.Timeline)
	}
	if len(h.ReplaceCalls) != 2 {
		t.Fatalf("ReplaceCalls = %d, want 2 (agent then xpanel); %#v", len(h.ReplaceCalls), h.ReplaceCalls)
	}
	if h.ReplaceCalls[0].Dst != h.AgentPath {
		t.Fatalf("first replace dst = %s, want agent %s", h.ReplaceCalls[0].Dst, h.AgentPath)
	}
	if h.ReplaceCalls[1].Dst != h.XPanelPath {
		t.Fatalf("second replace dst = %s, want xpanel %s", h.ReplaceCalls[1].Dst, h.XPanelPath)
	}
	h.Runner.assertNoEnableDisable(t)
	if h.RestartCalls != 1 {
		t.Fatalf("RestartXPanel calls = %d, want 1", h.RestartCalls)
	}
	if h.Runner.active != "active" {
		t.Fatalf("agent active = %q, want active after restore", h.Runner.active)
	}
	if h.Runner.enabled != "enabled" {
		t.Fatalf("agent enabled = %q, want enabled preserved", h.Runner.enabled)
	}
}

func TestComponentUpgradeInactiveEnabledStaysStoppedEnabled(t *testing.T) {
	h := newComponentUpgradeHarness(t, "inactive", "enabled")
	archive := h.newArchive(t)

	if err := applyComponentPackage(h.deps(), archive); err != nil {
		t.Fatalf("applyComponentPackage() error = %v", err)
	}

	h.assertContains(t, h.XPanelPath, newXPanelMarker)
	h.assertContains(t, h.AgentPath, newAgentMarker)
	h.assertConfigUnchanged(t)

	verbs := h.Runner.systemctlVerbs()
	for _, v := range verbs {
		if v == "stop" || v == "start" || v == "restart" {
			t.Fatalf("inactive agent must stay stopped; verbs=%v", verbs)
		}
	}
	h.Runner.assertNoEnableDisable(t)
	if h.Runner.active != "inactive" {
		t.Fatalf("active = %q, want inactive", h.Runner.active)
	}
	if h.Runner.enabled != "enabled" {
		t.Fatalf("enabled = %q, want enabled", h.Runner.enabled)
	}
}

func TestComponentUpgradeInactiveDisabledStaysStoppedDisabled(t *testing.T) {
	h := newComponentUpgradeHarness(t, "inactive", "disabled")
	archive := h.newArchive(t)

	if err := applyComponentPackage(h.deps(), archive); err != nil {
		t.Fatalf("applyComponentPackage() error = %v", err)
	}

	h.assertContains(t, h.XPanelPath, newXPanelMarker)
	h.assertContains(t, h.AgentPath, newAgentMarker)
	h.assertConfigUnchanged(t)

	verbs := h.Runner.systemctlVerbs()
	for _, v := range verbs {
		if v == "stop" || v == "start" || v == "restart" || v == "enable" || v == "disable" {
			t.Fatalf("disabled inactive agent must not be mutated; verbs=%v", verbs)
		}
	}
	if h.Runner.active != "inactive" || h.Runner.enabled != "disabled" {
		t.Fatalf("state = active:%s enabled:%s, want inactive/disabled", h.Runner.active, h.Runner.enabled)
	}
}

func TestComponentUpgradeActiveDisabledPreservesBothDimensions(t *testing.T) {
	// Unusual but allowed: process active while unit disabled (manual start).
	// Upgrade must restore the active dimension and leave enabled alone.
	h := newComponentUpgradeHarness(t, "active", "disabled")
	archive := h.newArchive(t)

	if err := applyComponentPackage(h.deps(), archive); err != nil {
		t.Fatalf("applyComponentPackage() error = %v", err)
	}

	h.assertContains(t, h.XPanelPath, newXPanelMarker)
	h.assertContains(t, h.AgentPath, newAgentMarker)
	h.assertConfigUnchanged(t)
	h.Runner.assertNoEnableDisable(t)

	verbs := h.Runner.systemctlVerbs()
	if indexOf(verbs, "stop") < 0 {
		t.Fatalf("active agent must be stopped before replace; verbs=%v", verbs)
	}
	if indexOfAny(verbs, "start", "restart") < 0 {
		t.Fatalf("previously-active agent must be restored; verbs=%v", verbs)
	}
	if h.Runner.active != "active" {
		t.Fatalf("active = %q, want active restored", h.Runner.active)
	}
	if h.Runner.enabled != "disabled" {
		t.Fatalf("enabled = %q, want disabled preserved", h.Runner.enabled)
	}
}

func TestComponentUpgradePreservesConfigBytesExactly(t *testing.T) {
	h := newComponentUpgradeHarness(t, "active", "enabled")
	// Include a decoy config.yml in the archive; production must never apply it.
	archive := filepath.Join(h.Root, "with-config.tar.gz")
	writeGzipTar(t, archive, []tarEntry{
		{Name: componentArchiveXPanelName, Body: elfWithMarker(t, runtime.GOARCH, newXPanelMarker)},
		{Name: componentArchiveAgentName, Body: elfWithMarker(t, runtime.GOARCH, newAgentMarker)},
		{Name: "nezha-agent/config.yml", Body: []byte("client_secret: MUST-NOT-LAND\n")},
	})
	before := h.read(t, h.ConfigPath)

	if err := applyComponentPackage(h.deps(), archive); err != nil {
		// Archive may reject the extra config entry entirely — also fine, as long as live config is untouched.
		if !bytes.Equal(h.read(t, h.ConfigPath), before) {
			t.Fatalf("config.yml changed after rejected/partial package")
		}
		// If extraction rejects unknown members, config preservation still holds.
		// Prefer success path when implementation ignores non-required members.
		if strings.Contains(err.Error(), "config") || strings.Contains(strings.ToLower(err.Error()), "unexpected") {
			h.assertLiveUntouched(t)
			return
		}
		t.Fatalf("applyComponentPackage() error = %v", err)
	}
	after := h.read(t, h.ConfigPath)
	if !bytes.Equal(before, after) {
		t.Fatalf("config.yml bytes changed:\n before=%q\n after=%q", before, after)
	}
	if bytes.Contains(after, []byte("MUST-NOT-LAND")) {
		t.Fatal("archive config.yml was applied to live config")
	}
}

func TestComponentUpgradeAgentReplaceFailureLeavesXPanelUntouched(t *testing.T) {
	h := newComponentUpgradeHarness(t, "active", "enabled")
	h.failReplaceDst = h.AgentPath
	archive := h.newArchive(t)

	err := applyComponentPackage(h.deps(), archive)
	if err == nil {
		t.Fatal("applyComponentPackage() error = nil, want agent replace failure")
	}

	h.assertContains(t, h.XPanelPath, liveXPanelMarker)
	// Agent must remain the live marker (failed replace / rolled back).
	h.assertContains(t, h.AgentPath, liveAgentMarker)
	h.assertConfigUnchanged(t)
	h.assertNoDotNew(t)
	if h.RestartCalls != 0 {
		t.Fatalf("RestartXPanel must not run when agent replace fails; calls=%d", h.RestartCalls)
	}
	// XPanel replace must not have been attempted after agent failure.
	for _, c := range h.ReplaceCalls {
		if c.Dst == h.XPanelPath {
			t.Fatalf("XPanel replace must not run after agent failure; calls=%#v", h.ReplaceCalls)
		}
	}
}

func TestComponentUpgradeXPanelReplaceFailureRollsBackAgent(t *testing.T) {
	h := newComponentUpgradeHarness(t, "active", "enabled")
	h.failReplaceDst = h.XPanelPath
	archive := h.newArchive(t)

	err := applyComponentPackage(h.deps(), archive)
	if err == nil {
		t.Fatal("applyComponentPackage() error = nil, want xpanel replace failure")
	}

	h.assertContains(t, h.XPanelPath, liveXPanelMarker)
	h.assertContains(t, h.AgentPath, liveAgentMarker)
	h.assertConfigUnchanged(t)
	h.assertNoDotNew(t)
	if h.RestartCalls != 0 {
		t.Fatalf("RestartXPanel must not run when xpanel replace fails; calls=%d", h.RestartCalls)
	}
}

func TestComponentUpgradeAgentRestartFailureRollsBackBothAndRestoresActive(t *testing.T) {
	h := newComponentUpgradeHarness(t, "active", "enabled")
	h.Runner.failOps["start"] = errors.New("injected agent start failure")
	h.Runner.failOps["restart"] = errors.New("injected agent restart failure")
	archive := h.newArchive(t)

	err := applyComponentPackage(h.deps(), archive)
	if err == nil {
		t.Fatal("applyComponentPackage() error = nil, want agent restore failure")
	}

	// Both binaries rolled back to pre-upgrade content.
	h.assertContains(t, h.XPanelPath, liveXPanelMarker)
	h.assertContains(t, h.AgentPath, liveAgentMarker)
	h.assertConfigUnchanged(t)
	h.assertNoDotNew(t)
	if h.RestartCalls != 0 {
		t.Fatalf("RestartXPanel must not run when agent restore fails; calls=%d", h.RestartCalls)
	}

	// Production should attempt to put the original active Agent back after rollback.
	// At least one restore attempt (start/restart) must appear after stop.
	verbs := h.Runner.systemctlVerbs()
	if indexOf(verbs, "stop") < 0 {
		t.Fatalf("expected initial stop; verbs=%v", verbs)
	}
	if indexOfAny(verbs, "start", "restart") < 0 {
		t.Fatalf("expected agent restore attempt; verbs=%v", verbs)
	}
}

func TestComponentUpgradeXPanelRestartFailureStopsNewAgentBeforeRollback(t *testing.T) {
	h := newComponentUpgradeHarness(t, "active", "enabled")
	h.failRestart = true

	err := applyComponentPackage(h.deps(), h.newArchive(t))
	if err == nil {
		t.Fatal("applyComponentPackage() error = nil, want X-Panel restart failure")
	}
	h.assertContains(t, h.XPanelPath, liveXPanelMarker)
	h.assertContains(t, h.AgentPath, liveAgentMarker)
	h.assertConfigUnchanged(t)
	h.assertNoDotNew(t)

	restartIdx := indexOf(h.Timeline, "restart-xpanel")
	if restartIdx < 0 {
		t.Fatalf("missing X-Panel restart attempt: %v", h.Timeline)
	}
	stopAfterRestart := indexOf(h.Timeline[restartIdx+1:], "systemctl:stop")
	startAfterRestart := indexOf(h.Timeline[restartIdx+1:], "systemctl:start")
	if stopAfterRestart < 0 || startAfterRestart < 0 || stopAfterRestart >= startAfterRestart {
		t.Fatalf("rollback must stop the new Agent before restoring/starting the old one: %v", h.Timeline)
	}
}

func TestComponentUpgradeMissingOriginalAgentIgnoresStaleFixedBackup(t *testing.T) {
	h := newComponentUpgradeHarness(t, "inactive", "disabled")
	if err := os.Remove(h.AgentPath); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(h.AgentPath+".bak", []byte("STALE-AGENT-MUST-NOT-RETURN"), 0o755); err != nil {
		t.Fatal(err)
	}
	h.failReplaceDst = h.XPanelPath

	err := applyComponentPackage(h.deps(), h.newArchive(t))
	if err == nil {
		t.Fatal("applyComponentPackage() error = nil, want X-Panel replace failure")
	}
	if _, err := os.Lstat(h.AgentPath); !os.IsNotExist(err) {
		t.Fatalf("AgentPath exists after rollback although it was originally absent: err=%v data=%q", err, h.read(t, h.AgentPath))
	}
	h.assertContains(t, h.XPanelPath, liveXPanelMarker)
	h.assertConfigUnchanged(t)
}

func TestComponentUpgradeFailureCleansDotNewWithoutRealOpt(t *testing.T) {
	h := newComponentUpgradeHarness(t, "inactive", "enabled")
	// Custom ReplaceBinary that leaves a .new behind on failure, then production
	// cleanup contract must still remove it — here we simulate a rename failure
	// after staging so the transaction's deferred cleanup is observable.
	deps := h.deps()
	deps.ReplaceBinary = func(src, dst string) error {
		h.ReplaceCalls = append(h.ReplaceCalls, componentReplaceCall{Src: src, Dst: dst})
		staging := dst + ".new"
		data, err := os.ReadFile(src)
		if err != nil {
			return err
		}
		if err := os.WriteFile(staging, data, 0o755); err != nil {
			return err
		}
		if dst == h.AgentPath {
			// Leave staging file and fail — applyComponentPackage must clean it.
			return errors.New("injected rename failure after staging")
		}
		if err := os.Rename(staging, dst); err != nil {
			_ = os.Remove(staging)
			return err
		}
		return nil
	}

	err := applyComponentPackage(deps, h.newArchive(t))
	if err == nil {
		t.Fatal("applyComponentPackage() error = nil, want staging failure")
	}
	h.assertNoDotNew(t)
	// Live install still under TempDir — never touches /opt.
	if strings.HasPrefix(h.XPanelPath, "/opt/") || strings.HasPrefix(h.AgentPath, "/opt/") {
		t.Fatal("harness must not use real /opt paths")
	}
	h.assertContains(t, h.XPanelPath, liveXPanelMarker)
	h.assertContains(t, h.AgentPath, liveAgentMarker)
}

func TestComponentUpgradeDoesNotDeletePreexistingRollbackNamedFiles(t *testing.T) {
	h := newComponentUpgradeHarness(t, "inactive", "enabled")
	sentinels := []string{
		h.XPanelPath + ".rollback-user-sentinel",
		h.AgentPath + ".rollback-user-sentinel",
	}
	for _, p := range sentinels {
		if err := os.WriteFile(p, []byte("must-stay"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	if err := applyComponentPackage(h.deps(), h.newArchive(t)); err != nil {
		t.Fatalf("applyComponentPackage() error = %v", err)
	}
	for _, p := range sentinels {
		if got := string(mustRead(t, p)); got != "must-stay" {
			t.Fatalf("preexisting rollback-named file %s changed: %q", p, got)
		}
	}
}

func TestComponentUpgradeCreatesMissingAgentDirectoryMode0700(t *testing.T) {
	h := newComponentUpgradeHarness(t, "inactive", "disabled")
	agentDir := filepath.Dir(h.AgentPath)
	if err := os.RemoveAll(agentDir); err != nil {
		t.Fatal(err)
	}
	deps := h.deps()
	deps.ReplaceBinary = nil // exercise the production default replacement path

	if err := applyComponentPackage(deps, h.newArchive(t)); err != nil {
		t.Fatalf("applyComponentPackage() error = %v", err)
	}
	info, err := os.Stat(agentDir)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o700 {
		t.Fatalf("Agent directory mode = %#o, want 0700", got)
	}
}

// ----- helpers -----

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func assertDirHasNoRegularFiles(t *testing.T, dir string) {
	t.Helper()
	var found []string
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Mode().IsRegular() {
			rel, _ := filepath.Rel(dir, path)
			found = append(found, rel)
		}
		return nil
	})
	if len(found) > 0 {
		t.Fatalf("expected no extracted regular files, found %v", found)
	}
}

func indexOf(items []string, want string) int {
	for i, v := range items {
		if v == want {
			return i
		}
	}
	return -1
}

func indexOfAny(items []string, want ...string) int {
	set := make(map[string]struct{}, len(want))
	for _, w := range want {
		set[w] = struct{}{}
	}
	for i, v := range items {
		if _, ok := set[v]; ok {
			return i
		}
	}
	return -1
}
