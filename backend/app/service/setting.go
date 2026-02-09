package service

import (
	"fmt"
	"strconv"

	"xpanel/app/dto"
	"xpanel/buserr"
	"xpanel/constant"
	"xpanel/global"
)

// ISettingService 面板设置服务接口
type ISettingService interface {
	GetSettingInfo() (*dto.SettingInfo, error)
	Update(req dto.SettingUpdate) error
	UpdatePort(req dto.PortUpdate) error
	GetValueByKey(key string) (string, error)
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
		UserName:         settingMap["UserName"],
		Language:         settingMap["Language"],
		SessionTimeout:   settingMap["SessionTimeout"],
		PanelName:        settingMap["PanelName"],
		Theme:            settingMap["Theme"],
		SecurityEntrance: settingMap["SecurityEntrance"],
		MFAStatus:        settingMap["MFAStatus"],
		GitHubToken:      settingMap["GitHubToken"],
		ServerPort:       global.CONF.System.Port,
	}, nil
}

func (s *SettingService) Update(req dto.SettingUpdate) error {
	// 验证 Key 是否合法
	allowedKeys := map[string]bool{
		"Language": true, "SessionTimeout": true,
		"PanelName": true, "Theme": true,
		"SecurityEntrance": true, "GitHubToken": true,
		"UserName": true,
	}
	if !allowedKeys[req.Key] {
		return buserr.New(constant.ErrInvalidParams)
	}

	return settingRepo.Update(req.Key, req.Value)
}

func (s *SettingService) UpdatePort(req dto.PortUpdate) error {
	// 验证端口合法性
	port, err := strconv.Atoi(req.Port)
	if err != nil || port < 1 || port > 65535 {
		return buserr.New(constant.ErrInvalidParams)
	}

	// 更新 Viper 配置并写入文件
	if global.Vp == nil {
		return fmt.Errorf("viper instance not initialized")
	}
	global.Vp.Set("system.port", req.Port)
	if err := global.Vp.WriteConfig(); err != nil {
		return fmt.Errorf("failed to write config: %v", err)
	}

	// 更新运行时配置
	global.CONF.System.Port = req.Port
	return nil
}

func (s *SettingService) GetValueByKey(key string) (string, error) {
	return settingRepo.GetValueByKey(key)
}
