package service

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"xpanel/app/model"
	"xpanel/global"
)

func discardLogger() *log.Logger {
	return log.New(io.Discard, "", 0)
}

func installCertificateDatabase(t *testing.T) {
	t.Helper()
	installCertificateConsumerDatabase(t)
	if err := global.DB.AutoMigrate(&model.Certificate{}); err != nil {
		t.Fatal(err)
	}
}

func createAppliedCertificate(t *testing.T) model.Certificate {
	t.Helper()
	cert := model.Certificate{
		PrimaryDomain: "renew.example.com",
		Provider:      "http",
		Type:          "autoApply",
		Status:        "applied",
	}
	if err := global.DB.Create(&cert).Error; err != nil {
		t.Fatal(err)
	}
	return cert
}

func TestActivateCertificateConsumersReportsNginxReloadFailure(t *testing.T) {
	installCertificateDatabase(t)
	root := filepath.Dir(installFakeNginx(t, true))
	if err := os.WriteFile(filepath.Join(root, "sbin", "nginx"), []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	cert := createAppliedCertificate(t)
	svc := NewICertificateService().(*CertificateService)
	err := svc.activateCertificateConsumers(cert.ID, discardLogger())
	if err == nil {
		t.Fatal("expected activation error")
	}

	var stored model.Certificate
	if dbErr := global.DB.First(&stored, cert.ID).Error; dbErr != nil {
		t.Fatal(dbErr)
	}
	if stored.Status != "applied" {
		t.Fatalf("status = %q, want applied", stored.Status)
	}
	if !strings.HasPrefix(stored.Message, certificateActivationFailedPrefix) {
		t.Fatalf("message = %q", stored.Message)
	}
}

func TestActivateCertificateConsumersClearsMessageOnSuccess(t *testing.T) {
	installCertificateDatabase(t)
	installFakeNginx(t, true)
	cert := createAppliedCertificate(t)
	if err := global.DB.Model(&cert).Update("message", certificateActivationFailedPrefix+"old").Error; err != nil {
		t.Fatal(err)
	}

	svc := NewICertificateService().(*CertificateService)
	if err := svc.activateCertificateConsumers(cert.ID, discardLogger()); err != nil {
		t.Fatal(err)
	}

	var stored model.Certificate
	if err := global.DB.First(&stored, cert.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Message != "" {
		t.Fatalf("message = %q, want empty", stored.Message)
	}
}

func TestActivateCertificateConsumersSkipsUnusedHAProxyAndGOST(t *testing.T) {
	installCertificateDatabase(t)
	installFakeNginx(t, true)
	cert := createAppliedCertificate(t)
	if err := global.DB.Create(&model.HAProxyLB{
		Name:          "other",
		BindPort:      443,
		CertificateID: cert.ID + 99,
		EnableSSL:     true,
		Enabled:       true,
	}).Error; err != nil {
		t.Fatal(err)
	}

	targets, err := findCertificateConsumerTargets([]uint{cert.ID})
	if err != nil {
		t.Fatal(err)
	}
	if targets.HAProxy || targets.GOST {
		t.Fatalf("unused consumers selected: %#v", targets)
	}

	svc := NewICertificateService().(*CertificateService)
	if err := svc.activateCertificateConsumers(cert.ID, discardLogger()); err != nil {
		t.Fatal(err)
	}
}

func TestShouldRetryCertificateActivation(t *testing.T) {
	if shouldRetryCertificateActivation(model.Certificate{Status: "applied", Message: ""}) {
		t.Fatal("successful applied cert must not skip CA")
	}
	if !shouldRetryCertificateActivation(model.Certificate{
		Status:  "applied",
		Message: certificateActivationFailedPrefix + "Nginx reload: failed",
	}) {
		t.Fatal("activation failure must be retryable without CA")
	}
	if shouldRetryCertificateActivation(model.Certificate{
		Status:  "error",
		Message: certificateActivationFailedPrefix + "x",
	}) {
		t.Fatal("error status is not an activation retry")
	}
}
