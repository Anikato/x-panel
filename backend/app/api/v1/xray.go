package v1

import (
	"xpanel/app/api/v1/helper"
	"xpanel/app/dto"
	"xpanel/app/service"

	"github.com/gin-gonic/gin"
)

type XrayAPI struct{}

var xrayService = service.NewIXrayService()

func (a *XrayAPI) ControlXrayService(c *gin.Context) {
	var req struct {
		Action string `json:"action" binding:"required,oneof=start stop restart enable disable"`
	}
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := xrayService.ControlService(req.Action); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, xrayService.GetStatus())
}

func (a *XrayAPI) FixXrayPermissions(c *gin.Context) {
	if err := xrayService.FixPermissions(); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *XrayAPI) GetXrayLogSettings(c *gin.Context) {
	helper.SuccessWithData(c, xrayService.GetLogSettings())
}

func (a *XrayAPI) UpdateXrayLogSettings(c *gin.Context) {
	var req dto.XrayLogSettings
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := xrayService.UpdateLogSettings(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *XrayAPI) CheckXrayUpdate(c *gin.Context) {
	info, err := xrayService.CheckUpdate()
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, info)
}

func (a *XrayAPI) DoXrayUpgrade(c *gin.Context) {
	if err := xrayService.DoUpgrade(); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *XrayAPI) ListXrayOutbounds(c *gin.Context) {
	list, err := xrayService.ListOutbounds()
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, list)
}

func (a *XrayAPI) CreateXrayOutbound(c *gin.Context) {
	var req dto.XrayOutboundCreate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := xrayService.CreateOutbound(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *XrayAPI) UpdateXrayOutbound(c *gin.Context) {
	var req dto.XrayOutboundUpdate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := xrayService.UpdateOutbound(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *XrayAPI) DeleteXrayOutbound(c *gin.Context) {
	var req dto.OperateByID
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := xrayService.DeleteOutbound(req.ID); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *XrayAPI) GetXrayStatus(c *gin.Context) {
	helper.SuccessWithData(c, xrayService.GetStatus())
}

func (a *XrayAPI) ListXrayNodes(c *gin.Context) {
	nodes, err := xrayService.ListNodes()
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, nodes)
}

func (a *XrayAPI) CreateXrayNode(c *gin.Context) {
	var req dto.XrayNodeCreate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := xrayService.CreateNode(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *XrayAPI) UpdateXrayNode(c *gin.Context) {
	var req dto.XrayNodeUpdate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := xrayService.UpdateNode(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *XrayAPI) DeleteXrayNode(c *gin.Context) {
	var req dto.OperateByID
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := xrayService.DeleteNode(req.ID); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *XrayAPI) ToggleXrayNode(c *gin.Context) {
	var req dto.OperateByID
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := xrayService.ToggleNode(req.ID); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *XrayAPI) StartInstall(c *gin.Context) {
	if err := xrayService.StartInstall(); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *XrayAPI) GetInstallLog(c *gin.Context) {
	log := xrayService.GetInstallLog()
	helper.SuccessWithData(c, gin.H{
		"log":     log,
		"running": xrayService.IsInstallRunning(),
	})
}

func (a *XrayAPI) GetTrafficHistory(c *gin.Context) {
	var req dto.OperateByID
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	rows, err := xrayService.GetTrafficHistory(req.ID, 30)
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, rows)
}

func (a *XrayAPI) SearchXrayUsers(c *gin.Context) {
	var req dto.XrayUserSearch
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	total, items, err := xrayService.SearchUsers(req)
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithPage(c, total, items)
}

func (a *XrayAPI) CreateXrayUser(c *gin.Context) {
	var req dto.XrayUserCreate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := xrayService.CreateUser(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *XrayAPI) UpdateXrayUser(c *gin.Context) {
	var req dto.XrayUserUpdate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := xrayService.UpdateUser(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *XrayAPI) DeleteXrayUser(c *gin.Context) {
	var req dto.OperateByID
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := xrayService.DeleteUser(req.ID); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *XrayAPI) GenerateRealityKeys(c *gin.Context) {
	resp, err := xrayService.GenerateRealityKeys()
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, resp)
}

func (a *XrayAPI) GetXrayShareLink(c *gin.Context) {
	var req dto.OperateByID
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	resp, err := xrayService.GetShareLink(req.ID)
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, resp)
}
