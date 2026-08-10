package service

import (
	"archive/tar"
	"compress/gzip"
	"debug/elf"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	// component archive members (slash-normalized, no leading ./)
	componentArchiveXPanelRel = "xpanel"
	componentArchiveAgentRel  = "nezha-agent/nezha-agent"

	// Per-asset size cap for staged binaries (panel + agent are far smaller).
	componentArchiveMaxAssetBytes = 256 << 20 // 256 MiB
)

// componentCmdRunner runs external commands for the upgrade transaction
// (production: os/exec; tests inject fakes — never real systemctl).
type componentCmdRunner interface {
	CombinedOutput(name string, args ...string) ([]byte, error)
}

// componentUpgradeDeps is the injectable surface for applyComponentPackage.
// Production fills paths + default Runner/ReplaceBinary/RestartXPanel;
// tests inject a TempDir layout and fakes so no root/systemd/network is needed.
type componentUpgradeDeps struct {
	XPanelPath    string
	AgentPath     string
	ConfigPath    string // held for DI only — never read or rewritten
	AgentUnit     string
	Runner        componentCmdRunner
	RestartXPanel func() error
	ReplaceBinary func(src, dst string) error
}

type execComponentRunner struct{}

func (execComponentRunner) CombinedOutput(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).CombinedOutput()
}

// defaultReplaceBinary stages dst+".new" in the same directory, chmod 0755, then rename.
func defaultReplaceBinary(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	staging := dst + ".new"
	_ = os.Remove(staging)
	if err := copyFile(src, staging); err != nil {
		_ = os.Remove(staging)
		return err
	}
	if err := os.Chmod(staging, 0o755); err != nil {
		_ = os.Remove(staging)
		return err
	}
	if err := os.Rename(staging, dst); err != nil {
		_ = os.Remove(staging)
		return err
	}
	return nil
}

// defaultRestartXPanel starts systemctl restart xpanel asynchronously so the
// current process can exit when systemd stops it.
func defaultRestartXPanel() error {
	cmd := exec.Command("systemctl", "restart", "xpanel")
	return cmd.Start()
}

// productionComponentUpgradeDeps builds live-install deps for the running panel.
func productionComponentUpgradeDeps(xpanelPath string) componentUpgradeDeps {
	return componentUpgradeDeps{
		XPanelPath:    xpanelPath,
		AgentPath:     NezhaAgentBinaryPath,
		ConfigPath:    NezhaAgentConfigPath,
		AgentUnit:     NezhaAgentUnitName,
		Runner:        execComponentRunner{},
		RestartXPanel: defaultRestartXPanel,
		ReplaceBinary: defaultReplaceBinary,
	}
}

func (d *componentUpgradeDeps) withDefaults() {
	if d.AgentUnit == "" {
		d.AgentUnit = NezhaAgentUnitName
	}
	if d.Runner == nil {
		d.Runner = execComponentRunner{}
	}
	if d.ReplaceBinary == nil {
		d.ReplaceBinary = defaultReplaceBinary
	}
	if d.RestartXPanel == nil {
		d.RestartXPanel = defaultRestartXPanel
	}
}

