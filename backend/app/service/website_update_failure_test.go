package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"xpanel/app/dto"
	"xpanel/app/model"
	"xpanel/app/repo"
	"xpanel/global"
	"xpanel/security/credentials"
)

func writeFailingNginx(t *testing.T, root string) {
	t.Helper()
	script := "#!/bin/sh\n" +
		"for arg in \"$@\"; do\n" +
		"  if [ \"$arg\" = \"-t\" ]; then echo 'nginx: config test failed' >&2; exit 1; fi\n" +
		"done\n" +
		"exit 0\n"
	if err := os.WriteFile(filepath.Join(root, "sbin", "nginx"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestUpdateKeepsDatabaseAndConfigWhenNginxTestFails(t *testing.T) {
	setupManagedWebsiteNginx(t)
	writeFailingNginx(t, global.CONF.Nginx.InstallDir)

	originalConf := "server { server_name old.example.com; }\n"
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
	if err := os.WriteFile(confPath, []byte(originalConf), 0o644); err != nil {
		t.Fatal(err)
	}

	svc := &WebsiteService{websiteRepo: repo.NewIWebsiteRepo(), certRepo: repo.NewICertificateRepo()}
	err := svc.Update(dto.WebsiteUpdate{
		ID:            site.ID,
		PrimaryDomain: "new.example.com",
		SiteDir:       site.SiteDir,
		IndexFile:     site.IndexFile,
	})
	if err == nil {
		t.Fatal("expected nginx test failure")
	}

	stored, getErr := repo.NewIWebsiteRepo().Get(repo.WithByID(site.ID))
	if getErr != nil {
		t.Fatal(getErr)
	}
	if stored.PrimaryDomain != "old.example.com" {
		t.Fatalf("database domain = %q, want original", stored.PrimaryDomain)
	}
	got, readErr := os.ReadFile(confPath)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(got) != originalConf {
		t.Fatalf("nginx config mutated: %q", got)
	}
}

func TestSaveMainConfDoesNotLeaveEmptyFileWhenBackupMissingAndTestFails(t *testing.T) {
	installFakeNginx(t, false)
	writeFailingNginx(t, global.CONF.Nginx.InstallDir)
	mainConf := global.CONF.Nginx.GetMainConf()
	if err := os.Remove(mainConf); err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}

	svc := &WebsiteService{}
	err := svc.SaveMainConf("this is not valid nginx config {")
	if err == nil {
		t.Fatal("expected nginx test failure")
	}
	info, statErr := os.Stat(mainConf)
	if statErr == nil && info.Size() == 0 {
		t.Fatal("main config was truncated to empty file")
	}
	if statErr == nil {
		t.Fatalf("invalid main config left behind (%d bytes)", info.Size())
	}
	if !os.IsNotExist(statErr) {
		t.Fatal(statErr)
	}
}

func TestDisableKeepsSiteAndConfigWhenReloadFails(t *testing.T) {
	setupManagedWebsiteNginx(t)
	writeReloadFailingNginx(t, global.CONF.Nginx.InstallDir)
	site := model.Website{
		PrimaryDomain: "keep.example.com",
		Alias:         "keep_example_com",
		Type:          "static",
		Status:        "running",
		SiteDir:       "/var/www/keep",
		IndexFile:     "index.html",
	}
	if err := repo.NewIWebsiteRepo().Create(&site); err != nil {
		t.Fatal(err)
	}
	confPath := GetSiteConfPath(site.Alias)
	original := "server { server_name keep.example.com; }\n"
	if err := os.WriteFile(confPath, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	svc := &WebsiteService{websiteRepo: repo.NewIWebsiteRepo(), certRepo: repo.NewICertificateRepo()}
	if err := svc.Disable(site.ID); err == nil {
		t.Fatal("expected reload failure")
	}
	stored, err := repo.NewIWebsiteRepo().Get(repo.WithByID(site.ID))
	if err != nil {
		t.Fatal(err)
	}
	if stored.Status != "running" {
		t.Fatalf("status = %q, want running", stored.Status)
	}
	got, err := os.ReadFile(confPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != original {
		t.Fatalf("config = %q, want original", got)
	}
}

func TestEnableKeepsStoppedStateWhenSaveFails(t *testing.T) {
	setupManagedWebsiteNginx(t)
	site := model.Website{
		PrimaryDomain: "off.example.com",
		Alias:         "off_example_com",
		Type:          "static",
		Status:        "stopped",
		SiteDir:       "/var/www/off",
		IndexFile:     "index.html",
	}
	base := repo.NewIWebsiteRepo()
	if err := base.Create(&site); err != nil {
		t.Fatal(err)
	}
	confPath := GetSiteConfPath(site.Alias)
	svc := &WebsiteService{
		websiteRepo: saveFailingWebsiteRepo{IWebsiteRepo: base},
		certRepo:    repo.NewICertificateRepo(),
	}
	if err := svc.Enable(site.ID); err == nil {
		t.Fatal("expected save failure")
	}
	stored, err := base.Get(repo.WithByID(site.ID))
	if err != nil {
		t.Fatal(err)
	}
	if stored.Status != "stopped" {
		t.Fatalf("status = %q, want stopped", stored.Status)
	}
	if _, statErr := os.Stat(confPath); !os.IsNotExist(statErr) {
		t.Fatalf("enable left config behind: %v", statErr)
	}
}

func TestSaveConfFileRejectsSiblingPathAndSymlink(t *testing.T) {
	installFakeNginx(t, false)
	confDir := global.CONF.Nginx.GetConfDir()
	svc := &WebsiteService{}
	sibling := filepath.Join(filepath.Dir(confDir), "conf2", "a.conf")
	if err := os.MkdirAll(filepath.Dir(sibling), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := svc.SaveConfFile(dto.NginxConfUpdate{FilePath: sibling, Content: "events {}\n"}); err == nil {
		t.Fatal("expected sibling path to be rejected")
	}
	if _, err := os.Stat(sibling); !os.IsNotExist(err) {
		t.Fatalf("sibling file created: %v", err)
	}

	outside := filepath.Join(t.TempDir(), "secret")
	if err := os.WriteFile(outside, []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(confDir, "link.conf")
	if err := os.Symlink(outside, link); err != nil {
		t.Fatal(err)
	}
	if err := svc.SaveConfFile(dto.NginxConfUpdate{FilePath: link, Content: "pwned\n"}); err == nil {
		t.Fatal("expected symlink to be rejected")
	}
	got, err := os.ReadFile(outside)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "secret" {
		t.Fatalf("symlink target changed: %q", got)
	}
}

func TestSaveConfFileDoesNotLeaveEmptyFileWhenTestFails(t *testing.T) {
	installFakeNginx(t, false)
	writeFailingNginx(t, global.CONF.Nginx.InstallDir)
	confDir := global.CONF.Nginx.GetConfDir()
	target := filepath.Join(confDir, "site.conf")
	original := "events {}\nhttp {}\n"
	if err := os.WriteFile(target, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	svc := &WebsiteService{}
	if err := svc.SaveConfFile(dto.NginxConfUpdate{FilePath: target, Content: "not nginx"}); err == nil {
		t.Fatal("expected nginx test failure")
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != original {
		t.Fatalf("config = %q, want original", got)
	}

	missing := filepath.Join(confDir, "missing.conf")
	if err := svc.SaveConfFile(dto.NginxConfUpdate{FilePath: missing, Content: "not nginx"}); err == nil {
		t.Fatal("expected nginx test failure for new file")
	}
	if _, err := os.Stat(missing); !os.IsNotExist(err) {
		t.Fatalf("invalid new config left behind: %v", err)
	}
}

func TestSaveMainConfRestoresOriginalWhenTestFails(t *testing.T) {
	installFakeNginx(t, false)
	writeFailingNginx(t, global.CONF.Nginx.InstallDir)
	mainConf := global.CONF.Nginx.GetMainConf()
	original := "events {}\nhttp {\n}\n"
	if err := os.WriteFile(mainConf, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	svc := &WebsiteService{}
	if err := svc.SaveMainConf("not nginx"); err == nil {
		t.Fatal("expected nginx test failure")
	}
	got, err := os.ReadFile(mainConf)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != original {
		t.Fatalf("original main config lost: %q", got)
	}
}

func writeReloadFailingNginx(t *testing.T, root string) {
	t.Helper()
	script := "#!/bin/sh\n" +
		"for arg in \"$@\"; do\n" +
		"  if [ \"$arg\" = \"reload\" ]; then echo 'reload failed' >&2; exit 1; fi\n" +
		"done\n" +
		"exit 0\n"
	if err := os.WriteFile(filepath.Join(root, "sbin", "nginx"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
}

type saveFailingWebsiteRepo struct {
	repo.IWebsiteRepo
}

func (s saveFailingWebsiteRepo) Save(item *model.Website) error {
	return os.ErrInvalid
}

func TestUpdateRestoresConfigWhenReloadFails(t *testing.T) {
	setupManagedWebsiteNginx(t)
	writeReloadFailingNginx(t, global.CONF.Nginx.InstallDir)
	originalConf := "server { server_name old.example.com; }\n"
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
	if err := os.WriteFile(confPath, []byte(originalConf), 0o644); err != nil {
		t.Fatal(err)
	}
	svc := &WebsiteService{websiteRepo: repo.NewIWebsiteRepo(), certRepo: repo.NewICertificateRepo()}
	err := svc.Update(dto.WebsiteUpdate{
		ID: site.ID, PrimaryDomain: "new.example.com", SiteDir: site.SiteDir, IndexFile: site.IndexFile,
	})
	if err == nil {
		t.Fatal("expected reload failure")
	}
	got, readErr := os.ReadFile(confPath)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(got) != originalConf {
		t.Fatalf("config after reload failure = %q", got)
	}
	stored, _ := repo.NewIWebsiteRepo().Get(repo.WithByID(site.ID))
	if stored.PrimaryDomain != "old.example.com" {
		t.Fatalf("database domain = %q", stored.PrimaryDomain)
	}
}

type saveFailingAfterHtpasswdRepo struct {
	repo.IWebsiteRepo
	t            *testing.T
	htpasswdPath string
}

func (s saveFailingAfterHtpasswdRepo) Save(item *model.Website) error {
	got, err := os.ReadFile(s.htpasswdPath)
	if err != nil {
		s.t.Fatalf("htpasswd missing before save: %v", err)
	}
	if !strings.Contains(string(got), "newuser:") {
		s.t.Fatalf("htpasswd was not rewritten before save: %q", got)
	}
	return os.ErrInvalid
}

func TestUpdateRollsBackHtpasswdAndAllowsLaterUpdate(t *testing.T) {
	setupManagedWebsiteNginx(t)
	manager, _, credErr := credentials.LoadOrCreate(
		filepath.Join(t.TempDir(), "secrets", "credential-keyring.json"),
		true,
	)
	if credErr != nil {
		t.Fatal(credErr)
	}
	previousCreds := global.CREDENTIALS
	global.CREDENTIALS = manager
	t.Cleanup(func() { global.CREDENTIALS = previousCreds })
	originalConf := "server { server_name old.example.com; }\n"
	originalAuth := "olduser:$apr1$keep\n"
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
	if err := os.WriteFile(confPath, []byte(originalConf), 0o644); err != nil {
		t.Fatal(err)
	}
	htpasswdPath := filepath.Join(global.CONF.Nginx.GetConfDir(), "auth", site.Alias+".htpasswd")
	if err := os.MkdirAll(filepath.Dir(htpasswdPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(htpasswdPath, []byte(originalAuth), 0o644); err != nil {
		t.Fatal(err)
	}

	svc := &WebsiteService{
		websiteRepo: saveFailingAfterHtpasswdRepo{IWebsiteRepo: repo.NewIWebsiteRepo(), t: t, htpasswdPath: htpasswdPath},
		certRepo:    repo.NewICertificateRepo(),
	}
	err := svc.Update(dto.WebsiteUpdate{
		ID: site.ID, PrimaryDomain: "new.example.com", SiteDir: site.SiteDir, IndexFile: site.IndexFile,
		BasicAuth: true, BasicUser: "newuser", BasicPassword: "secret",
	})
	if err == nil {
		t.Fatal("expected database save failure")
	}
	gotConf, _ := os.ReadFile(confPath)
	if string(gotConf) != originalConf {
		t.Fatalf("conf after failed auth update = %q", gotConf)
	}
	gotAuth, err := os.ReadFile(htpasswdPath)
	if err != nil {
		t.Fatalf("htpasswd missing after rollback: %v", err)
	}
	if string(gotAuth) != originalAuth {
		t.Fatalf("htpasswd after rollback = %q", gotAuth)
	}

	svc.websiteRepo = repo.NewIWebsiteRepo()
	if err := svc.Update(dto.WebsiteUpdate{
		ID: site.ID, PrimaryDomain: "retry.example.com", SiteDir: site.SiteDir, IndexFile: site.IndexFile,
		BasicAuth: true, BasicUser: "retry", BasicPassword: "secret",
	}); err != nil {
		t.Fatalf("retry update after rollback failed: %v", err)
	}
	stored, err := repo.NewIWebsiteRepo().Get(repo.WithByID(site.ID))
	if err != nil {
		t.Fatal(err)
	}
	if stored.PrimaryDomain != "retry.example.com" {
		t.Fatalf("database domain after retry = %q", stored.PrimaryDomain)
	}
	retryConf, err := os.ReadFile(confPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(retryConf), "retry.example.com") {
		t.Fatalf("conf after retry missing new domain: %s", retryConf)
	}
	retryAuth, err := os.ReadFile(htpasswdPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(retryAuth), "retry:") {
		t.Fatalf("htpasswd after retry missing new user: %q", retryAuth)
	}
	if strings.Contains(string(retryAuth), "olduser:") {
		t.Fatalf("htpasswd after retry still has original user: %q", retryAuth)
	}
}

func TestUpdateRestoresConfigWhenDatabaseSaveFails(t *testing.T) {
	setupManagedWebsiteNginx(t)
	originalConf := "server { server_name old.example.com; }\n"
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
	if err := os.WriteFile(confPath, []byte(originalConf), 0o644); err != nil {
		t.Fatal(err)
	}
	svc := &WebsiteService{
		websiteRepo: saveFailingWebsiteRepo{IWebsiteRepo: repo.NewIWebsiteRepo()},
		certRepo:    repo.NewICertificateRepo(),
	}
	err := svc.Update(dto.WebsiteUpdate{
		ID: site.ID, PrimaryDomain: "new.example.com", SiteDir: site.SiteDir, IndexFile: site.IndexFile,
	})
	if err == nil {
		t.Fatal("expected database save failure")
	}
	got, readErr := os.ReadFile(confPath)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(got) != originalConf {
		t.Fatalf("config after save failure = %q", got)
	}
}

func TestGeneratedLimitConnIncludesZoneInMainConfig(t *testing.T) {
	setupManagedWebsiteNginx(t)
	if err := os.WriteFile(global.CONF.Nginx.GetMainConf(), []byte("events {}\nhttp {\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	site := model.Website{
		PrimaryDomain: "limit.example.com",
		Alias:         "limit_example_com",
		Type:          "static",
		Status:        "running",
		SiteDir:       "/var/www/limit",
		IndexFile:     "index.html",
		LimitConn:     20,
	}
	gen := NewNginxConfigGenerator()
	config, err := gen.Generate(site)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(config, "limit_conn perip 20;") {
		t.Fatalf("site config missing limit_conn: %s", config)
	}
	if err := EnsureLimitConnZone(); err != nil {
		t.Fatal(err)
	}
	main, err := os.ReadFile(global.CONF.Nginx.GetMainConf())
	if err != nil {
		t.Fatal(err)
	}
	if !hasPeripLimitConnZone(string(main)) {
		t.Fatalf("main config missing perip zone: %s", main)
	}
}

func TestEnsureLimitConnZoneIgnoresUnrelatedZones(t *testing.T) {
	setupManagedWebsiteNginx(t)
	mainConf := global.CONF.Nginx.GetMainConf()
	original := "events {}\nhttp {\n    limit_conn_zone $binary_remote_addr zone=other:10m;\n}\n"
	if err := os.WriteFile(mainConf, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := EnsureLimitConnZone(); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(mainConf)
	if err != nil {
		t.Fatal(err)
	}
	if !hasPeripLimitConnZone(string(got)) {
		t.Fatalf("perip zone not added beside other zone: %s", got)
	}
	if !strings.Contains(string(got), "zone=other:10m") {
		t.Fatalf("existing zone lost: %s", got)
	}
}
