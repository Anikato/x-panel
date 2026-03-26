package migration

import (
	"xpanel/app/model"
	"xpanel/global"
)

// Init 执行数据库自动迁移
func Init() {
	if err := global.DB.AutoMigrate(
		&model.Setting{},
		&model.LoginLog{},
		&model.OperationLog{},
		&model.Host{},
		&model.Command{},
		&model.Group{},
		&model.AcmeAccount{},
		&model.DnsAccount{},
		&model.Certificate{},
		&model.Website{},
		&model.Cronjob{},
		&model.CronjobRecord{},
		&model.DatabaseServer{},
		&model.DatabaseInstance{},
		&model.BackupAccount{},
		&model.BackupRecord{},
		&model.Node{},
		&model.TrafficConfig{},
		&model.TrafficHourly{},
		&model.TrafficSnapshot{},
		&model.XrayNode{},
		&model.XrayUser{},
		&model.XrayTrafficDaily{},
		&model.XrayOutbound{},
	); err != nil {
		panic("Failed to auto-migrate database: " + err.Error())
	}

	runOnceDataMigrations()

	initDefaultSettings()
	global.LOG.Info("Database migration completed")
}

func runOnceDataMigrations() {
	migrated := func(key string) bool {
		var count int64
		global.DB.Model(&model.Setting{}).Where("`key` = ?", key).Count(&count)
		return count > 0
	}
	markDone := func(key string) {
		global.DB.Create(&model.Setting{Key: key, Value: "done"})
	}

	if !migrated("_mig_website_perf_defaults") {
		global.DB.Exec("UPDATE websites SET gzip_enable = 1, security_headers = 1 WHERE gzip_enable = 0 AND security_headers = 0")
		markDone("_mig_website_perf_defaults")
		global.LOG.Info("Migration: enabled gzip & security headers for existing websites")
	}
}

func initDefaultSettings() {
	defaults := []model.Setting{
		{Key: "UserName", Value: "admin"},
		{Key: "Password", Value: ""},
		{Key: "Language", Value: "zh"},
		{Key: "SessionTimeout", Value: "86400"},
		{Key: "PanelName", Value: "X-Panel"},
		{Key: "Theme", Value: "auto"},
		{Key: "SecurityEntrance", Value: ""},
		{Key: "MFAStatus", Value: "Disable"},
		{Key: "MFASecret", Value: ""},
		{Key: "SSLDir", Value: ""},
		{Key: "UpgradeURL", Value: ""},
		{Key: "GitHubToken", Value: ""},
		{Key: "AgentToken", Value: ""},
		// Xray 日志设置
		{Key: "XrayLogLevel", Value: "warning"},
		{Key: "XrayAccessLog", Value: "/data/xray/log/access.log"},
		{Key: "XrayErrorLog", Value: "/data/xray/log/error.log"},
	}

	for _, s := range defaults {
		var count int64
		global.DB.Model(&model.Setting{}).Where("`key` = ?", s.Key).Count(&count)
		if count == 0 {
			global.DB.Create(&s)
		}
	}
}
