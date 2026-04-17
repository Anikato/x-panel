package repo

import (
	"context"

	"xpanel/app/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type IAppRepo interface {
	// 查询选项
	WithKey(key string) DBOption
	WithType(typeStr string) DBOption
	WithKeyIn(keys []string) DBOption
	WithArch(arch string) DBOption
	OrderByRecommend() DBOption

	// CRUD 操作
	Page(page, size int, opts ...DBOption) (int64, []model.App, error)
	GetFirst(opts ...DBOption) (model.App, error)
	GetBy(opts ...DBOption) ([]model.App, error)
	Create(ctx context.Context, app *model.App) error
	Save(ctx context.Context, app *model.App) error
	BatchCreate(ctx context.Context, apps []model.App) error
	DeleteBy(ctx context.Context, opts ...DBOption) error
}

type AppRepo struct{}

func NewIAppRepo() IAppRepo {
	return &AppRepo{}
}

func (a *AppRepo) WithKey(key string) DBOption {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("`key` = ?", key)
	}
}

func (a *AppRepo) WithType(typeStr string) DBOption {
	return func(g *gorm.DB) *gorm.DB {
		return g.Where("`type` = ?", typeStr)
	}
}

func (a *AppRepo) WithKeyIn(keys []string) DBOption {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("`key` in (?)", keys)
	}
}

func (a *AppRepo) WithArch(arch string) DBOption {
	return func(g *gorm.DB) *gorm.DB {
		return g.Where("architectures like ?", "%"+arch+"%")
	}
}

func (a *AppRepo) OrderByRecommend() DBOption {
	return func(g *gorm.DB) *gorm.DB {
		return g.Order("recommend desc")
	}
}

func (a *AppRepo) Page(page, size int, opts ...DBOption) (int64, []model.App, error) {
	var apps []model.App
	db := getDb(opts...).Model(&model.App{})
	count := int64(0)
	db = db.Count(&count)
	err := db.Limit(size).Offset(size * (page - 1)).Find(&apps).Error
	return count, apps, err
}

func (a *AppRepo) GetFirst(opts ...DBOption) (model.App, error) {
	var app model.App
	db := getDb(opts...).Model(&model.App{})
	if err := db.First(&app).Error; err != nil {
		return app, err
	}
	return app, nil
}

func (a *AppRepo) GetBy(opts ...DBOption) ([]model.App, error) {
	var apps []model.App
	db := getDb(opts...).Model(&model.App{})
	if err := db.Find(&apps).Error; err != nil {
		return apps, err
	}
	return apps, nil
}

func (a *AppRepo) Create(ctx context.Context, app *model.App) error {
	return getTx(ctx).Omit(clause.Associations).Create(app).Error
}

func (a *AppRepo) Save(ctx context.Context, app *model.App) error {
	return getTx(ctx).Omit(clause.Associations).Save(app).Error
}

func (a *AppRepo) BatchCreate(ctx context.Context, apps []model.App) error {
	return getTx(ctx).Omit(clause.Associations).Create(&apps).Error
}

func (a *AppRepo) DeleteBy(ctx context.Context, opts ...DBOption) error {
	return getTx(ctx, opts...).Delete(&model.App{}).Error
}
