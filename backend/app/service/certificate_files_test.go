package service

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"xpanel/app/model"
)

func TestSyncedCertFileTransactionRestoresPreviousPairOnRollback(t *testing.T) {
	sslDir := t.TempDir()
	oldPEM, oldKey := newTestCertificatePair(t, "old.example.test")
	newPEM, newKey := newTestCertificatePair(t, "new.example.test")
	cert := model.Certificate{
		BaseModel:  model.BaseModel{ID: 7},
		Pem:        newPEM,
		PrivateKey: newKey,
	}
	certPath, keyPath := certFilePaths(sslDir, cert)
	if err := os.MkdirAll(filepath.Dir(certPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(certPath, []byte(oldPEM), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyPath, []byte(oldKey), 0600); err != nil {
		t.Fatal(err)
	}

	tx, err := prepareSyncedCertFileTransaction(sslDir, cert, true)
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}

	gotPEM, err := os.ReadFile(certPath)
	if err != nil {
		t.Fatal(err)
	}
	gotKey, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(gotPEM) != oldPEM || string(gotKey) != oldKey {
		t.Fatal("rollback must restore both original certificate files")
	}
}

func TestSyncedCertFileTransactionRollbackBeforeCommitKeepsActivePair(t *testing.T) {
	sslDir := t.TempDir()
	oldPEM, oldKey := newTestCertificatePair(t, "old.example.test")
	newPEM, newKey := newTestCertificatePair(t, "new.example.test")
	cert := model.Certificate{
		BaseModel:  model.BaseModel{ID: 8},
		Pem:        newPEM,
		PrivateKey: newKey,
	}
	certPath, keyPath := certFilePaths(sslDir, cert)
	if err := os.MkdirAll(filepath.Dir(certPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(certPath, []byte(oldPEM), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyPath, []byte(oldKey), 0600); err != nil {
		t.Fatal(err)
	}

	tx, err := prepareSyncedCertFileTransaction(sslDir, cert, true)
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}

	gotPEM, err := os.ReadFile(certPath)
	if err != nil {
		t.Fatal(err)
	}
	gotKey, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(gotPEM) != oldPEM || string(gotKey) != oldKey {
		t.Fatal("rollback before commit must not modify active certificate files")
	}
}

func newTestCertificatePair(t *testing.T, commonName string) (string, string) {
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
