package model

import "time"

// AcmeAccount ACME 证书颁发机构账户
type AcmeAccount struct {
	BaseModel
	Email      string `gorm:"not null" json:"email"`
	URL        string `json:"url"`
	PrivateKey string `gorm:"type:text" json:"-"`
	Type       string `gorm:"not null;default:letsencrypt" json:"type"` // letsencrypt | zerossl | buypass | google | custom
	KeyType    string `gorm:"not null;default:2048" json:"keyType"`     // P256 | P384 | 2048 | 3072 | 4096
	EabKid     string `json:"eabKid"`
	EabHmacKey string `json:"eabHmacKey"`
	CaDirURL   string `json:"caDirURL"`
}

// DnsAccount DNS 提供商账户
type DnsAccount struct {
	BaseModel
	Name          string `gorm:"not null" json:"name"`
	Type          string `gorm:"not null" json:"type"` // CloudFlare | AliYun | DnsPod | TencentCloud | NameSilo | GoDaddy | HuaweiCloud
	Authorization string `gorm:"type:text;not null" json:"-"`
}

// Certificate SSL 证书
type Certificate struct {
	BaseModel
	PrimaryDomain string    `gorm:"not null" json:"primaryDomain"`
	Domains       string    `json:"domains"`
	Provider      string    `gorm:"not null" json:"provider"` // dns | http | manual
	Type          string    `gorm:"not null;default:autoApply" json:"type"` // autoApply | upload
	AcmeAccountID uint      `json:"acmeAccountID"`
	DnsAccountID  uint      `json:"dnsAccountID"`
	KeyType       string    `gorm:"not null;default:2048" json:"keyType"`
	Pem           string    `gorm:"type:text" json:"-"`
	PrivateKey    string    `gorm:"type:text" json:"-"`
	CertURL       string    `json:"certURL"`
	AutoRenew     bool      `gorm:"default:true" json:"autoRenew"`
	ExpireDate    time.Time `json:"expireDate"`
	StartDate     time.Time `json:"startDate"`
	Status        string    `gorm:"default:ready" json:"status"` // ready | applying | applied | error
	Message       string    `json:"message"`
	Description   string    `json:"description"`

	AcmeAccount AcmeAccount `json:"acmeAccount" gorm:"-:migration"`
	DnsAccount  DnsAccount  `json:"dnsAccount" gorm:"-:migration"`
}
