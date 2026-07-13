package service

import (
	"errors"
	"testing"
	"time"

	"xpanel/app/model"
)

func TestShouldAutoRenewCertificateRetriesExistingFailedAndInterruptedRenewals(t *testing.T) {
	now := time.Date(2026, 7, 13, 0, 0, 0, 0, time.UTC)
	for _, status := range []string{"applied", "error", "applying"} {
		cert := model.Certificate{
			Type:       "autoApply",
			SourceType: "acme",
			AutoRenew:  true,
			Status:     status,
			Pem:        "pem",
			PrivateKey: "key",
			ExpireDate: now.Add(24 * time.Hour),
		}
		if !shouldAutoRenewCertificate(cert, now, 15*24*time.Hour) {
			t.Fatalf("status %q must remain retryable", status)
		}
	}
}

func TestShouldAutoRenewCertificateRejectsUnissuedOrIneligibleCertificates(t *testing.T) {
	now := time.Date(2026, 7, 13, 0, 0, 0, 0, time.UTC)
	base := model.Certificate{
		Type:       "autoApply",
		SourceType: "acme",
		AutoRenew:  true,
		Pem:        "pem",
		PrivateKey: "key",
		ExpireDate: now.Add(24 * time.Hour),
	}

	tests := []struct {
		name string
		cert model.Certificate
	}{
		{name: "auto renew disabled", cert: func() model.Certificate { c := base; c.AutoRenew = false; return c }()},
		{name: "missing pem", cert: func() model.Certificate { c := base; c.Pem = ""; return c }()},
		{name: "missing private key", cert: func() model.Certificate { c := base; c.PrivateKey = ""; return c }()},
		{name: "uploaded certificate", cert: func() model.Certificate { c := base; c.Type = "upload"; return c }()},
		{name: "synced certificate type", cert: func() model.Certificate { c := base; c.Type = "synced"; return c }()},
		{name: "synced certificate source", cert: func() model.Certificate { c := base; c.SourceType = "synced"; return c }()},
		{name: "missing expiration", cert: func() model.Certificate { c := base; c.ExpireDate = time.Time{}; return c }()},
		{name: "outside renewal window", cert: func() model.Certificate {
			c := base
			c.ExpireDate = now.Add(15*time.Hour*24 + time.Nanosecond)
			return c
		}()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if shouldAutoRenewCertificate(tt.cert, now, 15*24*time.Hour) {
				t.Fatal("certificate must not be automatically renewed")
			}
		})
	}
}

func TestCertificateRenewalLockExcludesConcurrentCallersAndReleases(t *testing.T) {
	const certificateID uint = 987654321

	release, err := acquireCertificateRenewal(certificateID)
	if err != nil {
		t.Fatalf("first lock acquisition failed: %v", err)
	}

	if _, err := acquireCertificateRenewal(certificateID); !errors.Is(err, errCertificateRenewalInProgress) {
		t.Fatalf("second lock acquisition error = %v, want %v", err, errCertificateRenewalInProgress)
	}

	release()

	release, err = acquireCertificateRenewal(certificateID)
	if err != nil {
		t.Fatalf("lock was not released: %v", err)
	}
	release()
}
