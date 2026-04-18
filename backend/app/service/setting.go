package service

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"xpanel/app/dto"
	"xpanel/app/repo"
	"xpanel/buserr"
	"xpanel/constant"
	"xpanel/global"

	"golang.org/x/net/proxy"
)

// ISettingService 面板设置服务接口
type ISettingService interface {
	GetSettingInfo() (*dto.SettingInfo, error)
	Update(req dto.SettingUpdate) error
	UpdatePort(req dto.PortUpdate) error
	GetValueByKey(key string) (string, error)
	TestProxy(req dto.ProxyTest) error
	SyncProxyToSystem()
	GetPanelSSL() (*dto.PanelSSLInfo, error)
	UpdatePanelSSL(req dto.PanelSSLUpdate) error
}

// NewISettingService 创建设置服务实例
func NewISettingService() ISettingService {
	return &SettingService{}
}

type SettingService struct{}

func (s *SettingService) GetSettingInfo() (*dto.SettingInfo, error) {
	settings, err := settingRepo.GetList()
	if err != nil {
		return nil, err
	}

	settingMap := make(map[string]string)
	for _, item := range settings {
		settingMap[item.Key] = item.Value
	}

	return &dto.SettingInfo{
		UserName:         settingMap["UserName"],
		Language:         settingMap["Language"],
		SessionTimeout:   settingMap["SessionTimeout"],
		PanelName:        settingMap["PanelName"],
		Theme:            settingMap["Theme"],
		SecurityEntrance: settingMap["SecurityEntrance"],
		MFAStatus:        settingMap["MFAStatus"],
		GitHubToken:      settingMap["GitHubToken"],
		AppStoreURL:      settingMap["AppStoreURL"],
		ServerPort:       global.CONF.System.Port,
		AgentToken:       settingMap["AgentToken"],
		AutoUpgrade:      settingMap["AutoUpgrade"],
		AppearanceConfig: settingMap["AppearanceConfig"],
		ProxyEnable:      settingMap["ProxyEnable"],
		ProxyType:        settingMap["ProxyType"],
		ProxyAddress:     settingMap["ProxyAddress"],
		ProxyNoProxy:     settingMap["ProxyNoProxy"],
	}, nil
}

func (s *SettingService) Update(req dto.SettingUpdate) error {
	allowedKeys := map[string]bool{
		"Language": true, "SessionTimeout": true,
		"PanelName": true, "Theme": true,
		"SecurityEntrance": true, "GitHubToken": true,
			"AppStoreURL": true,
		"UserName": true, "AgentToken": true,
		"AutoUpgrade": true, "AppearanceConfig": true,
		"ProxyEnable": true, "ProxyType": true,
		"ProxyAddress": true, "ProxyNoProxy": true,
	}
	if !allowedKeys[req.Key] {
		return buserr.New(constant.ErrInvalidParams)
	}

	if err := settingRepo.Update(req.Key, req.Value); err != nil {
		return err
	}

	proxyKeys := map[string]bool{
		"ProxyEnable": true, "ProxyType": true,
		"ProxyAddress": true, "ProxyNoProxy": true,
	}
	if proxyKeys[req.Key] {
		s.SyncProxyToSystem()
	}
	return nil
}

func (s *SettingService) UpdatePort(req dto.PortUpdate) error {
	port, err := strconv.Atoi(req.Port)
	if err != nil || port < 1 || port > 65535 {
		return buserr.New(constant.ErrInvalidParams)
	}

	if global.Vp == nil {
		return fmt.Errorf("viper instance not initialized")
	}
	global.Vp.Set("system.port", req.Port)
	if err := global.Vp.WriteConfig(); err != nil {
		return fmt.Errorf("failed to write config: %v", err)
	}

	global.CONF.System.Port = req.Port
	return nil
}

