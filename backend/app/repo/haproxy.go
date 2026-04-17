package repo

import (
	"xpanel/app/model"
)

// --- HAProxyLB Repo ---

type IHAProxyLBRepo interface {
	Page(page, pageSize int, opts ...DBOption) (int64, []model.HAProxyLB, error)
	GetList(opts ...DBOption) ([]model.HAProxyLB, error)
	Get(opts ...DBOption) (model.HAProxyLB, error)
	Create(item *model.HAProxyLB) error
	Update(id uint, updates map[string]interface{}) error
	Delete(opts ...DBOption) error
	CountByBindPort(port int, excludeID uint) (int64, error)
}

func NewIHAProxyLBRepo() IHAProxyLBRepo { return &HAProxyLBRepo{} }

type HAProxyLBRepo struct{}

func (r *HAProxyLBRepo) Page(page, pageSize int, opts ...DBOption) (int64, []model.HAProxyLB, error) {
	var items []model.HAProxyLB
	var total int64
	db := getDb(opts...).Model(&model.HAProxyLB{})
	for _, opt := range opts {
		db = opt(db)
	}
	db.Count(&total)
	err := db.Offset((page - 1) * pageSize).Limit(pageSize).Order("created_at DESC").Find(&items).Error
	return total, items, err
}

func (r *HAProxyLBRepo) GetList(opts ...DBOption) ([]model.HAProxyLB, error) {
	var items []model.HAProxyLB
	db := getDb(opts...).Model(&model.HAProxyLB{})
	for _, opt := range opts {
		db = opt(db)
	}
	err := db.Order("name ASC").Find(&items).Error
	return items, err
}

func (r *HAProxyLBRepo) Get(opts ...DBOption) (model.HAProxyLB, error) {
	var item model.HAProxyLB
	db := getDb(opts...).Model(&model.HAProxyLB{})
	for _, opt := range opts {
		db = opt(db)
	}
	err := db.First(&item).Error
	return item, err
}

func (r *HAProxyLBRepo) Create(item *model.HAProxyLB) error {
	return getDb().Create(item).Error
}

func (r *HAProxyLBRepo) Update(id uint, updates map[string]interface{}) error {
	return getDb().Model(&model.HAProxyLB{}).Where("id = ?", id).Updates(updates).Error
}

func (r *HAProxyLBRepo) Delete(opts ...DBOption) error {
	db := getDb(opts...)
	for _, opt := range opts {
		db = opt(db)
	}
	return db.Delete(&model.HAProxyLB{}).Error
}

func (r *HAProxyLBRepo) CountByBindPort(port int, excludeID uint) (int64, error) {
	var count int64
	db := getDb().Model(&model.HAProxyLB{}).Where("bind_port = ?", port)
	if excludeID > 0 {
		db = db.Where("id <> ?", excludeID)
	}
	err := db.Count(&count).Error
	return count, err
}

// --- HAProxyBackend Repo ---

type IHAProxyBackendRepo interface {
	Page(page, pageSize int, opts ...DBOption) (int64, []model.HAProxyBackend, error)
	GetList(opts ...DBOption) ([]model.HAProxyBackend, error)
	Get(opts ...DBOption) (model.HAProxyBackend, error)
	Create(item *model.HAProxyBackend) error
	Update(id uint, updates map[string]interface{}) error
	Delete(opts ...DBOption) error
}

func NewIHAProxyBackendRepo() IHAProxyBackendRepo { return &HAProxyBackendRepo{} }

type HAProxyBackendRepo struct{}

func (r *HAProxyBackendRepo) Page(page, pageSize int, opts ...DBOption) (int64, []model.HAProxyBackend, error) {
	var items []model.HAProxyBackend
	var total int64
	db := getDb(opts...).Model(&model.HAProxyBackend{})
	for _, opt := range opts {
		db = opt(db)
	}
	db.Count(&total)
	err := db.Offset((page - 1) * pageSize).Limit(pageSize).Order("created_at DESC").Find(&items).Error
	return total, items, err
}

