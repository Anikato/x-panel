package service

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"xpanel/app/dto"
	"xpanel/app/repo"
)

const (
	// NezhaAgentDir is the root directory for the bundled Nezha Agent.
	NezhaAgentDir = "/opt/xpanel/nezha-agent"
	// NezhaAgentBinaryPath is the absolute path of the official Agent binary.
	NezhaAgentBinaryPath = "/opt/xpanel/nezha-agent/nezha-agent"
	// NezhaAgentConfigPath is the absolute path of the Agent config file.
	NezhaAgentConfigPath = "/opt/xpanel/nezha-agent/config.yml"
	// NezhaAgentUnitName is the systemd unit managed by X-Panel.
	NezhaAgentUnitName = "xpanel-nezha-agent"
	// NezhaAgentUnitPath is the bundled systemd unit installed by X-Panel.
	NezhaAgentUnitPath = "/etc/systemd/system/xpanel-nezha-agent.service"

	nezhaEnabledSettingKey      = "NezhaEnabled"
	nezhaServerSettingKey       = "NezhaServer"
	nezhaClientSecretSettingKey = "NezhaClientSecret"

	defaultNezhaStopTimeout = 15 * time.Second
	defaultNezhaPollEvery   = 200 * time.Millisecond

	// Conflict kinds returned in dto.NezhaAgentConflict.Kind.
	nezhaConflictKindUnit      = "unit"
	nezhaConflictKindProcess   = "process"
	nezhaConflictKindDirectory = "directory"
)

// defaultNezhaExternalAgentDirs are common external install locations.
// Production scans these; the test constructor leaves ExternalDirs empty.
var defaultNezhaExternalAgentDirs = []string{
	"/opt/nezha/agent",
	"/opt/nezha-agent",
	"/etc/nezha",
}

// nezhaCmdRunner runs external commands (production: os/exec).
type nezhaCmdRunner interface {
	CombinedOutput(name string, args ...string) ([]byte, error)
}

// nezhaSettingStore persists panel settings used by the Agent lifecycle.
type nezhaSettingStore interface {
	CreateOrUpdate(key, value string) error
	CreateOrUpdateMany(values map[string]string) error
	GetValueByKey(key string) (string, error)
}

type execNezhaCmdRunner struct{}

func (execNezhaCmdRunner) CombinedOutput(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).CombinedOutput()
}

type repoNezhaSettingStore struct {
	repo repo.ISettingRepo
}

func (s repoNezhaSettingStore) CreateOrUpdate(key, value string) error {
	return s.repo.CreateOrUpdate(key, value)
}

func (s repoNezhaSettingStore) CreateOrUpdateMany(values map[string]string) error {
	return s.repo.CreateOrUpdateMany(values)
}

func (s repoNezhaSettingStore) GetValueByKey(key string) (string, error) {
	return s.repo.GetValueByKey(key)
}

// nezhaAgentDeps holds injectable collaborators for tests and production wiring.
type nezhaAgentDeps struct {
	ConfigPath  string
	BinaryPath  string
	UnitPath    string
	Unit        string
	Runner      nezhaCmdRunner
	Settings    nezhaSettingStore
	Sleep       func(time.Duration)
	Now         func() time.Time
	StopTimeout time.Duration
	PollEvery   time.Duration

	// Optional file hooks; nil uses the Task 1 helpers.
	ReadConfig  func(path string) ([]byte, error)
	WriteConfig func(path string, data []byte) error

	// ListProcessExecutables returns real executable paths of running processes.
	// nil skips process scanning (test default; avoids host /proc dependency).
	ListProcessExecutables func() ([]string, error)
	// ExternalDirs are paths that indicate an external Agent install if present.
	// nil/empty skips directory scanning (test default).
	ExternalDirs []string
	// Lstat is used only for external directory existence checks.
	// nil defaults to os.Lstat.
	Lstat func(name string) (os.FileInfo, error)

	// CheckConflictFree is an optional extra hook for tests.
	// nil means “no extra check”. Non-nil error blocks enable --now on EnableAndStart.
	CheckConflictFree func() error
	InstallBundle     func() (archivePath string, cleanup func(), err error)
}

// NezhaAgentService manages the bundled Nezha Agent systemd lifecycle and config saves.
type NezhaAgentService struct {
	configPath string
	binaryPath string
	unitPath   string
	unit       string
	runner     nezhaCmdRunner
	settings   nezhaSettingStore
	sleep      func(time.Duration)
	now        func() time.Time
	stopWait   time.Duration
	pollEvery  time.Duration

	readConfig  func(path string) ([]byte, error)
	writeConfig func(path string, data []byte) error

	listProcessExecutables func() ([]string, error)
	externalDirs           []string
	lstat                  func(name string) (os.FileInfo, error)

	checkConflictFree func() error
	installBundle     func() (archivePath string, cleanup func(), err error)
}

// NewNezhaAgentService constructs the production service with real os/exec,
// setting repo, and host conflict detection (/proc + common external dirs).
func NewNezhaAgentService() *NezhaAgentService {
	return newNezhaAgentService(nezhaAgentDeps{
		ConfigPath:             NezhaAgentConfigPath,
		BinaryPath:             NezhaAgentBinaryPath,
		UnitPath:               NezhaAgentUnitPath,
		Unit:                   NezhaAgentUnitName,
		Runner:                 execNezhaCmdRunner{},
		Settings:               repoNezhaSettingStore{repo: repo.NewISettingRepo()},
		Sleep:                  time.Sleep,
		Now:                    time.Now,
		StopTimeout:            defaultNezhaStopTimeout,
		PollEvery:              defaultNezhaPollEvery,
		ListProcessExecutables: listNezhaProcessExecutablesFromProc,
		ExternalDirs:           append([]string(nil), defaultNezhaExternalAgentDirs...),
		InstallBundle:          downloadCurrentNezhaAgentBundle,
	})
}

