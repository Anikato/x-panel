package service

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func ptrString(v string) *string { return &v }
func ptrBool(v bool) *bool       { return &v }

func assertYAMLValue(t *testing.T, data []byte, key string, want any) {
	t.Helper()
	var m map[string]any
	if err := yaml.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal yaml: %v", err)
	}
	got, ok := m[key]
	if !ok {
		t.Fatalf("missing key %q in %s", key, string(data))
	}
	switch want := want.(type) {
	case string:
		if got != want {
			t.Fatalf("%s = %#v, want %q", key, got, want)
		}
	case bool:
		if got != want {
			t.Fatalf("%s = %#v, want %v", key, got, want)
		}
	default:
		t.Fatalf("unsupported want type %T", want)
	}
}

func assertYAMLMissing(t *testing.T, data []byte, key string) {
	t.Helper()
	var m map[string]any
	if err := yaml.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal yaml: %v", err)
	}
	if _, ok := m[key]; ok {
		t.Fatalf("key %q should be absent in %s", key, string(data))
	}
}

func yamlMap(t *testing.T, data []byte) map[string]any {
	t.Helper()
	var m map[string]any
	if err := yaml.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal yaml: %v", err)
	}
	return m
}

func TestNormalizeNezhaDashboardOrigin(t *testing.T) {
	got, err := normalizeNezhaDashboardOrigin("https://dashboard.example.com")
	if err != nil {
		t.Fatalf("normalize default origin: %v", err)
	}
	if got.Origin != "https://dashboard.example.com" {
		t.Fatalf("origin = %q, want https://dashboard.example.com", got.Origin)
	}
	if got.Server != "dashboard.example.com:443" || !got.TLS || got.InsecureTLS {
		t.Fatalf("normalize = %#v", got)
	}
}

func TestNormalizeNezhaDashboardOriginExplicitPortAndCase(t *testing.T) {
	got, err := normalizeNezhaDashboardOrigin("https://Dashboard.Example.COM:8443/")
	if err != nil {
		t.Fatalf("normalize explicit port: %v", err)
	}
	if got.Origin != "https://dashboard.example.com:8443" {
		t.Fatalf("origin = %q", got.Origin)
	}
	if got.Server != "dashboard.example.com:8443" || !got.TLS || got.InsecureTLS {
		t.Fatalf("normalize = %#v", got)
	}
}

func TestNormalizeNezhaDashboardOriginIPv6(t *testing.T) {
	got, err := normalizeNezhaDashboardOrigin("https://[2001:DB8::1]:9443/")
	if err != nil {
		t.Fatalf("normalize IPv6: %v", err)
	}
	if got.Origin != "https://[2001:db8::1]:9443" {
		t.Fatalf("origin = %q", got.Origin)
	}
	if got.Server != "[2001:db8::1]:9443" || !got.TLS || got.InsecureTLS {
		t.Fatalf("normalize = %#v", got)
	}

	got, err = normalizeNezhaDashboardOrigin("https://[2001:db8::2]")
	if err != nil {
		t.Fatalf("normalize IPv6 default port: %v", err)
	}
	if got.Origin != "https://[2001:db8::2]" {
		t.Fatalf("origin = %q", got.Origin)
	}
	if got.Server != "[2001:db8::2]:443" || !got.TLS || got.InsecureTLS {
		t.Fatalf("normalize = %#v", got)
	}
}

func TestNormalizeNezhaDashboardOriginRejectsInvalid(t *testing.T) {
	for _, input := range []string{
		"",
		"http://dashboard.example.com",
		"https://user:pass@dashboard.example.com",
		"https://dashboard.example.com/path",
		"https://dashboard.example.com?q=1",
		"https://dashboard.example.com#frag",
		"https://",
		"https:///path",
		"https://dashboard.example.com:0",
		"https://dashboard.example.com:65536",
		"https://dashboard.example.com:abc",
		"ftp://dashboard.example.com",
		"dashboard.example.com",
		"//dashboard.example.com",
	} {
		if _, err := normalizeNezhaDashboardOrigin(input); err == nil {
			t.Fatalf("expected rejection for %q", input)
		}
	}
}

