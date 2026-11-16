package service

import (
	"fmt"
	"strconv"

	"xpanel/app/dto"
	"xpanel/app/repo"
	"xpanel/buserr"
	"xpanel/constant"
	"xpanel/global"
	"xpanel/security/credentials"
)

// ISettingService 面板设置服务接口
type ISettingService interface {
	GetSettingInfo() (*dto.SettingInfo, error)
	Update(req dto.SettingUpdate) error
	UpdatePort(req dto.PortUpdate) error
	GetValueByKey(key string) (string, error)
	TestProxy(req dto.ProxyTest) error
	SaveProxy(req dto.ProxyConfigUpdate) (*dto.ProxyConfigResult, error)
	GetPanelSSL() (*dto.PanelSSLInfo, error)
	UpdatePanelSSL(req dto.PanelSSLUpdate) error
}

// NewISettingService 创建设置服务实例
func NewISettingService() ISettingService {
	return &SettingService{}
}

type SettingService struct{}

func (s *SettingService) GetSettingInfo() (*dto.SettingInfo, error) {
	settings, err := settingRepo.GetList()
	if err != nil {
		return nil, err
	}

	settingMap := make(map[string]string)
	for _, item := range settings {
		settingMap[item.Key] = item.Value
	}

	return &dto.SettingInfo{
		UserName:            settingMap["UserName"],
		Language:            settingMap["Language"],
		SessionTimeout:      settingMap["SessionTimeout"],
		PanelName:           settingMap["PanelName"],
		Theme:               settingMap["Theme"],
		SecurityEntranceSet: settingMap["SecurityEntrance"] != "",
		MFAStatus:           settingMap["MFAStatus"],
		UpgradeURL:          settingMap["UpgradeURL"],
		GitHubTokenSet:      settingMap["GitHubToken"] != "",
		ServerPort:          global.CONF.System.Port,
		AgentTokenSet:       settingMap["AgentToken"] != "",
		AutoUpgrade:         settingMap["AutoUpgrade"],
		AppearanceConfig:    settingMap["AppearanceConfig"],
		ProxyEnable:         settingMap["ProxyEnable"],
		ProxyType:           settingMap["ProxyType"],
		ProxyAddressSet:     settingMap["ProxyAddress"] != "",
		ProxyNoProxy:        settingMap["ProxyNoProxy"],
		ProxyDocker:         settingMap["ProxyDocker"] == "enable",
		LegacySystemProxy:   hasLegacySystemProxy(),
	}, nil
}

func (s *SettingService) Update(req dto.SettingUpdate) error {
	allowedKeys := map[string]bool{
		"Language": true, "SessionTimeout": true,
		"PanelName": true, "Theme": true,
		"SecurityEntrance": true, "UpgradeURL": true, "GitHubToken": true,
		"UserName": true, "AgentToken": true,
		"AutoUpgrade": true, "AppearanceConfig": true,
		"ProxyEnable": true, "ProxyType": true,
		"ProxyAddress": true, "ProxyNoProxy": true,
	}
	if !allowedKeys[req.Key] {
		return buserr.New(constant.ErrInvalidParams)
	}
	if req.Clear {
		if !credentials.IsSecretSetting(req.Key) {
			return buserr.New(constant.ErrInvalidParams)
		}
		req.Value = ""
	} else if credentials.IsSecretSetting(req.Key) && req.Value == "" {
		return nil
	}

	switch req.Key {
	case "ProxyEnable", "ProxyType", "ProxyAddress", "ProxyNoProxy":
		return s.updateLegacyProxySetting(req)
	}
	return settingRepo.Update(req.Key, req.Value)
}

func (s *SettingService) UpdatePort(req dto.PortUpdate) error {
	port, err := strconv.Atoi(req.Port)
	if err != nil || port < 1 || port > 65535 {
		return buserr.New(constant.ErrInvalidParams)
	}

	if global.Vp == nil {
		return fmt.Errorf("viper instance not initialized")
	}
	global.Vp.Set("system.port", req.Port)
	if err := global.Vp.WriteConfig(); err != nil {
		return fmt.Errorf("failed to write config: %v", err)
	}

	global.CONF.System.Port = req.Port
	return nil
}

// GetPanelSSL 返回当前 config 中的面板 TLS 配置及上次在 UI 选择的证书 ID（若有）
func (s *SettingService) GetPanelSSL() (*dto.PanelSSLInfo, error) {
	info := &dto.PanelSSLInfo{
		Enable:   global.CONF.System.SSL.Enable,
		CertPath: global.CONF.System.SSL.CertPath,
		KeyPath:  global.CONF.System.SSL.KeyPath,
	}
	idStr, err := settingRepo.GetValueByKey("PanelSSLCertificateID")
	if err == nil && idStr != "" {
		if id, e := strconv.ParseUint(idStr, 10, 64); e == nil {
			info.CertificateID = uint(id)
		}
	}
	if info.CertificateID > 0 {
		cert, err := repo.NewICertificateRepo().Get(repo.WithByID(info.CertificateID))
		if err == nil {
			info.PrimaryDomain = cert.PrimaryDomain
		}
	}
	return info, nil
}

// UpdatePanelSSL 将面板 HTTPS 证书切换为证书管理中指定记录对应落盘文件，并写回 config.yaml
func (s *SettingService) UpdatePanelSSL(req dto.PanelSSLUpdate) error {
	if req.CertificateID == 0 {
		return buserr.New(constant.ErrInvalidParams)
	}
	certPath, keyPath, err := NewICertificateService().ResolveCertFilePaths(req.CertificateID)
	if err != nil {
		return err
	}
	if global.Vp == nil {
		return fmt.Errorf("viper instance not initialized")
	}
	global.Vp.Set("system.ssl.enable", true)
	global.Vp.Set("system.ssl.cert_path", certPath)
	global.Vp.Set("system.ssl.key_path", keyPath)
	if err := global.Vp.WriteConfig(); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}
	global.CONF.System.SSL.Enable = true
	global.CONF.System.SSL.CertPath = certPath
	global.CONF.System.SSL.KeyPath = keyPath
	idStr := strconv.FormatUint(uint64(req.CertificateID), 10)
	return settingRepo.CreateOrUpdate("PanelSSLCertificateID", idStr)
}

func (s *SettingService) GetValueByKey(key string) (string, error) {
	return settingRepo.GetValueByKey(key)
}
