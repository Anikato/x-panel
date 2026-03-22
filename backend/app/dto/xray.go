package dto

import "time"

// ============================================================
// 传输方式子配置（NetworkSettings）
// ============================================================

// XrayRawSettings RAW(TCP) 传输配置
type XrayRawSettings struct {
	// none | http（HTTP 流量伪装）
	HeaderType          string `json:"headerType"`
	// 是否接受 Proxy Protocol（nginx 透传真实客户端 IP）
	AcceptProxyProtocol bool   `json:"acceptProxyProtocol"`
}

// XrayWSSettings WebSocket 传输配置
type XrayWSSettings struct {
	Path                string `json:"path"`                // 路径，如 /ws
	Host                string `json:"host"`                // Host 头（用于 CDN 或 nginx）
	AcceptProxyProtocol bool   `json:"acceptProxyProtocol"` // 接受 Proxy Protocol
}

// XrayGRPCSettings gRPC 传输配置
type XrayGRPCSettings struct {
	ServiceName         string `json:"serviceName"`         // gRPC 服务名
	MultiMode           bool   `json:"multiMode"`           // 多路复用
	IdleTimeout         int    `json:"idleTimeout"`         // 空闲超时（秒），默认 60
	HealthCheckTimeout  int    `json:"healthCheckTimeout"`  // 健康检查超时（秒），默认 20
	PermitWithoutStream bool   `json:"permitWithoutStream"` // 允许无流时保持 keepalive
	InitialWindowsSize  int    `json:"initialWindowsSize"`  // 初始窗口大小（字节），0=默认
}

// XrayXHTTPSettings XHTTP(SplitHTTP) 传输配置
// 文档：https://github.com/XTLS/Xray-core/discussions/4113
type XrayXHTTPSettings struct {
	Host                 string `json:"host"`                 // Host 头
	Path                 string `json:"path"`                 // 路径，如 /xhttp
	Mode                 string `json:"mode"`                 // auto | packet-up | stream-up | stream-one
	NoSSEHeader          bool   `json:"noSSEHeader"`          // 不发送 SSE Content-Type 头
	XPaddingBytes        string `json:"xPaddingBytes"`        // padding 范围，如 "100-1000"
	ScStreamUpServerSecs string `json:"scStreamUpServerSecs"` // 服务端 stream-up 持续秒数，如 "20-80"
	ScMaxBufferedPosts   int    `json:"scMaxBufferedPosts"`   // 最大缓冲 POST 数量
}

// XrayHTTPUpgradeSettings HTTPUpgrade 传输配置
type XrayHTTPUpgradeSettings struct {
	Path                string `json:"path"`                // 路径
	Host                string `json:"host"`                // Host 头
	AcceptProxyProtocol bool   `json:"acceptProxyProtocol"` // 接受 Proxy Protocol
}

// ============================================================
// 安全方式子配置（SecuritySettings）
// ============================================================

// XrayTLSSettings TLS 安全配置
type XrayTLSSettings struct {
	ServerName       string   `json:"serverName"`       // SNI，留空用连接地址
	CertFile         string   `json:"certFile"`         // 证书文件路径（fullchain）
	KeyFile          string   `json:"keyFile"`          // 私钥文件路径
	ALPN             []string `json:"alpn"`             // 如 ["h2","http/1.1"]
	Fingerprint      string   `json:"fingerprint"`      // uTLS 指纹：chrome/firefox/safari/ios/android/edge/360/qq/random/randomized
	MinVersion       string   `json:"minVersion"`       // 最低 TLS 版本：1.0/1.1/1.2/1.3
	RejectUnknownSni bool     `json:"rejectUnknownSni"` // SNI 不匹配则拒绝
}

