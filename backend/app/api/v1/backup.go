package v1

import (
	"net/http"
	"os"
	"path"
	"strconv"

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

func (a *BackupAPI) TestBackupAccount(c *gin.Context) {
	var req dto.BackupAccountTest
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.ErrorWithDetail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := backupService.TestAccount(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
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

func (a *BackupAPI) ListStorageObjects(c *gin.Context) {
	var req dto.BackupStorageReq
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.ErrorWithDetail(c, http.StatusBadRequest, err.Error())
		return
	}
	items, err := backupService.ListStorageObjects(req)
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, items)
}

func (a *BackupAPI) ReadStorageObject(c *gin.Context) {
	var req dto.BackupStorageReq
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.ErrorWithDetail(c, http.StatusBadRequest, err.Error())
		return
	}
	content, err := backupService.ReadStorageObject(req)
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, dto.BackupStorageReadResp{Content: content})
}

func (a *BackupAPI) SaveStorageObject(c *gin.Context) {
	var req dto.BackupStorageReq
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.ErrorWithDetail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := backupService.SaveStorageObject(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *BackupAPI) DeleteStorageObject(c *gin.Context) {
	var req dto.BackupStorageReq
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.ErrorWithDetail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := backupService.DeleteStorageObject(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *BackupAPI) UploadStorageObject(c *gin.Context) {
	accountID, err := strconv.ParseUint(c.PostForm("accountID"), 10, 64)
	if err != nil || accountID == 0 {
		helper.ErrorWithDetail(c, http.StatusBadRequest, "accountID is required")
		return
	}
	targetPath := c.PostForm("path")
	prefix := c.PostForm("prefix")
	file, err := c.FormFile("file")
	if err != nil {
		helper.ErrorWithDetail(c, http.StatusBadRequest, err.Error())
		return
	}
	tmp, err := os.CreateTemp("", "xpanel-storage-upload-*")
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	tmpPath := tmp.Name()
	tmp.Close()
	defer os.Remove(tmpPath)
	if err := c.SaveUploadedFile(file, tmpPath); err != nil {
		helper.HandleError(c, err)
		return
	}
	if targetPath == "" {
		targetPath = path.Join(prefix, file.Filename)
	}
	if err := backupService.UploadStorageObject(uint(accountID), targetPath, tmpPath); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *BackupAPI) DownloadStorageObject(c *gin.Context) {
	var req dto.BackupStorageReq
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.ErrorWithDetail(c, http.StatusBadRequest, err.Error())
		return
	}
	tmp, release, err := backupService.PrepareStorageObject(req)
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	defer release()
	c.FileAttachment(tmp, path.Base(req.Path))
}
