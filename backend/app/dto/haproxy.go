package dto

// --- 状态 / 安装 / 操作 ---

type HAProxyStatus struct {
	IsInstalled   bool   `json:"isInstalled"`
	IsRunning     bool   `json:"isRunning"`
	Version       string `json:"version"`
	ConfigPath    string `json:"configPath"`
	SocketPath    string `json:"socketPath"`
	SocketReady   bool   `json:"socketReady"`
	StatsEnable   bool   `json:"statsEnable"`
	StatsBind     string `json:"statsBind"`
	StatsURI      string `json:"statsURI"`
	StatsUser     string `json:"statsUser"`
	AutoStart     bool   `json:"autoStart"`
}

type HAProxyInstallReq struct{}

type HAProxyOperateReq struct {
	Operation string `json:"operation" binding:"required,oneof=start stop restart reload"`
}

type HAProxyInstallProgress struct {
	Phase   string `json:"phase"`
	Message string `json:"message"`
	Percent int    `json:"percent"`
}

type HAProxyCheckUpdateResp struct {
	CurrentVersion   string `json:"currentVersion"`
	AvailableVersion string `json:"availableVersion"`
	HasUpdate        bool   `json:"hasUpdate"`
}

type HAProxyUpgradeReq struct{}

// --- LB ---

type HAProxyLBCreate struct {
	Name             string `json:"name" binding:"required"`
	Mode             string `json:"mode" binding:"required,oneof=http tcp"`
	BindAddr         string `json:"bindAddr"`
	BindPort         int    `json:"bindPort" binding:"required,min=1,max=65535"`
	EnableSSL        bool   `json:"enableSSL"`
	CertificateID    uint   `json:"certificateID"`
	SSLRedirect      bool   `json:"sslRedirect"`
	DefaultBackendID uint   `json:"defaultBackendID"`
	XForwardedFor    bool   `json:"xForwardedFor"`
	MaxConn          int    `json:"maxConn"`
	TimeoutConnect   int    `json:"timeoutConnect"`
	TimeoutClient    int    `json:"timeoutClient"`
	TimeoutServer    int    `json:"timeoutServer"`
	Remark           string `json:"remark"`
}

type HAProxyLBUpdate struct {
	ID uint `json:"id" binding:"required"`
	HAProxyLBCreate
}

type HAProxyLBSearch struct {
	PageInfo
	Info string `json:"info"`
	Mode string `json:"mode"`
}

type HAProxyLBInfo struct {
	ID               uint   `json:"id"`
	Name             string `json:"name"`
	Mode             string `json:"mode"`
	Enabled          bool   `json:"enabled"`
	BindAddr         string `json:"bindAddr"`
	BindPort         int    `json:"bindPort"`
	EnableSSL        bool   `json:"enableSSL"`
	CertificateID    uint   `json:"certificateID"`
	CertDomain       string `json:"certDomain"`
	SSLRedirect      bool   `json:"sslRedirect"`
	DefaultBackendID uint   `json:"defaultBackendID"`
	DefaultBackend   string `json:"defaultBackend"`
	XForwardedFor    bool   `json:"xForwardedFor"`
	MaxConn          int    `json:"maxConn"`
	TimeoutConnect   int    `json:"timeoutConnect"`
	TimeoutClient    int    `json:"timeoutClient"`
	TimeoutServer    int    `json:"timeoutServer"`
	Remark           string `json:"remark"`
	CurrentConns     uint64 `json:"currentConns"`
	TotalConns       uint64 `json:"totalConns"`
	BytesIn          uint64 `json:"bytesIn"`
	BytesOut         uint64 `json:"bytesOut"`
}

type HAProxyLBToggle struct {
	ID      uint `json:"id" binding:"required"`
	Enabled bool `json:"enabled"`
}

// --- Backend ---

type HAProxyBackendCreate struct {
	Name         string                 `json:"name" binding:"required"`
	Mode         string                 `json:"mode" binding:"required,oneof=http tcp"`
	Balance      string                 `json:"balance" binding:"required,oneof=roundrobin leastconn source uri"`
	StickyType   string                 `json:"stickyType"`
	StickyName   string                 `json:"stickyName"`
	HealthType   string                 `json:"healthType"`
	HealthPath   string                 `json:"healthPath"`
	HealthMethod string                 `json:"healthMethod"`
	HealthHost   string                 `json:"healthHost"`
	HealthExpect string                 `json:"healthExpect"`
	HealthInter  int                    `json:"healthInter"`
	HealthRise   int                    `json:"healthRise"`
	HealthFall   int                    `json:"healthFall"`
	Remark       string                 `json:"remark"`
	Servers      []HAProxyServerCreate  `json:"servers"`
}

type HAProxyBackendUpdate struct {
	ID uint `json:"id" binding:"required"`
	HAProxyBackendCreate
}

type HAProxyBackendSearch struct {
	PageInfo
	Info string `json:"info"`
	Mode string `json:"mode"`
}

type HAProxyBackendInfo struct {
	ID           uint                `json:"id"`
	Name         string              `json:"name"`
	Mode         string              `json:"mode"`
	Balance      string              `json:"balance"`
	StickyType   string              `json:"stickyType"`
	StickyName   string              `json:"stickyName"`
	HealthType   string              `json:"healthType"`
	HealthPath   string              `json:"healthPath"`
	HealthMethod string              `json:"healthMethod"`
	HealthHost   string              `json:"healthHost"`
	HealthExpect string              `json:"healthExpect"`
	HealthInter  int                 `json:"healthInter"`
	HealthRise   int                 `json:"healthRise"`
	HealthFall   int                 `json:"healthFall"`
	Remark       string              `json:"remark"`
	ServerCount  int                 `json:"serverCount"`
	RefCount     int64               `json:"refCount"`
	Servers      []HAProxyServerInfo `json:"servers"`
}

