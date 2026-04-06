package repo

import (
	"strings"

	"xpanel/app/model"

	"gorm.io/gorm"
)

// --- GostService Repo ---

type IGostServiceRepo interface {
	Page(page, pageSize int, opts ...DBOption) (int64, []model.GostService, error)
	GetList(opts ...DBOption) ([]model.GostService, error)
	Get(opts ...DBOption) (model.GostService, error)
	Create(svc *model.GostService) error
	Update(id uint, updates map[string]interface{}) error
	Delete(opts ...DBOption) error
	CountByChainID(chainID uint) (int64, error)
}

func NewIGostServiceRepo() IGostServiceRepo { return &GostServiceRepo{} }

type GostServiceRepo struct{}

func (r *GostServiceRepo) Page(page, pageSize int, opts ...DBOption) (int64, []model.GostService, error) {
	var (
		items []model.GostService
		total int64
	)
	db := getDB().Model(&model.GostService{})
	for _, opt := range opts {
		db = opt(db)
	}
	db.Count(&total)
	err := db.Offset((page-1) * pageSize).Limit(pageSize).Order("created_at DESC").Find(&items).Error
	return total, items, err
}

func (r *GostServiceRepo) GetList(opts ...DBOption) ([]model.GostService, error) {
	var items []model.GostService
	db := getDB().Model(&model.GostService{})
	for _, opt := range opts {
		db = opt(db)
	}
	err := db.Find(&items).Error
	return items, err
}

func (r *GostServiceRepo) Get(opts ...DBOption) (model.GostService, error) {
	var item model.GostService
	db := getDB().Model(&model.GostService{})
	for _, opt := range opts {
		db = opt(db)
	}
	err := db.First(&item).Error
	return item, err
}

func (r *GostServiceRepo) Create(svc *model.GostService) error {
	return getDB().Create(svc).Error
}

func (r *GostServiceRepo) Update(id uint, updates map[string]interface{}) error {
	return getDB().Model(&model.GostService{}).Where("id = ?", id).Updates(updates).Error
}

func (r *GostServiceRepo) Delete(opts ...DBOption) error {
	db := getDB()
	for _, opt := range opts {
		db = opt(db)
	}
	return db.Delete(&model.GostService{}).Error
}

func (r *GostServiceRepo) CountByChainID(chainID uint) (int64, error) {
	var count int64
	err := getDB().Model(&model.GostService{}).Where("chain_id = ?", chainID).Count(&count).Error
	return count, err
}

// --- GostChain Repo ---

type IGostChainRepo interface {
	Page(page, pageSize int, opts ...DBOption) (int64, []model.GostChain, error)
	GetList(opts ...DBOption) ([]model.GostChain, error)
	Get(opts ...DBOption) (model.GostChain, error)
	Create(chain *model.GostChain) error
	Update(id uint, updates map[string]interface{}) error
	Delete(opts ...DBOption) error
}

func NewIGostChainRepo() IGostChainRepo { return &GostChainRepo{} }

type GostChainRepo struct{}

func (r *GostChainRepo) Page(page, pageSize int, opts ...DBOption) (int64, []model.GostChain, error) {
	var (
		items []model.GostChain
		total int64
	)
	db := getDB().Model(&model.GostChain{})
	for _, opt := range opts {
		db = opt(db)
	}
	db.Count(&total)
	err := db.Offset((page-1) * pageSize).Limit(pageSize).Order("created_at DESC").Find(&items).Error
	return total, items, err
}

func (r *GostChainRepo) GetList(opts ...DBOption) ([]model.GostChain, error) {
	var items []model.GostChain
	db := getDB().Model(&model.GostChain{})
	for _, opt := range opts {
		db = opt(db)
	}
	err := db.Find(&items).Error
	return items, err
}

func (r *GostChainRepo) Get(opts ...DBOption) (model.GostChain, error) {
	var item model.GostChain
	db := getDB().Model(&model.GostChain{})
	for _, opt := range opts {
		db = opt(db)
	}
	err := db.First(&item).Error
	return item, err
}

func (r *GostChainRepo) Create(chain *model.GostChain) error {
	return getDB().Create(chain).Error
}

func (r *GostChainRepo) Update(id uint, updates map[string]interface{}) error {
	return getDB().Model(&model.GostChain{}).Where("id = ?", id).Updates(updates).Error
}

func (r *GostChainRepo) Delete(opts ...DBOption) error {
	db := getDB()
	for _, opt := range opts {
		db = opt(db)
	}
	return db.Delete(&model.GostChain{}).Error
}

// WithByGostType 按 GOST 服务类型查询，支持逗号分隔的多类型
func WithByGostType(t string) DBOption {
	return func(db *gorm.DB) *gorm.DB {
		if t == "" {
			return db
		}
		types := strings.Split(t, ",")
		if len(types) == 1 {
			return db.Where("type = ?", t)
		}
		return db.Where("type IN ?", types)
	}
}
