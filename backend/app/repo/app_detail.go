package repo

import (
	"context"

	"xpanel/app/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type IAppDetailRepo interface {
	// 查询选项
	WithAppID(appID uint) DBOption
	WithVersion(version string) DBOption
	WithAppIDAndVersion(appID uint, version string) DBOption
	WithAppIDIn(appIDs []uint) DBOption

	// CRUD 操作
	GetFirst(opts ...DBOption) (model.AppDetail, error)
	GetBy(opts ...DBOption) ([]model.AppDetail, error)
	Create(ctx context.Context, detail *model.AppDetail) error
	Save(ctx context.Context, detail *model.AppDetail) error
	BatchCreate(ctx context.Context, details []model.AppDetail) error
	DeleteBy(ctx context.Context, opts ...DBOption) error
}

type AppDetailRepo struct{}

func NewIAppDetailRepo() IAppDetailRepo {
	return &AppDetailRepo{}
}

func (a *AppDetailRepo) WithAppID(appID uint) DBOption {
	return func(g *gorm.DB) *gorm.DB {
		return g.Where("app_id = ?", appID)
	}
}

func (a *AppDetailRepo) WithVersion(version string) DBOption {
	return func(g *gorm.DB) *gorm.DB {
		return g.Where("version = ?", version)
	}
}

func (a *AppDetailRepo) WithAppIDAndVersion(appID uint, version string) DBOption {
	return func(g *gorm.DB) *gorm.DB {
		return g.Where("app_id = ? AND version = ?", appID, version)
	}
}

func (a *AppDetailRepo) WithAppIDIn(appIDs []uint) DBOption {
	return func(g *gorm.DB) *gorm.DB {
		return g.Where("app_id IN (?)", appIDs)
	}
}

func (a *AppDetailRepo) GetFirst(opts ...DBOption) (model.AppDetail, error) {
	var detail model.AppDetail
	db := getDb(opts...).Model(&model.AppDetail{})
	if err := db.First(&detail).Error; err != nil {
		return detail, err
	}
	return detail, nil
}

func (a *AppDetailRepo) GetBy(opts ...DBOption) ([]model.AppDetail, error) {
	var details []model.AppDetail
	db := getDb(opts...).Model(&model.AppDetail{})
	if err := db.Find(&details).Error; err != nil {
		return details, err
	}
	return details, nil
}

func (a *AppDetailRepo) Create(ctx context.Context, detail *model.AppDetail) error {
	return getTx(ctx).Omit(clause.Associations).Create(detail).Error
}

func (a *AppDetailRepo) Save(ctx context.Context, detail *model.AppDetail) error {
	return getTx(ctx).Omit(clause.Associations).Save(detail).Error
}

func (a *AppDetailRepo) BatchCreate(ctx context.Context, details []model.AppDetail) error {
	return getTx(ctx).Omit(clause.Associations).Create(&details).Error
}

func (a *AppDetailRepo) DeleteBy(ctx context.Context, opts ...DBOption) error {
	return getTx(ctx, opts...).Delete(&model.AppDetail{}).Error
}
