package db

import (
	"fmt"
	"net/url"
	"path/filepath"

	"xpanel/global"
	initPermission "xpanel/init/permission"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// InitMonitorDB 初始化独立的监控数据库
func InitMonitorDB() {
	dbDir := filepath.Dir(global.CONF.System.DbPath)
	monitorPath := filepath.Join(dbDir, "monitor.db")
	if err := initPermission.EnsurePrivateDirectory(dbDir); err != nil {
		global.LOG.Errorf("Failed to harden monitor database directory: %v", err)
		return
	}

	logLevel := logger.Silent
	db, err := openSQLite(monitorPath, logLevel)
	if err != nil {
		global.LOG.Errorf("Failed to open monitor database: %v", err)
		return
	}
	if err := hardenSQLiteFiles(monitorPath); err != nil {
		global.LOG.Errorf("Failed to harden monitor database: %v", err)
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
	if err := initPermission.EnsurePrivateDirectory(dbDir); err != nil {
		panic(fmt.Sprintf("Failed to create db directory %s: %v", dbDir, err))
	}

	// 配置 GORM 日志级别
	logLevel := logger.Silent
	if global.CONF.System.Mode == "debug" {
		logLevel = logger.Info
	}

	db, err := openSQLite(dbPath, logLevel)
	if err != nil {
		panic(fmt.Sprintf("Failed to connect database: %v", err))
	}
	if err := hardenSQLiteFiles(dbPath); err != nil {
		panic(fmt.Sprintf("Failed to harden database: %v", err))
	}

	global.DB = db
	global.LOG.Info("Database initialized")
}

func sqliteDSN(path string) string {
	return (&url.URL{
		Scheme:   "file",
		Path:     path,
		RawQuery: "_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)",
	}).String()
}

func openSQLite(path string, logLevel logger.LogLevel) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(sqliteDSN(path)), &gorm.Config{
		Logger:                                   logger.Default.LogMode(logLevel),
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		return nil, err
	}
	if err := configureSQLite(db); err != nil {
		if sqlDB, dbErr := db.DB(); dbErr == nil {
			_ = sqlDB.Close()
		}
		return nil, err
	}
	return db, nil
}

func configureSQLite(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	if err := db.Exec("PRAGMA busy_timeout = 5000").Error; err != nil {
		return err
	}
	return db.Exec("PRAGMA journal_mode = WAL").Error
}

func hardenSQLiteFiles(path string) error {
	return initPermission.Harden(initPermission.Paths{Files: []string{
		path,
		path + "-wal",
		path + "-shm",
	}})
}
