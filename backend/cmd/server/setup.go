package main

import (
	"flag"
	"fmt"
	"os"

	"xpanel/utils/encrypt"
	initViper "xpanel/init/viper"
	initDB "xpanel/init/db"
	"xpanel/init/migration"
	"xpanel/global"
)

// runSetup 初始化管理员用户名和密码
// 用法: xpanel setup --username admin --password mypass123
func runSetup(args []string) {
	fs := flag.NewFlagSet("setup", flag.ExitOnError)
	username := fs.String("username", "", "管理员用户名")
	password := fs.String("password", "", "管理员密码")
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

	// 初始化配置和数据库（不启动 HTTP 服务）
	initViper.Init()
	initDB.Init()
	migration.Init()

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

	fmt.Printf("✓ 管理员账户已设置: %s\n", *username)
}
