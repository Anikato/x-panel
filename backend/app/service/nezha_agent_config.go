package service

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// nezhaDashboardOrigin is the normalized HTTPS origin used for Agent config.
type nezhaDashboardOrigin struct {
	Origin      string
	Server      string
	TLS         bool
	InsecureTLS bool
}

// nezhaConfigPatch holds only the fields the panel explicitly intends to change.
// A nil ClientSecret, or a non-nil empty string, leaves the existing secret unchanged.
// FirstConfig applies the initial disable_* defaults and never invents a uuid.
type nezhaConfigPatch struct {
	Server                  *string
	ClientSecret            *string
	XPanelName              *string
	RemoteOperationsEnabled *bool
	TLS                     *bool
	InsecureTLS             *bool
	FirstConfig             bool
}

// normalizeNezhaDashboardOrigin accepts a pure HTTPS origin and returns the
// canonical origin plus the Agent server host:port and fixed TLS settings.
func normalizeNezhaDashboardOrigin(raw string) (nezhaDashboardOrigin, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nezhaDashboardOrigin{}, fmt.Errorf("dashboard URL is required")
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return nezhaDashboardOrigin{}, fmt.Errorf("invalid dashboard URL: %w", err)
	}
	if parsed.Scheme != "https" {
		return nezhaDashboardOrigin{}, fmt.Errorf("dashboard URL must use https")
	}
	if parsed.User != nil {
		return nezhaDashboardOrigin{}, fmt.Errorf("dashboard URL must not include userinfo")
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return nezhaDashboardOrigin{}, fmt.Errorf("dashboard URL must not include query or fragment")
	}
	if parsed.Path != "" && parsed.Path != "/" {
		return nezhaDashboardOrigin{}, fmt.Errorf("dashboard URL must be an origin without a path")
	}
	if parsed.Host == "" {
		return nezhaDashboardOrigin{}, fmt.Errorf("dashboard URL host is required")
	}

	host := parsed.Hostname()
	if host == "" {
		return nezhaDashboardOrigin{}, fmt.Errorf("dashboard URL host is required")
	}
	host = strings.ToLower(host)

	port := parsed.Port()
	if port == "" {
		port = "443"
	} else {
		n, err := strconv.Atoi(port)
		if err != nil || n < 1 || n > 65535 {
			return nezhaDashboardOrigin{}, fmt.Errorf("dashboard URL port must be between 1 and 65535")
		}
		port = strconv.Itoa(n)
	}

	server := net.JoinHostPort(host, port)
	originHost := host
	if ip := net.ParseIP(host); ip != nil && ip.To4() == nil {
		originHost = "[" + host + "]"
	}
	origin := "https://" + originHost
	if port != "443" {
		origin += ":" + port
	}

	return nezhaDashboardOrigin{
		Origin:      origin,
		Server:      server,
		TLS:         true,
		InsecureTLS: false,
	}, nil
}

// mergeNezhaConfig merges an explicit panel patch into the existing Agent YAML.
// Unknown keys, uuid, and an Agent-rotated client_secret are preserved unless
// the patch supplies a non-empty replacement secret. This function never reads
// NezhaClientSecret from the database.
func mergeNezhaConfig(input []byte, patch nezhaConfigPatch) ([]byte, error) {
	cfg := map[string]any{}
	if len(input) > 0 {
		if err := yaml.Unmarshal(input, &cfg); err != nil {
			return nil, fmt.Errorf("parse nezha agent config: %w", err)
		}
		if cfg == nil {
			cfg = map[string]any{}
		}
	}

	if patch.FirstConfig {
		// Seed initial defaults only. Never invent or strip uuid — empty input
		// simply has none; an existing Agent identity must be preserved.
		cfg["disable_auto_update"] = true
		cfg["disable_force_update"] = true
		cfg["disable_command_execute"] = false
	}

	if patch.Server != nil {
		cfg["server"] = *patch.Server
	}
	if patch.TLS != nil {
		cfg["tls"] = *patch.TLS
	}
	if patch.InsecureTLS != nil {
		cfg["insecure_tls"] = *patch.InsecureTLS
	}
	if patch.ClientSecret != nil && strings.TrimSpace(*patch.ClientSecret) != "" {
		cfg["client_secret"] = *patch.ClientSecret
	}
	if patch.XPanelName != nil && strings.TrimSpace(*patch.XPanelName) != "" {
		cfg["xpanel_name"] = strings.TrimSpace(*patch.XPanelName)
	}
	if patch.RemoteOperationsEnabled != nil {
		cfg["disable_command_execute"] = !*patch.RemoteOperationsEnabled
	}

	out, err := yaml.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("marshal nezha agent config: %w", err)
	}
	return out, nil
}

// readNezhaConfigFile loads config.yml. A missing file returns (nil, nil).
// Symlinks and non-regular files are rejected. Overly permissive modes are
// still readable so the panel can repair them on the next successful write.
func readNezhaConfigFile(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("stat nezha agent config %s: %w", path, err)
	}
	if err := rejectUnsafeNezhaConfigFile(path, info); err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read nezha agent config %s: %w", path, err)
	}
	return data, nil
}

// nezhaConfigWriteFault is an optional test hook invoked at named stages of
// writeNezhaConfigFile. Production code leaves it nil.
var nezhaConfigWriteFault func(stage string) error

// writeNezhaConfigFile atomically persists config.yml with directory mode 0700
// and file mode 0600. Failures leave any existing file content intact and
// remove temporary files. Never reconstructs secrets from the database.
func writeNezhaConfigFile(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create nezha agent config directory %s: %w", dir, err)
	}
	if err := os.Chmod(dir, 0700); err != nil {
		return fmt.Errorf("harden nezha agent config directory %s: %w", dir, err)
	}

	if info, err := os.Lstat(path); err == nil {
		if err := rejectUnsafeNezhaConfigFile(path, info); err != nil {
			return err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("stat nezha agent config %s: %w", path, err)
	}

	tmp, err := os.CreateTemp(dir, ".config-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary nezha agent config: %w", err)
	}
	tmpName := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpName)
		}
	}()

	if err := tmp.Chmod(0600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("chmod temporary nezha agent config: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temporary nezha agent config: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync temporary nezha agent config: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temporary nezha agent config: %w", err)
	}
	if err := invokeNezhaConfigWriteFault("before-rename"); err != nil {
		return err
	}

	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("replace nezha agent config %s: %w", path, err)
	}
	cleanup = false

	if err := os.Chmod(path, 0600); err != nil {
		return fmt.Errorf("chmod nezha agent config %s: %w", path, err)
	}
	if err := syncNezhaConfigDirectory(dir); err != nil {
		return err
	}
	return nil
}

func invokeNezhaConfigWriteFault(stage string) error {
	if nezhaConfigWriteFault == nil {
		return nil
	}
	return nezhaConfigWriteFault(stage)
}

func rejectUnsafeNezhaConfigFile(path string, info os.FileInfo) error {
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("nezha agent config must not be a symlink: %s", path)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("nezha agent config must be a regular file: %s", path)
	}
	return nil
}

func syncNezhaConfigDirectory(dir string) error {
	f, err := os.Open(dir)
	if err != nil {
		return fmt.Errorf("open nezha agent config directory %s: %w", dir, err)
	}
	defer f.Close()
	if err := f.Sync(); err != nil {
		return fmt.Errorf("sync nezha agent config directory %s: %w", dir, err)
	}
	return nil
}
