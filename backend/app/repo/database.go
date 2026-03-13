package repo

import (
	"xpanel/app/model"
	"xpanel/global"

	"gorm.io/gorm"
)

type IDatabaseRepo interface {
	CreateServer(s *model.DatabaseServer) error
	UpdateServer(id uint, fields map[string]interface{}) error
	DeleteServer(id uint) error
	GetServer(id uint) (*model.DatabaseServer, error)
	PageServer(page, pageSize int, opts ...DBOption) (int64, []model.DatabaseServer, error)
	ListServers(opts ...DBOption) ([]model.DatabaseServer, error)

	CreateInstance(i *model.DatabaseInstance) error
	UpdateInstance(id uint, fields map[string]interface{}) error
	DeleteInstance(id uint) error
	GetInstance(id uint) (*model.DatabaseInstance, error)
	PageInstance(page, pageSize int, opts ...DBOption) (int64, []model.DatabaseInstance, error)
	ListInstancesByServerID(serverID uint) ([]model.DatabaseInstance, error)
	DeleteInstanceByServerID(serverID uint) error
}

func NewIDatabaseRepo() IDatabaseRepo {
	return &DatabaseRepo{}
}

type DatabaseRepo struct{}

func (r *DatabaseRepo) CreateServer(s *model.DatabaseServer) error {
	return global.DB.Create(s).Error
}

func (r *DatabaseRepo) UpdateServer(id uint, fields map[string]interface{}) error {
	return global.DB.Model(&model.DatabaseServer{}).Where("id = ?", id).Updates(fields).Error
}

func (r *DatabaseRepo) DeleteServer(id uint) error {
	return global.DB.Delete(&model.DatabaseServer{}, id).Error
}

func (r *DatabaseRepo) GetServer(id uint) (*model.DatabaseServer, error) {
	var s model.DatabaseServer
	if err := global.DB.First(&s, id).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *DatabaseRepo) PageServer(page, pageSize int, opts ...DBOption) (int64, []model.DatabaseServer, error) {
	var total int64
	var items []model.DatabaseServer
	db := global.DB.Model(&model.DatabaseServer{})
	for _, opt := range opts {
		db = opt(db)
	}
	if err := db.Count(&total).Error; err != nil {
		return 0, nil, err
	}
	if err := db.Offset((page - 1) * pageSize).Limit(pageSize).Order("created_at desc").Find(&items).Error; err != nil {
		return 0, nil, err
	}
	return total, items, nil
}

func (r *DatabaseRepo) ListServers(opts ...DBOption) ([]model.DatabaseServer, error) {
	var items []model.DatabaseServer
	db := global.DB.Model(&model.DatabaseServer{})
	for _, opt := range opts {
		db = opt(db)
	}
	return items, db.Find(&items).Error
}

func (r *DatabaseRepo) CreateInstance(i *model.DatabaseInstance) error {
	return global.DB.Create(i).Error
}

func (r *DatabaseRepo) UpdateInstance(id uint, fields map[string]interface{}) error {
	return global.DB.Model(&model.DatabaseInstance{}).Where("id = ?", id).Updates(fields).Error
}

func (r *DatabaseRepo) DeleteInstance(id uint) error {
	return global.DB.Delete(&model.DatabaseInstance{}, id).Error
}

func (r *DatabaseRepo) GetInstance(id uint) (*model.DatabaseInstance, error) {
	var i model.DatabaseInstance
	if err := global.DB.First(&i, id).Error; err != nil {
		return nil, err
	}
	return &i, nil
}

func (r *DatabaseRepo) PageInstance(page, pageSize int, opts ...DBOption) (int64, []model.DatabaseInstance, error) {
	var total int64
	var items []model.DatabaseInstance
	db := global.DB.Model(&model.DatabaseInstance{})
	for _, opt := range opts {
		db = opt(db)
	}
	if err := db.Count(&total).Error; err != nil {
		return 0, nil, err
	}
	if err := db.Offset((page - 1) * pageSize).Limit(pageSize).Order("created_at desc").Find(&items).Error; err != nil {
		return 0, nil, err
	}
	return total, items, nil
}

func (r *DatabaseRepo) ListInstancesByServerID(serverID uint) ([]model.DatabaseInstance, error) {
	var items []model.DatabaseInstance
	if err := global.DB.Where("server_id = ?", serverID).Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (r *DatabaseRepo) DeleteInstanceByServerID(serverID uint) error {
	return global.DB.Where("server_id = ?", serverID).Delete(&model.DatabaseInstance{}).Error
}

func WithServerType(t string) DBOption {
	return func(db *gorm.DB) *gorm.DB {
		if t != "" {
			return db.Where("type = ?", t)
		}
		return db
	}
}

func WithServerID(id uint) DBOption {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("server_id = ?", id)
	}
}