func (r *HAProxyBackendRepo) GetList(opts ...DBOption) ([]model.HAProxyBackend, error) {
	var items []model.HAProxyBackend
	db := getDb(opts...).Model(&model.HAProxyBackend{})
	for _, opt := range opts {
		db = opt(db)
	}
	err := db.Order("name ASC").Find(&items).Error
	return items, err
}

func (r *HAProxyBackendRepo) Get(opts ...DBOption) (model.HAProxyBackend, error) {
	var item model.HAProxyBackend
	db := getDb(opts...).Model(&model.HAProxyBackend{})
	for _, opt := range opts {
		db = opt(db)
	}
	err := db.First(&item).Error
	return item, err
}

func (r *HAProxyBackendRepo) Create(item *model.HAProxyBackend) error {
	return getDb().Create(item).Error
}

func (r *HAProxyBackendRepo) Update(id uint, updates map[string]interface{}) error {
	return getDb().Model(&model.HAProxyBackend{}).Where("id = ?", id).Updates(updates).Error
}

func (r *HAProxyBackendRepo) Delete(opts ...DBOption) error {
	db := getDb(opts...)
	for _, opt := range opts {
		db = opt(db)
	}
	return db.Delete(&model.HAProxyBackend{}).Error
}

// --- HAProxyServer Repo ---

type IHAProxyServerRepo interface {
	GetList(opts ...DBOption) ([]model.HAProxyServer, error)
	GetListByBackend(backendID uint) ([]model.HAProxyServer, error)
	Get(opts ...DBOption) (model.HAProxyServer, error)
	Create(item *model.HAProxyServer) error
	Update(id uint, updates map[string]interface{}) error
	Delete(opts ...DBOption) error
}

func NewIHAProxyServerRepo() IHAProxyServerRepo { return &HAProxyServerRepo{} }

type HAProxyServerRepo struct{}

func (r *HAProxyServerRepo) GetList(opts ...DBOption) ([]model.HAProxyServer, error) {
	var items []model.HAProxyServer
	db := getDb(opts...).Model(&model.HAProxyServer{})
	for _, opt := range opts {
		db = opt(db)
	}
	err := db.Order("created_at ASC").Find(&items).Error
	return items, err
}

func (r *HAProxyServerRepo) GetListByBackend(backendID uint) ([]model.HAProxyServer, error) {
	var items []model.HAProxyServer
	err := getDb().Where("backend_id = ?", backendID).Order("created_at ASC").Find(&items).Error
	return items, err
}

func (r *HAProxyServerRepo) Get(opts ...DBOption) (model.HAProxyServer, error) {
	var item model.HAProxyServer
	db := getDb(opts...).Model(&model.HAProxyServer{})
	for _, opt := range opts {
		db = opt(db)
	}
	err := db.First(&item).Error
	return item, err
}

func (r *HAProxyServerRepo) Create(item *model.HAProxyServer) error {
	return getDb().Create(item).Error
}

func (r *HAProxyServerRepo) Update(id uint, updates map[string]interface{}) error {
	return getDb().Model(&model.HAProxyServer{}).Where("id = ?", id).Updates(updates).Error
}

func (r *HAProxyServerRepo) Delete(opts ...DBOption) error {
	db := getDb(opts...)
	for _, opt := range opts {
		db = opt(db)
	}
	return db.Delete(&model.HAProxyServer{}).Error
}

// --- HAProxyACLRule Repo ---

type IHAProxyACLRepo interface {
	GetList(opts ...DBOption) ([]model.HAProxyACLRule, error)
	GetListByLB(lbID uint) ([]model.HAProxyACLRule, error)
	Get(opts ...DBOption) (model.HAProxyACLRule, error)
	Create(item *model.HAProxyACLRule) error
	Update(id uint, updates map[string]interface{}) error
	Delete(opts ...DBOption) error
	CountByBackendID(backendID uint) (int64, error)
}

func NewIHAProxyACLRepo() IHAProxyACLRepo { return &HAProxyACLRepo{} }

