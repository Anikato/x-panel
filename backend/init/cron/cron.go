package cron

import (
	"xpanel/app/dto"
	"xpanel/app/service"
	"xpanel/global"

	"github.com/robfig/cron/v3"
)

func Init() {
	global.CRON = cron.New()
	global.CRON.Start()

	cronjobService := service.NewICronjobService()
	cronjobService.StartAllJobs()

	trafficService := service.NewITrafficService()
	trafficService.StartCollector()

	// 每天凌晨 2 点检查证书续期
	global.CRON.AddFunc("0 2 * * *", func() {
		service.AutoRenewCerts()
	})

	// 每天凌晨 3:30 自动升级（如果启用）
	global.CRON.AddFunc("30 3 * * *", func() {
		autoUpgrade()
	})

	global.LOG.Info("Cron scheduler initialized")
}

func autoUpgrade() {
	settingService := service.NewISettingService()
	val, err := settingService.GetValueByKey("AutoUpgrade")
	if err != nil || val != "enable" {
		return
	}

	upgradeService := service.NewIUpgradeService()
	info, err := upgradeService.CheckUpdate(dto.UpgradeCheckReq{})
	if err != nil || info == nil || !info.HasUpdate {
		return
	}

	global.LOG.Infof("Auto-upgrade: new version %s found, starting upgrade...", info.LatestVersion)
	if err := upgradeService.DoUpgrade(dto.UpgradeReq{
		Version:     info.LatestVersion,
		DownloadURL: info.DownloadURL,
		ChecksumURL: info.ChecksumURL,
	}); err != nil {
		global.LOG.Errorf("Auto-upgrade failed: %v", err)
	}
}
