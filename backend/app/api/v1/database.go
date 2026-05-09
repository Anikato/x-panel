package v1

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"xpanel/app/api/v1/helper"
	"xpanel/app/dto"
	"xpanel/app/service"
	"xpanel/global"

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

func (a *DatabaseAPI) ChangeInstancePassword(c *gin.Context) {
	var req dto.DatabaseInstanceChangePassword
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.ErrorWithDetail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := databaseService.ChangeInstancePassword(req); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *DatabaseAPI) BackupDatabaseInstance(c *gin.Context) {
	var req dto.OperateByID
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.ErrorWithDetail(c, http.StatusBadRequest, err.Error())
		return
	}
	task, err := databaseService.BackupInstanceAsync(req.ID)
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, map[string]string{"taskID": task.ID})
}

func (a *DatabaseAPI) RestoreDatabaseInstance(c *gin.Context) {
	var req dto.DatabaseInstanceRestore
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.ErrorWithDetail(c, http.StatusBadRequest, err.Error())
		return
	}
	task, err := databaseService.RestoreInstanceAsync(req)
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, map[string]string{"taskID": task.ID})
}

func (a *DatabaseAPI) UploadRestoreFile(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		helper.ErrorWithDetail(c, http.StatusBadRequest, "file is required")
		return
	}
	defer file.Close()

	fileName := filepath.Base(header.Filename)
	if !isSupportedDatabaseRestoreFile(fileName) {
		helper.ErrorWithDetail(c, http.StatusBadRequest, "unsupported backup file type")
		return
	}

	uploadDir := filepath.Join(global.CONF.System.DataDir, "uploads", "database-restore")
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		helper.HandleError(c, err)
		return
	}

	dstName := fmt.Sprintf("%s_%s", time.Now().Format("20060102150405"), fileName)
	dstPath := filepath.Join(uploadDir, dstName)
	dst, err := os.OpenFile(dstPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		_ = os.Remove(dstPath)
		helper.HandleError(c, err)
		return
	}

	helper.SuccessWithData(c, map[string]string{"file": dstPath})
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

func isSupportedDatabaseRestoreFile(fileName string) bool {
	lower := strings.ToLower(fileName)
	suffixes := []string{".sql", ".sql.gz", ".zip", ".tar", ".tar.gz", ".dump"}
	for _, suffix := range suffixes {
		if strings.HasSuffix(lower, suffix) {
			return true
		}
	}
	return false
}
