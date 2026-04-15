package service

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"xpanel/app/dto"
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
		ServerPort:       global.CONF.System.Port,
		AgentToken:       settingMap["AgentToken"],
		AutoUpgrade:      settingMap["AutoUpgrade"],
		AppearanceConfig: settingMap["AppearanceConfig"],
		ProxyEnable:      settingMap["ProxyEnable"],
		ProxyAddress:     settingMap["ProxyAddress"],
		ProxyNoProxy:     settingMap["ProxyNoProxy"],
	}, nil
}

func (s *SettingService) Update(req dto.SettingUpdate) error {
	allowedKeys := map[string]bool{
		"Language": true, "SessionTimeout": true,
		"PanelName": true, "Theme": true,
		"SecurityEntrance": true, "GitHubToken": true,
		"UserName": true, "AgentToken": true,
		"AutoUpgrade": true, "AppearanceConfig": true,
		"ProxyEnable": true, "ProxyAddress": true, "ProxyNoProxy": true,
	}
	if !allowedKeys[req.Key] {
		return buserr.New(constant.ErrInvalidParams)
	}

	if err := settingRepo.Update(req.Key, req.Value); err != nil {
		return err
	}

	if req.Key == "ProxyEnable" || req.Key == "ProxyAddress" || req.Key == "ProxyNoProxy" {
		go s.syncProxyToSystem()
	}
	return nil
}

func (s *SettingService) UpdatePort(req dto.PortUpdate) error {
	// 验证端口合法性
	port, err := strconv.Atoi(req.Port)
	if err != nil || port < 1 || port > 65535 {
		return buserr.New(constant.ErrInvalidParams)
	}

	// 更新 Viper 配置并写入文件
	if global.Vp == nil {
		return fmt.Errorf("viper instance not initialized")
	}
	global.Vp.Set("system.port", req.Port)
	if err := global.Vp.WriteConfig(); err != nil {
		return fmt.Errorf("failed to write config: %v", err)
	}

	// 更新运行时配置
	global.CONF.System.Port = req.Port
	return nil
}

func (s *SettingService) GetValueByKey(key string) (string, error) {
	return settingRepo.GetValueByKey(key)
}

const proxyProfilePath = "/etc/profile.d/xpanel-proxy.sh"

func (s *SettingService) syncProxyToSystem() {
	enable, _ := settingRepo.GetValueByKey("ProxyEnable")
	addr, _ := settingRepo.GetValueByKey("ProxyAddress")
	noProxy, _ := settingRepo.GetValueByKey("ProxyNoProxy")

	if enable != "enable" || strings.TrimSpace(addr) == "" {
		os.Remove(proxyProfilePath)
		removeProxyFromEnvironment()
		global.LOG.Info("System proxy disabled, config files removed")
		return
	}

	addr = strings.TrimSpace(addr)
	noProxy = strings.TrimSpace(noProxy)
	if noProxy == "" {
		noProxy = "localhost,127.0.0.1,::1"
	}

	// /etc/profile.d/xpanel-proxy.sh
	profileContent := fmt.Sprintf(`# Managed by X-Panel - do not edit manually
export http_proxy="%s"
export https_proxy="%s"
export HTTP_PROXY="%s"
export HTTPS_PROXY="%s"
export no_proxy="%s"
export NO_PROXY="%s"
`, addr, addr, addr, addr, noProxy, noProxy)

	if err := os.WriteFile(proxyProfilePath, []byte(profileContent), 0644); err != nil {
		global.LOG.Errorf("Failed to write proxy profile: %v", err)
		return
	}

	writeProxyToEnvironment(addr, noProxy)
	global.LOG.Infof("System proxy enabled: %s", addr)
}

func writeProxyToEnvironment(addr, noProxy string) {
	const envPath = "/etc/environment"
	content, _ := os.ReadFile(envPath)
	lines := strings.Split(string(content), "\n")

	// Remove existing proxy lines managed by us
	var cleaned []string
	for _, line := range lines {
		lower := strings.ToLower(strings.TrimSpace(line))
		if strings.HasPrefix(lower, "http_proxy=") || strings.HasPrefix(lower, "https_proxy=") ||
			strings.HasPrefix(lower, "no_proxy=") {
			continue
		}
		cleaned = append(cleaned, line)
	}

	// Remove trailing empty lines
	for len(cleaned) > 0 && strings.TrimSpace(cleaned[len(cleaned)-1]) == "" {
		cleaned = cleaned[:len(cleaned)-1]
	}

	cleaned = append(cleaned,
		fmt.Sprintf(`http_proxy="%s"`, addr),
		fmt.Sprintf(`https_proxy="%s"`, addr),
		fmt.Sprintf(`HTTP_PROXY="%s"`, addr),
		fmt.Sprintf(`HTTPS_PROXY="%s"`, addr),
		fmt.Sprintf(`no_proxy="%s"`, noProxy),
		fmt.Sprintf(`NO_PROXY="%s"`, noProxy),
		"",
	)

	if err := os.WriteFile(envPath, []byte(strings.Join(cleaned, "\n")), 0644); err != nil {
		global.LOG.Errorf("Failed to write /etc/environment: %v", err)
	}
}

func removeProxyFromEnvironment() {
	const envPath = "/etc/environment"
	content, err := os.ReadFile(envPath)
	if err != nil {
		return
	}
	lines := strings.Split(string(content), "\n")
	var cleaned []string
	for _, line := range lines {
		lower := strings.ToLower(strings.TrimSpace(line))
		if strings.HasPrefix(lower, "http_proxy=") || strings.HasPrefix(lower, "https_proxy=") ||
			strings.HasPrefix(lower, "no_proxy=") {
			continue
		}
		cleaned = append(cleaned, line)
	}
	os.WriteFile(envPath, []byte(strings.Join(cleaned, "\n")), 0644)
}

func (s *SettingService) TestProxy(req dto.ProxyTest) error {
	addr := strings.TrimSpace(req.Address)
	if addr == "" {
		return buserr.New(constant.ErrInvalidParams)
	}

	parsed, err := url.Parse(addr)
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
