package service

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"xpanel/app/dto"
	"xpanel/app/model"
	"xpanel/app/repo"
	"xpanel/buserr"
	"xpanel/constant"
	"xpanel/global"
)

type nginxSiteMetadata struct {
	PrimaryDomain string
	Domains       []string
	Type          string
	Root          string
	ProxyPass     string
	HTTPPort      int
	HTTPSPort     int
	SSL           bool
	AccessLogPath string
	ErrorLogPath  string
	CertPath      string
	KeyPath       string
	Warnings      []string
}

func tokenizeNginx(content string) ([]string, error) {
	var tokens []string
	var word strings.Builder
	var quote rune
	escaped := false
	comment := false
	flush := func() {
		if word.Len() > 0 {
			tokens = append(tokens, word.String())
			word.Reset()
		}
	}

	for _, current := range content {
		if comment {
			if current == '\n' {
				comment = false
			}
			continue
		}
		if escaped {
			word.WriteRune(current)
			escaped = false
			continue
		}
		if current == '\\' {
			escaped = true
			continue
		}
		if quote != 0 {
			if current == quote {
				quote = 0
			} else {
				word.WriteRune(current)
			}
			continue
		}
		switch {
		case current == '#' && word.Len() == 0:
			comment = true
		case current == '\'' || current == '"':
			quote = current
		case unicode.IsSpace(current):
			flush()
		case current == '{' || current == '}' || current == ';':
			flush()
			tokens = append(tokens, string(current))
		default:
			word.WriteRune(current)
		}
	}
	if quote != 0 {
		return nil, fmt.Errorf("nginx 配置包含未闭合的引号")
	}
	if escaped {
		word.WriteRune('\\')
	}
	flush()
	return tokens, nil
}

func parseNginxSiteMetadata(content string) (nginxSiteMetadata, error) {
	tokens, err := tokenizeNginx(content)
	if err != nil {
		return nginxSiteMetadata{}, err
	}

	metadata := nginxSiteMetadata{Type: "static"}
	domainSeen := make(map[string]struct{})
	serverBlocks := 0
	for i := 0; i+1 < len(tokens); i++ {
		if tokens[i] != "server" || tokens[i+1] != "{" {
			continue
		}
		end := matchingBrace(tokens, i+1)
		if end < 0 {
			return nginxSiteMetadata{}, fmt.Errorf("nginx server 块未闭合")
		}
		serverBlocks++
		for _, directive := range nginxDirectives(tokens[i+2 : end]) {
			applyNginxDirective(&metadata, directive, domainSeen)
		}
		i = end
	}
	if serverBlocks == 0 {
		return nginxSiteMetadata{}, fmt.Errorf("nginx 配置未包含 server 块")
	}
	if len(metadata.Domains) == 0 {
		return nginxSiteMetadata{}, fmt.Errorf("nginx 配置未包含有效的 server_name")
	}
	metadata.PrimaryDomain = metadata.Domains[0]
	metadata.SSL = metadata.SSL || metadata.CertPath != "" || metadata.KeyPath != ""
	if metadata.ProxyPass != "" {
		metadata.Type = "reverse_proxy"
	}
	return metadata, nil
}