type HAProxyACLRepo struct{}

func (r *HAProxyACLRepo) GetList(opts ...DBOption) ([]model.HAProxyACLRule, error) {
	var items []model.HAProxyACLRule
	db := getDb(opts...).Model(&model.HAProxyACLRule{})
	for _, opt := range opts {
		db = opt(db)
	}
	err := db.Order("priority ASC").Find(&items).Error
	return items, err
}

func (r *HAProxyACLRepo) GetListByLB(lbID uint) ([]model.HAProxyACLRule, error) {
	var items []model.HAProxyACLRule
	err := getDb().Where("lb_id = ?", lbID).Order("priority ASC").Find(&items).Error
	return items, err
}

func (r *HAProxyACLRepo) Get(opts ...DBOption) (model.HAProxyACLRule, error) {
	var item model.HAProxyACLRule
	db := getDb(opts...).Model(&model.HAProxyACLRule{})
	for _, opt := range opts {
		db = opt(db)
	}
	err := db.First(&item).Error
	return item, err
}

func (r *HAProxyACLRepo) Create(item *model.HAProxyACLRule) error {
	return getDb().Create(item).Error
}

func (r *HAProxyACLRepo) Update(id uint, updates map[string]interface{}) error {
	return getDb().Model(&model.HAProxyACLRule{}).Where("id = ?", id).Updates(updates).Error
}

func (r *HAProxyACLRepo) Delete(opts ...DBOption) error {
	db := getDb(opts...)
	for _, opt := range opts {
		db = opt(db)
	}
	return db.Delete(&model.HAProxyACLRule{}).Error
}

func (r *HAProxyACLRepo) CountByBackendID(backendID uint) (int64, error) {
	var count int64
	err := getDb().Model(&model.HAProxyACLRule{}).Where("target_backend_id = ?", backendID).Count(&count).Error
	return count, err
}

// --- HAProxyConfigVersion Repo ---

type IHAProxyConfigVersionRepo interface {
	List(limit int) ([]model.HAProxyConfigVersion, error)
	Get(id uint) (model.HAProxyConfigVersion, error)
	Create(item *model.HAProxyConfigVersion) error
	PruneOld(keep int) error
}

func NewIHAProxyConfigVersionRepo() IHAProxyConfigVersionRepo {
	return &HAProxyConfigVersionRepo{}
}

type HAProxyConfigVersionRepo struct{}

func (r *HAProxyConfigVersionRepo) List(limit int) ([]model.HAProxyConfigVersion, error) {
	var items []model.HAProxyConfigVersion
	db := getDb().Model(&model.HAProxyConfigVersion{}).Order("created_at DESC")
	if limit > 0 {
		db = db.Limit(limit)
	}
	err := db.Find(&items).Error
	return items, err
}

func (r *HAProxyConfigVersionRepo) Get(id uint) (model.HAProxyConfigVersion, error) {
	var item model.HAProxyConfigVersion
	err := getDb().Where("id = ?", id).First(&item).Error
	return item, err
}

func (r *HAProxyConfigVersionRepo) Create(item *model.HAProxyConfigVersion) error {
	return getDb().Create(item).Error
}

func (r *HAProxyConfigVersionRepo) PruneOld(keep int) error {
	if keep <= 0 {
		keep = 50
	}
	var ids []uint
	err := getDb().Model(&model.HAProxyConfigVersion{}).
		Order("created_at DESC").
		Offset(keep).
		Pluck("id", &ids).Error
	if err != nil || len(ids) == 0 {
		return err
	}
	return getDb().Where("id IN ?", ids).Delete(&model.HAProxyConfigVersion{}).Error
}

// CountByBackendIDInLB 查询 backend 是否被任何 LB 引用为默认 backend
func CountHAProxyLBByDefaultBackend(backendID uint) (int64, error) {
	var count int64
	err := getDb().Model(&model.HAProxyLB{}).Where("default_backend_id = ?", backendID).Count(&count).Error
	return count, err
}
