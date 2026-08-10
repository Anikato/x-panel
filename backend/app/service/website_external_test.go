package service

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"xpanel/app/dto"
	"xpanel/app/model"
	"xpanel/app/repo"
	"xpanel/constant"
	"xpanel/global"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func openWebsiteExternalTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "website-external.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open website test database: %v", err)
	}
	if err := db.AutoMigrate(&model.Website{}, &model.Certificate{}, &model.Setting{}); err != nil {
		t.Fatalf("migrate website: %v", err)
	}
	previous := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = previous })
	return db
}

func setupExternalWebsiteFixture(t *testing.T, content string) (string, *WebsiteService) {
	t.Helper()
	openWebsiteExternalTestDB(t)
	root := t.TempDir()
	confPath := writeNginxFixture(t, filepath.Join(root, "external", "site.conf"), content)
	writeNginxFixture(t, filepath.Join(root, "conf", "nginx.conf"), "include "+confPath+";\n")
	previousConf := global.CONF
	global.CONF.Nginx = global.NginxConfig{InstallDir: root, Mode: "prefix"}
	global.CONF.Nginx.DetectNginx()
	global.CONF.System.DataDir = filepath.Join(root, "data")
	t.Cleanup(func() { global.CONF = previousConf })
	return confPath, &WebsiteService{websiteRepo: repo.NewIWebsiteRepo(), certRepo: repo.NewICertificateRepo()}
}

func TestWebsiteExternalPathPersistsAndQueries(t *testing.T) {
	openWebsiteExternalTestDB(t)
	path := "/data/site/example/conf/site.conf"
	site := model.Website{
		PrimaryDomain: "example.com",
		Alias:         "example",
		ConfigMode:    "source",
		NginxConfPath: path,
	}
	websiteRepo := repo.NewIWebsiteRepo()
	if err := websiteRepo.Create(&site); err != nil {
		t.Fatalf("create website: %v", err)
	}

	got, err := websiteRepo.Get(repo.WithByNginxConfPath(path))
	if err != nil {
		t.Fatalf("query website: %v", err)
	}
	if got.ID != site.ID || got.NginxConfPath != path {
		t.Fatalf("unexpected website: %#v", got)
	}
}

