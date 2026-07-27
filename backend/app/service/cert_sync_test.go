package service

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"xpanel/app/dto"
	"xpanel/app/model"
	"xpanel/app/repo"
	"xpanel/global"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
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

func TestSyncedCertificatePersistsLocalAutoRenewDisabled(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	previousDB := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = previousDB })
	if err := db.AutoMigrate(&model.Certificate{}); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}

	cert := model.Certificate{
		PrimaryDomain: "synced.example.com",
		Type:          "autoApply",
		SourceType:    "acme",
		AutoRenew:     true,
	}
	applySyncedCertificateMetadata(&cert, 9, "upstream")
	if err := repo.NewICertificateRepo().Create(&cert); err != nil {
		t.Fatalf("create synchronized certificate: %v", err)
	}

	var storedAutoRenew bool
	if err := db.Model(&model.Certificate{}).
		Select("auto_renew").
		Where("id = ?", cert.ID).
		Scan(&storedAutoRenew).Error; err != nil {
		t.Fatalf("read stored auto renewal flag: %v", err)
	}
	if storedAutoRenew {
		t.Fatal("synchronized certificate persisted local auto renewal as enabled")
	}
}

func TestApplyRemoteRenewalMetadataCopiesUpstreamWithoutEnablingLocalRenewal(t *testing.T) {
	lastRenewedAt := time.Date(2026, 6, 1, 2, 0, 0, 0, time.UTC)
	nextRenewAt := time.Date(2026, 8, 1, 2, 0, 0, 0, time.UTC)
	cert := model.Certificate{AutoRenew: true}
	remote := dto.CertServerItem{
		AutoRenew:            true,
		RenewalMetadataKnown: true,
		LastAutoRenewedAt:    &lastRenewedAt,
		NextAutoRenewAt:      &nextRenewAt,
	}

	applyRemoteRenewalMetadata(&cert, remote)

	if cert.AutoRenew {
		t.Fatal("synchronized certificate must not enable local renewal")
	}
	if !cert.UpstreamAutoRenew {
		t.Fatal("upstream auto-renew flag was not copied")
	}
	if !cert.UpstreamRenewalMetadataKnown {
		t.Fatal("upstream renewal metadata presence was not copied")
	}
	if cert.LastAutoRenewedAt == nil || !cert.LastAutoRenewedAt.Equal(lastRenewedAt) {
		t.Fatalf("last auto renewal = %v, want %v", cert.LastAutoRenewedAt, lastRenewedAt)
	}
	if cert.UpstreamNextAutoRenewAt == nil || !cert.UpstreamNextAutoRenewAt.Equal(nextRenewAt) {
		t.Fatalf("upstream next renewal = %v, want %v", cert.UpstreamNextAutoRenewAt, nextRenewAt)
	}
}

func TestApplyRemoteRenewalMetadataAcceptsLegacyPayload(t *testing.T) {
	now := time.Date(2026, 6, 1, 2, 0, 0, 0, time.UTC)
	cert := model.Certificate{
		AutoRenew:               true,
		UpstreamAutoRenew:       true,
		LastAutoRenewedAt:       &now,
		UpstreamNextAutoRenewAt: &now,
	}

	applyRemoteRenewalMetadata(&cert, dto.CertServerItem{})

	if cert.AutoRenew || cert.UpstreamAutoRenew || cert.UpstreamRenewalMetadataKnown ||
		cert.LastAutoRenewedAt != nil || cert.UpstreamNextAutoRenewAt != nil {
		t.Fatalf("legacy payload must produce safe zero-value renewal metadata: %+v", cert)
	}
}

func TestApplyEffectiveRenewalMetadataToServerItemForwardsSyncedOwner(t *testing.T) {
	now := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	lastRenewedAt := now.Add(-20 * 24 * time.Hour)
	nextRenewAt := now.Add(30 * 24 * time.Hour)
	cert := model.Certificate{
		Type:                         "synced",
		SourceType:                   "synced",
		UpstreamAutoRenew:            true,
		UpstreamRenewalMetadataKnown: true,
		LastAutoRenewedAt:            &lastRenewedAt,
		UpstreamNextAutoRenewAt:      &nextRenewAt,
	}
	item := dto.CertServerItem{}

	applyEffectiveRenewalMetadataToServerItem(&item, cert, now, time.UTC)

	if !item.AutoRenew {
		t.Fatal("server item must forward upstream automatic-renewal ownership")
	}
	if !item.RenewalMetadataKnown {
		t.Fatal("server item must forward upstream renewal metadata presence")
	}
	if item.LastAutoRenewedAt == nil || !item.LastAutoRenewedAt.Equal(lastRenewedAt) {
		t.Fatalf("last auto renewal = %v, want %v", item.LastAutoRenewedAt, lastRenewedAt)
	}
	if item.NextAutoRenewAt == nil || !item.NextAutoRenewAt.Equal(nextRenewAt) {
		t.Fatalf("next auto renewal = %v, want %v", item.NextAutoRenewAt, nextRenewAt)
	}
}

func TestRemoteRenewalMetadataChangedDetectsScheduleOnlyUpdates(t *testing.T) {
	oldTime := time.Date(2026, 6, 1, 2, 0, 0, 0, time.UTC)
	newTime := oldTime.Add(24 * time.Hour)
	cert := model.Certificate{
		UpstreamAutoRenew:       true,
		LastAutoRenewedAt:       &oldTime,
		UpstreamNextAutoRenewAt: &oldTime,
	}
	remote := dto.CertServerItem{
		AutoRenew:            true,
		RenewalMetadataKnown: true,
		LastAutoRenewedAt:    &newTime,
		NextAutoRenewAt:      &newTime,
	}

	if !remoteRenewalMetadataChanged(cert, remote) {
		t.Fatal("schedule-only upstream change must be persisted")
	}
	applyRemoteRenewalMetadata(&cert, remote)
	if remoteRenewalMetadataChanged(cert, remote) {
		t.Fatal("equal upstream renewal metadata must not cause another update")
	}
}

func TestRemoteRenewalMetadataChangedRepairsHistoricalLocalRenewalFlag(t *testing.T) {
	cert := model.Certificate{
		Type:       "synced",
		SourceType: "synced",
		AutoRenew:  true,
	}

	if !remoteRenewalMetadataChanged(cert, dto.CertServerItem{}) {
		t.Fatal("historical synchronized certificate with local renewal enabled must be repaired")
	}
}
