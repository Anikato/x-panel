package ssl

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"time"

	"github.com/go-acme/lego/v4/certcrypto"
	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/challenge/dns01"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/registration"
)

type KeyType = certcrypto.KeyType

const (
	KeyEC256   = certcrypto.EC256
	KeyEC384   = certcrypto.EC384
	KeyRSA2048 = certcrypto.RSA2048
	KeyRSA3072 = certcrypto.RSA3072
	KeyRSA4096 = certcrypto.RSA4096
)

// AcmeUser 实现 lego 的 registration.User 接口
type AcmeUser struct {
	Email        string
	Registration *registration.Resource
	Key          crypto.PrivateKey
}

func (u *AcmeUser) GetEmail() string                        { return u.Email }
func (u *AcmeUser) GetRegistration() *registration.Resource { return u.Registration }
func (u *AcmeUser) GetPrivateKey() crypto.PrivateKey         { return u.Key }

// AcmeClient ACME 客户端封装
type AcmeClient struct {
	Config *lego.Config
	Client *lego.Client
	User   *AcmeUser
}

// GetCaDirURL 根据 CA 类型获取 ACME 目录 URL
func GetCaDirURL(caType, customURL string) string {
	switch caType {
	case "letsencrypt":
		return "https://acme-v02.api.letsencrypt.org/directory"
	case "zerossl":
		return "https://acme.zerossl.com/v2/DV90"
	case "buypass":
		return "https://api.buypass.com/acme/directory"
	case "google":
		return "https://dv.acme-v02.api.pki.goog/directory"
	case "custom":
		return customURL
	default:
		return "https://acme-v02.api.letsencrypt.org/directory"
	}
}

// NewAcmeClient 创建 ACME 客户端（已注册的账户）
// accountURL 是注册时 CA 返回的账户 URL，作为 JWS kid 使用
func NewAcmeClient(email, privateKeyPEM, keyType, caType, caDirURL, accountURL string) (*AcmeClient, error) {
	priKey, err := ParsePrivateKey(privateKeyPEM, keyType)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %v", err)
	}

	user := &AcmeUser{Email: email, Key: priKey}

	// 必须在 NewClient 之前设置 Registration
	// 因为 lego.NewClient 在创建时读取 user.GetRegistration().URI 作为 JWS kid
	// 如果 kid 为空，所有后续请求都会被 CA 拒绝 (No Key ID in JWS header)
	user.Registration = &registration.Resource{
		URI: accountURL,
	}

	config := lego.NewConfig(user)
	config.CADirURL = GetCaDirURL(caType, caDirURL)
	config.Certificate.KeyType = certcrypto.KeyType(keyType)
	config.Certificate.Timeout = 60 * time.Second
	config.UserAgent = "X-Panel"

	// 调试日志：确认 kid 值
	if reg := user.GetRegistration(); reg != nil {
		fmt.Printf("[ACME DEBUG] kid (Registration.URI) = %q\n", reg.URI)
	} else {
		fmt.Println("[ACME DEBUG] WARNING: Registration is nil, kid will be empty!")
	}

	client, err := lego.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("create lego client: %v", err)
	}

	return &AcmeClient{Config: config, Client: client, User: user}, nil
}

// RegisterAccount 注册新 ACME 账户
func RegisterAccount(email, keyType, caType, caDirURL string) (privateKeyPEM string, url string, err error) {
	priKey, err := certcrypto.GeneratePrivateKey(certcrypto.KeyType(keyType))
	if err != nil {
		return "", "", fmt.Errorf("generate private key: %v", err)
	}

	user := &AcmeUser{Email: email, Key: priKey}
	config := lego.NewConfig(user)
	config.CADirURL = GetCaDirURL(caType, caDirURL)
	config.Certificate.KeyType = certcrypto.KeyType(keyType)
	config.UserAgent = "X-Panel"

	client, err := lego.NewClient(config)
	if err != nil {
		return "", "", fmt.Errorf("create lego client: %v", err)
	}

	reg, err := client.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: true})
	if err != nil {
		return "", "", fmt.Errorf("register account: %v", err)
	}

	// 序列化私钥
	keyPEM, err := EncodePrivateKey(priKey, keyType)
	if err != nil {
		return "", "", err
	}

	return string(keyPEM), reg.URI, nil
}

// ObtainCertificate 申请证书
func (c *AcmeClient) ObtainCertificate(domains []string, keyType string) (*certificate.Resource, error) {
	privKey, err := certcrypto.GeneratePrivateKey(certcrypto.KeyType(keyType))
	if err != nil {
		return nil, fmt.Errorf("generate cert key: %v", err)
	}

	request := certificate.ObtainRequest{
		Domains:    domains,
		Bundle:     true,
		PrivateKey: privKey,
	}

	cert, err := c.Client.Certificate.Obtain(request)
	if err != nil {
		return nil, fmt.Errorf("obtain certificate: %v", err)
	}

	return cert, nil
}

// SetDNSProvider 设置 DNS 验证提供商
func (c *AcmeClient) SetDNSProvider(dnsType, authJSON string) error {
	provider, err := GetDNSProvider(dnsType, authJSON)
	if err != nil {
		return err
	}
	return c.Client.Challenge.SetDNS01Provider(provider,
		dns01.AddDNSTimeout(30*time.Minute),
	)
}

// RenewCertificate 续签证书（重新申请方式）
func (c *AcmeClient) RenewCertificate(domains []string, keyType string) (*certificate.Resource, error) {
	return c.ObtainCertificate(domains, keyType)
}

// EncodePrivateKey 将私钥编码为 PEM
func EncodePrivateKey(priKey crypto.PrivateKey, keyType string) ([]byte, error) {
	var (
		marshal []byte
		block   *pem.Block
		err     error
	)
	switch certcrypto.KeyType(keyType) {
	case KeyEC256, KeyEC384:
		key := priKey.(*ecdsa.PrivateKey)
		marshal, err = x509.MarshalECPrivateKey(key)
		if err != nil {
			return nil, err
		}
		block = &pem.Block{Type: "EC PRIVATE KEY", Bytes: marshal}
	default: // RSA
		key := priKey.(*rsa.PrivateKey)
		marshal = x509.MarshalPKCS1PrivateKey(key)
		block = &pem.Block{Type: "RSA PRIVATE KEY", Bytes: marshal}
	}
	return pem.EncodeToMemory(block), nil
}

// ParsePrivateKey 从 PEM 解析私钥
func ParsePrivateKey(pemStr, keyType string) (crypto.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, fmt.Errorf("invalid PEM block")
	}
	switch certcrypto.KeyType(keyType) {
	case KeyEC256, KeyEC384:
		return x509.ParseECPrivateKey(block.Bytes)
	default:
		return x509.ParsePKCS1PrivateKey(block.Bytes)
	}
}
