package repo

import (
	"xpanel/app/model"
	"xpanel/global"

	"gorm.io/gorm"
)

type ICronjobRepo interface {
	Create(job *model.Cronjob) error
	Update(id uint, fields map[string]interface{}) error
	Delete(id uint) error
	Get(id uint) (*model.Cronjob, error)
	Page(page, pageSize int, opts ...DBOption) (int64, []model.Cronjob, error)
	List(opts ...DBOption) ([]model.Cronjob, error)
	CreateRecord(record *model.CronjobRecord) error
	PageRecord(page, pageSize int, opts ...DBOption) (int64, []model.CronjobRecord, error)
	DeleteRecordByCronjobID(cronjobID uint) error
	CleanRecords(cronjobID uint, retain int) error
}

func NewICronjobRepo() ICronjobRepo {
	return &CronjobRepo{}
}

type CronjobRepo struct{}

func (r *CronjobRepo) Create(job *model.Cronjob) error {
	return global.DB.Create(job).Error
}

func (r *CronjobRepo) Update(id uint, fields map[string]interface{}) error {
	return global.DB.Model(&model.Cronjob{}).Where("id = ?", id).Updates(fields).Error
}

func (r *CronjobRepo) Delete(id uint) error {
	return global.DB.Delete(&model.Cronjob{}, id).Error
}

func (r *CronjobRepo) Get(id uint) (*model.Cronjob, error) {
	var job model.Cronjob
	if err := global.DB.First(&job, id).Error; err != nil {
		return nil, err
	}
	return &job, nil
}

func (r *CronjobRepo) Page(page, pageSize int, opts ...DBOption) (int64, []model.Cronjob, error) {
	var total int64
	var items []model.Cronjob
	db := global.DB.Model(&model.Cronjob{})
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

func (r *CronjobRepo) List(opts ...DBOption) ([]model.Cronjob, error) {
	var items []model.Cronjob
	db := global.DB.Model(&model.Cronjob{})
	for _, opt := range opts {
		db = opt(db)
	}
	if err := db.Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (r *CronjobRepo) CreateRecord(record *model.CronjobRecord) error {
	return global.DB.Create(record).Error
}

func (r *CronjobRepo) PageRecord(page, pageSize int, opts ...DBOption) (int64, []model.CronjobRecord, error) {
	var total int64
	var items []model.CronjobRecord
	db := global.DB.Model(&model.CronjobRecord{})
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

func (r *CronjobRepo) DeleteRecordByCronjobID(cronjobID uint) error {
	return global.DB.Where("cronjob_id = ?", cronjobID).Delete(&model.CronjobRecord{}).Error
}

func (r *CronjobRepo) CleanRecords(cronjobID uint, retain int) error {
	var ids []uint
	global.DB.Model(&model.CronjobRecord{}).Where("cronjob_id = ?", cronjobID).
		Order("created_at desc").Offset(retain).Pluck("id", &ids)
	if len(ids) == 0 {
		return nil
	}
	return global.DB.Where("id IN ?", ids).Delete(&model.CronjobRecord{}).Error
}

func WithCronjobType(t string) DBOption {
	return func(db *gorm.DB) *gorm.DB {
		if t != "" {
			return db.Where("type = ?", t)
		}
		return db
	}
}

func WithCronjobStatus(s string) DBOption {
	return func(db *gorm.DB) *gorm.DB {
		if s != "" {
			return db.Where("status = ?", s)
		}
		return db
	}
}

func WithCronjobID(id uint) DBOption {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("cronjob_id = ?", id)
	}
}

func WithRecordStatus(s string) DBOption {
	return func(db *gorm.DB) *gorm.DB {
		if s != "" {
			return db.Where("status = ?", s)
		}
		return db
	}
}
