package service

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"xpanel/app/dto"
	"xpanel/app/model"
	"xpanel/app/repo"
	"xpanel/buserr"
	"xpanel/constant"
	"xpanel/global"
	"xpanel/utils/cmd"
)

type IWebsiteService interface {
	Create(req dto.WebsiteCreate) error
	InspectExternalNginxSite(path string) (*dto.ExternalNginxSitePreview, error)
	CreateExternalNginxSite(req dto.ExternalNginxSiteCreateReq) (*model.Website, error)
	RefreshExternalNginxSite(id uint) (*model.Website, error)
	CheckWebsiteCertificateHealth(id uint) (*dto.WebsiteCertificateHealthResp, error)
	CheckWebsiteCertificateHealthBatch(req dto.WebsiteCertificateHealthBatchReq) ([]dto.WebsiteCertificateHealthResp, error)
	Update(req dto.WebsiteUpdate) error
	Delete(id uint) error
	SearchWithPage(req dto.WebsiteSearch) (int64, []dto.WebsiteInfo, error)
	GetDetail(id uint) (*dto.WebsiteDetail, error)
	Enable(id uint) error
	Disable(id uint) error
	GetNginxConfig(id uint) (string, error)
	GetSiteLog(req dto.WebsiteLogReq) (string, error)

	// Source-mode config editing
	GetSiteConfContent(id uint) (*dto.SiteConfContentResp, error)
	SaveSiteConfContent(req dto.SaveSiteConfReq) error
	SwitchConfigMode(id uint, mode string) error

	// Nginx config file management
	GetMainConf() (string, error)
	SaveMainConf(content string) error
	ListConfFiles() ([]dto.NginxConfFileInfo, error)
	GetConfFile(name string) (string, error)
	SaveConfFile(req dto.NginxConfUpdate) error
	ListConfBackups(filePath string) ([]dto.NginxConfBackupInfo, error)
	RestoreConfBackup(req dto.NginxConfRestoreReq) error

	// Low-cost diagnostics
	CheckHealth(id uint) (*dto.WebsiteHealthResp, error)
	InspectSite(id uint) (*dto.WebsiteInspectResp, error)
	DetectLogPaths(id uint) (*dto.WebsiteLogPathDetectResp, error)
	GetLogAlerts(req dto.WebsiteLogAlertReq) ([]dto.WebsiteLogAlert, error)
}

type WebsiteService struct {
	websiteRepo          repo.IWebsiteRepo
	certRepo             repo.ICertificateRepo
	sourceSaveOps        *sourceConfigSaveOps
	certificateHealthOps *certificateHealthOps
}

type sourceConfigSaveOps struct {
	backup      func(string) error
	writeAtomic func(string, []byte, os.FileMode) error
	nginxTest   func() error
	reload      func() error
	syncSite    func(*model.Website, nginxSiteMetadata) error
}

var sourceConfigPathLocks sync.Map

func NewIWebsiteService() IWebsiteService {
	return &WebsiteService{
		websiteRepo: repo.NewIWebsiteRepo(),
		certRepo:    repo.NewICertificateRepo(),
	}
}

func (s *WebsiteService) Create(req dto.WebsiteCreate) error {
	exist, _ := s.websiteRepo.Get(repo.WithByPrimaryDomain(req.PrimaryDomain))
	if exist.ID > 0 {
		return buserr.New(constant.ErrWebsiteDomainExist)
	}

	alias := strings.TrimSpace(req.Alias)
	if alias == "" {
		alias = domainToAlias(req.PrimaryDomain)
	}
	// 检查 alias 唯一性
	existAlias, _ := s.websiteRepo.Get(repo.WithByAlias(alias))
	if existAlias.ID > 0 {
		return buserr.WithDetail(constant.ErrRecordExist, "alias already exists: "+alias, nil)
	}

	site := model.Website{
		PrimaryDomain:     req.PrimaryDomain,
		Domains:           normalizeExtraDomains(req.PrimaryDomain, req.Domains),
		Alias:             alias,
		Type:              req.Type,
		Status:            "stopped",
		Remark:            req.Remark,
		SiteDir:           req.SiteDir,
		ProxyPass:         req.ProxyPass,
		ConfigMode:        req.ConfigMode,
		AccessLogPath:     strings.TrimSpace(req.AccessLogPath),
		ErrorLogPath:      strings.TrimSpace(req.ErrorLogPath),
		HttpPort:          req.HttpPort,
		HttpsPort:         req.HttpsPort,
		IndexFile:         "index.html index.htm",
		HttpConfig:        "HTTPSRedirect",
		SSLProtocols:      "TLSv1.2 TLSv1.3",
		AccessLog:         true,
		ErrorLog:          true,
		GzipEnable:        true,
		SecurityHeaders:   true,
		StaticCacheEnable: false,
	}
	if site.ConfigMode == "" {
		site.ConfigMode = "managed"
	}
	if site.ConfigMode == "source" {
		site.Status = "running"
	}

	if site.Type == "static" && site.SiteDir == "" {
		// 优先用 alias 作为目录名，更短且不含特殊字符
		site.SiteDir = fmt.Sprintf("/var/www/%s", alias)
	}

	if err := s.websiteRepo.Create(&site); err != nil {
		return buserr.WithDetail(constant.ErrInternalServer, err.Error(), err)
	}

	if site.ConfigMode != "source" && site.Type == "static" && site.SiteDir != "" {
		os.MkdirAll(site.SiteDir, 0755)
		indexPath := filepath.Join(site.SiteDir, "index.html")
		if _, err := os.Stat(indexPath); os.IsNotExist(err) {
			defaultHTML := fmt.Sprintf("<!DOCTYPE html>\n<html>\n<head><title>%s</title></head>\n<body>\n<h1>Welcome to %s</h1>\n<p>Site is working.</p>\n</body>\n</html>", site.PrimaryDomain, site.PrimaryDomain)
			os.WriteFile(indexPath, []byte(defaultHTML), 0644)
		}
	}

	global.LOG.Infof("Website created: %s (%s)", site.PrimaryDomain, site.Type)

	// Auto-enable if nginx is installed
	if site.ConfigMode != "source" && global.CONF.Nginx.IsInstalled() {
		if err := s.autoEnable(&site); err != nil {
			global.LOG.Warnf("Auto-enable website %s failed: %v", site.PrimaryDomain, err)
		}
	}

	return nil
}

func (s *WebsiteService) autoEnable(site *model.Website) error {
	if err := EnsureNginxInclude(); err != nil {
		global.LOG.Warnf("Ensure nginx include failed: %v", err)
	}
	if err := s.applyConfig(*site); err != nil {
		return err
	}
	site.Status = "running"
	return s.websiteRepo.Save(site)
}

