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
	CreateOrUpdate(key, value string) error
	Delete(opts ...DBOption) error
}

// NewISettingRepo 创建 Setting 仓库实例
func NewISettingRepo() ISettingRepo {
	return &SettingRepo{}
}

type SettingRepo struct{}

func (s *SettingRepo) GetList(opts ...DBOption) ([]model.Setting, error) {
	var settings []model.Setting
	db := getDb(opts...).Model(&model.Setting{})
	for _, opt := range opts {
		db = opt(db)
	}
	err := db.Find(&settings).Error
	return settings, err
}

func (s *SettingRepo) Get(opts ...DBOption) (model.Setting, error) {
	var setting model.Setting
	db := getDb(opts...).Model(&model.Setting{})
	for _, opt := range opts {
		db = opt(db)
	}
	err := db.First(&setting).Error
	return setting, err
}

func (s *SettingRepo) GetValueByKey(key string) (string, error) {
	var setting model.Setting
	err := getDb().Model(&model.Setting{}).Where("`key` = ?", key).First(&setting).Error
	if err != nil {
		return "", err
	}
	return setting.Value, nil
}

func (s *SettingRepo) Create(setting *model.Setting) error {
	return getDb().Create(setting).Error
}

func (s *SettingRepo) Update(key, value string) error {
	return getDb().Model(&model.Setting{}).Where("`key` = ?", key).Update("value", value).Error
}

func (s *SettingRepo) CreateOrUpdate(key, value string) error {
	var count int64
	getDb().Model(&model.Setting{}).Where("`key` = ?", key).Count(&count)
	if count == 0 {
		return getDb().Create(&model.Setting{Key: key, Value: value}).Error
	}
	return getDb().Model(&model.Setting{}).Where("`key` = ?", key).Update("value", value).Error
}

func (s *SettingRepo) Delete(opts ...DBOption) error {
	db := getDb(opts...)
	for _, opt := range opts {
		db = opt(db)
	}
	return db.Delete(&model.Setting{}).Error
}
