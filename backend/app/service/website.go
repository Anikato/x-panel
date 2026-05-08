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
	Update(req dto.WebsiteUpdate) error
	Delete(id uint) error
	SearchWithPage(req dto.WebsiteSearch) (int64, []dto.WebsiteInfo, error)
	GetDetail(id uint) (*dto.WebsiteDetail, error)
	Enable(id uint) error
	Disable(id uint) error
	GetNginxConfig(id uint) (string, error)
	GetSiteLog(req dto.WebsiteLogReq) (string, error)

	// Source-mode config editing
	GetSiteConfContent(id uint) (string, error)
	SaveSiteConfContent(id uint, content string) error
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
	websiteRepo repo.IWebsiteRepo
	certRepo    repo.ICertificateRepo
}

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
		Domains:           req.Domains,
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

	// 如果域名变了，检查唯一性
	if req.PrimaryDomain != "" && req.PrimaryDomain != site.PrimaryDomain {
		exist, _ := s.websiteRepo.Get(repo.WithByPrimaryDomain(req.PrimaryDomain))
		if exist.ID > 0 && exist.ID != site.ID {
			return buserr.New(constant.ErrWebsiteDomainExist)
		}
		site.PrimaryDomain = req.PrimaryDomain
		site.Alias = domainToAlias(req.PrimaryDomain)
	}

	site.Domains = req.Domains
	site.SiteDir = req.SiteDir
	site.IndexFile = req.IndexFile
	site.HttpPort = req.HttpPort
	site.HttpsPort = req.HttpsPort
	site.ProxyPass = req.ProxyPass
	site.WebSocket = req.WebSocket
	site.SSLEnable = req.SSLEnable
	site.CertificateID = req.CertificateID
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

	if err := s.websiteRepo.Save(&site); err != nil {
		return buserr.WithDetail(constant.ErrInternalServer, err.Error(), err)
	}

	// 如果网站是运行中的且为托管模式，自动重新生成配置并 reload
	if site.Status == "running" && site.ConfigMode != "source" {
		if err := s.applyConfig(site); err != nil {
			global.LOG.Warnf("Auto apply config failed for %s: %v", site.PrimaryDomain, err)
			return buserr.WithDetail(constant.ErrWebsiteApplyConfig, err.Error(), err)
		}
	}

	return nil
}

func (s *WebsiteService) Delete(id uint) error {
	site, err := s.websiteRepo.Get(repo.WithByID(id))
	if err != nil {
		return buserr.New(constant.ErrRecordNotFound)
	}

	nc := global.CONF.Nginx
	needReload := site.Status == "running" && site.ConfigMode != "source"

	// 清理所有 nginx 配置文件（无论网站状态）
	if site.ConfigMode != "source" {
		if nc.IsSystemMode() {
			os.Remove(filepath.Join(nc.GetSitesDir(), site.Alias+".conf"))
			os.Remove(filepath.Join(nc.GetSitesAvailableDir(), site.Alias+".conf"))
		} else {
			os.Remove(GetSiteConfPath(site.Alias))
		}
	}

	// 删除 htpasswd 文件
	authDir := filepath.Join(nc.GetConfDir(), "auth")
	os.Remove(filepath.Join(authDir, site.Alias+".htpasswd"))

	if needReload {
		s.reloadNginx()
	}

	return s.websiteRepo.Delete(repo.WithByID(id))
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
		items = append(items, dto.WebsiteInfo{
			ID:            site.ID,
			PrimaryDomain: site.PrimaryDomain,
			Domains:       site.Domains,
			Alias:         site.Alias,
			Type:          site.Type,
			Status:        site.Status,
			SSLEnable:     site.SSLEnable,
			ConfigMode:    site.ConfigMode,
			Remark:        site.Remark,
			SiteDir:       site.SiteDir,
			CreatedAt:     site.CreatedAt,
		})
	}
	return total, items, nil
}

