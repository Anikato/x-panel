package service

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"xpanel/app/dto"
	"xpanel/app/model"
	"xpanel/app/repo"
	"xpanel/buserr"
	"xpanel/constant"
	"xpanel/global"
	sslutil "xpanel/utils/ssl"
)

type ICertificateService interface {
	Create(req dto.CertificateCreate) error
	Update(req dto.CertificateUpdate) error
	Upload(req dto.CertificateUpload) error
	Delete(id uint) error
	SearchWithPage(req dto.SearchCertReq) (int64, []dto.CertificateInfo, error)
	GetDetail(id uint) (*dto.CertificateDetail, error)
	Apply(id uint) error
	Renew(id uint) error
	GetSSLDir() string
	UpdateSSLDir(dir string) error
	GetLog(id uint) (string, error)
}

type CertificateService struct {
	certRepo    repo.ICertificateRepo
	acmeRepo    repo.IAcmeAccountRepo
	dnsRepo     repo.IDnsAccountRepo
	settingRepo repo.ISettingRepo
}

func NewICertificateService() ICertificateService {
	return &CertificateService{
		certRepo:    repo.NewICertificateRepo(),
		acmeRepo:    repo.NewIAcmeAccountRepo(),
		dnsRepo:     repo.NewIDnsAccountRepo(),
		settingRepo: repo.NewISettingRepo(),
	}
}

func (s *CertificateService) Create(req dto.CertificateCreate) error {
	cert := model.Certificate{
		PrimaryDomain: req.PrimaryDomain,
		Domains:       req.OtherDomains,
		Provider:      req.Provider,
		Type:          "autoApply",
		AcmeAccountID: req.AcmeAccountID,
		DnsAccountID:  req.DnsAccountID,
		KeyType:       req.KeyType,
		AutoRenew:     req.AutoRenew,
		Description:   req.Description,
		Status:        "ready",
	}
	if cert.KeyType == "" {
		cert.KeyType = "2048"
	}
	if err := s.certRepo.Create(&cert); err != nil {
		return buserr.WithDetail(constant.ErrInternalServer, err.Error(), err)
	}
	global.LOG.Infof("Certificate created for domain: %s", req.PrimaryDomain)

	if req.Apply {
		go func() {
			if err := s.Apply(cert.ID); err != nil {
				global.LOG.Errorf("Auto apply certificate failed: %v", err)
			}
		}()
	}
	return nil
}

func (s *CertificateService) Update(req dto.CertificateUpdate) error {
	updates := map[string]interface{}{
		"auto_renew":  req.AutoRenew,
		"description": req.Description,
	}
	if req.PrimaryDomain != "" {
		updates["primary_domain"] = req.PrimaryDomain
	}
	if req.OtherDomains != "" {
		updates["domains"] = req.OtherDomains
	}
	return s.certRepo.Update(req.ID, updates)
}

func (s *CertificateService) Upload(req dto.CertificateUpload) error {
	// 解析证书获取域名和过期时间
	certInfo, err := parseCertPEM(req.Certificate)
	if err != nil {
		return buserr.WithDetail(constant.ErrInvalidParams, "证书解析失败: "+err.Error(), err)
	}

	cert := model.Certificate{
		PrimaryDomain: certInfo.primaryDomain,
		Domains:       strings.Join(certInfo.domains, ","),
		Provider:      "manual",
		Type:          "upload",
		Pem:           req.Certificate,
		PrivateKey:    req.PrivateKey,
		ExpireDate:    certInfo.expireDate,
		StartDate:     certInfo.startDate,
		Status:        "applied",
		Description:   req.Description,
		KeyType:       "upload",
	}
	if err := s.certRepo.Create(&cert); err != nil {
		return buserr.WithDetail(constant.ErrInternalServer, err.Error(), err)
	}

	// 保存到文件系统
	if err := s.saveCertFiles(cert); err != nil {
		global.LOG.Warnf("Save cert files failed: %v", err)
	}
	return nil
}

