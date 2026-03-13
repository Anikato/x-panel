package v1

import (
	"net/http"

	"xpanel/app/api/v1/helper"
	"xpanel/app/dto"
	"xpanel/app/service"

	"github.com/gin-gonic/gin"
)

type BackupAPI struct{}

var backupService = service.NewIBackupService()

func (a *BackupAPI) CreateBackupAccount(c *gin.Context) {
	var req dto.BackupAccountCreate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.ErrorWithDetail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := backupService.CreateAccount(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithMsg(c, "MsgCreateSuccess")
}

func (a *BackupAPI) UpdateBackupAccount(c *gin.Context) {
	var req dto.BackupAccountUpdate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.ErrorWithDetail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := backupService.UpdateAccount(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithMsg(c, "MsgUpdateSuccess")
}

func (a *BackupAPI) DeleteBackupAccount(c *gin.Context) {
	var req dto.OperateByID
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.ErrorWithDetail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := backupService.DeleteAccount(req.ID); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithMsg(c, "MsgDeleteSuccess")
}

func (a *BackupAPI) ListBackupAccounts(c *gin.Context) {
	items, err := backupService.ListAccounts()
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, items)
}

func (a *BackupAPI) CreateBackup(c *gin.Context) {
	var req dto.BackupCreate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.ErrorWithDetail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := backupService.Backup(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *BackupAPI) SearchBackupRecords(c *gin.Context) {
	var req dto.BackupRecordSearch
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.ErrorWithDetail(c, http.StatusBadRequest, err.Error())
		return
	}
	total, items, err := backupService.SearchRecords(req)
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithPage(c, total, items)
}

func (a *BackupAPI) DeleteBackupRecord(c *gin.Context) {
	var req dto.OperateByID
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.ErrorWithDetail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := backupService.DeleteRecord(req.ID); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithMsg(c, "MsgDeleteSuccess")
}
