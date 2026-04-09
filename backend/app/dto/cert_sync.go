package dto

import "time"

// --- 证书源 CRUD ---

type CertSourceCreate struct {
	Name            string `json:"name" binding:"required"`
	ServerAddr      string `json:"serverAddr" binding:"required"`
	Token           string `json:"token" binding:"required"`
	SyncInterval    int    `json:"syncInterval"`
	PostSyncCommand string `json:"postSyncCommand"`
	Enabled         bool   `json:"enabled"`
}

type CertSourceUpdate struct {
	ID              uint   `json:"id" binding:"required"`
	Name            string `json:"name" binding:"required"`
	ServerAddr      string `json:"serverAddr" binding:"required"`
	Token           string `json:"token"`
	SyncInterval    int    `json:"syncInterval"`
	PostSyncCommand string `json:"postSyncCommand"`
	Enabled         bool   `json:"enabled"`
}

type CertSourceInfo struct {
	ID              uint       `json:"id"`
	Name            string     `json:"name"`
	ServerAddr      string     `json:"serverAddr"`
	SyncInterval    int        `json:"syncInterval"`
	PostSyncCommand string     `json:"postSyncCommand"`
	Enabled         bool       `json:"enabled"`
	LastSyncAt      *time.Time `json:"lastSyncAt"`
	LastSyncStatus  string     `json:"lastSyncStatus"`
	LastSyncMessage string     `json:"lastSyncMessage"`
	CreatedAt       time.Time  `json:"createdAt"`
}

// --- 证书服务端暴露给远程拉取的结构 ---

type CertServerItem struct {
	PrimaryDomain string    `json:"primaryDomain"`
	Domains       string    `json:"domains"`
	Pem           string    `json:"pem"`
	PrivateKey    string    `json:"privateKey"`
	ExpireDate    time.Time `json:"expireDate"`
	StartDate     time.Time `json:"startDate"`
	KeyType       string    `json:"keyType"`
}

// --- 同步日志 ---

type CertSyncLogInfo struct {
	ID            uint      `json:"id"`
	SourceID      uint      `json:"sourceID"`
	SourceName    string    `json:"sourceName"`
	Domain        string    `json:"domain"`
	Status        string    `json:"status"`
	Message       string    `json:"message"`
	CertificateID uint      `json:"certificateID"`
	CreatedAt     time.Time `json:"createdAt"`
}

type SearchCertSyncLogReq struct {
	PageInfo
	SourceID uint `json:"sourceID"`
}

// --- 证书服务设置 ---

type CertServerSetting struct {
	Enabled bool   `json:"enabled"`
	Token   string `json:"token"`
}
