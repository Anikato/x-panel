package v1

import (
	"xpanel/app/api/v1/helper"
	"xpanel/app/service"

	"github.com/gin-gonic/gin"
)

type SSHKeyAPI struct{}

func (a *SSHKeyAPI) ListSSHKeys(c *gin.Context) {
	data, err := service.NewISSHKeyService().List()
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, data)
}

func (a *SSHKeyAPI) GetSSHPrivateKey(c *gin.Context) {
	name := c.Query("name")
	if name == "" {
		helper.ErrorWithDetail(c, 400, "name is required")
		return
	}
	data, err := service.NewISSHKeyService().GetPrivateKey(name)
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, data)
}

func (a *SSHKeyAPI) GenerateSSHKey(c *gin.Context) {
	var req service.SSHKeyCreate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	info, privateKey, err := service.NewISSHKeyService().Generate(req)
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, map[string]interface{}{
		"info":       info,
		"privateKey": privateKey,
	})
}

func (a *SSHKeyAPI) ImportSSHKey(c *gin.Context) {
	var req service.SSHKeyImport
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewISSHKeyService().Import(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *SSHKeyAPI) DeleteSSHKey(c *gin.Context) {
	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewISSHKeyService().Delete(req.Name); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}
