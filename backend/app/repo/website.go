package repo

import (
	"xpanel/app/model"

	"gorm.io/gorm"
)

type IWebsiteRepo interface {
	Page(page, pageSize int, opts ...DBOption) (int64, []model.Website, error)
	GetList(opts ...DBOption) ([]model.Website, error)
	Get(opts ...DBOption) (model.Website, error)
	Create(item *model.Website) error
	Save(item *model.Website) error
	Delete(opts ...DBOption) error
}

func NewIWebsiteRepo() IWebsiteRepo { return &WebsiteRepo{} }

type WebsiteRepo struct{}

func (r *WebsiteRepo) Page(page, pageSize int, opts ...DBOption) (int64, []model.Website, error) {
	var (
		items []model.Website
		total int64
	)
	db := getDB().Model(&model.Website{})
	for _, opt := range opts {
		db = opt(db)
	}
	db.Count(&total)
	err := db.Offset((page - 1) * pageSize).Limit(pageSize).Order("created_at DESC").Find(&items).Error
	return total, items, err
}

func (r *WebsiteRepo) GetList(opts ...DBOption) ([]model.Website, error) {
	var items []model.Website
	db := getDB().Model(&model.Website{})
	for _, opt := range opts {
		db = opt(db)
	}
	err := db.Order("created_at DESC").Find(&items).Error
	return items, err
}

func (r *WebsiteRepo) Get(opts ...DBOption) (model.Website, error) {
	var item model.Website
	db := getDB().Model(&model.Website{})
	for _, opt := range opts {
		db = opt(db)
	}
	err := db.First(&item).Error
	return item, err
}

func (r *WebsiteRepo) Create(item *model.Website) error {
	return getDB().Create(item).Error
}

func (r *WebsiteRepo) Save(item *model.Website) error {
	return getDB().Save(item).Error
}

func (r *WebsiteRepo) Delete(opts ...DBOption) error {
	db := getDB()
	for _, opt := range opts {
		db = opt(db)
	}
	return db.Delete(&model.Website{}).Error
}

func WithByPrimaryDomain(domain string) DBOption {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("primary_domain = ?", domain)
	}
}

func WithLikeWebsite(info string) DBOption {
	return func(db *gorm.DB) *gorm.DB {
		if info == "" {
			return db
		}
		return db.Where("primary_domain LIKE ? OR domains LIKE ? OR alias LIKE ? OR remark LIKE ?",
			"%"+info+"%", "%"+info+"%", "%"+info+"%", "%"+info+"%")
	}
}