func (s *WebsiteService) Update(req dto.WebsiteUpdate) error {
	site, err := s.websiteRepo.Get(repo.WithByID(req.ID))
	if err != nil {
		return buserr.New(constant.ErrRecordNotFound)
	}
	if site.NginxConfPath != "" {
		return buserr.New(constant.ErrWebsiteExternalOperationDenied)
	}

	// 如果域名变了，检查唯一性
	if req.PrimaryDomain != "" && req.PrimaryDomain != site.PrimaryDomain {
		exist, _ := s.websiteRepo.Get(repo.WithByPrimaryDomain(req.PrimaryDomain))
		if exist.ID > 0 && exist.ID != site.ID {
			return buserr.New(constant.ErrWebsiteDomainExist)
		}
		site.PrimaryDomain = req.PrimaryDomain
	}

	site.Domains = normalizeExtraDomains(site.PrimaryDomain, req.Domains)
	site.SiteDir = req.SiteDir
	site.IndexFile = req.IndexFile
	site.HttpPort = req.HttpPort
	site.HttpsPort = req.HttpsPort
	site.ProxyPass = req.ProxyPass
	site.WebSocket = req.WebSocket
	site.SSLEnable = req.SSLEnable
	site.CertificateID = req.CertificateID
	site.SkipCertAdapt = req.SkipCertAdapt
	site.HttpConfig = req.HttpConfig
	site.HSTS = req.HSTS
	site.Http2Enable = req.Http2Enable
	site.SSLProtocols = req.SSLProtocols
	site.BasicAuth = req.BasicAuth
	site.BasicUser = req.BasicUser
	site.AntiLeech = req.AntiLeech
	site.LeechReferers = req.LeechReferers
	site.LimitRate = req.LimitRate
	site.LimitConn = req.LimitConn
	site.Rewrite = req.Rewrite
	site.Redirects = req.Redirects
	site.AccessLog = req.AccessLog
	site.ErrorLog = req.ErrorLog
	site.AccessLogPath = strings.TrimSpace(req.AccessLogPath)
	site.ErrorLogPath = strings.TrimSpace(req.ErrorLogPath)
	site.GzipEnable = req.GzipEnable
	site.SecurityHeaders = req.SecurityHeaders
	site.StaticCacheEnable = req.StaticCacheEnable
	site.Upstream = req.Upstream
	site.CustomNginx = req.CustomNginx
	site.DefaultServer = req.DefaultServer
	site.Remark = req.Remark
	if req.ConfigMode != "" {
		site.ConfigMode = req.ConfigMode
	}

	// 处理 Basic Auth 密码
	if req.BasicAuth && req.BasicPassword != "" {
		site.BasicPassword = req.BasicPassword
	}
	if !req.BasicAuth {
		site.BasicPassword = ""
		site.BasicUser = ""
	}

	var rollback func() error
	if site.Status == "running" && site.ConfigMode != "source" {
		var applyErr error
		rollback, applyErr = s.applyConfigWithRollback(site)
		if applyErr != nil {
			global.LOG.Warnf("Auto apply config failed for %s: %v", site.PrimaryDomain, applyErr)
			return buserr.WithDetail(constant.ErrWebsiteApplyConfig, applyErr.Error(), applyErr)
		}
	}

	if err := s.websiteRepo.Save(&site); err != nil {
		if rollback != nil {
			if rbErr := rollback(); rbErr != nil {
				global.LOG.Errorf("Rollback website config after save failure: %v", rbErr)
			}
		}
		return buserr.WithDetail(constant.ErrInternalServer, err.Error(), err)
	}

	return nil
}

func (s *WebsiteService) Delete(id uint) error {
	site, err := s.websiteRepo.Get(repo.WithByID(id))
	if err != nil {
		return buserr.New(constant.ErrRecordNotFound)
	}
	if site.NginxConfPath != "" {
		return s.websiteRepo.Delete(repo.WithByID(id))
	}

	paths := websiteRuntimePaths(site, true)
	snaps, err := snapshotFiles(paths)
	if err != nil {
		return buserr.WithDetail(constant.ErrInternalServer, err.Error(), err)
	}
	if err := removeFiles(paths); err != nil {
		_ = restoreFiles(snaps)
		return buserr.WithDetail(constant.ErrInternalServer, err.Error(), err)
	}
	reload := siteServesTraffic(site)
	if reload {
		if err := s.reloadNginx(); err != nil {
			_ = restoreFiles(snaps)
			_ = s.reloadNginx()
			return buserr.WithDetail(constant.ErrInternalServer, err.Error(), err)
		}
	}
	if err := s.websiteRepo.Delete(repo.WithByID(id)); err != nil {
		_ = restoreFiles(snaps)
		if reload {
			_ = s.reloadNginx()
		}
		return buserr.WithDetail(constant.ErrInternalServer, err.Error(), err)
	}
	return nil
}

func (s *WebsiteService) SearchWithPage(req dto.WebsiteSearch) (int64, []dto.WebsiteInfo, error) {
	var opts []repo.DBOption
	if req.Info != "" {
		opts = append(opts, repo.WithLikeWebsite(req.Info))
	}
	if req.Type != "" {
		opts = append(opts, repo.WithByType(req.Type))
	}
	if req.Status != "" {
		opts = append(opts, repo.WithByStatus(req.Status))
	}

	total, sites, err := s.websiteRepo.Page(req.Page, req.PageSize, opts...)
	if err != nil {
		return 0, nil, err
	}

	var items []dto.WebsiteInfo
	for _, site := range sites {
		configActive, configIssues := s.externalConfigState(site)
		configuredCertificate := s.configuredCertificateSnapshot(site)
		items = append(items, dto.WebsiteInfo{
			ID:                    site.ID,
			PrimaryDomain:         site.PrimaryDomain,
			Domains:               site.Domains,
			Alias:                 site.Alias,
			Type:                  site.Type,
			Status:                site.Status,
			SSLEnable:             site.SSLEnable,
			ConfigMode:            site.ConfigMode,
			Remark:                site.Remark,
			SiteDir:               site.SiteDir,
			NginxConfPath:         site.NginxConfPath,
			ConfigActive:          configActive,
			ConfigIssues:          configIssues,
			ConfiguredCertificate: configuredCertificate,
			CreatedAt:             site.CreatedAt,
		})
	}
	return total, items, nil
}