func (s *WebsiteService) GetDetail(id uint) (*dto.WebsiteDetail, error) {
	site, err := s.websiteRepo.Get(repo.WithByID(id))
	if err != nil {
		return nil, buserr.New(constant.ErrRecordNotFound)
	}

	detail := &dto.WebsiteDetail{
		ID:                site.ID,
		PrimaryDomain:     site.PrimaryDomain,
		Domains:           site.Domains,
		Alias:             site.Alias,
		Type:              site.Type,
		Status:            site.Status,
		SiteDir:           site.SiteDir,
		IndexFile:         site.IndexFile,
		HttpPort:          site.HttpPort,
		HttpsPort:         site.HttpsPort,
		ProxyPass:         site.ProxyPass,
		WebSocket:         site.WebSocket,
		SSLEnable:         site.SSLEnable,
		CertificateID:     site.CertificateID,
		HttpConfig:        site.HttpConfig,
		HSTS:              site.HSTS,
		Http2Enable:       site.Http2Enable,
		SSLProtocols:      site.SSLProtocols,
		BasicAuth:         site.BasicAuth,
		BasicUser:         site.BasicUser,
		BasicPassword:     site.BasicPassword,
		AntiLeech:         site.AntiLeech,
		LeechReferers:     site.LeechReferers,
		LimitRate:         site.LimitRate,
		LimitConn:         site.LimitConn,
		Rewrite:           site.Rewrite,
		Redirects:         site.Redirects,
		AccessLog:         site.AccessLog,
		ErrorLog:          site.ErrorLog,
		AccessLogPath:     site.AccessLogPath,
		ErrorLogPath:      site.ErrorLogPath,
		GzipEnable:        site.GzipEnable,
		SecurityHeaders:   site.SecurityHeaders,
		StaticCacheEnable: site.StaticCacheEnable,
		Upstream:          site.Upstream,
		CustomNginx:       site.CustomNginx,
		DefaultServer:     site.DefaultServer,
		Remark:            site.Remark,
		ConfigMode:        site.ConfigMode,
	}

	if site.CertificateID > 0 {
		cert, err := s.certRepo.Get(repo.WithByID(site.CertificateID))
		if err == nil {
			detail.CertificateDomain = cert.PrimaryDomain
		}
	}

	// 生成当前配置预览
	gen := NewNginxConfigGenerator()
	config, _ := gen.Generate(site)
	detail.NginxConfig = config

	return detail, nil
}

func (s *WebsiteService) Enable(id uint) error {
	if !global.CONF.Nginx.IsInstalled() {
		return buserr.New(constant.ErrNginxNotInstalled)
	}

	site, err := s.websiteRepo.Get(repo.WithByID(id))
	if err != nil {
		return buserr.New(constant.ErrRecordNotFound)
	}
	if site.Status == "running" {
		return nil
	}

	// 确保 nginx.conf 包含 conf.d
	if err := EnsureNginxInclude(); err != nil {
		global.LOG.Warnf("Ensure nginx include failed: %v", err)
	}

	if err := s.applyConfig(site); err != nil {
		return err
	}

	site.Status = "running"
	return s.websiteRepo.Save(&site)
}

func (s *WebsiteService) Disable(id uint) error {
	site, err := s.websiteRepo.Get(repo.WithByID(id))
	if err != nil {
		return buserr.New(constant.ErrRecordNotFound)
	}
	if site.Status == "stopped" {
		return nil
	}

	s.removeConfig(site)
	s.reloadNginx()

	site.Status = "stopped"
	return s.websiteRepo.Save(&site)
}

