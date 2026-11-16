package service

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/http/httpproxy"
	"gorm.io/gorm"

	"xpanel/app/dto"
	"xpanel/buserr"
	"xpanel/constant"
	"xpanel/global"
)

const defaultProxyBypass = "localhost,127.0.0.1,::1"

var proxySettingsMu sync.Mutex

func readProxySettings() (map[string]string, error) {
	keys := []string{"ProxyEnable", "ProxyType", "ProxyAddress", "ProxyNoProxy", "ProxyDocker", "ProxySystem"}
	settings, err := settingRepo.GetList(func(db *gorm.DB) *gorm.DB {
		return db.Where("`key` IN ?", keys)
	})
	if err != nil {
		return nil, err
	}
	values := make(map[string]string, len(keys))
	for _, setting := range settings {
		values[setting.Key] = setting.Value
	}
	return values, nil
}

func proxyState(enabled bool) string {
	if enabled {
		return "enable"
	}
	return "disable"
}

func proxyScope(values map[string]string) string {
	if values["ProxyEnable"] != "enable" {
		return "off"
	}
	if values["ProxySystem"] == "enable" {
		return "system"
	}
	if values["ProxyDocker"] == "enable" {
		return "panel-docker"
	}
	return "panel"
}

func proxyURL(proxyType, address string) (*url.URL, error) {
	addr := strings.TrimSpace(address)
	if proxyType == "" {
		proxyType = "mix"
	}
	if proxyType == "mix" {
		if scheme, host, ok := strings.Cut(addr, "://"); ok && (scheme == "http" || scheme == "https" || scheme == "socks5" || scheme == "socks5h") {
			addr = host
		}
	}
	if !strings.Contains(addr, "://") {
		scheme := "http"
		if proxyType == "socks5" {
			scheme = "socks5"
		}
		addr = scheme + "://" + addr
	}
	parsed, err := url.Parse(addr)
	if err != nil || parsed.Hostname() == "" || parsed.ForceQuery || parsed.RawQuery != "" || parsed.Fragment != "" || (parsed.Path != "" && parsed.Path != "/") {
		return nil, fmt.Errorf("代理地址格式无效")
	}
	switch proxyType {
	case "mix", "http":
		if parsed.Scheme == "http" || parsed.Scheme == "https" {
			return parsed, nil
		}
	case "socks5":
		if parsed.Scheme == "socks5" || parsed.Scheme == "socks5h" {
			return parsed, nil
		}
	}
	return nil, fmt.Errorf("代理地址与所选协议不匹配")
}

func proxyBypass(value string) string {
	if value = strings.TrimSpace(value); value != "" {
		return value
	}
	return defaultProxyBypass
}

// NewPanelHTTPClient uses the saved proxy for panel downloads without modifying the process environment.
func NewPanelHTTPClient(timeout time.Duration) *http.Client {
	client := &http.Client{Timeout: timeout}
	if global.DB == nil {
		return client
	}
	values, err := readProxySettings()
	if err != nil || values["ProxyEnable"] != "enable" {
		return client
	}
	address, err := proxyURL(values["ProxyType"], values["ProxyAddress"])
	if err != nil {
		return client
	}
	cfg := httpproxy.Config{
		HTTPProxy: address.String(), HTTPSProxy: address.String(), NoProxy: proxyBypass(values["ProxyNoProxy"]),
	}
	proxyForURL := cfg.ProxyFunc()
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DisableKeepAlives = true
	transport.Proxy = func(req *http.Request) (*url.URL, error) {
		return proxyForURL(req.URL)
	}
	client.Transport = transport
	return client
}

func (s *SettingService) SaveProxy(req dto.ProxyConfigUpdate) (*dto.ProxyConfigResult, error) {
	proxySettingsMu.Lock()
	defer proxySettingsMu.Unlock()
	previous, err := readProxySettings()
	if err != nil {
		return nil, err
	}
	values := map[string]string{
		"ProxyType":    req.Type,
		"ProxyEnable":  proxyState(req.Enabled && !req.ClearAddress),
		"ProxyDocker":  proxyState(req.Docker && !req.ClearAddress),
		"ProxyAddress": previous["ProxyAddress"],
		"ProxySystem":  previous["ProxySystem"],
		"ProxyNoProxy": proxyBypass(req.NoProxy),
	}
	switch req.Scope {
	case "off", "panel", "panel-docker", "system":
		values["ProxyEnable"] = proxyState(req.Scope != "off")
		values["ProxyDocker"] = proxyState(req.Scope == "panel-docker" || req.Scope == "system")
		values["ProxySystem"] = proxyState(req.Scope == "system")
	case "":
		// Older clients have no system scope field.
		if req.Enabled && previous["ProxySystem"] == "enable" {
			values["ProxyDocker"] = "enable"
		} else {
			values["ProxySystem"] = "disable"
		}
	default:
		return nil, buserr.New(constant.ErrInvalidParams)
	}
	if req.ClearAddress {
		values["ProxyAddress"] = ""
		values["ProxyEnable"], values["ProxyDocker"], values["ProxySystem"] = "disable", "disable", "disable"
	} else if strings.TrimSpace(req.Address) != "" {
		values["ProxyAddress"] = strings.TrimSpace(req.Address)
	}
	return saveProxySettings(previous, values, hostProxyPaths, req.Scope != "")
}

