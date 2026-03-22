package model

import "time"

// XrayNode 代表一个 Xray 入站节点（inbound）
type XrayNode struct {
	BaseModel
	Name      string `gorm:"not null" json:"name"`
	Protocol  string `gorm:"not null;default:'vless'" json:"protocol"` // vless | vmess | trojan
	Port      int    `gorm:"not null;uniqueIndex" json:"port"`
	Transport string `gorm:"not null;default:'tcp'" json:"transport"` // tcp | ws | grpc
	Security  string `gorm:"not null;default:'none'" json:"security"` // none | tls | reality

	// TLS/Reality 公共
	Domain string `json:"domain"`
	// TLS
	TLSCert string `gorm:"type:text" json:"-"`
	TLSKey  string `gorm:"type:text" json:"-"`
	// Reality 专用
	RealityPrivateKey  string `gorm:"type:text" json:"-"`
	RealityPublicKey   string `json:"realityPublicKey"`
	RealityShortIds    string `json:"realityShortIds"`    // JSON 数组字符串
	RealityServerNames string `json:"realityServerNames"` // JSON 数组字符串，dest SNI

	// WebSocket 路径
	Path string `gorm:"default:'/'" json:"path"`
	// gRPC serviceName
	ServiceName string `json:"serviceName"`

	Remark  string `json:"remark"`
	Enabled bool   `gorm:"default:true" json:"enabled"`
}

// XrayTrafficDaily 每日流量快照（供历史图表使用）
type XrayTrafficDaily struct {
	BaseModel
	UserID   uint   `gorm:"not null;uniqueIndex:idx_user_date" json:"userId"`
	Date     string `gorm:"not null;uniqueIndex:idx_user_date" json:"date"` // YYYY-MM-DD
	Upload   int64  `gorm:"default:0" json:"upload"`
	Download int64  `gorm:"default:0" json:"download"`
}

// XrayUser 代表一个 Xray 代理用户
type XrayUser struct {
	BaseModel
	NodeID    uint      `gorm:"not null;index" json:"nodeId"`
	Name      string    `gorm:"not null" json:"name"`
	UUID      string    `gorm:"not null;uniqueIndex" json:"uuid"`
	Email     string    `gorm:"not null;uniqueIndex" json:"email"` // 用于流量统计的唯一 key
	Level     int       `gorm:"default:0" json:"level"`
	ExpireAt  *time.Time `json:"expireAt"` // nil = 永不过期
	Enabled   bool      `gorm:"default:true" json:"enabled"`
	Remark    string    `json:"remark"`

	// 流量统计（字节）
	UploadTotal   int64 `gorm:"default:0" json:"uploadTotal"`
	DownloadTotal int64 `gorm:"default:0" json:"downloadTotal"`
}