// normalizeComponentArchiveName rejects absolute paths and ".." traversal, and
// normalizes "./xpanel" / "./nezha-agent/nezha-agent" to slash-clean relatives.
func normalizeComponentArchiveName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("empty archive entry name")
	}
	// Absolute paths (unix, unc-ish, drive-letter).
	if strings.HasPrefix(name, "/") || strings.HasPrefix(name, `\`) {
		return "", fmt.Errorf("absolute archive path rejected: %q", name)
	}
	if len(name) >= 2 && name[1] == ':' {
		return "", fmt.Errorf("absolute archive path rejected: %q", name)
	}
	name = filepath.ToSlash(name)
	for strings.HasPrefix(name, "./") {
		name = name[2:]
	}
	if name == "" || name == "." {
		return "", fmt.Errorf("invalid archive entry name")
	}
	// Reject ".." before and after Clean so Clean cannot mask traversal.
	for _, part := range strings.Split(name, "/") {
		if part == ".." {
			return "", fmt.Errorf("path traversal rejected: %q", name)
		}
	}
	cleaned := path.Clean(name)
	if cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", fmt.Errorf("path traversal rejected: %q", name)
	}
	for _, part := range strings.Split(cleaned, "/") {
		if part == ".." || part == "" {
			if part == ".." {
				return "", fmt.Errorf("path traversal rejected: %q", name)
			}
		}
	}
	return cleaned, nil
}

// extractComponentArchive safely extracts only xpanel and nezha-agent/nezha-agent
// using archive/tar + compress/gzip. On any error it removes regular files already
// written under destDir so callers never observe a partial extract.
func extractComponentArchive(archivePath, destDir string) (xpanelPath, agentPath string, err error) {
	var written []string
	cleanup := func() {
		for _, p := range written {
			_ = os.Remove(p)
		}
	}
	defer func() {
		if err != nil {
			cleanup()
		}
	}()

	if err = os.MkdirAll(destDir, 0o755); err != nil {
		return "", "", err
	}

	f, err := os.Open(archivePath)
	if err != nil {
		return "", "", err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return "", "", fmt.Errorf("gzip open: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	var gotXPanel, gotAgent bool

	for {
		var hdr *tar.Header
		hdr, err = tr.Next()
		if err == io.EOF {
			err = nil
			break
		}
		if err != nil {
			return "", "", fmt.Errorf("tar read: %w", err)
		}

		rel, nerr := normalizeComponentArchiveName(hdr.Name)
		if nerr != nil {
			err = nerr
			return "", "", err
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			// Directories are allowed (e.g. nezha-agent/) but never create paths
			// outside destDir; only materialize parents when writing targets.
			continue
		case tar.TypeReg, tar.TypeRegA:
			// only regular files may land — and only the two component assets
		case tar.TypeSymlink, tar.TypeLink:
			err = fmt.Errorf("archive entry %q: links are not allowed", hdr.Name)
			return "", "", err
		default:
			err = fmt.Errorf("archive entry %q: unsupported type %v (only regular files and directories)", hdr.Name, hdr.Typeflag)
			return "", "", err
		}

		switch rel {
		case componentArchiveXPanelRel, componentArchiveAgentRel:
			// accepted targets below
		default:
			// Ignore non-required regular members (e.g. decoy config.yml) without
			// writing them — configPath must never be applied from the archive.
			// tar.Reader.Next discards any unread body on the following entry.
			continue
		}

		if hdr.Size < 0 || hdr.Size > componentArchiveMaxAssetBytes {
			err = fmt.Errorf("archive entry %q exceeds size limit (%d bytes)", hdr.Name, componentArchiveMaxAssetBytes)
			return "", "", err
		}

		switch rel {
		case componentArchiveXPanelRel:
			if gotXPanel {
				err = fmt.Errorf("duplicate archive target %q", rel)
				return "", "", err
			}
		case componentArchiveAgentRel:
			if gotAgent {
				err = fmt.Errorf("duplicate archive target %q", rel)
				return "", "", err
			}
		}

		dest := filepath.Join(destDir, filepath.FromSlash(rel))
		// Ensure join stays under destDir (defense in depth after normalize).
		if !isPathInside(destDir, dest) {
			err = fmt.Errorf("archive entry escapes destination: %q", hdr.Name)
			return "", "", err
		}
		if err = os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return "", "", err
		}

		var out *os.File
		out, err = os.OpenFile(dest, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o755)
		if err != nil {
			return "", "", err
		}
		written = append(written, dest)

		limited := io.LimitReader(tr, componentArchiveMaxAssetBytes+1)
		var n int64
		n, err = io.Copy(out, limited)
		closeErr := out.Close()
		if err != nil {
			return "", "", err
		}
		if closeErr != nil {
			err = closeErr
			return "", "", err
		}
		if n > componentArchiveMaxAssetBytes {
			err = fmt.Errorf("archive entry %q exceeds size limit (%d bytes)", hdr.Name, componentArchiveMaxAssetBytes)
			return "", "", err
		}

		switch rel {
		case componentArchiveXPanelRel:
			gotXPanel = true
			xpanelPath = dest
		case componentArchiveAgentRel:
			gotAgent = true
			agentPath = dest
		}
	}

	if !gotXPanel {
		err = fmt.Errorf("component archive missing %q", componentArchiveXPanelRel)
		return "", "", err
	}
	if !gotAgent {
		err = fmt.Errorf("component archive missing %q", componentArchiveAgentRel)
		return "", "", err
	}
	return xpanelPath, agentPath, nil
}

func isPathInside(root, target string) bool {
	absRoot, err1 := filepath.Abs(root)
	absTarget, err2 := filepath.Abs(target)
	if err1 != nil || err2 != nil {
		return false
	}
	rel, err := filepath.Rel(absRoot, absTarget)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}

// validateComponentELF ensures path is a parseable ELF whose machine matches runtime.GOARCH.
func validateComponentELF(path string) error {
	f, err := elf.Open(path)
	if err != nil {
		return fmt.Errorf("open ELF %s: %w", path, err)
	}
	defer f.Close()

	want, err := elfMachineForGOARCH(runtime.GOARCH)
	if err != nil {
		return err
	}
	if f.Machine != want {
		return fmt.Errorf("ELF arch mismatch for %s: machine=%v want=%v (GOARCH=%s)", path, f.Machine, want, runtime.GOARCH)
	}
	return nil
}

func elfMachineForGOARCH(goarch string) (elf.Machine, error) {
	switch goarch {
	case "amd64":
		return elf.EM_X86_64, nil
	case "arm64":
		return elf.EM_AARCH64, nil
	default:
		return 0, fmt.Errorf("unsupported GOARCH %q for ELF preflight", goarch)
	}
}

// preflightLiveBinary rejects existing symlink / non-regular live targets.
// Missing path is allowed (first-time Agent install via upgrade).
func preflightLiveBinary(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("live target %s is a symlink", path)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("live target %s is not a regular file", path)
	}
	return nil
}

func componentSystemctl(deps componentUpgradeDeps, args ...string) ([]byte, error) {
	return deps.Runner.CombinedOutput("systemctl", args...)
}

// componentKnownActiveState is the closed set of systemctl is-active outputs we
// accept. Non-zero exit with one of these states is fine; empty, permission/
// bus noise, or any other text is a hard error (fail closed before mutation).
func componentKnownActiveState(state string) bool {
	switch state {
	case "active", "inactive", "failed", "activating", "deactivating", "reloading":
		return true
	default:
		return false
	}
}

// componentKnownEnabledState is the closed set of systemctl is-enabled outputs
// we accept. Upgrade never enable/disable; it only needs a reliable capture.
func componentKnownEnabledState(state string) bool {
	switch state {
	case "enabled", "enabled-runtime", "linked", "linked-runtime", "alias",
		"masked", "masked-runtime", "static", "indirect", "disabled",
		"generated", "transient", "not-found":
		return true
	default:
		return false
	}
}

// componentUnitWasActive reports whether the unit was in a running state that
// must be stopped before replace and restored afterward.
func componentUnitWasActive(state string) bool {
	switch state {
	case "active", "activating", "reloading", "deactivating":
		return true
	default:
		return false
	}
}

// captureAgentActive reports whether the unit is active. systemctl is-active
// returns non-zero for inactive/failed; that is accepted when the printed state
// is known. Unknown text, empty output, or transport failures fail closed.
func captureAgentActive(deps componentUpgradeDeps) (bool, error) {
	out, err := componentSystemctl(deps, "is-active", deps.AgentUnit)
	state := strings.TrimSpace(string(out))
	if state == "" {
		if err != nil {
			return false, fmt.Errorf("systemctl is-active %s: %w", deps.AgentUnit, err)
		}
		return false, fmt.Errorf("systemctl is-active %s: empty state", deps.AgentUnit)
	}
	if !componentKnownActiveState(state) {
		if err != nil {
			return false, fmt.Errorf("systemctl is-active %s: unexpected state %q: %w", deps.AgentUnit, state, err)
		}
		return false, fmt.Errorf("systemctl is-active %s: unexpected state %q", deps.AgentUnit, state)
	}
	return componentUnitWasActive(state), nil
}

// captureAgentEnabled records the enabled dimension for preservation only.
// Upgrade must never call enable/disable. Non-zero exit with a known state
// (disabled/static/masked/not-found/…) is fine; unknown text fails closed.
func captureAgentEnabled(deps componentUpgradeDeps) (string, error) {
	out, err := componentSystemctl(deps, "is-enabled", deps.AgentUnit)
	state := strings.TrimSpace(string(out))
	if state == "" {
		if err != nil {
			return "", fmt.Errorf("systemctl is-enabled %s: %w", deps.AgentUnit, err)
		}
		return "", fmt.Errorf("systemctl is-enabled %s: empty state", deps.AgentUnit)
	}
	if !componentKnownEnabledState(state) {
		if err != nil {
			return "", fmt.Errorf("systemctl is-enabled %s: unexpected state %q: %w", deps.AgentUnit, state, err)
		}
		return "", fmt.Errorf("systemctl is-enabled %s: unexpected state %q", deps.AgentUnit, state)
	}
	return state, nil
}

// componentFileBackup tracks a unique same-directory temp backup of a live
// binary. Fixed dst+".bak" is never used — historical .bak files must not be
// read or deleted by this transaction.
type componentFileBackup struct {
	livePath string
	// path is the unique temp backup; empty when the live file did not exist.
	path string
	// existed is whether livePath was present before replace.
	existed bool
}

func (b componentFileBackup) cleanup() {
	if b.path != "" {
		_ = os.Remove(b.path)
	}
}

// backupLiveBinary creates a unique 0600 temp backup next to livePath when the
// live regular file exists. Missing livePath is recorded (existed=false) and is
// not an error — rollback will remove any newly installed file.
func backupLiveBinary(livePath string) (componentFileBackup, error) {
	b := componentFileBackup{livePath: livePath}
	info, err := os.Lstat(livePath)
	if err != nil {
		if os.IsNotExist(err) {
			return b, nil
		}
		return b, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return b, fmt.Errorf("cannot backup non-regular %s", livePath)
	}
	b.existed = true

	dir := filepath.Dir(livePath)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return b, err
	}
	// CreateTemp uses 0600; same directory so rename-based restore stays atomic.
	f, err := os.CreateTemp(dir, filepath.Base(livePath)+".upgrade-bak-*")
	if err != nil {
		return b, err
	}
	tmpPath := f.Name()
	if err := f.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return b, err
	}
	if err := copyFile(livePath, tmpPath); err != nil {
		_ = os.Remove(tmpPath)
		return b, err
	}
	// Restored binaries must be executable; keep backup content restore-ready.
	if err := os.Chmod(tmpPath, 0o755); err != nil {
		_ = os.Remove(tmpPath)
		return b, err
	}
	b.path = tmpPath
	return b, nil
}

// restoreLiveBinary atomically restores from the unique backup via same-dir
// staging+rename. If the original live file did not exist, the new file is
// removed — historical .bak paths are never consulted.
func restoreLiveBinary(b componentFileBackup) error {
	if !b.existed {
		if err := os.Remove(b.livePath); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	if b.path == "" {
		return fmt.Errorf("missing unique backup for %s", b.livePath)
	}
	info, err := os.Lstat(b.path)
	if err != nil {
		return fmt.Errorf("stat backup %s: %w", b.path, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("backup %s is not a regular file", b.path)
	}

	dir := filepath.Dir(b.livePath)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	sf, err := os.CreateTemp(dir, filepath.Base(b.livePath)+".rollback-*")
	if err != nil {
		return err
	}
	staging := sf.Name()
	if err := sf.Close(); err != nil {
		_ = os.Remove(staging)
		return err
	}
	// Always remove rollback staging if it still exists (rename success moves it).
	defer func() { _ = os.Remove(staging) }()

	if err := copyFile(b.path, staging); err != nil {
		return err
	}
	if err := os.Chmod(staging, 0o755); err != nil {
		return err
	}
	if err := os.Rename(staging, b.livePath); err != nil {
		return err
	}
	return nil
}

// cleanStagingNew removes only the exact dst+".new" staging path for each live
// target this transaction knows about. It must never Glob or delete any
// preexisting *.rollback-* names — restoreLiveBinary owns its CreateTemp path
// via defer and is sufficient for in-flight rollback staging cleanup.
func cleanStagingNew(paths ...string) {
	for _, p := range paths {
		_ = os.Remove(p + ".new")
	}
}

// ensureAgentParentDir prepares the Agent binary parent directory before
// replace. Missing parents are created 0700; existing paths must be real
// (non-symlink) directories and are chmod'd to 0700. XPanel parents are not
// touched here — only AgentPath.
func ensureAgentParentDir(agentPath string) error {
	dir := filepath.Dir(agentPath)
	if dir == "" || dir == "." {
		return fmt.Errorf("invalid agent parent directory for %s", agentPath)
	}
	info, err := os.Lstat(dir)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return err
		}
		if err := os.Chmod(dir, 0o700); err != nil {
			return err
		}
		return nil
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("agent parent directory %s is a symlink", dir)
	}
	if !info.IsDir() {
		return fmt.Errorf("agent parent path %s is not a directory", dir)
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return err
	}
	return nil
}

// restoreAgentRunning best-effort starts the Agent unit after rollback.
// Prefer start; if that fails (e.g. stop during rollback failed and unit is
// still running), try restart. Errors are returned, not swallowed.
func restoreAgentRunning(deps componentUpgradeDeps) error {
	out, startErr := componentSystemctl(deps, "start", deps.AgentUnit)
	if startErr == nil {
		return nil
	}
	out2, restartErr := componentSystemctl(deps, "restart", deps.AgentUnit)
	if restartErr == nil {
		return nil
	}
	return errors.Join(
		fmt.Errorf("start agent: %w (%s)", startErr, strings.TrimSpace(string(out))),
		fmt.Errorf("restart agent: %w (%s)", restartErr, strings.TrimSpace(string(out2))),
	)
}

// applyComponentPackage extracts, preflights, and transactionally replaces Agent
// then X-Panel. ConfigPath is never read or written. On any post-stop failure:
// roll back replaced binaries, clean exact .new staging and unique backups,
// and restore the Agent active dimension (never enable/disable).
func applyComponentPackage(deps componentUpgradeDeps, archivePath string) (err error) {
	deps.withDefaults()

	// ConfigPath must never be read or overwritten by this transaction.
	_ = deps.ConfigPath

	tmpDir, err := os.MkdirTemp("", "xpanel-component-upgrade-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	extractDir := filepath.Join(tmpDir, "extract")
	if err := os.MkdirAll(extractDir, 0o755); err != nil {
		return err
	}

	newXPanel, newAgent, err := extractComponentArchive(archivePath, extractDir)
	if err != nil {
		return err
	}
	if err := validateComponentELF(newXPanel); err != nil {
		return err
	}
	if err := validateComponentELF(newAgent); err != nil {
		return err
	}

	if err := preflightLiveBinary(deps.XPanelPath); err != nil {
		return err
	}
	if err := preflightLiveBinary(deps.AgentPath); err != nil {
		return err
	}

	// Capture both systemd dimensions before any mutation. Fail closed on
	// unknown/empty/transport errors so stop/backup/replace never run blind.
	// Enabled is observed only so we preserve it by never calling enable/disable.
	wasActive, err := captureAgentActive(deps)
	if err != nil {
		return err
	}
	if _, err := captureAgentEnabled(deps); err != nil {
		return err
	}

	if wasActive {
		if out, stopErr := componentSystemctl(deps, "stop", deps.AgentUnit); stopErr != nil {
			return fmt.Errorf("stop agent: %w (%s)", stopErr, strings.TrimSpace(string(out)))
		}
	}

	// Unique same-dir backups (never fixed dst+".bak"). Always removed on exit.
	agentBackup, err := backupLiveBinary(deps.AgentPath)
	if err != nil {
		if wasActive {
			_ = restoreAgentRunning(deps)
		}
		return fmt.Errorf("backup agent: %w", err)
	}
	xpanelBackup, err := backupLiveBinary(deps.XPanelPath)
	if err != nil {
		agentBackup.cleanup()
		if wasActive {
			_ = restoreAgentRunning(deps)
		}
		return fmt.Errorf("backup xpanel: %w", err)
	}
	defer func() {
		agentBackup.cleanup()
		xpanelBackup.cleanup()
		cleanStagingNew(deps.AgentPath, deps.XPanelPath)
	}()

	// Agent parent must be a real 0700 directory before any replace. Create only
	// the AgentPath parent (not XPanel). Fail before mutation and restore the
	// original active dimension if we already stopped the unit.
	if err := ensureAgentParentDir(deps.AgentPath); err != nil {
		if wasActive {
			_ = restoreAgentRunning(deps)
		}
		return fmt.Errorf("ensure agent directory: %w", err)
	}

	agentReplaced := false
	xpanelReplaced := false

	rollbackReplaced := func() error {
		var errs []error
		if xpanelReplaced {
			if rerr := restoreLiveBinary(xpanelBackup); rerr != nil {
				errs = append(errs, fmt.Errorf("rollback xpanel: %w", rerr))
			}
			xpanelReplaced = false
		}
		if agentReplaced {
			if rerr := restoreLiveBinary(agentBackup); rerr != nil {
				errs = append(errs, fmt.Errorf("rollback agent: %w", rerr))
			}
			agentReplaced = false
		}
		cleanStagingNew(deps.AgentPath, deps.XPanelPath)
		return errors.Join(errs...)
	}

	// Replace order: Agent → XPanel.
	if err := deps.ReplaceBinary(newAgent, deps.AgentPath); err != nil {
		cleanStagingNew(deps.AgentPath, deps.XPanelPath)
		var restoreErr error
		if wasActive {
			restoreErr = restoreAgentRunning(deps)
		}
		return errors.Join(fmt.Errorf("replace agent: %w", err), restoreErr)
	}
	agentReplaced = true

	if err := deps.ReplaceBinary(newXPanel, deps.XPanelPath); err != nil {
		// Agent already replaced — roll it back; XPanel live file untouched.
		rbErr := rollbackReplaced()
		var restoreErr error
		if wasActive {
			restoreErr = restoreAgentRunning(deps)
		}
		return errors.Join(fmt.Errorf("replace xpanel: %w", err), rbErr, restoreErr)
	}
	xpanelReplaced = true

	// Restore Agent active only if it was active before; never enable/disable.
	if wasActive {
		if out, startErr := componentSystemctl(deps, "start", deps.AgentUnit); startErr != nil {
			// New Agent failed to start — roll both binaries back, then start old Agent.
			rbErr := rollbackReplaced()
			restoreErr := restoreAgentRunning(deps)
			return errors.Join(
				fmt.Errorf("restore agent active: %w (%s)", startErr, strings.TrimSpace(string(out))),
				rbErr,
				restoreErr,
			)
		}
	}

	// XPanel restart is last. On failure: stop new Agent → rollback both → start old Agent.
	if err := deps.RestartXPanel(); err != nil {
		var errs []error
		errs = append(errs, fmt.Errorf("restart xpanel: %w", err))

		if wasActive {
			if out, stopErr := componentSystemctl(deps, "stop", deps.AgentUnit); stopErr != nil {
				// Stop failed — still roll back binaries and try restart/start recovery.
				errs = append(errs, fmt.Errorf("stop agent before rollback: %w (%s)", stopErr, strings.TrimSpace(string(out))))
			}
		}

		if rbErr := rollbackReplaced(); rbErr != nil {
			errs = append(errs, rbErr)
		}

		if wasActive {
			if restoreErr := restoreAgentRunning(deps); restoreErr != nil {
				errs = append(errs, restoreErr)
			}
		}
		return errors.Join(errs...)
	}

	return nil
}
