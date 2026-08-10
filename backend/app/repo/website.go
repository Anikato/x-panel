package repo

import (
	"xpanel/app/model"

	"gorm.io/gorm"
)

type IWebsiteRepo interface {
	Page(page, pageSize int, opts ...DBOption) (int64, []model.Website, error)
	GetList(opts ...DBOption) ([]model.Website, error)
	Get(opts ...DBOption) (model.Website, error)
	Count(opts ...DBOption) (int64, error)
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
	if err := db.Offset((page - 1) * pageSize).Limit(pageSize).Order("created_at DESC").Find(&items).Error; err != nil {
		return total, nil, err
	}
	for i := range items {
		if err := revealWebsite(&items[i]); err != nil {
			return total, nil, err
		}
	}
	return total, items, nil
}

func (r *WebsiteRepo) GetList(opts ...DBOption) ([]model.Website, error) {
	var items []model.Website
	db := getDB().Model(&model.Website{})
	for _, opt := range opts {
		db = opt(db)
	}
	if err := db.Order("created_at DESC").Find(&items).Error; err != nil {
		return nil, err
	}
	for i := range items {
		if err := revealWebsite(&items[i]); err != nil {
			return nil, err
		}
	}
	return items, nil
}

func (r *WebsiteRepo) Get(opts ...DBOption) (model.Website, error) {
	var item model.Website
	db := getDB().Model(&model.Website{})
	for _, opt := range opts {
		db = opt(db)
	}
	if err := db.First(&item).Error; err != nil {
		return item, err
	}
	return item, revealWebsite(&item)
}

func (r *WebsiteRepo) Create(item *model.Website) error {
	stored := *item
	if err := protectWebsite(&stored); err != nil {
		return err
	}
	if err := getDB().Create(&stored).Error; err != nil {
		return err
	}
	*item = stored
	return revealWebsite(item)
}

func (r *WebsiteRepo) Save(item *model.Website) error {
	stored := *item
	if err := protectWebsite(&stored); err != nil {
		return err
	}
	if err := getDB().Save(&stored).Error; err != nil {
		return err
	}
	*item = stored
	return revealWebsite(item)
}

func (r *WebsiteRepo) Count(opts ...DBOption) (int64, error) {
	var count int64
	db := getDB().Model(&model.Website{})
	for _, opt := range opts {
		db = opt(db)
	}
	err := db.Count(&count).Error
	return count, err
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

func WithByAlias(alias string) DBOption {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("alias = ?", alias)
	}
}

func WithByNginxConfPath(path string) DBOption {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("nginx_conf_path = ?", path)
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

func protectWebsite(item *model.Website) error {
	return protectFields(secureField{Scope: "websites.basic_password", Value: &item.BasicPassword})
}

func revealWebsite(item *model.Website) error {
	return revealFields(secureField{Scope: "websites.basic_password", Value: &item.BasicPassword})
}
