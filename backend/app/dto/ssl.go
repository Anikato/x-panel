package dto

import "time"

// --- ACME 账户 ---

type AcmeAccountCreate struct {
	Email    string `json:"email" binding:"required,email"`
	Type     string `json:"type" binding:"required,oneof=letsencrypt zerossl buypass google custom"`
	KeyType  string `json:"keyType" binding:"required"`
	CaDirURL string `json:"caDirURL"`
}

type AcmeAccountInfo struct {
	ID       uint   `json:"id"`
	Email    string `json:"email"`
	URL      string `json:"url"`
	Type     string `json:"type"`
	KeyType  string `json:"keyType"`
	CaDirURL string `json:"caDirURL"`
}

// --- DNS 账户 ---

type DnsAccountCreate struct {
	Name          string            `json:"name" binding:"required"`
	Type          string            `json:"type" binding:"required"`
	Authorization map[string]string `json:"authorization" binding:"required"`
}

type DnsAccountUpdate struct {
	ID            uint              `json:"id" binding:"required"`
	Name          string            `json:"name" binding:"required"`
	Type          string            `json:"type" binding:"required"`
	Authorization map[string]string `json:"authorization" binding:"required"`
}

type DnsAccountInfo struct {
	ID            uint              `json:"id"`
	Name          string            `json:"name"`
	Type          string            `json:"type"`
	Authorization map[string]string `json:"authorization"`
}

// --- 证书 ---

type CertificateCreate struct {
	PrimaryDomain string `json:"primaryDomain" binding:"required"`
	OtherDomains  string `json:"otherDomains"`
	Provider      string `json:"provider" binding:"required,oneof=dns http manual"`
	AcmeAccountID uint   `json:"acmeAccountID"`
	DnsAccountID  uint   `json:"dnsAccountID"`
	KeyType       string `json:"keyType"`
	AutoRenew     bool   `json:"autoRenew"`
	Description   string `json:"description"`
	Apply         bool   `json:"apply"` // 创建后立即申请
}

type CertificateUpdate struct {
	ID            uint   `json:"id" binding:"required"`
	AutoRenew     bool   `json:"autoRenew"`
	Description   string `json:"description"`
	PrimaryDomain string `json:"primaryDomain"`
	OtherDomains  string `json:"otherDomains"`
}

type CertificateUpload struct {
	PrivateKey  string `json:"privateKey" binding:"required"`
	Certificate string `json:"certificate" binding:"required"`
	Description string `json:"description"`
}

type CertificateInfo struct {
	ID            uint      `json:"id"`
	PrimaryDomain string    `json:"primaryDomain"`
	Domains       string    `json:"domains"`
	Provider      string    `json:"provider"`
	Type          string    `json:"type"`
	AcmeAccountID uint      `json:"acmeAccountID"`
	DnsAccountID  uint      `json:"dnsAccountID"`
	KeyType       string    `json:"keyType"`
	AutoRenew     bool      `json:"autoRenew"`
	ExpireDate    time.Time `json:"expireDate"`
	StartDate     time.Time `json:"startDate"`
	Status        string    `json:"status"`
	Message       string    `json:"message"`
	Description   string    `json:"description"`
	CertURL       string    `json:"certURL"`
	Organization  string    `json:"organization"`
	// 关联
	AcmeAccountEmail string `json:"acmeAccountEmail"`
	DnsAccountName   string `json:"dnsAccountName"`
}

type CertificateDetail struct {
	CertificateInfo
	Pem        string `json:"pem"`
	PrivateKey string `json:"privateKey"`
	FilePath   string `json:"filePath"`
}

type SearchCertReq struct {
	PageInfo
	Info string `json:"info"`
}

// --- 导入导出 ---

type AccountExport struct {
	AcmeAccounts []AcmeAccountExportItem `json:"acmeAccounts"`
	DnsAccounts  []DnsAccountExportItem  `json:"dnsAccounts"`
}

type AcmeAccountExportItem struct {
	Email      string `json:"email"`
	Type       string `json:"type"`
	KeyType    string `json:"keyType"`
	PrivateKey string `json:"privateKey"`
	URL        string `json:"url"`
	CaDirURL   string `json:"caDirURL"`
	EabKid     string `json:"eabKid"`
	EabHmacKey string `json:"eabHmacKey"`
}

type DnsAccountExportItem struct {
	Name          string            `json:"name"`
	Type          string            `json:"type"`
	Authorization map[string]string `json:"authorization"`
}

// --- SSL 路径设置 ---

type SSLDirUpdate struct {
	Dir string `json:"dir" binding:"required"`
}
