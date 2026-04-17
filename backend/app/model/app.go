package model

import "time"

// App 应用商店中的应用
type App struct {
	BaseModel
	Name                 string    `gorm:"not null" json:"name"`
	Key                  string    `gorm:"not null;uniqueIndex" json:"key"`
	ShortDescZh          string    `json:"shortDescZh"`
	ShortDescEn          string    `json:"shortDescEn"`
	Description          string    `json:"description"`                    // JSON 多语言
	Icon                 string    `json:"icon"`
	Type                 string    `gorm:"not null" json:"type"`           // website/database/tool
	Status               string    `gorm:"not null;default:ready" json:"status"`
	Required             string    `json:"required"`                       // JSON 依赖要求
	CrossVersionUpdate   bool      `gorm:"default:false" json:"crossVersionUpdate"`
	LimitNum             int       `gorm:"default:0" json:"limitNum"`      // 0=无限制
	Website              string    `json:"website"`
	Github               string    `json:"github"`
	Document             string    `json:"document"`
	Recommend            int       `gorm:"default:0" json:"recommend"`
	Resource             string    `gorm:"default:remote" json:"resource"` // remote/local/custom
	ReadMe               string    `json:"readMe"`
	LastModified         int64     `json:"lastModified"`
	Architectures        string    `json:"architectures"`                  // JSON: ["amd64","arm64"]
	MemoryRequired       int       `gorm:"default:0" json:"memoryRequired"`
	GpuSupport           bool      `gorm:"default:false" json:"gpuSupport"`
	RequiredPanelVersion string    `json:"requiredPanelVersion"`
	BatchInstallSupport  bool      `gorm:"default:false" json:"batchInstallSupport"`
}

// AppDetail 应用的不同版本
type AppDetail struct {
	BaseModel
	AppID               uint      `gorm:"not null;index" json:"appId"`
	Version             string    `gorm:"not null" json:"version"`
	Params              string    `json:"params"`                         // JSON 安装参数
	DockerCompose       string    `json:"dockerCompose"`
	Status              string    `gorm:"not null;default:ready" json:"status"`
	LastVersion         string    `json:"lastVersion"`
	LastModified        int64     `json:"lastModified"`
	DownloadURL         string    `json:"downloadUrl"`
	DownloadCallbackURL string    `json:"downloadCallbackUrl"`
	IsUpdate            bool      `gorm:"default:false" json:"isUpdate"`
}

// AppInstall 已安装的应用实例
type AppInstall struct {
	BaseModel
	Name          string    `gorm:"not null;uniqueIndex" json:"name"`
	AppID         uint      `gorm:"not null;index" json:"appId"`
	AppDetailID   uint      `gorm:"not null" json:"appDetailId"`
	Version       string    `gorm:"not null" json:"version"`
	Param         string    `json:"param"`                              // JSON 安装参数
	Env           string    `json:"env"`                                // JSON 环境变量（加密）
	DockerCompose string    `json:"dockerCompose"`
	Status        string    `gorm:"not null;default:running" json:"status"` // running/stopped/error/installing
	Description   string    `json:"description"`
	Message       string    `json:"message"`                            // 错误信息
	ContainerName string    `gorm:"not null" json:"containerName"`
	ServiceName   string    `gorm:"not null" json:"serviceName"`
	HttpPort      int       `gorm:"default:0" json:"httpPort"`
	HttpsPort     int       `gorm:"default:0" json:"httpsPort"`
	WebUI         string    `json:"webUi"`
	Favorite      bool      `gorm:"default:false" json:"favorite"`
	SortOrder     int       `gorm:"default:0" json:"sortOrder"`
	InstalledAt   time.Time `gorm:"autoCreateTime" json:"installedAt"`

	// 关联
	App       App       `gorm:"foreignKey:AppID" json:"app,omitempty"`
	AppDetail AppDetail `gorm:"foreignKey:AppDetailID" json:"appDetail,omitempty"`
}

// AppTag 应用标签关联
type AppTag struct {
	BaseModel
	AppID  uint   `gorm:"not null;index" json:"appId"`
	TagKey string `gorm:"not null;index" json:"tagKey"`
}

// Tag 标签
type Tag struct {
	BaseModel
	Key  string `gorm:"not null;uniqueIndex" json:"key"`
	Name string `gorm:"not null" json:"name"`
	Sort int    `gorm:"default:0" json:"sort"`
}

// AppBackupRecord 应用备份记录
type AppBackupRecord struct {
	BaseModel
	AppInstallID uint      `gorm:"not null;index" json:"appInstallId"`
	BackupName   string    `gorm:"not null" json:"backupName"`
	BackupPath   string    `gorm:"not null" json:"backupPath"`
	BackupType   string    `gorm:"default:full" json:"backupType"` // full/incremental
	Size         int64     `gorm:"default:0" json:"size"`
	Checksum     string    `json:"checksum"`
	Metadata     string    `json:"metadata"`                       // JSON
	Status       string    `gorm:"default:success" json:"status"`  // success/failed
	Message      string    `json:"message"`

	// 关联
	AppInstall AppInstall `gorm:"foreignKey:AppInstallID" json:"appInstall,omitempty"`
}

// TableName 指定表名
func (App) TableName() string {
	return "apps"
}

func (AppDetail) TableName() string {
	return "app_details"
}

func (AppInstall) TableName() string {
	return "app_installs"
}

func (AppTag) TableName() string {
	return "app_tags"
}

func (Tag) TableName() string {
	return "tags"
}

func (AppBackupRecord) TableName() string {
	return "app_backup_records"
}
