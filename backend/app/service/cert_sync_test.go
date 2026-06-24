package service

import (
	"testing"

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
