package repo

import (
	"xpanel/app/model"
	"xpanel/security/credentials"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ISettingRepo Setting 仓库接口
type ISettingRepo interface {
	GetList(opts ...DBOption) ([]model.Setting, error)
	Get(opts ...DBOption) (model.Setting, error)
	GetValueByKey(key string) (string, error)
	Create(setting *model.Setting) error
	Update(key, value string) error
	CreateOrUpdate(key, value string) error
	CreateOrUpdateMany(values map[string]string) error
	CreateIfMissingOrEmpty(key, value string) (bool, error)
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
	if err := db.Find(&settings).Error; err != nil {
		return nil, err
	}
	for i := range settings {
		if err := revealSetting(&settings[i]); err != nil {
			return nil, err
		}
	}
	return settings, nil
}

func (s *SettingRepo) Get(opts ...DBOption) (model.Setting, error) {
	var setting model.Setting
	db := getDB().Model(&model.Setting{})
	for _, opt := range opts {
		db = opt(db)
	}
	if err := db.First(&setting).Error; err != nil {
		return setting, err
	}
	return setting, revealSetting(&setting)
}

func (s *SettingRepo) GetValueByKey(key string) (string, error) {
	var setting model.Setting
	err := getDB().Model(&model.Setting{}).Where("`key` = ?", key).First(&setting).Error
	if err != nil {
		return "", err
	}
	if err := revealSetting(&setting); err != nil {
		return "", err
	}
	return setting.Value, nil
}

func (s *SettingRepo) Create(setting *model.Setting) error {
	stored := *setting
	if err := protectSetting(&stored); err != nil {
		return err
	}
	if err := getDB().Create(&stored).Error; err != nil {
		return err
	}
	*setting = stored
	return revealSetting(setting)
}

func (s *SettingRepo) Update(key, value string) error {
	protected, err := protectSettingValue(key, value)
	if err != nil {
		return err
	}
	return getDB().Model(&model.Setting{}).Where("`key` = ?", key).Update("value", protected).Error
}

func (s *SettingRepo) CreateOrUpdate(key, value string) error {
	protected, err := protectSettingValue(key, value)
	if err != nil {
		return err
	}
	var count int64
	getDB().Model(&model.Setting{}).Where("`key` = ?", key).Count(&count)
	if count == 0 {
		return getDB().Create(&model.Setting{Key: key, Value: protected}).Error
	}
	return getDB().Model(&model.Setting{}).Where("`key` = ?", key).Update("value", protected).Error
}

func (s *SettingRepo) CreateOrUpdateMany(values map[string]string) error {
	protected := make(map[string]string, len(values))
	for key, value := range values {
		stored, err := protectSettingValue(key, value)
		if err != nil {
			return err
		}
		protected[key] = stored
	}
	return getDB().Transaction(func(tx *gorm.DB) error {
		for key, value := range protected {
			var count int64
			if err := tx.Model(&model.Setting{}).Where("`key` = ?", key).Count(&count).Error; err != nil {
				return err
			}
			if count == 0 {
				if err := tx.Create(&model.Setting{Key: key, Value: value}).Error; err != nil {
					return err
				}
				continue
			}
			if err := tx.Model(&model.Setting{}).Where("`key` = ?", key).Update("value", value).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *SettingRepo) CreateIfMissingOrEmpty(key, value string) (bool, error) {
	protected, err := protectSettingValue(key, value)
	if err != nil {
		return false, err
	}
	written := false
	err = getDB().Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&model.Setting{}).
			Where("`key` = ? AND `value` = ''", key).
			Update("value", protected)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 1 {
			written = true
			return nil
		}
		result = tx.Clauses(clause.OnConflict{DoNothing: true}).
			Create(&model.Setting{Key: key, Value: protected})
		if result.Error != nil {
			return result.Error
		}
		written = result.RowsAffected == 1
		return nil
	})
	return written, err
}

func (s *SettingRepo) Delete(opts ...DBOption) error {
	db := getDB()
	for _, opt := range opts {
		db = opt(db)
	}
	return db.Delete(&model.Setting{}).Error
}

func protectSetting(setting *model.Setting) error {
	protected, err := protectSettingValue(setting.Key, setting.Value)
	if err != nil {
		return err
	}
	setting.Value = protected
	return nil
}

func protectSettingValue(key, value string) (string, error) {
	if !credentials.IsSecretSetting(key) || value == "" {
		return value, nil
	}
	field := secureField{Scope: credentials.SettingScope(key), Value: &value}
	if err := protectFields(field); err != nil {
		return "", err
	}
	return value, nil
}

func revealSetting(setting *model.Setting) error {
	if !credentials.IsSecretSetting(setting.Key) || setting.Value == "" {
		return nil
	}
	return revealFields(secureField{
		Scope: credentials.SettingScope(setting.Key),
		Value: &setting.Value,
	})
}