func TestExternalWebsitePreviewAndCreateDoNotMutateConfig(t *testing.T) {
	confPath, websiteService := setupExternalWebsiteFixture(t, `server {
    listen 8080;
    listen 8443 ssl;
    server_name first.example.com alias.example.com;
    root /data/site/example/public;
    access_log /data/site/example/log/access.log combined;
    ssl_certificate /data/site/example/cert/fullchain.pem;
    ssl_certificate_key /data/site/example/cert/privkey.pem;
}`)
	before, err := os.ReadFile(confPath)
	if err != nil {
		t.Fatalf("read config before create: %v", err)
	}

	preview, err := websiteService.InspectExternalNginxSite(confPath)
	if err != nil {
		t.Fatalf("inspect external site: %v", err)
	}
	if preview.PrimaryDomain != "first.example.com" || len(preview.Domains) != 2 || preview.HTTPSPort != 8443 {
		t.Fatalf("preview = %#v", preview)
	}

	site, err := websiteService.CreateExternalNginxSite(dto.ExternalNginxSiteCreateReq{
		Path: confPath, Alias: "kept-alias", Remark: "kept remark",
	})
	if err != nil {
		t.Fatalf("create external site: %v", err)
	}
	if site.ConfigMode != "source" || site.Status != "running" || site.NginxConfPath == "" {
		t.Fatalf("created site = %#v", site)
	}
	after, err := os.ReadFile(confPath)
	if err != nil {
		t.Fatalf("read config after create: %v", err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("external create changed the source config")
	}

	_, err = websiteService.CreateExternalNginxSite(dto.ExternalNginxSiteCreateReq{Path: confPath})
	requireBusinessErrorKey(t, err, constant.ErrWebsiteExternalConfigDuplicate)
}

func TestExternalWebsiteRefreshReparsesAndPreservesUserMetadata(t *testing.T) {
	confPath, websiteService := setupExternalWebsiteFixture(t, `server {
    listen 80;
    server_name first.example.com;
    root /data/site/first;
}`)
	site, err := websiteService.CreateExternalNginxSite(dto.ExternalNginxSiteCreateReq{
		Path: confPath, Alias: "stable-alias", Remark: "stable remark",
	})
	if err != nil {
		t.Fatalf("create external site: %v", err)
	}
	writeNginxFixture(t, confPath, `server {
    listen 8081;
    server_name second.example.com www.second.example.com;
    root /data/site/second;
}`)

	refreshed, err := websiteService.RefreshExternalNginxSite(site.ID)
	if err != nil {
		t.Fatalf("refresh external site: %v", err)
	}
	if refreshed.PrimaryDomain != "second.example.com" || refreshed.Domains != "www.second.example.com" {
		t.Fatalf("refreshed domains = %#v", refreshed)
	}
	if refreshed.SiteDir != "/data/site/second" || refreshed.HttpPort != 8081 {
		t.Fatalf("refreshed metadata = %#v", refreshed)
	}
	if refreshed.Alias != "stable-alias" || refreshed.Remark != "stable remark" || refreshed.NginxConfPath == "" {
		t.Fatalf("user metadata was not preserved: %#v", refreshed)
	}
}

func TestExternalWebsiteDetailReportsCurrentIncludeState(t *testing.T) {
	confPath, websiteService := setupExternalWebsiteFixture(t, `server {
    listen 80;
    server_name state.example.com;
}`)
	site, err := websiteService.CreateExternalNginxSite(dto.ExternalNginxSiteCreateReq{Path: confPath})
	if err != nil {
		t.Fatalf("create external site: %v", err)
	}
	detail, err := websiteService.GetDetail(site.ID)
	if err != nil {
		t.Fatalf("get active detail: %v", err)
	}
	if !detail.ConfigActive || detail.NginxConfPath == "" || len(detail.ConfigIssues) != 0 {
		t.Fatalf("active detail = %#v", detail)
	}

	writeNginxFixture(t, global.CONF.Nginx.GetMainConf(), "events {}\nhttp {}\n")
	detail, err = websiteService.GetDetail(site.ID)
	if err != nil {
		t.Fatalf("get inactive detail: %v", err)
	}
	if detail.ConfigActive || len(detail.ConfigIssues) == 0 {
		t.Fatalf("inactive detail = %#v", detail)
	}
	stored, err := websiteService.websiteRepo.Get(repo.WithByID(site.ID))
	if err != nil || stored.ID != site.ID {
		t.Fatalf("inactive registration was removed: %#v, %v", stored, err)
	}
}

func TestSourceConfigGetReturnsHashAndRejectsStaleSave(t *testing.T) {
	confPath, websiteService := setupExternalWebsiteFixture(t, `server { listen 80; server_name first.example.com; }`)
	site, err := websiteService.CreateExternalNginxSite(dto.ExternalNginxSiteCreateReq{Path: confPath})
	if err != nil {
		t.Fatalf("create external site: %v", err)
	}
	loaded, err := websiteService.GetSiteConfContent(site.ID)
	if err != nil {
		t.Fatalf("get source config: %v", err)
	}
	if loaded.Path == "" || loaded.Content == "" || len(loaded.Hash) != 64 {
		t.Fatalf("loaded source = %#v", loaded)
	}

	externalContent := []byte(`server { listen 80; server_name changed-outside.example.com; }`)
	if err := os.WriteFile(confPath, externalContent, 0o644); err != nil {
		t.Fatalf("external edit: %v", err)
	}
	err = websiteService.SaveSiteConfContent(dto.SaveSiteConfReq{
		ID: site.ID, Content: `server { listen 80; server_name panel-edit.example.com; }`, Hash: loaded.Hash,
	})
	requireBusinessErrorKey(t, err, constant.ErrWebsiteExternalConfigConflict)
	got, err := os.ReadFile(confPath)
	if err != nil || !bytes.Equal(got, externalContent) {
		t.Fatalf("stale save changed file: %q, %v", got, err)
	}
}

func TestSourceConfigSaveRollsBackWhenNginxTestFails(t *testing.T) {
	confPath, websiteService := setupExternalWebsiteFixture(t, `server { listen 80; server_name rollback.example.com; }`)
	site, err := websiteService.CreateExternalNginxSite(dto.ExternalNginxSiteCreateReq{Path: confPath})
	if err != nil {
		t.Fatalf("create external site: %v", err)
	}
	loaded, err := websiteService.GetSiteConfContent(site.ID)
	if err != nil {
		t.Fatalf("get source config: %v", err)
	}
	original, _ := os.ReadFile(confPath)
	reloads := 0
	websiteService.sourceSaveOps = &sourceConfigSaveOps{
		nginxTest: func() error { return errors.New("invalid nginx") },
		reload: func() error {
			reloads++
			return nil
		},
	}
	err = websiteService.SaveSiteConfContent(dto.SaveSiteConfReq{
		ID: site.ID, Content: `server { listen 81; server_name replacement.example.com; }`, Hash: loaded.Hash,
	})
	if err == nil {
		t.Fatal("nginx test failure was ignored")
	}
	after, _ := os.ReadFile(confPath)
	if !bytes.Equal(after, original) || reloads != 0 {
		t.Fatalf("rollback failed: reloads=%d content=%q", reloads, after)
	}
}

func TestSourceConfigSaveRejectsDuplicatePrimaryDomainBeforeWrite(t *testing.T) {
	confPath, websiteService := setupExternalWebsiteFixture(t, `server { listen 80; server_name original.example.com; }`)
	site, err := websiteService.CreateExternalNginxSite(dto.ExternalNginxSiteCreateReq{Path: confPath})
	if err != nil {
		t.Fatalf("create external site: %v", err)
	}
	conflict := model.Website{PrimaryDomain: "conflict.example.com", Alias: "conflict", ConfigMode: "managed"}
	if err := websiteService.websiteRepo.Create(&conflict); err != nil {
		t.Fatalf("create conflicting site: %v", err)
	}
	loaded, err := websiteService.GetSiteConfContent(site.ID)
	if err != nil {
		t.Fatalf("load source config: %v", err)
	}
	original, err := os.ReadFile(confPath)
	if err != nil {
		t.Fatalf("read original config: %v", err)
	}
	nginxTests := 0
	websiteService.sourceSaveOps = &sourceConfigSaveOps{
		nginxTest: func() error { nginxTests++; return nil },
		reload:    func() error { return nil },
	}
	err = websiteService.SaveSiteConfContent(dto.SaveSiteConfReq{
		ID: site.ID, Content: `server { listen 80; server_name conflict.example.com; }`, Hash: loaded.Hash,
	})
	requireBusinessErrorKey(t, err, constant.ErrWebsiteDomainExist)
	after, readErr := os.ReadFile(confPath)
	if readErr != nil || !bytes.Equal(after, original) || nginxTests != 0 {
		t.Fatalf("duplicate-domain save mutated config: tests=%d content=%q err=%v", nginxTests, after, readErr)
	}
}

func TestExternalWebsiteDeleteOnlyUnregisters(t *testing.T) {
	confPath, websiteService := setupExternalWebsiteFixture(t, `server { listen 80; server_name delete.example.com; }`)
	site, err := websiteService.CreateExternalNginxSite(dto.ExternalNginxSiteCreateReq{Path: confPath})
	if err != nil {
		t.Fatalf("create external site: %v", err)
	}
	before, _ := os.ReadFile(confPath)
	if err := websiteService.Delete(site.ID); err != nil {
		t.Fatalf("unregister external site: %v", err)
	}
	after, err := os.ReadFile(confPath)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatalf("unregister changed config: %q, %v", after, err)
	}
	if _, err := websiteService.websiteRepo.Get(repo.WithByID(site.ID)); err == nil {
		t.Fatal("external registration still exists")
	}
}

