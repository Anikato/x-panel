package service

import (
	"xpanel/app/dto"
	"xpanel/buserr"
	"xpanel/constant"
)

// ISettingService 面板设置服务接口
type ISettingService interface {
	GetSettingInfo() (*dto.SettingInfo, error)
	Update(req dto.SettingUpdate) error
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
	}, nil
}

func (s *SettingService) Update(req dto.SettingUpdate) error {
	// 验证 Key 是否合法
	allowedKeys := map[string]bool{
		"Language": true, "SessionTimeout": true,
		"PanelName": true, "Theme": true,
		"SecurityEntrance": true, "GitHubToken": true,
	}
	if !allowedKeys[req.Key] {
		return buserr.New(constant.ErrInvalidParams)
	}

	return settingRepo.Update(req.Key, req.Value)
}

func (s *SettingService) GetValueByKey(key string) (string, error) {
	return settingRepo.GetValueByKey(key)
}
