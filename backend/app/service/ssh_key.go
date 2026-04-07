package service

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/ssh"

	"xpanel/global"
)

type SSHKeyInfo struct {
	Name       string `json:"name"`
	PublicKey  string `json:"publicKey"`
	KeyType    string `json:"keyType"`
	Bits       int    `json:"bits"`
	Fingerprint string `json:"fingerprint"`
}

type SSHKeyCreate struct {
	Name string `json:"name" binding:"required"`
	Bits int    `json:"bits"`
}

type SSHKeyImport struct {
	Name       string `json:"name" binding:"required"`
	PrivateKey string `json:"privateKey" binding:"required"`
}

type ISSHKeyService interface {
	List() ([]SSHKeyInfo, error)
	GetPrivateKey(name string) (string, error)
	Generate(req SSHKeyCreate) (*SSHKeyInfo, string, error)
	Import(req SSHKeyImport) error
	Delete(name string) error
}

type SSHKeyService struct{}

func NewISSHKeyService() ISSHKeyService { return &SSHKeyService{} }

func sshKeyDir() string {
	return filepath.Join(global.CONF.System.DataDir, "ssh-keys")
}

func (s *SSHKeyService) List() ([]SSHKeyInfo, error) {
	dir := sshKeyDir()
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return []SSHKeyInfo{}, nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var keys []SSHKeyInfo
	for _, entry := range entries {
		if entry.IsDir() || strings.HasSuffix(entry.Name(), ".pub") {
			continue
		}
		name := entry.Name()
		info := SSHKeyInfo{Name: name}

		pubPath := filepath.Join(dir, name+".pub")
		if pubData, err := os.ReadFile(pubPath); err == nil {
			pubContent := strings.TrimSpace(string(pubData))
			info.PublicKey = pubContent
			parts := strings.Fields(pubContent)
			if len(parts) >= 1 {
				info.KeyType = parts[0]
			}
			if pubKey, _, _, _, err := ssh.ParseAuthorizedKey(pubData); err == nil {
				info.Fingerprint = ssh.FingerprintSHA256(pubKey)
				info.Bits = pubKeyBits(pubKey)
			}
		}

		keys = append(keys, info)
	}
	return keys, nil
}

func (s *SSHKeyService) GetPrivateKey(name string) (string, error) {
	path := filepath.Join(sshKeyDir(), name)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("key not found: %s", name)
	}
	return string(data), nil
}

func (s *SSHKeyService) Generate(req SSHKeyCreate) (*SSHKeyInfo, string, error) {
	bits := req.Bits
	if bits == 0 {
		bits = 4096
	}
	if bits < 2048 {
		bits = 2048
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, "", fmt.Errorf("generate key failed: %v", err)
	}

	privPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	pubKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, "", fmt.Errorf("generate public key failed: %v", err)
	}
	pubBytes := ssh.MarshalAuthorizedKey(pubKey)

	dir := sshKeyDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, "", err
	}

	privPath := filepath.Join(dir, req.Name)
	pubPath := filepath.Join(dir, req.Name+".pub")

	if _, err := os.Stat(privPath); err == nil {
		return nil, "", fmt.Errorf("key '%s' already exists", req.Name)
	}

	if err := os.WriteFile(privPath, privPEM, 0600); err != nil {
		return nil, "", err
	}
	if err := os.WriteFile(pubPath, pubBytes, 0644); err != nil {
		return nil, "", err
	}

	info := &SSHKeyInfo{
		Name:        req.Name,
		PublicKey:    strings.TrimSpace(string(pubBytes)),
		KeyType:     pubKey.Type(),
		Bits:        bits,
		Fingerprint: ssh.FingerprintSHA256(pubKey),
	}

	global.LOG.Infof("SSH key generated: %s (%d bits)", req.Name, bits)
	return info, string(privPEM), nil
}

func (s *SSHKeyService) Import(req SSHKeyImport) error {
	dir := sshKeyDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	privPath := filepath.Join(dir, req.Name)
	if _, err := os.Stat(privPath); err == nil {
		return fmt.Errorf("key '%s' already exists", req.Name)
	}

	signer, err := ssh.ParsePrivateKey([]byte(req.PrivateKey))
	if err != nil {
		return fmt.Errorf("invalid private key: %v", err)
	}

	pubBytes := ssh.MarshalAuthorizedKey(signer.PublicKey())

	if err := os.WriteFile(privPath, []byte(req.PrivateKey), 0600); err != nil {
		return err
	}
	pubPath := filepath.Join(dir, req.Name+".pub")
	if err := os.WriteFile(pubPath, pubBytes, 0644); err != nil {
		return err
	}

	global.LOG.Infof("SSH key imported: %s", req.Name)
	return nil
}

func (s *SSHKeyService) Delete(name string) error {
	dir := sshKeyDir()
	privPath := filepath.Join(dir, name)
	pubPath := filepath.Join(dir, name+".pub")

	if _, err := os.Stat(privPath); os.IsNotExist(err) {
		return fmt.Errorf("key not found: %s", name)
	}

	os.Remove(privPath)
	os.Remove(pubPath)
	global.LOG.Infof("SSH key deleted: %s", name)
	return nil
}

func pubKeyBits(key ssh.PublicKey) int {
	if cpk, ok := key.(ssh.CryptoPublicKey); ok {
		switch k := cpk.CryptoPublicKey().(type) {
		case *rsa.PublicKey:
			return k.N.BitLen()
		}
	}
	return 0
}
