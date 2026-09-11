package service

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"xpanel/app/dto"
	"xpanel/app/model"
	"xpanel/app/repo"
	"xpanel/global"

	"github.com/glebarez/sqlite"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func setupCertificateServiceDB(t *testing.T) *CertificateService {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "cert.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Certificate{}, &model.Setting{}, &model.AcmeAccount{}); err != nil {
		t.Fatal(err)
	}
	previousDB, previousLog := global.DB, global.LOG
	global.DB = db
	global.LOG = logrus.New()
	t.Cleanup(func() {
		global.DB = previousDB
		global.LOG = previousLog
	})
	return NewICertificateService().(*CertificateService)
}

func generateUploadPEMs(t *testing.T) (string, string) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "upload.example.com"},
		DNSNames:     []string{"upload.example.com"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	return string(certPEM), string(keyPEM)
}

func TestUploadDoesNotKeepRecordWhenFilesCannotBeWritten(t *testing.T) {
	svc := setupCertificateServiceDB(t)
	blocked := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(blocked, []byte("file"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := svc.settingRepo.Create(&model.Setting{Key: "SSLDir", Value: blocked}); err != nil {
		t.Fatal(err)
	}
	certPEM, keyPEM := generateUploadPEMs(t)
	err := svc.Upload(dto.CertificateUpload{
		Certificate: certPEM,
		PrivateKey:  keyPEM,
	})
	if err == nil {
		t.Fatal("expected write failure")
	}
	var count int64
	if dbErr := global.DB.Model(&model.Certificate{}).Count(&count).Error; dbErr != nil {
		t.Fatal(dbErr)
	}
	if count != 0 {
		t.Fatalf("certificate rows = %d, want 0 after failed upload", count)
	}
}

func TestApplyDoesNotMarkApplyingWhenLockHeld(t *testing.T) {
	svc := setupCertificateServiceDB(t)
	cert := model.Certificate{
		PrimaryDomain: "lock.example.com",
		Type:          "autoApply",
		Status:        "ready",
	}
	if err := svc.certRepo.Create(&cert); err != nil {
		t.Fatal(err)
	}
	release, err := acquireCertificateRenewal(cert.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer release()

	if err := svc.Apply(cert.ID); err == nil {
		t.Fatal("expected in-progress error")
	}
	stored, getErr := svc.certRepo.Get(repo.WithByID(cert.ID))
	if getErr != nil {
		t.Fatal(getErr)
	}
	if stored.Status != "ready" {
		t.Fatalf("status = %q, want ready", stored.Status)
	}
}
