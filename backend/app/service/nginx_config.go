package service

import (
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"xpanel/app/model"
	"xpanel/app/repo"
	"xpanel/global"
)

type NginxConfigGenerator struct {
	certRepo    repo.ICertificateRepo
	settingRepo repo.ISettingRepo
}

func NewNginxConfigGenerator() *NginxConfigGenerator {
	return &NginxConfigGenerator{
		certRepo:    repo.NewICertificateRepo(),
		settingRepo: repo.NewISettingRepo(),
	}
}

// Generate 生成网站的完整 Nginx server block 配置
func (g *NginxConfigGenerator) Generate(site model.Website) (string, error) {
	var b strings.Builder

	domains := g.collectDomains(site)
	serverName := strings.Join(domains, " ")

	hasSSL := site.SSLEnable && site.CertificateID > 0
	var certPath, keyPath string
	if hasSSL {
		var err error
		certPath, keyPath, err = g.getCertPaths(site.CertificateID)
		if err != nil {
			return "", fmt.Errorf("获取证书路径失败: %v", err)
		}
	}

	needsHTTPRedirect := hasSSL && site.HttpConfig == "HTTPSRedirect"
	needsHTTPBlock := !hasSSL || site.HttpConfig == "httpOnly" || site.HttpConfig == "HTTPAlso" || site.HttpConfig == "HTTPSRedirect"
	needsHTTPSBlock := hasSSL && site.HttpConfig != "httpOnly"

	// Upstream block (before server blocks)
	if site.Upstream != "" {
		for _, line := range strings.Split(site.Upstream, "\n") {
			if line != "" {
				b.WriteString(line + "\n")
			}
		}
		b.WriteString("\n")
	}

	// HTTP -> HTTPS redirect server block
	if needsHTTPRedirect {
		b.WriteString("server {\n")
		g.writeListenHTTP(&b, site)
		fmt.Fprintf(&b, "    server_name %s;\n", serverName)
		b.WriteString("    return 301 https://$host$request_uri;\n")
		b.WriteString("}\n\n")
	}

	// HTTP-only server block (when no SSL, or HTTPAlso)
	if needsHTTPBlock && !needsHTTPRedirect {
		b.WriteString("server {\n")
		g.writeListenHTTP(&b, site)
		fmt.Fprintf(&b, "    server_name %s;\n", serverName)
		b.WriteString("\n")
		g.writeServerBody(&b, site, false, "", "")
		b.WriteString("}\n")
		if needsHTTPSBlock {
			b.WriteString("\n")
		}
	}

	// HTTPS server block
	if needsHTTPSBlock {
		b.WriteString("server {\n")
		g.writeListenHTTPS(&b, site)
		fmt.Fprintf(&b, "    server_name %s;\n", serverName)
		b.WriteString("\n")
		g.writeSSLBlock(&b, site, certPath, keyPath)
		g.writeServerBody(&b, site, true, certPath, keyPath)
		b.WriteString("}\n")
	}

	return b.String(), nil
}

func (g *NginxConfigGenerator) collectDomains(site model.Website) []string {
	domains := []string{site.PrimaryDomain}
	if site.Domains != "" {
		for _, d := range strings.Split(site.Domains, ",") {
			d = strings.TrimSpace(d)
			if d != "" && d != site.PrimaryDomain {
				domains = append(domains, d)
			}
		}
	}
	return domains
}

func (g *NginxConfigGenerator) writeListenHTTP(b *strings.Builder, site model.Website) {
	defaultStr := ""
	if site.DefaultServer {
		defaultStr = " default_server"
	}
	fmt.Fprintf(b, "    listen 80%s;\n", defaultStr)
	fmt.Fprintf(b, "    listen [::]:80%s;\n", defaultStr)
}

func (g *NginxConfigGenerator) writeListenHTTPS(b *strings.Builder, site model.Website) {
	defaultStr := ""
	if site.DefaultServer {
		defaultStr = " default_server"
	}
	fmt.Fprintf(b, "    listen 443 ssl%s;\n", defaultStr)
	fmt.Fprintf(b, "    listen [::]:443 ssl%s;\n", defaultStr)
}

// certHasOCSP 检查证书是否包含 OCSP responder URL
func (g *NginxConfigGenerator) certHasOCSP(certID uint) bool {
	if certID == 0 {
		return false
	}
	cert, err := g.certRepo.Get(repo.WithByID(certID))
	if err != nil || cert.Pem == "" {
		return false
	}
	block, _ := pem.Decode([]byte(cert.Pem))
	if block == nil {
		return false
	}
	x509Cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return false
	}
	return len(x509Cert.OCSPServer) > 0
}