func TestSourceConfigSaveRollsBackAndRecoversWhenReloadFails(t *testing.T) {
	confPath, websiteService := setupExternalWebsiteFixture(t, `server { listen 80; server_name reload.example.com; }`)
	site, err := websiteService.CreateExternalNginxSite(dto.ExternalNginxSiteCreateReq{Path: confPath})
	if err != nil {
		t.Fatalf("create external site: %v", err)
	}
	loaded, _ := websiteService.GetSiteConfContent(site.ID)
	original, _ := os.ReadFile(confPath)
	reloads := 0
	websiteService.sourceSaveOps = &sourceConfigSaveOps{
		nginxTest: func() error { return nil },
		reload: func() error {
			reloads++
			if reloads == 1 {
				return errors.New("reload failed")
			}
			return nil
		},
	}
	err = websiteService.SaveSiteConfContent(dto.SaveSiteConfReq{
		ID: site.ID, Content: `server { listen 81; server_name replacement.example.com; }`, Hash: loaded.Hash,
	})
	if err == nil {
		t.Fatal("reload failure was ignored")
	}
	after, _ := os.ReadFile(confPath)
	if !bytes.Equal(after, original) || reloads != 2 {
		t.Fatalf("recovery failed: reloads=%d content=%q", reloads, after)
	}
}

