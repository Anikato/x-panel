package service

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"xpanel/global"
)

func TestNginxLogDetectsActiveExternalInclude(t *testing.T) {
	root := t.TempDir()
	external := writeNginxFixture(t, filepath.Join(root, "outside", "site.conf"), `server {
    server_name logs.example.com;
    access_log /data/site/logs/access.log combined;
    error_log /data/site/logs/error.log warn;
}`)
	writeNginxFixture(t, filepath.Join(root, "conf", "nginx.conf"), "include "+external+";\n")
	writeNginxFixture(t, filepath.Join(root, "sbin", "nginx"), "")
	previous := global.CONF
	global.CONF.Nginx = global.NginxConfig{InstallDir: root, Mode: "prefix"}
	global.CONF.Nginx.DetectNginx()
	t.Cleanup(func() { global.CONF = previous })

	sites, err := (&NginxLogService{}).DetectSites()
	if err != nil {
		t.Fatalf("detect nginx sites: %v", err)
	}
	if len(sites) != 1 || sites[0].Name != "logs.example.com" {
		t.Fatalf("sites = %#v", sites)
	}
	if sites[0].AccessLog != "/data/site/logs/access.log" || sites[0].ErrorLog != "/data/site/logs/error.log" {
		t.Fatalf("log paths = %#v", sites[0])
	}
}

func TestNginxLogParsesCombined(t *testing.T) {
	path := filepath.Join(t.TempDir(), "access.log")
	now := time.Now().UTC()
	line := "203.0.113.10 - - [" + now.Format("02/Jan/2006:15:04:05 -0700") + `] "GET /health HTTP/1.1" 200 12 "-" "test-agent"` + "\n"
	if err := os.WriteFile(path, []byte(line), 0o644); err != nil {
		t.Fatalf("write access log: %v", err)
	}
	entries, err := parseAccessLog(path, now.Add(-time.Minute), 0)
	if err != nil {
		t.Fatalf("parse access log: %v", err)
	}
	if len(entries) != 1 || entries[0].Status != 200 || entries[0].URL != "/health" {
		t.Fatalf("entries = %#v", entries)
	}
}
