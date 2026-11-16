package service

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	legacyProxyProfile  = "# Managed by X-Panel - do not edit manually\n"
	legacyProxyAPT      = "// Managed by X-Panel\n"
	legacyProxyDocker   = "# Managed by X-Panel\n[Service]\n"
	managedProxyProfile = "# XPanel system proxy\n"
	managedProxyAPT     = "// XPanel system proxy\n"
	proxyEnvStart       = "# BEGIN XPanel system proxy"
	proxyEnvEnd         = "# END XPanel system proxy"
)

type proxyConfigPaths struct {
	environment, profile, apt, docker, dockerDropIn string
}

var hostProxyPaths = proxyConfigPaths{
	environment:  "/etc/environment",
	profile:      "/etc/profile.d/xpanel-proxy.sh",
	apt:          "/etc/apt/apt.conf.d/99xpanel-proxy",
	docker:       daemonJSONPath,
	dockerDropIn: "/etc/systemd/system/docker.service.d/http-proxy.conf",
}

type proxyFile struct {
	path    string
	content []byte
	info    os.FileInfo
}

func readProxyFile(path string) (proxyFile, error) {
	file := proxyFile{path: path}
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return file, nil
	}
	if err != nil {
		return file, err
	}
	if !info.Mode().IsRegular() {
		return file, fmt.Errorf("代理配置路径不是普通文件（%s）", path)
	}
	file.info = info
	file.content, err = os.ReadFile(path)
	return file, err
}

func (file proxyFile) hasMarker(marker string) bool {
	return bytes.HasPrefix(file.content, []byte(marker))
}

func hasLegacySystemProxy(paths proxyConfigPaths) bool {
	for path, marker := range map[string]string{
		paths.profile: legacyProxyProfile, paths.apt: legacyProxyAPT, paths.dockerDropIn: legacyProxyDocker,
	} {
		if file, err := readProxyFile(path); err == nil && file.hasMarker(marker) {
			return true
		}
	}
	return false
}

type proxyFileChange struct {
	file    proxyFile
	content []byte
}

func (change proxyFileChange) apply() (func() error, error) {
	file := change.file
	if bytes.Equal(file.content, change.content) && (file.info == nil || change.content != nil) {
		return nil, nil
	}
	mode := os.FileMode(0644)
	if file.info != nil {
		mode = file.info.Mode().Perm()
	}
	if change.content == nil {
		if err := os.Remove(file.path); err != nil {
			return nil, err
		}
	} else if err := writeFileAtomic(file.path, change.content, mode); err != nil {
		return nil, err
	}
	return func() error {
		if file.info == nil {
			return os.Remove(file.path)
		}
		return writeFileAtomic(file.path, file.content, mode)
	}, nil
}

func planSystemProxy(paths proxyConfigPaths, proxy, bypass string) ([]proxyFileChange, error) {
	profile, err := readProxyFile(paths.profile)
	if err != nil {
		return nil, err
	}
	apt, err := readProxyFile(paths.apt)
	if err != nil {
		return nil, err
	}
	environment, err := readProxyFile(paths.environment)
	if err != nil {
		return nil, err
	}

	var aptContent, profileContent []byte
	if proxy != "" {
		if profile.info != nil && !profile.hasMarker(legacyProxyProfile) && !profile.hasMarker(managedProxyProfile) {
			return nil, fmt.Errorf("系统已有独立的 Shell 代理配置（%s）", paths.profile)
		}
		if apt.info != nil && !apt.hasMarker(legacyProxyAPT) && !apt.hasMarker(managedProxyAPT) {
			return nil, fmt.Errorf("系统已有独立的 apt 代理配置（%s）", paths.apt)
		}
		// These values also have to fit in plain, double-quoted PAM and apt assignments.
		if strings.ContainsAny(proxy+bypass, "\"\\\r\n\x00") {
			return nil, fmt.Errorf("代理地址或排除地址包含系统代理配置无法使用的字符")
		}
		var shell strings.Builder
		shell.WriteString(managedProxyProfile)
		for _, key := range proxyEnvironmentKeys {
			value := proxy
			if strings.EqualFold(key, "no_proxy") {
				value = bypass
			}
			fmt.Fprintf(&shell, "export %s='%s'\n", key, strings.ReplaceAll(value, "'", "'\\''"))
		}
		profileContent = []byte(shell.String())
		if _, err := os.Stat(filepath.Dir(paths.apt)); err == nil {
			aptContent = []byte(fmt.Sprintf("%sAcquire::http::Proxy \"%s\";\nAcquire::https::Proxy \"%s\";\n", managedProxyAPT, proxy, proxy))
		} else if !os.IsNotExist(err) {
			return nil, err
		}
	}

	legacy := map[string]bool{}
	if profile.hasMarker(legacyProxyProfile) {
		for _, line := range strings.Split(string(profile.content), "\n") {
			if assignment, ok := strings.CutPrefix(line, "export "); ok {
				legacy[assignment] = true
			}
		}
	}
	envContent, err := editProxyEnvironment(environment.content, legacy, proxy, bypass)
	if err != nil {
		return nil, err
	}

	changes := []proxyFileChange{{file: environment, content: envContent}}
	if proxy != "" || profile.hasMarker(legacyProxyProfile) || profile.hasMarker(managedProxyProfile) {
		changes = append(changes, proxyFileChange{file: profile, content: profileContent})
	}
	if aptContent != nil || apt.hasMarker(legacyProxyAPT) || apt.hasMarker(managedProxyAPT) {
		changes = append(changes, proxyFileChange{file: apt, content: aptContent})
	}
	return changes, nil
}

var proxyEnvironmentKeys = []string{"http_proxy", "https_proxy", "HTTP_PROXY", "HTTPS_PROXY", "no_proxy", "NO_PROXY"}

func editProxyEnvironment(original []byte, legacy map[string]bool, proxy, bypass string) ([]byte, error) {
	var content strings.Builder
	inBlock := false
	for _, line := range strings.SplitAfter(string(original), "\n") {
		trimmed := strings.TrimSpace(line)
		switch trimmed {
		case proxyEnvStart, proxyEnvEnd:
			if inBlock == (trimmed == proxyEnvStart) {
				return nil, fmt.Errorf("系统代理环境变量标记不完整（/etc/environment）")
			}
			inBlock = trimmed == proxyEnvStart
			continue
		}
		if inBlock {
			continue
		}
		key, _, hasValue := strings.Cut(strings.TrimPrefix(trimmed, "export "), "=")
		switch strings.ToLower(strings.TrimSpace(key)) {
		case "http_proxy", "https_proxy", "no_proxy":
			if hasValue {
				if legacy[trimmed] {
					continue
				}
				if proxy != "" {
					return nil, fmt.Errorf("系统已有独立的代理环境变量（%s）", strings.TrimSpace(key))
				}
			}
		}
		content.WriteString(line)
	}
	if inBlock {
		return nil, fmt.Errorf("系统代理环境变量标记不完整（/etc/environment）")
	}
	if proxy != "" {
		if content.Len() != 0 && !strings.HasSuffix(content.String(), "\n") {
			content.WriteByte('\n')
		}
		fmt.Fprintln(&content, proxyEnvStart)
		for _, key := range proxyEnvironmentKeys {
			value := proxy
			if strings.EqualFold(key, "no_proxy") {
				value = bypass
			}
			fmt.Fprintf(&content, "%s=\"%s\"\n", key, value)
		}
		fmt.Fprintln(&content, proxyEnvEnd)
	}
	return []byte(content.String()), nil
}
