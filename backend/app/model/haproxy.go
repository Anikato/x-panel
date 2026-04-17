package model

// HAProxyLB 负载均衡器（对应 HAProxy frontend + 默认 backend 的一体化抽象）
type HAProxyLB struct {
	BaseModel
	Name             string `gorm:"not null;uniqueIndex" json:"name"`
	Mode             string `gorm:"not null;default:http" json:"mode"` // http / tcp
	Enabled          bool   `gorm:"default:true" json:"enabled"`
	BindAddr         string `gorm:"not null;default:0.0.0.0" json:"bindAddr"`
	BindPort         int    `gorm:"not null" json:"bindPort"`
	EnableSSL        bool   `json:"enableSSL"`
	CertificateID    uint   `gorm:"default:0" json:"certificateID"`
	SSLRedirect      bool   `json:"sslRedirect"`
	DefaultBackendID uint   `gorm:"default:0" json:"defaultBackendID"`
	XForwardedFor    bool   `gorm:"default:true" json:"xForwardedFor"`
	MaxConn          int    `gorm:"default:2000" json:"maxConn"`
	TimeoutConnect   int    `gorm:"default:5" json:"timeoutConnect"`
	TimeoutClient    int    `gorm:"default:30" json:"timeoutClient"`
	TimeoutServer    int    `gorm:"default:30" json:"timeoutServer"`
	Remark           string `json:"remark"`
}

// HAProxyBackend 后端池
type HAProxyBackend struct {
	BaseModel
	Name         string `gorm:"not null;uniqueIndex" json:"name"`
	Mode         string `gorm:"not null;default:http" json:"mode"`        // http / tcp
	Balance      string `gorm:"not null;default:roundrobin" json:"balance"` // roundrobin / leastconn / source / uri
	StickyType   string `gorm:"default:''" json:"stickyType"`             // "" / cookie / source
	StickyName   string `json:"stickyName"`
	HealthType   string `gorm:"default:tcp" json:"healthType"` // tcp / http / mysql / pgsql / redis / ssl-hello / none
	HealthPath   string `json:"healthPath"`
	HealthMethod string `json:"healthMethod"`
	HealthHost   string `json:"healthHost"`
	HealthExpect string `json:"healthExpect"`
	HealthInter  int    `gorm:"default:2000" json:"healthInter"`
	HealthRise   int    `gorm:"default:2" json:"healthRise"`
	HealthFall   int    `gorm:"default:3" json:"healthFall"`
	Remark       string `json:"remark"`

	Servers []HAProxyServer `gorm:"foreignKey:BackendID" json:"servers"`
}

// HAProxyServer 后端成员
type HAProxyServer struct {
	BaseModel
	BackendID uint   `gorm:"not null;index" json:"backendID"`
	Name      string `gorm:"not null" json:"name"`
	Address   string `gorm:"not null" json:"address"`
	Port      int    `gorm:"not null" json:"port"`
	Weight    int    `gorm:"default:100" json:"weight"`
	MaxConn   int    `gorm:"default:0" json:"maxConn"`
	Backup    bool   `json:"backup"`
	Disabled  bool   `json:"disabled"`
	SSL       bool   `json:"ssl"`
	SSLVerify bool   `gorm:"default:false" json:"sslVerify"`
}

// HAProxyACLRule HTTP 路由规则（属于某个 HTTP LB）
type HAProxyACLRule struct {
	BaseModel
	LBID            uint   `gorm:"not null;index" json:"lbID"`
	Priority        int    `gorm:"default:100" json:"priority"`
	MatchType       string `gorm:"not null" json:"matchType"` // host / host_end / path_beg / path_end / path_reg / hdr / src
	MatchHeader     string `json:"matchHeader"`               // when matchType = hdr
	MatchValue      string `gorm:"not null" json:"matchValue"`
	TargetBackendID uint   `gorm:"not null" json:"targetBackendID"`
	Enabled         bool   `gorm:"default:true" json:"enabled"`
	Remark          string `json:"remark"`
}

// HAProxyConfigVersion 配置历史快照
type HAProxyConfigVersion struct {
	BaseModel
	Content  string `gorm:"type:text" json:"content"`
	Reason   string `json:"reason"`
	Success  bool   `json:"success"`
	Operator string `json:"operator"`
}