func (s *CertificateService) Delete(id uint) error {
	cert, err := s.certRepo.Get(repo.WithByID(id))
	if err != nil {
		return buserr.New(constant.ErrRecordNotFound)
	}
	// 删除证书文件
	sslDir := s.GetSSLDir()
	certDir := filepath.Join(sslDir, "certs", cert.PrimaryDomain)
	os.RemoveAll(certDir)

	return s.certRepo.Delete(repo.WithByID(id))
}

func (s *CertificateService) SearchWithPage(req dto.SearchCertReq) (int64, []dto.CertificateInfo, error) {
	var opts []repo.DBOption
	if req.Info != "" {
		opts = append(opts, repo.WithLikeDomain(req.Info))
	}
	total, certs, err := s.certRepo.Page(req.Page, req.PageSize, opts...)
	if err != nil {
		return 0, nil, err
	}

	// 预加载账户信息
	acmeAccounts, _ := s.acmeRepo.GetList()
	dnsAccounts, _ := s.dnsRepo.GetList()
	acmeMap := make(map[uint]string)
	dnsMap := make(map[uint]string)
	for _, a := range acmeAccounts {
		acmeMap[a.ID] = a.Email
	}
	for _, d := range dnsAccounts {
		dnsMap[d.ID] = d.Name
	}

	var items []dto.CertificateInfo
	for _, c := range certs {
		items = append(items, dto.CertificateInfo{
			ID:               c.ID,
			PrimaryDomain:    c.PrimaryDomain,
			Domains:          c.Domains,
			Provider:         c.Provider,
			Type:             c.Type,
			AcmeAccountID:    c.AcmeAccountID,
			DnsAccountID:     c.DnsAccountID,
			KeyType:          c.KeyType,
			AutoRenew:        c.AutoRenew,
			ExpireDate:       c.ExpireDate,
			StartDate:        c.StartDate,
			Status:           c.Status,
			Message:          c.Message,
			Description:      c.Description,
			CertURL:          c.CertURL,
			AcmeAccountEmail: acmeMap[c.AcmeAccountID],
			DnsAccountName:   dnsMap[c.DnsAccountID],
		})
	}
	return total, items, nil
}

func (s *CertificateService) GetDetail(id uint) (*dto.CertificateDetail, error) {
	cert, err := s.certRepo.Get(repo.WithByID(id))
	if err != nil {
		return nil, buserr.New(constant.ErrRecordNotFound)
	}

	sslDir := s.GetSSLDir()
	certPath := filepath.Join(sslDir, "certs", cert.PrimaryDomain)

	acmeEmail := ""
	dnsName := ""
	if cert.AcmeAccountID > 0 {
		if a, err := s.acmeRepo.Get(repo.WithByID(cert.AcmeAccountID)); err == nil {
			acmeEmail = a.Email
		}
	}
	if cert.DnsAccountID > 0 {
		if d, err := s.dnsRepo.Get(repo.WithByID(cert.DnsAccountID)); err == nil {
			dnsName = d.Name
		}
	}

	return &dto.CertificateDetail{
		CertificateInfo: dto.CertificateInfo{
			ID:               cert.ID,
			PrimaryDomain:    cert.PrimaryDomain,
			Domains:          cert.Domains,
			Provider:         cert.Provider,
			Type:             cert.Type,
			AcmeAccountID:    cert.AcmeAccountID,
			DnsAccountID:     cert.DnsAccountID,
			KeyType:          cert.KeyType,
			AutoRenew:        cert.AutoRenew,
			ExpireDate:       cert.ExpireDate,
			StartDate:        cert.StartDate,
			Status:           cert.Status,
			Message:          cert.Message,
			Description:      cert.Description,
			CertURL:          cert.CertURL,
			AcmeAccountEmail: acmeEmail,
			DnsAccountName:   dnsName,
		},
		Pem:        cert.Pem,
		PrivateKey: cert.PrivateKey,
		FilePath:   certPath,
	}, nil
}

