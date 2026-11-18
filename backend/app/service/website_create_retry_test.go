package service

import (
	"os"
	"path/filepath"
	"testing"

	"xpanel/app/dto"
	"xpanel/app/repo"
	"xpanel/global"
)

func TestWebsiteCreateCanRetryAfterDirectoryFailure(t *testing.T) {
	setupManagedWebsiteNginx(t)
	dir := filepath.Join(t.TempDir(), "site")
	if err := os.WriteFile(dir, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	req := dto.WebsiteCreate{PrimaryDomain: "retry.example", Type: "static", SiteDir: dir}
	svc := NewIWebsiteService()
	if _, err := svc.Create(req); err == nil {
		t.Fatal("accepted occupied directory")
	}
	if count, err := repo.NewIWebsiteRepo().Count(); err != nil || count != 0 {
		t.Fatalf("left rows: %d %v", count, err)
	}
	if data, err := os.ReadFile(dir); err != nil || string(data) != "keep" {
		t.Fatal("existing file changed")
	}
	if err := os.Remove(dir); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(dir, 0750); err != nil {
		t.Fatal(err)
	}
	index := filepath.Join(dir, "index.html")
	if err := os.WriteFile(index, []byte("my page"), 0600); err != nil {
		t.Fatal(err)
	}
	result, err := svc.Create(req)
	if err != nil || result.ID == 0 {
		t.Fatalf("retry failed: %v", err)
	}
	if data, err := os.ReadFile(index); err != nil || string(data) != "my page" {
		t.Fatal("overwrote existing index")
	}
	if _, err := svc.Create(req); err == nil {
		t.Fatal("accepted a duplicate website")
	}
}

func TestWebsiteActivationCanRetryWithoutRecreation(t *testing.T) {
	setupManagedWebsiteNginx(t)
	root := global.CONF.Nginx.InstallDir
	conf := filepath.Join(root, "conf", "nginx.conf")
	if err := os.WriteFile(conf, []byte("events {}\nhttp {\n}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(root, "sbin", "nginx")
	original, err := os.ReadFile(binary)
	if err != nil {
		t.Fatal(err)
	}
	writeFailingNginx(t, root)
	svc := NewIWebsiteService()
	result, err := svc.Create(dto.WebsiteCreate{PrimaryDomain: "activate.example", Type: "static", SiteDir: t.TempDir()})
	if err != nil || result.Status != "stopped" || result.ActivationError == "" {
		t.Fatalf("activation result: %#v %v", result, err)
	}
	if err := os.WriteFile(binary, original, 0755); err != nil {
		t.Fatal(err)
	}
	if err := svc.Enable(result.ID); err != nil {
		t.Fatal(err)
	}
	site, err := repo.NewIWebsiteRepo().Get(repo.WithByID(result.ID))
	if err != nil || site.Status != "running" {
		t.Fatalf("retry status: %v %v", site.Status, err)
	}
}
