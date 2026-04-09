package model

import "time"

// CertSource 证书源：从远程证书服务器拉取证书的配置
type CertSource struct {
	BaseModel
	Name            string `gorm:"not null" json:"name"`
	ServerAddr      string `gorm:"not null" json:"serverAddr"`
	Token           string `gorm:"not null" json:"-"`
	SyncInterval    int    `gorm:"not null;default:360" json:"syncInterval"` // minutes, 0=manual only
	PostSyncCommand string `json:"postSyncCommand"`
	ConflictPolicy  string `gorm:"not null;default:skip" json:"conflictPolicy"` // skip | overwrite
	Enabled         bool   `gorm:"not null;default:true" json:"enabled"`
	LastSyncAt      *time.Time `json:"lastSyncAt"`
	LastSyncStatus  string `json:"lastSyncStatus"` // success | error | ""
	LastSyncMessage string `json:"lastSyncMessage"`
}

// CertSyncLog 证书同步日志
type CertSyncLog struct {
	BaseModel
	SourceID      uint   `gorm:"not null;index" json:"sourceID"`
	SourceName    string `json:"sourceName"`
	Domain        string `json:"domain"`
	Status        string `gorm:"not null" json:"status"` // success | skipped | error
	Message       string `json:"message"`
	CertificateID uint   `json:"certificateID"` // 关联的本地证书 ID（如有）
}
