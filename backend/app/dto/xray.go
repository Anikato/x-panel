package dto

import "time"

// ==================== Node DTOs ====================

type XrayNodeCreate struct {
	Name               string `json:"name" binding:"required"`
	Protocol           string `json:"protocol" binding:"required,oneof=vless vmess trojan"`
	Port               int    `json:"port" binding:"required,min=1,max=65535"`
	Transport          string `json:"transport" binding:"required,oneof=tcp ws grpc"`
	Security           string `json:"security" binding:"required,oneof=none tls reality"`
	Domain             string `json:"domain"`
	TLSCert            string `json:"tlsCert"`
	TLSKey             string `json:"tlsKey"`
	RealityPrivateKey  string `json:"realityPrivateKey"`
	RealityPublicKey   string `json:"realityPublicKey"`
	RealityShortIds    string `json:"realityShortIds"`
	RealityServerNames string `json:"realityServerNames"`
	Path               string `json:"path"`
	ServiceName        string `json:"serviceName"`
	Remark             string `json:"remark"`
}

type XrayNodeUpdate struct {
	ID                 uint   `json:"id" binding:"required"`
	Name               string `json:"name" binding:"required"`
	Transport          string `json:"transport" binding:"required,oneof=tcp ws grpc"`
	Security           string `json:"security" binding:"required,oneof=none tls reality"`
	Domain             string `json:"domain"`
	TLSCert            string `json:"tlsCert"`
	TLSKey             string `json:"tlsKey"`
	RealityPrivateKey  string `json:"realityPrivateKey"`
	RealityPublicKey   string `json:"realityPublicKey"`
	RealityShortIds    string `json:"realityShortIds"`
	RealityServerNames string `json:"realityServerNames"`
	Path               string `json:"path"`
	ServiceName        string `json:"serviceName"`
	Remark             string `json:"remark"`
	Enabled            bool   `json:"enabled"`
}

type XrayNodeResponse struct {
	ID                 uint      `json:"id"`
	Name               string    `json:"name"`
	Protocol           string    `json:"protocol"`
	Port               int       `json:"port"`
	Transport          string    `json:"transport"`
	Security           string    `json:"security"`
	Domain             string    `json:"domain"`
	RealityPublicKey   string    `json:"realityPublicKey"`
	RealityShortIds    string    `json:"realityShortIds"`
	RealityServerNames string    `json:"realityServerNames"`
	Path               string    `json:"path"`
	ServiceName        string    `json:"serviceName"`
	Remark             string    `json:"remark"`
	Enabled            bool      `json:"enabled"`
	UserCount          int64     `json:"userCount"`
	CreatedAt          time.Time `json:"createdAt"`
}

type XrayNodeSearch struct {
	Page     int `json:"page" binding:"required,min=1"`
	PageSize int `json:"pageSize" binding:"required,min=1,max=100"`
}

// ==================== User DTOs ====================

type XrayUserCreate struct {
	NodeID   uint       `json:"nodeId" binding:"required"`
	Name     string     `json:"name" binding:"required"`
	UUID     string     `json:"uuid"` // 若为空则自动生成
	Level    int        `json:"level"`
	ExpireAt *time.Time `json:"expireAt"`
	Remark   string     `json:"remark"`
}

type XrayUserUpdate struct {
	ID       uint       `json:"id" binding:"required"`
	Name     string     `json:"name" binding:"required"`
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
	Level         int        `json:"level"`
	ExpireAt      *time.Time `json:"expireAt"`
	Enabled       bool       `json:"enabled"`
	Remark        string     `json:"remark"`
	UploadTotal   int64      `json:"uploadTotal"`
	DownloadTotal int64      `json:"downloadTotal"`
	CreatedAt     time.Time  `json:"createdAt"`
}

// ==================== Stats DTOs ====================

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
