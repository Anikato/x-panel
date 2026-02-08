package dto

// SettingInfo 面板设置信息
type SettingInfo struct {
	UserName         string `json:"userName"`
	Language         string `json:"language"`
	SessionTimeout   string `json:"sessionTimeout"`
	PanelName        string `json:"panelName"`
	Theme            string `json:"theme"`
	SecurityEntrance string `json:"securityEntrance"`
	MFAStatus        string `json:"mfaStatus"`
	GitHubToken      string `json:"githubToken"`
}

// SettingUpdate 设置更新请求
type SettingUpdate struct {
	Key   string `json:"key" binding:"required"`
	Value string `json:"value"`
}

// LoginSetting 登录页面需要的设置
type LoginSetting struct {
	Language  string `json:"language"`
	PanelName string `json:"panelName"`
	Theme     string `json:"theme"`
}
