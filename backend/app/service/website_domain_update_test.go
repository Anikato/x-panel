package service

import (
	"os"
	"strings"
	"testing"

	"xpanel/app/dto"
	"xpanel/app/model"
	"xpanel/app/repo"
	"xpanel/global"
)

func setupManagedWebsiteNginx(t *testing.T) {
	t.Helper()
	openWebsiteExternalTestDB(t)
	installFakeNginx(t, true)
	global.CONF.System.DataDir = t.TempDir()
	if err := os.MkdirAll(global.CONF.Nginx.GetSitesDir(), 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestUpdatePrimaryDomainKeepsAliasAndDoesNotLeaveOldConfig(t *testing.T) {
	setupManagedWebsiteNginx(t)

	site := model.Website{
		PrimaryDomain: "old.example.com",
		Alias:         "old_example_com",
		Type:          "static",
		Status:        "running",
		SiteDir:       "/var/www/old",
		IndexFile:     "index.html",
	}
	if err := repo.NewIWebsiteRepo().Create(&site); err != nil {
		t.Fatal(err)
	}
	oldPath := GetSiteConfPath(site.Alias)
	if err := os.WriteFile(oldPath, []byte("server { server_name old.example.com; }\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	svc := &WebsiteService{websiteRepo: repo.NewIWebsiteRepo(), certRepo: repo.NewICertificateRepo()}
	err := svc.Update(dto.WebsiteUpdate{
		ID:            site.ID,
		PrimaryDomain: "new.example.com",
		SiteDir:       site.SiteDir,
		IndexFile:     site.IndexFile,
	})
	if err != nil {
		t.Fatal(err)
	}

	stored, err := repo.NewIWebsiteRepo().Get(repo.WithByID(site.ID))
	if err != nil {
		t.Fatal(err)
	}
	if stored.Alias != "old_example_com" {
		t.Fatalf("alias changed to %q", stored.Alias)
	}
	if stored.PrimaryDomain != "new.example.com" {
		t.Fatalf("domain = %q", stored.PrimaryDomain)
	}

	newPath := GetSiteConfPath("new_example_com")
	if _, statErr := os.Stat(newPath); !os.IsNotExist(statErr) {
		t.Fatalf("new alias config should not be created: %v", statErr)
	}
	content, readErr := os.ReadFile(oldPath)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !strings.Contains(string(content), "new.example.com") {
		t.Fatalf("updated config missing new domain: %s", content)
	}
}

func TestUpdatePrimaryDomainPreservesCustomAlias(t *testing.T) {
	setupManagedWebsiteNginx(t)

	site := model.Website{
		PrimaryDomain: "old.example.com",
		Alias:         "custom-site",
		Type:          "static",
		Status:        "running",
		SiteDir:       "/var/www/custom",
		IndexFile:     "index.html",
	}
	if err := repo.NewIWebsiteRepo().Create(&site); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(GetSiteConfPath(site.Alias), []byte("server { server_name old.example.com; }\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	svc := &WebsiteService{websiteRepo: repo.NewIWebsiteRepo(), certRepo: repo.NewICertificateRepo()}
	if err := svc.Update(dto.WebsiteUpdate{
		ID:            site.ID,
		PrimaryDomain: "new.example.com",
		SiteDir:       site.SiteDir,
		IndexFile:     site.IndexFile,
	}); err != nil {
		t.Fatal(err)
	}

	stored, err := repo.NewIWebsiteRepo().Get(repo.WithByID(site.ID))
	if err != nil {
		t.Fatal(err)
	}
	if stored.Alias != "custom-site" {
		t.Fatalf("custom alias changed to %q", stored.Alias)
	}
}

func TestDeleteAfterDomainChangeCleansManagedConfig(t *testing.T) {
	setupManagedWebsiteNginx(t)

	site := model.Website{
		PrimaryDomain: "old.example.com",
		Alias:         "old_example_com",
		Type:          "static",
		Status:        "running",
		SiteDir:       "/var/www/old",
		IndexFile:     "index.html",
	}
	if err := repo.NewIWebsiteRepo().Create(&site); err != nil {
		t.Fatal(err)
	}
	confPath := GetSiteConfPath(site.Alias)
	if err := os.WriteFile(confPath, []byte("server {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	svc := &WebsiteService{websiteRepo: repo.NewIWebsiteRepo(), certRepo: repo.NewICertificateRepo()}
	if err := svc.Update(dto.WebsiteUpdate{
		ID:            site.ID,
		PrimaryDomain: "new.example.com",
		SiteDir:       site.SiteDir,
		IndexFile:     site.IndexFile,
	}); err != nil {
		t.Fatal(err)
	}
	if err := svc.Delete(site.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(confPath); !os.IsNotExist(err) {
		t.Fatalf("managed config not cleaned: %v", err)
	}
	if _, err := os.Stat(GetSiteConfPath("new_example_com")); !os.IsNotExist(err) {
		t.Fatalf("new alias config leftover: %v", err)
	}
}

func TestUpdateDoesNotTouchExternalSiteFiles(t *testing.T) {
	confPath, websiteService := setupExternalWebsiteFixture(t, `server {
    listen 8080;
    server_name old.example.com;
}
`)
	original, err := os.ReadFile(confPath)
	if err != nil {
		t.Fatal(err)
	}
	site := model.Website{
		PrimaryDomain: "old.example.com",
		Alias:         "external-old",
		ConfigMode:    "source",
		NginxConfPath: confPath,
		Status:        "running",
	}
	if err := repo.NewIWebsiteRepo().Create(&site); err != nil {
		t.Fatal(err)
	}
	err = websiteService.Update(dto.WebsiteUpdate{ID: site.ID, PrimaryDomain: "new.example.com"})
	if err == nil {
		t.Fatal("expected external site update to be denied")
	}
	got, readErr := os.ReadFile(confPath)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(got) != string(original) {
		t.Fatalf("external config mutated")
	}
}
