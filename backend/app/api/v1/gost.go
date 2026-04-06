package v1

import (
	"xpanel/app/api/v1/helper"
	"xpanel/app/dto"
	"xpanel/app/service"

	"github.com/gin-gonic/gin"
)

type GostAPI struct{}

// --- Install / Status / Operate ---

func (a *GostAPI) GetGostStatus(c *gin.Context) {
	status, err := service.NewIGostInstallService().GetStatus()
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, status)
}

func (a *GostAPI) InstallGost(c *gin.Context) {
	var req dto.GostInstallReq
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewIGostInstallService().Install(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *GostAPI) GetGostInstallProgress(c *gin.Context) {
	progress := service.NewIGostInstallService().GetProgress()
	helper.SuccessWithData(c, progress)
}

func (a *GostAPI) UninstallGost(c *gin.Context) {
	if err := service.NewIGostInstallService().Uninstall(); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *GostAPI) OperateGost(c *gin.Context) {
	var req dto.GostOperateReq
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewIGostInstallService().Operate(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *GostAPI) CheckGostUpdate(c *gin.Context) {
	resp, err := service.NewIGostInstallService().CheckUpdate()
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, resp)
}

func (a *GostAPI) UpgradeGost(c *gin.Context) {
	var req dto.GostUpgradeReq
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewIGostInstallService().Upgrade(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

// --- Gost Service (port forward / relay) ---

func (a *GostAPI) SearchGostService(c *gin.Context) {
	var req dto.GostServiceSearch
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	total, items, err := service.NewIGostService().SearchService(req)
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithPage(c, total, items)
}

func (a *GostAPI) CreateGostService(c *gin.Context) {
	var req dto.GostServiceCreate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewIGostService().CreateService(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *GostAPI) UpdateGostService(c *gin.Context) {
	var req dto.GostServiceUpdate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewIGostService().UpdateService(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *GostAPI) DeleteGostService(c *gin.Context) {
	var req dto.OperateByID
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewIGostService().DeleteService(req.ID); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *GostAPI) ToggleGostService(c *gin.Context) {
	var req dto.GostServiceToggle
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewIGostService().ToggleService(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

// --- Gost Chain ---

func (a *GostAPI) SearchGostChain(c *gin.Context) {
	var req dto.GostChainSearch
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	total, items, err := service.NewIGostService().SearchChain(req)
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithPage(c, total, items)
}

func (a *GostAPI) CreateGostChain(c *gin.Context) {
	var req dto.GostChainCreate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewIGostService().CreateChain(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *GostAPI) UpdateGostChain(c *gin.Context) {
	var req dto.GostChainUpdate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewIGostService().UpdateChain(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *GostAPI) DeleteGostChain(c *gin.Context) {
	var req dto.OperateByID
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewIGostService().DeleteChain(req.ID); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *GostAPI) SyncGost(c *gin.Context) {
	if err := service.NewIGostService().SyncAll(); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}
