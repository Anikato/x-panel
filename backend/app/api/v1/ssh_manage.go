package v1

import (
	"xpanel/app/api/v1/helper"
	"xpanel/app/dto"
	"xpanel/app/service"

	"github.com/gin-gonic/gin"
)

type SSHManageAPI struct{}

func (a *SSHManageAPI) GetSSHInfo(c *gin.Context) {
	info, err := service.NewISSHManageService().GetSSHInfo()
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, info)
}

func (a *SSHManageAPI) OperateSSH(c *gin.Context) {
	var req dto.SSHOperateReq
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewISSHManageService().OperateSSH(req.Operation); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *SSHManageAPI) UpdateSSHConfig(c *gin.Context) {
	var req dto.SSHUpdate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewISSHManageService().UpdateSSHConfig(req.Key, req.Value); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *SSHManageAPI) LoadSSHLog(c *gin.Context) {
	var req dto.SSHLogSearch
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	total, items, err := service.NewISSHManageService().LoadSSHLog(req)
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithPage(c, total, items)
}

func (a *SSHManageAPI) GetSSHDConfig(c *gin.Context) {
	content, err := service.NewISSHManageService().GetSSHDConfig()
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, content)
}

func (a *SSHManageAPI) SaveSSHDConfig(c *gin.Context) {
	var req struct {
		Content string `json:"content" binding:"required"`
	}
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewISSHManageService().SaveSSHDConfig(req.Content); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *SSHManageAPI) ListAuthorizedKeys(c *gin.Context) {
	keys, err := service.NewISSHManageService().ListAuthorizedKeys()
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, keys)
}

func (a *SSHManageAPI) AddAuthorizedKey(c *gin.Context) {
	var req dto.AuthorizedKeyCreate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewISSHManageService().AddAuthorizedKey(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *SSHManageAPI) DeleteAuthorizedKey(c *gin.Context) {
	var req struct {
		Fingerprint string `json:"fingerprint" binding:"required"`
	}
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewISSHManageService().DeleteAuthorizedKey(req.Fingerprint); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}
