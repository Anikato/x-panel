package repo

import (
	"xpanel/app/model"
)

// --- CertSource Repo ---

type ICertSourceRepo interface {
	GetList(opts ...DBOption) ([]model.CertSource, error)
	Get(opts ...DBOption) (model.CertSource, error)
	Create(item *model.CertSource) error
	Save(item *model.CertSource) error
	Update(id uint, updates map[string]interface{}) error
	Delete(opts ...DBOption) error
}

func NewICertSourceRepo() ICertSourceRepo { return &CertSourceRepo{} }

type CertSourceRepo struct{}

func (r *CertSourceRepo) GetList(opts ...DBOption) ([]model.CertSource, error) {
	var items []model.CertSource
	db := getDb(opts...).Model(&model.CertSource{})
	err := db.Order("created_at DESC").Find(&items).Error
	return items, err
}

func (r *CertSourceRepo) Get(opts ...DBOption) (model.CertSource, error) {
	var item model.CertSource
	db := getDb(opts...).Model(&model.CertSource{})
	err := db.First(&item).Error
	return item, err
}

func (r *CertSourceRepo) Create(item *model.CertSource) error {
	return getDb().Create(item).Error
}

func (r *CertSourceRepo) Save(item *model.CertSource) error {
	return getDb().Save(item).Error
}

func (r *CertSourceRepo) Update(id uint, updates map[string]interface{}) error {
	return getDb().Model(&model.CertSource{}).Where("id = ?", id).Updates(updates).Error
}

func (r *CertSourceRepo) Delete(opts ...DBOption) error {
	db := getDb(opts...)
	return db.Delete(&model.CertSource{}).Error
}

// --- CertSyncLog Repo ---

type ICertSyncLogRepo interface {
	Page(page, pageSize int, opts ...DBOption) (int64, []model.CertSyncLog, error)
	Create(item *model.CertSyncLog) error
	DeleteBySourceID(sourceID uint) error
}

func NewICertSyncLogRepo() ICertSyncLogRepo { return &CertSyncLogRepo{} }

type CertSyncLogRepo struct{}

func (r *CertSyncLogRepo) Page(page, pageSize int, opts ...DBOption) (int64, []model.CertSyncLog, error) {
	var (
		items []model.CertSyncLog
		total int64
	)
	db := getDb(opts...).Model(&model.CertSyncLog{})
	db.Count(&total)
	err := db.Offset((page - 1) * pageSize).Limit(pageSize).Order("created_at DESC").Find(&items).Error
	return total, items, err
}

func (r *CertSyncLogRepo) Create(item *model.CertSyncLog) error {
	return getDb().Create(item).Error
}

func (r *CertSyncLogRepo) DeleteBySourceID(sourceID uint) error {
	return getDb().Where("source_id = ?", sourceID).Delete(&model.CertSyncLog{}).Error
}
