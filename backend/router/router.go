package router

import (
	"io/fs"
	"net/http"

	v1 "xpanel/app/api/v1"
	"xpanel/app/version"
	"xpanel/cmd/server/web"
	"xpanel/middleware"

	"github.com/gin-gonic/gin"
)

// Setup 初始化路由
func Setup(mode string) *gin.Engine {
	gin.SetMode(mode)
	r := gin.New()

	r.Use(gin.Recovery())
	r.Use(middleware.CORS())
	r.Use(middleware.SecurityEntrance())

	api := v1.ApiGroupApp

	// 公开路由
	publicGroup := r.Group("/api/v1")
	{
		publicGroup.GET("/auth/setting", api.GetLoginSetting)
		publicGroup.GET("/auth/is-init", api.CheckIsInitialized)
		publicGroup.POST("/auth/init", api.InitUser)
		publicGroup.POST("/auth/login", api.Login)
		publicGroup.GET("/auth/captcha", api.GetCaptcha)

		// 版本信息（公开，无需认证）
		publicGroup.GET("/version", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"code": 200, "data": version.Get()})
		})
	}

	// 嵌入的前端静态文件（生产模式）
	setupFrontend(r)

	// 私有路由
	privateGroup := r.Group("/api/v1")
	privateGroup.Use(middleware.JWTAuth())
	privateGroup.Use(middleware.NodeProxy())
	privateGroup.Use(middleware.OperationLog())
	{
		// 认证
		privateGroup.POST("/auth/logout", api.Logout)
		privateGroup.POST("/auth/password", api.UpdatePassword)

		// 设置
		privateGroup.GET("/settings", api.GetSettingInfo)
		privateGroup.POST("/settings/update", api.Update)
		privateGroup.POST("/settings/port/update", api.UpdatePort)
		privateGroup.POST("/settings/proxy/test", api.TestProxy)
		privateGroup.POST("/settings/reboot", api.RebootServer)
		privateGroup.POST("/settings/shutdown", api.ShutdownServer)
		privateGroup.POST("/settings/restart-panel", api.RestartPanel)

		// 日志
		privateGroup.POST("/logs/login", api.PageLoginLog)
		privateGroup.POST("/logs/operation", api.PageOperationLog)
		privateGroup.POST("/logs/login/clean", api.CleanLoginLog)
		privateGroup.POST("/logs/operation/clean", api.CleanOperationLog)

		// 文件管理
		privateGroup.POST("/files/search", api.ListFiles)
		privateGroup.POST("/files", api.CreateFile)
		privateGroup.POST("/files/del", api.DeleteFile)
		privateGroup.POST("/files/batch-del", api.BatchDeleteFile)
		privateGroup.POST("/files/rename", api.RenameFile)
		privateGroup.POST("/files/move", api.MoveFile)
		privateGroup.POST("/files/content", api.GetFileContent)
		privateGroup.POST("/files/save", api.SaveFileContent)
		privateGroup.POST("/files/mode", api.ChangeMode)
		privateGroup.POST("/files/owner", api.ChangeOwner)
		privateGroup.POST("/files/compress", api.CompressFile)
		privateGroup.POST("/files/decompress", api.DecompressFile)
		privateGroup.POST("/files/upload", api.UploadFile)
		privateGroup.GET("/files/download", api.DownloadFile)
		privateGroup.POST("/files/wget", api.WgetFile)
		privateGroup.POST("/files/tree", api.GetFileTree)
		privateGroup.POST("/files/size", api.GetDirSize)
		privateGroup.POST("/files/user/group", api.GetUsersAndGroups)

		// 主机管理
		privateGroup.POST("/hosts", api.CreateHost)
		privateGroup.POST("/hosts/update", api.UpdateHost)
		privateGroup.POST("/hosts/del", api.DeleteHost)
		privateGroup.POST("/hosts/search", api.SearchHost)
		privateGroup.GET("/hosts/tree", api.GetHostTree)
		privateGroup.GET("/hosts/test", api.TestHost)
		privateGroup.POST("/hosts/test-conn", api.TestHostConn)

		// 快速命令
		privateGroup.POST("/commands", api.CreateCommand)
		privateGroup.POST("/commands/update", api.UpdateCommand)
		privateGroup.POST("/commands/del", api.DeleteCommand)
		privateGroup.POST("/commands/search", api.SearchCommand)
		privateGroup.GET("/commands/tree", api.GetCommandTree)

		// 分组管理
		privateGroup.POST("/groups", api.CreateGroup)
		privateGroup.POST("/groups/update", api.UpdateGroup)
		privateGroup.POST("/groups/del", api.DeleteGroup)
		privateGroup.GET("/groups", api.GetGroupList)

		// SSL 证书
		privateGroup.POST("/certificates/search", api.SearchCertificate)
		privateGroup.POST("/certificates", api.CreateCertificate)
		privateGroup.POST("/certificates/update", api.UpdateCertificate)
		privateGroup.POST("/certificates/upload", api.UploadCertificate)
		privateGroup.POST("/certificates/del", api.DeleteCertificate)
		privateGroup.POST("/certificates/detail", api.GetCertificateDetail)
		privateGroup.POST("/certificates/apply", api.ApplyCertificate)
		privateGroup.POST("/certificates/renew", api.RenewCertificate)
		privateGroup.POST("/certificates/log", api.GetCertificateLog)

		// ACME 账户
		privateGroup.GET("/acme-accounts", api.ListAcmeAccount)
		privateGroup.POST("/acme-accounts", api.CreateAcmeAccount)
		privateGroup.POST("/acme-accounts/del", api.DeleteAcmeAccount)

		// DNS 账户
		privateGroup.GET("/dns-accounts", api.ListDnsAccount)
		privateGroup.POST("/dns-accounts", api.CreateDnsAccount)
		privateGroup.POST("/dns-accounts/update", api.UpdateDnsAccount)
		privateGroup.POST("/dns-accounts/del", api.DeleteDnsAccount)

		// 账户导入导出
		privateGroup.GET("/ssl/accounts/export", api.ExportAccounts)
		privateGroup.POST("/ssl/accounts/import", api.ImportAccounts)

		// SSL 设置
		privateGroup.GET("/ssl/dir", api.GetSSLDir)
		privateGroup.POST("/ssl/dir", api.UpdateSSLDir)
		privateGroup.GET("/ssl/dns-providers", api.GetDnsProviders)

		// 系统监控
		privateGroup.GET("/monitor/stats", api.GetCurrentStats)
		privateGroup.POST("/monitor/history", api.LoadMonitorHistory)
		privateGroup.GET("/monitor/setting", api.GetMonitorSetting)
		privateGroup.POST("/monitor/setting/update", api.UpdateMonitorSetting)
		privateGroup.POST("/monitor/history/clean", api.CleanMonitorData)
		privateGroup.GET("/monitor/io-options", api.GetIOOptions)
		privateGroup.GET("/monitor/network-options", api.GetNetworkOptions)

		// 进程管理
		privateGroup.POST("/process/search", api.ListProcesses)
		privateGroup.POST("/process/stop", api.StopProcess)
		privateGroup.GET("/process/connections", api.ListConnections)

		// SSH 管理
		privateGroup.GET("/ssh/info", api.GetSSHInfo)
		privateGroup.POST("/ssh/operate", api.OperateSSH)
		privateGroup.POST("/ssh/update", api.UpdateSSHConfig)
		privateGroup.POST("/ssh/log", api.LoadSSHLog)
		privateGroup.GET("/ssh/sshd-config", api.GetSSHDConfig)
		privateGroup.POST("/ssh/sshd-config", api.SaveSSHDConfig)
		privateGroup.GET("/ssh/authorized-keys", api.ListAuthorizedKeys)
		privateGroup.POST("/ssh/authorized-keys", api.AddAuthorizedKey)
		privateGroup.POST("/ssh/authorized-keys/delete", api.DeleteAuthorizedKey)

		// 防火墙
		privateGroup.GET("/firewall/base", api.GetBaseInfo)
		privateGroup.POST("/firewall/operate", api.Operate)
		privateGroup.POST("/firewall/port/search", api.ListPortRules)
		privateGroup.POST("/firewall/port", api.CreatePortRule)
		privateGroup.POST("/firewall/port/del", api.DeletePortRule)
		privateGroup.GET("/firewall/ip", api.ListIPRules)
		privateGroup.POST("/firewall/ip", api.CreateIPRule)
		privateGroup.POST("/firewall/ip/del", api.DeleteIPRule)

		// 磁盘管理
		privateGroup.GET("/disk/info", api.GetDiskInfo)
		privateGroup.GET("/disk/remote", api.ListRemoteMounts)
		privateGroup.POST("/disk/remote/mount", api.MountRemote)
		privateGroup.POST("/disk/remote/unmount", api.UnmountRemote)
		privateGroup.GET("/disk/block-devices", api.ListBlockDevices)
		privateGroup.POST("/disk/local/mount", api.MountLocal)
		privateGroup.POST("/disk/local/unmount", api.UnmountLocal)

		// 升级管理
		privateGroup.GET("/upgrade/current", api.GetVersion)
		privateGroup.POST("/upgrade/check", api.CheckUpdate)
		privateGroup.POST("/upgrade/do", api.DoUpgrade)
		privateGroup.GET("/upgrade/log", api.GetUpgradeLog)

		// Nginx 管理
		privateGroup.GET("/nginx/status", api.GetNginxStatus)
		privateGroup.POST("/nginx/operate", api.OperateNginx)
		privateGroup.GET("/nginx/config-test", api.TestNginxConfig)
		privateGroup.POST("/nginx/install", api.InstallNginx)
		privateGroup.GET("/nginx/install/progress", api.GetInstallProgress)
		privateGroup.POST("/nginx/uninstall", api.UninstallNginx)
		privateGroup.GET("/nginx/versions", api.ListNginxVersions)
		privateGroup.GET("/nginx/update/check", api.CheckNginxUpdate)
		privateGroup.POST("/nginx/update/upgrade", api.UpgradeNginx)
		privateGroup.POST("/nginx/autostart", api.SetNginxAutoStart)

		// Nginx 配置文件管理
		privateGroup.GET("/nginx/conf", api.GetNginxMainConf)
		privateGroup.POST("/nginx/conf", api.SaveNginxMainConf)
		privateGroup.GET("/nginx/conf-files", api.ListNginxConfFiles)
		privateGroup.POST("/nginx/conf-file", api.GetNginxConfFile)
		privateGroup.POST("/nginx/conf-file/save", api.SaveNginxConfFile)

		// 计划任务
		privateGroup.POST("/cronjobs", api.CreateCronjob)
		privateGroup.POST("/cronjobs/update", api.UpdateCronjob)
		privateGroup.POST("/cronjobs/del", api.DeleteCronjob)
		privateGroup.POST("/cronjobs/search", api.SearchCronjob)
		privateGroup.POST("/cronjobs/detail", api.GetCronjob)
		privateGroup.POST("/cronjobs/status", api.UpdateCronjobStatus)
		privateGroup.POST("/cronjobs/handle-once", api.HandleOnceCronjob)
		privateGroup.POST("/cronjobs/records", api.SearchCronjobRecords)

		// 数据库管理
		privateGroup.POST("/databases/servers", api.CreateDatabaseServer)
		privateGroup.POST("/databases/servers/update", api.UpdateDatabaseServer)
		privateGroup.POST("/databases/servers/del", api.DeleteDatabaseServer)
		privateGroup.POST("/databases/servers/search", api.SearchDatabaseServer)
		privateGroup.POST("/databases/servers/test", api.TestDatabaseConnection)
		privateGroup.POST("/databases/instances", api.CreateDatabaseInstance)
		privateGroup.POST("/databases/instances/del", api.DeleteDatabaseInstance)
		privateGroup.POST("/databases/instances/search", api.SearchDatabaseInstance)
		privateGroup.POST("/databases/instances/sync", api.SyncDatabaseInstances)
		privateGroup.POST("/databases/instances/password", api.ChangeInstancePassword)
		privateGroup.POST("/databases/instances/backup", api.BackupDatabaseInstance)
		privateGroup.POST("/databases/instances/restore", api.RestoreDatabaseInstance)

		// 节点管理
		privateGroup.GET("/nodes", api.ListNodes)
		privateGroup.POST("/nodes", api.CreateNode)
		privateGroup.POST("/nodes/update", api.UpdateNode)
		privateGroup.POST("/nodes/del", api.DeleteNode)
		privateGroup.POST("/nodes/test", api.TestNodeConnection)
		privateGroup.POST("/nodes/ssh-test", api.TestSSH)
		privateGroup.POST("/nodes/agent-action", api.AgentAction)

		// 备份管理
		privateGroup.GET("/backup/accounts", api.ListBackupAccounts)
		privateGroup.POST("/backup/accounts", api.CreateBackupAccount)
		privateGroup.POST("/backup/accounts/update", api.UpdateBackupAccount)
		privateGroup.POST("/backup/accounts/del", api.DeleteBackupAccount)
		privateGroup.POST("/backup", api.CreateBackup)
		privateGroup.POST("/backup/records/search", api.SearchBackupRecords)
		privateGroup.POST("/backup/records/del", api.DeleteBackupRecord)

		// 容器管理
		privateGroup.GET("/containers/docker/status", api.DockerStatus)
		privateGroup.POST("/containers/docker/install", api.InstallDocker)
		privateGroup.GET("/containers/docker/install/log", api.GetDockerInstallLog)
		privateGroup.POST("/containers/search", api.ListContainers)
		privateGroup.POST("/containers", api.CreateContainer)
		privateGroup.POST("/containers/operate", api.OperateContainer)
		privateGroup.POST("/containers/logs", api.ContainerLogs)
		privateGroup.POST("/containers/del", api.RemoveContainer)
		privateGroup.GET("/containers/image", api.ListImages)
		privateGroup.POST("/containers/image/pull", api.PullImage)
		privateGroup.POST("/containers/image/del", api.RemoveImage)
		privateGroup.GET("/containers/network", api.ListNetworks)
		privateGroup.POST("/containers/network", api.CreateNetwork)
		privateGroup.POST("/containers/network/del", api.RemoveNetwork)
		privateGroup.GET("/containers/volume", api.ListVolumes)
		privateGroup.POST("/containers/volume", api.CreateVolume)
		privateGroup.POST("/containers/volume/del", api.RemoveVolume)
		privateGroup.GET("/containers/compose", api.ListCompose)
		privateGroup.POST("/containers/compose", api.CreateCompose)
		privateGroup.POST("/containers/compose/operate", api.OperateCompose)

		// 流量统计
		privateGroup.GET("/traffic/configs", api.ListConfigs)
		privateGroup.POST("/traffic/configs", api.CreateConfig)
		privateGroup.POST("/traffic/configs/del", api.DeleteConfig)
		privateGroup.GET("/traffic/interfaces", api.ListInterfaces)
		privateGroup.POST("/traffic/stats", api.GetStats)
		privateGroup.GET("/traffic/summary", api.GetSummary)

		// GOST 代理管理
		privateGroup.GET("/gost/status", api.GetGostStatus)
		privateGroup.POST("/gost/install", api.InstallGost)
		privateGroup.GET("/gost/install/progress", api.GetGostInstallProgress)
		privateGroup.POST("/gost/uninstall", api.UninstallGost)
		privateGroup.POST("/gost/operate", api.OperateGost)
		privateGroup.GET("/gost/check-update", api.CheckGostUpdate)
		privateGroup.POST("/gost/upgrade", api.UpgradeGost)
		privateGroup.POST("/gost/services/search", api.SearchGostService)
		privateGroup.POST("/gost/services", api.CreateGostService)
		privateGroup.POST("/gost/services/update", api.UpdateGostService)
		privateGroup.POST("/gost/services/del", api.DeleteGostService)
		privateGroup.POST("/gost/services/toggle", api.ToggleGostService)
		privateGroup.POST("/gost/chains/search", api.SearchGostChain)
		privateGroup.POST("/gost/chains", api.CreateGostChain)
		privateGroup.POST("/gost/chains/update", api.UpdateGostChain)
		privateGroup.POST("/gost/chains/del", api.DeleteGostChain)
		privateGroup.POST("/gost/sync", api.SyncGost)

		// 工具箱 - Samba
		privateGroup.GET("/toolbox/samba/status", api.GetSambaStatus)
		privateGroup.POST("/toolbox/samba/install", api.InstallSamba)
		privateGroup.POST("/toolbox/samba/uninstall", api.UninstallSamba)
		privateGroup.POST("/toolbox/samba/operate", api.OperateSamba)
		privateGroup.GET("/toolbox/samba/shares", api.ListSambaShares)
		privateGroup.POST("/toolbox/samba/shares/create", api.CreateSambaShare)
		privateGroup.POST("/toolbox/samba/shares/update", api.UpdateSambaShare)
		privateGroup.POST("/toolbox/samba/shares/del", api.DeleteSambaShare)
		privateGroup.GET("/toolbox/samba/users", api.ListSambaUsers)
		privateGroup.POST("/toolbox/samba/users/create", api.CreateSambaUser)
		privateGroup.POST("/toolbox/samba/users/del", api.DeleteSambaUser)
		privateGroup.POST("/toolbox/samba/users/password", api.UpdateSambaPassword)
		privateGroup.POST("/toolbox/samba/users/toggle", api.ToggleSambaUser)
		privateGroup.GET("/toolbox/samba/config", api.GetSambaGlobalConfig)
		privateGroup.POST("/toolbox/samba/config/update", api.UpdateSambaGlobalConfig)
		privateGroup.GET("/toolbox/samba/connections", api.GetSambaConnections)

		// 工具箱 - NFS
		privateGroup.GET("/toolbox/nfs/status", api.GetNfsStatus)
		privateGroup.POST("/toolbox/nfs/install", api.InstallNfs)
		privateGroup.POST("/toolbox/nfs/uninstall", api.UninstallNfs)
		privateGroup.POST("/toolbox/nfs/operate", api.OperateNfs)
		privateGroup.GET("/toolbox/nfs/exports", api.ListNfsExports)
		privateGroup.POST("/toolbox/nfs/exports/create", api.CreateNfsExport)
		privateGroup.POST("/toolbox/nfs/exports/update", api.UpdateNfsExport)
		privateGroup.POST("/toolbox/nfs/exports/del", api.DeleteNfsExport)
		privateGroup.GET("/toolbox/nfs/connections", api.GetNfsConnections)

		// 工具箱 - Fail2ban
		privateGroup.GET("/toolbox/fail2ban/status", api.GetFail2banStatus)
		privateGroup.POST("/toolbox/fail2ban/install", api.InstallFail2ban)
		privateGroup.POST("/toolbox/fail2ban/uninstall", api.UninstallFail2ban)
		privateGroup.POST("/toolbox/fail2ban/operate", api.OperateFail2ban)
		privateGroup.GET("/toolbox/fail2ban/jails", api.ListFail2banJails)
		privateGroup.POST("/toolbox/fail2ban/jails/update", api.UpdateFail2banJail)
		privateGroup.POST("/toolbox/fail2ban/jails/ssh", api.SetFail2banSSH)
		privateGroup.GET("/toolbox/fail2ban/banned", api.ListFail2banBanned)
		privateGroup.POST("/toolbox/fail2ban/ban", api.BanFail2banIP)
		privateGroup.POST("/toolbox/fail2ban/unban", api.UnbanFail2banIP)
		privateGroup.GET("/toolbox/fail2ban/logs", api.GetFail2banLogs)

		// 工具箱 - 服务管理
		privateGroup.GET("/toolbox/services", api.ListSystemdServices)
		privateGroup.GET("/toolbox/services/detail", api.GetSystemdServiceDetail)
		privateGroup.POST("/toolbox/services/create", api.CreateSystemdService)
		privateGroup.POST("/toolbox/services/update", api.UpdateSystemdService)
		privateGroup.POST("/toolbox/services/delete", api.DeleteSystemdService)
		privateGroup.POST("/toolbox/services/operate", api.OperateSystemdService)
		privateGroup.GET("/toolbox/services/logs", api.GetSystemdServiceLogs)

		// 工具箱 - IP 归属地
		privateGroup.GET("/toolbox/ip/lookup", api.LookupIP)
		privateGroup.POST("/toolbox/ip/lookup/batch", api.LookupIPBatch)
		privateGroup.GET("/toolbox/ip/db/info", api.GetIPDBInfo)
		privateGroup.POST("/toolbox/ip/db/download", api.DownloadIPDB)

		// SSH 私钥管理
		privateGroup.GET("/ssh/keys", api.ListSSHKeys)
		privateGroup.GET("/ssh/keys/private", api.GetSSHPrivateKey)
		privateGroup.POST("/ssh/keys/generate", api.GenerateSSHKey)
		privateGroup.POST("/ssh/keys/import", api.ImportSSHKey)
		privateGroup.POST("/ssh/keys/delete", api.DeleteSSHKey)

		// 系统用户管理
		privateGroup.GET("/host/users", api.ListUsers)
		privateGroup.POST("/host/users/create", api.CreateUser)
		privateGroup.POST("/host/users/update", api.UpdateUser)
		privateGroup.POST("/host/users/delete", api.DeleteUser)
		privateGroup.GET("/host/users/shells", api.ListShells)
		privateGroup.GET("/host/users/groups", api.ListGroups)

		// 系统设置（主机名/时区/DNS/Swap）
		privateGroup.GET("/host/system/info", api.GetSystemInfo)
		privateGroup.POST("/host/system/hostname", api.SetHostname)
		privateGroup.POST("/host/system/timezone", api.SetTimezone)
		privateGroup.GET("/host/system/timezones", api.ListTimezones)
		privateGroup.GET("/host/system/dns", api.GetDNS)
		privateGroup.POST("/host/system/dns", api.SetDNS)
		privateGroup.GET("/host/system/swap", api.GetSwap)
		privateGroup.POST("/host/system/swap/create", api.CreateSwap)
		privateGroup.POST("/host/system/swap/delete", api.DeleteSwap)
		privateGroup.POST("/host/system/swap/operate", api.SwapOperate)

		// 网站管理
		privateGroup.POST("/websites/search", api.SearchWebsite)
		privateGroup.POST("/websites", api.CreateWebsite)
		privateGroup.POST("/websites/update", api.UpdateWebsite)
		privateGroup.POST("/websites/del", api.DeleteWebsite)
		privateGroup.POST("/websites/detail", api.GetWebsiteDetail)
		privateGroup.POST("/websites/enable", api.EnableWebsite)
		privateGroup.POST("/websites/disable", api.DisableWebsite)
		privateGroup.POST("/websites/nginx-config", api.GetWebsiteNginxConfig)
		privateGroup.POST("/websites/log", api.GetWebsiteLog)
		privateGroup.POST("/websites/conf-content", api.GetSiteConfContent)
		privateGroup.POST("/websites/conf-content/save", api.SaveSiteConfContent)
		privateGroup.POST("/websites/config-mode", api.SwitchConfigMode)
		privateGroup.POST("/websites/log-analysis", api.AnalyzeNginxLog)

		// Nginx 日志分析（全局）
		privateGroup.GET("/nginx/log/sites", api.DetectNginxSites)
		privateGroup.POST("/nginx/log/analyze", api.AnalyzeNginxSiteLog)
		privateGroup.POST("/nginx/log/tail", api.TailNginxLog)
		privateGroup.POST("/nginx/log/drilldown", api.DrilldownNginxLog)

		// 证书同步 - 证书源管理
		privateGroup.GET("/cert-sources", api.ListCertSources)
		privateGroup.POST("/cert-sources", api.CreateCertSource)
		privateGroup.POST("/cert-sources/update", api.UpdateCertSource)
		privateGroup.POST("/cert-sources/del", api.DeleteCertSource)
		privateGroup.POST("/cert-sources/sync", api.SyncCertSource)
		privateGroup.POST("/cert-sources/test", api.TestCertSource)
		privateGroup.POST("/cert-sync/logs", api.SearchSyncLogs)

		// 证书服务端设置
		privateGroup.GET("/cert-server/setting", api.GetCertServerSetting)
		privateGroup.POST("/cert-server/setting", api.UpdateCertServerSetting)
	}

	// 证书服务端 API（Token 认证，供其他面板拉取证书）
	certServerGroup := r.Group("/api/v1/cert-server")
	certServerGroup.Use(middleware.CertServerAuth())
	{
		certServerGroup.GET("/certs", api.ServeCerts)
	}

	// WebSocket
	wsGroup := r.Group("/api/v1")
	wsGroup.Use(middleware.JWTAuth())
	{
		wsGroup.GET("/terminal", api.WsTerminal)
	}

	return r
}

// setupFrontend 注册前端静态文件服务
// 生产模式下，Go 二进制内嵌前端构建产物，直接提供 SPA 服务
func setupFrontend(r *gin.Engine) {
	frontendFS, err := web.GetFS()
	if err != nil {
		return
	}

	// 检查是否有真正的前端资源（非占位文件）
	indexFile, err := fs.ReadFile(frontendFS.(fs.ReadFileFS), "index.html")
	if err != nil || len(indexFile) < 100 {
		// 开发模式：没有前端构建产物，跳过
		return
	}

	// 静态文件服务器
	staticHandler := http.FileServer(http.FS(frontendFS))

	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path

		// API 和 WebSocket 请求不走前端
		if len(path) >= 4 && path[:4] == "/api" {
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "not found"})
			return
		}

		// 尝试提供静态文件
		if _, err := fs.Stat(frontendFS, path[1:]); err == nil {
			staticHandler.ServeHTTP(c.Writer, c.Request)
			return
		}

		// SPA 回退：所有其他路径返回 index.html（显式 200 避免 Gin NoRoute 默认 404）
		c.Data(http.StatusOK, "text/html; charset=utf-8", indexFile)
	})
}
