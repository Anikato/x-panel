package service

import (
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-acme/lego/v4/certificate"
	"github.com/google/uuid"
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
	// ResolveCertFilePaths 解析证书管理中某条记录在磁盘上的 fullchain/privkey 路径并校验可读与密钥匹配
	ResolveCertFilePaths(certID uint) (certPath, keyPath string, err error)
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
		LineageUID:    uuid.NewString(),
		PrimaryDomain: req.PrimaryDomain,
		Domains:       req.OtherDomains,
		Provider:      req.Provider,
		Type:          "autoApply",
		AcmeAccountID: req.AcmeAccountID,
		DnsAccountID:  req.DnsAccountID,
		WebsiteID:     req.WebsiteID,
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
	if _, err := tls.X509KeyPair([]byte(req.Certificate), []byte(req.PrivateKey)); err != nil {
		return buserr.WithDetail(constant.ErrInvalidParams, "证书和私钥不匹配: "+err.Error(), err)
	}

	// 解析证书获取域名和过期时间
	certInfo, err := parseCertPEM(req.Certificate)
	if err != nil {
		return buserr.WithDetail(constant.ErrInvalidParams, "证书解析失败: "+err.Error(), err)
	}

	cert := model.Certificate{
		LineageUID:    uuid.NewString(),
		PrimaryDomain: certInfo.primaryDomain,
		Domains:       strings.Join(otherDomains(certInfo.primaryDomain, certInfo.domains), ","),
		Provider:      "manual",
		Type:          "upload",
		Pem:           req.Certificate,
		PrivateKey:    req.PrivateKey,
		ExpireDate:    certInfo.expireDate,
		StartDate:     certInfo.startDate,
		Status:        "applied",
		Description:   req.Description,
		KeyType:       "upload",
		SourceType:    "upload",
	}
	applyParsedMetadata(&cert, certInfo)
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
	// 只删除该记录拥有的 ID 目录。旧域名目录可能仍被其他记录或服务配置引用。
	sslDir := s.GetSSLDir()
	os.RemoveAll(certDirPath(sslDir, cert))

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
		info := certificateToInfo(c)
		info.AcmeAccountEmail = acmeMap[c.AcmeAccountID]
		info.DnsAccountName = dnsMap[c.DnsAccountID]
		items = append(items, info)
	}
	return total, items, nil
}

