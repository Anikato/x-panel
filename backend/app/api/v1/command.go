package v1

import (
	"xpanel/app/api/v1/helper"
	"xpanel/app/dto"
	"xpanel/app/service"

	"github.com/gin-gonic/gin"
)

type CommandAPI struct{}

func (a *CommandAPI) CreateCommand(c *gin.Context) {
	var req dto.CommandCreate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewICommandService().Create(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *CommandAPI) UpdateCommand(c *gin.Context) {
	var req dto.CommandUpdate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewICommandService().Update(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *CommandAPI) DeleteCommand(c *gin.Context) {
	var req dto.OperateByID
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.NewICommandService().Delete(req.ID); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *CommandAPI) SearchCommand(c *gin.Context) {
	var req dto.SearchCommandReq
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	total, items, err := service.NewICommandService().SearchWithPage(req)
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithPage(c, total, items)
}

func (a *CommandAPI) GetCommandTree(c *gin.Context) {
	tree, err := service.NewICommandService().GetTree()
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, tree)
}
