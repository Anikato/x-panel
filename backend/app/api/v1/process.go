package v1

import (
	"xpanel/app/api/v1/helper"
	"xpanel/app/dto"
	"xpanel/app/service"

	"github.com/gin-gonic/gin"
)

type ProcessAPI struct{}

func (a *ProcessAPI) ListProcesses(c *gin.Context) {
	var req dto.ProcessSearchReq
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	items, err := service.NewIProcessService().ListProcesses(req)
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, items)
}

func (a *ProcessAPI) StopProcess(c *gin.Context) {
	var req dto.ProcessStopReq
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewIProcessService().StopProcess(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *ProcessAPI) ListConnections(c *gin.Context) {
	items, err := service.NewIProcessService().ListConnections()
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, items)
}
