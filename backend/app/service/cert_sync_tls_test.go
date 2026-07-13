package service

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCertSourceTLSConfigAcceptsSelfSignedInternalServer(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := newCertSourceTLSConfig()
	client := &http.Client{Transport: &http.Transport{TLSClientConfig: config}}
	response, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("trusted internal self-signed TLS server must be accepted: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.StatusCode)
	}
}

func TestCertSourceTLSConfigKeepsTLS12Minimum(t *testing.T) {
	config := newCertSourceTLSConfig()
	if config.MinVersion != tls.VersionTLS12 {
		t.Fatalf("expected TLS 1.2 minimum, got %d", config.MinVersion)
	}
}
