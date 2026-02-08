package server

import (
	"fmt"

	"xpanel/global"
	"xpanel/i18n"
	initDB "xpanel/init/db"
	initLog "xpanel/init/log"
	"xpanel/init/migration"
	initViper "xpanel/init/viper"
	"xpanel/router"
)

// Start 启动服务器（按顺序初始化各模块）
func Start() {
	// 1. Viper 加载配置
	initViper.Init()

	// 2. 日志模块
	initLog.Init()

	// 3. 数据库连接
	initDB.Init()

	// 4. 数据库迁移 + 默认数据
	migration.Init()

	// 5. i18n 国际化
	i18n.Init()

	// 6. 初始化路由并启动服务
	r := router.Setup(global.CONF.System.Mode)

	port := global.CONF.System.Port
	sslConf := global.CONF.System.SSL

	if sslConf.Enable && sslConf.CertPath != "" && sslConf.KeyPath != "" {
		global.LOG.Infof("X-Panel server starting on HTTPS :%s", port)
		if err := r.RunTLS(":"+port, sslConf.CertPath, sslConf.KeyPath); err != nil {
			panic(fmt.Sprintf("Server failed to start with TLS: %v", err))
		}
	} else {
		global.LOG.Infof("X-Panel server starting on HTTP :%s", port)
		if err := r.Run(":" + port); err != nil {
			panic(fmt.Sprintf("Server failed to start: %v", err))
		}
	}
}