func (s *CertificateService) Apply(id uint) error {
	cert, err := s.certRepo.Get(repo.WithByID(id))
	if err != nil {
		return buserr.New(constant.ErrRecordNotFound)
	}

	// 更新状态为 applying
	s.certRepo.Update(id, map[string]interface{}{"status": "applying", "message": ""})

	// 创建日志文件
	logger, logFile := s.openSSLLog(cert)
	if logFile != nil {
		defer logFile.Close()
	}

	logger.Printf("[开始] 申请证书: %s", cert.PrimaryDomain)

	acme, err := s.acmeRepo.Get(repo.WithByID(cert.AcmeAccountID))
	if err != nil {
		errMsg := "ACME 账户不存在"
		logger.Printf("[错误] %s (ID=%d)", errMsg, cert.AcmeAccountID)
		s.certRepo.Update(id, map[string]interface{}{"status": "error", "message": errMsg})
		return fmt.Errorf("ACME account not found")
	}
	logger.Printf("[信息] ACME 账户: %s (%s)", acme.Email, acme.Type)

	logger.Printf("[信息] 正在创建 ACME 客户端 (账户URL: %s)...", acme.URL)
	client, err := sslutil.NewAcmeClient(acme.Email, acme.PrivateKey, acme.KeyType, acme.Type, acme.CaDirURL, acme.URL)
	if err != nil {
		logger.Printf("[错误] ACME 客户端创建失败: %v", err)
		s.certRepo.Update(id, map[string]interface{}{"status": "error", "message": err.Error()})
		return err
	}
	logger.Printf("[成功] ACME 客户端创建完成")

	// 设置 DNS 验证
	if cert.Provider == "dns" {
		dns, err := s.dnsRepo.Get(repo.WithByID(cert.DnsAccountID))
		if err != nil {
			errMsg := "DNS 账户不存在"
			logger.Printf("[错误] %s (ID=%d)", errMsg, cert.DnsAccountID)
			s.certRepo.Update(id, map[string]interface{}{"status": "error", "message": errMsg})
			return fmt.Errorf("DNS account not found")
		}
		logger.Printf("[信息] DNS 账户: %s (%s)", dns.Name, dns.Type)
		logger.Printf("[信息] 正在配置 DNS 验证提供商...")
		if err := client.SetDNSProvider(dns.Type, dns.Authorization); err != nil {
			logger.Printf("[错误] DNS 提供商配置失败: %v", err)
			s.certRepo.Update(id, map[string]interface{}{"status": "error", "message": err.Error()})
			return err
		}
		logger.Printf("[成功] DNS 提供商配置完成")
	}

	// 收集域名
	domains := []string{cert.PrimaryDomain}
	if cert.Domains != "" {
		domains = append(domains, strings.Split(cert.Domains, ",")...)
	}
	logger.Printf("[信息] 申请域名: %s", strings.Join(domains, ", "))
	logger.Printf("[信息] 密钥类型: %s", cert.KeyType)

	global.LOG.Infof("Applying certificate for domains: %v", domains)

	// 申请证书
	logger.Printf("[信息] 正在向 CA 发起证书申请（此步骤可能耗时数分钟）...")
	certRes, err := client.ObtainCertificate(domains, cert.KeyType)
	if err != nil {
		logger.Printf("[错误] 证书申请失败: %v", err)
		s.certRepo.Update(id, map[string]interface{}{"status": "error", "message": err.Error()})
		return err
	}
	logger.Printf("[成功] 证书申请成功!")

	// 解析证书信息
	certInfo, _ := parseCertPEM(string(certRes.Certificate))

	// 更新数据库
	updates := map[string]interface{}{
		"pem":         string(certRes.Certificate),
		"private_key": string(certRes.PrivateKey),
		"cert_url":    certRes.CertURL,
		"status":      "applied",
		"message":     "",
	}
	if certInfo != nil {
		updates["expire_date"] = certInfo.expireDate
		updates["start_date"] = certInfo.startDate
		logger.Printf("[信息] 证书有效期: %s 至 %s", certInfo.startDate.Format("2006-01-02"), certInfo.expireDate.Format("2006-01-02"))
	}
	s.certRepo.Update(id, updates)

	// 保存到文件系统
	cert.Pem = string(certRes.Certificate)
	cert.PrivateKey = string(certRes.PrivateKey)
	if err := s.saveCertFiles(cert); err != nil {
		logger.Printf("[警告] 证书文件保存失败: %v", err)
	} else {
		sslDir := s.GetSSLDir()
		logger.Printf("[成功] 证书文件已保存到: %s", filepath.Join(sslDir, "certs", cert.PrimaryDomain))
	}

	logger.Printf("[完成] 证书申请流程结束")
	global.LOG.Infof("Certificate applied successfully for: %s", cert.PrimaryDomain)
	return nil
}

