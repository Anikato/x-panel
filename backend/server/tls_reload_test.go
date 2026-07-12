package server

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestTLSCertificateReloaderUsesUpdatedCertificateForNewHandshake(t *testing.T) {
	dir := t.TempDir()
	certPath := filepath.Join(dir, "fullchain.pem")
	keyPath := filepath.Join(dir, "privkey.pem")
	firstCert, firstKey := newServerTestCertificatePair(t, "first.example.test")
	secondCert, secondKey := newServerTestCertificatePair(t, "second.example.test")
	if err := os.WriteFile(certPath, []byte(firstCert), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyPath, []byte(firstKey), 0600); err != nil {
		t.Fatal(err)
	}

	reloader, err := newTLSCertificateReloader(certPath, keyPath)
	if err != nil {
		t.Fatal(err)
	}
	first, err := reloader.GetCertificate(&tls.ClientHelloInfo{})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(certPath, []byte(secondCert), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyPath, []byte(secondKey), 0600); err != nil {
		t.Fatal(err)
	}
	second, err := reloader.GetCertificate(&tls.ClientHelloInfo{})
	if err != nil {
		t.Fatal(err)
	}
	firstFingerprint := sha256.Sum256(first.Certificate[0])
	secondFingerprint := sha256.Sum256(second.Certificate[0])
	if firstFingerprint == secondFingerprint {
		t.Fatal("new TLS handshake must use the replacement certificate")
	}
}

func newServerTestCertificatePair(t *testing.T, commonName string) (string, string) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	template := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: commonName},
		DNSNames:     []string{commonName},
		NotBefore:    now.Add(-time.Minute),
		NotAfter:     now.Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	return string(certPEM), string(keyPEM)
}
