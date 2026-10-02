package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"xpanel/global"
)

func TestEnsureAccessLogFormatDefinesFormatBeforeSiteIncludes(t *testing.T) {
	confPath := withPrefixNginxConf(t, debianStyleNginxConf(""))
	if err := EnsureAccessLogFormat(); err != nil {
		t.Fatalf("EnsureAccessLogFormat: %v", err)
	}
	assertXpanelLogFormatBeforeSites(t, readTestFile(t, confPath))
}

func TestEnsureAccessLogFormatMovesFormatThatFollowsSiteInclude(t *testing.T) {
	confPath := withPrefixNginxConf(t, debianStyleNginxConf("\n    "+xpanelLogFormat+"\n"))
	if err := EnsureAccessLogFormat(); err != nil {
		t.Fatalf("EnsureAccessLogFormat: %v", err)
	}
	assertXpanelLogFormatBeforeSites(t, readTestFile(t, confPath))
}

func TestEnsureAccessLogFormatLeavesLeadingFormatInPlace(t *testing.T) {
	original := "events {}\nhttp {\n    " + xpanelLogFormat + "\n    include /etc/nginx/conf.d/*.conf;\n    include /etc/nginx/sites-enabled/*;\n}\n"
	confPath := withPrefixNginxConf(t, original)
	if err := EnsureAccessLogFormat(); err != nil {
		t.Fatalf("EnsureAccessLogFormat: %v", err)
	}
	if got := readTestFile(t, confPath); got != original {
		t.Fatalf("rewrote a config that was already valid:\n%s", got)
	}
}

func debianStyleNginxConf(beforeClose string) string {
	return `user www-data;
events {
    worker_connections 768;
}
http {
    include /etc/nginx/mime.types;
    access_log /var/log/nginx/access.log;

    include /etc/nginx/conf.d/*.conf;
    include /etc/nginx/sites-enabled/*;
` + beforeClose + `}
`
}

func withPrefixNginxConf(t *testing.T, conf string) string {
	t.Helper()
	root := t.TempDir()
	writeNginxFixture(t, filepath.Join(root, "sbin", "nginx"), "")
	confPath := writeNginxFixture(t, filepath.Join(root, "conf", "nginx.conf"), conf)
	prev := global.CONF.Nginx
	global.CONF.Nginx = global.NginxConfig{InstallDir: root, Mode: "prefix"}
	global.CONF.Nginx.DetectNginx()
	if !global.CONF.Nginx.IsInstalled() {
		t.Fatal("fixture nginx is not installed")
	}
	t.Cleanup(func() { global.CONF.Nginx = prev })
	return confPath
}

func readTestFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

func assertXpanelLogFormatBeforeSites(t *testing.T, content string) {
	t.Helper()
	if strings.Count(content, "log_format xpanel") != 1 {
		t.Fatalf("log_format xpanel count = %d\n%s", strings.Count(content, "log_format xpanel"), content)
	}
	formatAt := strings.Index(content, "log_format xpanel")
	for _, include := range []string{
		"include /etc/nginx/conf.d/*.conf;",
		"include /etc/nginx/sites-enabled/*;",
	} {
		includeAt := strings.Index(content, include)
		if formatAt < 0 || includeAt < 0 || formatAt > includeAt {
			t.Fatalf("log_format at %d, %s at %d\n%s", formatAt, include, includeAt, content)
		}
	}
}
