package service

import (
	"encoding/json"

	"xpanel/app/dto"
	"xpanel/app/model"
	"xpanel/app/repo"
	"xpanel/buserr"
	"xpanel/constant"
	"xpanel/global"
	sslutil "xpanel/utils/ssl"
)

type IAcmeAccountService interface {
	Create(req dto.AcmeAccountCreate) error
	Delete(id uint) error
	GetList() ([]dto.AcmeAccountInfo, error)
}

type AcmeAccountService struct {
	acmeRepo repo.IAcmeAccountRepo
}

func NewIAcmeAccountService() IAcmeAccountService {
	return &AcmeAccountService{acmeRepo: repo.NewIAcmeAccountRepo()}
}

func (s *AcmeAccountService) Create(req dto.AcmeAccountCreate) error {
	caDirURL := sslutil.GetCaDirURL(req.Type, req.CaDirURL)

	// 注册账户
	privateKeyPEM, url, err := sslutil.RegisterAccount(req.Email, req.KeyType, req.Type, req.CaDirURL)
	if err != nil {
		return buserr.WithDetail(constant.ErrSSLAcmeRegister, err.Error(), err)
	}

	account := model.AcmeAccount{
		Email:      req.Email,
		Type:       req.Type,
		KeyType:    req.KeyType,
		PrivateKey: privateKeyPEM,
		URL:        url,
		CaDirURL:   caDirURL,
	}
	if err := s.acmeRepo.Create(&account); err != nil {
		return buserr.WithDetail(constant.ErrInternalServer, err.Error(), err)
	}

	global.LOG.Infof("ACME account registered: %s (%s)", req.Email, req.Type)
	return nil
}

func (s *AcmeAccountService) Delete(id uint) error {
	return s.acmeRepo.Delete(repo.WithByID(id))
}

func (s *AcmeAccountService) GetList() ([]dto.AcmeAccountInfo, error) {
	accounts, err := s.acmeRepo.GetList()
	if err != nil {
		return nil, err
	}
	var items []dto.AcmeAccountInfo
	for _, a := range accounts {
		items = append(items, dto.AcmeAccountInfo{
			ID:       a.ID,
			Email:    a.Email,
			URL:      a.URL,
			Type:     a.Type,
			KeyType:  a.KeyType,
			CaDirURL: a.CaDirURL,
		})
	}
	return items, nil
}

// --- DNS Account Service ---

type IDnsAccountService interface {
	Create(req dto.DnsAccountCreate) error
	Update(req dto.DnsAccountUpdate) error
	Delete(id uint) error
	GetList() ([]dto.DnsAccountInfo, error)
}

type DnsAccountService struct {
	dnsRepo repo.IDnsAccountRepo
}

func NewIDnsAccountService() IDnsAccountService {
	return &DnsAccountService{dnsRepo: repo.NewIDnsAccountRepo()}
}

func (s *DnsAccountService) Create(req dto.DnsAccountCreate) error {
	authJSON, err := json.Marshal(req.Authorization)
	if err != nil {
		return buserr.WithDetail(constant.ErrInvalidParams, err.Error(), err)
	}
	account := model.DnsAccount{
		Name:          req.Name,
		Type:          req.Type,
		Authorization: string(authJSON),
	}
	if err := s.dnsRepo.Create(&account); err != nil {
		return buserr.WithDetail(constant.ErrInternalServer, err.Error(), err)
	}
	global.LOG.Infof("DNS account created: %s (%s)", req.Name, req.Type)
	return nil
}

func (s *DnsAccountService) Update(req dto.DnsAccountUpdate) error {
	authJSON, err := json.Marshal(req.Authorization)
	if err != nil {
		return buserr.WithDetail(constant.ErrInvalidParams, err.Error(), err)
	}
	updates := map[string]interface{}{
		"name":          req.Name,
		"type":          req.Type,
		"authorization": string(authJSON),
	}
	return s.dnsRepo.Update(req.ID, updates)
}

func (s *DnsAccountService) Delete(id uint) error {
	return s.dnsRepo.Delete(repo.WithByID(id))
}

func (s *DnsAccountService) GetList() ([]dto.DnsAccountInfo, error) {
	accounts, err := s.dnsRepo.GetList()
	if err != nil {
		return nil, err
	}
	var items []dto.DnsAccountInfo
	for _, a := range accounts {
		var auth map[string]string
		json.Unmarshal([]byte(a.Authorization), &auth)
		items = append(items, dto.DnsAccountInfo{
			ID:            a.ID,
			Name:          a.Name,
			Type:          a.Type,
			Authorization: auth,
		})
	}
	return items, nil
}

// --- 账户导入导出 ---

type IAccountExportService interface {
	Export() (*dto.AccountExport, error)
	Import(data dto.AccountExport) error
}

type AccountExportService struct {
	acmeRepo repo.IAcmeAccountRepo
	dnsRepo  repo.IDnsAccountRepo
}

func NewIAccountExportService() IAccountExportService {
	return &AccountExportService{
		acmeRepo: repo.NewIAcmeAccountRepo(),
		dnsRepo:  repo.NewIDnsAccountRepo(),
	}
}

func (s *AccountExportService) Export() (*dto.AccountExport, error) {
	acmeAccounts, err := s.acmeRepo.GetList()
	if err != nil {
		return nil, err
	}
	dnsAccounts, err := s.dnsRepo.GetList()
	if err != nil {
		return nil, err
	}

	export := &dto.AccountExport{}
	for _, a := range acmeAccounts {
		export.AcmeAccounts = append(export.AcmeAccounts, dto.AcmeAccountExportItem{
			Email:      a.Email,
			Type:       a.Type,
			KeyType:    a.KeyType,
			PrivateKey: a.PrivateKey,
			URL:        a.URL,
			CaDirURL:   a.CaDirURL,
			EabKid:     a.EabKid,
			EabHmacKey: a.EabHmacKey,
		})
	}
	for _, d := range dnsAccounts {
		var auth map[string]string
		json.Unmarshal([]byte(d.Authorization), &auth)
		export.DnsAccounts = append(export.DnsAccounts, dto.DnsAccountExportItem{
			Name:          d.Name,
			Type:          d.Type,
			Authorization: auth,
		})
	}
	return export, nil
}

func (s *AccountExportService) Import(data dto.AccountExport) error {
	imported := 0
	for _, a := range data.AcmeAccounts {
		account := model.AcmeAccount{
			Email:      a.Email,
			Type:       a.Type,
			KeyType:    a.KeyType,
			PrivateKey: a.PrivateKey,
			URL:        a.URL,
			CaDirURL:   a.CaDirURL,
			EabKid:     a.EabKid,
			EabHmacKey: a.EabHmacKey,
		}
		if err := s.acmeRepo.Create(&account); err != nil {
			global.LOG.Warnf("Import ACME account %s failed: %v", a.Email, err)
			continue
		}
		imported++
	}
	for _, d := range data.DnsAccounts {
		authJSON, _ := json.Marshal(d.Authorization)
		account := model.DnsAccount{
			Name:          d.Name,
			Type:          d.Type,
			Authorization: string(authJSON),
		}
		if err := s.dnsRepo.Create(&account); err != nil {
			global.LOG.Warnf("Import DNS account %s failed: %v", d.Name, err)
			continue
		}
		imported++
	}
	global.LOG.Infof("Imported %d accounts", imported)
	return nil
}
