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

// InitMonitorDB 初始化独立的监控数据库
func InitMonitorDB() {
	dbDir := filepath.Dir(global.CONF.System.DbPath)
	monitorPath := filepath.Join(dbDir, "monitor.db")

	logLevel := logger.Silent
	db, err := gorm.Open(sqlite.Open(monitorPath), &gorm.Config{
		Logger:                                   logger.Default.LogMode(logLevel),
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		global.LOG.Errorf("Failed to open monitor database: %v", err)
		return
	}
	global.MonitorDB = db
	global.LOG.Info("Monitor database initialized")
}

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