func newNezhaAgentService(deps nezhaAgentDeps) *NezhaAgentService {
	if deps.ConfigPath == "" {
		deps.ConfigPath = NezhaAgentConfigPath
	}
	if deps.BinaryPath == "" {
		deps.BinaryPath = NezhaAgentBinaryPath
	}
	if deps.UnitPath == "" {
		deps.UnitPath = NezhaAgentUnitPath
	}
	if deps.Unit == "" {
		deps.Unit = NezhaAgentUnitName
	}
	if deps.Runner == nil {
		deps.Runner = execNezhaCmdRunner{}
	}
	if deps.Sleep == nil {
		deps.Sleep = time.Sleep
	}
	if deps.Now == nil {
		deps.Now = time.Now
	}
	if deps.StopTimeout <= 0 {
		deps.StopTimeout = defaultNezhaStopTimeout
	}
	if deps.PollEvery <= 0 {
		deps.PollEvery = defaultNezhaPollEvery
	}
	if deps.InstallBundle == nil {
		deps.InstallBundle = downloadCurrentNezhaAgentBundle
	}
	readFn := deps.ReadConfig
	if readFn == nil {
		readFn = readNezhaConfigFile
	}
	writeFn := deps.WriteConfig
	if writeFn == nil {
		writeFn = writeNezhaConfigFile
	}
	// ExternalDirs: preserve nil/empty (no host dir scan). Copy when non-empty.
	var extDirs []string
	if len(deps.ExternalDirs) > 0 {
		extDirs = append([]string(nil), deps.ExternalDirs...)
	}
	lstatFn := deps.Lstat
	if lstatFn == nil {
		lstatFn = os.Lstat
	}
	return &NezhaAgentService{
		configPath:             deps.ConfigPath,
		binaryPath:             deps.BinaryPath,
		unitPath:               deps.UnitPath,
		unit:                   deps.Unit,
		runner:                 deps.Runner,
		settings:               deps.Settings,
		sleep:                  deps.Sleep,
		now:                    deps.Now,
		stopWait:               deps.StopTimeout,
		pollEvery:              deps.PollEvery,
		readConfig:             readFn,
		writeConfig:            writeFn,
		listProcessExecutables: deps.ListProcessExecutables,
		externalDirs:           extDirs,
		lstat:                  lstatFn,
		checkConflictFree:      deps.CheckConflictFree,
		installBundle:          deps.InstallBundle,
	}
}

// Status reports component/config/service state for the bundled Agent.
// AgentSecret is never returned; only SecretConfigured is exposed.
// Partial failures fill configError / serviceError / permissionsWarning and still
// return a status object with a nil top-level error.
// Conflict detection failures enter serviceError without dropping other fields.
func (s *NezhaAgentService) Status() (*dto.NezhaAgentStatus, error) {
	st := &dto.NezhaAgentStatus{
		Conflicts: []dto.NezhaAgentConflict{},
	}

	st.ComponentAvailable = s.binaryAvailable() && s.unitAvailable()
	if st.ComponentAvailable {
		st.Version = s.binaryVersion()
	}

	s.fillConfigStatus(st)
	// Healthy config.yml is the sole source of truth: mirror server/secret into settings.
	// Store failures are diagnostic only and must not drop the rest of Status.
	if st.ConfigHealthy {
		if err := s.SyncConfigToSettings(); err != nil {
			st.ServiceError = mergeServiceDiag(st.ServiceError, err.Error())
		}
	}
	st.PermissionsWarning = s.permissionsWarning()

	state, err := s.probeActiveState()
	if err != nil {
		st.ServiceError = mergeServiceDiag(st.ServiceError, err.Error())
	}
	if state != "" {
		st.ServiceState = state
		st.Active = state == "active"
	}

	enabled, err := s.probeEnabledState()
	if err != nil {
		st.ServiceError = mergeServiceDiag(st.ServiceError, err.Error())
	} else {
		st.Enabled = enabled
	}

	st.DesiredEnabled = s.desiredEnabled()
	st.Drift = st.Enabled != st.DesiredEnabled

	conflicts, err := s.detectConflicts()
	if err != nil {
		st.ServiceError = mergeServiceDiag(st.ServiceError, "conflict detection failed: "+err.Error())
	} else if len(conflicts) > 0 {
		st.Conflicts = conflicts
	}
	if !st.ComponentAvailable {
		st.ServiceError = filterMissingUnitDiagnostics(st.ServiceError)
	}
	return st, nil
}

// mergeServiceDiag appends a diagnostic without overwriting earlier ones.
func mergeServiceDiag(existing, next string) string {
	next = strings.TrimSpace(next)
	if next == "" {
		return existing
	}
	existing = strings.TrimSpace(existing)
	if existing == "" {
		return next
	}
	return existing + "; " + next
}

func (s *NezhaAgentService) binaryAvailable() bool {
	info, err := os.Lstat(s.binaryPath)
	if err != nil {
		return false
	}
	if !info.Mode().IsRegular() {
		return false
	}
	// Require owner/group/other execute bit (0755 satisfies this).
	return info.Mode().Perm()&0o111 != 0
}

