package model

import "time"

// TrafficConfig 流量监控配置（每个网卡独立配置）
type TrafficConfig struct {
	BaseModel
	InterfaceName string `gorm:"not null;uniqueIndex" json:"interfaceName"`
	MonthlyLimit  uint64 `gorm:"not null;default:0" json:"monthlyLimit"` // bytes, 0 = unlimited
	ResetDay      int    `gorm:"not null;default:1" json:"resetDay"`     // 1-28
	Enabled       bool   `gorm:"not null;default:true" json:"enabled"`
}

// TrafficHourly 小时粒度流量记录
type TrafficHourly struct {
	ID            uint      `gorm:"primarykey;autoIncrement" json:"id"`
	InterfaceName string    `gorm:"not null;index:idx_traffic_hourly,unique" json:"interfaceName"`
	Timestamp     time.Time `gorm:"not null;index:idx_traffic_hourly,unique;index:idx_traffic_ts" json:"timestamp"`
	BytesSent     uint64    `gorm:"not null;default:0" json:"bytesSent"`
	BytesRecv     uint64    `gorm:"not null;default:0" json:"bytesRecv"`
}

// TrafficSnapshot 网卡计数器快照（每网卡仅保留最新一条）
type TrafficSnapshot struct {
	ID            uint      `gorm:"primarykey;autoIncrement" json:"id"`
	InterfaceName string    `gorm:"not null;uniqueIndex" json:"interfaceName"`
	BytesSent     uint64    `gorm:"not null;default:0" json:"bytesSent"`
	BytesRecv     uint64    `gorm:"not null;default:0" json:"bytesRecv"`
	SampledAt     time.Time `gorm:"not null" json:"sampledAt"`
}
