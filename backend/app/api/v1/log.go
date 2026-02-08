package v1

import (
	"net/http"

	"xpanel/app/api/v1/helper"
	"xpanel/app/dto"
	"xpanel/app/service"

	"github.com/gin-gonic/gin"
)

// LogAPI 日志接口
type LogAPI struct{}

var logService = service.NewILogService()

// PageLoginLog 分页查询登录日志
func (l *LogAPI) PageLoginLog(c *gin.Context) {
	var req dto.SearchWithPage
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.ErrorWithDetail(c, http.StatusBadRequest, err.Error())
		return
	}

	total, logs, err := logService.PageLoginLog(req)
	if err != nil {
		helper.HandleError(c, err)
		return
	}

	helper.SuccessWithPage(c, total, logs)
}

// PageOperationLog 分页查询操作日志
func (l *LogAPI) PageOperationLog(c *gin.Context) {
	var req dto.SearchWithPage
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.ErrorWithDetail(c, http.StatusBadRequest, err.Error())
		return
	}

	total, logs, err := logService.PageOperationLog(req)
	if err != nil {
		helper.HandleError(c, err)
		return
	}

	helper.SuccessWithPage(c, total, logs)
}

// CleanLoginLog 清空登录日志
func (l *LogAPI) CleanLoginLog(c *gin.Context) {
	if err := logService.CleanLoginLog(); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithMsg(c, "MsgDeleteSuccess")
}

// CleanOperationLog 清空操作日志
func (l *LogAPI) CleanOperationLog(c *gin.Context) {
	if err := logService.CleanOperationLog(); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithMsg(c, "MsgDeleteSuccess")
}
