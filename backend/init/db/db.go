package db

import (
	"fmt"
	"os"
	"path/filepath"

	"xpanel/global"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Init 初始化数据库连接
func Init() {
	dbPath := global.CONF.System.DbPath

	// 确保数据库目录存在
	dbDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		panic(fmt.Sprintf("Failed to create db directory %s: %v", dbDir, err))
	}

	// 配置 GORM 日志级别
	logLevel := logger.Silent
	if global.CONF.System.Mode == "debug" {
		logLevel = logger.Info
	}

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger:                 logger.Default.LogMode(logLevel),
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		panic(fmt.Sprintf("Failed to connect database: %v", err))
	}

	global.DB = db
	global.LOG.Info("Database initialized")
}
