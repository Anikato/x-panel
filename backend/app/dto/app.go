package dto

// AppSearchReq 应用搜索请求
type AppSearchReq struct {
	PageInfo
	Name string   `json:"name"`
	Type string   `json:"type"`
	Tags []string `json:"tags"`
}

// AppDTO 应用响应
type AppDTO struct {
	ID                   uint     `json:"id"`
	Name                 string   `json:"name"`
	Key                  string   `json:"key"`
	ShortDescZh          string   `json:"shortDescZh"`
	ShortDescEn          string   `json:"shortDescEn"`
	Description          string   `json:"description"`
	Icon                 string   `json:"icon"`
	Type                 string   `json:"type"`
	Status               string   `json:"status"`
	CrossVersionUpdate   bool     `json:"crossVersionUpdate"`
	LimitNum             int      `json:"limitNum"`
	Website              string   `json:"website"`
	Github               string   `json:"github"`
	Document             string   `json:"document"`
	Recommend            int      `json:"recommend"`
	Resource             string   `json:"resource"`
	LastModified         int64    `json:"lastModified"`
	Architectures        []string `json:"architectures"`
	MemoryRequired       int      `json:"memoryRequired"`
	GpuSupport           bool     `json:"gpuSupport"`
	RequiredPanelVersion string   `json:"requiredPanelVersion"`
	Tags                 []string `json:"tags"`
	Versions             []string `json:"versions"`
	InstalledCount       int      `json:"installedCount"`
	CreatedAt            string   `json:"createdAt"`
}

// AppDetailDTO 应用版本详情
type AppDetailDTO struct {
	ID                  uint   `json:"id"`
	AppID               uint   `json:"appId"`
	Version             string `json:"version"`
	Params              string `json:"params"`
	DockerCompose       string `json:"dockerCompose"`
	Status              string `json:"status"`
	LastVersion         string `json:"lastVersion"`
	LastModified        int64  `json:"lastModified"`
	DownloadURL         string `json:"downloadUrl"`
	DownloadCallbackURL string `json:"downloadCallbackUrl"`
	IsUpdate            bool   `json:"isUpdate"`
}

// AppInstallReq 应用安装请求
type AppInstallReq struct {
	Name        string                 `json:"name" binding:"required"`
	AppID       uint                   `json:"appId" binding:"required"`
	AppDetailID uint                   `json:"appDetailId" binding:"required"`
	Params      map[string]interface{} `json:"params"`
}

// AppInstallDTO 已安装应用响应
type AppInstallDTO struct {
	ID            uint   `json:"id"`
	Name          string `json:"name"`
	AppID         uint   `json:"appId"`
	AppKey        string `json:"appKey"`
	AppName       string `json:"appName"`
	AppIcon       string `json:"appIcon"`
	Version       string `json:"version"`
	Status        string `json:"status"`
	Message       string `json:"message"`
	ContainerName string `json:"containerName"`
	HttpPort      int    `json:"httpPort"`
	HttpsPort     int    `json:"httpsPort"`
	WebUI         string `json:"webUi"`
	Favorite      bool   `json:"favorite"`
	SortOrder     int    `json:"sortOrder"`
	InstalledAt   string `json:"installedAt"`
	CanUpdate     bool   `json:"canUpdate"`
	LatestVersion string `json:"latestVersion"`
}

// AppInstallSearchReq 已安装应用搜索请求
type AppInstallSearchReq struct {
	PageInfo
	Name string `json:"name"`
	Type string `json:"type"`
}

// AppOperateReq 应用操作请求
type AppOperateReq struct {
	InstallID uint   `json:"installId" binding:"required"`
	Operation string `json:"operation" binding:"required,oneof=start stop restart"`
}

// AppUninstallReq 应用卸载请求
type AppUninstallReq struct {
	InstallID   uint `json:"installId" binding:"required"`
	DeleteData  bool `json:"deleteData"`
	ForceDelete bool `json:"forceDelete"`
}

// AppUpdateReq 应用更新请求
type AppUpdateReq struct {
	InstallID   uint `json:"installId" binding:"required"`
	AppDetailID uint `json:"appDetailId" binding:"required"`
}

// AppBackupReq 应用备份请求
type AppBackupReq struct {
	InstallID   uint   `json:"installId" binding:"required"`
	BackupName  string `json:"backupName"`
	Description string `json:"description"`
}

// AppRestoreReq 应用恢复请求
type AppRestoreReq struct {
	InstallID uint   `json:"installId" binding:"required"`
	BackupID  uint   `json:"backupId" binding:"required"`
}

// AppImportReq 从备份导入应用请求
type AppImportReq struct {
	Name       string `json:"name" binding:"required"`        // 安装名称
	BackupPath string `json:"backupPath" binding:"required"`  // 备份文件路径
	AppKey     string `json:"appKey"`                         // 应用 key（可选，用于匹配应用商店）
	Version    string `json:"version"`                        // 版本（可选）
}

// AppBackupDTO 应用备份响应
type AppBackupDTO struct {
	ID           uint   `json:"id"`
	AppInstallID uint   `json:"appInstallId"`
	AppName      string `json:"appName"`
	BackupName   string `json:"backupName"`
	BackupPath   string `json:"backupPath"`
	BackupType   string `json:"backupType"`
	Size         int64  `json:"size"`
	SizeStr      string `json:"sizeStr"`
	Checksum     string `json:"checksum"`
	Status       string `json:"status"`
	Message      string `json:"message"`
	CreatedAt    string `json:"createdAt"`
}

// AppSyncReq 应用商店同步请求
type AppSyncReq struct {
	Force bool `json:"force"` // 强制同步
}

// TagDTO 标签响应
type TagDTO struct {
	Key  string `json:"key"`
	Name string `json:"name"`
	Sort int    `json:"sort"`
}
