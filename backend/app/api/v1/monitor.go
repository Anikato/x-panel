package v1

import (
	"xpanel/app/api/v1/helper"
	"xpanel/app/service"

	"github.com/gin-gonic/gin"
)

type MonitorAPI struct{}

func (a *MonitorAPI) GetCurrentStats(c *gin.Context) {
	stats, err := service.NewIMonitorService().GetCurrentStats()
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, stats)
}
