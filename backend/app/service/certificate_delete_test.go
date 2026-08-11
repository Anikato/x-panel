package service

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"xpanel/app/model"
	"xpanel/app/repo"
	"xpanel/global"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupCertificateDeleteTest(t *testing.T) (*CertificateService, string) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&model.Certificate{},
		&model.Website{},
		&model.HAProxyLB{},
		&model.GostService{},
		&model.Setting{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	previousDB := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = previousDB })

	sslDir := t.TempDir()
	if err := db.Create(&model.Setting{Key: "SSLDir", Value: sslDir}).Error; err != nil {
		t.Fatalf("create SSLDir setting: %v", err)
	}
	return &CertificateService{
		certRepo:    repo.NewICertificateRepo(),
		settingRepo: repo.NewISettingRepo(),
	}, sslDir
}

func TestCertificateBatchDeleteSkipsAllConsumerTypesAndApplying(t *testing.T) {
	svc, sslDir := setupCertificateDeleteTest(t)
	certs := []model.Certificate{
		{BaseModel: model.BaseModel{ID: 11}, PrimaryDomain: "website.example.com", Provider: "manual", Status: "applied"},
		{BaseModel: model.BaseModel{ID: 12}, PrimaryDomain: "haproxy.example.com", Provider: "manual", Status: "applied"},
		{BaseModel: model.BaseModel{ID: 13}, PrimaryDomain: "gost.example.com", Provider: "manual", Status: "applied"},
		{BaseModel: model.BaseModel{ID: 14}, PrimaryDomain: "panel.example.com", Provider: "manual", Status: "applied"},
		{BaseModel: model.BaseModel{ID: 15}, PrimaryDomain: "applying.example.com", Provider: "dns", Status: "applying"},
	}
	for _, cert := range certs {
		createCertificateDeleteFixture(t, svc, sslDir, cert)
	}
	if err := global.DB.Create(&model.Website{PrimaryDomain: "website.example.com", Alias: "website", CertificateID: 11}).Error; err != nil {
		t.Fatal(err)
	}
	if err := global.DB.Create(&model.HAProxyLB{Name: "tls", BindPort: 443, CertificateID: 12}).Error; err != nil {
		t.Fatal(err)
	}
	if err := global.DB.Create(&model.GostService{Name: "tls", Type: "tcp_forward", ListenAddr: ":443", CertificateID: 13}).Error; err != nil {
		t.Fatal(err)
	}
	if err := global.DB.Create(&model.Setting{Key: "PanelSSLCertificateID", Value: "14"}).Error; err != nil {
		t.Fatal(err)
	}

	result, err := svc.BatchDelete([]uint{11, 12, 13, 14, 15})
	if err != nil {
		t.Fatalf("BatchDelete: %v", err)
	}
	if result.DeletedCount != 0 || len(result.Skipped) != 5 || len(result.Failed) != 0 {
		t.Fatalf("result = %#v", result)
	}
	for _, cert := range certs {
		if _, err := os.Stat(certDirPath(sslDir, cert)); err != nil {
			t.Fatalf("skipped directory %d must remain: %v", cert.ID, err)
		}
	}
}

func TestCertificateCleanupExpiredOnlyDeletesPastNonZeroExpiry(t *testing.T) {
	svc, sslDir := setupCertificateDeleteTest(t)
	now := time.Now().UTC().Truncate(time.Second)
	certs := []model.Certificate{
		{BaseModel: model.BaseModel{ID: 21}, PrimaryDomain: "expired.example.com", Provider: "manual", Status: "applied", ExpireDate: now.Add(-time.Hour)},
		{BaseModel: model.BaseModel{ID: 22}, PrimaryDomain: "future.example.com", Provider: "manual", Status: "applied", ExpireDate: now.Add(time.Hour)},
		{BaseModel: model.BaseModel{ID: 23}, PrimaryDomain: "unknown.example.com", Provider: "manual", Status: "error"},
	}
	for _, cert := range certs {
		createCertificateDeleteFixture(t, svc, sslDir, cert)
	}

	result, err := svc.CleanupExpired(now)
	if err != nil {
		t.Fatalf("CleanupExpired: %v", err)
	}
	if result.DeletedCount != 1 || len(result.Skipped) != 0 || len(result.Failed) != 0 {
		t.Fatalf("result = %#v", result)
	}
	var remainingIDs []uint
	if err := global.DB.Model(&model.Certificate{}).Order("id").Pluck("id", &remainingIDs).Error; err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(remainingIDs) != "[22 23]" {
		t.Fatalf("remaining IDs = %v", remainingIDs)
	}
}

func TestCertificateSingleDeleteUsesReferenceProtection(t *testing.T) {
	svc, sslDir := setupCertificateDeleteTest(t)
	cert := model.Certificate{BaseModel: model.BaseModel{ID: 31}, PrimaryDomain: "used.example.com", Provider: "manual", Status: "applied"}
	createCertificateDeleteFixture(t, svc, sslDir, cert)
	if err := global.DB.Create(&model.Website{PrimaryDomain: cert.PrimaryDomain, Alias: "used", CertificateID: cert.ID}).Error; err != nil {
		t.Fatal(err)
	}

	if err := svc.Delete(cert.ID); err == nil {
		t.Fatal("Delete error = nil, want reference protection")
	}
	var count int64
	if err := global.DB.Model(&model.Certificate{}).Where("id = ?", cert.ID).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("certificate count = %d, want 1", count)
	}
}

