package repo

import (
	"xpanel/app/model"
	"xpanel/global"

	"gorm.io/gorm"
)

type IBackupRepo interface {
	CreateAccount(a *model.BackupAccount) error
	UpdateAccount(id uint, fields map[string]interface{}) error
	DeleteAccount(id uint) error
	GetAccount(id uint) (*model.BackupAccount, error)
	ListAccounts() ([]model.BackupAccount, error)

	CreateRecord(r *model.BackupRecord) error
	PageRecord(page, pageSize int, opts ...DBOption) (int64, []model.BackupRecord, error)
	DeleteRecord(id uint) error
	GetRecord(id uint) (*model.BackupRecord, error)
}

func NewIBackupRepo() IBackupRepo {
	return &BackupRepo{}
}

type BackupRepo struct{}

func (r *BackupRepo) CreateAccount(a *model.BackupAccount) error {
	return global.DB.Create(a).Error
}

func (r *BackupRepo) UpdateAccount(id uint, fields map[string]interface{}) error {
	return global.DB.Model(&model.BackupAccount{}).Where("id = ?", id).Updates(fields).Error
}

func (r *BackupRepo) DeleteAccount(id uint) error {
	return global.DB.Delete(&model.BackupAccount{}, id).Error
}

func (r *BackupRepo) GetAccount(id uint) (*model.BackupAccount, error) {
	var a model.BackupAccount
	if err := global.DB.First(&a, id).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *BackupRepo) ListAccounts() ([]model.BackupAccount, error) {
	var items []model.BackupAccount
	return items, global.DB.Order("created_at desc").Find(&items).Error
}

func (r *BackupRepo) CreateRecord(rec *model.BackupRecord) error {
	return global.DB.Create(rec).Error
}

func (r *BackupRepo) PageRecord(page, pageSize int, opts ...DBOption) (int64, []model.BackupRecord, error) {
	var total int64
	var items []model.BackupRecord
	db := global.DB.Model(&model.BackupRecord{})
	for _, opt := range opts {
		db = opt(db)
	}
	if err := db.Count(&total).Error; err != nil {
		return 0, nil, err
	}
	if err := db.Offset((page-1)*pageSize).Limit(pageSize).Order("created_at desc").Find(&items).Error; err != nil {
		return 0, nil, err
	}
	return total, items, nil
}

func (r *BackupRepo) DeleteRecord(id uint) error {
	return global.DB.Delete(&model.BackupRecord{}, id).Error
}

func (r *BackupRepo) GetRecord(id uint) (*model.BackupRecord, error) {
	var rec model.BackupRecord
	if err := global.DB.First(&rec, id).Error; err != nil {
		return nil, err
	}
	return &rec, nil
}

func WithBackupType(t string) DBOption {
	return func(db *gorm.DB) *gorm.DB {
		if t != "" {
			return db.Where("type = ?", t)
		}
		return db
	}
}

func WithAccountID(id uint) DBOption {
	return func(db *gorm.DB) *gorm.DB {
		if id > 0 {
			return db.Where("account_id = ?", id)
		}
		return db
	}
}