// GetPanelSSL 返回当前 config 中的面板 TLS 配置及上次在 UI 选择的证书 ID（若有）
func (s *SettingService) GetPanelSSL() (*dto.PanelSSLInfo, error) {
	info := &dto.PanelSSLInfo{
		Enable:   global.CONF.System.SSL.Enable,
		CertPath: global.CONF.System.SSL.CertPath,
		KeyPath:  global.CONF.System.SSL.KeyPath,
	}
	idStr, err := settingRepo.GetValueByKey("PanelSSLCertificateID")
	if err == nil && idStr != "" {
		if id, e := strconv.ParseUint(idStr, 10, 64); e == nil {
			info.CertificateID = uint(id)
		}
	}
	if info.CertificateID > 0 {
		cert, err := repo.NewICertificateRepo().Get(repo.WithByID(info.CertificateID))
		if err == nil {
			info.PrimaryDomain = cert.PrimaryDomain
		}
	}
	return info, nil
}

// UpdatePanelSSL 将面板 HTTPS 证书切换为证书管理中指定记录对应落盘文件，并写回 config.yaml
func (s *SettingService) UpdatePanelSSL(req dto.PanelSSLUpdate) error {
	if req.CertificateID == 0 {
		return buserr.New(constant.ErrInvalidParams)
	}
	certPath, keyPath, err := NewICertificateService().ResolveCertFilePaths(req.CertificateID)
	if err != nil {
		return err
	}
	if global.Vp == nil {
		return fmt.Errorf("viper instance not initialized")
	}
	global.Vp.Set("system.ssl.enable", true)
	global.Vp.Set("system.ssl.cert_path", certPath)
	global.Vp.Set("system.ssl.key_path", keyPath)
	if err := global.Vp.WriteConfig(); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}
	global.CONF.System.SSL.Enable = true
	global.CONF.System.SSL.CertPath = certPath
	global.CONF.System.SSL.KeyPath = keyPath
	idStr := strconv.FormatUint(uint64(req.CertificateID), 10)
	return settingRepo.CreateOrUpdate("PanelSSLCertificateID", idStr)
}

func (s *SettingService) GetValueByKey(key string) (string, error) {
	return settingRepo.GetValueByKey(key)
}

// --- proxy file paths ---

const (
	proxyProfilePath = "/etc/profile.d/xpanel-proxy.sh"
	proxyAptPath     = "/etc/apt/apt.conf.d/99xpanel-proxy"
	proxyDockerDir   = "/etc/systemd/system/docker.service.d"
	proxyDockerPath  = "/etc/systemd/system/docker.service.d/http-proxy.conf"
	envPath          = "/etc/environment"
)

var proxyEnvKeys = []string{
	"http_proxy", "HTTP_PROXY",
	"https_proxy", "HTTPS_PROXY",
	"no_proxy", "NO_PROXY",
}

func resolveProxyURLs(proxyType, addr string) (envURL, httpURL string, hasHTTP bool) {
	switch proxyType {
	case "mix":
		host := strings.TrimPrefix(strings.TrimPrefix(addr, "http://"), "https://")
		host = strings.TrimPrefix(host, "socks5://")
		httpURL = "http://" + host
		envURL = httpURL
		hasHTTP = true
	case "http":
		if !strings.HasPrefix(addr, "http://") && !strings.HasPrefix(addr, "https://") {
			addr = "http://" + addr
		}
		envURL = addr
		httpURL = addr
		hasHTTP = true
	case "socks5":
		if !strings.HasPrefix(addr, "socks5://") {
			addr = "socks5://" + addr
		}
		envURL = addr
		httpURL = ""
		hasHTTP = false
	default:
		envURL = addr
		httpURL = addr
		hasHTTP = true
	}
	return
}

func (s *SettingService) SyncProxyToSystem() {
	enable, _ := settingRepo.GetValueByKey("ProxyEnable")
	proxyType, _ := settingRepo.GetValueByKey("ProxyType")
	addr, _ := settingRepo.GetValueByKey("ProxyAddress")
	noProxy, _ := settingRepo.GetValueByKey("ProxyNoProxy")

	if enable != "enable" || strings.TrimSpace(addr) == "" {
		disableAllProxy()
		return
	}

	addr = strings.TrimSpace(addr)
	noProxy = strings.TrimSpace(noProxy)
	if noProxy == "" {
		noProxy = "localhost,127.0.0.1,::1"
	}
	if proxyType == "" {
		proxyType = "mix"
	}

	envURL, httpURL, hasHTTP := resolveProxyURLs(proxyType, addr)

	writeProfileProxy(envURL, noProxy)
	writeEnvironmentProxy(envURL, noProxy)
	setProcessProxy(envURL, noProxy)

	if hasHTTP {
		writeAptProxy(httpURL)
		writeDockerProxy(httpURL, noProxy)
	} else {
		os.Remove(proxyAptPath)
		removeDockerProxy()
	}

	global.LOG.Infof("System proxy enabled: type=%s url=%s", proxyType, envURL)
}

