package v1

import (
	"net/http"

	"xpanel/app/api/v1/helper"
	"xpanel/app/dto"
	"xpanel/app/service"

	"github.com/gin-gonic/gin"
)

type TrafficAPI struct{}

var trafficService = service.NewITrafficService()

func (a *TrafficAPI) ListConfigs(c *gin.Context) {
	configs, err := trafficService.ListConfigs()
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, configs)
}

func (a *TrafficAPI) CreateConfig(c *gin.Context) {
	var req dto.TrafficConfigCreate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.ErrorWithDetail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := trafficService.CreateConfig(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *TrafficAPI) DeleteConfig(c *gin.Context) {
	var req dto.TrafficDeleteConfig
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.ErrorWithDetail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := trafficService.DeleteConfig(req.InterfaceName); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *TrafficAPI) ListInterfaces(c *gin.Context) {
	interfaces, err := trafficService.ListInterfaces()
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, interfaces)
}

func (a *TrafficAPI) GetStats(c *gin.Context) {
	var req dto.TrafficStatsRequest
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.ErrorWithDetail(c, http.StatusBadRequest, err.Error())
		return
	}
	stats, err := trafficService.GetStats(req)
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, stats)
}

func (a *TrafficAPI) GetSummary(c *gin.Context) {
	summary, err := trafficService.GetSummary()
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, summary)
}