// --- Server ---

type HAProxyServerCreate struct {
	BackendID uint   `json:"backendID"`
	Name      string `json:"name" binding:"required"`
	Address   string `json:"address" binding:"required"`
	Port      int    `json:"port" binding:"required,min=1,max=65535"`
	Weight    int    `json:"weight"`
	MaxConn   int    `json:"maxConn"`
	Backup    bool   `json:"backup"`
	Disabled  bool   `json:"disabled"`
	SSL       bool   `json:"ssl"`
	SSLVerify bool   `json:"sslVerify"`
}

type HAProxyServerUpdate struct {
	ID uint `json:"id" binding:"required"`
	HAProxyServerCreate
}

type HAProxyServerInfo struct {
	ID        uint   `json:"id"`
	BackendID uint   `json:"backendID"`
	Name      string `json:"name"`
	Address   string `json:"address"`
	Port      int    `json:"port"`
	Weight    int    `json:"weight"`
	MaxConn   int    `json:"maxConn"`
	Backup    bool   `json:"backup"`
	Disabled  bool   `json:"disabled"`
	SSL       bool   `json:"ssl"`
	SSLVerify bool   `json:"sslVerify"`
	// 实时
	LiveStatus   string `json:"liveStatus"`   // UP / DOWN / MAINT / ""
	LiveCurConns uint64 `json:"liveCurConns"`
	LiveTotConns uint64 `json:"liveTotConns"`
	LiveBytesIn  uint64 `json:"liveBytesIn"`
	LiveBytesOut uint64 `json:"liveBytesOut"`
}

type HAProxyServerToggleLive struct {
	ID      uint `json:"id" binding:"required"`
	Disable bool `json:"disable"`
}

type HAProxyServerWeightLive struct {
	ID     uint `json:"id" binding:"required"`
	Weight int  `json:"weight" binding:"min=0,max=256"`
}

// --- ACL ---

type HAProxyACLCreate struct {
	LBID            uint   `json:"lbID" binding:"required"`
	Priority        int    `json:"priority"`
	MatchType       string `json:"matchType" binding:"required,oneof=host host_end path_beg path_end path_reg hdr src"`
	MatchHeader     string `json:"matchHeader"`
	MatchValue      string `json:"matchValue" binding:"required"`
	TargetBackendID uint   `json:"targetBackendID" binding:"required"`
	Enabled         bool   `json:"enabled"`
	Remark          string `json:"remark"`
}

type HAProxyACLUpdate struct {
	ID uint `json:"id" binding:"required"`
	HAProxyACLCreate
}

type HAProxyACLSearch struct {
	LBID uint `json:"lbID"`
}

type HAProxyACLInfo struct {
	ID              uint   `json:"id"`
	LBID            uint   `json:"lbID"`
	Priority        int    `json:"priority"`
	MatchType       string `json:"matchType"`
	MatchHeader     string `json:"matchHeader"`
	MatchValue      string `json:"matchValue"`
	TargetBackendID uint   `json:"targetBackendID"`
	TargetBackend   string `json:"targetBackend"`
	Enabled         bool   `json:"enabled"`
	Remark          string `json:"remark"`
}

// --- Stats ---

type HAProxyStatsInfo struct {
	Frontends []HAProxyFrontendStat `json:"frontends"`
	Backends  []HAProxyBackendStat  `json:"backends"`
	Servers   []HAProxyServerStat   `json:"servers"`
}

type HAProxyFrontendStat struct {
	Name         string `json:"name"`
	Status       string `json:"status"`
	CurConns     uint64 `json:"curConns"`
	MaxConns     uint64 `json:"maxConns"`
	TotalConns   uint64 `json:"totalConns"`
	BytesIn      uint64 `json:"bytesIn"`
	BytesOut     uint64 `json:"bytesOut"`
	ReqRate      uint64 `json:"reqRate"`
	TotalReq     uint64 `json:"totalReq"`
}

type HAProxyBackendStat struct {
	Name        string `json:"name"`
	Status      string `json:"status"`
	CurConns    uint64 `json:"curConns"`
	TotalConns  uint64 `json:"totalConns"`
	BytesIn     uint64 `json:"bytesIn"`
	BytesOut    uint64 `json:"bytesOut"`
	ActServers  int    `json:"actServers"`
	BckServers  int    `json:"bckServers"`
	TotalServers int   `json:"totalServers"`
}

type HAProxyServerStat struct {
	Backend    string `json:"backend"`
	Name       string `json:"name"`
	Status     string `json:"status"`
	CurConns   uint64 `json:"curConns"`
	TotalConns uint64 `json:"totalConns"`
	BytesIn    uint64 `json:"bytesIn"`
	BytesOut   uint64 `json:"bytesOut"`
	CheckStatus string `json:"checkStatus"`
	LastChange uint64 `json:"lastChange"`
	Weight     int    `json:"weight"`
}

type HAProxyRuntimeInfo struct {
	Raw string `json:"raw"`
}

// --- 原始配置 / 版本 ---

type HAProxyRawConfig struct {
	Content string `json:"content"`
}

type HAProxyConfigTestReq struct {
	Content string `json:"content" binding:"required"`
}

type HAProxyConfigTestResp struct {
	Valid  bool   `json:"valid"`
	Output string `json:"output"`
}

type HAProxyConfigVersionInfo struct {
	ID        uint   `json:"id"`
	Version   string `json:"version"`
	Reason    string `json:"reason"`
	Operator  string `json:"operator"`
	Success   bool   `json:"success"`
	CreatedAt string `json:"createdAt"`
}

type HAProxyConfigRollbackReq struct {
	ID uint `json:"id" binding:"required"`
}
