package service

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

type HostSystemInfo struct {
	Hostname string `json:"hostname"`
	Timezone string `json:"timezone"`
	DNS      []string `json:"dns"`
	Swap     SwapInfo `json:"swap"`
}

type SwapInfo struct {
	Total   int64  `json:"total"`
	Used    int64  `json:"used"`
	File    string `json:"file"`
	Enabled bool   `json:"enabled"`
}

type IHostSystemService interface {
	GetInfo() (*HostSystemInfo, error)
	SetHostname(hostname string) error
	SetTimezone(tz string) error
	ListTimezones() ([]string, error)
	GetDNS() ([]string, error)
	SetDNS(servers []string) error
	GetSwap() (*SwapInfo, error)
	CreateSwap(sizeMB int) error
	DeleteSwap() error
	SwapOn() error
	SwapOff() error
}

type HostSystemService struct{}

func NewIHostSystemService() IHostSystemService { return &HostSystemService{} }

func (s *HostSystemService) GetInfo() (*HostSystemInfo, error) {
	info := &HostSystemInfo{}

	if out, err := exec.Command("hostname").Output(); err == nil {
		info.Hostname = strings.TrimSpace(string(out))
	}
	if out, err := exec.Command("timedatectl", "show", "-p", "Timezone", "--value").Output(); err == nil {
		info.Timezone = strings.TrimSpace(string(out))
	}
	info.DNS, _ = s.GetDNS()
	swap, _ := s.GetSwap()
	if swap != nil {
		info.Swap = *swap
	}
	return info, nil
}

func (s *HostSystemService) SetHostname(hostname string) error {
	// 获取当前主机名，用于后续替换 /etc/hosts
	oldHostname := ""
	if out, err := exec.Command("hostname").Output(); err == nil {
		oldHostname = strings.TrimSpace(string(out))
	}

	out, err := exec.Command("hostnamectl", "set-hostname", hostname).CombinedOutput()
	if err != nil {
		return fmt.Errorf("set hostname failed: %s", strings.TrimSpace(string(out)))
	}

	// 同步更新 /etc/hosts 中的主机名条目，避免 sudo 变慢及本机解析失败
	if oldHostname != "" && oldHostname != hostname {
		if err := s.updateHostsFile(oldHostname, hostname); err != nil {
			// 非致命错误，仅记录，不阻断主流程
			fmt.Printf("warning: update /etc/hosts failed: %v\n", err)
		}
	}

	return nil
}

// updateHostsFile 将 /etc/hosts 中所有出现旧主机名的行替换为新主机名
func (s *HostSystemService) updateHostsFile(oldHostname, newHostname string) error {
	data, err := os.ReadFile("/etc/hosts")
	if err != nil {
		return err
	}

	lines := strings.Split(string(data), "\n")
	changed := false
	for i, line := range lines {
		// 跳过注释行
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		// 按字段拆分，检查是否含有旧主机名
		fields := strings.Fields(line)
		for j, f := range fields {
			if j == 0 {
				// 第一列是 IP，跳过
				continue
			}
			if f == oldHostname {
				fields[j] = newHostname
				changed = true
			}
		}
		if changed {
			// 重建行，保留原始 IP 部分格式
			lines[i] = strings.Join(fields, "\t")
		}
	}

	if !changed {
		return nil
	}

	return os.WriteFile("/etc/hosts", []byte(strings.Join(lines, "\n")), 0644)
}

func (s *HostSystemService) SetTimezone(tz string) error {
	out, err := exec.Command("timedatectl", "set-timezone", tz).CombinedOutput()
	if err != nil {
		return fmt.Errorf("set timezone failed: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

func (s *HostSystemService) ListTimezones() ([]string, error) {
	out, err := exec.Command("timedatectl", "list-timezones").Output()
	if err != nil {
		return nil, fmt.Errorf("list timezones failed: %v", err)
	}
	var tzs []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			tzs = append(tzs, line)
		}
	}
	return tzs, nil
}

func (s *HostSystemService) GetDNS() ([]string, error) {
	data, err := os.ReadFile("/etc/resolv.conf")
	if err != nil {
		return nil, err
	}
	var servers []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "nameserver") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				servers = append(servers, parts[1])
			}
		}
	}
	return servers, nil
}

