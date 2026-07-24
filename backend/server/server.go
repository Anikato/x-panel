package server

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"xpanel/app/service"
	"xpanel/global"
	"xpanel/i18n"
	initAuth "xpanel/init/auth"
	initCredential "xpanel/init/credential"
	initCron "xpanel/init/cron"
	initDB "xpanel/init/db"
	initLog "xpanel/init/log"
	"xpanel/init/migration"
	initPermission "xpanel/init/permission"
	initViper "xpanel/init/viper"
	"xpanel/router"
	"xpanel/utils/iplocation"
)

// Start 启动服务器（按顺序初始化各模块）
func Start() {
	// 1. Viper 加载配置
	initViper.Init()
	hardenRuntimePaths()

	// 1.5 检测 Nginx 安装模式
	global.CONF.Nginx.DetectNginx()

	// 2. 日志模块
	initLog.Init()

	// 2.5 Nginx 模式日志
	if nc := global.CONF.Nginx; nc.IsInstalled() {
		if nc.HasBothInstalled() {
			global.LOG.Warnf("Detected both system nginx and prefix nginx, using %s mode (config nginx.mode=%s)",
				map[bool]string{true: "system", false: "prefix"}[nc.IsSystemMode()], nc.Mode)
		} else if nc.IsSystemMode() {
			global.LOG.Info("Using system nginx (/etc/nginx)")
		} else {
			global.LOG.Infof("Using prefix nginx (%s)", nc.InstallDir)
		}
	}

	initDatabaseAndMigrations()

	// 4.5 恢复面板进程代理环境
	service.SyncProxyOnStartup()
	service.CleanBackupTempDir(24 * time.Hour)

	// 5. i18n 国际化
	i18n.Init()

	// 6. 登录 IP 跟踪器
	global.IPTracker = initAuth.NewIPTracker()

	// 6.5 IP 归属地数据库
	iplocation.GetService().Init(global.CONF.System.DataDir)

	// 7. Cron 定时任务
	initCron.Init()

	// 8. 节点心跳
	nodeService := service.NewINodeService()
	nodeService.StartHeartbeat()

	// 8.5 Fleet Center 默认上报（失败不影响面板启动）
	service.NewIFleetReporterService().Start()

	// 8.6 GOST 配置同步（如果 GOST 已安装且运行中，全量推送规则）
	go func() {
		gostSvc := service.NewIGostService()
		if err := gostSvc.SyncAll(); err != nil {
			global.LOG.Debugf("GOST sync on startup skipped: %v", err)
		}
	}()

	// 9. 初始化路由并启动服务
	r := router.Setup(global.CONF.System.Mode)

	port := global.CONF.System.Port
	sslConf := global.CONF.System.SSL

	if sslConf.Enable && sslConf.CertPath != "" && sslConf.KeyPath != "" {
		global.LOG.Infof("X-Panel server starting on HTTPS :%s", port)
		reloader, err := newTLSCertificateReloader(sslConf.CertPath, sslConf.KeyPath)
		if err != nil {
			panic(fmt.Sprintf("Server failed to load TLS certificate: %v", err))
		}
		srv := &http.Server{
			Addr:     ":" + port,
			Handler:  r,
			ErrorLog: newTLSFilteredLogger(),
			TLSConfig: &tls.Config{
				MinVersion:     tls.VersionTLS12,
				GetCertificate: reloader.GetCertificate,
			},
		}
		if err := srv.ListenAndServeTLS("", ""); err != nil {
			panic(fmt.Sprintf("Server failed to start with TLS: %v", err))
		}
	} else {
		global.LOG.Infof("X-Panel server starting on HTTP :%s", port)
		if err := r.Run(":" + port); err != nil {
			panic(fmt.Sprintf("Server failed to start: %v", err))
		}
	}
}

// Migrate runs the same database migration entry point as server startup without
// starting HTTP, cron jobs, heartbeats, or other background services.
func Migrate() {
	initViper.Init()
	hardenRuntimePaths()
	initLog.Init()
	initDatabaseAndMigrations()
}

func hardenRuntimePaths() {
	configPath := ""
	if global.Vp != nil {
		configPath = global.Vp.ConfigFileUsed()
	}
	if err := initPermission.HardenRuntime(
		global.CONF.System.DataDir,
		global.CONF.System.DbPath,
		global.CONF.Log.Path,
		global.CONF.System.CredentialKeyPath,
		configPath,
	); err != nil {
		panic(fmt.Sprintf("Failed to harden X-Panel runtime paths: %v", err))
	}
}

func initDatabaseAndMigrations() {
	// 3. 数据库连接
	initDB.Init()
	initDB.InitMonitorDB()

	// 4. 数据库迁移 + 默认数据
	migration.Init()

	// 4.1 凭据密钥环 + 历史明文迁移。任何失败都必须阻止业务任务启动，
	// 避免把密文误当成外部系统密码或 Token 使用。
	if err := initCredential.Init(); err != nil {
		panic(fmt.Sprintf("Failed to initialize credential protection: %v", err))
	}
}

type tlsFilterWriter struct{}

func (w *tlsFilterWriter) Write(p []byte) (int, error) {
	msg := string(p)
	if strings.Contains(msg, "TLS handshake error") {
		return len(p), nil
	}
	return io.Discard.Write(p)
}

func newTLSFilteredLogger() *log.Logger {
	return log.New(&tlsFilterWriter{}, "", 0)
}