func (s *WebsiteService) GetDetail(id uint) (*dto.WebsiteDetail, error) {
	site, err := s.websiteRepo.Get(repo.WithByID(id))
	if err != nil {
		return nil, buserr.New(constant.ErrRecordNotFound)
	}

	configActive, configIssues := s.externalConfigState(site)
	configuredCertificate := s.configuredCertificateSnapshot(site)
	detail := &dto.WebsiteDetail{
		ID:                    site.ID,
		PrimaryDomain:         site.PrimaryDomain,
		Domains:               site.Domains,
		Alias:                 site.Alias,
		Type:                  site.Type,
		Status:                site.Status,
		SiteDir:               site.SiteDir,
		IndexFile:             site.IndexFile,
		HttpPort:              site.HttpPort,
		HttpsPort:             site.HttpsPort,
		ProxyPass:             site.ProxyPass,
		WebSocket:             site.WebSocket,
		SSLEnable:             site.SSLEnable,
		CertificateID:         site.CertificateID,
		SkipCertAdapt:         site.SkipCertAdapt,
		HttpConfig:            site.HttpConfig,
		HSTS:                  site.HSTS,
		Http2Enable:           site.Http2Enable,
		SSLProtocols:          site.SSLProtocols,
		BasicAuth:             site.BasicAuth,
		BasicUser:             site.BasicUser,
		BasicPasswordSet:      site.BasicPassword != "",
		AntiLeech:             site.AntiLeech,
		LeechReferers:         site.LeechReferers,
		LimitRate:             site.LimitRate,
		LimitConn:             site.LimitConn,
		Rewrite:               site.Rewrite,
		Redirects:             site.Redirects,
		AccessLog:             site.AccessLog,
		ErrorLog:              site.ErrorLog,
		AccessLogPath:         site.AccessLogPath,
		ErrorLogPath:          site.ErrorLogPath,
		GzipEnable:            site.GzipEnable,
		SecurityHeaders:       site.SecurityHeaders,
		StaticCacheEnable:     site.StaticCacheEnable,
		Upstream:              site.Upstream,
		CustomNginx:           site.CustomNginx,
		DefaultServer:         site.DefaultServer,
		Remark:                site.Remark,
		ConfigMode:            site.ConfigMode,
		NginxConfPath:         site.NginxConfPath,
		ConfigActive:          configActive,
		ConfigIssues:          configIssues,
		ConfiguredCertificate: configuredCertificate,
	}
	redactWebsiteSecret(detail, site)

	if site.CertificateID > 0 {
		cert, err := s.certRepo.Get(repo.WithByID(site.CertificateID))
		if err == nil {
			detail.CertificateDomain = cert.PrimaryDomain
		}
	}

	if site.NginxConfPath != "" {
		if config, err := os.ReadFile(site.NginxConfPath); err == nil {
			detail.NginxConfig = string(config)
		}
	} else {
		gen := NewNginxConfigGenerator()
		config, _ := gen.Generate(site)
		detail.NginxConfig = config
	}

	return detail, nil
}

func (s *WebsiteService) externalConfigState(site model.Website) (bool, []string) {
	if site.NginxConfPath == "" {
		return false, nil
	}
	_, metadata, err := s.inspectExternalNginxSite(site.NginxConfPath)
	if err != nil {
		return false, []string{err.Error()}
	}
	return true, metadata.Warnings
}

func redactWebsiteSecret(detail *dto.WebsiteDetail, site model.Website) {
	detail.BasicPassword = ""
	detail.BasicPasswordSet = site.BasicPassword != ""
}

func (s *WebsiteService) Enable(id uint) error {
	site, err := s.websiteRepo.Get(repo.WithByID(id))
	if err != nil {
		return buserr.New(constant.ErrRecordNotFound)
	}
	if site.NginxConfPath != "" {
		return buserr.New(constant.ErrWebsiteExternalOperationDenied)
	}
	if !global.CONF.Nginx.IsInstalled() {
		return buserr.New(constant.ErrNginxNotInstalled)
	}
	if site.Status == "running" {
		return nil
	}

	// 确保 nginx.conf 包含 conf.d
	if err := EnsureNginxInclude(); err != nil {
		global.LOG.Warnf("Ensure nginx include failed: %v", err)
	}

	snaps, err := snapshotFiles(websiteRuntimePaths(site, true))
	if err != nil {
		return buserr.WithDetail(constant.ErrInternalServer, err.Error(), err)
	}
	if err := s.applyConfig(site); err != nil {
		return err
	}

	site.Status = "running"
	if err := s.websiteRepo.Save(&site); err != nil {
		_ = restoreFiles(snaps)
		_ = s.reloadNginx()
		return buserr.WithDetail(constant.ErrInternalServer, err.Error(), err)
	}
	return nil
}

func (s *WebsiteService) Disable(id uint) error {
	site, err := s.websiteRepo.Get(repo.WithByID(id))
	if err != nil {
		return buserr.New(constant.ErrRecordNotFound)
	}
	if site.NginxConfPath != "" {
		return buserr.New(constant.ErrWebsiteExternalOperationDenied)
	}
	if site.Status == "stopped" {
		return nil
	}

	paths := websiteRuntimePaths(site, false)
	snaps, err := snapshotFiles(paths)
	if err != nil {
		return buserr.WithDetail(constant.ErrInternalServer, err.Error(), err)
	}
	if err := removeFiles(paths); err != nil {
		_ = restoreFiles(snaps)
		return buserr.WithDetail(constant.ErrInternalServer, err.Error(), err)
	}
	if err := s.reloadNginx(); err != nil {
		_ = restoreFiles(snaps)
		_ = s.reloadNginx()
		return buserr.WithDetail(constant.ErrInternalServer, err.Error(), err)
	}

	site.Status = "stopped"
	if err := s.websiteRepo.Save(&site); err != nil {
		_ = restoreFiles(snaps)
		_ = s.reloadNginx()
		return buserr.WithDetail(constant.ErrInternalServer, err.Error(), err)
	}
	return nil
}

func (s *WebsiteService) GetNginxConfig(id uint) (string, error) {
	site, err := s.websiteRepo.Get(repo.WithByID(id))
	if err != nil {
		return "", buserr.New(constant.ErrRecordNotFound)
	}
	if site.NginxConfPath != "" {
		content, err := os.ReadFile(site.NginxConfPath)
		return string(content), err
	}
	gen := NewNginxConfigGenerator()
	return gen.Generate(site)
}

