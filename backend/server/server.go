package server

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"xpanel/global"
	"xpanel/i18n"
	"xpanel/app/service"
	initAuth "xpanel/init/auth"
	initCron "xpanel/init/cron"
	initDB "xpanel/init/db"
	initLog "xpanel/init/log"
	"xpanel/init/migration"
	initViper "xpanel/init/viper"
	"xpanel/router"
	"xpanel/utils/iplocation"
)

// Start 启动服务器（按顺序初始化各模块）
func Start() {
	// 1. Viper 加载配置
	initViper.Init()

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

	// 3. 数据库连接
	initDB.Init()
	initDB.InitMonitorDB()

	// 4. 数据库迁移 + 默认数据
	migration.Init()

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

	// 8.5 GOST 配置同步（如果 GOST 已安装且运行中，全量推送规则）
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
		srv := &http.Server{
			Addr:     ":" + port,
			Handler:  r,
			ErrorLog: newTLSFilteredLogger(),
		}
		if err := srv.ListenAndServeTLS(sslConf.CertPath, sslConf.KeyPath); err != nil {
			panic(fmt.Sprintf("Server failed to start with TLS: %v", err))
		}
	} else {
		global.LOG.Infof("X-Panel server starting on HTTP :%s", port)
		if err := r.Run(":" + port); err != nil {
			panic(fmt.Sprintf("Server failed to start: %v", err))
		}
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
