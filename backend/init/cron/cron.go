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

	// 启动监控数据采集
	monitorStatus, _ := service.NewISettingService().GetValueByKey("MonitorStatus")
	if monitorStatus == "enable" {
		monitorInterval, _ := service.NewISettingService().GetValueByKey("MonitorInterval")
		if monitorInterval == "" {
			monitorInterval = "300"
		}
		if err := service.StartMonitorCollector(false, monitorInterval); err != nil {
			global.LOG.Errorf("Failed to start monitor collector: %v", err)
		}
	}

	// 每天凌晨 2 点检查证书续期
	global.CRON.AddFunc("0 2 * * *", func() {
		service.AutoRenewCerts()
	})
	go service.AutoRenewCerts()

	// 每天凌晨 3:30 自动升级（如果启用）
	global.CRON.AddFunc("30 3 * * *", func() {
		autoUpgrade()
	})

	// 每分钟检查证书源同步；各来源仍由 SyncInterval 控制实际拉取频率
	global.CRON.AddFunc("* * * * *", func() {
		service.NewICertSourceService().SyncAll()
	})

	global.LOG.Info("Cron scheduler initialized")
}

func autoUpgrade() {
	settingService := service.NewISettingService()

	local, _ := settingService.GetValueByKey("AutoUpgrade")
	if local != "enable" {
		return
	}

	upgradeService := service.NewIUpgradeService()
	// ReleaseURL 为空时由升级服务回落到 UpgradeURL / 默认更新源。
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