func (s *WebsiteService) GetSiteLog(req dto.WebsiteLogReq) (string, error) {
	site, err := s.websiteRepo.Get(repo.WithByID(req.ID))
	if err != nil {
		return "", buserr.New(constant.ErrRecordNotFound)
	}

	var logPath string
	switch req.Type {
	case "access":
		logPath = s.getWebsiteLogPath(site, "access")
	case "error":
		logPath = s.getWebsiteLogPath(site, "error")
	default:
		return "", fmt.Errorf("invalid log type")
	}

	tail := req.Tail
	if tail <= 0 {
		tail = 200
	}

	output, err := cmd.ExecWithOutput("tail", "-n", fmt.Sprintf("%d", tail), logPath)
	if err != nil {
		if strings.Contains(err.Error(), "No such file") {
			return "暂无日志", nil
		}
		return "", err
	}
	return output, nil
}

func (s *WebsiteService) getWebsiteLogPath(site model.Website, logType string) string {
	if logType == "error" && strings.TrimSpace(site.ErrorLogPath) != "" {
		return strings.TrimSpace(site.ErrorLogPath)
	}
	if logType == "access" && strings.TrimSpace(site.AccessLogPath) != "" {
		return strings.TrimSpace(site.AccessLogPath)
	}
	logDir := filepath.Join(global.CONF.Nginx.GetLogDir(), "sites")
	if logType == "error" {
		return filepath.Join(logDir, site.PrimaryDomain+".error.log")
	}
	return filepath.Join(logDir, site.PrimaryDomain+".access.log")
}

// --- 源码模式配置编辑 ---

func (s *WebsiteService) GetSiteConfContent(id uint) (*dto.SiteConfContentResp, error) {
	site, err := s.websiteRepo.Get(repo.WithByID(id))
	if err != nil {
		return nil, buserr.New(constant.ErrRecordNotFound)
	}
	confPath, err := s.resolveSiteConfPath(site)
	if err != nil {
		return nil, err
	}
	content, err := os.ReadFile(confPath)
	if os.IsNotExist(err) && site.NginxConfPath == "" {
		generated, generateErr := NewNginxConfigGenerator().Generate(site)
		if generateErr != nil {
			return nil, generateErr
		}
		content = []byte(generated)
		err = nil
	}
	if err != nil {
		return nil, err
	}
	return &dto.SiteConfContentResp{
		Path: confPath, Content: string(content), Hash: contentSHA256(content),
	}, nil
}

func (s *WebsiteService) SaveSiteConfContent(req dto.SaveSiteConfReq) error {
	site, err := s.websiteRepo.Get(repo.WithByID(req.ID))
	if err != nil {
		return buserr.New(constant.ErrRecordNotFound)
	}
	confPath, err := s.resolveSiteConfPath(site)
	if err != nil {
		return err
	}
	lockValue, _ := sourceConfigPathLocks.LoadOrStore(confPath, &sync.Mutex{})
	lock := lockValue.(*sync.Mutex)
	lock.Lock()
	defer lock.Unlock()

	if site.NginxConfPath != "" {
		currentPath, err := validateExternalNginxConfigPath(site.NginxConfPath, global.CONF.Nginx.GetMainConf())
		if err != nil {
			return err
		}
		if currentPath != confPath {
			return buserr.New(constant.ErrWebsiteExternalConfigConflict)
		}
	}

	original, existed, mode, err := s.currentSourceConfig(site, confPath)
	if err != nil {
		return err
	}
	if contentSHA256(original) != strings.TrimSpace(req.Hash) {
		return buserr.New(constant.ErrWebsiteExternalConfigConflict)
	}
	metadata, err := parseNginxSiteMetadata(req.Content)
	if err != nil {
		return buserr.WithDetail(constant.ErrWebsiteExternalConfigInvalid, confPath, err)
	}
	if err := s.ensureWebsitePrimaryDomainAvailable(metadata.PrimaryDomain, site.ID); err != nil {
		return err
	}
	return s.saveSourceConfigTransaction(&site, confPath, original, []byte(req.Content), existed, mode, metadata)
}

func (s *WebsiteService) resolveSiteConfPath(site model.Website) (string, error) {
	if site.NginxConfPath != "" {
		return validateExternalNginxConfigPath(site.NginxConfPath, global.CONF.Nginx.GetMainConf())
	}
	nc := global.CONF.Nginx
	if nc.IsSystemMode() {
		available := filepath.Join(nc.GetSitesAvailableDir(), site.Alias+".conf")
		if _, err := os.Stat(available); err == nil {
			return available, nil
		}
		enabled := filepath.Join(nc.GetSitesDir(), site.Alias+".conf")
		if _, err := os.Stat(enabled); err == nil {
			return enabled, nil
		}
		return available, nil
	}
	return GetSiteConfPath(site.Alias), nil
}

func (s *WebsiteService) currentSourceConfig(site model.Website, confPath string) ([]byte, bool, os.FileMode, error) {
	content, err := os.ReadFile(confPath)
	if err == nil {
		info, statErr := os.Stat(confPath)
		if statErr != nil {
			return nil, false, 0, statErr
		}
		return content, true, info.Mode().Perm(), nil
	}
	if !os.IsNotExist(err) || site.NginxConfPath != "" {
		return nil, false, 0, err
	}
	generated, err := NewNginxConfigGenerator().Generate(site)
	return []byte(generated), false, 0o644, err
}

func (s *WebsiteService) saveSourceConfigTransaction(
	site *model.Website,
	confPath string,
	original, replacement []byte,
	existed bool,
	mode os.FileMode,
	metadata nginxSiteMetadata,
) error {
	ops := s.effectiveSourceConfigSaveOps()
	if err := os.MkdirAll(filepath.Dir(confPath), 0o755); err != nil {
		return fmt.Errorf("创建配置目录 %s: %w", filepath.Dir(confPath), err)
	}
	if existed {
		if err := ops.backup(confPath); err != nil {
			return fmt.Errorf("备份配置 %s: %w", confPath, err)
		}
	}
	if err := ops.writeAtomic(confPath, replacement, mode); err != nil {
		return fmt.Errorf("写入配置 %s: %w", confPath, err)
	}
	rollback := func() error {
		if !existed {
			return os.Remove(confPath)
		}
		return ops.writeAtomic(confPath, original, mode)
	}
	if err := ops.nginxTest(); err != nil {
		rollbackErr := rollback()
		return buserr.WithDetail(constant.ErrNginxConfigTest, sourceSaveFailureDetail("test", confPath, err, rollbackErr), err)
	}

	reloaded := false
	if site.Status == "running" {
		if err := ops.reload(); err != nil {
			rollbackErr := rollback()
			recoveryErr := recoverSourceConfig(ops)
			return buserr.WithDetail(constant.ErrWebsiteApplyConfig, sourceSaveRecoveryDetail("reload", confPath, err, rollbackErr, recoveryErr), err)
		}
		reloaded = true
	}

	updated := *site
	applyNginxSiteMetadata(&updated, metadata)
	updated.ConfigMode = "source"
	updated.AccessLog = metadata.AccessLogPath != ""
	updated.ErrorLog = metadata.ErrorLogPath != ""
	updated.CertificateID = s.matchPanelCertificate(metadata.CertPath, metadata.KeyPath)
	if err := ops.syncSite(&updated, metadata); err != nil {
		rollbackErr := rollback()
		var recoveryErr error
		if reloaded {
			recoveryErr = recoverSourceConfig(ops)
		}
		return buserr.WithDetail(constant.ErrWebsiteApplyConfig, sourceSaveRecoveryDetail("sync", confPath, err, rollbackErr, recoveryErr), err)
	}
	return nil
}

