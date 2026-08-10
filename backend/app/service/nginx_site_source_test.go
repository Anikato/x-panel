package service

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"xpanel/app/dto"
	"xpanel/buserr"
	"xpanel/constant"
)

func TestTokenizeNginxHandlesCommentsQuotesAndEscapes(t *testing.T) {
	tokens, err := tokenizeNginx(`
# ignored
server {
    server_name "first.example.com" escaped\ name.example.com; # trailing
    root '/data/site/public files';
}
`)
	if err != nil {
		t.Fatalf("tokenize nginx: %v", err)
	}
	want := []string{
		"server", "{",
		"server_name", "first.example.com", "escaped name.example.com", ";",
		"root", "/data/site/public files", ";",
		"}",
	}
	if !slices.Equal(tokens, want) {
		t.Fatalf("tokens = %#v, want %#v", tokens, want)
	}
}

func TestParseNginxSiteMetadataMergesServerBlocks(t *testing.T) {
	input := `
server {
    listen 8080;
    server_name first.example.com alias.example.com _ $host ~^regex;
    root /data/site/example/public;
    access_log /data/site/example/log/access.log combined;
}
server {
    listen [::]:8443 ssl default_server;
    server_name alias.example.com second.example.com;
    location / { proxy_pass http://127.0.0.1:9000; }
    error_log /data/site/example/log/error.log warn;
    ssl_certificate /data/site/example/cert/fullchain.pem;
    ssl_certificate_key /data/site/example/cert/privkey.pem;
}
`
	got, err := parseNginxSiteMetadata(input)
	if err != nil {
		t.Fatalf("parse nginx site metadata: %v", err)
	}
	if got.PrimaryDomain != "first.example.com" {
		t.Fatalf("primary = %q", got.PrimaryDomain)
	}
	wantDomains := []string{"first.example.com", "alias.example.com", "second.example.com"}
	if !slices.Equal(got.Domains, wantDomains) {
		t.Fatalf("domains = %#v, want %#v", got.Domains, wantDomains)
	}
	if got.HTTPPort != 8080 || got.HTTPSPort != 8443 || !got.SSL {
		t.Fatalf("listen metadata = %#v", got)
	}
	if got.Type != "reverse_proxy" || got.Root != "/data/site/example/public" || got.ProxyPass != "http://127.0.0.1:9000" {
		t.Fatalf("site metadata = %#v", got)
	}
	if got.AccessLogPath != "/data/site/example/log/access.log" || got.ErrorLogPath != "/data/site/example/log/error.log" {
		t.Fatalf("log metadata = %#v", got)
	}
	if got.CertPath != "/data/site/example/cert/fullchain.pem" || got.KeyPath != "/data/site/example/cert/privkey.pem" {
		t.Fatalf("certificate metadata = %#v", got)
	}
}

func TestParseNginxSiteMetadataWarnsForUnsupportedValues(t *testing.T) {
	input := `server {
    listen 443 ssl;
    server_name example.com;
    root relative/root;
    access_log syslog:server=127.0.0.1 combined;
    ssl_certificate $certificate_path;
    ssl_certificate /absolute/second.pem;
    ssl_certificate_key /absolute/key.pem;
}`
	got, err := parseNginxSiteMetadata(input)
	if err != nil {
		t.Fatalf("parse nginx site metadata: %v", err)
	}
	if got.Root != "" || got.AccessLogPath != "" || got.CertPath != "/absolute/second.pem" {
		t.Fatalf("unsupported values were accepted: %#v", got)
	}
	if len(got.Warnings) < 3 {
		t.Fatalf("warnings = %#v", got.Warnings)
	}
}

func TestParseNginxSiteMetadataRejectsMissingDomain(t *testing.T) {
	_, err := parseNginxSiteMetadata(`server { listen 80; server_name _ $host ~^regex; }`)
	if err == nil || !strings.Contains(err.Error(), "server_name") {
		t.Fatalf("error = %v", err)
	}
}

func writeNginxFixture(t *testing.T, path, content string) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create fixture directory: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func requireBusinessErrorKey(t *testing.T, err error, key string) {
	t.Helper()
	var businessErr buserr.BusinessError
	if !errors.As(err, &businessErr) || businessErr.Msg != key {
		t.Fatalf("error = %#v, want business key %q", err, key)
	}
}

