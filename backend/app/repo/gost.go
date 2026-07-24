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
	if err := db.Offset((page - 1) * pageSize).Limit(pageSize).Order("created_at DESC").Find(&items).Error; err != nil {
		return total, nil, err
	}
	for i := range items {
		if err := revealGostService(&items[i]); err != nil {
			return total, nil, err
		}
	}
	return total, items, nil
}

func (r *GostServiceRepo) GetList(opts ...DBOption) ([]model.GostService, error) {
	var items []model.GostService
	db := getDB().Model(&model.GostService{})
	for _, opt := range opts {
		db = opt(db)
	}
	if err := db.Find(&items).Error; err != nil {
		return nil, err
	}
	for i := range items {
		if err := revealGostService(&items[i]); err != nil {
			return nil, err
		}
	}
	return items, nil
}

func (r *GostServiceRepo) Get(opts ...DBOption) (model.GostService, error) {
	var item model.GostService
	db := getDB().Model(&model.GostService{})
	for _, opt := range opts {
		db = opt(db)
	}
	if err := db.First(&item).Error; err != nil {
		return item, err
	}
	return item, revealGostService(&item)
}

func (r *GostServiceRepo) Create(svc *model.GostService) error {
	stored := *svc
	if err := protectGostService(&stored); err != nil {
		return err
	}
	if err := getDB().Create(&stored).Error; err != nil {
		return err
	}
	*svc = stored
	return revealGostService(svc)
}

func (r *GostServiceRepo) Update(id uint, updates map[string]interface{}) error {
	protected, err := protectUpdates("gost_services", updates)
	if err != nil {
		return err
	}
	return getDB().Model(&model.GostService{}).Where("id = ?", id).Updates(protected).Error
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
	if err := db.Offset((page - 1) * pageSize).Limit(pageSize).Order("created_at DESC").Find(&items).Error; err != nil {
		return total, nil, err
	}
	for i := range items {
		if err := revealGostChain(&items[i]); err != nil {
			return total, nil, err
		}
	}
	return total, items, nil
}

func (r *GostChainRepo) GetList(opts ...DBOption) ([]model.GostChain, error) {
	var items []model.GostChain
	db := getDB().Model(&model.GostChain{})
	for _, opt := range opts {
		db = opt(db)
	}
	if err := db.Find(&items).Error; err != nil {
		return nil, err
	}
	for i := range items {
		if err := revealGostChain(&items[i]); err != nil {
			return nil, err
		}
	}
	return items, nil
}

func (r *GostChainRepo) Get(opts ...DBOption) (model.GostChain, error) {
	var item model.GostChain
	db := getDB().Model(&model.GostChain{})
	for _, opt := range opts {
		db = opt(db)
	}
	if err := db.First(&item).Error; err != nil {
		return item, err
	}
	return item, revealGostChain(&item)
}

func (r *GostChainRepo) Create(chain *model.GostChain) error {
	stored := *chain
	if err := protectGostChain(&stored); err != nil {
		return err
	}
	if err := getDB().Create(&stored).Error; err != nil {
		return err
	}
	*chain = stored
	return revealGostChain(chain)
}

func (r *GostChainRepo) Update(id uint, updates map[string]interface{}) error {
	protected, err := protectUpdates("gost_chains", updates)
	if err != nil {
		return err
	}
	return getDB().Model(&model.GostChain{}).Where("id = ?", id).Updates(protected).Error
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

func protectGostService(item *model.GostService) error {
	return protectFields(secureField{Scope: "gost_services.auth_pass", Value: &item.AuthPass})
}

func revealGostService(item *model.GostService) error {
	return revealFields(secureField{Scope: "gost_services.auth_pass", Value: &item.AuthPass})
}

func protectGostChain(item *model.GostChain) error {
	return protectFields(secureField{Scope: "gost_chains.hops", Value: &item.Hops})
}

func revealGostChain(item *model.GostChain) error {
	return revealFields(secureField{Scope: "gost_chains.hops", Value: &item.Hops})
}
