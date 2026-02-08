package v1

import (
	"net/http"

	"xpanel/app/api/v1/helper"
	"xpanel/app/dto"
	"xpanel/app/service"

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

	info, err := authService.Login(req)
	if err != nil {
		service.SaveLoginLog(helper.GetClientIP(c), helper.GetUserAgent(c), err)
		helper.HandleError(c, err)
		return
	}

	// MFA 未完成时不记录登录成功日志
	if info.Token != "" {
		service.SaveLoginLog(helper.GetClientIP(c), helper.GetUserAgent(c), nil)
	}

	helper.SuccessWithData(c, info)
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

// Logout 退出登录
func (a *AuthAPI) Logout(c *gin.Context) {
	// JWT 是无状态的，客户端直接删除 token 即可
	// 如需要服务端 token 黑名单，可在此实现
	helper.SuccessWithMsg(c, "MsgLogoutSuccess")
}