func (s *NezhaAgentService) unitAvailable() bool {
	info, err := os.Lstat(s.unitPath)
	return err == nil && info.Mode().IsRegular()
}

func filterMissingUnitDiagnostics(message string) string {
	parts := strings.Split(message, "; ")
	kept := parts[:0]
	for _, part := range parts {
		lower := strings.ToLower(part)
		if strings.Contains(lower, "systemctl") &&
			(strings.Contains(lower, "not-found") || strings.Contains(lower, "not found")) {
			continue
		}
		if strings.TrimSpace(part) != "" {
			kept = append(kept, part)
		}
	}
	return strings.Join(kept, "; ")
}

func (s *NezhaAgentService) binaryVersion() string {
	out, err := s.runner.CombinedOutput(s.binaryPath, "-v")
	if err != nil {
		return ""
	}
	fields := strings.Fields(strings.TrimSpace(string(out)))
	if len(fields) == 0 {
		return ""
	}
	return fields[len(fields)-1]
}

// fillConfigStatus populates config-related Status fields. Failures are recorded
// on the status object; the function never returns a hard error and never rebuilds
// config from the database.
func (s *NezhaAgentService) fillConfigStatus(st *dto.NezhaAgentStatus) {
	info, err := os.Lstat(s.configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			st.Configured = false
			st.ConfigHealthy = false
			st.ConfigError = "nezha agent config missing"
			// disable_command_execute missing defaults to remote ops allowed only
			// when a healthy config is present; leave false on missing file.
			return
		}
		st.Configured = false
		st.ConfigHealthy = false
		st.ConfigError = "stat nezha agent config failed"
		return
	}

	// Path exists (regular, symlink, or other) → configured.
	st.Configured = true

	if err := rejectUnsafeNezhaConfigFile(s.configPath, info); err != nil {
		st.ConfigHealthy = false
		// Safe diagnostic: path type issue only, never file content.
		st.ConfigError = err.Error()
		return
	}

	data, err := s.readConfig(s.configPath)
	if err != nil {
		st.ConfigHealthy = false
		// Avoid leaking raw OS/path details that might include sensitive material.
		msg := err.Error()
		if strings.Contains(msg, "symlink") || strings.Contains(msg, "regular file") {
			st.ConfigError = msg
		} else {
			st.ConfigError = "read nezha agent config failed"
		}
		return
	}
	if len(data) == 0 {
		// Empty regular file: path exists so configured, but not healthy.
		st.ConfigHealthy = false
		st.ConfigError = "nezha agent config missing"
		return
	}

	var cfg map[string]any
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		st.ConfigHealthy = false
		// Do not wrap the YAML library error: it may echo line snippets.
		st.ConfigError = "nezha agent config is not valid YAML"
		return
	}
	if cfg == nil {
		cfg = map[string]any{}
	}

	// uuid is display-only and only when it is a string.
	if u, ok := cfg["uuid"].(string); ok {
		st.UUID = u
	}
	st.TLS = nezhaYAMLBool(cfg["tls"])
	st.InsecureTLS = nezhaYAMLBool(cfg["insecure_tls"])
	// disable_command_execute missing defaults to false → remote ops enabled.
	st.RemoteOperationsEnabled = !nezhaYAMLBool(cfg["disable_command_execute"])

	server, serverOK := nezhaRequiredConfigString(cfg["server"])
	_, secretOK := nezhaRequiredConfigString(cfg["client_secret"])
	if serverOK {
		st.Server = server
		st.DashboardURL = nezhaDashboardURLFromServer(st.Server, st.TLS)
	}
	// SecretConfigured only; never copy secret material into the status struct.
	st.SecretConfigured = secretOK

	if !serverOK || !secretOK {
		st.ConfigHealthy = false
		var parts []string
		if !serverOK {
			parts = append(parts, "server must be a non-empty string")
		}
		if !secretOK {
			parts = append(parts, "client_secret must be a non-empty string")
		}
		// Never include the actual values in the diagnostic.
		st.ConfigError = strings.Join(parts, "; ")
		return
	}

	st.ConfigHealthy = true
	st.ConfigError = ""
}

// nezhaRequiredConfigString accepts only non-empty YAML strings.
// Wrong types and empty values fail without surfacing the value.
func nezhaRequiredConfigString(v any) (string, bool) {
	if v == nil {
		return "", false
	}
	s, ok := v.(string)
	if !ok {
		return "", false
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return "", false
	}
	return s, true
}

// permissionsWarning reports actual mode deviations from the expected layout:
// config directory 0700, binary 0755, existing config 0600. Never includes secrets.
func (s *NezhaAgentService) permissionsWarning() string {
	var parts []string

	dir := filepath.Dir(s.configPath)
	if dirInfo, err := os.Lstat(dir); err == nil && dirInfo.IsDir() {
		if perm := dirInfo.Mode().Perm(); perm != 0o700 {
			parts = append(parts, fmt.Sprintf("config directory mode is %04o, want 0700", perm))
		}
	}

	if binInfo, err := os.Lstat(s.binaryPath); err == nil && binInfo.Mode().IsRegular() {
		if perm := binInfo.Mode().Perm(); perm != 0o755 {
			parts = append(parts, fmt.Sprintf("binary mode is %04o, want 0755", perm))
		}
	}

	if cfgInfo, err := os.Lstat(s.configPath); err == nil && cfgInfo.Mode().IsRegular() {
		if perm := cfgInfo.Mode().Perm(); perm != 0o600 {
			parts = append(parts, fmt.Sprintf("config file mode is %04o, want 0600", perm))
		}
	}

	return strings.Join(parts, "; ")
}

