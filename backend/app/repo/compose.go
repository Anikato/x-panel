package repo

import (
	"xpanel/app/model"
	"xpanel/global"
)

type IComposeRepo interface {
	Create(item *model.ComposeProject) error
	Get(id uint) (*model.ComposeProject, error)
	GetByName(name string) (*model.ComposeProject, error)
	List() ([]model.ComposeProject, error)
	Delete(id uint) error
}

func NewIComposeRepo() IComposeRepo {
	return &ComposeRepo{}
}

type ComposeRepo struct{}

func (r *ComposeRepo) Create(item *model.ComposeProject) error {
	return global.DB.Create(item).Error
}

func (r *ComposeRepo) Get(id uint) (*model.ComposeProject, error) {
	var item model.ComposeProject
	if err := global.DB.First(&item, id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *ComposeRepo) GetByName(name string) (*model.ComposeProject, error) {
	var item model.ComposeProject
	if err := global.DB.Where("name = ?", name).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *ComposeRepo) List() ([]model.ComposeProject, error) {
	var items []model.ComposeProject
	if err := global.DB.Order("id desc").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (r *ComposeRepo) Delete(id uint) error {
	return global.DB.Delete(&model.ComposeProject{}, id).Error
}
