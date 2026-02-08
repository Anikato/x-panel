package repo

import (
	"xpanel/app/model"
)

// --- AcmeAccount Repo ---

type IAcmeAccountRepo interface {
	GetList(opts ...DBOption) ([]model.AcmeAccount, error)
	Get(opts ...DBOption) (model.AcmeAccount, error)
	Create(item *model.AcmeAccount) error
	Update(id uint, updates map[string]interface{}) error
	Delete(opts ...DBOption) error
}

func NewIAcmeAccountRepo() IAcmeAccountRepo { return &AcmeAccountRepo{} }

type AcmeAccountRepo struct{}

func (r *AcmeAccountRepo) GetList(opts ...DBOption) ([]model.AcmeAccount, error) {
	var items []model.AcmeAccount
	db := getDB().Model(&model.AcmeAccount{})
	for _, opt := range opts {
		db = opt(db)
	}
	err := db.Order("created_at DESC").Find(&items).Error
	return items, err
}

func (r *AcmeAccountRepo) Get(opts ...DBOption) (model.AcmeAccount, error) {
	var item model.AcmeAccount
	db := getDB().Model(&model.AcmeAccount{})
	for _, opt := range opts {
		db = opt(db)
	}
	err := db.First(&item).Error
	return item, err
}

func (r *AcmeAccountRepo) Create(item *model.AcmeAccount) error {
	return getDB().Create(item).Error
}

func (r *AcmeAccountRepo) Update(id uint, updates map[string]interface{}) error {
	return getDB().Model(&model.AcmeAccount{}).Where("id = ?", id).Updates(updates).Error
}

func (r *AcmeAccountRepo) Delete(opts ...DBOption) error {
	db := getDB()
	for _, opt := range opts {
		db = opt(db)
	}
	return db.Delete(&model.AcmeAccount{}).Error
}

// --- DnsAccount Repo ---

type IDnsAccountRepo interface {
	GetList(opts ...DBOption) ([]model.DnsAccount, error)
	Get(opts ...DBOption) (model.DnsAccount, error)
	Create(item *model.DnsAccount) error
	Update(id uint, updates map[string]interface{}) error
	Delete(opts ...DBOption) error
}

func NewIDnsAccountRepo() IDnsAccountRepo { return &DnsAccountRepo{} }

type DnsAccountRepo struct{}

func (r *DnsAccountRepo) GetList(opts ...DBOption) ([]model.DnsAccount, error) {
	var items []model.DnsAccount
	db := getDB().Model(&model.DnsAccount{})
	for _, opt := range opts {
		db = opt(db)
	}
	err := db.Order("created_at DESC").Find(&items).Error
	return items, err
}

func (r *DnsAccountRepo) Get(opts ...DBOption) (model.DnsAccount, error) {
	var item model.DnsAccount
	db := getDB().Model(&model.DnsAccount{})
	for _, opt := range opts {
		db = opt(db)
	}
	err := db.First(&item).Error
	return item, err
}

func (r *DnsAccountRepo) Create(item *model.DnsAccount) error {
	return getDB().Create(item).Error
}

func (r *DnsAccountRepo) Update(id uint, updates map[string]interface{}) error {
	return getDB().Model(&model.DnsAccount{}).Where("id = ?", id).Updates(updates).Error
}

func (r *DnsAccountRepo) Delete(opts ...DBOption) error {
	db := getDB()
	for _, opt := range opts {
		db = opt(db)
	}
	return db.Delete(&model.DnsAccount{}).Error
}

// --- Certificate Repo ---

type ICertificateRepo interface {
	Page(page, pageSize int, opts ...DBOption) (int64, []model.Certificate, error)
	GetList(opts ...DBOption) ([]model.Certificate, error)
	Get(opts ...DBOption) (model.Certificate, error)
	Create(item *model.Certificate) error
	Update(id uint, updates map[string]interface{}) error
	Save(item *model.Certificate) error
	Delete(opts ...DBOption) error
}

func NewICertificateRepo() ICertificateRepo { return &CertificateRepo{} }

type CertificateRepo struct{}

func (r *CertificateRepo) Page(page, pageSize int, opts ...DBOption) (int64, []model.Certificate, error) {
	var (
		items []model.Certificate
		total int64
	)
	db := getDB().Model(&model.Certificate{})
	for _, opt := range opts {
		db = opt(db)
	}
	db.Count(&total)
	err := db.Offset((page - 1) * pageSize).Limit(pageSize).Order("created_at DESC").Find(&items).Error
	return total, items, err
}

func (r *CertificateRepo) GetList(opts ...DBOption) ([]model.Certificate, error) {
	var items []model.Certificate
	db := getDB().Model(&model.Certificate{})
	for _, opt := range opts {
		db = opt(db)
	}
	err := db.Find(&items).Error
	return items, err
}

func (r *CertificateRepo) Get(opts ...DBOption) (model.Certificate, error) {
	var item model.Certificate
	db := getDB().Model(&model.Certificate{})
	for _, opt := range opts {
		db = opt(db)
	}
	err := db.First(&item).Error
	return item, err
}

func (r *CertificateRepo) Create(item *model.Certificate) error {
	return getDB().Create(item).Error
}

func (r *CertificateRepo) Update(id uint, updates map[string]interface{}) error {
	return getDB().Model(&model.Certificate{}).Where("id = ?", id).Updates(updates).Error
}

func (r *CertificateRepo) Save(item *model.Certificate) error {
	return getDB().Save(item).Error
}

func (r *CertificateRepo) Delete(opts ...DBOption) error {
	db := getDB()
	for _, opt := range opts {
		db = opt(db)
	}
	return db.Delete(&model.Certificate{}).Error
}
