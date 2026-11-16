package service

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"xpanel/buserr"
	"xpanel/constant"
)

var dockerDaemonConfigMu sync.Mutex

func dockerProxySettings(values map[string]string) (map[string]string, error) {
	if values["ProxyDocker"] != "enable" || values["ProxyEnable"] != "enable" {
		return nil, nil
	}
	address, err := proxyURL(values["ProxyType"], values["ProxyAddress"])
	if err != nil {
		return nil, err
	}
	if address.Scheme != "http" && address.Scheme != "https" {
		return nil, buserr.WithDetail(constant.ErrInvalidParams, "Docker 代理需要 HTTP、HTTPS 或混合代理", nil)
	}
	return map[string]string{
		"http-proxy": address.String(), "https-proxy": address.String(), "no-proxy": proxyBypass(values["ProxyNoProxy"]),
	}, nil
}

// The caller holds dockerDaemonConfigMu until both the file and panel settings are saved.
func saveDockerProxyConfig(path string, previous, desired map[string]string) (func() error, error) {
	config := make(map[string]json.RawMessage)
	info, err := os.Lstat(path)
	existed := err == nil
	var original []byte
	if existed {
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("Docker 配置路径不是普通文件")
		}
		original, err = os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(original, &config); err != nil || config == nil {
			return nil, fmt.Errorf("Docker 配置文件不是有效的 JSON 对象")
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	proxies := make(map[string]json.RawMessage)
	if raw, ok := config["proxies"]; ok {
		if err := json.Unmarshal(raw, &proxies); err != nil || proxies == nil {
			return nil, fmt.Errorf("Docker 代理配置不是有效的 JSON 对象")
		}
	}

	changed := false
	for _, key := range []string{"http-proxy", "https-proxy", "no-proxy"} {
		existing, present := proxies[key]
		wanted, useProxy := desired[key]
		if present {
			var current string
			if err := json.Unmarshal(existing, &current); err != nil {
				return nil, fmt.Errorf("Docker 代理字段 %s 不是字符串", key)
			}
			if saved, managed := previous[key]; !managed || current != saved {
				return nil, fmt.Errorf("Docker 已有独立的代理设置（%s）", key)
			}
			if useProxy && current == wanted {
				continue
			}
		}
		if useProxy {
			proxies[key], _ = json.Marshal(wanted)
			changed = true
		} else if present {
			delete(proxies, key)
			changed = true
		}
	}
	if !changed {
		return nil, nil
	}
	if len(proxies) == 0 {
		delete(config, "proxies")
	} else {
		config["proxies"], err = json.Marshal(proxies)
		if err != nil {
			return nil, err
		}
	}
	content, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, err
	}
	if err := writeFileAtomic(path, append(content, '\n'), 0600); err != nil {
		return nil, err
	}
	return func() error {
		if !existed {
			return os.Remove(path)
		}
		return writeFileAtomic(path, original, info.Mode().Perm())
	}, nil
}