func (s *SettingService) updateLegacyProxySetting(req dto.SettingUpdate) error {
	proxySettingsMu.Lock()
	defer proxySettingsMu.Unlock()
	previous, err := readProxySettings()
	if err != nil {
		return err
	}
	values := make(map[string]string, len(previous))
	for key, value := range previous {
		values[key] = value
	}
	values[req.Key] = req.Value
	if req.Key == "ProxyAddress" && req.Clear {
		values["ProxyEnable"], values["ProxyDocker"], values["ProxySystem"] = "disable", "disable", "disable"
	}
	if req.Key == "ProxyEnable" && req.Value != "enable" {
		values["ProxySystem"] = "disable"
	}
	_, err = saveProxySettings(previous, values, hostProxyPaths, false)
	return err
}

func saveProxySettings(previous, values map[string]string, paths proxyConfigPaths, migrateLegacy bool) (*dto.ProxyConfigResult, error) {
	if values["ProxyEnable"] == "enable" || (values["ProxyAddress"] != "" && values["ProxyAddress"] != previous["ProxyAddress"]) {
		if _, err := proxyURL(values["ProxyType"], values["ProxyAddress"]); err != nil {
			return nil, buserr.WithDetail(constant.ErrInvalidParams, err.Error(), nil)
		}
	}
	if strings.ContainsAny(values["ProxyNoProxy"], "\r\n\x00") {
		return nil, buserr.WithDetail(constant.ErrInvalidParams, "代理排除地址必须在同一行", nil)
	}
	oldDocker, err := dockerProxySettings(previous)
	if err != nil {
		return nil, err
	}
	newDocker, err := dockerProxySettings(values)
	if err != nil {
		return nil, err
	}

	dockerDaemonConfigMu.Lock()
	defer dockerDaemonConfigMu.Unlock()

	legacy := hasLegacySystemProxy(paths)
	migrateLegacy = migrateLegacy && legacy
	var changes []proxyFileChange
	if previous["ProxySystem"] == "enable" || values["ProxySystem"] == "enable" || migrateLegacy {
		systemAddress := ""
		if values["ProxySystem"] == "enable" {
			systemAddress = newDocker["http-proxy"]
		}
		changes, err = planSystemProxy(paths, systemAddress, proxyBypass(values["ProxyNoProxy"]))
		if err != nil {
			return nil, err
		}
	}

	pending, err := dockerProxyRestartPending()
	if err != nil {
		return nil, err
	}
	var rollbacks []func() error
	revert := func(err error) (*dto.ProxyConfigResult, error) {
		for i := len(rollbacks) - 1; i >= 0; i-- {
			err = errors.Join(err, rollbacks[i]())
		}
		return nil, err
	}

	if oldDocker != nil || newDocker != nil {
		rollback, err := saveDockerProxyConfig(paths.docker, oldDocker, newDocker)
		if err != nil {
			return nil, err
		}
		if rollback != nil {
			rollbacks = append(rollbacks, rollback)
		}
	}
	if migrateLegacy {
		dropIn, err := readProxyFile(paths.dockerDropIn)
		if err != nil {
			return revert(err)
		}
		if dropIn.hasMarker(legacyProxyDocker) {
			rollback, err := (proxyFileChange{file: dropIn}).apply()
			if err != nil {
				return revert(err)
			}
			rollbacks = append(rollbacks, rollback)
		}
	}
	dockerChanged := len(rollbacks) != 0
	if dockerChanged {
		pending = true
		values["ProxyDockerRestartPending"] = "enable"
	}
	for _, change := range changes {
		rollback, err := change.apply()
		if err != nil {
			return revert(err)
		}
		if rollback != nil {
			rollbacks = append(rollbacks, rollback)
		}
	}
	if err := settingRepo.CreateOrUpdateMany(values); err != nil {
		return revert(err)
	}
	return &dto.ProxyConfigResult{
		DockerChanged: dockerChanged, DockerRestartPending: pending, LegacySystemProxy: legacy && !migrateLegacy,
	}, nil
}

func dockerProxyRestartPending() (bool, error) {
	state, err := settingRepo.GetValueByKey("ProxyDockerRestartPending")
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	return state == "enable", err
}

func (s *SettingService) TestProxy(req dto.ProxyTest) error {
	values, err := readProxySettings()
	if err != nil {
		return err
	}
	proxyType, address := req.Type, req.Address
	if proxyType == "" {
		proxyType = values["ProxyType"]
	}
	if strings.TrimSpace(address) == "" {
		address = values["ProxyAddress"]
	}
	parsed, err := proxyURL(proxyType, address)
	if err != nil {
		return err
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = http.ProxyURL(parsed)
	defer transport.CloseIdleConnections()
	client := &http.Client{Timeout: 10 * time.Second, Transport: transport}
	resp, err := client.Get("https://www.google.com/generate_204")
	if err != nil {
		return fmt.Errorf("代理连接失败")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("测试目标返回 HTTP %d", resp.StatusCode)
	}
	return nil
}