func (s *NezhaAgentService) desiredEnabled() bool {
	if s.settings == nil {
		return false
	}
	val, err := s.settings.GetValueByKey(nezhaEnabledSettingKey)
	if err != nil {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(val)) {
	case "true", "enable", "enabled", "1":
		return true
	default:
		return false
	}
}

func nezhaYAMLBool(v any) bool {
	if v == nil {
		return false
	}
	switch t := v.(type) {
	case bool:
		return t
	case string:
		switch strings.ToLower(strings.TrimSpace(t)) {
		case "true", "yes", "1", "on":
			return true
		default:
			return false
		}
	case int:
		return t != 0
	case int64:
		return t != 0
	case float64:
		return t != 0
	default:
		return false
	}
}

// nezhaDashboardURLFromServer builds a panel-facing origin from Agent server host:port and tls.
// IPv6 hosts keep brackets for both default and non-default ports.
func nezhaDashboardURLFromServer(server string, tls bool) string {
	server = strings.TrimSpace(server)
	if server == "" {
		return ""
	}
	host, port, err := net.SplitHostPort(server)
	if err != nil {
		// Bare host without port — treat as-is (may already include brackets).
		scheme := "http"
		if tls {
			scheme = "https"
		}
		return scheme + "://" + server
	}
	scheme := "http"
	if tls {
		scheme = "https"
	}
	displayHost := nezhaFormatHostForURL(host)
	if (tls && port == "443") || (!tls && port == "80") {
		return scheme + "://" + displayHost
	}
	return scheme + "://" + net.JoinHostPort(host, port)
}

// nezhaFormatHostForURL brackets IPv6 literals for use in a URL origin without a port.
func nezhaFormatHostForURL(host string) string {
	if ip := net.ParseIP(host); ip != nil && ip.To4() == nil {
		return "[" + host + "]"
	}
	return host
}

// Operate accepts start|stop|restart|enable|disable only.
// start/stop/restart never touch NezhaEnabled. enable/disable write the setting
// only after a successful systemctl enable/disable --now.
// External conflicts block start/restart/enable before any bundled unit write.
// stop/disable still operate only on the bundled unit and never touch externals.
// start/restart/enable sync config.yml → settings after a successful start path.
func (s *NezhaAgentService) Operate(operation string) error {
	switch operation {
	case "start", "restart":
		if err := s.ensureNoConflicts(); err != nil {
			return err
		}
		if err := s.systemctl(operation, s.unit); err != nil {
			return err
		}
		return s.SyncConfigToSettings()
	case "stop":
		return s.systemctl("stop", s.unit)
	case "enable":
		if err := s.ensureNoConflicts(); err != nil {
			return err
		}
		if err := s.systemctl("enable", "--now", s.unit); err != nil {
			return err
		}
		if err := s.persistEnabled(true); err != nil {
			return err
		}
		return s.SyncConfigToSettings()
	case "disable":
		if err := s.systemctl("disable", "--now", s.unit); err != nil {
			return err
		}
		if err := s.persistEnabled(false); err != nil {
			return err
		}
		return nil
	default:
		return fmt.Errorf("unsupported nezha agent operation %q", operation)
	}
}

// Configure merges the panel DTO into config.yml with a safe stop/start lifecycle.
// EnableAndStart is blocked by external conflicts before any enable/start.
// When the bundled unit was active, plain configure still needs a restore start,
// so conflicts are checked before any stop/read/write.
// Plain configure on an inactive unit may still save when conflicts exist.
func (s *NezhaAgentService) Configure(req dto.NezhaAgentConfigUpdate) error {
	missing, err := s.configMissing()
	if err != nil {
		return err
	}
	if err := s.validateConfigure(req, missing); err != nil {
		return err
	}

	// Block starting a second Agent copy before any stop/read/write when requested.
	if req.EnableAndStart {
		if err := s.ensureNoConflicts(); err != nil {
			return err
		}
	}

	wasActive, err := s.isActive()
	if err != nil {
		// Unknown/empty active state must not be treated as stopped: refuse before
		// any stop/read/write so a running unit is never left half-configured.
		return err
	}
	// Record enabled for lifecycle bookkeeping (status work is a later batch).
	if _, err := s.isEnabled(); err != nil {
		return err
	}

	if wasActive {
		// Restoring a previously-running unit ends with start; refuse conflicts first.
		if err := s.ensureNoConflicts(); err != nil {
			return err
		}
		if err := s.systemctl("stop", s.unit); err != nil {
			return err
		}
		if err := s.waitUntilInactive(); err != nil {
			// Service was running; restore on wait/query failure so it is not left stopped.
			return s.restoreActive(wasActive, err)
		}
	}

	existing, err := s.readConfig(s.configPath)
	if err != nil {
		return s.restoreActive(wasActive, err)
	}

	// FirstConfig applies only when the config file is missing. Prefer the
	// post-stop on-disk observation used for the merge.
	firstConfig := missing
	if _, statErr := os.Lstat(s.configPath); statErr != nil {
		if errors.Is(statErr, os.ErrNotExist) {
			firstConfig = true
			existing = nil
		}
	} else {
		firstConfig = false
	}

	// Existing file: capture the latest on-disk server/secret (e.g. Dashboard
	// rotation) into settings before merging panel changes. Failure must not write.
	if !firstConfig {
		if err := s.SyncConfigToSettings(); err != nil {
			return s.restoreActive(wasActive, err)
		}
	}

	patch, err := s.buildConfigPatch(req, firstConfig)
	if err != nil {
		return s.restoreActive(wasActive, err)
	}

	merged, err := mergeNezhaConfig(existing, patch)
	if err != nil {
		return s.restoreActive(wasActive, err)
	}

	if err := s.writeConfig(s.configPath, merged); err != nil {
		return s.restoreActive(wasActive, err)
	}

	// Re-read the final file and refresh the DB mirror before restoring runtime.
	// Keep the new file if this sync fails; still restore the original active unit.
	if err := s.SyncConfigToSettings(); err != nil {
		return s.restoreActive(wasActive, err)
	}

	if req.EnableAndStart {
		// Re-check immediately before enable --now (hook + detector).
		if err := s.ensureNoConflicts(); err != nil {
			return err
		}
		if s.checkConflictFree != nil {
			if err := s.checkConflictFree(); err != nil {
				return err
			}
		}
		if err := s.systemctl("enable", "--now", s.unit); err != nil {
			return err
		}
		if err := s.persistEnabled(true); err != nil {
			return err
		}
		return s.SyncConfigToSettings()
	}

	if wasActive {
		// Successful replacement keeps the new file even if start fails.
		// Restarting a previously-running bundled unit is not "starting a second copy".
		if err := s.systemctl("start", s.unit); err != nil {
			return err
		}
		// After restore start, re-mirror config.yml (Agent may rotate client_secret).
		return s.SyncConfigToSettings()
	}
	return nil
}