// customHasLocationRoot 检查自定义配置中是否包含 location / 块
func customHasLocationRoot(customNginx string) bool {
	for _, line := range strings.Split(customNginx, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "location / {" || trimmed == "location /{" ||
			strings.HasPrefix(trimmed, "location / {") || strings.HasPrefix(trimmed, "location /{") {
			return true
		}
	}
	return false
}

// customHasDirective 检查自定义配置中是否已包含某个 Nginx 指令
func customHasDirective(customNginx, directive string) bool {
	for _, line := range strings.Split(customNginx, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, directive+" ") || strings.HasPrefix(trimmed, directive+"\t") {
			return true
		}
	}
	return false
}

func (g *NginxConfigGenerator) writeSSLBlock(b *strings.Builder, site model.Website, certPath, keyPath string) {
	custom := site.CustomNginx

	if site.Http2Enable && !customHasDirective(custom, "http2") {
		b.WriteString("    http2 on;\n")
	}

	if !customHasDirective(custom, "ssl_certificate_key") {
		fmt.Fprintf(b, "    ssl_certificate %s;\n", certPath)
		fmt.Fprintf(b, "    ssl_certificate_key %s;\n", keyPath)
	}

	if !customHasDirective(custom, "ssl_protocols") {
		protocols := site.SSLProtocols
		if protocols == "" {
			protocols = "TLSv1.2 TLSv1.3"
		}
		fmt.Fprintf(b, "    ssl_protocols %s;\n", protocols)
	}

	if !customHasDirective(custom, "ssl_ciphers") {
		b.WriteString("    ssl_ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384:ECDHE-ECDSA-CHACHA20-POLY1305:ECDHE-RSA-CHACHA20-POLY1305:DHE-RSA-AES128-GCM-SHA256:DHE-RSA-AES256-GCM-SHA384:DHE-RSA-CHACHA20-POLY1305;\n")
		b.WriteString("    ssl_prefer_server_ciphers off;\n")
	}

	if !customHasDirective(custom, "ssl_ecdh_curve") {
		b.WriteString("    ssl_ecdh_curve X25519:prime256v1:secp384r1;\n")
	}

	if !customHasDirective(custom, "ssl_session_cache") {
		b.WriteString("    ssl_session_cache shared:MozSSL:10m;\n")
	}
	if !customHasDirective(custom, "ssl_session_timeout") {
		b.WriteString("    ssl_session_timeout 1d;\n")
	}
	if !customHasDirective(custom, "ssl_session_tickets") {
		b.WriteString("    ssl_session_tickets off;\n")
	}

	if !customHasDirective(custom, "ssl_stapling") && g.certHasOCSP(site.CertificateID) {
		b.WriteString("    ssl_stapling on;\n")
		b.WriteString("    ssl_stapling_verify on;\n")
		fmt.Fprintf(b, "    ssl_trusted_certificate %s;\n", certPath)
	}

	if !customHasDirective(custom, "resolver") {
		b.WriteString("    resolver 1.1.1.1 8.8.8.8 valid=300s;\n")
		b.WriteString("    resolver_timeout 5s;\n")
	}

	if site.HSTS && !customHasDirective(custom, "add_header Strict-Transport-Security") {
		b.WriteString("    add_header Strict-Transport-Security \"max-age=63072000; includeSubDomains; preload\" always;\n")
	}
	b.WriteString("\n")
}

// writeGzipBlock 生成 Gzip 压缩配置
func (g *NginxConfigGenerator) writeGzipBlock(b *strings.Builder, site model.Website) {
	if !site.GzipEnable {
		return
	}
	custom := site.CustomNginx
	if customHasDirective(custom, "gzip") {
		return
	}
	b.WriteString("    # Gzip Compression\n")
	b.WriteString("    gzip on;\n")
	b.WriteString("    gzip_vary on;\n")
	b.WriteString("    gzip_proxied any;\n")
	b.WriteString("    gzip_comp_level 6;\n")
	b.WriteString("    gzip_min_length 256;\n")
	b.WriteString("    gzip_buffers 16 8k;\n")
	b.WriteString("    gzip_http_version 1.1;\n")
	b.WriteString("    gzip_types text/plain text/css text/csv text/javascript text/xml application/javascript application/x-javascript application/json application/xml application/xml+rss application/atom+xml application/rss+xml application/xhtml+xml application/manifest+json application/vnd.ms-fontobject font/opentype font/ttf font/otf image/svg+xml image/x-icon;\n")
	b.WriteString("\n")
}