func (s *WebsiteService) GetNginxConfig(id uint) (string, error) {
	site, err := s.websiteRepo.Get(repo.WithByID(id))
	if err != nil {
		return "", buserr.New(constant.ErrRecordNotFound)
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

func (s *WebsiteService) GetSiteConfContent(id uint) (string, error) {
	site, err := s.websiteRepo.Get(repo.WithByID(id))
	if err != nil {
		return "", buserr.New(constant.ErrRecordNotFound)
	}

	nc := global.CONF.Nginx

	// Try reading from the actual config file
	var confPath string
	if nc.IsSystemMode() {
		// System mode: try sites-available first, then sites-enabled
		availPath := filepath.Join(nc.GetSitesAvailableDir(), site.Alias+".conf")
		enabledPath := filepath.Join(nc.GetSitesDir(), site.Alias+".conf")
		if _, err := os.Stat(availPath); err == nil {
			confPath = availPath
		} else if _, err := os.Stat(enabledPath); err == nil {
			confPath = enabledPath
		}
	} else {
		confPath = GetSiteConfPath(site.Alias)
	}

	if confPath != "" {
		data, err := os.ReadFile(confPath)
		if err == nil {
			return string(data), nil
		}
		if !os.IsNotExist(err) {
			return "", err
		}
	}

	// No config file on disk yet, generate from DB
	gen := NewNginxConfigGenerator()
	return gen.Generate(site)
}

func (s *WebsiteService) SaveSiteConfContent(id uint, content string) error {
	site, err := s.websiteRepo.Get(repo.WithByID(id))
	if err != nil {
		return buserr.New(constant.ErrRecordNotFound)
	}

	nc := global.CONF.Nginx
	confPath := s.getSiteConfWritePath(site.Alias)

	os.MkdirAll(filepath.Dir(confPath), 0755)
	backup, _ := os.ReadFile(confPath)
	_ = s.createConfigBackup(confPath)

	if err := os.WriteFile(confPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("write config failed: %v", err)
	}

	// System mode: ensure symlink from sites-enabled
	if nc.IsSystemMode() {
		enabledPath := filepath.Join(nc.GetSitesDir(), site.Alias+".conf")
		if _, err := os.Lstat(enabledPath); os.IsNotExist(err) {
			os.Symlink(confPath, enabledPath)
		}
	}

	if err := s.testNginxConfig(); err != nil {
		if backup != nil {
			os.WriteFile(confPath, backup, 0644)
		} else {
			if nc.IsSystemMode() {
				enabledPath := filepath.Join(nc.GetSitesDir(), site.Alias+".conf")
				os.Remove(enabledPath)
			}
			os.Remove(confPath)
		}
		return buserr.WithDetail(constant.ErrNginxConfigTest, err.Error(), err)
	}

	if site.Status == "running" {
		s.reloadNginx()
	}

	if site.ConfigMode != "source" {
		site.ConfigMode = "source"
		s.websiteRepo.Save(&site)
	}

	return nil
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

	site.ConfigMode = mode
	if err := s.websiteRepo.Save(&site); err != nil {
		return err
	}

	// When switching back to managed, regenerate config
	if mode == "managed" && site.Status == "running" {
		return s.applyConfig(site)
	}
	return nil
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
	mainConf := nc.GetMainConf()

	backup, _ := os.ReadFile(mainConf)
	_ = s.createConfigBackup(mainConf)

	if err := os.WriteFile(mainConf, []byte(content), 0644); err != nil {
		return err
	}

	if err := s.testNginxConfig(); err != nil {
		os.WriteFile(mainConf, backup, 0644)
		return buserr.WithDetail(constant.ErrNginxConfigTest, err.Error(), err)
	}

	return s.reloadNginx()
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
	nc := global.CONF.Nginx
	confDir := nc.GetConfDir()
	filePath := filepath.Clean(req.FilePath)
	if !strings.HasPrefix(filePath, confDir) {
		return buserr.New(constant.ErrInvalidParams)
	}

	backup, _ := os.ReadFile(filePath)
	_ = s.createConfigBackup(filePath)

	if err := os.WriteFile(filePath, []byte(req.Content), 0644); err != nil {
		return err
	}

	if err := s.testNginxConfig(); err != nil {
		os.WriteFile(filePath, backup, 0644)
		return buserr.WithDetail(constant.ErrNginxConfigTest, err.Error(), err)
	}

	return s.reloadNginx()
}

func (s *WebsiteService) ListConfBackups(filePath string) ([]dto.NginxConfBackupInfo, error) {
	dir := s.configBackupDir(filepath.Clean(filePath))
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
	backupPath := filepath.Join(s.configBackupDir(filePath), filepath.Base(req.BackupName))
	content, err := os.ReadFile(backupPath)
	if err != nil {
		return err
	}
	current, _ := os.ReadFile(filePath)
	_ = s.createConfigBackup(filePath)
	if err := os.WriteFile(filePath, content, 0644); err != nil {
		return err
	}
	if err := s.testNginxConfig(); err != nil {
		if current != nil {
			_ = os.WriteFile(filePath, current, 0644)
		}
		return buserr.WithDetail(constant.ErrNginxConfigTest, err.Error(), err)
	}
	return s.reloadNginx()
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
	dir := s.configBackupDir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	name := time.Now().Format("20060102_150405") + "_" + filepath.Base(filePath)
	return os.WriteFile(filepath.Join(dir, name), data, 0644)
}

func (s *WebsiteService) configBackupDir(filePath string) string {
	safe := strings.NewReplacer("/", "__", "\\", "__", ":", "_").Replace(filepath.Clean(filePath))
	return filepath.Join(global.CONF.System.DataDir, "nginx-backups", safe)
}

func (s *WebsiteService) isAllowedNginxConfPath(filePath string) bool {
	nc := global.CONF.Nginx
	confDir := filepath.Clean(nc.GetConfDir())
	mainConf := filepath.Clean(nc.GetMainConf())
	return filePath == mainConf || strings.HasPrefix(filePath, confDir+string(os.PathSeparator))
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
	indexFiles := strings.Fields(site.IndexFile)
	if len(indexFiles) == 0 {
		indexFiles = []string{"index.html", "index.htm", "index.php"}
	}
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
	gen := NewNginxConfigGenerator()
	config, err := gen.Generate(site)
	if err != nil {
		return err
	}

	nc := global.CONF.Nginx

	// System mode: write to sites-available and symlink to sites-enabled
	if nc.IsSystemMode() {
		availDir := nc.GetSitesAvailableDir()
		enabledDir := nc.GetSitesDir()
		os.MkdirAll(availDir, 0755)
		os.MkdirAll(enabledDir, 0755)

		availPath := filepath.Join(availDir, site.Alias+".conf")
		enabledPath := filepath.Join(enabledDir, site.Alias+".conf")
		backup, _ := os.ReadFile(availPath)
		_ = s.createConfigBackup(availPath)

		if err := os.WriteFile(availPath, []byte(config), 0644); err != nil {
			return fmt.Errorf("write config failed: %v", err)
		}

		// Create symlink if not exists
		if _, err := os.Lstat(enabledPath); os.IsNotExist(err) {
			os.Symlink(availPath, enabledPath)
		}

		if site.BasicAuth && site.BasicUser != "" && site.BasicPassword != "" {
			s.writeHtpasswd(site)
		}

		if err := s.testNginxConfig(); err != nil {
			if backup != nil {
				os.WriteFile(availPath, backup, 0644)
			} else {
				os.Remove(enabledPath)
				os.Remove(availPath)
			}
			return buserr.WithDetail(constant.ErrNginxConfigTest, err.Error(), err)
		}

		return s.reloadNginx()
	}

	// Prefix mode: write to conf.d
	confPath := GetSiteConfPath(site.Alias)
	backup, _ := os.ReadFile(confPath)
	_ = s.createConfigBackup(confPath)

	os.MkdirAll(filepath.Dir(confPath), 0755)
	if err := os.WriteFile(confPath, []byte(config), 0644); err != nil {
		return fmt.Errorf("write config failed: %v", err)
	}

	if site.BasicAuth && site.BasicUser != "" && site.BasicPassword != "" {
		s.writeHtpasswd(site)
	}

	if err := s.testNginxConfig(); err != nil {
		if backup != nil {
			os.WriteFile(confPath, backup, 0644)
		} else {
			os.Remove(confPath)
		}
		return buserr.WithDetail(constant.ErrNginxConfigTest, err.Error(), err)
	}

	return s.reloadNginx()
}

func (s *WebsiteService) removeConfig(site model.Website) {
	nc := global.CONF.Nginx
	if nc.IsSystemMode() {
		enabledPath := filepath.Join(nc.GetSitesDir(), site.Alias+".conf")
		os.Remove(enabledPath)
		// Keep sites-available for reference
	} else {
		confPath := GetSiteConfPath(site.Alias)
		os.Remove(confPath)
	}
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

func (s *WebsiteService) writeHtpasswd(site model.Website) {
	authDir := filepath.Join(global.CONF.Nginx.GetConfDir(), "auth")
	os.MkdirAll(authDir, 0755)
	htpasswdPath := filepath.Join(authDir, site.Alias+".htpasswd")

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
