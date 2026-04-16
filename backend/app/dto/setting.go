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
	ServerPort       string `json:"serverPort"`
	AgentToken       string `json:"agentToken"`
	AutoUpgrade      string `json:"autoUpgrade"`
	AppearanceConfig string `json:"appearanceConfig"`
	ProxyEnable      string `json:"proxyEnable"`
	ProxyType        string `json:"proxyType"`
	ProxyAddress     string `json:"proxyAddress"`
	ProxyNoProxy     string `json:"proxyNoProxy"`
}

// ProxyTest 代理连通性测试请求
type ProxyTest struct {
	Address string `json:"address" binding:"required"`
}

// SettingUpdate 设置更新请求
type SettingUpdate struct {
	Key   string `json:"key" binding:"required"`
	Value string `json:"value"`
}

// PortUpdate 端口更新请求
type PortUpdate struct {
	Port string `json:"port" binding:"required"`
}

// LoginSetting 登录页面需要的设置
type LoginSetting struct {
	Language  string `json:"language"`
	PanelName string `json:"panelName"`
	Theme     string `json:"theme"`
}