func matchingBrace(tokens []string, open int) int {
	depth := 0
	for i := open; i < len(tokens); i++ {
		switch tokens[i] {
		case "{":
			depth++
		case "}":
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func nginxDirectives(tokens []string) [][]string {
	var directives [][]string
	var current []string
	for _, token := range tokens {
		switch token {
		case "{", "}":
			current = nil
		case ";":
			if len(current) > 0 {
				directives = append(directives, current)
			}
			current = nil
		default:
			current = append(current, token)
		}
	}
	return directives
}

func applyNginxDirective(metadata *nginxSiteMetadata, directive []string, domainSeen map[string]struct{}) {
	if len(directive) < 2 {
		return
	}
	name, values := strings.ToLower(directive[0]), directive[1:]
	switch name {
	case "server_name":
		for _, domain := range values {
			if !validNginxServerName(domain) {
				continue
			}
			if _, exists := domainSeen[domain]; exists {
				continue
			}
			domainSeen[domain] = struct{}{}
			metadata.Domains = append(metadata.Domains, domain)
		}
	case "listen":
		port, ssl := nginxListen(values)
		if ssl {
			metadata.SSL = true
			if metadata.HTTPSPort == 0 {
				metadata.HTTPSPort = port
			}
		} else if metadata.HTTPPort == 0 {
			metadata.HTTPPort = port
		}
	case "root":
		setNginxPath(&metadata.Root, values[0], "root", metadata)
	case "proxy_pass":
		value := values[0]
		if metadata.ProxyPass == "" && !strings.Contains(value, "$") &&
			(strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://")) {
			metadata.ProxyPass = value
		} else if metadata.ProxyPass == "" {
			metadata.Warnings = append(metadata.Warnings, "proxy_pass 使用了无法可靠识别的值")
		}
	case "access_log":
		setNginxPath(&metadata.AccessLogPath, values[0], "access_log", metadata)
	case "error_log":
		setNginxPath(&metadata.ErrorLogPath, values[0], "error_log", metadata)
	case "ssl_certificate":
		setNginxPath(&metadata.CertPath, values[0], "ssl_certificate", metadata)
	case "ssl_certificate_key":
		setNginxPath(&metadata.KeyPath, values[0], "ssl_certificate_key", metadata)
	}
}

func validNginxServerName(value string) bool {
	if value == "" || value == "_" || strings.HasPrefix(value, "$") || strings.HasPrefix(value, "~") {
		return false
	}
	if strings.ContainsAny(value, " /{};") {
		return false
	}
	return net.ParseIP(value) != nil || strings.Contains(value, ".")
}

func nginxListen(values []string) (int, bool) {
	ssl := false
	for _, value := range values {
		if strings.EqualFold(value, "ssl") {
			ssl = true
		}
	}
	port := 0
	if len(values) > 0 && !strings.HasPrefix(values[0], "unix:") {
		address := values[0]
		if parsed, err := strconv.Atoi(address); err == nil {
			port = parsed
		} else if _, value, err := net.SplitHostPort(address); err == nil {
			port, _ = strconv.Atoi(value)
		} else if index := strings.LastIndex(address, ":"); index >= 0 {
			port, _ = strconv.Atoi(address[index+1:])
		}
	}
	if port == 0 {
		if ssl {
			port = 443
		} else {
			port = 80
		}
	}
	return port, ssl
}

func setNginxPath(target *string, value, directive string, metadata *nginxSiteMetadata) {
	if strings.Contains(value, "$") || strings.HasPrefix(value, "syslog:") || !filepath.IsAbs(value) {
		metadata.Warnings = append(metadata.Warnings, directive+" 使用了变量、相对路径或非文件目标")
		return
	}
	clean := filepath.Clean(value)
	if *target == "" {
		*target = clean
		return
	}
	if *target != clean {
		metadata.Warnings = append(metadata.Warnings, directive+" 存在多个不同值，已采用第一个")
	}
}

func applyNginxSiteMetadata(site *model.Website, metadata nginxSiteMetadata) {
	site.PrimaryDomain = metadata.PrimaryDomain
	site.Domains = ""
	if len(metadata.Domains) > 1 {
		site.Domains = strings.Join(metadata.Domains[1:], ",")
	}
	site.Type = metadata.Type
	site.SiteDir = metadata.Root
	site.ProxyPass = metadata.ProxyPass
	site.HttpPort = metadata.HTTPPort
	site.HttpsPort = metadata.HTTPSPort
	site.SSLEnable = metadata.SSL
	site.AccessLogPath = metadata.AccessLogPath
	site.ErrorLogPath = metadata.ErrorLogPath
}

func collectActiveNginxConfigPaths(mainConf string) (map[string]struct{}, error) {
	canonicalMain, err := canonicalNginxConfigFile(mainConf)
	if err != nil {
		return nil, err
	}
	confPrefix := filepath.Dir(canonicalMain)
	active := make(map[string]struct{})
	visited := make(map[string]struct{})
	var visit func(string) error
	visit = func(path string) error {
		canonical, err := canonicalNginxConfigFile(path)
		if err != nil {
			return err
		}
		if _, ok := visited[canonical]; ok {
			return nil
		}
		visited[canonical] = struct{}{}
		active[canonical] = struct{}{}

		content, err := os.ReadFile(canonical)
		if err != nil {
			return fmt.Errorf("读取 Nginx 配置 %s: %w", canonical, err)
		}
		for _, pattern := range parseIncludeLines(string(content), confPrefix) {
			matches, err := filepath.Glob(pattern)
			if err != nil {
				return fmt.Errorf("解析 Nginx include %s: %w", pattern, err)
			}
			sort.Strings(matches)
			for _, match := range matches {
				info, err := os.Stat(match)
				if err != nil || !info.Mode().IsRegular() {
					continue
				}
				if err := visit(match); err != nil {
					return err
				}
			}
		}
		return nil
	}
	if err := visit(canonicalMain); err != nil {
		return nil, err
	}
	return active, nil
}

func validateExternalNginxConfigPath(rawPath, mainConf string) (string, error) {
	if !filepath.IsAbs(rawPath) {
		return "", buserr.WithDetail(constant.ErrWebsiteExternalConfigInvalid, "必须使用绝对路径", nil)
	}
	canonical, err := canonicalNginxConfigFile(rawPath)
	if err != nil {
		return "", buserr.WithDetail(constant.ErrWebsiteExternalConfigInvalid, filepath.Clean(rawPath), err)
	}
	active, err := collectActiveNginxConfigPaths(mainConf)
	if err != nil {
		return "", buserr.WithDetail(constant.ErrWebsiteExternalConfigInvalid, filepath.Clean(mainConf), err)
	}
	if _, ok := active[canonical]; !ok {
		return "", buserr.WithDetail(constant.ErrWebsiteExternalConfigInactive, canonical, nil)
	}
	return canonical, nil
}

func canonicalNginxConfigFile(path string) (string, error) {
	canonical, err := filepath.EvalSymlinks(filepath.Clean(path))
	if err != nil {
		return "", err
	}
	info, err := os.Stat(canonical)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("不是普通文件")
	}
	return filepath.Clean(canonical), nil
}

func contentSHA256(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

func (s *WebsiteService) InspectExternalNginxSite(path string) (*dto.ExternalNginxSitePreview, error) {
	canonical, metadata, err := s.inspectExternalNginxSite(path)
	if err != nil {
		return nil, err
	}
	return &dto.ExternalNginxSitePreview{
		Path:          canonical,
		PrimaryDomain: metadata.PrimaryDomain,
		Domains:       metadata.Domains,
		Type:          metadata.Type,
		Root:          metadata.Root,
		ProxyPass:     metadata.ProxyPass,
		HTTPPort:      metadata.HTTPPort,
		HTTPSPort:     metadata.HTTPSPort,
		SSL:           metadata.SSL,
		AccessLogPath: metadata.AccessLogPath,
		ErrorLogPath:  metadata.ErrorLogPath,
		CertPath:      metadata.CertPath,
		KeyPath:       metadata.KeyPath,
		Warnings:      metadata.Warnings,
	}, nil
}

func (s *WebsiteService) CreateExternalNginxSite(req dto.ExternalNginxSiteCreateReq) (*model.Website, error) {
	canonical, metadata, err := s.inspectExternalNginxSite(req.Path)
	if err != nil {
		return nil, err
	}
	if existing, _ := s.websiteRepo.Get(repo.WithByNginxConfPath(canonical)); existing.ID > 0 {
		return nil, buserr.New(constant.ErrWebsiteExternalConfigDuplicate)
	}
	if err := s.ensureWebsitePrimaryDomainAvailable(metadata.PrimaryDomain, 0); err != nil {
		return nil, err
	}

	alias := strings.TrimSpace(req.Alias)
	if alias == "" {
		alias = domainToAlias(metadata.PrimaryDomain)
	}
	if existing, _ := s.websiteRepo.Get(repo.WithByAlias(alias)); existing.ID > 0 {
		return nil, buserr.WithDetail(constant.ErrRecordExist, "alias already exists: "+alias, nil)
	}

	site := model.Website{
		Alias:           alias,
		Remark:          req.Remark,
		Status:          "running",
		ConfigMode:      "source",
		NginxConfPath:   canonical,
		IndexFile:       "index.html index.htm",
		HttpConfig:      "HTTPSRedirect",
		SSLProtocols:    "TLSv1.2 TLSv1.3",
		AccessLog:       metadata.AccessLogPath != "",
		ErrorLog:        metadata.ErrorLogPath != "",
		GzipEnable:      true,
		SecurityHeaders: true,
	}
	applyNginxSiteMetadata(&site, metadata)
	site.CertificateID = s.matchPanelCertificate(metadata.CertPath, metadata.KeyPath)
	if err := s.websiteRepo.Create(&site); err != nil {
		return nil, buserr.WithDetail(constant.ErrInternalServer, err.Error(), err)
	}
	return &site, nil
}

func (s *WebsiteService) RefreshExternalNginxSite(id uint) (*model.Website, error) {
	site, err := s.websiteRepo.Get(repo.WithByID(id))
	if err != nil {
		return nil, buserr.New(constant.ErrRecordNotFound)
	}
	if site.NginxConfPath == "" {
		return nil, buserr.New(constant.ErrWebsiteExternalOperationDenied)
	}
	canonical, metadata, err := s.inspectExternalNginxSite(site.NginxConfPath)
	if err != nil {
		return nil, err
	}
	if err := s.ensureWebsitePrimaryDomainAvailable(metadata.PrimaryDomain, site.ID); err != nil {
		return nil, err
	}

	applyNginxSiteMetadata(&site, metadata)
	site.NginxConfPath = canonical
	site.ConfigMode = "source"
	site.Status = "running"
	site.AccessLog = metadata.AccessLogPath != ""
	site.ErrorLog = metadata.ErrorLogPath != ""
	site.CertificateID = s.matchPanelCertificate(metadata.CertPath, metadata.KeyPath)
	if err := s.websiteRepo.Save(&site); err != nil {
		return nil, buserr.WithDetail(constant.ErrInternalServer, err.Error(), err)
	}
	return &site, nil
}

func (s *WebsiteService) ensureWebsitePrimaryDomainAvailable(primaryDomain string, exceptID uint) error {
	existing, _ := s.websiteRepo.Get(repo.WithByPrimaryDomain(primaryDomain))
	if existing.ID > 0 && existing.ID != exceptID {
		return buserr.New(constant.ErrWebsiteDomainExist)
	}
	return nil
}

func (s *WebsiteService) synchronizeSourceSiteMetadata(site model.Website) model.Website {
	if site.ConfigMode != "source" {
		return site
	}
	confPath, err := s.resolveSiteConfPath(site)
	if err != nil {
		return site
	}
	content, err := os.ReadFile(confPath)
	if err != nil {
		return site
	}
	metadata, err := parseNginxSiteMetadata(string(content))
	if err != nil || s.ensureWebsitePrimaryDomainAvailable(metadata.PrimaryDomain, site.ID) != nil {
		return site
	}

	updated := site
	applyNginxSiteMetadata(&updated, metadata)
	updated.ConfigMode = "source"
	updated.AccessLog = metadata.AccessLogPath != ""
	updated.ErrorLog = metadata.ErrorLogPath != ""
	updated.CertificateID = s.matchPanelCertificate(metadata.CertPath, metadata.KeyPath)
	if updated.NginxConfPath != "" {
		updated.NginxConfPath = confPath
		updated.Status = "running"
	}
	if err := s.websiteRepo.Save(&updated); err != nil {
		global.LOG.Warnf("Synchronize source website %d metadata failed: %v", site.ID, err)
		return site
	}
	return updated
}

func (s *WebsiteService) inspectExternalNginxSite(path string) (string, nginxSiteMetadata, error) {
	canonical, err := validateExternalNginxConfigPath(path, global.CONF.Nginx.GetMainConf())
	if err != nil {
		return "", nginxSiteMetadata{}, err
	}
	content, err := os.ReadFile(canonical)
	if err != nil {
		return "", nginxSiteMetadata{}, buserr.WithDetail(constant.ErrWebsiteExternalConfigInvalid, canonical, err)
	}
	metadata, err := parseNginxSiteMetadata(string(content))
	if err != nil {
		return "", nginxSiteMetadata{}, buserr.WithDetail(constant.ErrWebsiteExternalConfigInvalid, canonical, err)
	}
	return canonical, metadata, nil
}

func (s *WebsiteService) matchPanelCertificate(certPath, keyPath string) uint {
	if certPath == "" || keyPath == "" {
		return 0
	}
	canonicalCert, err := canonicalNginxConfigFile(certPath)
	if err != nil {
		return 0
	}
	canonicalKey, err := canonicalNginxConfigFile(keyPath)
	if err != nil {
		return 0
	}
	certificates, err := s.certRepo.GetList()
	if err != nil || len(certificates) == 0 {
		return 0
	}
	sslDir := NewICertificateService().GetSSLDir()
	for _, certificate := range certificates {
		managedCert, managedKey := existingCertFilePaths(sslDir, certificate)
		managedCert, certErr := canonicalNginxConfigFile(managedCert)
		managedKey, keyErr := canonicalNginxConfigFile(managedKey)
		if certErr == nil && keyErr == nil && managedCert == canonicalCert && managedKey == canonicalKey {
			return certificate.ID
		}
	}
	return 0
}
