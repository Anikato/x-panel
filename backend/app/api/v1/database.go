package v1

import (
	"net/http"

	"xpanel/app/api/v1/helper"
	"xpanel/app/dto"
	"xpanel/app/service"

	"github.com/gin-gonic/gin"
)

type DatabaseAPI struct{}

var databaseService = service.NewIDatabaseService()

func (a *DatabaseAPI) CreateDatabaseServer(c *gin.Context) {
	var req dto.DatabaseServerCreate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.ErrorWithDetail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := databaseService.CreateServer(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithMsg(c, "MsgCreateSuccess")
}

func (a *DatabaseAPI) UpdateDatabaseServer(c *gin.Context) {
	var req dto.DatabaseServerUpdate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.ErrorWithDetail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := databaseService.UpdateServer(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithMsg(c, "MsgUpdateSuccess")
}

func (a *DatabaseAPI) DeleteDatabaseServer(c *gin.Context) {
	var req dto.OperateByID
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.ErrorWithDetail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := databaseService.DeleteServer(req.ID); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithMsg(c, "MsgDeleteSuccess")
}

func (a *DatabaseAPI) SearchDatabaseServer(c *gin.Context) {
	var req dto.DatabaseServerSearch
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.ErrorWithDetail(c, http.StatusBadRequest, err.Error())
		return
	}
	total, items, err := databaseService.SearchServer(req)
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithPage(c, total, items)
}

func (a *DatabaseAPI) TestDatabaseConnection(c *gin.Context) {
	var req dto.OperateByID
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.ErrorWithDetail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := databaseService.TestConnection(req.ID); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *DatabaseAPI) CreateDatabaseInstance(c *gin.Context) {
	var req dto.DatabaseInstanceCreate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.ErrorWithDetail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := databaseService.CreateInstance(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithMsg(c, "MsgCreateSuccess")
}

func (a *DatabaseAPI) DeleteDatabaseInstance(c *gin.Context) {
	var req dto.OperateByID
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.ErrorWithDetail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := databaseService.DeleteInstance(req.ID); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithMsg(c, "MsgDeleteSuccess")
}

func (a *DatabaseAPI) SearchDatabaseInstance(c *gin.Context) {
	var req dto.DatabaseInstanceSearch
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.ErrorWithDetail(c, http.StatusBadRequest, err.Error())
		return
	}
	total, items, err := databaseService.SearchInstance(req)
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithPage(c, total, items)
}

func (a *DatabaseAPI) SyncDatabaseInstances(c *gin.Context) {
	var req dto.OperateByID
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.ErrorWithDetail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := databaseService.SyncInstances(req.ID); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}
