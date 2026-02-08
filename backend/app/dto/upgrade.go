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
	ReleaseURL string `json:"releaseUrl"` // 自定义更新源，留空使用默认 GitHub
}

// UpgradeInfo 更新信息响应
type UpgradeInfo struct {
	CurrentVersion string `json:"currentVersion"`
	LatestVersion  string `json:"latestVersion"`
	ReleaseNote    string `json:"releaseNote"`
	HasUpdate      bool   `json:"hasUpdate"`
	DownloadURL    string `json:"downloadUrl"`
	ChecksumURL    string `json:"checksumUrl"`
	PublishDate    string `json:"publishDate"`
}

// UpgradeReq 执行升级请求
type UpgradeReq struct {
	Version     string `json:"version" binding:"required"`
	DownloadURL string `json:"downloadUrl" binding:"required"`
	ChecksumURL string `json:"checksumUrl"`
}

// --------- GitHub Releases API 响应结构 ---------

// GitHubRelease GitHub Release API 响应
type GitHubRelease struct {
	TagName     string        `json:"tag_name"`
	Name        string        `json:"name"`
	Body        string        `json:"body"`
	Draft       bool          `json:"draft"`
	Prerelease  bool          `json:"prerelease"`
	PublishedAt string        `json:"published_at"`
	Assets      []GitHubAsset `json:"assets"`
}

// GitHubAsset GitHub Release 附件
type GitHubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
	ContentType        string `json:"content_type"`
}

// --------- 旧版自建服务器格式（兼容保留） ---------

// RemoteVersionInfo 远端版本信息（version.json 格式）
type RemoteVersionInfo struct {
	Version     string `json:"version"`
	ReleaseNote string `json:"releaseNote"`
	PublishDate string `json:"publishDate"`
}