func disableAllProxy() {
	os.Remove(proxyProfilePath)
	os.Remove(proxyAptPath)
	removeDockerProxy()
	removeEnvironmentProxy()
	unsetProcessProxy()
	global.LOG.Info("System proxy disabled, all config files removed")
}

func writeProfileProxy(envURL, noProxy string) {
	content := fmt.Sprintf(`# Managed by X-Panel - do not edit manually
export http_proxy="%s"
export https_proxy="%s"
export HTTP_PROXY="%s"
export HTTPS_PROXY="%s"
export no_proxy="%s"
export NO_PROXY="%s"
`, envURL, envURL, envURL, envURL, noProxy, noProxy)

	if err := os.WriteFile(proxyProfilePath, []byte(content), 0644); err != nil {
		global.LOG.Errorf("Failed to write %s: %v", proxyProfilePath, err)
	}
}

func writeEnvironmentProxy(envURL, noProxy string) {
	content, _ := os.ReadFile(envPath)
	cleaned := filterProxyLines(string(content))

	cleaned = append(cleaned,
		fmt.Sprintf(`http_proxy="%s"`, envURL),
		fmt.Sprintf(`https_proxy="%s"`, envURL),
		fmt.Sprintf(`HTTP_PROXY="%s"`, envURL),
		fmt.Sprintf(`HTTPS_PROXY="%s"`, envURL),
		fmt.Sprintf(`no_proxy="%s"`, noProxy),
		fmt.Sprintf(`NO_PROXY="%s"`, noProxy),
		"",
	)

	if err := os.WriteFile(envPath, []byte(strings.Join(cleaned, "\n")), 0644); err != nil {
		global.LOG.Errorf("Failed to write %s: %v", envPath, err)
	}
}

func removeEnvironmentProxy() {
	content, err := os.ReadFile(envPath)
	if err != nil {
		return
	}
	cleaned := filterProxyLines(string(content))
	_ = os.WriteFile(envPath, []byte(strings.Join(cleaned, "\n")), 0644)
}

func filterProxyLines(content string) []string {
	lines := strings.Split(content, "\n")
	var cleaned []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		key := strings.ToLower(strings.SplitN(trimmed, "=", 2)[0])
		if key == "http_proxy" || key == "https_proxy" || key == "no_proxy" {
			continue
		}
		cleaned = append(cleaned, line)
	}
	for len(cleaned) > 0 && strings.TrimSpace(cleaned[len(cleaned)-1]) == "" {
		cleaned = cleaned[:len(cleaned)-1]
	}
	return cleaned
}

func writeAptProxy(httpURL string) {
	content := fmt.Sprintf(`// Managed by X-Panel
Acquire::http::Proxy "%s";
Acquire::https::Proxy "%s";
`, httpURL, httpURL)

	if err := os.WriteFile(proxyAptPath, []byte(content), 0644); err != nil {
		global.LOG.Errorf("Failed to write %s: %v", proxyAptPath, err)
	}
}

func writeDockerProxy(httpURL, noProxy string) {
	if _, err := exec.LookPath("docker"); err != nil {
		return
	}

	if err := os.MkdirAll(proxyDockerDir, 0755); err != nil {
		global.LOG.Errorf("Failed to create %s: %v", proxyDockerDir, err)
		return
	}

	content := fmt.Sprintf(`# Managed by X-Panel
[Service]
Environment="HTTP_PROXY=%s"
Environment="HTTPS_PROXY=%s"
Environment="NO_PROXY=%s"
`, httpURL, httpURL, noProxy)

	if err := os.WriteFile(proxyDockerPath, []byte(content), 0644); err != nil {
		global.LOG.Errorf("Failed to write %s: %v", proxyDockerPath, err)
		return
	}

	_ = exec.Command("systemctl", "daemon-reload").Run()
}