// SyncConfigToSettings reads config.yml and atomically mirrors NezhaServer and
// NezhaClientSecret into the setting store. config.yml is the sole source of truth:
// this never rebuilds the file from DB and never reads NezhaClientSecret from DB.
// Missing config is a normal no-op. Corrupt, unsafe, incomplete, or illegal server
// values do not write DB and return a safe error without secret material.
func (s *NezhaAgentService) SyncConfigToSettings() error {
	server, secret, err := s.readNezhaSyncCredentials()
	if err != nil {
		return err
	}
	if server == "" && secret == "" {
		return nil
	}
	if s.settings == nil {
		return fmt.Errorf("sync nezha agent settings failed")
	}
	values := map[string]string{
		nezhaServerSettingKey:       server,
		nezhaClientSecretSettingKey: secret,
	}
	if err := s.settings.CreateOrUpdateMany(values); err != nil {
		// Never wrap store errors with map values (secret must not leak).
		return fmt.Errorf("sync nezha agent settings failed")
	}
	return nil
}

// readNezhaSyncCredentials loads and validates server/client_secret from config.yml.
// Missing path or empty file → ("", "", nil). Invalid cases return a safe error.
func (s *NezhaAgentService) readNezhaSyncCredentials() (server, secret string, err error) {
	info, err := os.Lstat(s.configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", "", nil
		}
		return "", "", fmt.Errorf("stat nezha agent config failed")
	}
	if err := rejectUnsafeNezhaConfigFile(s.configPath, info); err != nil {
		return "", "", err
	}

	data, err := s.readConfig(s.configPath)
	if err != nil {
		msg := err.Error()
		if strings.Contains(msg, "symlink") || strings.Contains(msg, "regular file") {
			return "", "", err
		}
		return "", "", fmt.Errorf("read nezha agent config failed")
	}
	if len(data) == 0 {
		return "", "", nil
	}

	var cfg map[string]any
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return "", "", fmt.Errorf("nezha agent config is not valid YAML")
	}
	if cfg == nil {
		cfg = map[string]any{}
	}

	rawServer, serverOK := nezhaRequiredConfigString(cfg["server"])
	rawSecret, secretOK := nezhaRequiredConfigString(cfg["client_secret"])
	if !serverOK || !secretOK {
		var parts []string
		if !serverOK {
			parts = append(parts, "server must be a non-empty string")
		}
		if !secretOK {
			parts = append(parts, "client_secret must be a non-empty string")
		}
		return "", "", fmt.Errorf("%s", strings.Join(parts, "; "))
	}

	normalized, err := normalizeNezhaServerHostPort(rawServer)
	if err != nil {
		return "", "", err
	}
	return normalized, rawSecret, nil
}

// normalizeNezhaServerHostPort canonicalizes Agent server host:port:
// host lowercased, IPv6 bracketed via net.JoinHostPort, port 1..65535 without leading zeros.
func normalizeNezhaServerHostPort(server string) (string, error) {
	server = strings.TrimSpace(server)
	if server == "" {
		return "", fmt.Errorf("nezha agent server is invalid")
	}
	host, port, err := net.SplitHostPort(server)
	if err != nil {
		return "", fmt.Errorf("nezha agent server is invalid")
	}
	host = strings.ToLower(strings.TrimSpace(host))
	if host == "" {
		return "", fmt.Errorf("nezha agent server is invalid")
	}
	n, err := strconv.Atoi(port)
	if err != nil || n < 1 || n > 65535 {
		return "", fmt.Errorf("nezha agent server port must be between 1 and 65535")
	}
	return net.JoinHostPort(host, strconv.Itoa(n)), nil
}