func TestNormalizeNezhaDashboardOriginTrimsWhitespace(t *testing.T) {
	got, err := normalizeNezhaDashboardOrigin("  https://dashboard.example.com/  ")
	if err != nil {
		t.Fatalf("normalize trimmed origin: %v", err)
	}
	if got.Server != "dashboard.example.com:443" {
		t.Fatalf("server = %q", got.Server)
	}
	if strings.Contains(got.Origin, " ") {
		t.Fatalf("origin retains whitespace: %q", got.Origin)
	}
}

func TestMergeNezhaConfigPreservesRotatedSecretUUIDAndUnknownFields(t *testing.T) {
	input := []byte("server: old:443\nclient_secret: rotated\nuuid: node-uuid\ncustom_ip_api:\n  - https://ip.example\nunknown_map:\n  nested: true\nextra_flag: keep-me\n")
	updated, err := mergeNezhaConfig(input, nezhaConfigPatch{
		Server:      ptrString("new:443"),
		TLS:         ptrBool(true),
		InsecureTLS: ptrBool(false),
	})
	if err != nil {
		t.Fatal(err)
	}
	assertYAMLValue(t, updated, "client_secret", "rotated")
	assertYAMLValue(t, updated, "uuid", "node-uuid")
	assertYAMLValue(t, updated, "server", "new:443")
	assertYAMLValue(t, updated, "tls", true)
	assertYAMLValue(t, updated, "insecure_tls", false)
	assertYAMLValue(t, updated, "extra_flag", "keep-me")

	m := yamlMap(t, updated)
	list, ok := m["custom_ip_api"].([]any)
	if !ok || len(list) != 1 || list[0] != "https://ip.example" {
		t.Fatalf("custom_ip_api = %#v", m["custom_ip_api"])
	}
	nested, ok := m["unknown_map"].(map[string]any)
	if !ok || nested["nested"] != true {
		t.Fatalf("unknown_map = %#v", m["unknown_map"])
	}
}

func TestMergeNezhaConfigEmptySecretLeavesExisting(t *testing.T) {
	input := []byte("server: old:443\nclient_secret: keep-secret\n")
	updated, err := mergeNezhaConfig(input, nezhaConfigPatch{
		ClientSecret: ptrString(""),
		Server:       ptrString("new:443"),
	})
	if err != nil {
		t.Fatal(err)
	}
	assertYAMLValue(t, updated, "client_secret", "keep-secret")
	assertYAMLValue(t, updated, "server", "new:443")

	updated, err = mergeNezhaConfig(input, nezhaConfigPatch{Server: ptrString("other:443")})
	if err != nil {
		t.Fatal(err)
	}
	assertYAMLValue(t, updated, "client_secret", "keep-secret")
}

func TestMergeNezhaConfigRemoteOperationsMapsToDisableCommandExecute(t *testing.T) {
	input := []byte("server: dash:443\nclient_secret: secret\ndisable_command_execute: false\n")
	updated, err := mergeNezhaConfig(input, nezhaConfigPatch{
		RemoteOperationsEnabled: ptrBool(false),
	})
	if err != nil {
		t.Fatal(err)
	}
	assertYAMLValue(t, updated, "disable_command_execute", true)

	updated, err = mergeNezhaConfig(updated, nezhaConfigPatch{
		RemoteOperationsEnabled: ptrBool(true),
	})
	if err != nil {
		t.Fatal(err)
	}
	assertYAMLValue(t, updated, "disable_command_execute", false)
}

