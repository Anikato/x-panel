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
	); err != nil {
		panic("Failed to auto-migrate database: " + err.Error())
	}

	initDefaultSettings()
	global.LOG.Info("Database migration completed")
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
	}

	for _, s := range defaults {
		var count int64
		global.DB.Model(&model.Setting{}).Where("`key` = ?", s.Key).Count(&count)
		if count == 0 {
			global.DB.Create(&s)
		}
	}
}