func (s *WebsiteService) effectiveSourceConfigSaveOps() sourceConfigSaveOps {
	ops := sourceConfigSaveOps{
		backup:      s.createConfigBackup,
		writeAtomic: writeFileAtomic,
		nginxTest:   s.testNginxConfig,
		reload:      s.reloadNginx,
		syncSite: func(site *model.Website, _ nginxSiteMetadata) error {
			return s.websiteRepo.Save(site)
		},
	}
	if s.sourceSaveOps == nil {
		return ops
	}
	if s.sourceSaveOps.backup != nil {
		ops.backup = s.sourceSaveOps.backup
	}
	if s.sourceSaveOps.writeAtomic != nil {
		ops.writeAtomic = s.sourceSaveOps.writeAtomic
	}
	if s.sourceSaveOps.nginxTest != nil {
		ops.nginxTest = s.sourceSaveOps.nginxTest
	}
	if s.sourceSaveOps.reload != nil {
		ops.reload = s.sourceSaveOps.reload
	}
	if s.sourceSaveOps.syncSite != nil {
		ops.syncSite = s.sourceSaveOps.syncSite
	}
	return ops
}

func writeFileAtomic(path string, content []byte, mode os.FileMode) error {
	temporary, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp-")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(mode); err != nil {
		temporary.Close()
		return err
	}
	if _, err := temporary.Write(content); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryPath, path)
}

func recoverSourceConfig(ops sourceConfigSaveOps) error {
	if err := ops.nginxTest(); err != nil {
		return err
	}
	return ops.reload()
}

func sourceSaveFailureDetail(stage, path string, stageErr, rollbackErr error) string {
	detail := fmt.Sprintf("%s failed for %s: %v", stage, path, stageErr)
	if rollbackErr != nil {
		detail += fmt.Sprintf("; rollback failed: %v", rollbackErr)
	}
	return detail
}

func sourceSaveRecoveryDetail(stage, path string, stageErr, rollbackErr, recoveryErr error) string {
	detail := sourceSaveFailureDetail(stage, path, stageErr, rollbackErr)
	if recoveryErr != nil {
		detail += fmt.Sprintf("; recovery failed: %v", recoveryErr)
	}
	return detail
}

func (s *WebsiteService) getSiteConfWritePath(alias string) string {
	nc := global.CONF.Nginx
	if nc.IsSystemMode() {
		return filepath.Join(nc.GetSitesAvailableDir(), alias+".conf")
	}
	return GetSiteConfPath(alias)
}

func (s *WebsiteService) SwitchConfigMode(id uint, mode string) error {
	site, err := s.websiteRepo.Get(repo.WithByID(id))
	if err != nil {
		return buserr.New(constant.ErrRecordNotFound)
	}

	if mode != "managed" && mode != "source" {
		return buserr.New(constant.ErrInvalidParams)
	}
	if site.NginxConfPath != "" {
		if mode != "source" {
			return buserr.New(constant.ErrWebsiteExternalOperationDenied)
		}
		return nil
	}

	previousMode := site.ConfigMode
	site.ConfigMode = mode
	if mode == "managed" && site.Status == "running" {
		snaps, err := snapshotFiles(websiteRuntimePaths(site, true))
		if err != nil {
			return buserr.WithDetail(constant.ErrInternalServer, err.Error(), err)
		}
		if err := s.applyConfig(site); err != nil {
			return err
		}
		if err := s.websiteRepo.Save(&site); err != nil {
			site.ConfigMode = previousMode
			_ = restoreFiles(snaps)
			_ = s.reloadNginx()
			return buserr.WithDetail(constant.ErrInternalServer, err.Error(), err)
		}
		return nil
	}
	return s.websiteRepo.Save(&site)
}

// --- Nginx 配置文件管理 ---

func (s *WebsiteService) GetMainConf() (string, error) {
	nc := global.CONF.Nginx
	if !nc.IsInstalled() {
		return "", buserr.New(constant.ErrNginxNotInstalled)
	}
	data, err := os.ReadFile(nc.GetMainConf())
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (s *WebsiteService) SaveMainConf(content string) error {
	nc := global.CONF.Nginx
	if !nc.IsInstalled() {
		return buserr.New(constant.ErrNginxNotInstalled)
	}
	return s.replaceNginxFile(nc.GetMainConf(), []byte(content))
}

func (s *WebsiteService) ListConfFiles() ([]dto.NginxConfFileInfo, error) {
	if !global.CONF.Nginx.IsInstalled() {
		return nil, buserr.New(constant.ErrNginxNotInstalled)
	}
	nc := global.CONF.Nginx

	// List from sites-available (system) or conf.d (prefix)
	confDir := nc.GetSitesAvailableDir()
	entries, err := os.ReadDir(confDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []dto.NginxConfFileInfo{}, nil
		}
		return nil, err
	}

	var files []dto.NginxConfFileInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		files = append(files, dto.NginxConfFileInfo{
			Name: entry.Name(),
			Size: info.Size(),
		})
	}
	return files, nil
}

