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
	db := getDB().Model(&model.CertSource{})
	for _, opt := range opts {
		db = opt(db)
	}
	if err := db.Order("created_at DESC").Find(&items).Error; err != nil {
		return nil, err
	}
	for i := range items {
		if err := revealCertSource(&items[i]); err != nil {
			return nil, err
		}
	}
	return items, nil
}

func (r *CertSourceRepo) Get(opts ...DBOption) (model.CertSource, error) {
	var item model.CertSource
	db := getDB().Model(&model.CertSource{})
	for _, opt := range opts {
		db = opt(db)
	}
	if err := db.First(&item).Error; err != nil {
		return item, err
	}
	return item, revealCertSource(&item)
}

func (r *CertSourceRepo) Create(item *model.CertSource) error {
	stored := *item
	if err := protectCertSource(&stored); err != nil {
		return err
	}
	if err := getDB().Create(&stored).Error; err != nil {
		return err
	}
	*item = stored
	return revealCertSource(item)
}

func (r *CertSourceRepo) Save(item *model.CertSource) error {
	stored := *item
	if err := protectCertSource(&stored); err != nil {
		return err
	}
	if err := getDB().Save(&stored).Error; err != nil {
		return err
	}
	*item = stored
	return revealCertSource(item)
}

func (r *CertSourceRepo) Update(id uint, updates map[string]interface{}) error {
	protected, err := protectUpdates("cert_sources", updates)
	if err != nil {
		return err
	}
	return getDB().Model(&model.CertSource{}).Where("id = ?", id).Updates(protected).Error
}

func (r *CertSourceRepo) Delete(opts ...DBOption) error {
	db := getDB()
	for _, opt := range opts {
		db = opt(db)
	}
	return db.Delete(&model.CertSource{}).Error
}

func protectCertSource(item *model.CertSource) error {
	return protectFields(secureField{Scope: "cert_sources.token", Value: &item.Token})
}

func revealCertSource(item *model.CertSource) error {
	return revealFields(secureField{Scope: "cert_sources.token", Value: &item.Token})
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
	db := getDB().Model(&model.CertSyncLog{})
	for _, opt := range opts {
		db = opt(db)
	}
	db.Count(&total)
	err := db.Offset((page - 1) * pageSize).Limit(pageSize).Order("created_at DESC").Find(&items).Error
	return total, items, err
}

func (r *CertSyncLogRepo) Create(item *model.CertSyncLog) error {
	return getDB().Create(item).Error
}

func (r *CertSyncLogRepo) DeleteBySourceID(sourceID uint) error {
	return getDB().Where("source_id = ?", sourceID).Delete(&model.CertSyncLog{}).Error
}
