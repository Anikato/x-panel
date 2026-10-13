package v1

import (
	"errors"
	"net/http"

	"xpanel/app/api/v1/helper"
	"xpanel/app/service"

	"github.com/gin-gonic/gin"
)

type terminalSessionRenameReq struct {
	ID   string `json:"id" binding:"required"`
	Name string `json:"name" binding:"required"`
}

type terminalSessionIDReq struct {
	ID string `json:"id" binding:"required"`
}

func (a *TerminalAPI) ListTerminalSessions(c *gin.Context) {
	helper.SuccessWithData(c, service.ListTerminalSessions())
}

func (a *TerminalAPI) RenameTerminalSession(c *gin.Context) {
	var req terminalSessionRenameReq
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.RenameTerminalSession(req.ID, req.Name); err != nil {
		writeTerminalSessionError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

type terminalSessionLockReq struct {
	ID     string `json:"id" binding:"required"`
	Locked bool   `json:"locked"`
	Name   string `json:"name"`
}

func (a *TerminalAPI) LockTerminalSession(c *gin.Context) {
	var req terminalSessionLockReq
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.SetTerminalSessionLocked(req.ID, req.Locked, req.Name); err != nil {
		if errors.Is(err, service.ErrTooManyTerminalSessions) {
			helper.ErrorWithDetail(c, http.StatusBadRequest, "后台终端已有 8 个，先终止一个再留在后台")
			return
		}
		writeTerminalSessionError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func (a *TerminalAPI) CloseTerminalSession(c *gin.Context) {
	var req terminalSessionIDReq
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.HandleError(c, err)
		return
	}
	if err := service.CloseTerminalSession(req.ID); err != nil {
		writeTerminalSessionError(c, err)
		return
	}
	helper.SuccessWithOutData(c)
}

func writeTerminalSessionError(c *gin.Context, err error) {
	if errors.Is(err, service.ErrTerminalSessionNotFound) {
		helper.ErrorWithDetail(c, http.StatusNotFound, "后台终端已经没了")
		return
	}
	helper.ErrorWithDetail(c, http.StatusBadRequest, err.Error())
}