// writeSecurityHeaders 生成安全响应头
func (g *NginxConfigGenerator) writeSecurityHeaders(b *strings.Builder, site model.Website) {
	if !site.SecurityHeaders {
		return
	}
	custom := site.CustomNginx
	b.WriteString("    # Security Headers\n")
	if !customHasDirective(custom, "add_header X-Content-Type-Options") {
		b.WriteString("    add_header X-Content-Type-Options \"nosniff\" always;\n")
	}
	if !customHasDirective(custom, "add_header X-Frame-Options") {
		b.WriteString("    add_header X-Frame-Options \"SAMEORIGIN\" always;\n")
	}
	if !customHasDirective(custom, "add_header Referrer-Policy") {
		b.WriteString("    add_header Referrer-Policy \"strict-origin-when-cross-origin\" always;\n")
	}
	if !customHasDirective(custom, "add_header Permissions-Policy") {
		b.WriteString("    add_header Permissions-Policy \"camera=(), microphone=(), geolocation=()\" always;\n")
	}
	if !customHasDirective(custom, "server_tokens") {
		b.WriteString("    server_tokens off;\n")
	}
	b.WriteString("\n")
}

// writeStaticCacheBlock 生成静态资源缓存 location
func (g *NginxConfigGenerator) writeStaticCacheBlock(b *strings.Builder, site model.Website) {
	if !site.StaticCacheEnable {
		return
	}
	custom := site.CustomNginx
	if strings.Contains(custom, "~* \\.(jpg|") || strings.Contains(custom, "~* \\.(css|") {
		return
	}
	b.WriteString("    # Static file caching\n")
	b.WriteString("    location ~* \\.(jpg|jpeg|png|gif|ico|webp|avif|svg|svgz)$ {\n")
	if site.Type == "static" {
		b.WriteString("        expires 30d;\n")
	} else {
		b.WriteString("        proxy_pass " + site.ProxyPass + ";\n")
		b.WriteString("        expires 30d;\n")
	}
	b.WriteString("        add_header Cache-Control \"public, immutable\";\n")
	b.WriteString("        access_log off;\n")
	b.WriteString("    }\n\n")

	b.WriteString("    location ~* \\.(css|js|woff|woff2|ttf|otf|eot)$ {\n")
	if site.Type == "static" {
		b.WriteString("        expires 7d;\n")
	} else {
		b.WriteString("        proxy_pass " + site.ProxyPass + ";\n")
		b.WriteString("        expires 7d;\n")
	}
	b.WriteString("        add_header Cache-Control \"public\";\n")
	b.WriteString("        access_log off;\n")
	b.WriteString("    }\n\n")
}