// XrayRealitySettings Reality 安全配置
type XrayRealitySettings struct {
	PrivateKey  string   `json:"privateKey"`  // 服务端私钥（x25519）
	PublicKey   string   `json:"publicKey"`   // 对应公钥（提供给客户端）
	ShortIds    []string `json:"shortIds"`    // ShortId 列表，如 ["abc123de"]
	ServerNames []string `json:"serverNames"` // 伪装目标域名列表，如 ["www.apple.com"]
	Dest        string   `json:"dest"`        // 转发目标，如 "www.apple.com:443"
	Fingerprint string   `json:"fingerprint"` // 浏览器指纹：chrome/firefox/safari/ios/android/edge/360/qq
	SpiderX     string   `json:"spiderX"`     // 爬虫路径（可选）
	Xver        int      `json:"xver"`        // Proxy Protocol 版本，0=不启用
	Show        bool     `json:"show"`        // 调试模式，生产环境为 false
}

// ============================================================
// Node DTOs
// ============================================================

type XrayNodeCreate struct {
	Name       string `json:"name" binding:"required"`
	Protocol   string `json:"protocol" binding:"required,oneof=vless vmess trojan shadowsocks"`
	ListenAddr string `json:"listenAddr"` // 默认 0.0.0.0
	Port       int    `json:"port" binding:"required,min=1,max=65535"`
	Network    string `json:"network" binding:"required,oneof=raw tcp ws grpc xhttp httpupgrade"`
	Security   string `json:"security" binding:"required,oneof=none tls reality"`

	// VLESS flow，仅 VLESS+TCP(RAW)+TLS/Reality 时有效
	Flow string `json:"flow"` // "" | "xtls-rprx-vision"

	// 流量探测
	SniffEnabled     bool     `json:"sniffEnabled"`
	SniffDestOverride []string `json:"sniffDestOverride"`

	// 传输方式配置（只填对应 network 的那一项）
	RawSettings         *XrayRawSettings         `json:"rawSettings,omitempty"`
	WSSettings          *XrayWSSettings          `json:"wsSettings,omitempty"`
	GRPCSettings        *XrayGRPCSettings        `json:"grpcSettings,omitempty"`
	XHTTPSettings       *XrayXHTTPSettings       `json:"xhttpSettings,omitempty"`
	HTTPUpgradeSettings *XrayHTTPUpgradeSettings `json:"httpUpgradeSettings,omitempty"`

	// 安全配置（只填对应 security 的那一项）
	TLSSettings     *XrayTLSSettings     `json:"tlsSettings,omitempty"`
	RealitySettings *XrayRealitySettings `json:"realitySettings,omitempty"`

	Remark string `json:"remark"`
}

type XrayNodeUpdate struct {
	ID         uint   `json:"id" binding:"required"`
	Name       string `json:"name" binding:"required"`
	ListenAddr string `json:"listenAddr"`
	Network    string `json:"network" binding:"required,oneof=raw tcp ws grpc xhttp httpupgrade"`
	Security   string `json:"security" binding:"required,oneof=none tls reality"`
	Flow       string `json:"flow"`

	SniffEnabled     bool     `json:"sniffEnabled"`
	SniffDestOverride []string `json:"sniffDestOverride"`

	RawSettings         *XrayRawSettings         `json:"rawSettings,omitempty"`
	WSSettings          *XrayWSSettings          `json:"wsSettings,omitempty"`
	GRPCSettings        *XrayGRPCSettings        `json:"grpcSettings,omitempty"`
	XHTTPSettings       *XrayXHTTPSettings       `json:"xhttpSettings,omitempty"`
	HTTPUpgradeSettings *XrayHTTPUpgradeSettings `json:"httpUpgradeSettings,omitempty"`

	TLSSettings     *XrayTLSSettings     `json:"tlsSettings,omitempty"`
	RealitySettings *XrayRealitySettings `json:"realitySettings,omitempty"`

	Remark  string `json:"remark"`
	Enabled bool   `json:"enabled"`
}

