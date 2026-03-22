package repo

import (
	"xpanel/app/model"

	"gorm.io/gorm"
)

// ==================== XrayNode Repo ====================

type IXrayNodeRepo interface {
	GetList(opts ...DBOption) ([]model.XrayNode, error)
	Get(opts ...DBOption) (model.XrayNode, error)
	Create(item *model.XrayNode) error
	Save(item *model.XrayNode) error
	Delete(opts ...DBOption) error
}

func NewIXrayNodeRepo() IXrayNodeRepo { return &XrayNodeRepo{} }

type XrayNodeRepo struct{}

func (r *XrayNodeRepo) GetList(opts ...DBOption) ([]model.XrayNode, error) {
	var items []model.XrayNode
	db := getDB().Model(&model.XrayNode{})
	for _, opt := range opts {
		db = opt(db)
	}
	err := db.Order("created_at ASC").Find(&items).Error
	return items, err
}

func (r *XrayNodeRepo) Get(opts ...DBOption) (model.XrayNode, error) {
	var item model.XrayNode
	db := getDB().Model(&model.XrayNode{})
	for _, opt := range opts {
		db = opt(db)
	}
	err := db.First(&item).Error
	return item, err
}

func (r *XrayNodeRepo) Create(item *model.XrayNode) error {
	return getDB().Create(item).Error
}

func (r *XrayNodeRepo) Save(item *model.XrayNode) error {
	return getDB().Save(item).Error
}

func (r *XrayNodeRepo) Delete(opts ...DBOption) error {
	db := getDB().Model(&model.XrayNode{})
	for _, opt := range opts {
		db = opt(db)
	}
	return db.Delete(&model.XrayNode{}).Error
}

// ==================== XrayUser Repo ====================

type IXrayUserRepo interface {
	Page(page, pageSize int, opts ...DBOption) (int64, []model.XrayUser, error)
	GetList(opts ...DBOption) ([]model.XrayUser, error)
	Get(opts ...DBOption) (model.XrayUser, error)
	Count(opts ...DBOption) (int64, error)
	Create(item *model.XrayUser) error
	Save(item *model.XrayUser) error
	Updates(id uint, fields map[string]interface{}) error
	Delete(opts ...DBOption) error
}

func NewIXrayUserRepo() IXrayUserRepo { return &XrayUserRepo{} }

type XrayUserRepo struct{}

func (r *XrayUserRepo) Page(page, pageSize int, opts ...DBOption) (int64, []model.XrayUser, error) {
	var (
		items []model.XrayUser
		total int64
	)
	db := getDB().Model(&model.XrayUser{})
	for _, opt := range opts {
		db = opt(db)
	}
	db.Count(&total)
	err := db.Offset((page - 1) * pageSize).Limit(pageSize).Order("created_at DESC").Find(&items).Error
	return total, items, err
}

func (r *XrayUserRepo) GetList(opts ...DBOption) ([]model.XrayUser, error) {
	var items []model.XrayUser
	db := getDB().Model(&model.XrayUser{})
	for _, opt := range opts {
		db = opt(db)
	}
	err := db.Find(&items).Error
	return items, err
}

func (r *XrayUserRepo) Get(opts ...DBOption) (model.XrayUser, error) {
	var item model.XrayUser
	db := getDB().Model(&model.XrayUser{})
	for _, opt := range opts {
		db = opt(db)
	}
	err := db.First(&item).Error
	return item, err
}

func (r *XrayUserRepo) Count(opts ...DBOption) (int64, error) {
	var total int64
	db := getDB().Model(&model.XrayUser{})
	for _, opt := range opts {
		db = opt(db)
	}
	err := db.Count(&total).Error
	return total, err
}

func (r *XrayUserRepo) Create(item *model.XrayUser) error {
	return getDB().Create(item).Error
}

func (r *XrayUserRepo) Save(item *model.XrayUser) error {
	return getDB().Save(item).Error
}

func (r *XrayUserRepo) Updates(id uint, fields map[string]interface{}) error {
	return getDB().Model(&model.XrayUser{}).Where("id = ?", id).Updates(fields).Error
}

func (r *XrayUserRepo) Delete(opts ...DBOption) error {
	db := getDB().Model(&model.XrayUser{})
	for _, opt := range opts {
		db = opt(db)
	}
	return db.Delete(&model.XrayUser{}).Error
}

// ==================== DBOption helpers for Xray ====================

func WithXrayNodeID(id uint) DBOption {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("node_id = ?", id)
	}
}
