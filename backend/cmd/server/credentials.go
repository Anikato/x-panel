package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"

	"xpanel/global"
	initViper "xpanel/init/viper"
	"xpanel/security/credentials"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func parseCredentialVerifyArgs(args []string) (string, error) {
	fs := flag.NewFlagSet("credentials verify", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	databasePath := fs.String("db", "", "待验证的 SQLite 数据库路径")
	if err := fs.Parse(args); err != nil {
		return "", err
	}
	if fs.NArg() != 0 {
		return "", errors.New("credentials verify 不接受位置参数")
	}
	if *databasePath == "" {
		return "", errors.New("--db 为必填项")
	}
	absolutePath, err := filepath.Abs(*databasePath)
	if err != nil {
		return "", fmt.Errorf("resolve credential database path: %w", err)
	}
	return absolutePath, nil
}

func runCredentials(args []string) {
	if len(args) == 0 || args[0] != "verify" {
		fmt.Fprintln(os.Stderr, "用法: xpanel credentials verify --db <candidate.db>")
		os.Exit(1)
	}
	databasePath, err := parseCredentialVerifyArgs(args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		os.Exit(1)
	}

	initViper.Init()
	if err := verifyCredentialDatabase(
		databasePath,
		global.CONF.System.CredentialKeyPath,
	); err != nil {
		fmt.Fprintf(os.Stderr, "凭据兼容性验证失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ 数据库凭据与当前密钥环兼容")
}

func verifyCredentialDatabase(databasePath, keyPath string) error {
	info, err := os.Lstat(databasePath)
	if err != nil {
		return fmt.Errorf("inspect candidate database %s: %w", databasePath, err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return fmt.Errorf("candidate database has unsafe type: %s", databasePath)
	}

	manager, _, err := credentials.LoadOrCreate(keyPath, false)
	if err != nil {
		return fmt.Errorf("load existing credential keyring: %w", err)
	}

	databaseURL := (&url.URL{Scheme: "file", Path: databasePath}).String()
	db, err := gorm.Open(sqlite.Open(databaseURL+"?mode=ro"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return fmt.Errorf("open candidate database read-only: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("access candidate database connection: %w", err)
	}
	defer sqlDB.Close()

	if err := db.Exec("PRAGMA query_only = ON").Error; err != nil {
		return fmt.Errorf("enable candidate database query-only mode: %w", err)
	}
	if err := credentials.ValidateDatabase(db, manager); err != nil {
		return fmt.Errorf("validate candidate database credentials: %w", err)
	}
	return nil
}
