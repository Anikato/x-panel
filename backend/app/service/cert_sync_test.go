package service

import (
	"os"
	"path/filepath"
	"testing"

	"xpanel/app/dto"
	"xpanel/app/model"
)

func TestSyncedCertificateIsNotRenewable(t *testing.T) {
	cases := []model.Certificate{
		{Type: "synced", SourceType: "synced", AutoRenew: true, Status: "applied"},
		{Type: "autoApply", SourceType: "synced", AutoRenew: true, Status: "applied"},
	}

	for _, cert := range cases {
		if isRenewableCertificate(cert) {
			t.Fatalf("synced certificate should not be renewable: type=%s sourceType=%s", cert.Type, cert.SourceType)
		}
	}
}

func TestSaveSyncedCertFilesAtomicRejectsInvalidPairWithoutOverwrite(t *testing.T) {
	sslDir := t.TempDir()
	cert := model.Certificate{
		BaseModel:  model.BaseModel{ID: 7},
		Pem:        "invalid certificate",
		PrivateKey: "invalid key",
	}
	dir := filepath.Join(sslDir, "certs", "cert-7")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	certPath := filepath.Join(dir, "fullchain.pem")
	keyPath := filepath.Join(dir, "privkey.pem")
	if err := os.WriteFile(certPath, []byte("old certificate"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyPath, []byte("old key"), 0600); err != nil {
		t.Fatal(err)
	}

	if err := saveSyncedCertFilesAtomic(sslDir, cert, true); err == nil {
		t.Fatal("expected invalid key pair to fail")
	}
	gotCert, _ := os.ReadFile(certPath)
	gotKey, _ := os.ReadFile(keyPath)
	if string(gotCert) != "old certificate" || string(gotKey) != "old key" {
		t.Fatal("active certificate files were modified after validation failure")
	}
}

func TestCanonicalCertificateDomainsIgnoreOrderAndCase(t *testing.T) {
	got := canonicalCertificateDomains("Example.COM.", "www.example.com,api.example.com", `["API.EXAMPLE.COM","example.com"]`)
	want := "api.example.com,example.com,www.example.com"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestSelectLegacyCertificatePrefersExactFingerprint(t *testing.T) {
	remote := dto.CertServerItem{
		PrimaryDomain: "example.com",
		Domains:       "www.example.com",
		Fingerprint:   "NEW",
	}
	candidates := []model.Certificate{
		{BaseModel: model.BaseModel{ID: 1}, PrimaryDomain: "example.com", Domains: "www.example.com", Fingerprint: "OLD"},
		{BaseModel: model.BaseModel{ID: 2}, PrimaryDomain: "example.com", Domains: "www.example.com", Fingerprint: "NEW"},
	}

	got, err := selectLegacyCertificate(remote, candidates)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != 2 {
		t.Fatalf("expected exact fingerprint candidate 2, got %d", got.ID)
	}
}

func TestSelectLegacyCertificateRejectsAmbiguousSAN(t *testing.T) {
	remote := dto.CertServerItem{PrimaryDomain: "example.com", Domains: "www.example.com", Fingerprint: "NEW"}
	candidates := []model.Certificate{
		{BaseModel: model.BaseModel{ID: 1}, PrimaryDomain: "example.com", Domains: "www.example.com", Fingerprint: "OLD-1"},
		{BaseModel: model.BaseModel{ID: 2}, PrimaryDomain: "example.com", Domains: "www.example.com", Fingerprint: "OLD-2"},
	}

	if _, err := selectLegacyCertificate(remote, candidates); err == nil {
		t.Fatal("expected ambiguous legacy candidates to be rejected")
	}
}

func TestSelectLegacyCertificatePrefersOnlyReferencedSANMatch(t *testing.T) {
	remote := dto.CertServerItem{PrimaryDomain: "example.com", Domains: "www.example.com", Fingerprint: "NEW"}
	candidates := []model.Certificate{
		{BaseModel: model.BaseModel{ID: 1}, PrimaryDomain: "example.com", Domains: "www.example.com", Fingerprint: "OLD-1"},
		{BaseModel: model.BaseModel{ID: 2}, PrimaryDomain: "example.com", Domains: "www.example.com", Fingerprint: "OLD-2"},
	}
	got, err := selectLegacyCertificateWithReferences(remote, candidates, map[uint]bool{1: true})
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != 1 {
		t.Fatalf("expected referenced certificate 1, got %d", got.ID)
	}
}

func TestApplySyncedCertificateMetadataDisablesLocalRenewal(t *testing.T) {
	cert := model.Certificate{
		Type:          "autoApply",
		Provider:      "dns",
		AcmeAccountID: 12,
		DnsAccountID:  34,
		CertURL:       "https://example.test/acme/cert/1",
		AutoRenew:     true,
		SourceType:    "acme",
	}

	applySyncedCertificateMetadata(&cert, 9, "upstream")

	if cert.Type != "synced" {
		t.Fatalf("expected type synced, got %s", cert.Type)
	}
	if cert.SourceType != "synced" {
		t.Fatalf("expected source type synced, got %s", cert.SourceType)
	}
	if cert.AutoRenew {
		t.Fatal("expected auto renew disabled for synced certificate")
	}
	if cert.Provider != "manual" {
		t.Fatalf("expected provider manual, got %s", cert.Provider)
	}
	if cert.AcmeAccountID != 0 || cert.DnsAccountID != 0 || cert.CertURL != "" {
		t.Fatalf("expected local ACME renewal metadata cleared, got acme=%d dns=%d certURL=%q",
			cert.AcmeAccountID, cert.DnsAccountID, cert.CertURL)
	}
	if cert.SourceID != 9 || cert.SourceName != "upstream" {
		t.Fatalf("expected upstream source metadata, got id=%d name=%q", cert.SourceID, cert.SourceName)
	}
}
