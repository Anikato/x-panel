package v1

import (
	"net/http"

	"xpanel/app/api/v1/helper"
	"xpanel/app/dto"
	"xpanel/app/service"

	"github.com/gin-gonic/gin"
)

type NodeAPI struct{}

var nodeService = service.NewINodeService()

func (a *NodeAPI) CreateNode(c *gin.Context) {
	var req dto.NodeCreate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.ErrorWithDetail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := nodeService.Create(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithMsg(c, "MsgCreateSuccess")
}

func (a *NodeAPI) UpdateNode(c *gin.Context) {
	var req dto.NodeUpdate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.ErrorWithDetail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := nodeService.Update(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithMsg(c, "MsgUpdateSuccess")
}

func (a *NodeAPI) DeleteNode(c *gin.Context) {
	var req dto.OperateByID
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.ErrorWithDetail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := nodeService.Delete(req.ID); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithMsg(c, "MsgDeleteSuccess")
}

func (a *NodeAPI) ListNodes(c *gin.Context) {
	items, err := nodeService.List()
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, items)
}

func (a *NodeAPI) TestNodeConnection(c *gin.Context) {
	var req dto.OperateByID
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.ErrorWithDetail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := nodeService.TestConnection(req.ID); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}
