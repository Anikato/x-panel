package model

// GostService GOST 服务规则（端口转发 / 中继服务）
type GostService struct {
	BaseModel
	Name           string `gorm:"not null;uniqueIndex" json:"name"`
	Type           string `gorm:"not null" json:"type"`                       // tcp_forward / udp_forward / relay_server
	ListenAddr     string `gorm:"not null" json:"listenAddr"`                 // 监听地址，如 :8080
	TargetAddr     string `json:"targetAddr"`                                 // 转发目标，如 192.168.1.1:80（中继服务为空）
	ListenerType   string `gorm:"not null;default:tcp" json:"listenerType"`   // 传输层：tcp / tls / ws / wss
	AuthUser       string `json:"authUser"`
	AuthPass       string `json:"-"`
	ChainID        uint   `gorm:"default:0" json:"chainID"`
	CertificateID  uint   `gorm:"default:0" json:"certificateID"`            // 关联 X-Panel 证书（tls/wss 时使用）
	CustomCertPath string `json:"customCertPath"`                             // 自定义证书路径（优先级高于 CertificateID）
	CustomKeyPath  string `json:"customKeyPath"`                              // 自定义私钥路径
	EnableStats    bool   `gorm:"default:true" json:"enableStats"`
	Enabled        bool   `gorm:"default:true" json:"enabled"`
	Remark         string `json:"remark"`
}

// GostChain GOST 转发链（链式代理）
type GostChain struct {
	BaseModel
	Name   string `gorm:"not null;uniqueIndex" json:"name"`
	Hops   string `gorm:"type:text" json:"hops"` // JSON: 跳跃点定义，格式与 GOST 原生一致
	Remark string `json:"remark"`
}
