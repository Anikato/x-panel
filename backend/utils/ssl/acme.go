package ssl

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/go-acme/lego/v4/certcrypto"
	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/challenge/dns01"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/registration"
)

// certRenewMu 防止同一进程内同一域名并发续签（仅粗粒度保护）
var certRenewMu = make(chan struct{}, 5) // 最多 5 个并发续签

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
	// accountURL 不能为空：lego 用它作为 JWS kid，空值会导致 CA 拒绝所有请求
	if accountURL == "" {
		return nil, fmt.Errorf("ACME account URL is empty; please re-register the ACME account to obtain a valid account URL")
	}

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

// ObtainCertificate 申请证书，logWriter 可选，用于将 lego 内部日志重定向到调用方
// 内置自动重试：首次失败后等待 60s 再重试一次（给 DNS 传播留足时间）
func (c *AcmeClient) ObtainCertificate(domains []string, keyType string, logWriter ...io.Writer) (*certificate.Resource, error) {
	if len(logWriter) > 0 && logWriter[0] != nil {
		origOut := log.Writer()
		origFlags := log.Flags()
		origPrefix := log.Prefix()
		log.SetOutput(io.MultiWriter(os.Stderr, logWriter[0]))
		log.SetFlags(log.LstdFlags)
		log.SetPrefix("[lego] ")
		defer func() {
			log.SetOutput(origOut)
			log.SetFlags(origFlags)
			log.SetPrefix(origPrefix)
		}()
	}

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
		// 等待更长时间（60s），给 DNS 记录更多传播时间，复用相同私钥避免消耗配额
		log.Printf("[lego] first attempt failed: %v, retrying after 60s...", err)
		time.Sleep(60 * time.Second)
		cert, err = c.Client.Certificate.Obtain(request)
		if err != nil {
			return nil, fmt.Errorf("obtain certificate: %v", err)
		}
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
		dns01.AddDNSTimeout(5*time.Minute),
		dns01.AddRecursiveNameservers([]string{
			"1.1.1.1:53",
			"8.8.8.8:53",
			"1.0.0.1:53",
			"8.8.4.4:53",
		}),
	)
}

// RenewCertificate 续签证书
// 使用 lego 原生 Renew API（复用旧证书 URL），避免重新申请消耗速率配额
// 若旧证书 URL 为空则 fallback 到重新申请
func (c *AcmeClient) RenewCertificate(domains []string, keyType string, existingCert *certificate.Resource, logWriter ...io.Writer) (*certificate.Resource, error) {
	if len(logWriter) > 0 && logWriter[0] != nil {
		origOut := log.Writer()
		origFlags := log.Flags()
		origPrefix := log.Prefix()
		log.SetOutput(io.MultiWriter(os.Stderr, logWriter[0]))
		log.SetFlags(log.LstdFlags)
		log.SetPrefix("[lego] ")
		defer func() {
			log.SetOutput(origOut)
			log.SetFlags(origFlags)
			log.SetPrefix(origPrefix)
		}()
	}

	// 若有旧证书资源则用原生 Renew API（不重新验证域名所有权，省配额）
	if existingCert != nil && existingCert.CertURL != "" {
		log.Printf("[lego] renewing via existing cert URL: %s", existingCert.CertURL)
		renewed, err := c.Client.Certificate.Renew(*existingCert, true, false, "")
		if err == nil {
			return renewed, nil
		}
		log.Printf("[lego] renew via cert URL failed: %v, falling back to re-obtain", err)
	}

	// Fallback：重新申请（首次申请或 CertURL 过期失效）
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
// 支持 EC（SEC1） / RSA PKCS#1 / PKCS#8（lego 注册账户时的默认格式）三种格式
func ParsePrivateKey(pemStr, keyType string) (crypto.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, fmt.Errorf("invalid PEM block")
	}
	switch certcrypto.KeyType(keyType) {
	case KeyEC256, KeyEC384:
		// EC 私钥：先尝试 SEC1 格式（x509.ParseECPrivateKey），再尝试 PKCS#8
		if key, err := x509.ParseECPrivateKey(block.Bytes); err == nil {
			return key, nil
		}
		// fallback: PKCS#8 包装的 EC 私钥
		key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parse EC private key failed (tried SEC1 and PKCS#8): %v", err)
		}
		if ecKey, ok := key.(*ecdsa.PrivateKey); ok {
			return ecKey, nil
		}
		return nil, fmt.Errorf("PKCS#8 key is not an EC key")
	default:
		// RSA 私钥：lego 注册时用 PKCS#8，手动生成的可能是 PKCS#1；两种都支持
		if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
			if rsaKey, ok := key.(*rsa.PrivateKey); ok {
				return rsaKey, nil
			}
			// PKCS#8 里存的是 EC Key，说明 keyType 与实际不符，仍尝试继续
		}
		// fallback: PKCS#1 格式
		return x509.ParsePKCS1PrivateKey(block.Bytes)
	}
}
