package credentials

import (
	"bytes"
	"encoding/base64"
	"strings"
	"testing"
)

func TestEnvelopeRoundTripUsesRandomNonce(t *testing.T) {
	key := bytes.Repeat([]byte{0x42}, 32)

	first, err := sealEnvelope("kek-test", key, "hosts.password", "secret")
	if err != nil {
		t.Fatalf("seal first envelope: %v", err)
	}
	second, err := sealEnvelope("kek-test", key, "hosts.password", "secret")
	if err != nil {
		t.Fatalf("seal second envelope: %v", err)
	}
	if first == second {
		t.Fatal("two envelopes for the same plaintext must use different nonces")
	}
	if !IsEncrypted(first) {
		t.Fatalf("IsEncrypted(%q) = false", first)
	}
	if !strings.HasPrefix(first, envelopePrefix) {
		t.Fatalf("envelope %q does not use prefix %q", first, envelopePrefix)
	}
	keyID, err := EnvelopeKeyID(first)
	if err != nil {
		t.Fatalf("read key id: %v", err)
	}
	if keyID != "kek-test" {
		t.Fatalf("key id = %q, want kek-test", keyID)
	}

	plaintext, err := openEnvelope(key, "hosts.password", first)
	if err != nil {
		t.Fatalf("open envelope: %v", err)
	}
	if plaintext != "secret" {
		t.Fatalf("plaintext = %q, want secret", plaintext)
	}
}

func TestEnvelopeRejectsWrongScopeAndTampering(t *testing.T) {
	key := bytes.Repeat([]byte{0x24}, 32)
	value, err := sealEnvelope("kek-test", key, "hosts.password", "secret")
	if err != nil {
		t.Fatalf("seal envelope: %v", err)
	}

	if _, err := openEnvelope(key, "nodes.token", value); err == nil {
		t.Fatal("opening an envelope under a different scope must fail")
	}

	parts := strings.Split(value, ":")
	ciphertext, err := base64.RawURLEncoding.DecodeString(parts[5])
	if err != nil {
		t.Fatalf("decode ciphertext for tampering: %v", err)
	}
	ciphertext[len(ciphertext)-1] ^= 0x01
	parts[5] = base64.RawURLEncoding.EncodeToString(ciphertext)
	tampered := strings.Join(parts, ":")
	if _, err := openEnvelope(key, "hosts.password", tampered); err == nil {
		t.Fatal("opening a tampered envelope must fail")
	}
}

func TestEnvelopeRejectsInvalidKeyLength(t *testing.T) {
	if _, err := sealEnvelope("kek-test", bytes.Repeat([]byte{1}, 16), "hosts.password", "secret"); err == nil {
		t.Fatal("sealEnvelope must reject a non-256-bit KEK")
	}
}

func TestEnvelopeRejectsMalformedEncoding(t *testing.T) {
	key := bytes.Repeat([]byte{0x33}, 32)
	tests := []string{
		"",
		"xpanel:enc:v2:kek-test:wrapped:ciphertext",
		"xpanel:enc:v1::wrapped:ciphertext",
		"xpanel:enc:v1:kek-test:only-one-payload",
		"xpanel:enc:v1:kek-test:***:***",
		"xpanel:enc:v1:kek-test:YQ:YQ",
	}

	for _, value := range tests {
		t.Run(value, func(t *testing.T) {
			if _, err := openEnvelope(key, "hosts.password", value); err == nil {
				t.Fatalf("openEnvelope(%q) must fail", value)
			}
			if strings.HasPrefix(value, "xpanel:enc:") {
				if _, err := EnvelopeKeyID(value); err == nil && !strings.HasPrefix(value, envelopePrefix) {
					t.Fatalf("EnvelopeKeyID(%q) accepted an unsupported version", value)
				}
			}
		})
	}
}
