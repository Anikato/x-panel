package repo

import (
	"xpanel/app/model"
	"xpanel/global"
)

type INodeRepo interface {
	Create(n *model.Node) error
	Update(id uint, fields map[string]interface{}) error
	Delete(id uint) error
	Get(id uint) (*model.Node, error)
	List(opts ...DBOption) ([]model.Node, error)
}

func NewINodeRepo() INodeRepo {
	return &NodeRepo{}
}

type NodeRepo struct{}

func (r *NodeRepo) Create(n *model.Node) error {
	return global.DB.Create(n).Error
}

func (r *NodeRepo) Update(id uint, fields map[string]interface{}) error {
	return global.DB.Model(&model.Node{}).Where("id = ?", id).Updates(fields).Error
}

func (r *NodeRepo) Delete(id uint) error {
	return global.DB.Delete(&model.Node{}, id).Error
}

func (r *NodeRepo) Get(id uint) (*model.Node, error) {
	var n model.Node
	if err := global.DB.First(&n, id).Error; err != nil {
		return nil, err
	}
	return &n, nil
}

func (r *NodeRepo) List(opts ...DBOption) ([]model.Node, error) {
	var items []model.Node
	db := global.DB.Model(&model.Node{})
	for _, opt := range opts {
		db = opt(db)
	}
	return items, db.Order("created_at desc").Find(&items).Error
}
