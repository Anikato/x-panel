package service

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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

	// Nginx config file management
	GetMainConf() (string, error)
	SaveMainConf(content string) error
	ListConfFiles() ([]dto.NginxConfFileInfo, error)
	GetConfFile(name string) (string, error)
	SaveConfFile(req dto.NginxConfUpdate) error
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
	// 检查域名唯一性
	exist, _ := s.websiteRepo.Get(repo.WithByPrimaryDomain(req.PrimaryDomain))
	if exist.ID > 0 {
		return buserr.New(constant.ErrWebsiteDomainExist)
	}

	alias := domainToAlias(req.PrimaryDomain)

	site := model.Website{
		PrimaryDomain: req.PrimaryDomain,
		Domains:       req.Domains,
		Alias:         alias,
		Type:          req.Type,
		Status:        "stopped",
		Remark:        req.Remark,
		SiteDir:       req.SiteDir,
		ProxyPass:     req.ProxyPass,
		IndexFile:     "index.html index.htm",
		HttpConfig:    "HTTPSRedirect",
		SSLProtocols:  "TLSv1.2 TLSv1.3",
		AccessLog:     true,
		ErrorLog:      true,
	}

	if site.Type == "static" && site.SiteDir == "" {
		site.SiteDir = fmt.Sprintf("/var/www/%s", req.PrimaryDomain)
	}

	if err := s.websiteRepo.Create(&site); err != nil {
		return buserr.WithDetail(constant.ErrInternalServer, err.Error(), err)
	}

	// 为静态站点创建目录
	if site.Type == "static" && site.SiteDir != "" {
		os.MkdirAll(site.SiteDir, 0755)
		indexPath := filepath.Join(site.SiteDir, "index.html")
		if _, err := os.Stat(indexPath); os.IsNotExist(err) {
			defaultHTML := fmt.Sprintf("<!DOCTYPE html>\n<html>\n<head><title>%s</title></head>\n<body>\n<h1>Welcome to %s</h1>\n<p>Site is working.</p>\n</body>\n</html>", site.PrimaryDomain, site.PrimaryDomain)
			os.WriteFile(indexPath, []byte(defaultHTML), 0644)
		}
	}

	global.LOG.Infof("Website created: %s (%s)", site.PrimaryDomain, site.Type)
	return nil
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
	site.ProxyPass = req.ProxyPass
	site.WebSocket = req.WebSocket
	site.SSLEnable = req.SSLEnable
	site.CertificateID = req.CertificateID
	site.HttpConfig = req.HttpConfig
	site.HSTS = req.HSTS
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
	site.CustomNginx = req.CustomNginx
	site.DefaultServer = req.DefaultServer
	site.Remark = req.Remark

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

	// 如果网站是运行中的，自动重新生成配置并 reload
	if site.Status == "running" {
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

	// 先禁用
	if site.Status == "running" {
		s.removeConfig(site)
		s.reloadNginx()
	}

	// 删除 htpasswd 文件
	authDir := filepath.Join(global.CONF.Nginx.GetConfDir(), "auth")
	os.Remove(filepath.Join(authDir, site.Alias+".htpasswd"))

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
			Remark:        site.Remark,
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
		ID:            site.ID,
		PrimaryDomain: site.PrimaryDomain,
		Domains:       site.Domains,
		Alias:         site.Alias,
		Type:          site.Type,
		Status:        site.Status,
		SiteDir:       site.SiteDir,
		IndexFile:     site.IndexFile,
		ProxyPass:     site.ProxyPass,
		WebSocket:     site.WebSocket,
		SSLEnable:     site.SSLEnable,
		CertificateID: site.CertificateID,
		HttpConfig:    site.HttpConfig,
		HSTS:          site.HSTS,
		SSLProtocols:  site.SSLProtocols,
		BasicAuth:     site.BasicAuth,
		BasicUser:     site.BasicUser,
		BasicPassword: site.BasicPassword,
		AntiLeech:     site.AntiLeech,
		LeechReferers: site.LeechReferers,
		LimitRate:     site.LimitRate,
		LimitConn:     site.LimitConn,
		Rewrite:       site.Rewrite,
		Redirects:     site.Redirects,
		AccessLog:     site.AccessLog,
		ErrorLog:      site.ErrorLog,
		CustomNginx:   site.CustomNginx,
		DefaultServer: site.DefaultServer,
		Remark:        site.Remark,
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

	logDir := filepath.Join(global.CONF.Nginx.GetLogDir(), "sites")
	var logPath string
	switch req.Type {
	case "access":
		logPath = filepath.Join(logDir, site.PrimaryDomain+".access.log")
	case "error":
		logPath = filepath.Join(logDir, site.PrimaryDomain+".error.log")
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

// --- Nginx 配置文件管理 ---

func (s *WebsiteService) GetMainConf() (string, error) {
	if !global.CONF.Nginx.IsInstalled() {
		return "", buserr.New(constant.ErrNginxNotInstalled)
	}
	data, err := os.ReadFile(global.CONF.Nginx.GetMainConf())
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (s *WebsiteService) SaveMainConf(content string) error {
	if !global.CONF.Nginx.IsInstalled() {
		return buserr.New(constant.ErrNginxNotInstalled)
	}
	mainConf := global.CONF.Nginx.GetMainConf()

	// 备份
	backup, _ := os.ReadFile(mainConf)

	if err := os.WriteFile(mainConf, []byte(content), 0644); err != nil {
		return err
	}

	// 测试配置
	if err := s.testNginxConfig(); err != nil {
		os.WriteFile(mainConf, backup, 0644)
		return buserr.WithDetail(constant.ErrNginxConfigTest, err.Error(), err)
	}

	return nil
}

func (s *WebsiteService) ListConfFiles() ([]dto.NginxConfFileInfo, error) {
	if !global.CONF.Nginx.IsInstalled() {
		return nil, buserr.New(constant.ErrNginxNotInstalled)
	}
	confDir := global.CONF.Nginx.GetSitesDir()
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
	filePath := filepath.Join(global.CONF.Nginx.GetSitesDir(), filepath.Base(name))
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (s *WebsiteService) SaveConfFile(req dto.NginxConfUpdate) error {
	if !global.CONF.Nginx.IsInstalled() {
		return buserr.New(constant.ErrNginxNotInstalled)
	}
	// 安全检查：只允许写 conf 目录下的文件
	confDir := global.CONF.Nginx.GetConfDir()
	filePath := filepath.Clean(req.FilePath)
	if !strings.HasPrefix(filePath, confDir) {
		return buserr.New(constant.ErrInvalidParams)
	}

	backup, _ := os.ReadFile(filePath)

	if err := os.WriteFile(filePath, []byte(req.Content), 0644); err != nil {
		return err
	}

	if err := s.testNginxConfig(); err != nil {
		os.WriteFile(filePath, backup, 0644)
		return buserr.WithDetail(constant.ErrNginxConfigTest, err.Error(), err)
	}

	return nil
}

// --- 内部方法 ---

func (s *WebsiteService) applyConfig(site model.Website) error {
	gen := NewNginxConfigGenerator()
	config, err := gen.Generate(site)
	if err != nil {
		return err
	}

	confPath := GetSiteConfPath(site.Alias)
	backup, _ := os.ReadFile(confPath)

	if err := os.WriteFile(confPath, []byte(config), 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %v", err)
	}

	// 写入 htpasswd 文件
	if site.BasicAuth && site.BasicUser != "" && site.BasicPassword != "" {
		s.writeHtpasswd(site)
	}

	// 测试配置
	if err := s.testNginxConfig(); err != nil {
		// 回滚
		if backup != nil {
			os.WriteFile(confPath, backup, 0644)
		} else {
			os.Remove(confPath)
		}
		return buserr.WithDetail(constant.ErrNginxConfigTest, err.Error(), err)
	}

	// 重载 Nginx
	return s.reloadNginx()
}

func (s *WebsiteService) removeConfig(site model.Website) {
	confPath := GetSiteConfPath(site.Alias)
	os.Remove(confPath)
}

func (s *WebsiteService) testNginxConfig() error {
	nc := global.CONF.Nginx
	if !nc.IsInstalled() {
		return fmt.Errorf("nginx not installed")
	}
	output, err := cmd.ExecWithOutput(nc.GetBinary(), "-p", nc.InstallDir, "-t")
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
	// 检查 Nginx 是否在运行
	pidPath := nc.GetPidPath()
	if _, err := os.Stat(pidPath); os.IsNotExist(err) {
		return nil
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
