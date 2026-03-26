package cron

import (
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

	xrayService := service.NewIXrayService()
	global.CRON.AddFunc("*/1 * * * *", func() {
		xrayService.SyncTraffic()
	})
	global.CRON.AddFunc("0 * * * *", func() {
		xrayService.CheckExpiredUsers()
	})
	// 每日零点做一次流量快照（用于历史图表）
	global.CRON.AddFunc("1 0 * * *", func() {
		xrayService.SnapshotDailyTraffic()
	})

	// 每天凌晨 2 点检查证书续期
	global.CRON.AddFunc("0 2 * * *", func() {
		service.AutoRenewCerts()
	})

	global.LOG.Info("Cron scheduler initialized")
}
