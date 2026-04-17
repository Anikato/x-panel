package repo

import (
	"context"

	"xpanel/app/model"
	"gorm.io/gorm"
)

type IAppBackupRepo interface {
	// 查询选项
	WithAppInstallID(installID uint) DBOption

	// CRUD 操作
	Page(page, size int, opts ...DBOption) (int64, []model.AppBackupRecord, error)
	GetFirst(opts ...DBOption) (model.AppBackupRecord, error)
	GetBy(opts ...DBOption) ([]model.AppBackupRecord, error)
	Create(ctx context.Context, record *model.AppBackupRecord) error
	Save(ctx context.Context, record *model.AppBackupRecord) error
	Delete(ctx context.Context, record *model.AppBackupRecord) error
	DeleteBy(ctx context.Context, opts ...DBOption) error
}

type AppBackupRepo struct{}

func NewIAppBackupRepo() IAppBackupRepo {
	return &AppBackupRepo{}
}

func (a *AppBackupRepo) WithAppInstallID(installID uint) DBOption {
	return func(g *gorm.DB) *gorm.DB {
		return g.Where("app_install_id = ?", installID)
	}
}

func (a *AppBackupRepo) Page(page, size int, opts ...DBOption) (int64, []model.AppBackupRecord, error) {
	var records []model.AppBackupRecord
	db := getDb(opts...).Model(&model.AppBackupRecord{})
	count := int64(0)
	db = db.Count(&count)
	err := db.Limit(size).Offset(size * (page - 1)).Preload("AppInstall").Order("created_at desc").Find(&records).Error
	return count, records, err
}

func (a *AppBackupRepo) GetFirst(opts ...DBOption) (model.AppBackupRecord, error) {
	var record model.AppBackupRecord
	db := getDb(opts...).Model(&model.AppBackupRecord{})
	err := db.Preload("AppInstall").First(&record).Error
	return record, err
}

func (a *AppBackupRepo) GetBy(opts ...DBOption) ([]model.AppBackupRecord, error) {
	var records []model.AppBackupRecord
	db := getDb(opts...).Model(&model.AppBackupRecord{})
	err := db.Preload("AppInstall").Order("created_at desc").Find(&records).Error
	return records, err
}

func (a *AppBackupRepo) Create(ctx context.Context, record *model.AppBackupRecord) error {
	return getTx(ctx).Create(record).Error
}

func (a *AppBackupRepo) Save(ctx context.Context, record *model.AppBackupRecord) error {
	return getTx(ctx).Save(record).Error
}

func (a *AppBackupRepo) Delete(ctx context.Context, record *model.AppBackupRecord) error {
	return getTx(ctx).Delete(record).Error
}

func (a *AppBackupRepo) DeleteBy(ctx context.Context, opts ...DBOption) error {
	return getTx(ctx, opts...).Delete(&model.AppBackupRecord{}).Error
}