// ensureNoConflicts fails when an external Agent unit/process/directory is present.
// Read-only: never stop/disable/delete/overwrite external installs or import UUID.
func (s *NezhaAgentService) ensureNoConflicts() error {
	conflicts, err := s.detectConflicts()
	if err != nil {
		return fmt.Errorf("nezha agent conflict detection failed: %w", err)
	}
	if len(conflicts) == 0 {
		return nil
	}
	return fmt.Errorf("external nezha agent conflict: %s", conflicts[0].Message)
}

// detectConflicts performs read-only detection of external Nezha Agent installations.
// Results are deduplicated and stably sorted by kind then detail.
// Never stops, disables, deletes, overwrites, reads external config, or imports UUID.
func (s *NezhaAgentService) detectConflicts() ([]dto.NezhaAgentConflict, error) {
	var out []dto.NezhaAgentConflict

	units, err := s.detectUnitConflicts()
	if err != nil {
		return nil, err
	}
	out = append(out, units...)

	procs, err := s.detectProcessConflicts()
	if err != nil {
		return nil, err
	}
	out = append(out, procs...)

	dirs, err := s.detectDirectoryConflicts()
	if err != nil {
		return nil, err
	}
	out = append(out, dirs...)

	return dedupeSortNezhaConflicts(out), nil
}

// detectUnitConflicts lists active nezha-agent.service and nezha-agent@*.service
// via systemctl list-units. The bundled xpanel-nezha-agent unit is never reported.
func (s *NezhaAgentService) detectUnitConflicts() ([]dto.NezhaAgentConflict, error) {
	out, err := s.runner.CombinedOutput(
		"systemctl",
		"list-units",
		"--type=service",
		"--state=active",
		"--no-pager",
		"--no-legend",
		"--plain",
		"nezha-agent.service",
		"nezha-agent@*.service",
	)
	if err != nil {
		// Fail closed on any list-units error. Command output may contain host
		// noise and must not be treated as a successful unit list (fail-open).
		// Do not echo raw output (may leak sensitive diagnostics).
		return nil, fmt.Errorf("systemctl list-units failed: %w", err)
	}

	var conflicts []dto.NezhaAgentConflict
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		name := fields[0]
		if !isExternalNezhaAgentUnit(name) {
			continue
		}
		// When state columns exist, require active; --state=active already filters.
		if len(fields) >= 3 {
			activeState := fields[2]
			if activeState != "active" {
				continue
			}
		}
		conflicts = append(conflicts, dto.NezhaAgentConflict{
			Kind:    nezhaConflictKindUnit,
			Detail:  name,
			Message: fmt.Sprintf("external active unit %s", name),
		})
	}
	return conflicts, nil
}

func isExternalNezhaAgentUnit(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" || isXpanelNezhaAgentUnit(name) {
		return false
	}
	if name == "nezha-agent.service" || name == "nezha-agent" {
		return true
	}
	// Instantiated: nezha-agent@instance.service
	base := strings.TrimSuffix(name, ".service")
	return strings.HasPrefix(base, "nezha-agent@")
}

// detectProcessConflicts finds running executables named nezha-agent whose real
// path is not the bundled BinaryPath. nil listProcessExecutables skips the scan.
func (s *NezhaAgentService) detectProcessConflicts() ([]dto.NezhaAgentConflict, error) {
	if s.listProcessExecutables == nil {
		return nil, nil
	}
	paths, err := s.listProcessExecutables()
	if err != nil {
		return nil, err
	}
	bundled := filepath.Clean(s.binaryPath)
	var conflicts []dto.NezhaAgentConflict
	for _, p := range paths {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		// Kernel may suffix deleted binaries.
		p = strings.TrimSuffix(p, " (deleted)")
		p = filepath.Clean(p)
		if filepath.Base(p) != "nezha-agent" {
			continue
		}
		if p == bundled {
			continue
		}
		conflicts = append(conflicts, dto.NezhaAgentConflict{
			Kind:    nezhaConflictKindProcess,
			Detail:  p,
			Message: fmt.Sprintf("external nezha-agent process at %s", p),
		})
	}
	return conflicts, nil
}

// detectDirectoryConflicts reports configured external install directories that exist.
// Empty externalDirs skips the scan (test default).
// NotExist is ignored; any other Lstat error fails closed as a detection error.
func (s *NezhaAgentService) detectDirectoryConflicts() ([]dto.NezhaAgentConflict, error) {
	if len(s.externalDirs) == 0 {
		return nil, nil
	}
	lstat := s.lstat
	if lstat == nil {
		lstat = os.Lstat
	}
	var conflicts []dto.NezhaAgentConflict
	for _, dir := range s.externalDirs {
		dir = strings.TrimSpace(dir)
		if dir == "" {
			continue
		}
		if _, err := lstat(dir); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, fmt.Errorf("stat external directory %s: %w", dir, err)
		}
		conflicts = append(conflicts, dto.NezhaAgentConflict{
			Kind:    nezhaConflictKindDirectory,
			Detail:  dir,
			Message: fmt.Sprintf("external nezha agent directory %s", dir),
		})
	}
	return conflicts, nil
}

func dedupeSortNezhaConflicts(in []dto.NezhaAgentConflict) []dto.NezhaAgentConflict {
	if len(in) == 0 {
		return nil
	}
	type key struct{ kind, detail string }
	seen := make(map[key]dto.NezhaAgentConflict, len(in))
	for _, c := range in {
		k := key{c.Kind, c.Detail}
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = c
	}
	out := make([]dto.NezhaAgentConflict, 0, len(seen))
	for _, c := range seen {
		out = append(out, c)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Kind != out[j].Kind {
			return out[i].Kind < out[j].Kind
		}
		return out[i].Detail < out[j].Detail
	})
	return out
}

