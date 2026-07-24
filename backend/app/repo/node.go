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
	stored := *n
	if err := protectNode(&stored); err != nil {
		return err
	}
	if err := global.DB.Create(&stored).Error; err != nil {
		return err
	}
	*n = stored
	return revealNode(n)
}

func (r *NodeRepo) Update(id uint, fields map[string]interface{}) error {
	protected, err := protectUpdates("nodes", fields)
	if err != nil {
		return err
	}
	return global.DB.Model(&model.Node{}).Where("id = ?", id).Updates(protected).Error
}

func (r *NodeRepo) Delete(id uint) error {
	return global.DB.Delete(&model.Node{}, id).Error
}

func (r *NodeRepo) Get(id uint) (*model.Node, error) {
	var n model.Node
	if err := global.DB.First(&n, id).Error; err != nil {
		return nil, err
	}
	if err := revealNode(&n); err != nil {
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
	if err := db.Order("created_at desc").Find(&items).Error; err != nil {
		return nil, err
	}
	if err := revealNodes(items); err != nil {
		return nil, err
	}
	return items, nil
}

func protectNode(item *model.Node) error {
	return protectFields(
		secureField{Scope: "nodes.token", Value: &item.Token},
		secureField{Scope: "nodes.ssh_password", Value: &item.SSHPassword},
	)
}

func revealNode(item *model.Node) error {
	return revealFields(
		secureField{Scope: "nodes.token", Value: &item.Token},
		secureField{Scope: "nodes.ssh_password", Value: &item.SSHPassword},
	)
}

func revealNodes(items []model.Node) error {
	for i := range items {
		if err := revealNode(&items[i]); err != nil {
			return err
		}
	}
	return nil
}
