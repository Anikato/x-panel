package dto

// SettingInfo 面板设置信息
type SettingInfo struct {
	UserName                string `json:"userName"`
	Language                string `json:"language"`
	SessionTimeout          string `json:"sessionTimeout"`
	PanelName               string `json:"panelName"`
	Theme                   string `json:"theme"`
	SecurityEntrance        string `json:"-"`
	SecurityEntranceSet     bool   `json:"securityEntranceSet"`
	MFAStatus               string `json:"mfaStatus"`
	UpgradeURL              string `json:"upgradeUrl"`
	GitHubToken             string `json:"-"`
	GitHubTokenSet          bool   `json:"githubTokenSet"`
	ServerPort              string `json:"serverPort"`
	AgentToken              string `json:"-"`
	AgentTokenSet           bool   `json:"agentTokenSet"`
	AutoUpgrade             string `json:"autoUpgrade"`
	AppearanceConfig        string `json:"appearanceConfig"`
	ManagedAppearanceConfig string `json:"managedAppearanceConfig,omitempty"`
	ProxyEnable             string `json:"proxyEnable"`
	ProxyType               string `json:"proxyType"`
	ProxyAddress            string `json:"-"`
	ProxyAddressSet         bool   `json:"proxyAddressSet"`
	ProxyNoProxy            string `json:"proxyNoProxy"`
	ProxyDocker             bool   `json:"proxyDocker"`
	ProxyScope              string `json:"proxyScope"`
	DockerRestartPending    bool   `json:"dockerRestartPending"`
	LegacySystemProxy       bool   `json:"legacySystemProxy"`
}

// ProxyTest 代理连通性测试请求
type ProxyTest struct {
	Address string `json:"address"`
	Type    string `json:"type" binding:"omitempty,oneof=mix http socks5"`
}

type ProxyConfigUpdate struct {
	Type         string `json:"type" binding:"required,oneof=mix http socks5"`
	Address      string `json:"address"`
	ClearAddress bool   `json:"clearAddress"`
	NoProxy      string `json:"noProxy"`
	Scope        string `json:"scope" binding:"omitempty,oneof=off panel panel-docker system"`
	Enabled      bool   `json:"enabled"`
	Docker       bool   `json:"docker"`
}

type ProxyConfigResult struct {
	DockerChanged        bool `json:"dockerChanged"`
	DockerRestartPending bool `json:"dockerRestartPending"`
	LegacySystemProxy    bool `json:"legacySystemProxy"`
}

// SettingUpdate 设置更新请求
type SettingUpdate struct {
	Key   string `json:"key" binding:"required"`
	Value string `json:"value"`
	Clear bool   `json:"clear"`
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

// PanelSSLInfo 面板 HTTPS 当前配置（路径来自 config.yaml，证书 ID 来自设置表回显）
type PanelSSLInfo struct {
	Enable        bool   `json:"enable"`
	CertPath      string `json:"certPath"`
	KeyPath       string `json:"keyPath"`
	CertificateID uint   `json:"certificateId"`
	PrimaryDomain string `json:"primaryDomain,omitempty"`
}

// PanelSSLUpdate 将面板 HTTPS 切换为证书管理中某张证书的落盘文件
type PanelSSLUpdate struct {
	CertificateID uint `json:"certificateId" binding:"required,gt=0"`
}

type CredentialRotationInfo struct {
	KeyID string `json:"keyId"`
}
