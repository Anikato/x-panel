package model

// FirewallForward 本机 nftables 端口转发。不依赖 ufw。
type FirewallForward struct {
	BaseModel
	Protocol   string `gorm:"not null;size:8" json:"protocol"`
	Port       string `gorm:"not null;size:32" json:"port"`
	TargetIP   string `gorm:"not null;size:64" json:"targetIP"`
	TargetPort string `gorm:"size:16" json:"targetPort"`
}
