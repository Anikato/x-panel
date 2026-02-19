package model

type Website struct {
	BaseModel
	PrimaryDomain string `gorm:"not null;uniqueIndex" json:"primaryDomain"`
	Domains       string `gorm:"type:text" json:"domains"`
	Alias         string `gorm:"not null;uniqueIndex" json:"alias"`
	Type          string `gorm:"not null;default:static" json:"type"`     // static | reverse_proxy
	Status        string `gorm:"not null;default:stopped" json:"status"` // running | stopped

	// Static site
	SiteDir   string `json:"siteDir"`
	IndexFile string `gorm:"default:'index.html index.htm'" json:"indexFile"`

	// Reverse proxy
	ProxyPass string `json:"proxyPass"`
	WebSocket bool   `gorm:"default:false" json:"webSocket"`

	// SSL
	SSLEnable     bool   `gorm:"default:false" json:"sslEnable"`
	CertificateID uint   `json:"certificateID"`
	HttpConfig    string `gorm:"default:'HTTPSRedirect'" json:"httpConfig"` // httpOnly | httpsOnly | HTTPSRedirect | HTTPAlso
	HSTS          bool   `gorm:"default:false" json:"hsts"`
	SSLProtocols  string `gorm:"default:'TLSv1.2 TLSv1.3'" json:"sslProtocols"`

	// Security
	BasicAuth     bool   `gorm:"default:false" json:"basicAuth"`
	BasicUser     string `json:"basicUser"`
	BasicPassword string `json:"-"`

	// Anti-hotlink
	AntiLeech     bool   `gorm:"default:false" json:"antiLeech"`
	LeechReferers string `json:"leechReferers"`

	// Traffic
	LimitRate string `json:"limitRate"`
	LimitConn int    `json:"limitConn"`

	// Rewrite
	Rewrite string `gorm:"type:text" json:"rewrite"`

	// Redirects JSON: [{"source":"/old","target":"/new","type":301}]
	Redirects string `gorm:"type:text" json:"redirects"`

	// Logs
	AccessLog bool `gorm:"default:true" json:"accessLog"`
	ErrorLog  bool `gorm:"default:true" json:"errorLog"`

	// Custom nginx directives
	CustomNginx string `gorm:"type:text" json:"customNginx"`

	DefaultServer bool   `gorm:"default:false" json:"defaultServer"`
	Remark        string `json:"remark"`
}
