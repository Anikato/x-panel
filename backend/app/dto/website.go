package dto

import "time"

// --- 创建 ---

type WebsiteCreate struct {
	PrimaryDomain string `json:"primaryDomain" binding:"required"`
	Alias         string `json:"alias"`
	Domains       string `json:"domains"`
	Type          string `json:"type" binding:"required,oneof=static reverse_proxy"`
	Remark        string `json:"remark"`

	// Static
	SiteDir string `json:"siteDir"`

	// Reverse proxy
	ProxyPass string `json:"proxyPass"`
}

// --- 更新 ---

type WebsiteUpdate struct {
	ID            uint   `json:"id" binding:"required"`
	PrimaryDomain string `json:"primaryDomain"`
	Domains       string `json:"domains"`
	SiteDir       string `json:"siteDir"`
	IndexFile     string `json:"indexFile"`

	// Reverse proxy
	ProxyPass string `json:"proxyPass"`
	WebSocket bool   `json:"webSocket"`

	// SSL
	SSLEnable     bool   `json:"sslEnable"`
	CertificateID uint   `json:"certificateID"`
	HttpConfig    string `json:"httpConfig"`
	HSTS          bool   `json:"hsts"`
	Http2Enable   bool   `json:"http2Enable"`
	SSLProtocols  string `json:"sslProtocols"`

	// Security
	BasicAuth     bool   `json:"basicAuth"`
	BasicUser     string `json:"basicUser"`
	BasicPassword string `json:"basicPassword"`

	// Anti-hotlink
	AntiLeech     bool   `json:"antiLeech"`
	LeechReferers string `json:"leechReferers"`

	// Traffic
	LimitRate string `json:"limitRate"`
	LimitConn int    `json:"limitConn"`

	// Rewrite / Redirect
	Rewrite   string `json:"rewrite"`
	Redirects string `json:"redirects"`

	// Logs
	AccessLog bool `json:"accessLog"`
	ErrorLog  bool `json:"errorLog"`

	// Performance & Optimization
	GzipEnable        bool `json:"gzipEnable"`
	SecurityHeaders   bool `json:"securityHeaders"`
	StaticCacheEnable bool `json:"staticCacheEnable"`

	// Upstream / Custom
	Upstream      string `json:"upstream"`
	CustomNginx   string `json:"customNginx"`
	DefaultServer bool   `json:"defaultServer"`
	Remark        string `json:"remark"`

	// Config mode
	ConfigMode string `json:"configMode"`
}

// --- 搜索 ---

type WebsiteSearch struct {
	PageInfo
	Info   string `json:"info"`
	Type   string `json:"type"`
	Status string `json:"status"`
}

// --- 响应 ---

type WebsiteInfo struct {
	ID            uint      `json:"id"`
	PrimaryDomain string    `json:"primaryDomain"`
	Domains       string    `json:"domains"`
	Alias         string    `json:"alias"`
	Type          string    `json:"type"`
	Status        string    `json:"status"`
	SSLEnable     bool      `json:"sslEnable"`
	Remark        string    `json:"remark"`
	CreatedAt     time.Time `json:"createdAt"`
}

type WebsiteDetail struct {
	ID            uint   `json:"id"`
	PrimaryDomain string `json:"primaryDomain"`
	Domains       string `json:"domains"`
	Alias         string `json:"alias"`
	Type          string `json:"type"`
	Status        string `json:"status"`

	SiteDir   string `json:"siteDir"`
	IndexFile string `json:"indexFile"`

	ProxyPass string `json:"proxyPass"`
	WebSocket bool   `json:"webSocket"`

	SSLEnable     bool   `json:"sslEnable"`
	CertificateID uint   `json:"certificateID"`
	HttpConfig    string `json:"httpConfig"`
	HSTS          bool   `json:"hsts"`
	Http2Enable   bool   `json:"http2Enable"`
	SSLProtocols  string `json:"sslProtocols"`

	BasicAuth     bool   `json:"basicAuth"`
	BasicUser     string `json:"basicUser"`
	BasicPassword string `json:"basicPassword"`

	AntiLeech     bool   `json:"antiLeech"`
	LeechReferers string `json:"leechReferers"`

	LimitRate string `json:"limitRate"`
	LimitConn int    `json:"limitConn"`

	Rewrite   string `json:"rewrite"`
	Redirects string `json:"redirects"`

	AccessLog bool `json:"accessLog"`
	ErrorLog  bool `json:"errorLog"`

	GzipEnable        bool `json:"gzipEnable"`
	SecurityHeaders   bool `json:"securityHeaders"`
	StaticCacheEnable bool `json:"staticCacheEnable"`

	Upstream      string `json:"upstream"`
	CustomNginx   string `json:"customNginx"`
	DefaultServer bool   `json:"defaultServer"`
	Remark        string `json:"remark"`

	ConfigMode string `json:"configMode"`

	// 额外信息
	CertificateDomain string `json:"certificateDomain"`
	NginxConfig       string `json:"nginxConfig"`
}

// --- 日志查看 ---

type WebsiteLogReq struct {
	ID   uint   `json:"id" binding:"required"`
	Type string `json:"type" binding:"required,oneof=access error"`
	Tail int    `json:"tail"`
}

// --- 源码模式配置编辑 ---

type SiteConfContentReq struct {
	ID uint `json:"id" binding:"required"`
}

type SaveSiteConfReq struct {
	ID      uint   `json:"id" binding:"required"`
	Content string `json:"content" binding:"required"`
}

type SwitchConfigModeReq struct {
	ID   uint   `json:"id" binding:"required"`
	Mode string `json:"mode" binding:"required,oneof=managed source"`
}

// --- Nginx 配置文件 ---

type NginxConfFileInfo struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
}

type NginxConfUpdate struct {
	FilePath string `json:"filePath" binding:"required"`
	Content  string `json:"content" binding:"required"`
}