type XrayNodeResponse struct {
	ID         uint      `json:"id"`
	Name       string    `json:"name"`
	Protocol   string    `json:"protocol"`
	ListenAddr string    `json:"listenAddr"`
	Port       int       `json:"port"`
	Network    string    `json:"network"`
	Security   string    `json:"security"`
	Flow       string    `json:"flow"`

	SniffEnabled     bool     `json:"sniffEnabled"`
	SniffDestOverride []string `json:"sniffDestOverride"`

	// 解析后的子配置，方便前端展示/编辑
	RawSettings         *XrayRawSettings         `json:"rawSettings,omitempty"`
	WSSettings          *XrayWSSettings          `json:"wsSettings,omitempty"`
	GRPCSettings        *XrayGRPCSettings        `json:"grpcSettings,omitempty"`
	XHTTPSettings       *XrayXHTTPSettings       `json:"xhttpSettings,omitempty"`
	HTTPUpgradeSettings *XrayHTTPUpgradeSettings `json:"httpUpgradeSettings,omitempty"`

	TLSSettings     *XrayTLSSettings     `json:"tlsSettings,omitempty"`
	RealitySettings *XrayRealitySettings `json:"realitySettings,omitempty"`

	Remark    string    `json:"remark"`
	Enabled   bool      `json:"enabled"`
	UserCount int64     `json:"userCount"`
	CreatedAt time.Time `json:"createdAt"`
}

// ============================================================
// User DTOs
// ============================================================

type XrayUserCreate struct {
	NodeID   uint       `json:"nodeId" binding:"required"`
	Name     string     `json:"name" binding:"required"`
	UUID     string     `json:"uuid"`      // 留空则自动生成
	Flow     string     `json:"flow"`      // 留空则继承节点 flow
	Level    int        `json:"level"`
	ExpireAt *time.Time `json:"expireAt"`
	Remark   string     `json:"remark"`
}

type XrayUserUpdate struct {
	ID       uint       `json:"id" binding:"required"`
	Name     string     `json:"name" binding:"required"`
	Flow     string     `json:"flow"`
	Level    int        `json:"level"`
	ExpireAt *time.Time `json:"expireAt"`
	Enabled  bool       `json:"enabled"`
	Remark   string     `json:"remark"`
}

type XrayUserSearch struct {
	NodeID   uint `json:"nodeId"`
	Page     int  `json:"page" binding:"required,min=1"`
	PageSize int  `json:"pageSize" binding:"required,min=1,max=100"`
}

type XrayUserResponse struct {
	ID            uint       `json:"id"`
	NodeID        uint       `json:"nodeId"`
	NodeName      string     `json:"nodeName"`
	Name          string     `json:"name"`
	UUID          string     `json:"uuid"`
	Email         string     `json:"email"`
	Flow          string     `json:"flow"`
	Level         int        `json:"level"`
	ExpireAt      *time.Time `json:"expireAt"`
	Enabled       bool       `json:"enabled"`
	Remark        string     `json:"remark"`
	UploadTotal   int64      `json:"uploadTotal"`
	DownloadTotal int64      `json:"downloadTotal"`
	CreatedAt     time.Time  `json:"createdAt"`
}

// ============================================================
// 状态 & 工具 DTOs
// ============================================================

type XrayStatusResponse struct {
	Installed  bool   `json:"installed"`
	Running    bool   `json:"running"`
	Version    string `json:"version"`
	ConfigPath string `json:"configPath"`
	BinPath    string `json:"binPath"`
}

type XrayInstallStatus struct {
	Running bool   `json:"running"`
	Log     string `json:"log"`
}

type XrayGenerateKeyResponse struct {
	PrivateKey string `json:"privateKey"`
	PublicKey  string `json:"publicKey"`
}

type XrayShareLinkResponse struct {
	Link string `json:"link"`
}

type XrayTrafficDaily struct {
	Date     string `json:"date"`
	Upload   int64  `json:"upload"`
	Download int64  `json:"download"`
}
