package v1

import (
	"net/http"

	"xpanel/app/api/v1/helper"
	"xpanel/app/dto"
	"xpanel/app/service"
	"xpanel/global"
	"xpanel/utils/accessticket"
	"xpanel/utils/captcha"

	"github.com/gin-gonic/gin"
)

// AuthAPI 认证接口
type AuthAPI struct{}

var authService = service.NewIAuthService()

// Login 用户登录
func (a *AuthAPI) Login(c *gin.Context) {
	var req dto.Login
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.ErrorWithDetail(c, http.StatusBadRequest, err.Error())
		return
	}

	clientIP := helper.GetClientIP(c)

	if global.IPTracker != nil && global.IPTracker.NeedCaptcha(clientIP) {
		if req.CaptchaID == "" || req.Captcha == "" || !captcha.VerifyCaptcha(req.CaptchaID, req.Captcha) {
			helper.SuccessWithData(c, &dto.UserLoginInfo{NeedCaptcha: true})
			return
		}
	}

	info, err := authService.Login(req)
	if err != nil {
		if global.IPTracker != nil {
			global.IPTracker.IncrementFail(clientIP)
		}
		needCaptcha := global.IPTracker != nil && global.IPTracker.NeedCaptcha(clientIP)
		service.SaveLoginLog(clientIP, helper.GetUserAgent(c), err)
		helper.SuccessWithData(c, &dto.UserLoginInfo{NeedCaptcha: needCaptcha})
		return
	}

	if global.IPTracker != nil {
		global.IPTracker.Clear(clientIP)
	}

	if info.Token != "" {
		service.SaveLoginLog(clientIP, helper.GetUserAgent(c), nil)
	}

	helper.SuccessWithData(c, info)
}

// GetCaptcha 获取验证码
func (a *AuthAPI) GetCaptcha(c *gin.Context) {
	id, imgData, err := captcha.CreateCaptcha()
	if err != nil {
		helper.ErrorWithDetail(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.SuccessWithData(c, dto.CaptchaResponse{CaptchaID: id, ImageData: imgData})
}

// InitUser 初始化用户（首次设置密码）
func (a *AuthAPI) InitUser(c *gin.Context) {
	var req dto.InitUser
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.ErrorWithDetail(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := authService.InitUser(req); err != nil {
		helper.HandleError(c, err)
		return
	}

	helper.SuccessWithOutData(c)
}

// CheckIsInitialized 检查是否已初始化
func (a *AuthAPI) CheckIsInitialized(c *gin.Context) {
	helper.SuccessWithData(c, authService.IsInitialized())
}

// GetLoginSetting 获取登录页面设置
func (a *AuthAPI) GetLoginSetting(c *gin.Context) {
	setting, err := authService.GetLoginSetting()
	if err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithData(c, setting)
}

// UpdatePassword 修改密码
func (a *AuthAPI) UpdatePassword(c *gin.Context) {
	var req dto.PasswordUpdate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.ErrorWithDetail(c, http.StatusBadRequest, err.Error())
		return
	}

	userName, _ := c.Get("userName")
	if err := authService.UpdatePassword(userName.(string), req); err != nil {
		helper.HandleError(c, err)
		return
	}

	helper.SuccessWithMsg(c, "MsgUpdateSuccess")
}

func (a *AuthAPI) Logout(c *gin.Context) {
	sessionID, _ := c.Get("sessionID")
	id, _ := sessionID.(string)
	if err := service.RevokeSession(id); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithMsg(c, "MsgLogoutSuccess")
}

func (a *AuthAPI) LogoutOthers(c *gin.Context) {
	userName, _ := c.Get("userName")
	sessionID, _ := c.Get("sessionID")
	if err := service.RevokeOtherSessions(userName.(string), sessionID.(string)); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithMsg(c, "MsgLogoutSuccess")
}

func (a *AuthAPI) LogoutAll(c *gin.Context) {
	userName, _ := c.Get("userName")
	if err := service.RevokeUserSessions(userName.(string)); err != nil {
		helper.HandleError(c, err)
		return
	}
	helper.SuccessWithMsg(c, "MsgLogoutSuccess")
}

func (a *AuthAPI) IssueAccessTicket(c *gin.Context) {
	if src, _ := c.Get("authSource"); src != "jwt" {
		helper.ErrorWithDetail(c, http.StatusUnauthorized, "login session required")
		return
	}
	var req dto.AccessTicketRequest
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		helper.ErrorWithDetail(c, http.StatusBadRequest, err.Error())
		return
	}
	if req.Scope == "download" && req.Path == "" {
		helper.ErrorWithDetail(c, http.StatusBadRequest, "path is required")
		return
	}
	userName, _ := c.Get("userName")
	sessionID, _ := c.Get("sessionID")
	name, _ := userName.(string)
	sid, _ := sessionID.(string)
	spec := accessticket.TerminalSpec(name, sid)
	if req.Scope == "download" {
		spec = accessticket.DownloadSpec(name, sid, req.Path)
	}
	ticket, err := accessticket.Default.Issue(spec)
	if err != nil {
		helper.ErrorWithDetail(c, http.StatusInternalServerError, err.Error())
		return
	}
	helper.SuccessWithData(c, dto.AccessTicketResponse{Ticket: ticket})
}