func (s *WebsiteService) GetConfFile(name string) (string, error) {
	if !global.CONF.Nginx.IsInstalled() {
		return "", buserr.New(constant.ErrNginxNotInstalled)
	}
	nc := global.CONF.Nginx
	safeName := filepath.Base(name)

	// Try sites-available first (system mode), then sites-enabled/conf.d
	filePath := filepath.Join(nc.GetSitesAvailableDir(), safeName)
	data, err := os.ReadFile(filePath)
	if err != nil && nc.IsSystemMode() {
		filePath = filepath.Join(nc.GetSitesDir(), safeName)
		data, err = os.ReadFile(filePath)
	}
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (s *WebsiteService) SaveConfFile(req dto.NginxConfUpdate) error {
	if !global.CONF.Nginx.IsInstalled() {
		return buserr.New(constant.ErrNginxNotInstalled)
	}
	filePath := filepath.Clean(req.FilePath)
	if !s.isAllowedNginxConfPath(filePath) {
		return buserr.New(constant.ErrInvalidParams)
	}
	return s.replaceNginxFile(filePath, []byte(req.Content))
}

func (s *WebsiteService) ListConfBackups(filePath string) ([]dto.NginxConfBackupInfo, error) {
	dir, err := s.configBackupDir(filepath.Clean(filePath))
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []dto.NginxConfBackupInfo{}, nil
		}
		return nil, err
	}
	var backups []dto.NginxConfBackupInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		backups = append(backups, dto.NginxConfBackupInfo{
			Name:      entry.Name(),
			FilePath:  filepath.Join(dir, entry.Name()),
			Size:      info.Size(),
			CreatedAt: info.ModTime(),
		})
	}
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].CreatedAt.After(backups[j].CreatedAt)
	})
	return backups, nil
}

func (s *WebsiteService) RestoreConfBackup(req dto.NginxConfRestoreReq) error {
	filePath := filepath.Clean(req.FilePath)
	if !s.isAllowedNginxConfPath(filePath) {
		return buserr.New(constant.ErrInvalidParams)
	}
	backupDir, err := s.configBackupDir(filePath)
	if err != nil {
		return err
	}
	backupPath := filepath.Join(backupDir, filepath.Base(req.BackupName))
	content, err := os.ReadFile(backupPath)
	if err != nil {
		return err
	}
	return s.replaceNginxFile(filePath, content)
}

func (s *WebsiteService) createConfigBackup(filePath string) error {
	filePath = filepath.Clean(filePath)
	if filePath == "" {
		return nil
	}
	data, err := os.ReadFile(filePath)
	if err != nil || len(data) == 0 {
		return err
	}
	dir, err := s.configBackupDir(filePath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	name := time.Now().Format("20060102_150405") + "_" + filepath.Base(filePath)
	return os.WriteFile(filepath.Join(dir, name), data, 0644)
}

func (s *WebsiteService) configBackupDir(filePath string) (string, error) {
	dataDir := strings.TrimSpace(global.CONF.System.DataDir)
	if dataDir == "" || !filepath.IsAbs(dataDir) {
		return "", fmt.Errorf("data directory is not configured")
	}
	safe := strings.NewReplacer("/", "__", "\\", "__", ":", "_").Replace(filepath.Clean(filePath))
	return filepath.Join(dataDir, "nginx-backups", safe), nil
}

func (s *WebsiteService) isAllowedNginxConfPath(filePath string) bool {
	nc := global.CONF.Nginx
	confDir := filepath.Clean(nc.GetConfDir())
	mainConf := filepath.Clean(nc.GetMainConf())
	filePath = filepath.Clean(filePath)
	lexicalOK := filePath == mainConf || strings.HasPrefix(filePath, confDir+string(os.PathSeparator))
	if !lexicalOK {
		return false
	}
	if info, err := os.Lstat(filePath); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return false
	} else if err != nil && !os.IsNotExist(err) {
		return false
	}
	parent := filepath.Dir(filePath)
	resolvedParent, err := filepath.EvalSymlinks(parent)
	if err != nil {
		return os.IsNotExist(err)
	}
	root, err := filepath.EvalSymlinks(confDir)
	if err != nil {
		root = confDir
	}
	resolvedParent = filepath.Clean(resolvedParent)
	root = filepath.Clean(root)
	if resolvedParent == root || strings.HasPrefix(resolvedParent, root+string(os.PathSeparator)) {
		return true
	}
	mainParent, err := filepath.EvalSymlinks(filepath.Dir(mainConf))
	if err != nil {
		return false
	}
	return filePath == mainConf && filepath.Clean(mainParent) == resolvedParent
}

func (s *WebsiteService) replaceNginxFile(path string, content []byte) error {
	snap, err := snapshotFile(path)
	if err != nil {
		return err
	}
	if snap.existed && snap.symlink == "" {
		_ = s.createConfigBackup(path)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	mode := os.FileMode(0644)
	if snap.existed && snap.symlink == "" && snap.mode != 0 {
		mode = snap.mode
	}
	if err := os.WriteFile(path, content, mode); err != nil {
		return err
	}
	if err := s.testNginxConfig(); err != nil {
		_ = snap.restore()
		return buserr.WithDetail(constant.ErrNginxConfigTest, err.Error(), err)
	}
	if err := s.reloadNginx(); err != nil {
		_ = snap.restore()
		_ = s.reloadNginx()
		return err
	}
	return nil
}

func siteServesTraffic(site model.Website) bool {
	return site.Status == "running" && site.ConfigMode != "source"
}

func websiteRuntimePaths(site model.Website, includeStored bool) []string {
	if site.ConfigMode == "source" {
		if includeStored {
			return []string{websiteHtpasswdPath(site.Alias)}
		}
		return nil
	}
	nc := global.CONF.Nginx
	var paths []string
	if nc.IsSystemMode() {
		paths = append(paths, filepath.Join(nc.GetSitesDir(), site.Alias+".conf"))
		if includeStored {
			paths = append(paths, filepath.Join(nc.GetSitesAvailableDir(), site.Alias+".conf"))
		}
	} else {
		paths = append(paths, GetSiteConfPath(site.Alias))
	}
	if includeStored {
		paths = append(paths, websiteHtpasswdPath(site.Alias))
	}
	return paths
}

type fileSnapshot struct {
	path    string
	existed bool
	data    []byte
	mode    os.FileMode
	symlink string
}

func snapshotFiles(paths []string) ([]fileSnapshot, error) {
	snaps := make([]fileSnapshot, 0, len(paths))
	for _, path := range paths {
		snap, err := snapshotFile(path)
		if err != nil {
			return nil, err
		}
		snaps = append(snaps, snap)
	}
	return snaps, nil
}

func snapshotFile(path string) (fileSnapshot, error) {
	snap := fileSnapshot{path: path, mode: 0644}
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return snap, nil
	}
	if err != nil {
		return fileSnapshot{}, err
	}
	snap.existed = true
	if perm := info.Mode().Perm(); perm != 0 {
		snap.mode = perm
	}
	if info.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(path)
		if err != nil {
			return fileSnapshot{}, err
		}
		snap.symlink = target
		return snap, nil
	}
	if !info.Mode().IsRegular() {
		return fileSnapshot{}, fmt.Errorf("not a regular file: %s", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fileSnapshot{}, err
	}
	snap.data = data
	return snap, nil
}