// listNezhaProcessExecutablesFromProc scans /proc/<pid>/exe for running process
// real paths. Missing or unreadable PIDs are skipped. Production only.
func listNezhaProcessExecutablesFromProc() ([]string, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, e := range entries {
		name := e.Name()
		if _, err := strconv.Atoi(name); err != nil {
			continue
		}
		target, err := os.Readlink(filepath.Join("/proc", name, "exe"))
		if err != nil {
			// Process exited or permission denied — ignore this PID.
			continue
		}
		target = strings.TrimSuffix(target, " (deleted)")
		if target != "" {
			paths = append(paths, target)
		}
	}
	return paths, nil
}

func (s *NezhaAgentService) persistEnabled(enabled bool) error {
	value := "false"
	if enabled {
		value = "true"
	}
	if err := s.settings.CreateOrUpdate(nezhaEnabledSettingKey, value); err != nil {
		return fmt.Errorf("systemctl succeeded but failed to persist %s=%s: %w", nezhaEnabledSettingKey, value, err)
	}
	return nil
}

func (s *NezhaAgentService) validateConfigure(req dto.NezhaAgentConfigUpdate, firstConfig bool) error {
	if firstConfig {
		if req.DashboardURL == nil || strings.TrimSpace(*req.DashboardURL) == "" {
			return fmt.Errorf("dashboard URL is required for first-time nezha agent configuration")
		}
		if _, err := normalizeNezhaDashboardOrigin(*req.DashboardURL); err != nil {
			return err
		}
		if req.ClientSecret == nil || strings.TrimSpace(*req.ClientSecret) == "" {
			return fmt.Errorf("client secret is required for first-time nezha agent configuration")
		}
		return nil
	}

	if req.DashboardURL != nil {
		if _, err := normalizeNezhaDashboardOrigin(*req.DashboardURL); err != nil {
			return err
		}
	}
	return nil
}

func (s *NezhaAgentService) buildConfigPatch(req dto.NezhaAgentConfigUpdate, firstConfig bool) (nezhaConfigPatch, error) {
	patch := nezhaConfigPatch{FirstConfig: firstConfig}
	if s.settings != nil {
		if panelName, err := s.settings.GetValueByKey("PanelName"); err == nil {
			panelName = strings.TrimSpace(panelName)
			if panelName != "" {
				patch.XPanelName = &panelName
			}
		}
	}
	if req.DashboardURL != nil {
		origin, err := normalizeNezhaDashboardOrigin(*req.DashboardURL)
		if err != nil {
			return patch, err
		}
		server := origin.Server
		tls := origin.TLS
		insecure := origin.InsecureTLS
		patch.Server = &server
		patch.TLS = &tls
		patch.InsecureTLS = &insecure
	}
	if req.ClientSecret != nil {
		// Empty secret is intentionally passed through; mergeNezhaConfig leaves the file value.
		patch.ClientSecret = req.ClientSecret
	}
	if req.RemoteOperationsEnabled != nil {
		patch.RemoteOperationsEnabled = req.RemoteOperationsEnabled
	}
	return patch, nil
}

func (s *NezhaAgentService) configMissing() (bool, error) {
	info, err := os.Lstat(s.configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return true, nil
		}
		return false, fmt.Errorf("stat nezha agent config %s: %w", s.configPath, err)
	}
	if err := rejectUnsafeNezhaConfigFile(s.configPath, info); err != nil {
		return false, err
	}
	return false, nil
}

func (s *NezhaAgentService) restoreActive(wasActive bool, cause error) error {
	if !wasActive {
		return cause
	}
	if err := s.systemctl("start", s.unit); err != nil {
		return fmt.Errorf("%v; restore start also failed: %w", cause, err)
	}
	return cause
}

func (s *NezhaAgentService) systemctl(args ...string) error {
	out, err := s.runner.CombinedOutput("systemctl", args...)
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("systemctl %s failed: %s", strings.Join(args, " "), msg)
	}
	return nil
}

// rawActiveState returns the trimmed systemctl is-active state.
// Empty output with a command error preserves the diagnostic; non-empty states
// (including inactive/failed) are returned even when systemctl exits non-zero.
func (s *NezhaAgentService) rawActiveState() (string, error) {
	out, err := s.runner.CombinedOutput("systemctl", "is-active", s.unit)
	state := strings.TrimSpace(string(out))
	if err != nil && state == "" {
		return "", fmt.Errorf("systemctl is-active %s failed: %w", s.unit, err)
	}
	return state, nil
}

// probeActiveState is used by Status. Known states map normally; empty failures
// and unknown non-empty states return a diagnostic while still yielding the raw state.
func (s *NezhaAgentService) probeActiveState() (string, error) {
	state, err := s.rawActiveState()
	if err != nil {
		return state, err
	}
	if state == "" {
		return "", fmt.Errorf("systemctl is-active %s failed: empty state", s.unit)
	}
	if !nezhaKnownActiveState(state) {
		return state, fmt.Errorf("systemctl is-active %s: unexpected state %q", s.unit, state)
	}
	return state, nil
}

