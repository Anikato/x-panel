package v1

// ApiGroup 聚合所有 API
type ApiGroup struct {
	AuthAPI
	SettingAPI
	LogAPI
	FileAPI
	TerminalAPI
	HostAPI
	CommandAPI
	GroupAPI
	SSLAPI
	ProcessAPI
	MonitorAPI
	SSHManageAPI
	FirewallAPI
	DiskAPI
	NginxAPI
	UpgradeAPI
	WebsiteAPI
	CronjobAPI
	DatabaseAPI
	ContainerAPI
	BackupAPI
	NodeAPI
	TrafficAPI
	GostAPI
	ToolboxAPI
	HostSystemAPI
}

// ApiGroupApp 全局 API 实例
var ApiGroupApp = new(ApiGroup)