func (s *HostSystemService) SetDNS(servers []string) error {
	data, _ := os.ReadFile("/etc/resolv.conf")
	var otherLines []string
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "nameserver") && trimmed != "" {
			otherLines = append(otherLines, line)
		}
	}

	var sb strings.Builder
	for _, line := range otherLines {
		sb.WriteString(line + "\n")
	}
	for _, dns := range servers {
		dns = strings.TrimSpace(dns)
		if dns != "" {
			sb.WriteString("nameserver " + dns + "\n")
		}
	}

	return os.WriteFile("/etc/resolv.conf", []byte(sb.String()), 0644)
}

const defaultSwapFile = "/swapfile"

func (s *HostSystemService) GetSwap() (*SwapInfo, error) {
	info := &SwapInfo{}

	out, err := exec.Command("swapon", "--show=NAME,SIZE,USED", "--noheadings", "--bytes").Output()
	if err == nil {
		for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
			fields := strings.Fields(line)
			if len(fields) >= 3 {
				info.File = fields[0]
				info.Total, _ = strconv.ParseInt(fields[1], 10, 64)
				info.Used, _ = strconv.ParseInt(fields[2], 10, 64)
				info.Enabled = true
				break
			}
		}
	}

	if !info.Enabled {
		if _, err := os.Stat(defaultSwapFile); err == nil {
			fi, _ := os.Stat(defaultSwapFile)
			info.File = defaultSwapFile
			info.Total = fi.Size()
		}
	}

	return info, nil
}

func (s *HostSystemService) CreateSwap(sizeMB int) error {
	if sizeMB < 64 {
		return fmt.Errorf("swap size must be at least 64 MB")
	}

	_ = exec.Command("swapoff", defaultSwapFile).Run()
	os.Remove(defaultSwapFile)

	out, err := exec.Command("dd", "if=/dev/zero", "of="+defaultSwapFile,
		"bs=1M", fmt.Sprintf("count=%d", sizeMB)).CombinedOutput()
	if err != nil {
		return fmt.Errorf("create swap file failed: %s", strings.TrimSpace(string(out)))
	}

	if err := os.Chmod(defaultSwapFile, 0600); err != nil {
		return fmt.Errorf("chmod failed: %v", err)
	}

	out, err = exec.Command("mkswap", defaultSwapFile).CombinedOutput()
	if err != nil {
		return fmt.Errorf("mkswap failed: %s", strings.TrimSpace(string(out)))
	}

	out, err = exec.Command("swapon", defaultSwapFile).CombinedOutput()
	if err != nil {
		return fmt.Errorf("swapon failed: %s", strings.TrimSpace(string(out)))
	}

	s.ensureFstabSwap()
	return nil
}

func (s *HostSystemService) DeleteSwap() error {
	_ = exec.Command("swapoff", defaultSwapFile).Run()
	os.Remove(defaultSwapFile)
	s.removeFstabSwap()
	return nil
}

func (s *HostSystemService) SwapOn() error {
	out, err := exec.Command("swapon", defaultSwapFile).CombinedOutput()
	if err != nil {
		return fmt.Errorf("swapon failed: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

func (s *HostSystemService) SwapOff() error {
	out, err := exec.Command("swapoff", defaultSwapFile).CombinedOutput()
	if err != nil {
		return fmt.Errorf("swapoff failed: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

func (s *HostSystemService) ensureFstabSwap() {
	data, _ := os.ReadFile("/etc/fstab")
	if strings.Contains(string(data), defaultSwapFile) {
		return
	}
	f, err := os.OpenFile("/etc/fstab", os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	f.WriteString(fmt.Sprintf("\n%s none swap sw 0 0\n", defaultSwapFile))
}

func (s *HostSystemService) removeFstabSwap() {
	data, err := os.ReadFile("/etc/fstab")
	if err != nil {
		return
	}
	var lines []string
	for _, line := range strings.Split(string(data), "\n") {
		if !strings.Contains(line, defaultSwapFile) {
			lines = append(lines, line)
		}
	}
	os.WriteFile("/etc/fstab", []byte(strings.Join(lines, "\n")), 0644)
}