func (s *CertificateService) Renew(id uint) error {
	cert, err := s.certRepo.Get(repo.WithByID(id))
	if err != nil {
		return buserr.New(constant.ErrRecordNotFound)
	}
	if cert.Type == "upload" {
		return fmt.Errorf("uploaded certificates cannot be renewed")
	}

	logger, logFile := s.openSSLLog(cert)
	if logFile != nil {
		defer logFile.Close()
	}

	logger.Printf("[开始] 续签证书: %s", cert.PrimaryDomain)
	s.certRepo.Update(id, map[string]interface{}{"status": "applying", "message": ""})

	acme, err := s.acmeRepo.Get(repo.WithByID(cert.AcmeAccountID))
	if err != nil {
		logger.Printf("[错误] ACME 账户不存在")
		s.certRepo.Update(id, map[string]interface{}{"status": "error", "message": "ACME 账户不存在"})
		return fmt.Errorf("ACME account not found")
	}
	logger.Printf("[信息] ACME 账户: %s (%s)", acme.Email, acme.Type)

	client, err := sslutil.NewAcmeClient(acme.Email, acme.PrivateKey, acme.KeyType, acme.Type, acme.CaDirURL, acme.URL)
	if err != nil {
		logger.Printf("[错误] ACME 客户端创建失败: %v", err)
		s.certRepo.Update(id, map[string]interface{}{"status": "error", "message": err.Error()})
		return err
	}

	if cert.Provider == "dns" {
		dns, err := s.dnsRepo.Get(repo.WithByID(cert.DnsAccountID))
		if err != nil {
			logger.Printf("[错误] DNS 账户不存在")
			s.certRepo.Update(id, map[string]interface{}{"status": "error", "message": "DNS 账户不存在"})
			return fmt.Errorf("DNS account not found")
		}
		logger.Printf("[信息] DNS 账户: %s (%s)", dns.Name, dns.Type)
		if err := client.SetDNSProvider(dns.Type, dns.Authorization); err != nil {
			logger.Printf("[错误] DNS 提供商配置失败: %v", err)
			s.certRepo.Update(id, map[string]interface{}{"status": "error", "message": err.Error()})
			return err
		}
	}

	// 收集域名
	renewDomains := []string{cert.PrimaryDomain}
	if cert.Domains != "" {
		renewDomains = append(renewDomains, strings.Split(cert.Domains, ",")...)
	}
	logger.Printf("[信息] 续签域名: %s", strings.Join(renewDomains, ", "))
	logger.Printf("[信息] 正在向 CA 发起续签请求...")

	renewed, err := client.RenewCertificate(renewDomains, cert.KeyType)
	if err != nil {
		logger.Printf("[错误] 续签失败: %v", err)
		s.certRepo.Update(id, map[string]interface{}{"status": "error", "message": "续签失败: " + err.Error()})
		return err
	}
	logger.Printf("[成功] 证书续签成功!")

	certInfo, _ := parseCertPEM(string(renewed.Certificate))
	updates := map[string]interface{}{
		"pem":         string(renewed.Certificate),
		"private_key": string(renewed.PrivateKey),
		"cert_url":    renewed.CertURL,
		"status":      "applied",
		"message":     "",
	}
	if certInfo != nil {
		updates["expire_date"] = certInfo.expireDate
		updates["start_date"] = certInfo.startDate
		logger.Printf("[信息] 新证书有效期: %s 至 %s", certInfo.startDate.Format("2006-01-02"), certInfo.expireDate.Format("2006-01-02"))
	}
	s.certRepo.Update(id, updates)

	cert.Pem = string(renewed.Certificate)
	cert.PrivateKey = string(renewed.PrivateKey)
	if err := s.saveCertFiles(cert); err != nil {
		logger.Printf("[警告] 证书文件保存失败: %v", err)
	} else {
		logger.Printf("[成功] 证书文件已更新")
	}

	logger.Printf("[完成] 证书续签流程结束")
	global.LOG.Infof("Certificate renewed for: %s", cert.PrimaryDomain)
	return nil
}

