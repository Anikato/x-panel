package service

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"xpanel/app/dto"
	"xpanel/app/model"
	"xpanel/app/repo"
	"xpanel/global"

	"github.com/sirupsen/logrus"
)

func TestCertificateConsumersIncludeBindingsAndNginxPaths(t *testing.T) {
	installCertificateConsumerDatabase(t)
	if err := global.DB.AutoMigrate(&model.Certificate{}, &model.Setting{}); err != nil {
		t.Fatal(err)
	}
	installFakeNginx(t, true)
	global.CONF.System.DataDir = t.TempDir()
	global.LOG = logrus.New()
	sslDir := global.CONF.GetDefaultSSLDir()
	cert := model.Certificate{PrimaryDomain: "example.com", Provider: "manual", Type: "upload", KeyType: "2048"}
	if err := global.DB.Create(&cert).Error; err != nil {
		t.Fatal(err)
	}
	site := model.Website{PrimaryDomain: "example.com", Alias: "example_com", Type: "static", Status: "running", CertificateID: cert.ID, SSLEnable: true}
	if err := global.DB.Create(&site).Error; err != nil {
		t.Fatal(err)
	}
	if err := global.DB.Create(&model.HAProxyLB{
		Name: "web", BindAddr: "0.0.0.0", BindPort: 443, CertificateID: cert.ID, EnableSSL: true, Enabled: true,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := global.DB.Create(&model.Setting{Key: "PanelSSLCertificateID", Value: fmt.Sprint(cert.ID)}).Error; err != nil {
		t.Fatal(err)
	}

	certDir := certDirPath(sslDir, cert)
	included := filepath.Join(global.CONF.Nginx.GetConfDir(), "hand.conf")
	if err := os.MkdirAll(filepath.Dir(included), 0o755); err != nil {
		t.Fatal(err)
	}
	certFile := filepath.Join(certDir, "fullchain.pem")
	if err := os.WriteFile(included, []byte("server {\n    server_name hand.example.com;\n    ssl_certificate "+certFile+";\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	main := "events {}\nhttp {\n    include " + included + ";\n}\n"
	if err := os.WriteFile(global.CONF.Nginx.GetMainConf(), []byte(main), 0o644); err != nil {
		t.Fatal(err)
	}

	grouped, err := listCertificateConsumers([]uint{cert.ID})
	if err != nil {
		t.Fatal(err)
	}
	kinds := map[string]bool{}
	for _, consumer := range grouped[cert.ID] {
		kinds[consumer.Kind] = true
	}
	for _, kind := range []string{dto.CertificateConsumerWebsite, dto.CertificateConsumerHAProxy, dto.CertificateConsumerPanel, dto.CertificateConsumerNginx} {
		if !kinds[kind] {
			t.Fatalf("consumers = %#v, missing %s", grouped[cert.ID], kind)
		}
	}
	targets, err := findCertificateConsumerTargets([]uint{cert.ID})
	if err != nil {
		t.Fatal(err)
	}
	if !targets.Nginx || !targets.HAProxy {
		t.Fatalf("targets = %#v", targets)
	}
}

func TestReplaceExpiredWebsiteCertificate(t *testing.T) {
	setupManagedWebsiteNginx(t)
	global.LOG = logrus.New()
	now := time.Now()
	sslDir := global.CONF.GetDefaultSSLDir()
	expired := model.Certificate{PrimaryDomain: "old.example.com", Provider: "manual", Type: "upload", KeyType: "2048", ExpireDate: now.Add(-time.Hour)}
	fresh := model.Certificate{PrimaryDomain: "example.com", Provider: "manual", Type: "upload", KeyType: "2048", ExpireDate: now.Add(90 * 24 * time.Hour)}
	if err := global.DB.Create(&expired).Error; err != nil {
		t.Fatal(err)
	}
	if err := global.DB.Create(&fresh).Error; err != nil {
		t.Fatal(err)
	}
	writeCertificateFixture(t, certDirPath(sslDir, expired), []string{"example.com"}, now.Add(-48*time.Hour), now.Add(-time.Hour))
	writeCertificateFixture(t, certDirPath(sslDir, fresh), []string{"example.com", "self.example.com"}, now.Add(-time.Hour), now.Add(90*24*time.Hour))
	site := model.Website{
		PrimaryDomain: "example.com", Alias: "example_com", Type: "static", Status: "stopped",
		CertificateID: expired.ID, SSLEnable: true, SiteDir: "/var/www/example", IndexFile: "index.html",
	}
	if err := repo.NewIWebsiteRepo().Create(&site); err != nil {
		t.Fatal(err)
	}
	source := model.Website{
		PrimaryDomain: "source.example.com", Alias: "source", Type: "static", Status: "stopped",
		ConfigMode: "source", CertificateID: expired.ID, SSLEnable: true,
	}
	kept := model.Website{
		PrimaryDomain: "self.example.com", Alias: "self_example_com", Type: "static", Status: "stopped",
		CertificateID: expired.ID, SSLEnable: true, SkipCertAdapt: true,
	}
	if err := repo.NewIWebsiteRepo().Create(&source); err != nil {
		t.Fatal(err)
	}
	if err := repo.NewIWebsiteRepo().Create(&kept); err != nil {
		t.Fatal(err)
	}
	replaceUnusableWebsiteCertificates(now)

	stored, err := repo.NewIWebsiteRepo().Get(repo.WithByID(site.ID))
	if err != nil {
		t.Fatal(err)
	}
	if stored.CertificateID != fresh.ID {
		t.Fatalf("certificate id = %d, want %d", stored.CertificateID, fresh.ID)
	}
	untouched, err := repo.NewIWebsiteRepo().Get(repo.WithByID(source.ID))
	if err != nil {
		t.Fatal(err)
	}
	if untouched.CertificateID != expired.ID {
		t.Fatalf("source-mode site changed certificate to %d", untouched.CertificateID)
	}
	skipped, err := repo.NewIWebsiteRepo().Get(repo.WithByID(kept.ID))
	if err != nil {
		t.Fatal(err)
	}
	if skipped.CertificateID != expired.ID {
		t.Fatalf("skip-adapt site changed certificate to %d", skipped.CertificateID)
	}
}