func TestCollectActiveNginxConfigPathsTraversesGlobsAndCycles(t *testing.T) {
	root := t.TempDir()
	mainConf := writeNginxFixture(t, filepath.Join(root, "nginx.conf"), "include sites/*.conf;\n")
	first := writeNginxFixture(t, filepath.Join(root, "sites", "first.conf"), "include nested/second.conf;\ninclude ../nested/wrong.conf;\n")
	second := writeNginxFixture(t, filepath.Join(root, "nested", "second.conf"), "include sites/first.conf;\n")
	wrong := writeNginxFixture(t, filepath.Join(root, "nested", "wrong.conf"), "# resolved only by the old containing-file-relative behavior\n")

	got, err := collectActiveNginxConfigPaths(mainConf)
	if err != nil {
		t.Fatalf("collect active nginx paths: %v", err)
	}
	for _, path := range []string{mainConf, first, second} {
		canonical, err := filepath.EvalSymlinks(path)
		if err != nil {
			t.Fatalf("canonical path: %v", err)
		}
		if _, ok := got[canonical]; !ok {
			t.Fatalf("active paths %#v do not include %q", got, canonical)
		}
	}
	wrongCanonical, err := filepath.EvalSymlinks(wrong)
	if err != nil {
		t.Fatalf("canonical wrong path: %v", err)
	}
	if _, ok := got[wrongCanonical]; ok {
		t.Fatalf("active paths %#v unexpectedly include containing-file-relative path %q", got, wrongCanonical)
	}
}

func TestBuildIncludeTreeUsesMainConfigPrefix(t *testing.T) {
	root := t.TempDir()
	mainConf := writeNginxFixture(t, filepath.Join(root, "nginx.conf"), "include sites/*.conf;\n")
	first := writeNginxFixture(t, filepath.Join(root, "sites", "first.conf"), "include nested/second.conf;\n")
	second := writeNginxFixture(t, filepath.Join(root, "nested", "second.conf"), "include sites/first.conf;\n")

	tree := (&NginxService{}).buildIncludeNode(mainConf, map[string]bool{}, 0)
	existing := make(map[string]struct{})
	var visit func(dto.NginxIncludeNode)
	visit = func(node dto.NginxIncludeNode) {
		if node.Exists {
			existing[filepath.Clean(node.Path)] = struct{}{}
		}
		for _, child := range node.Children {
			visit(child)
		}
	}
	visit(tree)

	for _, path := range []string{mainConf, first, second} {
		if _, ok := existing[filepath.Clean(path)]; !ok {
			t.Fatalf("include tree %#v does not contain %q", existing, path)
		}
	}
}

func TestValidateExternalNginxConfigPathRequiresActiveInclude(t *testing.T) {
	root := t.TempDir()
	active := writeNginxFixture(t, filepath.Join(root, "sites", "custom.conf"), "server { server_name example.com; }")
	mainConf := writeNginxFixture(t, filepath.Join(root, "nginx.conf"), "include sites/*.conf;\n")

	symlink := filepath.Join(root, "custom-link.conf")
	if err := os.Symlink(active, symlink); err != nil {
		t.Fatalf("create symlink: %v", err)
	}
	got, err := validateExternalNginxConfigPath(symlink, mainConf)
	if err != nil {
		t.Fatalf("validate active config: %v", err)
	}
	wantActive, err := filepath.EvalSymlinks(active)
	if err != nil {
		t.Fatalf("canonical active path: %v", err)
	}
	if got != wantActive {
		t.Fatalf("path = %q, want %q", got, wantActive)
	}

	inactive := writeNginxFixture(t, filepath.Join(root, "other", "inactive.conf"), "server { server_name inactive.example; }")
	_, err = validateExternalNginxConfigPath(inactive, mainConf)
	requireBusinessErrorKey(t, err, constant.ErrWebsiteExternalConfigInactive)

	_, err = validateExternalNginxConfigPath("relative.conf", mainConf)
	requireBusinessErrorKey(t, err, constant.ErrWebsiteExternalConfigInvalid)
}

func TestContentSHA256ChangesWithContent(t *testing.T) {
	first := contentSHA256([]byte("server { server_name first.example; }"))
	second := contentSHA256([]byte("server { server_name second.example; }"))
	if first == second || len(first) != 64 || len(second) != 64 {
		t.Fatalf("unexpected hashes: %q %q", first, second)
	}
}
