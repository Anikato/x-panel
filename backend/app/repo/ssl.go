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
	if err := db.Order("created_at DESC").Find(&items).Error; err != nil {
		return nil, err
	}
	for i := range items {
		if err := revealAcmeAccount(&items[i]); err != nil {
			return nil, err
		}
	}
	return items, nil
}

func (r *AcmeAccountRepo) Get(opts ...DBOption) (model.AcmeAccount, error) {
	var item model.AcmeAccount
	db := getDB().Model(&model.AcmeAccount{})
	for _, opt := range opts {
		db = opt(db)
	}
	if err := db.First(&item).Error; err != nil {
		return item, err
	}
	return item, revealAcmeAccount(&item)
}

func (r *AcmeAccountRepo) Create(item *model.AcmeAccount) error {
	stored := *item
	if err := protectAcmeAccount(&stored); err != nil {
		return err
	}
	if err := getDB().Create(&stored).Error; err != nil {
		return err
	}
	*item = stored
	return revealAcmeAccount(item)
}

func (r *AcmeAccountRepo) Update(id uint, updates map[string]interface{}) error {
	protected, err := protectUpdates("acme_accounts", updates)
	if err != nil {
		return err
	}
	return getDB().Model(&model.AcmeAccount{}).Where("id = ?", id).Updates(protected).Error
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
	if err := db.Order("created_at DESC").Find(&items).Error; err != nil {
		return nil, err
	}
	for i := range items {
		if err := revealDNSAccount(&items[i]); err != nil {
			return nil, err
		}
	}
	return items, nil
}

func (r *DnsAccountRepo) Get(opts ...DBOption) (model.DnsAccount, error) {
	var item model.DnsAccount
	db := getDB().Model(&model.DnsAccount{})
	for _, opt := range opts {
		db = opt(db)
	}
	if err := db.First(&item).Error; err != nil {
		return item, err
	}
	return item, revealDNSAccount(&item)
}

func (r *DnsAccountRepo) Create(item *model.DnsAccount) error {
	stored := *item
	if err := protectDNSAccount(&stored); err != nil {
		return err
	}
	if err := getDB().Create(&stored).Error; err != nil {
		return err
	}
	*item = stored
	return revealDNSAccount(item)
}

func (r *DnsAccountRepo) Update(id uint, updates map[string]interface{}) error {
	protected, err := protectUpdates("dns_accounts", updates)
	if err != nil {
		return err
	}
	return getDB().Model(&model.DnsAccount{}).Where("id = ?", id).Updates(protected).Error
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
	if err := db.Offset((page - 1) * pageSize).Limit(pageSize).Order("created_at DESC").Find(&items).Error; err != nil {
		return total, nil, err
	}
	for i := range items {
		if err := revealCertificate(&items[i]); err != nil {
			return total, nil, err
		}
	}
	return total, items, nil
}

func (r *CertificateRepo) GetList(opts ...DBOption) ([]model.Certificate, error) {
	var items []model.Certificate
	db := getDB().Model(&model.Certificate{})
	for _, opt := range opts {
		db = opt(db)
	}
	if err := db.Find(&items).Error; err != nil {
		return nil, err
	}
	for i := range items {
		if err := revealCertificate(&items[i]); err != nil {
			return nil, err
		}
	}
	return items, nil
}

func (r *CertificateRepo) Get(opts ...DBOption) (model.Certificate, error) {
	var item model.Certificate
	db := getDB().Model(&model.Certificate{})
	for _, opt := range opts {
		db = opt(db)
	}
	if err := db.First(&item).Error; err != nil {
		return item, err
	}
	return item, revealCertificate(&item)
}

func (r *CertificateRepo) Create(item *model.Certificate) error {
	stored := *item
	if err := protectCertificate(&stored); err != nil {
		return err
	}
	if err := getDB().Create(&stored).Error; err != nil {
		return err
	}
	*item = stored
	return revealCertificate(item)
}

func (r *CertificateRepo) Update(id uint, updates map[string]interface{}) error {
	protected, err := protectUpdates("certificates", updates)
	if err != nil {
		return err
	}
	return getDB().Model(&model.Certificate{}).Where("id = ?", id).Updates(protected).Error
}

func (r *CertificateRepo) Save(item *model.Certificate) error {
	stored := *item
	if err := protectCertificate(&stored); err != nil {
		return err
	}
	if err := getDB().Save(&stored).Error; err != nil {
		return err
	}
	*item = stored
	return revealCertificate(item)
}

func (r *CertificateRepo) Delete(opts ...DBOption) error {
	db := getDB()
	for _, opt := range opts {
		db = opt(db)
	}
	return db.Delete(&model.Certificate{}).Error
}

func protectAcmeAccount(item *model.AcmeAccount) error {
	return protectFields(
		secureField{Scope: "acme_accounts.private_key", Value: &item.PrivateKey},
		secureField{Scope: "acme_accounts.eab_hmac_key", Value: &item.EabHmacKey},
	)
}

func revealAcmeAccount(item *model.AcmeAccount) error {
	return revealFields(
		secureField{Scope: "acme_accounts.private_key", Value: &item.PrivateKey},
		secureField{Scope: "acme_accounts.eab_hmac_key", Value: &item.EabHmacKey},
	)
}

func protectDNSAccount(item *model.DnsAccount) error {
	return protectFields(secureField{Scope: "dns_accounts.authorization", Value: &item.Authorization})
}

func revealDNSAccount(item *model.DnsAccount) error {
	return revealFields(secureField{Scope: "dns_accounts.authorization", Value: &item.Authorization})
}

func protectCertificate(item *model.Certificate) error {
	return protectFields(secureField{Scope: "certificates.private_key", Value: &item.PrivateKey})
}

func revealCertificate(item *model.Certificate) error {
	return revealFields(secureField{Scope: "certificates.private_key", Value: &item.PrivateKey})
}
