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
	privateGroup.Use(middleware.OperationLog())
	{
		// 认证
		privateGroup.POST("/auth/logout", api.Logout)
		privateGroup.POST("/auth/password", api.UpdatePassword)

		// 设置
		privateGroup.GET("/settings", api.GetSettingInfo)
		privateGroup.POST("/settings/update", api.Update)
		privateGroup.POST("/settings/port/update", api.UpdatePort)

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

		// 进程管理
		privateGroup.POST("/process/search", api.ListProcesses)
		privateGroup.POST("/process/stop", api.StopProcess)
		privateGroup.GET("/process/connections", api.ListConnections)

		// SSH 管理
		privateGroup.GET("/ssh/info", api.GetSSHInfo)
		privateGroup.POST("/ssh/operate", api.OperateSSH)
		privateGroup.POST("/ssh/update", api.UpdateSSHConfig)
		privateGroup.POST("/ssh/log", api.LoadSSHLog)

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

		// SPA 回退：所有其他路径返回 index.html
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.Writer.Write(indexFile)
	})
}
