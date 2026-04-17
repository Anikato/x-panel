package repo

import (
	"context"

	"xpanel/app/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type IAppInstallRepo interface {
	// 查询选项
	WithAppID(appID uint) DBOption
	WithAppDetailID(detailID uint) DBOption
	WithServiceName(serviceName string) DBOption
	WithContainerName(containerName string) DBOption
	WithPort(port int) DBOption

	// CRUD 操作
	Page(page, size int, opts ...DBOption) (int64, []model.AppInstall, error)
	GetFirst(opts ...DBOption) (model.AppInstall, error)
	GetBy(opts ...DBOption) ([]model.AppInstall, error)
	Create(ctx context.Context, install *model.AppInstall) error
	Save(ctx context.Context, install *model.AppInstall) error
	Delete(ctx context.Context, install *model.AppInstall) error
	DeleteBy(ctx context.Context, opts ...DBOption) error
}

type AppInstallRepo struct{}

func NewIAppInstallRepo() IAppInstallRepo {
	return &AppInstallRepo{}
}

func (a *AppInstallRepo) WithAppID(appID uint) DBOption {
	return func(g *gorm.DB) *gorm.DB {
		return g.Where("app_id = ?", appID)
	}
}

func (a *AppInstallRepo) WithAppDetailID(detailID uint) DBOption {
	return func(g *gorm.DB) *gorm.DB {
		return g.Where("app_detail_id = ?", detailID)
	}
}

func (a *AppInstallRepo) WithServiceName(serviceName string) DBOption {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("service_name = ?", serviceName)
	}
}

func (a *AppInstallRepo) WithContainerName(containerName string) DBOption {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("container_name = ?", containerName)
	}
}

func (a *AppInstallRepo) WithPort(port int) DBOption {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("http_port = ? OR https_port = ?", port, port)
	}
}

func (a *AppInstallRepo) Page(page, size int, opts ...DBOption) (int64, []model.AppInstall, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}
	var installs []model.AppInstall
	var count int64
	if err := getDb(opts...).Model(&model.AppInstall{}).Count(&count).Error; err != nil {
		return 0, nil, err
	}
	err := getDb(opts...).Model(&model.AppInstall{}).Limit(size).Offset(size*(page-1)).Preload("App").Preload("AppDetail").Find(&installs).Error
	return count, installs, err
}

func (a *AppInstallRepo) GetFirst(opts ...DBOption) (model.AppInstall, error) {
	var install model.AppInstall
	db := getDb(opts...).Model(&model.AppInstall{})
	err := db.Preload("App").Preload("AppDetail").First(&install).Error
	return install, err
}

func (a *AppInstallRepo) GetBy(opts ...DBOption) ([]model.AppInstall, error) {
	var installs []model.AppInstall
	db := getDb(opts...).Model(&model.AppInstall{})
	err := db.Preload("App").Preload("AppDetail").Find(&installs).Error
	return installs, err
}

func (a *AppInstallRepo) Create(ctx context.Context, install *model.AppInstall) error {
	return getTx(ctx).Omit(clause.Associations).Create(install).Error
}

func (a *AppInstallRepo) Save(ctx context.Context, install *model.AppInstall) error {
	return getTx(ctx).Omit("App", "AppDetail").Save(install).Error
}

func (a *AppInstallRepo) Delete(ctx context.Context, install *model.AppInstall) error {
	return getTx(ctx).Delete(install).Error
}

func (a *AppInstallRepo) DeleteBy(ctx context.Context, opts ...DBOption) error {
	return getTx(ctx, opts...).Delete(&model.AppInstall{}).Error
}
