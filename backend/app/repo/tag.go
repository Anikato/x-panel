package repo

import (
	"context"

	"xpanel/app/model"
)

type ITagRepo interface {
	// CRUD 操作
	GetAll(opts ...DBOption) ([]model.Tag, error)
	GetFirst(opts ...DBOption) (model.Tag, error)
	Create(ctx context.Context, tag *model.Tag) error
	Save(ctx context.Context, tag *model.Tag) error
	BatchCreate(ctx context.Context, tags []model.Tag) error
	DeleteBy(ctx context.Context, opts ...DBOption) error
}

type TagRepo struct{}

func NewITagRepo() ITagRepo {
	return &TagRepo{}
}

func (t *TagRepo) GetAll(opts ...DBOption) ([]model.Tag, error) {
	var tags []model.Tag
	db := getDb(opts...).Model(&model.Tag{})
	if err := db.Order("sort asc").Find(&tags).Error; err != nil {
		return tags, err
	}
	return tags, nil
}

func (t *TagRepo) GetFirst(opts ...DBOption) (model.Tag, error) {
	var tag model.Tag
	db := getDb(opts...).Model(&model.Tag{})
	if err := db.First(&tag).Error; err != nil {
		return tag, err
	}
	return tag, nil
}

func (t *TagRepo) Create(ctx context.Context, tag *model.Tag) error {
	return getTx(ctx).Create(tag).Error
}

func (t *TagRepo) Save(ctx context.Context, tag *model.Tag) error {
	return getTx(ctx).Save(tag).Error
}

func (t *TagRepo) BatchCreate(ctx context.Context, tags []model.Tag) error {
	return getTx(ctx).Create(&tags).Error
}

func (t *TagRepo) DeleteBy(ctx context.Context, opts ...DBOption) error {
	return getTx(ctx, opts...).Delete(&model.Tag{}).Error
}
