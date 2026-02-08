package dto

// SSHInfo SSH 服务配置信息
type SSHInfo struct {
	IsExist                bool   `json:"isExist"`
	IsActive               bool   `json:"isActive"`
	Message                string `json:"message"`
	Port                   string `json:"port"`
	ListenAddress          string `json:"listenAddress"`
	PasswordAuthentication string `json:"passwordAuthentication"`
	PubkeyAuthentication   string `json:"pubkeyAuthentication"`
	PermitRootLogin        string `json:"permitRootLogin"`
	UseDNS                 string `json:"useDNS"`
	AutoStart              bool   `json:"autoStart"`
}

// SSHUpdate SSH 配置更新请求
type SSHUpdate struct {
	Key   string `json:"key" binding:"required"`
	Value string `json:"value" binding:"required"`
}

// SSHOperateReq SSH 服务操作请求（start/stop/restart/enable/disable）
type SSHOperateReq struct {
	Operation string `json:"operation" binding:"required,oneof=start stop restart enable disable"`
}

// SSHLogSearch SSH 登录日志搜索
type SSHLogSearch struct {
	PageInfo
	Status string `json:"status"` // success, failed, all
	Info   string `json:"info"`
}

// SSHLogEntry SSH 登录日志条目
type SSHLogEntry struct {
	Date    string `json:"date"`
	Status  string `json:"status"` // success, failed
	User    string `json:"user"`
	IP      string `json:"ip"`
	Port    string `json:"port"`
	Message string `json:"message"`
}
