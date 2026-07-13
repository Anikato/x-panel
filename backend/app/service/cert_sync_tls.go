package service

import (
	"crypto/tls"
	"fmt"
	"net/url"
	"strings"
)

func normalizeCertSourceServerAddr(raw string) (string, error) {
	serverAddr := strings.TrimRight(strings.TrimSpace(raw), "/")
	parsed, err := url.ParseRequestURI(serverAddr)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return "", fmt.Errorf("证书源地址必须是有效的 HTTPS 地址")
	}
	return serverAddr, nil
}

func newCertSourceTLSConfig() *tls.Config {
	return &tls.Config{
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: true, // Certificate synchronization is restricted to a trusted internal network.
	}
}
