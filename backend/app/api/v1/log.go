package v1

import (
	"net/http"
	"strconv"
	"strings"

	"xpanel/app/api/v1/helper"
	"xpanel/app/dto"
	"xpanel/app/model"
	"xpanel/app/service"
	"xpanel/global"

	"github.com/gin-gonic/gin"
)

// LogAPI 日志接口
type LogAPI struct{}

var logService = service.NewILogService()

func (l *LogAPI) PagePanelAccessLog(c *gin.Context) {
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 || page > 10000 {
		helper.ErrorWithDetail(c, http.StatusBadRequest, "页码无效")
		return
	}
	pageSize, err := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	if err != nil || pageSize < 1 || pageSize > 100 {
		helper.ErrorWithDetail(c, http.StatusBadRequest, "分页大小无效")
		return
	}
	total, logs, err := service.PagePanelAccessLogs(global.DB, page, pageSize, strings.TrimSpace(c.Query("ip")), model.PanelAccessKind(c.Query("kind")))
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithPage(c, total, logs)
}

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

// GetSystemLog 获取系统日志
func (l *LogAPI) GetSystemLog(c *gin.Context) {
	linesStr := c.DefaultQuery("lines", "100")
	level := c.Query("level")
	keyword := c.Query("keyword")

	lines, err := strconv.Atoi(linesStr)
	if err != nil || lines <= 0 {
		lines = 100
	}
	if lines > 5000 {
		lines = 5000
	}

	logs, err := logService.GetSystemLog(lines, level, keyword)
	if err != nil {
		helper.HandleError(c, err)
		return
	}

	helper.SuccessWithData(c, logs)
}

// CleanSystemLog 清空系统日志
func (l *LogAPI) CleanSystemLog(c *gin.Context) {
	if err := logService.CleanSystemLog(); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithMsg(c, "MsgDeleteSuccess")
}
