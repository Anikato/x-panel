package service

import (
	"crypto/sha256"
	"crypto/subtle"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"
	"time"

	"xpanel/app/model"
)

func normalizeCertSourceServerAddr(raw string) (string, error) {
	serverAddr := strings.TrimRight(strings.TrimSpace(raw), "/")
	parsed, err := url.ParseRequestURI(serverAddr)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return "", fmt.Errorf("证书源地址必须是有效的 HTTPS 地址")
	}
	return serverAddr, nil
}

func newCertSourceTLSConfig(source model.CertSource) (*tls.Config, error) {
	config := &tls.Config{MinVersion: tls.VersionTLS12}
	fingerprint, err := normalizeCertSourceTLSFingerprint(source.TLSFingerprint)
	if err != nil {
		return nil, err
	}
	if fingerprint == "" {
		return config, nil
	}

	expected, err := hex.DecodeString(fingerprint)
	if err != nil {
		return nil, fmt.Errorf("TLS 指纹格式无效: %w", err)
	}
	config.InsecureSkipVerify = true // VerifyConnection below performs the explicit pin check.
	config.VerifyConnection = func(state tls.ConnectionState) error {
		if len(state.PeerCertificates) == 0 {
			return fmt.Errorf("上游未返回 TLS 证书")
		}
		leaf := state.PeerCertificates[0]
		now := time.Now()
		if now.Before(leaf.NotBefore) || now.After(leaf.NotAfter) {
			return fmt.Errorf("上游 TLS 证书不在有效期内")
		}
		actual := sha256.Sum256(leaf.Raw)
		if subtle.ConstantTimeCompare(expected, actual[:]) != 1 {
			return fmt.Errorf("上游 TLS 证书指纹不匹配")
		}
		return nil
	}
	return config, nil
}

func normalizeCertSourceTLSFingerprint(raw string) (string, error) {
	fingerprint := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(raw), ":", ""))
	if fingerprint == "" {
		return "", nil
	}
	if len(fingerprint) != sha256.Size*2 {
		return "", fmt.Errorf("TLS 指纹必须是 64 位 SHA-256 十六进制值")
	}
	if _, err := hex.DecodeString(fingerprint); err != nil {
		return "", fmt.Errorf("TLS 指纹必须是十六进制值")
	}
	return fingerprint, nil
}
