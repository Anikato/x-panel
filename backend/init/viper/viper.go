package viper

import (
	"fmt"
	"path/filepath"

	"xpanel/global"

	"github.com/spf13/viper"
)

// Init 初始化 Viper 配置加载
func Init() {
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath("./configs")
	v.AddConfigPath("/opt/xpanel/conf")
	v.AddConfigPath(".")

	// 设置默认值
	v.SetDefault("system.port", "9999")
	v.SetDefault("system.mode", "debug")
	v.SetDefault("system.data_dir", "/opt/xpanel")
	v.SetDefault("system.db_path", "/opt/xpanel/db/xpanel.db")
	v.SetDefault("system.jwt_secret", "change-me-to-a-random-string")
	v.SetDefault("system.session_timeout", 86400)
	v.SetDefault("system.ssl.enable", false)
	v.SetDefault("system.ssl.cert_path", "")
	v.SetDefault("system.ssl.key_path", "")
	v.SetDefault("log.level", "info")
	v.SetDefault("log.path", "/opt/xpanel/log")
	v.SetDefault("log.max_size", 100)
	v.SetDefault("log.max_age", 30)
	v.SetDefault("log.compress", true)
	v.SetDefault("nginx.install_dir", "/opt/xpanel/nginx")
	v.SetDefault("nginx.version", "")
	v.SetDefault("nginx.build_repo", "Anikato/nginx-build")

	if err := v.ReadInConfig(); err != nil {
		fmt.Printf("Using default config, config file not found: %v\n", err)
	} else {
		fmt.Printf("Using config file: %s\n", v.ConfigFileUsed())
	}

	var conf global.ServerConfig
	if err := v.Unmarshal(&conf); err != nil {
		panic(fmt.Sprintf("Failed to unmarshal config: %v", err))
	}

	// 如果 db_path 是相对路径，基于 data_dir 拼接
	if !filepath.IsAbs(conf.System.DbPath) {
		conf.System.DbPath = filepath.Join(conf.System.DataDir, conf.System.DbPath)
	}
	if !filepath.IsAbs(conf.Log.Path) {
		conf.Log.Path = filepath.Join(conf.System.DataDir, conf.Log.Path)
	}

	global.CONF = conf
	global.Vp = v
}
