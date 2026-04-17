package repo

import (
	"context"

	"xpanel/app/model"
	"xpanel/global"
)

type IAppImportTaskRepo interface {
	Create(ctx context.Context, task *model.AppImportTask) error
	Update(ctx context.Context, task *model.AppImportTask) error
	GetByName(name string) (model.AppImportTask, error)
	GetList(opts ...DBOption) ([]model.AppImportTask, error)
	Delete(ctx context.Context, task *model.AppImportTask) error
}

type AppImportTaskRepo struct{}

func NewIAppImportTaskRepo() IAppImportTaskRepo {
	return &AppImportTaskRepo{}
}

func (r *AppImportTaskRepo) Create(ctx context.Context, task *model.AppImportTask) error {
	return global.DB.Create(task).Error
}

func (r *AppImportTaskRepo) Update(ctx context.Context, task *model.AppImportTask) error {
	return global.DB.Save(task).Error
}

func (r *AppImportTaskRepo) GetByName(name string) (model.AppImportTask, error) {
	var task model.AppImportTask
	err := global.DB.Where("name = ?", name).First(&task).Error
	return task, err
}

func (r *AppImportTaskRepo) GetList(opts ...DBOption) ([]model.AppImportTask, error) {
	var tasks []model.AppImportTask
	db := global.DB
	for _, opt := range opts {
		db = opt(db)
	}
	err := db.Find(&tasks).Error
	return tasks, err
}

func (r *AppImportTaskRepo) Delete(ctx context.Context, task *model.AppImportTask) error {
	return global.DB.Delete(task).Error
}