// probeEnabledState is used by Status. enabled/enabled-runtime → true;
// disabled/static/masked → false; empty failures and unknown states → error.
func (s *NezhaAgentService) probeEnabledState() (bool, error) {
	out, err := s.runner.CombinedOutput("systemctl", "is-enabled", s.unit)
	state := strings.TrimSpace(string(out))
	switch state {
	case "enabled", "enabled-runtime":
		return true, nil
	case "disabled", "static", "masked", "not-found":
		return false, nil
	case "":
		if err != nil {
			return false, fmt.Errorf("systemctl is-enabled %s failed: %w", s.unit, err)
		}
		return false, fmt.Errorf("systemctl is-enabled %s failed: empty state", s.unit)
	default:
		// Other non-empty values are unexpected for Status diagnostics.
		return false, fmt.Errorf("systemctl is-enabled %s: unexpected state %q", s.unit, state)
	}
}

func nezhaKnownActiveState(state string) bool {
	switch state {
	case "active", "inactive", "failed", "activating", "deactivating", "reloading":
		return true
	default:
		return false
	}
}

// nezhaUnitRunning reports whether state is a transitional or active run state
// that needs stop/restore during Configure.
func nezhaUnitRunning(state string) bool {
	switch state {
	case "active", "activating", "reloading", "deactivating":
		return true
	default:
		return false
	}
}

// nezhaUnitStopped reports whether state is a definitive stopped terminal state.
// waitUntilInactive only finishes on these; unknown states keep waiting.
func nezhaUnitStopped(state string) bool {
	switch state {
	case "inactive", "failed":
		return true
	default:
		return false
	}
}

// isActive reports whether the unit is in a running state that Configure must
// stop and later restore. Known stopped states (inactive/failed) return false.
// Empty or unknown states return an error so Configure does not treat them as stopped.
func (s *NezhaAgentService) isActive() (bool, error) {
	state, err := s.rawActiveState()
	if err != nil {
		return false, err
	}
	if state == "" {
		return false, fmt.Errorf("systemctl is-active %s failed: empty state", s.unit)
	}
	if !nezhaKnownActiveState(state) {
		return false, fmt.Errorf("systemctl is-active %s: unexpected state %q", s.unit, state)
	}
	return nezhaUnitRunning(state), nil
}

// isEnabled reports enabled state. disabled and other non-enabled results are normal.
func (s *NezhaAgentService) isEnabled() (bool, error) {
	out, err := s.runner.CombinedOutput("systemctl", "is-enabled", s.unit)
	state := strings.TrimSpace(string(out))
	if state == "enabled" || state == "enabled-runtime" {
		return true, nil
	}
	if err != nil && state == "" {
		return false, fmt.Errorf("systemctl is-enabled %s failed: %w", s.unit, err)
	}
	return false, nil
}

func (s *NezhaAgentService) waitUntilInactive() error {
	deadline := s.now().Add(s.stopWait)
	for {
		state, err := s.rawActiveState()
		if err != nil {
			return err
		}
		if nezhaUnitStopped(state) {
			return nil
		}
		if !s.now().Before(deadline) {
			return fmt.Errorf("timed out waiting for %s to become inactive", s.unit)
		}
		s.sleep(s.pollEvery)
	}
}

func isXpanelNezhaAgentUnit(name string) bool {
	return strings.TrimSuffix(name, ".service") == NezhaAgentUnitName
}

const nezhaJournalRedacted = "***"

// Patterns for secondary journal redaction. Case-insensitive; cover equals,
// colon, space-separated assignments, and JSON-style quoted keys/values.
// Separator is required (=, :, or at least one whitespace) to avoid zero-separator
// false matches on ordinary words like "secretly".
var (
	nezhaSecretAssignmentPattern = regexp.MustCompile(
		`(?i)("?\b(?:client[_-]?secret|agent[_-]?secret|secret)\b"?)(?:\s*[:=]\s*|\s+)(?:"[^"]*"|'[^']*'|[^\s,;]+)`,
	)
	nezhaAuthorizationPattern = regexp.MustCompile(
		`(?i)\bAuthorization\s*[:=]\s*(?:Bearer\s+)?[^\s,;]+`,
	)
	nezhaBearerPattern = regexp.MustCompile(
		`(?i)\bBearer\s+[A-Za-z0-9._~+/=-]+`,
	)
)

// readNezhaAgentSecretFromConfig safely reads client_secret from config.yml.
// Missing, corrupt, symlink, or non-regular files return "". Never reads the DB
// and never surfaces secret material in errors (callers only receive "" or the value).
func readNezhaAgentSecretFromConfig(path string) string {
	data, err := readNezhaConfigFile(path)
	if err != nil || len(data) == 0 {
		return ""
	}
	var cfg map[string]any
	if err := yaml.Unmarshal(data, &cfg); err != nil || cfg == nil {
		return ""
	}
	raw, ok := cfg["client_secret"]
	if !ok || raw == nil {
		return ""
	}
	secret, ok := raw.(string)
	if !ok {
		return ""
	}
	return secret
}

// redactNezhaJournal masks known AgentSecret occurrences and common auth metadata
// (client_secret / agent-secret / secret assignments, Authorization, Bearer).
func redactNezhaJournal(text, knownSecret string) string {
	if knownSecret != "" {
		text = strings.ReplaceAll(text, knownSecret, nezhaJournalRedacted)
	}
	text = nezhaSecretAssignmentPattern.ReplaceAllString(text, `${1}=`+nezhaJournalRedacted)
	text = nezhaAuthorizationPattern.ReplaceAllString(text, "Authorization: "+nezhaJournalRedacted)
	text = nezhaBearerPattern.ReplaceAllString(text, "Bearer "+nezhaJournalRedacted)
	return text
}
