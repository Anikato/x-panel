package service

import (
	"os"
	"path/filepath"
	"testing"

	"xpanel/app/dto"
	"xpanel/app/model"
	"xpanel/app/repo"
	"xpanel/global"
)

func TestParseNginxSiteMetadataReadsRootInsideLocation(t *testing.T) {
	got, err := parseNginxSiteMetadata(`
server {
    listen 80;
    server_name blog.example.com;
    location / {
        root /data/blog/public;
        index index.php index.html;
    }
}
`)
	if err != nil {
		t.Fatal(err)
	}
	if got.Root != "/data/blog/public" || got.Type != "static" {
		t.Fatalf("metadata = %#v", got)
	}
}

func TestParseNginxSiteMetadataWarnsForDisjointServerNames(t *testing.T) {
	got, err := parseNginxSiteMetadata(`
server { server_name a.example.com; root /data/a; }
server { server_name b.example.com; root /data/b; }
`)
	if err != nil {
		t.Fatal(err)
	}
	if got.PrimaryDomain != "a.example.com" || got.Root != "/data/a" {
		t.Fatalf("metadata = %#v", got)
	}
	if !containsString(got.Warnings, nginxDisjointServerNamesWarning) {
		t.Fatalf("warnings = %#v", got.Warnings)
	}
}

func TestScanActiveNginxSitesClassifiesFilesWithoutImporting(t *testing.T) {
	openWebsiteExternalTestDB(t)
	root := t.TempDir()
	confDir := filepath.Join(root, "conf", "conf.d")
	blogRoot := filepath.Join(root, "www", "blog")
	if err := os.MkdirAll(blogRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	writeNginxFixture(t, filepath.Join(root, "sbin", "nginx"), "")
	mainConf := writeNginxFixture(t, filepath.Join(root, "conf", "nginx.conf"), "include conf.d/*.conf;\n")
	writeNginxFixture(t, filepath.Join(confDir, "managed_example_com.conf"), "server { server_name managed.example.com; root /var/www/managed; }\n")
	blogConf := writeNginxFixture(t, filepath.Join(confDir, "blog.conf"), "server { server_name blog.example.com; location / { root "+blogRoot+"; } }\n")
	writeNginxFixture(t, filepath.Join(confDir, "snippet.conf"), "fastcgi_param SCRIPT_FILENAME $document_root$fastcgi_script_name;\n")
	writeNginxFixture(t, filepath.Join(confDir, "default.conf"), "server { listen 80 default_server; server_name _; root /var/www/html; }\n")
	writeNginxFixture(t, filepath.Join(confDir, "mixed.conf"), "server { server_name a.example.com; root /data/a; }\nserver { server_name b.example.com; root /data/b; }\n")
	takenConf := writeNginxFixture(t, filepath.Join(confDir, "taken.conf"), "server { server_name taken.example.com; root /data/taken; }\n")

	previous := global.CONF
	global.CONF.Nginx = global.NginxConfig{InstallDir: root, Mode: "prefix"}
	global.CONF.Nginx.DetectNginx()
	t.Cleanup(func() { global.CONF = previous })
	if global.CONF.Nginx.GetMainConf() != mainConf {
		t.Fatalf("main conf = %s", global.CONF.Nginx.GetMainConf())
	}

	websiteRepo := repo.NewIWebsiteRepo()
	for _, site := range []model.Website{
		{PrimaryDomain: "managed.example.com", Alias: "managed_example_com", Type: "static", Status: "running", ConfigMode: "managed"},
		{PrimaryDomain: "taken.example.com", Alias: "taken_example_com", Type: "static", Status: "running", ConfigMode: "managed"},
	} {
		if err := websiteRepo.Create(&site); err != nil {
			t.Fatal(err)
		}
	}

	items, err := (&WebsiteService{websiteRepo: websiteRepo, certRepo: repo.NewICertificateRepo()}).ScanActiveNginxSites()
	if err != nil {
		t.Fatal(err)
	}
	byDomain := map[string]dto.NginxSiteScanItem{}
	for _, item := range items {
		byDomain[item.PrimaryDomain] = item
	}
	if _, ok := byDomain["managed.example.com"]; ok {
		t.Fatal("managed site was offered for import")
	}
	if len(items) != 3 {
		t.Fatalf("items = %#v", items)
	}

	blog := byDomain["blog.example.com"]
	if !blog.Importable || blog.Root != blogRoot || blog.Path != blogConf {
		t.Fatalf("blog = %#v", blog)
	}
	mixed := byDomain["a.example.com"]
	if !mixed.Importable || !containsString(mixed.Warnings, nginxDisjointServerNamesWarning) || mixed.Root != "/data/a" {
		t.Fatalf("mixed = %#v", mixed)
	}
	if _, err := os.Stat("/data/a"); err == nil {
		t.Fatal("test should not require /data/a to exist")
	}
	if !containsString(mixed.Warnings, nginxSiteDirMissingWarning) {
		t.Fatalf("missing directory warning = %#v", mixed.Warnings)
	}
	taken := byDomain["taken.example.com"]
	if taken.Importable || taken.Path != takenConf || taken.Reason == "" {
		t.Fatalf("taken = %#v", taken)
	}
}

func TestImportScannedNginxSitesRegistersSourceSiteWithoutRewritingConfig(t *testing.T) {
	openWebsiteExternalTestDB(t)
	root := t.TempDir()
	siteRoot := filepath.Join(root, "www", "blog")
	if err := os.MkdirAll(siteRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	writeNginxFixture(t, filepath.Join(root, "sbin", "nginx"), "")
	writeNginxFixture(t, filepath.Join(root, "conf", "nginx.conf"), "include conf.d/*.conf;\n")
	original := "server { server_name blog.example.com; location / { root " + siteRoot + "; } }\n"
	blogConf := writeNginxFixture(t, filepath.Join(root, "conf", "conf.d", "blog.conf"), original)
	previous := global.CONF
	global.CONF.Nginx = global.NginxConfig{InstallDir: root, Mode: "prefix"}
	global.CONF.Nginx.DetectNginx()
	t.Cleanup(func() { global.CONF = previous })

	svc := &WebsiteService{websiteRepo: repo.NewIWebsiteRepo(), certRepo: repo.NewICertificateRepo()}
	results, err := svc.ImportScannedNginxSites(dto.NginxSiteImportReq{Paths: []string{blogConf}})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Error != "" || results[0].PrimaryDomain != "blog.example.com" || results[0].Root != siteRoot {
		t.Fatalf("results = %#v", results)
	}
	stored, err := repo.NewIWebsiteRepo().Get(repo.WithByID(results[0].ID))
	if err != nil {
		t.Fatal(err)
	}
	if stored.ConfigMode != "source" || stored.SiteDir != siteRoot || stored.Status != "running" {
		t.Fatalf("stored = %#v", stored)
	}
	got, err := os.ReadFile(blogConf)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != original {
		t.Fatalf("config was rewritten: %s", got)
	}

	again, err := svc.ImportScannedNginxSites(dto.NginxSiteImportReq{Paths: []string{blogConf}})
	if err != nil {
		t.Fatal(err)
	}
	if len(again) != 1 || again[0].Error == "" {
		t.Fatalf("second import = %#v", again)
	}
}
