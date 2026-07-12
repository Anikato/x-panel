package service

import (
	"crypto/tls"
	"testing"

	"xpanel/app/model"
)

func TestCertSourceTLSConfigUsesSystemVerificationByDefault(t *testing.T) {
	config, err := newCertSourceTLSConfig(model.CertSource{
		ServerAddr: "https://cert.example.test",
	})
	if err != nil {
		t.Fatal(err)
	}
	if config.InsecureSkipVerify {
		t.Fatal("default certificate source connection must verify the server certificate")
	}
	if config.MinVersion != tls.VersionTLS12 {
		t.Fatalf("expected TLS 1.2 minimum, got %d", config.MinVersion)
	}
}

func TestCertSourceTLSConfigRequiresValidPinnedFingerprint(t *testing.T) {
	config, err := newCertSourceTLSConfig(model.CertSource{
		ServerAddr:     "https://192.0.2.10:7777",
		TLSFingerprint: "AA:BB:CC:DD",
	})
	if err == nil {
		t.Fatal("expected malformed TLS fingerprint to be rejected")
	}
	if config != nil {
		t.Fatal("malformed TLS fingerprint must not produce a client configuration")
	}
}

func TestCertSourceTLSConfigPinsExplicitFingerprint(t *testing.T) {
	config, err := newCertSourceTLSConfig(model.CertSource{
		ServerAddr:     "https://192.0.2.10:7777",
		TLSFingerprint: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !config.InsecureSkipVerify || config.VerifyConnection == nil {
		t.Fatal("pinned source must verify the presented certificate through VerifyConnection")
	}
}