func TestMergeNezhaConfigFirstExplicitWritesDefaultsWithoutUUID(t *testing.T) {
	updated, err := mergeNezhaConfig(nil, nezhaConfigPatch{
		Server:       ptrString("dashboard.example.com:443"),
		ClientSecret: ptrString("initial-secret"),
		TLS:          ptrBool(true),
		InsecureTLS:  ptrBool(false),
		FirstConfig:  true,
	})
	if err != nil {
		t.Fatal(err)
	}
	assertYAMLValue(t, updated, "server", "dashboard.example.com:443")
	assertYAMLValue(t, updated, "client_secret", "initial-secret")
	assertYAMLValue(t, updated, "tls", true)
	assertYAMLValue(t, updated, "insecure_tls", false)
	assertYAMLValue(t, updated, "disable_auto_update", true)
	assertYAMLValue(t, updated, "disable_force_update", true)
	assertYAMLValue(t, updated, "disable_command_execute", false)
	assertYAMLMissing(t, updated, "uuid")
}

// FirstConfig only seeds initial disable_* defaults; it must never strip an
// existing Agent uuid or unknown keys already present in the YAML.
func TestMergeNezhaConfigFirstConfigPreservesExistingUUIDAndUnknownFields(t *testing.T) {
	input := []byte("server: old:443\nclient_secret: rotated\nuuid: existing-agent-uuid\nextra_flag: keep-me\ncustom_ip_api:\n  - https://ip.example\n")
	updated, err := mergeNezhaConfig(input, nezhaConfigPatch{
		FirstConfig: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	assertYAMLValue(t, updated, "uuid", "existing-agent-uuid")
	assertYAMLValue(t, updated, "extra_flag", "keep-me")
	assertYAMLValue(t, updated, "client_secret", "rotated")
	assertYAMLValue(t, updated, "server", "old:443")
	assertYAMLValue(t, updated, "disable_auto_update", true)
	assertYAMLValue(t, updated, "disable_force_update", true)
	assertYAMLValue(t, updated, "disable_command_execute", false)

	m := yamlMap(t, updated)
	list, ok := m["custom_ip_api"].([]any)
	if !ok || len(list) != 1 || list[0] != "https://ip.example" {
		t.Fatalf("custom_ip_api = %#v", m["custom_ip_api"])
	}
}

func TestMergeNezhaConfigRejectsCorruptYAML(t *testing.T) {
	_, err := mergeNezhaConfig([]byte("server: [unterminated"), nezhaConfigPatch{
		Server: ptrString("new:443"),
	})
	if err == nil {
		t.Fatal("corrupt YAML must be rejected")
	}
}

func TestMergeNezhaConfigLaterSaveDoesNotForceUpdateDefaults(t *testing.T) {
	input := []byte("server: old:443\nclient_secret: secret\ndisable_auto_update: false\ndisable_force_update: false\ndisable_command_execute: true\n")
	updated, err := mergeNezhaConfig(input, nezhaConfigPatch{
		Server:      ptrString("new:443"),
		TLS:         ptrBool(true),
		InsecureTLS: ptrBool(false),
	})
	if err != nil {
		t.Fatal(err)
	}
	assertYAMLValue(t, updated, "disable_auto_update", false)
	assertYAMLValue(t, updated, "disable_force_update", false)
	assertYAMLValue(t, updated, "disable_command_execute", true)
}

func TestReadNezhaConfigFileMissingIsAllowed(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yml")
	data, err := readNezhaConfigFile(path)
	if err != nil {
		t.Fatalf("missing file should be readable as empty: %v", err)
	}
	if data != nil {
		t.Fatalf("missing file data = %q, want nil", data)
	}
}

func TestReadNezhaConfigFileRejectsSymlinkAndNonRegular(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target.yml")
	if err := os.WriteFile(target, []byte("server: x:443\n"), 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "config.yml")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if _, err := readNezhaConfigFile(link); err == nil {
		t.Fatal("symlink must be rejected")
	}

	subdir := filepath.Join(dir, "not-a-file")
	if err := os.Mkdir(subdir, 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := readNezhaConfigFile(subdir); err == nil {
		t.Fatal("directory must be rejected")
	}
}

func TestWriteNezhaConfigFileCreatesWithSecureModes(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "agent")
	path := filepath.Join(dir, "config.yml")
	payload := []byte("server: dash:443\nclient_secret: secret\n")
	if err := writeNezhaConfigFile(path, payload); err != nil {
		t.Fatal(err)
	}
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !info.Mode().IsRegular() {
		t.Fatalf("config is not regular: %v", info.Mode())
	}
	if got := info.Mode().Perm(); got != 0600 {
		t.Fatalf("file mode = %04o, want 0600", got)
	}
	dirInfo, err := os.Lstat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := dirInfo.Mode().Perm(); got != 0700 {
		t.Fatalf("dir mode = %04o, want 0700", got)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(payload) {
		t.Fatalf("content = %q", got)
	}
}

func TestWriteNezhaConfigFileRepairsPermissiveMode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yml")
	original := []byte("server: old:443\nclient_secret: old-secret\n")
	if err := os.WriteFile(path, original, 0666); err != nil {
		t.Fatal(err)
	}
	// Still readable even when mode is too open.
	data, err := readNezhaConfigFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(original) {
		t.Fatalf("read permissive file = %q", data)
	}

	updated := []byte("server: new:443\nclient_secret: new-secret\n")
	if err := writeNezhaConfigFile(path, updated); err != nil {
		t.Fatal(err)
	}
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0600 {
		t.Fatalf("repaired mode = %04o, want 0600", got)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(updated) {
		t.Fatalf("content after repair write = %q", got)
	}
}

func TestWriteNezhaConfigFileRejectsSymlinkTarget(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target.yml")
	if err := os.WriteFile(target, []byte("server: x:443\n"), 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "config.yml")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if err := writeNezhaConfigFile(link, []byte("server: y:443\n")); err == nil {
		t.Fatal("writing through symlink must fail")
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "server: x:443\n" {
		t.Fatalf("symlink target mutated: %q", got)
	}
}

func TestWriteNezhaConfigFileFailureLeavesOriginalIntact(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yml")
	original := []byte("server: original:443\nclient_secret: keep-me\n")
	if err := os.WriteFile(path, original, 0600); err != nil {
		t.Fatal(err)
	}

	nezhaConfigWriteFault = func(stage string) error {
		if stage == "before-rename" {
			return errors.New("injected rename failure")
		}
		return nil
	}
	t.Cleanup(func() { nezhaConfigWriteFault = nil })

	err := writeNezhaConfigFile(path, []byte("server: destroyed:443\nclient_secret: gone\n"))
	if err == nil {
		t.Fatal("injected write failure must be returned")
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(original) {
		t.Fatalf("original content changed after failed write: %q", got)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.Name() == "config.yml" {
			continue
		}
		if strings.Contains(entry.Name(), "tmp") || strings.HasPrefix(entry.Name(), ".config-") {
			t.Fatalf("leftover temp entry after failed write: %s", entry.Name())
		}
	}
}

func TestWriteNezhaConfigFileDoesNotReadDBSecret(t *testing.T) {
	// Contract test: merge + write path never invents client_secret from outside the file/patch.
	input := []byte("server: old:443\nclient_secret: file-secret\nuuid: keep-uuid\n")
	updated, err := mergeNezhaConfig(input, nezhaConfigPatch{
		Server: ptrString("new:443"),
		// no ClientSecret in patch — must keep file value, never a DB value
	})
	if err != nil {
		t.Fatal(err)
	}
	assertYAMLValue(t, updated, "client_secret", "file-secret")
	assertYAMLValue(t, updated, "uuid", "keep-uuid")

	path := filepath.Join(t.TempDir(), "config.yml")
	if err := writeNezhaConfigFile(path, updated); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(got), "db-secret") {
		t.Fatal("written config must not contain a database secret value")
	}
	assertYAMLValue(t, got, "client_secret", "file-secret")
}
