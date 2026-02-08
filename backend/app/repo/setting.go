package repo

import (
	"xpanel/app/model"
)

// ISettingRepo Setting 仓库接口
type ISettingRepo interface {
	GetList(opts ...DBOption) ([]model.Setting, error)
	Get(opts ...DBOption) (model.Setting, error)
	GetValueByKey(key string) (string, error)
	Create(setting *model.Setting) error
	Update(key, value string) error
	Delete(opts ...DBOption) error
}

// NewISettingRepo 创建 Setting 仓库实例
func NewISettingRepo() ISettingRepo {
	return &SettingRepo{}
}

type SettingRepo struct{}

func (s *SettingRepo) GetList(opts ...DBOption) ([]model.Setting, error) {
	var settings []model.Setting
	db := getDB().Model(&model.Setting{})
	for _, opt := range opts {
		db = opt(db)
	}
	err := db.Find(&settings).Error
	return settings, err
}

func (s *SettingRepo) Get(opts ...DBOption) (model.Setting, error) {
	var setting model.Setting
	db := getDB().Model(&model.Setting{})
	for _, opt := range opts {
		db = opt(db)
	}
	err := db.First(&setting).Error
	return setting, err
}

func (s *SettingRepo) GetValueByKey(key string) (string, error) {
	var setting model.Setting
	err := getDB().Model(&model.Setting{}).Where("`key` = ?", key).First(&setting).Error
	if err != nil {
		return "", err
	}
	return setting.Value, nil
}

func (s *SettingRepo) Create(setting *model.Setting) error {
	return getDB().Create(setting).Error
}

func (s *SettingRepo) Update(key, value string) error {
	return getDB().Model(&model.Setting{}).Where("`key` = ?", key).Update("value", value).Error
}

func (s *SettingRepo) Delete(opts ...DBOption) error {
	db := getDB()
	for _, opt := range opts {
		db = opt(db)
	}
	return db.Delete(&model.Setting{}).Error
}
