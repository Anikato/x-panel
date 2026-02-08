package v1

import (
	"strconv"

	"xpanel/app/api/v1/helper"
	"xpanel/app/dto"
	"xpanel/app/service"

	"github.com/gin-gonic/gin"
)

type HostAPI struct{}

func (a *HostAPI) CreateHost(c *gin.Context) {
	var req dto.HostCreate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewIHostService().Create(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *HostAPI) UpdateHost(c *gin.Context) {
	var req dto.HostUpdate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewIHostService().Update(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *HostAPI) DeleteHost(c *gin.Context) {
	var req dto.OperateByID
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewIHostService().Delete(req.ID); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *HostAPI) SearchHost(c *gin.Context) {
	var req dto.SearchHostReq
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	total, items, err := service.NewIHostService().SearchWithPage(req)
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithPage(c, total, items)
}

func (a *HostAPI) GetHostTree(c *gin.Context) {
	tree, err := service.NewIHostService().GetTree()
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, tree)
}

func (a *HostAPI) TestHost(c *gin.Context) {
	idStr := c.Query("id")
	id, _ := strconv.ParseUint(idStr, 10, 64)
	if id == 0 {
		helper.SuccessWithData(c, true)
		return
	}
	result := service.NewIHostService().TestByID(uint(id))
	helper.SuccessWithData(c, result)
}

func (a *HostAPI) TestHostConn(c *gin.Context) {
	var req dto.HostCreate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	err := service.TestHostConn(req.Addr, req.Port, req.User, req.AuthMode, req.Password, req.PrivateKey, req.PassPhrase)
	if err != nil {
		helper.SuccessWithData(c, false)
		return
	}
	helper.SuccessWithData(c, true)
}
