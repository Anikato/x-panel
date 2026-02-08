package version

// 以下变量通过 -ldflags 在编译时注入
var (
	Version   = "dev"                // 语义化版本号，如 v1.0.0
	CommitHash = "unknown"           // Git commit hash
	BuildTime = "unknown"            // 构建时间
	GoVersion = "unknown"            // Go 编译器版本
)

// Info 返回版本信息结构
type Info struct {
	Version    string `json:"version"`
	CommitHash string `json:"commitHash"`
	BuildTime  string `json:"buildTime"`
	GoVersion  string `json:"goVersion"`
}

// Get 获取当前版本信息
func Get() Info {
	return Info{
		Version:    Version,
		CommitHash: CommitHash,
		BuildTime:  BuildTime,
		GoVersion:  GoVersion,
	}
}
