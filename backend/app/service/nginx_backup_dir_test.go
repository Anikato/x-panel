package service

import (
	"path/filepath"
	"testing"

	"xpanel/global"
)

func TestConfigBackupDirRequiresAbsoluteDataDir(t *testing.T) {
	previous := global.CONF.System.DataDir
	t.Cleanup(func() { global.CONF.System.DataDir = previous })
	svc := &WebsiteService{}

	global.CONF.System.DataDir = ""
	if _, err := svc.configBackupDir("/etc/nginx/nginx.conf"); err == nil {
		t.Fatal("empty data dir was accepted")
	}
	global.CONF.System.DataDir = "relative/data"
	if _, err := svc.configBackupDir("/etc/nginx/nginx.conf"); err == nil {
		t.Fatal("relative data dir was accepted")
	}
	global.CONF.System.DataDir = t.TempDir()
	dir, err := svc.configBackupDir("/etc/nginx/nginx.conf")
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(dir) || filepath.Dir(filepath.Dir(dir)) != global.CONF.System.DataDir {
		t.Fatalf("backup dir = %s", dir)
	}
}
