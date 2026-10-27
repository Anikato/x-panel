package service

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"xpanel/app/model"
	"xpanel/app/repo"
	"xpanel/global"
	sslutil "xpanel/utils/ssl"
)

func TestHTTPSRedirectKeepsACMEChallengeAheadOfTheRedirect(t *testing.T) {
	dataDir := filepath.Join(t.TempDir(), "data")
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		t.Fatal(err)
	}
	previous := global.CONF
	global.CONF.System.DataDir = dataDir
	t.Cleanup(func() { global.CONF = previous })
	openWebsiteExternalTestDB(t)

	cert := model.Certificate{PrimaryDomain: "www.example.com", Provider: "http", Status: "applied"}
	if err := global.DB.Create(&cert).Error; err != nil {
		t.Fatal(err)
	}
	certDir := filepath.Join(dataDir, "ssl", "certs", "cert-"+strconv.FormatUint(uint64(cert.ID), 10))
	if err := os.MkdirAll(certDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(certDir, "fullchain.pem"), []byte("cert"), 0o644); err != nil {
		t.Fatal(err)
	}

	config, err := NewNginxConfigGenerator().Generate(model.Website{
		PrimaryDomain: "www.example.com",
		Domains:       "example.com",
		Alias:         "example",
		Type:          "static",
		SSLEnable:     true,
		CertificateID: cert.ID,
		HttpConfig:    "HTTPSRedirect",
		SiteDir:       "/var/www/example",
	})
	if err != nil {
		t.Fatal(err)
	}

	block := firstServerBlock(t, config)
	if strings.Contains(block, "\n    return 301 ") {
		t.Fatalf("server-level redirect runs before location matching:\n%s", block)
	}
	wantRedirect := "    location / {\n        return 301 https://$host$request_uri;\n    }\n"
	if !strings.Contains(block, wantRedirect) {
		t.Fatalf("redirect block missing:\n%s", block)
	}
	wantRoot := "        root " + sslutil.HTTP01WebRoot() + ";\n"
	if !strings.Contains(block, wantRoot) {
		t.Fatalf("ACME root missing:\n%s", block)
	}
	if strings.Contains(sslutil.HTTP01WebRoot(), filepath.Clean(dataDir)+string(os.PathSeparator)) || sslutil.HTTP01WebRoot() == filepath.Clean(dataDir) {
		t.Fatalf("ACME root %s is inside the private data directory", sslutil.HTTP01WebRoot())
	}
}

func TestApplyConfigDefinesXpanelLogFormatBeforeWritingTheSite(t *testing.T) {
	setupManagedWebsiteNginx(t)
	confPath := global.CONF.Nginx.GetMainConf()
	original := debianStyleNginxConf("")
	if err := os.WriteFile(confPath, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	site := model.Website{
		PrimaryDomain: "logs.example.com",
		Alias:         "logs_example_com",
		Type:          "static",
		Status:        "running",
		SiteDir:       "/var/www/logs",
		IndexFile:     "index.html",
		AccessLog:     true,
		ErrorLog:      true,
	}
	svc := &WebsiteService{websiteRepo: repo.NewIWebsiteRepo(), certRepo: repo.NewICertificateRepo()}
	if _, err := svc.applyConfigWithRollback(site); err != nil {
		t.Fatal(err)
	}

	assertXpanelLogFormatBeforeSites(t, readTestFile(t, confPath))
	siteConf := readTestFile(t, GetSiteConfPath(site.Alias))
	if !strings.Contains(siteConf, "access_log ") || !strings.Contains(siteConf, " xpanel;") {
		t.Fatalf("site config does not use the xpanel log format:\n%s", siteConf)
	}
}

func TestApplyConfigWithoutAccessLogDoesNotRewriteMainConfig(t *testing.T) {
	setupManagedWebsiteNginx(t)
	confPath := global.CONF.Nginx.GetMainConf()
	original := debianStyleNginxConf("")
	if err := os.WriteFile(confPath, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	site := model.Website{
		PrimaryDomain: "plain.example.com",
		Alias:         "plain_example_com",
		Type:          "static",
		Status:        "running",
		SiteDir:       "/var/www/plain",
		IndexFile:     "index.html",
		AccessLog:     false,
	}
	svc := &WebsiteService{websiteRepo: repo.NewIWebsiteRepo(), certRepo: repo.NewICertificateRepo()}
	if _, err := svc.applyConfigWithRollback(site); err != nil {
		t.Fatal(err)
	}
	if got := readTestFile(t, confPath); got != original {
		t.Fatalf("main config changed for a site with access logs off:\n%s", got)
	}
}

func firstServerBlock(t *testing.T, config string) string {
	t.Helper()
	start := strings.Index(config, "server {")
	if start < 0 {
		t.Fatalf("no server block:\n%s", config)
	}
	rest := config[start+len("server {"):]
	end := strings.Index(rest, "\n}")
	if end < 0 {
		t.Fatalf("unterminated server block:\n%s", config)
	}
	return config[start : start+len("server {")+end+len("\n}")]
}
