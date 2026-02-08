package dto

// Login 登录请求
type Login struct {
	Name     string `json:"name" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// InitUser 初始化用户（首次设置密码）
type InitUser struct {
	Name     string `json:"name" binding:"required"`
	Password string `json:"password" binding:"required,min=6"`
}

// UserLoginInfo 登录响应
type UserLoginInfo struct {
	Name      string `json:"name"`
	Token     string `json:"token"`
	MfaStatus string `json:"mfaStatus"`
}

// MFALogin MFA 登录
type MFALogin struct {
	Name     string `json:"name" binding:"required"`
	Password string `json:"password" binding:"required"`
	Code     string `json:"code" binding:"required"`
}

// PasswordUpdate 修改密码
type PasswordUpdate struct {
	OldPassword string `json:"oldPassword" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required,min=6"`
}
