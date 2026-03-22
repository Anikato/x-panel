package model

import "time"

// XrayNode Xray 入站节点（inbound）配置
// NetworkSettings 和 SecuritySettings 以 JSON 字符串存储，前端透传各自的子配置对象
type XrayNode struct {
	BaseModel
	Name        string `gorm:"not null" json:"name"`
	Protocol    string `gorm:"not null;default:'vless'" json:"protocol"` // vless | vmess | trojan | shadowsocks
	ListenAddr  string `gorm:"default:'0.0.0.0'" json:"listenAddr"`      // 0.0.0.0 | 127.0.0.1
	Port        int    `gorm:"not null;uniqueIndex" json:"port"`

	// 传输方式：raw(tcp) | ws | grpc | xhttp | httpupgrade
	// v24.9.30 后 TCP 更名为 RAW，但两者互为别名
	Network string `gorm:"not null;default:'raw'" json:"network"`

	// 加密方式：none | tls | reality
	Security string `gorm:"not null;default:'none'" json:"security"`

	// 传输方式专属配置（JSON），格式见各 transport 文档
	NetworkSettings string `gorm:"type:text;default:'{}'" json:"networkSettings"`

	// 安全专属配置（JSON），TLS/Reality 分别有不同字段
	SecuritySettings string `gorm:"type:text;default:'{}'" json:"securitySettings"`

	// VLESS flow：""（普通 TLS）| "xtls-rprx-vision"（仅 TCP+TLS/Reality）
	Flow string `gorm:"default:''" json:"flow"`

	// Shadowsocks 专属
	SSMethod   string `gorm:"default:'aes-256-gcm'" json:"ssMethod"`
	SSPassword string `gorm:"default:''" json:"ssPassword"`

	// Fallbacks（VLESS/Trojan TCP 模式，JSON 数组）
	Fallbacks string `gorm:"type:text;default:'[]'" json:"fallbacks"`

	// 流量探测
	SniffEnabled      bool   `gorm:"default:true" json:"sniffEnabled"`
	SniffDestOverride string `gorm:"type:text;default:'[\"http\",\"tls\"]'" json:"sniffDestOverride"` // JSON 数组
	SniffMetadataOnly bool   `gorm:"default:false" json:"sniffMetadataOnly"`

	Remark  string `json:"remark"`
	Enabled bool   `gorm:"default:true" json:"enabled"`

	// 出站标签：指定此节点流量走哪个出站，空 = direct
	OutboundTag string `gorm:"default:''" json:"outboundTag"`
}

// XrayOutbound Xray 出站代理配置
type XrayOutbound struct {
	BaseModel
	Name     string `gorm:"not null" json:"name"`
	Tag      string `gorm:"not null;uniqueIndex" json:"tag"`
	Protocol string `gorm:"not null;default:'freedom'" json:"protocol"` // freedom|socks|http|shadowsocks|vmess|vless|trojan
	Settings string `gorm:"type:text;default:'{}'" json:"settings"`     // JSON，各协议格式不同
	Enabled  bool   `gorm:"default:true" json:"enabled"`
	Remark   string `json:"remark"`
}

// XrayUser Xray 代理用户
type XrayUser struct {
	BaseModel
	NodeID uint   `gorm:"not null;index" json:"nodeId"`
	Name   string `gorm:"not null" json:"name"`
	UUID   string `gorm:"not null;uniqueIndex" json:"uuid"`
	Email  string `gorm:"not null;uniqueIndex" json:"email"` // 流量统计唯一 key

	Level int    `gorm:"default:0" json:"level"`
	Flow  string `gorm:"default:''" json:"flow"` // 单独覆盖节点 flow，留空则继承节点默认

	ExpireAt *time.Time `json:"expireAt"` // nil = 永不过期
	Enabled  bool       `gorm:"default:true" json:"enabled"`
	Remark   string     `json:"remark"`

	// 累计流量（字节），由 SyncTraffic cron 更新
	UploadTotal   int64 `gorm:"default:0" json:"uploadTotal"`
	DownloadTotal int64 `gorm:"default:0" json:"downloadTotal"`
}

// XrayTrafficDaily 每日流量快照（供历史图表使用）
type XrayTrafficDaily struct {
	BaseModel
	UserID   uint   `gorm:"not null;uniqueIndex:idx_xray_user_date" json:"userId"`
	Date     string `gorm:"not null;uniqueIndex:idx_xray_user_date" json:"date"` // YYYY-MM-DD
	Upload   int64  `gorm:"default:0" json:"upload"`
	Download int64  `gorm:"default:0" json:"download"`
}
