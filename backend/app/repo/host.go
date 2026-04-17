package repo

import (
	"xpanel/app/model"

	"gorm.io/gorm"
)

type IHostRepo interface {
	Page(page, pageSize int, opts ...DBOption) (int64, []model.Host, error)
	GetList(opts ...DBOption) ([]model.Host, error)
	Get(opts ...DBOption) (model.Host, error)
	Create(host *model.Host) error
	Update(id uint, updates map[string]interface{}) error
	Delete(opts ...DBOption) error
}

func NewIHostRepo() IHostRepo { return &HostRepo{} }

type HostRepo struct{}

func (r *HostRepo) Page(page, pageSize int, opts ...DBOption) (int64, []model.Host, error) {
	var (
		items []model.Host
		total int64
	)
	db := getDb(opts...).Model(&model.Host{})
	for _, opt := range opts {
		db = opt(db)
	}
	db.Count(&total)
	err := db.Offset((page - 1) * pageSize).Limit(pageSize).Order("created_at DESC").Find(&items).Error
	return total, items, err
}

func (r *HostRepo) GetList(opts ...DBOption) ([]model.Host, error) {
	var items []model.Host
	db := getDb(opts...).Model(&model.Host{})
	for _, opt := range opts {
		db = opt(db)
	}
	err := db.Find(&items).Error
	return items, err
}

func (r *HostRepo) Get(opts ...DBOption) (model.Host, error) {
	var item model.Host
	db := getDb(opts...).Model(&model.Host{})
	for _, opt := range opts {
		db = opt(db)
	}
	err := db.First(&item).Error
	return item, err
}

func (r *HostRepo) Create(host *model.Host) error {
	return getDb().Create(host).Error
}

func (r *HostRepo) Update(id uint, updates map[string]interface{}) error {
	return getDb().Model(&model.Host{}).Where("id = ?", id).Updates(updates).Error
}

func (r *HostRepo) Delete(opts ...DBOption) error {
	db := getDb(opts...)
	for _, opt := range opts {
		db = opt(db)
	}
	return db.Delete(&model.Host{}).Error
}

// --- Command Repo ---

type ICommandRepo interface {
	Page(page, pageSize int, opts ...DBOption) (int64, []model.Command, error)
	GetList(opts ...DBOption) ([]model.Command, error)
	Get(opts ...DBOption) (model.Command, error)
	Create(cmd *model.Command) error
	Update(id uint, updates map[string]interface{}) error
	Delete(opts ...DBOption) error
}

func NewICommandRepo() ICommandRepo { return &CommandRepo{} }

type CommandRepo struct{}

func (r *CommandRepo) Page(page, pageSize int, opts ...DBOption) (int64, []model.Command, error) {
	var (
		items []model.Command
		total int64
	)
	db := getDb(opts...).Model(&model.Command{})
	for _, opt := range opts {
		db = opt(db)
	}
	db.Count(&total)
	err := db.Offset((page - 1) * pageSize).Limit(pageSize).Order("created_at DESC").Find(&items).Error
	return total, items, err
}

func (r *CommandRepo) GetList(opts ...DBOption) ([]model.Command, error) {
	var items []model.Command
	db := getDb(opts...).Model(&model.Command{})
	for _, opt := range opts {
		db = opt(db)
	}
	err := db.Find(&items).Error
	return items, err
}

func (r *CommandRepo) Get(opts ...DBOption) (model.Command, error) {
	var item model.Command
	db := getDb(opts...).Model(&model.Command{})
	for _, opt := range opts {
		db = opt(db)
	}
	err := db.First(&item).Error
	return item, err
}

func (r *CommandRepo) Create(cmd *model.Command) error {
	return getDb().Create(cmd).Error
}

func (r *CommandRepo) Update(id uint, updates map[string]interface{}) error {
	return getDb().Model(&model.Command{}).Where("id = ?", id).Updates(updates).Error
}

func (r *CommandRepo) Delete(opts ...DBOption) error {
	db := getDb(opts...)
	for _, opt := range opts {
		db = opt(db)
	}
	return db.Delete(&model.Command{}).Error
}

// --- Group Repo ---

type IGroupRepo interface {
	GetList(opts ...DBOption) ([]model.Group, error)
	Get(opts ...DBOption) (model.Group, error)
	Create(group *model.Group) error
	Update(id uint, updates map[string]interface{}) error
	Delete(opts ...DBOption) error
}

func NewIGroupRepo() IGroupRepo { return &GroupRepo{} }

type GroupRepo struct{}

func (r *GroupRepo) GetList(opts ...DBOption) ([]model.Group, error) {
	var items []model.Group
	db := getDb(opts...).Model(&model.Group{})
	for _, opt := range opts {
		db = opt(db)
	}
	err := db.Find(&items).Error
	return items, err
}

func (r *GroupRepo) Get(opts ...DBOption) (model.Group, error) {
	var item model.Group
	db := getDb(opts...).Model(&model.Group{})
	for _, opt := range opts {
		db = opt(db)
	}
	err := db.First(&item).Error
	return item, err
}

func (r *GroupRepo) Create(group *model.Group) error {
	return getDb().Create(group).Error
}

func (r *GroupRepo) Update(id uint, updates map[string]interface{}) error {
	return getDb().Model(&model.Group{}).Where("id = ?", id).Updates(updates).Error
}

func (r *GroupRepo) Delete(opts ...DBOption) error {
	db := getDb(opts...)
	for _, opt := range opts {
		db = opt(db)
	}
	return db.Delete(&model.Group{}).Error
}

// WithByGroupID 按 GroupID 查询
func WithByGroupID(groupID uint) DBOption {
	return func(db *gorm.DB) *gorm.DB {
		if groupID == 0 {
			return db
		}
		return db.Where("group_id = ?", groupID)
	}
}
