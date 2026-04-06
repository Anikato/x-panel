package dto

// --- GOST 状态 ---

type GostStatus struct {
	IsInstalled bool   `json:"isInstalled"`
	IsRunning   bool   `json:"isRunning"`
	Version     string `json:"version"`
	APIReady    bool   `json:"apiReady"`
}

type GostInstallReq struct {
	Version string `json:"version"`
}

type GostOperateReq struct {
	Operation string `json:"operation" binding:"required,oneof=start stop restart"`
}

type GostInstallProgress struct {
	Phase   string `json:"phase"`
	Message string `json:"message"`
	Percent int    `json:"percent"`
}

type GostCheckUpdateResp struct {
	CurrentVersion string `json:"currentVersion"`
	LatestVersion  string `json:"latestVersion"`
	HasUpdate      bool   `json:"hasUpdate"`
	ReleaseURL     string `json:"releaseURL"`
}

type GostUpgradeReq struct {
	Version string `json:"version" binding:"required"`
}

// --- GOST Service (端口转发 / 中继) ---

type GostServiceCreate struct {
	Name           string `json:"name" binding:"required"`
	Type           string `json:"type" binding:"required,oneof=tcp_forward udp_forward relay_server"`
	ListenAddr     string `json:"listenAddr" binding:"required"`
	TargetAddr     string `json:"targetAddr"`
	ListenerType   string `json:"listenerType" binding:"required,oneof=tcp tls ws wss"`
	AuthUser       string `json:"authUser"`
	AuthPass       string `json:"authPass"`
	ChainID        uint   `json:"chainID"`
	CertificateID  uint   `json:"certificateID"`
	CustomCertPath string `json:"customCertPath"`
	CustomKeyPath  string `json:"customKeyPath"`
	EnableStats    bool   `json:"enableStats"`
	Remark         string `json:"remark"`
}

type GostServiceUpdate struct {
	ID             uint   `json:"id" binding:"required"`
	Name           string `json:"name" binding:"required"`
	Type           string `json:"type" binding:"required,oneof=tcp_forward udp_forward relay_server"`
	ListenAddr     string `json:"listenAddr" binding:"required"`
	TargetAddr     string `json:"targetAddr"`
	ListenerType   string `json:"listenerType" binding:"required,oneof=tcp tls ws wss"`
	AuthUser       string `json:"authUser"`
	AuthPass       string `json:"authPass"`
	ChainID        uint   `json:"chainID"`
	CertificateID  uint   `json:"certificateID"`
	CustomCertPath string `json:"customCertPath"`
	CustomKeyPath  string `json:"customKeyPath"`
	EnableStats    bool   `json:"enableStats"`
	Remark         string `json:"remark"`
}

type GostServiceSearch struct {
	PageInfo
	Info string `json:"info"`
	Type string `json:"type"`
}

type GostServiceInfo struct {
	ID             uint   `json:"id"`
	Name           string `json:"name"`
	Type           string `json:"type"`
	ListenAddr     string `json:"listenAddr"`
	TargetAddr     string `json:"targetAddr"`
	ListenerType   string `json:"listenerType"`
	AuthUser       string `json:"authUser"`
	ChainID        uint   `json:"chainID"`
	ChainName      string `json:"chainName"`
	CertificateID  uint   `json:"certificateID"`
	CertDomain     string `json:"certDomain"`
	CustomCertPath string `json:"customCertPath"`
	CustomKeyPath  string `json:"customKeyPath"`
	EnableStats    bool   `json:"enableStats"`
	Enabled        bool   `json:"enabled"`
	Remark         string `json:"remark"`
}

type GostServiceToggle struct {
	ID      uint `json:"id" binding:"required"`
	Enabled bool `json:"enabled"`
}

// --- GOST Chain (转发链) ---

type GostChainCreate struct {
	Name   string `json:"name" binding:"required"`
	Hops   string `json:"hops" binding:"required"`
	Remark string `json:"remark"`
}

type GostChainUpdate struct {
	ID     uint   `json:"id" binding:"required"`
	Name   string `json:"name" binding:"required"`
	Hops   string `json:"hops" binding:"required"`
	Remark string `json:"remark"`
}

type GostChainSearch struct {
	PageInfo
	Info string `json:"info"`
}

type GostChainInfo struct {
	ID       uint   `json:"id"`
	Name     string `json:"name"`
	Hops     string `json:"hops"`
	HopCount int    `json:"hopCount"`
	RefCount int64  `json:"refCount"`
	Remark   string `json:"remark"`
}