func (g *NginxConfigGenerator) writeServerBody(b *strings.Builder, site model.Website, isHTTPS bool, certPath, keyPath string) {
	logDir := g.getSiteLogDir()
	if site.AccessLog {
		fmt.Fprintf(b, "    access_log %s/%s.access.log;\n", logDir, site.PrimaryDomain)
	} else {
		b.WriteString("    access_log off;\n")
	}
	if site.ErrorLog {
		fmt.Fprintf(b, "    error_log %s/%s.error.log;\n", logDir, site.PrimaryDomain)
	} else {
		b.WriteString("    error_log /dev/null;\n")
	}
	b.WriteString("\n")

	// Basic auth
	if site.BasicAuth && site.BasicUser != "" {
		htpasswdPath := g.getHtpasswdPath(site)
		fmt.Fprintf(b, "    auth_basic \"Restricted\";\n")
		fmt.Fprintf(b, "    auth_basic_user_file %s;\n", htpasswdPath)
		b.WriteString("\n")
	}

	// Anti-hotlink
	if site.AntiLeech {
		referers := "none blocked server_names"
		if site.LeechReferers != "" {
			referers += " " + strings.ReplaceAll(site.LeechReferers, ",", " ")
		}
		fmt.Fprintf(b, "    valid_referers %s;\n", referers)
		b.WriteString("    if ($invalid_referer) {\n")
		b.WriteString("        return 403;\n")
		b.WriteString("    }\n\n")
	}

	// Traffic limits
	if site.LimitRate != "" {
		fmt.Fprintf(b, "    limit_rate %s;\n", site.LimitRate)
	}
	if site.LimitConn > 0 {
		fmt.Fprintf(b, "    limit_conn perip %d;\n", site.LimitConn)
	}
	if site.LimitRate != "" || site.LimitConn > 0 {
		b.WriteString("\n")
	}

	// Redirects
	if site.Redirects != "" && site.Redirects != "[]" {
		b.WriteString("    # Redirects\n")
		b.WriteString(g.generateRedirects(site.Redirects))
		b.WriteString("\n")
	}

	// Gzip compression
	g.writeGzipBlock(b, site)

	// Security headers
	g.writeSecurityHeaders(b, site)

	// Site type specific config (skip managed location / if custom config defines one)
	hasCustomRootLocation := customHasLocationRoot(site.CustomNginx)
	if !hasCustomRootLocation {
		if site.ProxyPass != "" {
			g.writeReverseProxy(b, site)
		} else if site.Type == "static" {
			g.writeStaticSite(b, site)
		}
	}

	// Static file caching (after site-specific config)
	g.writeStaticCacheBlock(b, site)

	// Rewrite rules
	if site.Rewrite != "" {
		b.WriteString("    # Rewrite rules\n")
		for _, line := range strings.Split(site.Rewrite, "\n") {
			line = strings.TrimSpace(line)
			if line != "" {
				fmt.Fprintf(b, "    %s\n", line)
			}
		}
		b.WriteString("\n")
	}

	// Custom nginx directives
	if site.CustomNginx != "" {
		b.WriteString("    # Custom directives\n")
		for _, line := range strings.Split(site.CustomNginx, "\n") {
			if line != "" {
				fmt.Fprintf(b, "    %s\n", line)
			}
		}
		b.WriteString("\n")
	}
}

func (g *NginxConfigGenerator) writeStaticSite(b *strings.Builder, site model.Website) {
	siteDir := site.SiteDir
	if siteDir == "" {
		siteDir = fmt.Sprintf("/var/www/%s", site.PrimaryDomain)
	}
	indexFile := site.IndexFile
	if indexFile == "" {
		indexFile = "index.html index.htm"
	}
	fmt.Fprintf(b, "    root %s;\n", siteDir)
	fmt.Fprintf(b, "    index %s;\n", indexFile)
	b.WriteString("\n")
	b.WriteString("    location / {\n")
	b.WriteString("        try_files $uri $uri/ =404;\n")
	b.WriteString("    }\n\n")
}

func (g *NginxConfigGenerator) writeReverseProxy(b *strings.Builder, site model.Website) {
	if site.ProxyPass == "" {
		return
	}
	b.WriteString("    location / {\n")
	fmt.Fprintf(b, "        proxy_pass %s;\n", site.ProxyPass)
	b.WriteString("        proxy_set_header Host $host;\n")
	b.WriteString("        proxy_set_header X-Real-IP $remote_addr;\n")
	b.WriteString("        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;\n")
	b.WriteString("        proxy_set_header X-Forwarded-Proto $scheme;\n")
	b.WriteString("        proxy_http_version 1.1;\n")
	if site.WebSocket {
		b.WriteString("        proxy_set_header Upgrade $http_upgrade;\n")
		b.WriteString("        proxy_set_header Connection \"upgrade\";\n")
		b.WriteString("        proxy_read_timeout 86400s;\n")
		b.WriteString("        proxy_send_timeout 86400s;\n")
	} else {
		b.WriteString("        proxy_connect_timeout 60s;\n")
		b.WriteString("        proxy_send_timeout 60s;\n")
		b.WriteString("        proxy_read_timeout 600s;\n")
		b.WriteString("        proxy_buffering on;\n")
		b.WriteString("        proxy_buffer_size 8k;\n")
		b.WriteString("        proxy_buffers 8 8k;\n")
		b.WriteString("        proxy_busy_buffers_size 16k;\n")
	}
	b.WriteString("    }\n\n")
}

