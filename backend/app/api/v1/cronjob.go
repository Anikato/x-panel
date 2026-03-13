package v1

import (
	"net/http"

	"xpanel/app/api/v1/helper"
	"xpanel/app/dto"
	"xpanel/app/service"

	"github.com/gin-gonic/gin"
)

type CronjobAPI struct{}

var cronjobService = service.NewICronjobService()

func (a *CronjobAPI) CreateCronjob(c *gin.Context) {
	var req dto.CronjobCreate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.ErrorWithDetail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := cronjobService.Create(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithMsg(c, "MsgCreateSuccess")
}

func (a *CronjobAPI) UpdateCronjob(c *gin.Context) {
	var req dto.CronjobUpdate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.ErrorWithDetail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := cronjobService.Update(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithMsg(c, "MsgUpdateSuccess")
}

func (a *CronjobAPI) DeleteCronjob(c *gin.Context) {
	var req dto.OperateByID
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.ErrorWithDetail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := cronjobService.Delete(req.ID); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithMsg(c, "MsgDeleteSuccess")
}

func (a *CronjobAPI) GetCronjob(c *gin.Context) {
	var req dto.OperateByID
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.ErrorWithDetail(c, http.StatusBadRequest, err.Error())
		return
	}
	info, err := cronjobService.Get(req.ID)
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, info)
}

func (a *CronjobAPI) SearchCronjob(c *gin.Context) {
	var req dto.CronjobSearch
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.ErrorWithDetail(c, http.StatusBadRequest, err.Error())
		return
	}
	total, items, err := cronjobService.SearchWithPage(req)
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithPage(c, total, items)
}

func (a *CronjobAPI) UpdateCronjobStatus(c *gin.Context) {
	var req struct {
		ID     uint   `json:"id" binding:"required"`
		Status string `json:"status" binding:"required"`
	}
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.ErrorWithDetail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := cronjobService.UpdateStatus(req.ID, req.Status); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithMsg(c, "MsgUpdateSuccess")
}

func (a *CronjobAPI) HandleOnceCronjob(c *gin.Context) {
	var req dto.OperateByID
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.ErrorWithDetail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := cronjobService.HandleOnce(req.ID); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *CronjobAPI) SearchCronjobRecords(c *gin.Context) {
	var req dto.CronjobRecordSearch
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.ErrorWithDetail(c, http.StatusBadRequest, err.Error())
		return
	}
	total, items, err := cronjobService.SearchRecords(req)
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithPage(c, total, items)
}