func (s *CertificateService) GetSSLDir() string {
	dir, err := s.settingRepo.GetValueByKey("SSLDir")
	if err != nil || dir == "" {
		// 默认使用安装路径下的 ssl 目录
		execPath, _ := os.Executable()
		return filepath.Join(filepath.Dir(execPath), "ssl")
	}
	return dir
}

func (s *CertificateService) UpdateSSLDir(dir string) error {
	// 创建目录
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %v", err)
	}
	return s.settingRepo.Update("SSLDir", dir)
}

func (s *CertificateService) saveCertFiles(cert model.Certificate) error {
	sslDir := s.GetSSLDir()
	certDir := filepath.Join(sslDir, "certs", cert.PrimaryDomain)
	if err := os.MkdirAll(certDir, 0755); err != nil {
		return err
	}

	// 保存证书
	certPath := filepath.Join(certDir, "fullchain.pem")
	if err := os.WriteFile(certPath, []byte(cert.Pem), 0644); err != nil {
		return err
	}

	// 保存私钥
	keyPath := filepath.Join(certDir, "privkey.pem")
	if err := os.WriteFile(keyPath, []byte(cert.PrivateKey), 0600); err != nil {
		return err
	}

	global.LOG.Infof("Certificate files saved to: %s", certDir)
	return nil
}

// getSSLLogDir 获取 SSL 日志目录
func (s *CertificateService) getSSLLogDir() string {
	sslDir := s.GetSSLDir()
	logDir := filepath.Join(sslDir, "logs")
	os.MkdirAll(logDir, 0755)
	return logDir
}

// getSSLLogPath 获取证书日志路径
func (s *CertificateService) getSSLLogPath(cert model.Certificate) string {
	return filepath.Join(s.getSSLLogDir(), fmt.Sprintf("%s-ssl-%d.log", cert.PrimaryDomain, cert.ID))
}

// openSSLLog 创建/打开证书日志文件，返回 logger
func (s *CertificateService) openSSLLog(cert model.Certificate) (*log.Logger, *os.File) {
	logPath := s.getSSLLogPath(cert)
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		global.LOG.Warnf("Failed to open SSL log file: %v", err)
		return log.New(os.Stdout, "[SSL] ", log.LstdFlags), nil
	}
	return log.New(logFile, "", log.LstdFlags), logFile
}

// GetLog 读取证书日志
func (s *CertificateService) GetLog(id uint) (string, error) {
	cert, err := s.certRepo.Get(repo.WithByID(id))
	if err != nil {
		return "", buserr.New(constant.ErrRecordNotFound)
	}
	logPath := s.getSSLLogPath(cert)
	data, err := os.ReadFile(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "暂无日志记录", nil
		}
		return "", err
	}
	return string(data), nil
}

type certParsed struct {
	primaryDomain string
	domains       []string
	expireDate    time.Time
	startDate     time.Time
}

func parseCertPEM(pemStr string) (*certParsed, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, fmt.Errorf("invalid PEM")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}

	var domains []string
	if cert.Subject.CommonName != "" {
		domains = append(domains, cert.Subject.CommonName)
	}
	for _, name := range cert.DNSNames {
		if name != cert.Subject.CommonName {
			domains = append(domains, name)
		}
	}

	primaryDomain := cert.Subject.CommonName
	if primaryDomain == "" && len(cert.DNSNames) > 0 {
		primaryDomain = cert.DNSNames[0]
	}

	return &certParsed{
		primaryDomain: primaryDomain,
		domains:       domains,
		expireDate:    cert.NotAfter,
		startDate:     cert.NotBefore,
	}, nil
}
