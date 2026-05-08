package dto

import "time"

// --- 创建 ---

type WebsiteCreate struct {
	PrimaryDomain string `json:"primaryDomain" binding:"required"`
	Alias         string `json:"alias"`
	Domains       string `json:"domains"`
	Type          string `json:"type" binding:"required,oneof=static reverse_proxy"`
	Remark        string `json:"remark"`
	ConfigMode    string `json:"configMode" binding:"omitempty,oneof=managed source"`

	// Listen ports (0 = use nginx default: 80 / 443)
	HttpPort  int `json:"httpPort"`
	HttpsPort int `json:"httpsPort"`

	// Static
	SiteDir string `json:"siteDir"`

	// Reverse proxy
	ProxyPass string `json:"proxyPass"`

	// Source mode logs
	AccessLogPath string `json:"accessLogPath"`
	ErrorLogPath  string `json:"errorLogPath"`
}

// --- 更新 ---

type WebsiteUpdate struct {
	ID            uint   `json:"id" binding:"required"`
	PrimaryDomain string `json:"primaryDomain"`
	Domains       string `json:"domains"`
	SiteDir       string `json:"siteDir"`
	IndexFile     string `json:"indexFile"`

	// Listen ports
	HttpPort  int `json:"httpPort"`
	HttpsPort int `json:"httpsPort"`

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
	AccessLog     bool   `json:"accessLog"`
	ErrorLog      bool   `json:"errorLog"`
	AccessLogPath string `json:"accessLogPath"`
	ErrorLogPath  string `json:"errorLogPath"`

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
	ConfigMode    string    `json:"configMode"`
	Remark        string    `json:"remark"`
	SiteDir       string    `json:"siteDir"`
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

	// Listen ports
	HttpPort  int `json:"httpPort"`
	HttpsPort int `json:"httpsPort"`

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

	AccessLog     bool   `json:"accessLog"`
	ErrorLog      bool   `json:"errorLog"`
	AccessLogPath string `json:"accessLogPath"`
	ErrorLogPath  string `json:"errorLogPath"`

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

type NginxConfBackupInfo struct {
	Name      string    `json:"name"`
	FilePath  string    `json:"filePath"`
	Size      int64     `json:"size"`
	CreatedAt time.Time `json:"createdAt"`
}

type NginxConfBackupReq struct {
	FilePath string `json:"filePath" binding:"required"`
}

type NginxConfRestoreReq struct {
	FilePath   string `json:"filePath" binding:"required"`
	BackupName string `json:"backupName" binding:"required"`
}

type WebsiteHealthCheck struct {
	URL        string `json:"url"`
	OK         bool   `json:"ok"`
	StatusCode int    `json:"statusCode"`
	LatencyMS  int64  `json:"latencyMs"`
	Error      string `json:"error"`
}

type WebsiteHealthResp struct {
	Checks        []WebsiteHealthCheck `json:"checks"`
	CertNotAfter  string               `json:"certNotAfter"`
	CertDaysLeft  int                  `json:"certDaysLeft"`
	CertError     string               `json:"certError"`
	LastCheckedAt time.Time            `json:"lastCheckedAt"`
}

type WebsiteInspectResp struct {
	SiteDirExists bool     `json:"siteDirExists"`
	Readable      bool     `json:"readable"`
	IndexFiles    []string `json:"indexFiles"`
	Issues        []string `json:"issues"`
}

type WebsiteLogPathDetectResp struct {
	Access []string `json:"access"`
	Error  []string `json:"error"`
}

type WebsiteLogAlertReq struct {
	ID        uint   `json:"id" binding:"required"`
	TimeRange string `json:"timeRange"`
}

type WebsiteLogAlert struct {
	Level   string `json:"level"`
	Type    string `json:"type"`
	Message string `json:"message"`
	Count   int64  `json:"count"`
}
