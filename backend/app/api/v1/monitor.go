package v1

import (
	"net/http"
	"sort"

	"xpanel/app/api/v1/helper"
	"xpanel/app/dto"
	"xpanel/app/service"

	"github.com/gin-gonic/gin"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/net"
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

func (a *MonitorAPI) LoadMonitorHistory(c *gin.Context) {
	var req dto.MonitorSearch
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.ErrorWithDetail(c, http.StatusBadRequest, err.Error())
		return
	}
	data, err := service.NewIMonitorHistoryService().LoadMonitorData(req)
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, data)
}

func (a *MonitorAPI) GetMonitorSetting(c *gin.Context) {
	setting, err := service.NewIMonitorHistoryService().LoadSetting()
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, setting)
}

func (a *MonitorAPI) UpdateMonitorSetting(c *gin.Context) {
	var req dto.MonitorSettingUpdate
	if err := c.ShouldBindJSON(&req); err != nil {
		helper.ErrorWithDetail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := service.NewIMonitorHistoryService().UpdateSetting(req.Key, req.Value); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, nil)
}

func (a *MonitorAPI) CleanMonitorData(c *gin.Context) {
	if err := service.NewIMonitorHistoryService().CleanData(); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, nil)
}

func (a *MonitorAPI) GetIOOptions(c *gin.Context) {
	diskStat, _ := disk.IOCounters()
	options := []string{"all"}
	for _, d := range diskStat {
		options = append(options, d.Name)
	}
	sort.Strings(options[1:])
	helper.SuccessWithData(c, options)
}

func (a *MonitorAPI) GetNetworkOptions(c *gin.Context) {
	netStat, _ := net.IOCounters(true)
	options := []string{"all"}
	for _, n := range netStat {
		options = append(options, n.Name)
	}
	sort.Strings(options[1:])
	helper.SuccessWithData(c, options)
}
