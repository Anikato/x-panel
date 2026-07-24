package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"xpanel/global"
	initCredential "xpanel/init/credential"
	initDB "xpanel/init/db"
	"xpanel/init/migration"
	initPermission "xpanel/init/permission"
	initViper "xpanel/init/viper"
	"xpanel/utils/encrypt"

	"github.com/sirupsen/logrus"
)

// runSetup 初始化管理员用户名和密码
// 用法: xpanel setup --username admin --password mypass123
func runSetup(args []string) {
	fs := flag.NewFlagSet("setup", flag.ExitOnError)
	username := fs.String("username", "", "管理员用户名")
	password := fs.String("password", "", "管理员密码")
	waitSec := fs.Int("wait", 15, "等待数据库就绪的最大秒数")
	fs.Parse(args)

	if *username == "" || *password == "" {
		fmt.Fprintln(os.Stderr, "错误: --username 和 --password 均为必填项")
		fmt.Fprintln(os.Stderr, "用法: xpanel setup --username admin --password mypass123")
		os.Exit(1)
	}

	if len(*password) < 6 {
		fmt.Fprintln(os.Stderr, "错误: 密码长度不能少于 6 位")
		os.Exit(1)
	}

	initializeOfflineDatabase()

	// 等待 Password 字段就绪（migration 已创建，但第一次启动的 xpanel 服务
	// 可能还在并发写入，此处兜底等待确保字段存在）
	deadline := time.Now().Add(time.Duration(*waitSec) * time.Second)
	for time.Now().Before(deadline) {
		var count int64
		global.DB.Table("settings").Where("`key` = ?", "Password").Count(&count)
		if count > 0 {
			break
		}
		fmt.Fprintln(os.Stderr, "等待数据库初始化...")
		time.Sleep(time.Second)
	}

	// 哈希密码
	hashed, err := encrypt.HashPassword(*password)
	if err != nil {
		fmt.Fprintf(os.Stderr, "密码加密失败: %v\n", err)
		os.Exit(1)
	}

	// 写入数据库
	if err := global.DB.Exec("UPDATE settings SET value = ? WHERE `key` = 'UserName'", *username).Error; err != nil {
		fmt.Fprintf(os.Stderr, "写入用户名失败: %v\n", err)
		os.Exit(1)
	}
	if err := global.DB.Exec("UPDATE settings SET value = ? WHERE `key` = 'Password'", hashed).Error; err != nil {
		fmt.Fprintf(os.Stderr, "写入密码失败: %v\n", err)
		os.Exit(1)
	}

	// 验证写入成功
	var savedPwd string
	global.DB.Table("settings").Where("`key` = ?", "Password").Pluck("value", &savedPwd)
	if savedPwd == "" {
		fmt.Fprintln(os.Stderr, "错误: 密码写入后验证失败，请检查数据库状态")
		os.Exit(1)
	}

	fmt.Printf("✓ 管理员账户已设置: %s\n", *username)
}

// initializeOfflineDatabase 为不启动 HTTP 服务的管理命令初始化本地数据层。
func initializeOfflineDatabase() {
	initViper.Init()
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
		fmt.Fprintf(os.Stderr, "运行目录权限加固失败: %v\n", err)
		os.Exit(1)
	}

	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)
	logger.SetOutput(os.Stderr)
	global.LOG = logger

	initDB.Init()
	migration.Init()
	if err := initCredential.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "凭据保护初始化失败: %v\n", err)
		os.Exit(1)
	}
}
