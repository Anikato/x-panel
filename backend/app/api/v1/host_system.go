package v1

import (
	"net/http"

	"xpanel/app/api/v1/helper"
	"xpanel/app/service"

	"github.com/gin-gonic/gin"
)

type HostSystemAPI struct{}

// ====== Linux User ======

func (a *HostSystemAPI) ListUsers(c *gin.Context) {
	showSystem := c.Query("system") == "true"
	data, err := service.NewIHostUserService().List(showSystem)
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, data)
}

func (a *HostSystemAPI) CreateUser(c *gin.Context) {
	var req service.LinuxUserCreate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewIHostUserService().Create(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *HostSystemAPI) UpdateUser(c *gin.Context) {
	var req service.LinuxUserUpdate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewIHostUserService().Update(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *HostSystemAPI) DeleteUser(c *gin.Context) {
	var req service.LinuxUserDelete
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewIHostUserService().Delete(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *HostSystemAPI) ListShells(c *gin.Context) {
	data, err := service.NewIHostUserService().ListShells()
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, data)
}

func (a *HostSystemAPI) ListGroups(c *gin.Context) {
	data, err := service.NewIHostUserService().ListGroups()
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, data)
}

// ====== Host System ======

func (a *HostSystemAPI) GetSystemInfo(c *gin.Context) {
	data, err := service.NewIHostSystemService().GetInfo()
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, data)
}

func (a *HostSystemAPI) SetHostname(c *gin.Context) {
	var req struct {
		Hostname string `json:"hostname" validate:"required"`
	}
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewIHostSystemService().SetHostname(req.Hostname); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *HostSystemAPI) SetTimezone(c *gin.Context) {
	var req struct {
		Timezone string `json:"timezone" validate:"required"`
	}
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewIHostSystemService().SetTimezone(req.Timezone); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *HostSystemAPI) ListTimezones(c *gin.Context) {
	data, err := service.NewIHostSystemService().ListTimezones()
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, data)
}

func (a *HostSystemAPI) GetDNS(c *gin.Context) {
	data, err := service.NewIHostSystemService().GetDNS()
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, data)
}

func (a *HostSystemAPI) SetDNS(c *gin.Context) {
	var req struct {
		Servers []string `json:"servers" validate:"required"`
	}
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewIHostSystemService().SetDNS(req.Servers); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *HostSystemAPI) GetSwap(c *gin.Context) {
	data, err := service.NewIHostSystemService().GetSwap()
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, data)
}

func (a *HostSystemAPI) CreateSwap(c *gin.Context) {
	var req struct {
		SizeMB int `json:"sizeMB" validate:"required,min=64"`
	}
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewIHostSystemService().CreateSwap(req.SizeMB); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *HostSystemAPI) DeleteSwap(c *gin.Context) {
	if err := service.NewIHostSystemService().DeleteSwap(); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *HostSystemAPI) SwapOperate(c *gin.Context) {
	op := c.Query("op")
	var err error
	switch op {
	case "on":
		err = service.NewIHostSystemService().SwapOn()
	case "off":
		err = service.NewIHostSystemService().SwapOff()
	default:
		helper.ErrorWithDetail(c, http.StatusBadRequest, "op must be 'on' or 'off'")
		return
	}
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