func removeDockerProxy() {
	if _, err := os.Stat(proxyDockerPath); err != nil {
		return
	}
	os.Remove(proxyDockerPath)
	dir, _ := os.ReadDir(proxyDockerDir)
	if len(dir) == 0 {
		os.Remove(proxyDockerDir)
	}
	_ = exec.Command("systemctl", "daemon-reload").Run()
}

func setProcessProxy(envURL, noProxy string) {
	os.Setenv("http_proxy", envURL)
	os.Setenv("https_proxy", envURL)
	os.Setenv("HTTP_PROXY", envURL)
	os.Setenv("HTTPS_PROXY", envURL)
	os.Setenv("no_proxy", noProxy)
	os.Setenv("NO_PROXY", noProxy)
}

func unsetProcessProxy() {
	for _, key := range proxyEnvKeys {
		os.Unsetenv(key)
	}
}

// --- proxy test ---

func (s *SettingService) TestProxy(req dto.ProxyTest) error {
	addr := strings.TrimSpace(req.Address)
	if addr == "" {
		return buserr.New(constant.ErrInvalidParams)
	}

	proxyType, _ := settingRepo.GetValueByKey("ProxyType")
	testAddr := resolveTestAddr(proxyType, addr)

	parsed, err := url.Parse(testAddr)
	if err != nil {
		return fmt.Errorf("invalid proxy address: %v", err)
	}

	var transport *http.Transport

	switch parsed.Scheme {
	case "socks5", "socks5h":
		auth := &proxy.Auth{}
		if parsed.User != nil {
			auth.User = parsed.User.Username()
			auth.Password, _ = parsed.User.Password()
		} else {
			auth = nil
		}
		dialer, dialErr := proxy.SOCKS5("tcp", parsed.Host, auth, proxy.Direct)
		if dialErr != nil {
			return fmt.Errorf("socks5 dialer: %v", dialErr)
		}
		transport = &http.Transport{
			DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
				return dialer.Dial(network, address)
			},
		}
	case "http", "https":
		transport = &http.Transport{
			Proxy: http.ProxyURL(parsed),
		}
	default:
		return fmt.Errorf("unsupported proxy scheme: %s (use http, https, or socks5)", parsed.Scheme)
	}

	client := &http.Client{Transport: transport, Timeout: 10 * time.Second}
	resp, err := client.Get("https://www.google.com/generate_204")
	if err != nil {
		return fmt.Errorf("proxy test failed: %v", err)
	}
	resp.Body.Close()
	return nil
}

func resolveTestAddr(proxyType, addr string) string {
	if proxyType == "mix" {
		host := strings.TrimPrefix(strings.TrimPrefix(addr, "http://"), "https://")
		host = strings.TrimPrefix(host, "socks5://")
		return "http://" + host
	}
	return addr
}

// SyncProxyOnStartup is called during server boot to restore panel process proxy env.
// Only sets os.Setenv; file-level configs are already persisted.
func SyncProxyOnStartup() {
	enable, _ := settingRepo.GetValueByKey("ProxyEnable")
	proxyType, _ := settingRepo.GetValueByKey("ProxyType")
	addr, _ := settingRepo.GetValueByKey("ProxyAddress")
	noProxy, _ := settingRepo.GetValueByKey("ProxyNoProxy")

	if enable != "enable" || strings.TrimSpace(addr) == "" {
		return
	}

	addr = strings.TrimSpace(addr)
	noProxy = strings.TrimSpace(noProxy)
	if noProxy == "" {
		noProxy = "localhost,127.0.0.1,::1"
	}
	if proxyType == "" {
		proxyType = "mix"
	}

	envURL, _, _ := resolveProxyURLs(proxyType, addr)
	setProcessProxy(envURL, noProxy)
	global.LOG.Infof("Restored panel proxy on startup: type=%s url=%s", proxyType, envURL)
}

