package repo

import (
	"context"

	"xpanel/app/model"
	"gorm.io/gorm"
)

type IAppTagRepo interface {
	// 查询选项
	WithAppID(appID uint) DBOption
	WithTagKey(tagKey string) DBOption
	WithAppIDIn(appIDs []uint) DBOption

	// CRUD 操作
	GetBy(opts ...DBOption) ([]model.AppTag, error)
	Create(ctx context.Context, tag *model.AppTag) error
	BatchCreate(ctx context.Context, tags []model.AppTag) error
	DeleteBy(ctx context.Context, opts ...DBOption) error
}

type AppTagRepo struct{}

func NewIAppTagRepo() IAppTagRepo {
	return &AppTagRepo{}
}

func (a *AppTagRepo) WithAppID(appID uint) DBOption {
	return func(g *gorm.DB) *gorm.DB {
		return g.Where("app_id = ?", appID)
	}
}

func (a *AppTagRepo) WithTagKey(tagKey string) DBOption {
	return func(g *gorm.DB) *gorm.DB {
		return g.Where("tag_key = ?", tagKey)
	}
}

func (a *AppTagRepo) WithAppIDIn(appIDs []uint) DBOption {
	return func(g *gorm.DB) *gorm.DB {
		return g.Where("app_id IN (?)", appIDs)
	}
}

func (a *AppTagRepo) GetBy(opts ...DBOption) ([]model.AppTag, error) {
	var tags []model.AppTag
	db := getDb(opts...).Model(&model.AppTag{})
	if err := db.Find(&tags).Error; err != nil {
		return tags, err
	}
	return tags, nil
}

func (a *AppTagRepo) Create(ctx context.Context, tag *model.AppTag) error {
	return getTx(ctx).Create(tag).Error
}

func (a *AppTagRepo) BatchCreate(ctx context.Context, tags []model.AppTag) error {
	return getTx(ctx).Create(&tags).Error
}

func (a *AppTagRepo) DeleteBy(ctx context.Context, opts ...DBOption) error {
	return getTx(ctx, opts...).Delete(&model.AppTag{}).Error
}