func TestCertificateBatchDeleteReferenceQueryFailureDeletesNothing(t *testing.T) {
	svc, sslDir := setupCertificateDeleteTest(t)
	cert := model.Certificate{BaseModel: model.BaseModel{ID: 41}, PrimaryDomain: "safe.example.com", Provider: "manual", Status: "applied"}
	createCertificateDeleteFixture(t, svc, sslDir, cert)
	if err := global.DB.Migrator().DropTable(&model.HAProxyLB{}); err != nil {
		t.Fatal(err)
	}

	if _, err := svc.BatchDelete([]uint{cert.ID}); err == nil {
		t.Fatal("BatchDelete error = nil, want reference query failure")
	}
	var count int64
	if err := global.DB.Model(&model.Certificate{}).Where("id = ?", cert.ID).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("certificate count = %d, want 1", count)
	}
	if _, err := os.Stat(certDirPath(sslDir, cert)); err != nil {
		t.Fatalf("directory must remain: %v", err)
	}
}

func TestCertificateBatchDeleteDirectoryFailureContinuesOtherRecords(t *testing.T) {
	svc, sslDir := setupCertificateDeleteTest(t)
	failed := model.Certificate{BaseModel: model.BaseModel{ID: 51}, PrimaryDomain: "failed.example.com", Provider: "manual", Status: "applied"}
	deleted := model.Certificate{BaseModel: model.BaseModel{ID: 52}, PrimaryDomain: "deleted.example.com", Provider: "manual", Status: "applied"}
	createCertificateDeleteFixture(t, svc, sslDir, failed)
	createCertificateDeleteFixture(t, svc, sslDir, deleted)

	previousRemove := removeCertificateDirectory
	removeCertificateDirectory = func(path string) error {
		if path == certDirPath(sslDir, failed) {
			return fmt.Errorf("fixture remove failure")
		}
		return os.RemoveAll(path)
	}
	t.Cleanup(func() { removeCertificateDirectory = previousRemove })

	result, err := svc.BatchDelete([]uint{failed.ID, deleted.ID, 999})
	if err != nil {
		t.Fatalf("BatchDelete: %v", err)
	}
	if result.DeletedCount != 1 || len(result.Failed) != 2 || len(result.Skipped) != 0 {
		t.Fatalf("result = %#v", result)
	}
	var remainingIDs []uint
	if err := global.DB.Model(&model.Certificate{}).Order("id").Pluck("id", &remainingIDs).Error; err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(remainingIDs) != "[51]" {
		t.Fatalf("remaining IDs = %v", remainingIDs)
	}
}

func createCertificateDeleteFixture(t *testing.T, svc *CertificateService, sslDir string, cert model.Certificate) {
	t.Helper()
	if err := global.DB.Create(&cert).Error; err != nil {
		t.Fatalf("create certificate: %v", err)
	}
	dir := certDirPath(sslDir, cert)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatalf("create certificate directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "fullchain.pem"), []byte("fixture"), 0o600); err != nil {
		t.Fatalf("write certificate fixture: %v", err)
	}
}

func TestCertificateBatchDeleteDeletesUnreferencedAndSkipsWebsiteReference(t *testing.T) {
	svc, sslDir := setupCertificateDeleteTest(t)
	now := time.Now()
	deletable := model.Certificate{BaseModel: model.BaseModel{ID: 1}, PrimaryDomain: "old.example.com", Provider: "manual", Status: "applied", ExpireDate: now.Add(-time.Hour)}
	referenced := model.Certificate{BaseModel: model.BaseModel{ID: 2}, PrimaryDomain: "live.example.com", Provider: "manual", Status: "applied", ExpireDate: now.Add(-time.Hour)}
	createCertificateDeleteFixture(t, svc, sslDir, deletable)
	createCertificateDeleteFixture(t, svc, sslDir, referenced)
	if err := global.DB.Create(&model.Website{PrimaryDomain: "live.example.com", Alias: "live", CertificateID: referenced.ID}).Error; err != nil {
		t.Fatalf("create website reference: %v", err)
	}

	result, err := svc.BatchDelete([]uint{deletable.ID, referenced.ID, deletable.ID})
	if err != nil {
		t.Fatalf("BatchDelete: %v", err)
	}
	if result.DeletedCount != 1 || len(result.Skipped) != 1 || len(result.Failed) != 0 {
		t.Fatalf("result = %#v", result)
	}
	if result.Skipped[0].ID != referenced.ID {
		t.Fatalf("skipped = %#v, want certificate %d", result.Skipped, referenced.ID)
	}

	var remaining []model.Certificate
	if err := global.DB.Order("id").Find(&remaining).Error; err != nil {
		t.Fatalf("list remaining: %v", err)
	}
	if len(remaining) != 1 || remaining[0].ID != referenced.ID {
		t.Fatalf("remaining = %#v", remaining)
	}
	if _, err := os.Stat(certDirPath(sslDir, deletable)); !os.IsNotExist(err) {
		t.Fatalf("deleted directory stat error = %v, want not exists", err)
	}
	if _, err := os.Stat(certDirPath(sslDir, referenced)); err != nil {
		t.Fatalf("referenced directory must remain: %v", err)
	}
}