func (snap fileSnapshot) restore() error {
	if !snap.existed {
		if err := os.Remove(snap.path); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(snap.path), 0755); err != nil {
		return err
	}
	if snap.symlink != "" {
		if err := os.Remove(snap.path); err != nil && !os.IsNotExist(err) {
			return err
		}
		return os.Symlink(snap.symlink, snap.path)
	}
	return os.WriteFile(snap.path, snap.data, snap.mode)
}

func restoreFiles(snaps []fileSnapshot) error {
	var first error
	for i := len(snaps) - 1; i >= 0; i-- {
		if err := snaps[i].restore(); err != nil && first == nil {
			first = err
		}
	}
	return first
}

func removeFiles(paths []string) error {
	for _, path := range paths {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func (s *WebsiteService) CheckHealth(id uint) (*dto.WebsiteHealthResp, error) {
	site, err := s.websiteRepo.Get(repo.WithByID(id))
	if err != nil {
		return nil, buserr.New(constant.ErrRecordNotFound)
	}
	domain := strings.TrimSpace(site.PrimaryDomain)
	resp := &dto.WebsiteHealthResp{LastCheckedAt: time.Now()}
	client := &http.Client{Timeout: 8 * time.Second}
	for _, scheme := range []string{"http", "https"} {
		url := scheme + "://" + domain
		start := time.Now()
		check := dto.WebsiteHealthCheck{URL: url}
		r, err := client.Get(url)
		check.LatencyMS = time.Since(start).Milliseconds()
		if err != nil {
			check.Error = err.Error()
		} else {
			check.StatusCode = r.StatusCode
			check.OK = r.StatusCode >= 200 && r.StatusCode < 400
			_ = r.Body.Close()
		}
		resp.Checks = append(resp.Checks, check)
	}
	resp.CertNotAfter, resp.CertDaysLeft, resp.CertError = checkCert(domain)
	return resp, nil
}

func checkCert(domain string) (string, int, string) {
	dialer := &net.Dialer{Timeout: 5 * time.Second}
	conn, err := tls.DialWithDialer(dialer, "tcp", net.JoinHostPort(domain, "443"), &tls.Config{ServerName: domain})
	if err != nil {
		return "", 0, err.Error()
	}
	defer conn.Close()
	certs := conn.ConnectionState().PeerCertificates
	if len(certs) == 0 {
		return "", 0, "no certificate"
	}
	notAfter := certs[0].NotAfter
	return notAfter.Format(time.RFC3339), int(time.Until(notAfter).Hours() / 24), ""
}

func (s *WebsiteService) InspectSite(id uint) (*dto.WebsiteInspectResp, error) {
	site, err := s.websiteRepo.Get(repo.WithByID(id))
	if err != nil {
		return nil, buserr.New(constant.ErrRecordNotFound)
	}
	resp := &dto.WebsiteInspectResp{}
	dir := strings.TrimSpace(site.SiteDir)
	if dir == "" {
		resp.Issues = append(resp.Issues, "未配置网站目录")
		return resp, nil
	}
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		resp.Issues = append(resp.Issues, "网站目录不存在或不是目录")
		return resp, nil
	}
	resp.SiteDirExists = true
	if entries, err := os.ReadDir(dir); err == nil {
		resp.Readable = true
		_ = entries
	} else {
		resp.Issues = append(resp.Issues, "当前面板进程无法读取网站目录")
	}
	indexFiles := mergeIndexCandidates(strings.Fields(site.IndexFile), []string{"index.html", "index.htm", "index.php"})
	for _, name := range indexFiles {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			resp.IndexFiles = append(resp.IndexFiles, name)
		}
	}
	if len(resp.IndexFiles) == 0 && site.Type == "static" {
		resp.Issues = append(resp.Issues, "未发现常见首页文件")
	}
	mode := info.Mode().Perm()
	if mode&0444 == 0 || mode&0111 == 0 {
		resp.Issues = append(resp.Issues, "目录权限可能不足，Nginx 可能无法读取或进入")
	}
	return resp, nil
}

func mergeIndexCandidates(primary, fallback []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, group := range [][]string{primary, fallback} {
		for _, item := range group {
			item = strings.TrimSpace(item)
			if item == "" || seen[item] {
				continue
			}
			seen[item] = true
			result = append(result, item)
		}
	}
	return result
}

func (s *WebsiteService) DetectLogPaths(id uint) (*dto.WebsiteLogPathDetectResp, error) {
	site, err := s.websiteRepo.Get(repo.WithByID(id))
	if err != nil {
		return nil, buserr.New(constant.ErrRecordNotFound)
	}
	names := []string{site.PrimaryDomain, site.Alias}
	logDirs := []string{
		global.CONF.Nginx.GetLogDir(),
		filepath.Join(global.CONF.Nginx.GetLogDir(), "sites"),
		"/www/wwwlogs",
		"/var/log/nginx",
		"/var/log/nginx/sites",
	}
	resp := &dto.WebsiteLogPathDetectResp{}
	seen := make(map[string]bool)
	addIfExists := func(kind, p string) {
		if seen[p] {
			return
		}
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			seen[p] = true
			if kind == "error" {
				resp.Error = append(resp.Error, p)
			} else {
				resp.Access = append(resp.Access, p)
			}
		}
	}
	for _, dir := range logDirs {
		for _, name := range names {
			if strings.TrimSpace(name) == "" {
				continue
			}
			addIfExists("access", filepath.Join(dir, name+".access.log"))
			addIfExists("access", filepath.Join(dir, name+".log"))
			addIfExists("error", filepath.Join(dir, name+".error.log"))
			addIfExists("error", filepath.Join(dir, name+".err.log"))
		}
	}
	return resp, nil
}