func TestSourceConfigSaveRollsBackWhenMetadataSyncFails(t *testing.T) {
	confPath, websiteService := setupExternalWebsiteFixture(t, `server { listen 80; server_name sync.example.com; }`)
	site, err := websiteService.CreateExternalNginxSite(dto.ExternalNginxSiteCreateReq{Path: confPath})
	if err != nil {
		t.Fatalf("create external site: %v", err)
	}
	loaded, _ := websiteService.GetSiteConfContent(site.ID)
	original, _ := os.ReadFile(confPath)
	reloads := 0
	websiteService.sourceSaveOps = &sourceConfigSaveOps{
		nginxTest: func() error { return nil },
		reload: func() error {
			reloads++
			return nil
		},
		syncSite: func(*model.Website, nginxSiteMetadata) error {
			return errors.New("database unavailable")
		},
	}
	err = websiteService.SaveSiteConfContent(dto.SaveSiteConfReq{
		ID: site.ID, Content: `server { listen 82; server_name replacement.example.com; }`, Hash: loaded.Hash,
	})
	if err == nil {
		t.Fatal("metadata sync failure was ignored")
	}
	after, _ := os.ReadFile(confPath)
	stored, _ := websiteService.websiteRepo.Get(repo.WithByID(site.ID))
	if !bytes.Equal(after, original) || reloads != 2 || stored.PrimaryDomain != "sync.example.com" {
		t.Fatalf("sync rollback failed: reloads=%d stored=%#v content=%q", reloads, stored, after)
	}
}

func TestSourceConfigSaveSucceedsAndPreservesMode(t *testing.T) {
	confPath, websiteService := setupExternalWebsiteFixture(t, `server { listen 80; server_name old.example.com; }`)
	if err := os.Chmod(confPath, 0o600); err != nil {
		t.Fatalf("chmod source: %v", err)
	}
	site, err := websiteService.CreateExternalNginxSite(dto.ExternalNginxSiteCreateReq{Path: confPath})
	if err != nil {
		t.Fatalf("create external site: %v", err)
	}
	loaded, _ := websiteService.GetSiteConfContent(site.ID)
	websiteService.sourceSaveOps = &sourceConfigSaveOps{
		nginxTest: func() error { return nil },
		reload:    func() error { return nil },
	}
	err = websiteService.SaveSiteConfContent(dto.SaveSiteConfReq{
		ID: site.ID, Content: `server { listen 8088; server_name new.example.com www.new.example.com; }`, Hash: loaded.Hash,
	})
	if err != nil {
		t.Fatalf("save source config: %v", err)
	}
	info, err := os.Stat(confPath)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %v, err = %v", info.Mode().Perm(), err)
	}
	stored, err := websiteService.websiteRepo.Get(repo.WithByID(site.ID))
	if err != nil || stored.PrimaryDomain != "new.example.com" || stored.HttpPort != 8088 {
		t.Fatalf("stored metadata = %#v, %v", stored, err)
	}
	reloaded, err := websiteService.GetSiteConfContent(site.ID)
	if err != nil || reloaded.Hash == loaded.Hash {
		t.Fatalf("new source hash = %#v, %v", reloaded, err)
	}
}

func TestExternalWebsiteRejectsManagedOperations(t *testing.T) {
	confPath, websiteService := setupExternalWebsiteFixture(t, `server { listen 80; server_name guard.example.com; }`)
	site, err := websiteService.CreateExternalNginxSite(dto.ExternalNginxSiteCreateReq{Path: confPath})
	if err != nil {
		t.Fatalf("create external site: %v", err)
	}
	for name, operation := range map[string]func() error{
		"enable":  func() error { return websiteService.Enable(site.ID) },
		"disable": func() error { return websiteService.Disable(site.ID) },
		"managed": func() error { return websiteService.SwitchConfigMode(site.ID, "managed") },
		"update":  func() error { return websiteService.Update(dto.WebsiteUpdate{ID: site.ID}) },
	} {
		t.Run(name, func(t *testing.T) {
			requireBusinessErrorKey(t, operation(), constant.ErrWebsiteExternalOperationDenied)
		})
	}
}