func (s *CertificateService) GetDetail(id uint) (*dto.CertificateDetail, error) {
	cert, err := s.certRepo.Get(repo.WithByID(id))
	if err != nil {
		return nil, buserr.New(constant.ErrRecordNotFound)
	}

	sslDir := s.GetSSLDir()
	certPath, _ := existingCertFilePaths(sslDir, cert)
	certPath = filepath.Dir(certPath)

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
		CertificateInfo: func() dto.CertificateInfo {
			info := certificateToInfo(cert)
			info.AcmeAccountEmail = acmeEmail
			info.DnsAccountName = dnsName
			return info
		}(),
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

	// 收集域名
	domains := collectCertificateDomains(cert)
	if err := validateCertificateProvider(cert.Provider, domains); err != nil {
		logger.Printf("[错误] %s", err.Error())
		s.certRepo.Update(id, map[string]interface{}{"status": "error", "message": err.Error()})
		return err
	}

	switch cert.Provider {
	case "dns":
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
	case "http":
		logger.Printf("[信息] 正在配置 HTTP-01 验证提供商...")
		if err := s.prepareHTTP01Website(cert); err != nil {
			logger.Printf("[错误] HTTP-01 关联网站准备失败: %v", err)
			s.certRepo.Update(id, map[string]interface{}{"status": "error", "message": err.Error()})
			return err
		}
		if err := client.SetHTTPProvider(); err != nil {
			logger.Printf("[错误] HTTP-01 验证提供商配置失败: %v", err)
			s.certRepo.Update(id, map[string]interface{}{"status": "error", "message": err.Error()})
			return err
		}
		logger.Printf("[成功] HTTP-01 验证提供商配置完成")
	}
	logger.Printf("[信息] 申请域名: %s", strings.Join(domains, ", "))
	logger.Printf("[信息] 密钥类型: %s", cert.KeyType)

	global.LOG.Infof("Applying certificate for domains: %v", domains)

	// 申请证书
	logger.Printf("[信息] 正在向 CA 发起证书申请（此步骤可能耗时数分钟）...")
	if cert.Provider == "dns" {
		logger.Printf("[信息] DNS 验证中：创建 TXT 记录并等待 CA 验证（已跳过传播检查）...")
	} else if cert.Provider == "http" {
		logger.Printf("[信息] HTTP 验证中：请确保域名 80 端口可访问面板的 /.well-known/acme-challenge/ 路径...")
	}
	var logWriter *os.File
	if logFile != nil {
		logWriter = logFile
	}
	certRes, err := client.ObtainCertificate(domains, cert.KeyType, logWriter)
	if err != nil {
		logger.Printf("[错误] 证书申请失败: %v", err)
		s.certRepo.Update(id, map[string]interface{}{"status": "error", "message": err.Error()})
		return err
	}
	logger.Printf("[成功] 证书申请成功!")

	// 解析证书信息
	certInfo, _ := parseCertPEM(string(certRes.Certificate))

	cert.Pem = string(certRes.Certificate)
	cert.PrivateKey = string(certRes.PrivateKey)

	// 先写入文件，再更新 DB 状态，避免文件失败时 DB 显示“已签发”但磁盘无文件
	if err := s.saveCertFiles(cert); err != nil {
		logger.Printf("[错误] 证书文件保存失败，回滚状态为 error: %v", err)
		s.certRepo.Update(id, map[string]interface{}{"status": "error", "message": "证书文件写入失败: " + err.Error()})
		return err
	}
	sslDir := s.GetSSLDir()
	logger.Printf("[成功] 证书文件已保存到: %s", certDirPath(sslDir, cert))

	// File written — now write DB with all cert fields
	dbUpdates := map[string]interface{}{
		"pem":         cert.Pem,
		"private_key": cert.PrivateKey,
		"cert_url":    certRes.CertURL,
		"status":      "applied",
		"message":     "",
		"source_type": "acme",
	}
	if certInfo != nil {
		dbUpdates["expire_date"] = certInfo.expireDate
		dbUpdates["start_date"] = certInfo.startDate
		addParsedMetadataUpdates(dbUpdates, certInfo)
	}
	s.certRepo.Update(id, dbUpdates)
	cert.ID = id

	if cert.WebsiteID > 0 {
		logger.Printf("[信息] 正在绑定证书到关联网站 (ID=%d)...", cert.WebsiteID)
		if err := s.bindCertificateToWebsite(cert); err != nil {
			logger.Printf("[警告] 证书已签发，但自动绑定网站失败: %v", err)
			s.certRepo.Update(id, map[string]interface{}{"message": "证书已签发，但自动绑定网站失败: " + err.Error()})
		} else {
			logger.Printf("[成功] 证书已绑定到关联网站")
		}
	}

	// 如果有网站正在使用此证书，自动 reload nginx
	if global.CONF.Nginx.IsInstalled() {
		logger.Printf("[信息] 正在重载 Nginx 配置...")
		if err := reloadNginxGlobal(); err != nil {
			logger.Printf("[警告] Nginx 重载失败: %v", err)
		} else {
			logger.Printf("[成功] Nginx 已重载")
		}
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
	if !isRenewableCertificate(cert) {
		return fmt.Errorf("certificate cannot be renewed locally")
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

	renewDomains := collectCertificateDomains(cert)
	if err := validateCertificateProvider(cert.Provider, renewDomains); err != nil {
		logger.Printf("[错误] %s", err.Error())
		s.certRepo.Update(id, map[string]interface{}{"status": "error", "message": err.Error()})
		return err
	}

	switch cert.Provider {
	case "dns":
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
	case "http":
		logger.Printf("[信息] 正在配置 HTTP-01 验证提供商...")
		if err := s.prepareHTTP01Website(cert); err != nil {
			logger.Printf("[错误] HTTP-01 关联网站准备失败: %v", err)
			s.certRepo.Update(id, map[string]interface{}{"status": "error", "message": err.Error()})
			return err
		}
		if err := client.SetHTTPProvider(); err != nil {
			logger.Printf("[错误] HTTP-01 验证提供商配置失败: %v", err)
			s.certRepo.Update(id, map[string]interface{}{"status": "error", "message": err.Error()})
			return err
		}
	}

	// 收集域名
	logger.Printf("[信息] 续签域名: %s", strings.Join(renewDomains, ", "))
	logger.Printf("[信息] 正在向 CA 发起续签请求...")
	if cert.Provider == "dns" {
		logger.Printf("[信息] DNS 验证中：创建 TXT 记录并等待 CA 验证（已跳过传播检查）...")
	} else if cert.Provider == "http" {
		logger.Printf("[信息] HTTP 验证中：请确保域名 80 端口可访问 /.well-known/acme-challenge/ 路径...")
	}

	var renewLogWriter *os.File
	if logFile != nil {
		renewLogWriter = logFile
	}

	// Build existingCert resource to allow lego native Renew API (saves rate-limit quota)
	var existingCertRes *certificate.Resource
	if cert.CertURL != "" {
		existingCertRes = &certificate.Resource{
			CertURL:     cert.CertURL,
			Domain:      cert.PrimaryDomain,
			Certificate: []byte(cert.Pem),
			PrivateKey:  []byte(cert.PrivateKey),
		}
	}
	renewed, err := client.RenewCertificate(renewDomains, cert.KeyType, existingCertRes, renewLogWriter)
	if err != nil {
		logger.Printf("[ERROR] Renewal failed: %v", err)
		s.certRepo.Update(id, map[string]interface{}{"status": "error", "message": "renewal failed: " + err.Error()})
		return err
	}
	logger.Printf("[OK] Certificate renewed successfully!")

	certInfo, _ := parseCertPEM(string(renewed.Certificate))
	if certInfo != nil {
		logger.Printf("[INFO] New cert validity: %s to %s", certInfo.startDate.Format("2006-01-02"), certInfo.expireDate.Format("2006-01-02"))
	}

	cert.Pem = string(renewed.Certificate)
	cert.PrivateKey = string(renewed.PrivateKey)

	// Write files first, then update DB — prevents DB showing "applied" while disk has no cert
	if err := s.saveCertFiles(cert); err != nil {
		logger.Printf("[ERROR] Cert file save failed, rolling back status: %v", err)
		s.certRepo.Update(id, map[string]interface{}{"status": "error", "message": "cert file write failed: " + err.Error()})
		return err
	}
	logger.Printf("[OK] Certificate files updated on disk")

	renewUpdates := map[string]interface{}{
		"pem":         cert.Pem,
		"private_key": cert.PrivateKey,
		"cert_url":    renewed.CertURL,
		"status":      "applied",
		"message":     "",
	}
	if certInfo != nil {
		renewUpdates["expire_date"] = certInfo.expireDate
		renewUpdates["start_date"] = certInfo.startDate
		addParsedMetadataUpdates(renewUpdates, certInfo)
	}
	s.certRepo.Update(id, renewUpdates)

	// 自动 reload nginx 使新证书生效
	if global.CONF.Nginx.IsInstalled() {
		logger.Printf("[信息] 正在重载 Nginx 配置...")
		if err := reloadNginxGlobal(); err != nil {
			logger.Printf("[警告] Nginx 重载失败: %v", err)
		} else {
			logger.Printf("[成功] Nginx 已重载，新证书已生效")
		}
	}

	logger.Printf("[完成] 证书续签流程结束")
	global.LOG.Infof("Certificate renewed for: %s", cert.PrimaryDomain)
	return nil
}

func collectCertificateDomains(cert model.Certificate) []string {
	domains := []string{strings.TrimSpace(cert.PrimaryDomain)}
	if cert.Domains != "" {
		for _, domain := range strings.Split(cert.Domains, ",") {
			domain = strings.TrimSpace(domain)
			if domain != "" {
				domains = append(domains, domain)
			}
		}
	}
	return domains
}

func validateCertificateProvider(provider string, domains []string) error {
	switch provider {
	case "dns", "http":
	default:
		return fmt.Errorf("自动验证方式“%s”暂不支持", provider)
	}
	if provider == "http" {
		for _, domain := range domains {
			if strings.HasPrefix(strings.TrimSpace(domain), "*.") {
				return fmt.Errorf("HTTP 验证不支持通配符域名，请改用 DNS 验证")
			}
		}
	}
	return nil
}

func (s *CertificateService) bindCertificateToWebsite(cert model.Certificate) error {
	if cert.WebsiteID == 0 {
		return nil
	}
	websiteRepo := repo.NewIWebsiteRepo()
	site, err := websiteRepo.Get(repo.WithByID(cert.WebsiteID))
	if err != nil {
		return buserr.New(constant.ErrRecordNotFound)
	}
	site.SSLEnable = true
	site.CertificateID = cert.ID
	if site.HttpConfig == "" || site.HttpConfig == "httpOnly" {
		site.HttpConfig = "HTTPSRedirect"
	}
	if site.SSLProtocols == "" {
		site.SSLProtocols = "TLSv1.2 TLSv1.3"
	}
	if err := websiteRepo.Save(&site); err != nil {
		return err
	}
	if site.Status == "running" && site.ConfigMode != "source" {
		websiteSvc := &WebsiteService{
			websiteRepo: websiteRepo,
			certRepo:    s.certRepo,
		}
		return websiteSvc.applyConfig(site)
	}
	return nil
}

func (s *CertificateService) prepareHTTP01Website(cert model.Certificate) error {
	if cert.WebsiteID == 0 {
		return fmt.Errorf("HTTP 验证需要选择关联网站")
	}
	if !global.CONF.Nginx.IsInstalled() {
		return buserr.New(constant.ErrNginxNotInstalled)
	}
	websiteRepo := repo.NewIWebsiteRepo()
	site, err := websiteRepo.Get(repo.WithByID(cert.WebsiteID))
	if err != nil {
		return buserr.New(constant.ErrRecordNotFound)
	}
	if site.ConfigMode == "source" {
		return fmt.Errorf("HTTP 验证暂不支持源码模式网站，请改用托管模式或 DNS 验证")
	}
	if err := validateHTTP01WebsiteDomains(cert, site); err != nil {
		return err
	}
	websiteSvc := &WebsiteService{
		websiteRepo: websiteRepo,
		certRepo:    s.certRepo,
	}
	if site.Status != "running" {
		if err := websiteSvc.autoEnable(&site); err != nil {
			return err
		}
		return nil
	}
	return websiteSvc.applyConfig(site)
}

func validateHTTP01WebsiteDomains(cert model.Certificate, site model.Website) error {
	siteDomains := make(map[string]struct{})
	addDomain := func(domain string) {
		domain = normalizeCertDomain(domain)
		if domain != "" {
			siteDomains[domain] = struct{}{}
		}
	}
	addDomain(site.PrimaryDomain)
	for _, domain := range strings.Split(site.Domains, ",") {
		addDomain(domain)
	}

	for _, domain := range collectCertificateDomains(cert) {
		normalized := normalizeCertDomain(domain)
		if normalized == "" {
			continue
		}
		if _, ok := siteDomains[normalized]; !ok {
			return fmt.Errorf("HTTP 验证域名 %s 不在关联网站域名中，请先把域名加入网站或选择对应网站", domain)
		}
	}
	return nil
}

func normalizeCertDomain(domain string) string {
	return strings.TrimSuffix(strings.ToLower(strings.TrimSpace(domain)), ".")
}

func (s *CertificateService) GetSSLDir() string {
	dir, err := s.settingRepo.GetValueByKey("SSLDir")
	if err != nil || dir == "" {
		return global.CONF.GetDefaultSSLDir()
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

// ResolveCertFilePaths 返回证书管理落盘的 PEM 路径，并校验文件存在及证书私钥匹配
func (s *CertificateService) ResolveCertFilePaths(certID uint) (string, string, error) {
	if certID == 0 {
		return "", "", buserr.New(constant.ErrInvalidParams)
	}
	cert, err := s.certRepo.Get(repo.WithByID(certID))
	if err != nil {
		return "", "", buserr.New(constant.ErrRecordNotFound)
	}
	// 与证书管理一致：ACME/上传成功后为 applied；新建未申请为 ready；证书同步写入为 applied
	if cert.Status != "ready" && cert.Status != "applied" {
		return "", "", buserr.New(constant.ErrPanelSSLCertNotReady)
	}
	sslDir := s.GetSSLDir()
	certPath, keyPath := existingCertFilePaths(sslDir, cert)
	if _, err := os.Stat(certPath); err != nil {
		return "", "", buserr.WithDetail(constant.ErrPanelSSLCertFiles, certPath, err)
	}
	if _, err := os.Stat(keyPath); err != nil {
		return "", "", buserr.WithDetail(constant.ErrPanelSSLCertFiles, keyPath, err)
	}
	if _, err := tls.LoadX509KeyPair(certPath, keyPath); err != nil {
		return "", "", buserr.WithDetail(constant.ErrPanelSSLKeyPairInvalid, err.Error(), err)
	}
	return certPath, keyPath, nil
}

// safeDomainDir converts a domain name into a filesystem-safe directory name.
// Wildcard certs like "*.example.com" become "_wildcard.example.com".
func safeDomainDir(domain string) string {
	return strings.ReplaceAll(domain, "*", "_wildcard")
}

func certDirName(cert model.Certificate) string {
	if cert.ID > 0 {
		return fmt.Sprintf("cert-%d", cert.ID)
	}
	if cert.Fingerprint != "" && len(cert.Fingerprint) >= 8 {
		return fmt.Sprintf("%s-%s", safeDomainDir(cert.PrimaryDomain), strings.ToLower(cert.Fingerprint[:8]))
	}
	return safeDomainDir(cert.PrimaryDomain)
}

func certDirPath(sslDir string, cert model.Certificate) string {
	return filepath.Join(sslDir, "certs", certDirName(cert))
}

func certFilePaths(sslDir string, cert model.Certificate) (string, string) {
	certDir := certDirPath(sslDir, cert)
	return filepath.Join(certDir, "fullchain.pem"), filepath.Join(certDir, "privkey.pem")
}

func existingCertFilePaths(sslDir string, cert model.Certificate) (string, string) {
	certPath, keyPath := certFilePaths(sslDir, cert)
	if _, err := os.Stat(certPath); err == nil {
		return certPath, keyPath
	}
	legacyDir := filepath.Join(sslDir, "certs", safeDomainDir(cert.PrimaryDomain))
	return filepath.Join(legacyDir, "fullchain.pem"), filepath.Join(legacyDir, "privkey.pem")
}

func (s *CertificateService) saveCertFiles(cert model.Certificate) error {
	sslDir := s.GetSSLDir()
	certDir := certDirPath(sslDir, cert)
	if err := os.MkdirAll(certDir, 0755); err != nil {
		return fmt.Errorf("create cert dir %s: %w", certDir, err)
	}

	certPath, keyPath := certFilePaths(sslDir, cert)
	if err := os.WriteFile(certPath, []byte(cert.Pem), 0644); err != nil {
		return fmt.Errorf("write fullchain.pem: %w", err)
	}

	if err := os.WriteFile(keyPath, []byte(cert.PrivateKey), 0600); err != nil {
		return fmt.Errorf("write privkey.pem: %w", err)
	}

	ensureCertPermissions(certDir, certPath, keyPath)

	global.LOG.Infof("Certificate files saved to: %s", certDir)
	return nil
}

// ensureCertPermissions 确保证书文件权限正确，Nginx 进程可读取
// Nginx master 以 root 运行，可以读取 root:root 0644/0600 的文件
// 额外修正：确保目录链路上的权限允许遍历
func ensureCertPermissions(certDir, certPath, keyPath string) {
	os.Chmod(certDir, 0755)
	os.Chmod(certPath, 0644)
	os.Chmod(keyPath, 0600)
	// 确保上级目录可遍历
	parent := filepath.Dir(certDir)
	os.Chmod(parent, 0755)
	grandparent := filepath.Dir(parent)
	os.Chmod(grandparent, 0755)
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
	return filepath.Join(s.getSSLLogDir(), fmt.Sprintf("%s-ssl-%d.log", safeDomainDir(cert.PrimaryDomain), cert.ID))
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
	issuer        string
	subject       string
	serialNumber  string
	fingerprint   string
	dnsNames      []string
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

	dnsNames := append([]string(nil), cert.DNSNames...)
	var domains []string
	domains = append(domains, dnsNames...)
	if len(domains) == 0 && cert.Subject.CommonName != "" {
		domains = append(domains, cert.Subject.CommonName)
	}

	primaryDomain := ""
	if len(dnsNames) > 0 {
		primaryDomain = dnsNames[0]
	} else {
		primaryDomain = cert.Subject.CommonName
	}
	sum := sha256.Sum256(cert.Raw)

	return &certParsed{
		primaryDomain: primaryDomain,
		domains:       domains,
		issuer:        cert.Issuer.String(),
		subject:       cert.Subject.String(),
		serialNumber:  cert.SerialNumber.String(),
		fingerprint:   strings.ToUpper(hex.EncodeToString(sum[:])),
		dnsNames:      dnsNames,
		expireDate:    cert.NotAfter,
		startDate:     cert.NotBefore,
	}, nil
}

func applyParsedMetadata(cert *model.Certificate, parsed *certParsed) {
	if parsed == nil {
		return
	}
	cert.Issuer = parsed.issuer
	cert.Subject = parsed.subject
	cert.SerialNumber = parsed.serialNumber
	cert.Fingerprint = parsed.fingerprint
	cert.DNSNames = encodeStringList(parsed.dnsNames)
	cert.ExpireDate = parsed.expireDate
	cert.StartDate = parsed.startDate
	if cert.PrimaryDomain == "" {
		cert.PrimaryDomain = parsed.primaryDomain
	}
}

func addParsedMetadataUpdates(updates map[string]interface{}, parsed *certParsed) {
	updates["issuer"] = parsed.issuer
	updates["subject"] = parsed.subject
	updates["serial_number"] = parsed.serialNumber
	updates["fingerprint"] = parsed.fingerprint
	updates["dns_names"] = encodeStringList(parsed.dnsNames)
	if parsed.primaryDomain != "" {
		updates["primary_domain"] = parsed.primaryDomain
		updates["domains"] = strings.Join(otherDomains(parsed.primaryDomain, parsed.domains), ",")
	}
}

func encodeStringList(items []string) string {
	data, err := json.Marshal(items)
	if err != nil {
		return ""
	}
	return string(data)
}

func otherDomains(primary string, domains []string) []string {
	seen := map[string]struct{}{normalizeCertDomain(primary): {}}
	var result []string
	for _, domain := range domains {
		normalized := normalizeCertDomain(domain)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		result = append(result, strings.TrimSpace(domain))
	}
	return result
}

func certificateToInfo(c model.Certificate) dto.CertificateInfo {
	return dto.CertificateInfo{
		ID:            c.ID,
		PrimaryDomain: c.PrimaryDomain,
		Domains:       c.Domains,
		Provider:      c.Provider,
		Type:          c.Type,
		AcmeAccountID: c.AcmeAccountID,
		DnsAccountID:  c.DnsAccountID,
		WebsiteID:     c.WebsiteID,
		KeyType:       c.KeyType,
		AutoRenew:     c.AutoRenew,
		ExpireDate:    c.ExpireDate,
		StartDate:     c.StartDate,
		Status:        c.Status,
		Message:       c.Message,
		Description:   c.Description,
		CertURL:       c.CertURL,
		Issuer:        c.Issuer,
		Subject:       c.Subject,
		SerialNumber:  c.SerialNumber,
		Fingerprint:   c.Fingerprint,
		DNSNames:      c.DNSNames,
		SourceType:    c.SourceType,
		SourceID:      c.SourceID,
		SourceName:    c.SourceName,
		NotBefore:     c.StartDate,
		NotAfter:      c.ExpireDate,
	}
}

// reloadNginxGlobal 全局 reload nginx（供证书申请/续期后调用）
func reloadNginxGlobal() error {
	nc := global.CONF.Nginx
	if !nc.IsInstalled() {
		return nil
	}
	pidPath := nc.GetPidPath()
	if _, err := os.Stat(pidPath); os.IsNotExist(err) {
		return nil
	}
	if nc.IsSystemMode() {
		_, err := execCmd("systemctl", "reload", "nginx")
		return err
	}
	_, err := execCmd(nc.GetBinary(), "-p", nc.InstallDir, "-s", "reload")
	return err
}

func execCmd(name string, args ...string) (string, error) {
	out, err := exec.Command(name, args...).CombinedOutput()
	return string(out), err
}

func isRenewableCertificate(cert model.Certificate) bool {
	return cert.Type != "upload" && cert.Type != "synced" && cert.SourceType != "synced"
}

// autoRenewMu 防止同一证书被并发续签
var autoRenewMu sync.Map

// AutoRenewCerts 自动续期即将过期的证书（由 cron 调用）
func AutoRenewCerts() {
	certService := NewICertificateService().(*CertificateService)
	certs, err := certService.certRepo.GetList()
	if err != nil {
		global.LOG.Warnf("[auto-renew] Failed to list certificates: %v", err)
		return
	}

	now := time.Now()
	renewBefore := 15 * 24 * time.Hour // 提前 15 天开始续期，给失败重试留充分余量

	var wg sync.WaitGroup
	sem := make(chan struct{}, 3) // 最多 3 个并发续签

	for _, cert := range certs {
		if !cert.AutoRenew || !isRenewableCertificate(cert) || cert.Status != "applied" {
			continue
		}
		if cert.ExpireDate.IsZero() || cert.ExpireDate.Sub(now) > renewBefore {
			continue
		}

		// 追加鼠备并发锁：同一证书 ID 同时只运行一个续签
		if _, loaded := autoRenewMu.LoadOrStore(cert.ID, struct{}{}); loaded {
			global.LOG.Infof("[auto-renew] Certificate %d (%s) is already being renewed, skipping", cert.ID, cert.PrimaryDomain)
			continue
		}

		wg.Add(1)
		sem <- struct{}{}
		go func(c model.Certificate) {
			defer wg.Done()
			defer func() { <-sem }()
			defer autoRenewMu.Delete(c.ID)

			global.LOG.Infof("[auto-renew] Certificate %s (ID=%d) expires at %s, renewing...",
				c.PrimaryDomain, c.ID, c.ExpireDate.Format("2006-01-02"))
			if err := certService.Renew(c.ID); err != nil {
				global.LOG.Errorf("[auto-renew] Failed to renew %s: %v", c.PrimaryDomain, err)
			} else {
				global.LOG.Infof("[auto-renew] Successfully renewed %s", c.PrimaryDomain)
			}
		}(cert)
	}

	wg.Wait()
}
