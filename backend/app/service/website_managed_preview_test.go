package service

import (
	"os"
	"path/filepath"
	"testing"

	"xpanel/app/dto"
	"xpanel/app/model"
	"xpanel/global"
)

func TestManagedWebsiteConfigReadsDiskAndRejectsEditing(t *testing.T) {
	installCertificateConsumerDatabase(t)
	installFakeNginx(t, false)

	site := model.Website{PrimaryDomain: "managed.example.com", Alias: "managed", ConfigMode: "managed"}
	if err := global.DB.Create(&site).Error; err != nil {
		t.Fatal(err)
	}
	service := NewIWebsiteService()
	preview, err := service.GetSiteConfContent(site.ID)
	if err != nil || !preview.Generated || preview.Content == "" {
		t.Fatalf("expected generated fallback for missing config: %#v %v", preview, err)
	}
	path := GetSiteConfPath(site.Alias)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	const diskConfig = "server { listen 80; server_name managed.example.com; }\n"
	if err := os.WriteFile(path, []byte(diskConfig), 0644); err != nil {
		t.Fatal(err)
	}

	preview, err = service.GetSiteConfContent(site.ID)
	if err != nil || preview.Generated || preview.Content != diskConfig || preview.Path != path {
		t.Fatalf("expected the actual file: %#v %v", preview, err)
	}
	if err := service.SaveSiteConfContent(dto.SaveSiteConfReq{
		ID: site.ID, Hash: preview.Hash, Content: "server { listen 81; }",
	}); err == nil {
		t.Fatal("managed config accepted raw changes")
	}
	current, err := os.ReadFile(path)
	if err != nil || string(current) != diskConfig {
		t.Fatalf("managed config changed: %v", err)
	}
}