func (g *NginxConfigGenerator) generateRedirects(redirectsJSON string) string {
	// 简单解析 JSON 格式的重定向规则，生成 Nginx location 块
	// 格式: [{"source":"/old","target":"https://new.com/path","type":301}]
	// 这里使用简单的字符串处理，避免引入 JSON 解析
	var b strings.Builder
	redirectsJSON = strings.TrimSpace(redirectsJSON)
	if redirectsJSON == "" || redirectsJSON == "[]" {
		return ""
	}
	// 使用 JSON 解析
	type redirect struct {
		Source string `json:"source"`
		Target string `json:"target"`
		Type   int    `json:"type"`
	}
	var redirects []redirect
	if err := json.Unmarshal([]byte(redirectsJSON), &redirects); err != nil {
		return ""
	}
	for _, r := range redirects {
		if r.Source == "" || r.Target == "" {
			continue
		}
		code := r.Type
		if code == 0 {
			code = 301
		}
		fmt.Fprintf(&b, "    location = %s {\n", r.Source)
		fmt.Fprintf(&b, "        return %d %s;\n", code, r.Target)
		b.WriteString("    }\n")
	}
	return b.String()
}

func (g *NginxConfigGenerator) getCertPaths(certID uint) (string, string, error) {
	cert, err := g.certRepo.Get(repo.WithByID(certID))
	if err != nil {
		return "", "", err
	}
	sslDir := g.getSSLDir()
	certPath := filepath.Join(sslDir, "certs", cert.PrimaryDomain, "fullchain.pem")
	keyPath := filepath.Join(sslDir, "certs", cert.PrimaryDomain, "privkey.pem")

	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		return "", "", fmt.Errorf("证书文件不存在: %s", certPath)
	}
	return certPath, keyPath, nil
}

func (g *NginxConfigGenerator) getSSLDir() string {
	dir, err := g.settingRepo.GetValueByKey("SSLDir")
	if err != nil || dir == "" {
		return global.CONF.GetDefaultSSLDir()
	}
	return dir
}

func (g *NginxConfigGenerator) getSiteLogDir() string {
	logDir := filepath.Join(global.CONF.Nginx.GetLogDir(), "sites")
	os.MkdirAll(logDir, 0755)
	return logDir
}

func (g *NginxConfigGenerator) getHtpasswdPath(site model.Website) string {
	authDir := filepath.Join(global.CONF.Nginx.GetConfDir(), "auth")
	os.MkdirAll(authDir, 0755)
	return filepath.Join(authDir, site.Alias+".htpasswd")
}

// GetSiteConfPath 获取网站配置文件路径
func GetSiteConfPath(alias string) string {
	return filepath.Join(global.CONF.Nginx.GetSitesDir(), alias+".conf")
}

// EnsureNginxInclude 确保 nginx.conf 包含站点配置目录
func EnsureNginxInclude() error {
	nc := global.CONF.Nginx

	// System mode: Debian/Ubuntu nginx already includes sites-enabled/*
	if nc.IsSystemMode() {
		mainConf := nc.GetMainConf()
		data, err := os.ReadFile(mainConf)
		if err != nil {
			return nil
		}
		content := string(data)
		if strings.Contains(content, "sites-enabled") {
			// Ensure directories exist
			os.MkdirAll(nc.GetSitesAvailableDir(), 0755)
			os.MkdirAll(nc.GetSitesDir(), 0755)
			return nil
		}
		// If sites-enabled not included, add it
		return insertNginxInclude(mainConf, content, "include /etc/nginx/sites-enabled/*;")
	}

	// Prefix mode
	mainConf := nc.GetMainConf()
	data, err := os.ReadFile(mainConf)
	if err != nil {
		return err
	}
	content := string(data)
	if strings.Contains(content, "conf.d/*.conf") || strings.Contains(content, "conf/conf.d/*.conf") {
		return nil
	}
	return insertNginxInclude(mainConf, content, "include conf.d/*.conf;")
}

func insertNginxInclude(mainConf, content, includeLine string) error {
	httpIdx := strings.Index(content, "http {")
	if httpIdx < 0 {
		httpIdx = strings.Index(content, "http{")
	}
	if httpIdx < 0 {
		return fmt.Errorf("nginx.conf missing http block")
	}

	depth := 0
	insertPos := -1
	for i := httpIdx; i < len(content); i++ {
		if content[i] == '{' {
			depth++
		} else if content[i] == '}' {
			depth--
			if depth == 0 {
				insertPos = i
				break
			}
		}
	}
	if insertPos < 0 {
		return fmt.Errorf("nginx.conf malformed")
	}

	newContent := content[:insertPos] + "\n    " + includeLine + "\n" + content[insertPos:]
	return os.WriteFile(mainConf, []byte(newContent), 0644)
}