func (s *WebsiteService) GetLogAlerts(req dto.WebsiteLogAlertReq) ([]dto.WebsiteLogAlert, error) {
	site, err := s.websiteRepo.Get(repo.WithByID(req.ID))
	if err != nil {
		return nil, buserr.New(constant.ErrRecordNotFound)
	}
	logPath := s.getWebsiteLogPath(site, "access")
	cutoff := parseCutoff(req.TimeRange)
	entries, err := parseAccessLog(logPath, cutoff, maxLinesForRange(req.TimeRange))
	if err != nil {
		return []dto.WebsiteLogAlert{}, nil
	}
	if len(entries) == 0 {
		return []dto.WebsiteLogAlert{}, nil
	}
	var alerts []dto.WebsiteLogAlert
	var errors, notFound, threats int64
	ipCount := make(map[string]int64)
	for _, e := range entries {
		if e.Status >= 500 {
			errors++
		}
		if e.Status == 404 {
			notFound++
		}
		if classifyThreat(e.URL) != "" {
			threats++
		}
		ipCount[e.IP]++
	}
	total := int64(len(entries))
	if float64(errors)/float64(total) > 0.05 {
		alerts = append(alerts, dto.WebsiteLogAlert{Level: "danger", Type: "5xx", Message: "5xx 错误比例超过 5%", Count: errors})
	}
	if notFound > 100 || float64(notFound)/float64(total) > 0.2 {
		alerts = append(alerts, dto.WebsiteLogAlert{Level: "warning", Type: "404", Message: "404 请求偏多，可能存在资源缺失或扫描", Count: notFound})
	}
	if threats > 0 {
		alerts = append(alerts, dto.WebsiteLogAlert{Level: "warning", Type: "threat", Message: "发现疑似恶意探测请求", Count: threats})
	}
	for ip, count := range ipCount {
		if count > 1000 || float64(count)/float64(total) > 0.35 {
			alerts = append(alerts, dto.WebsiteLogAlert{Level: "warning", Type: "hot_ip", Message: "单个 IP 请求占比过高: " + ip, Count: count})
			break
		}
	}
	return alerts, nil
}

// --- 内部方法 ---

func (s *WebsiteService) applyConfig(site model.Website) error {
	_, err := s.applyConfigWithRollback(site)
	return err
}

func (s *WebsiteService) applyConfigWithRollback(site model.Website) (func() error, error) {
	if site.LimitConn > 0 {
		if err := EnsureLimitConnZone(); err != nil {
			return nil, err
		}
	}
	gen := NewNginxConfigGenerator()
	config, err := gen.Generate(site)
	if err != nil {
		return nil, err
	}

	nc := global.CONF.Nginx
	var confPath, enabledPath string
	if nc.IsSystemMode() {
		availDir := nc.GetSitesAvailableDir()
		enabledDir := nc.GetSitesDir()
		os.MkdirAll(availDir, 0755)
		os.MkdirAll(enabledDir, 0755)
		confPath = filepath.Join(availDir, site.Alias+".conf")
		enabledPath = filepath.Join(enabledDir, site.Alias+".conf")
	} else {
		confPath = GetSiteConfPath(site.Alias)
	}

	backup, readErr := os.ReadFile(confPath)
	existed := readErr == nil
	if readErr != nil && !os.IsNotExist(readErr) {
		return nil, readErr
	}
	htpasswdPath := websiteHtpasswdPath(site.Alias)
	htpasswdBackup, htpasswdReadErr := os.ReadFile(htpasswdPath)
	htpasswdExisted := htpasswdReadErr == nil
	if htpasswdReadErr != nil && !os.IsNotExist(htpasswdReadErr) {
		return nil, htpasswdReadErr
	}
	_ = s.createConfigBackup(confPath)
	os.MkdirAll(filepath.Dir(confPath), 0755)
	if err := os.WriteFile(confPath, []byte(config), 0644); err != nil {
		return nil, fmt.Errorf("write config failed: %v", err)
	}
	if enabledPath != "" {
		if _, err := os.Lstat(enabledPath); os.IsNotExist(err) {
			_ = os.Symlink(confPath, enabledPath)
		}
	}
	if site.BasicAuth && site.BasicUser != "" && site.BasicPassword != "" {
		s.writeHtpasswd(site)
	}

	restoreFiles := func() error {
		if existed {
			if err := os.WriteFile(confPath, backup, 0644); err != nil {
				return err
			}
		} else {
			_ = os.Remove(confPath)
			if enabledPath != "" {
				_ = os.Remove(enabledPath)
			}
		}
		if htpasswdExisted {
			if err := os.WriteFile(htpasswdPath, htpasswdBackup, 0644); err != nil {
				return err
			}
		} else {
			_ = os.Remove(htpasswdPath)
		}
		return nil
	}
	rollback := func() error {
		if err := restoreFiles(); err != nil {
			return err
		}
		return s.reloadNginx()
	}

	if err := s.testNginxConfig(); err != nil {
		_ = restoreFiles()
		return nil, buserr.WithDetail(constant.ErrNginxConfigTest, err.Error(), err)
	}
	if err := s.reloadNginx(); err != nil {
		_ = rollback()
		return nil, err
	}
	return rollback, nil
}

func (s *WebsiteService) testNginxConfig() error {
	nc := global.CONF.Nginx
	if !nc.IsInstalled() {
		return fmt.Errorf("nginx not installed")
	}

	var output string
	var err error
	if nc.IsSystemMode() {
		output, err = cmd.ExecWithOutput(nc.GetBinary(), "-t")
	} else {
		output, err = cmd.ExecWithOutput(nc.GetBinary(), "-p", nc.InstallDir, "-t")
	}
	if err != nil {
		errMsg := output
		if errMsg == "" {
			errMsg = err.Error()
		}
		return fmt.Errorf("%s", errMsg)
	}
	return nil
}

func (s *WebsiteService) reloadNginx() error {
	nc := global.CONF.Nginx
	if !nc.IsInstalled() {
		return nil
	}
	pidPath := nc.GetPidPath()
	if _, err := os.Stat(pidPath); os.IsNotExist(err) {
		return nil
	}

	if nc.IsSystemMode() {
		_, err := cmd.ExecWithOutput("systemctl", "reload", "nginx")
		return err
	}
	_, err := cmd.ExecWithOutput(nc.GetBinary(), "-p", nc.InstallDir, "-s", "reload")
	return err
}

func websiteHtpasswdPath(alias string) string {
	return filepath.Join(global.CONF.Nginx.GetConfDir(), "auth", alias+".htpasswd")
}

func (s *WebsiteService) writeHtpasswd(site model.Website) {
	htpasswdPath := websiteHtpasswdPath(site.Alias)
	os.MkdirAll(filepath.Dir(htpasswdPath), 0755)

	// 使用 openssl 或 htpasswd 生成密码行
	// 格式: user:{SHA}base64hash 或 user:$apr1$...
	// 简单方案：使用 openssl passwd
	output, err := exec.Command("openssl", "passwd", "-apr1", site.BasicPassword).Output()
	if err != nil {
		global.LOG.Warnf("Generate htpasswd failed: %v", err)
		return
	}
	line := fmt.Sprintf("%s:%s\n", site.BasicUser, strings.TrimSpace(string(output)))
	os.WriteFile(htpasswdPath, []byte(line), 0644)
}

func domainToAlias(domain string) string {
	alias := strings.ReplaceAll(domain, ".", "_")
	alias = strings.ReplaceAll(alias, "*", "wildcard")
	alias = strings.ReplaceAll(alias, ":", "_")
	return alias
}
