package dto

// VersionInfo 当前版本信息
type VersionInfo struct {
	Version    string `json:"version"`
	CommitHash string `json:"commitHash"`
	BuildTime  string `json:"buildTime"`
	GoVersion  string `json:"goVersion"`
}

// UpgradeCheckReq 检查更新请求
type UpgradeCheckReq struct {
	ReleaseURL string `json:"releaseUrl"`
}

// RemoteVersionInfo 远端版本信息（version.json 格式）
type RemoteVersionInfo struct {
	Version     string `json:"version"`
	ReleaseNote string `json:"releaseNote"`
	PublishDate string `json:"publishDate"`
}

// UpgradeInfo 更新信息响应
type UpgradeInfo struct {
	CurrentVersion string `json:"currentVersion"`
	LatestVersion  string `json:"latestVersion"`
	ReleaseNote    string `json:"releaseNote"`
	HasUpdate      bool   `json:"hasUpdate"`
	DownloadURL    string `json:"downloadUrl"`
	PublishDate    string `json:"publishDate"`
}

// UpgradeReq 执行升级请求
type UpgradeReq struct {
	Version     string `json:"version" binding:"required"`
	DownloadURL string `json:"downloadUrl" binding:"required"`
}
