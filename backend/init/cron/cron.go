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

	// 每天凌晨 2 点检查证书续期
	global.CRON.AddFunc("0 2 * * *", func() {
		service.AutoRenewCerts()
	})

	global.LOG.Info("Cron scheduler initialized")
}
