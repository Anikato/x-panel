package service

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"

	"xpanel/app/dto"
	"xpanel/global"
	"xpanel/utils/cmd"
)

const sshdConfigPath = "/etc/ssh/sshd_config"

type ISSHManageService interface {
	GetSSHInfo() (*dto.SSHInfo, error)
	OperateSSH(operation string) error
	UpdateSSHConfig(key, value string) error
	LoadSSHLog(req dto.SSHLogSearch) (int64, []dto.SSHLogEntry, error)
}

type SSHManageService struct{}

func NewISSHManageService() ISSHManageService { return &SSHManageService{} }

func (s *SSHManageService) GetSSHInfo() (*dto.SSHInfo, error) {
	info := &dto.SSHInfo{
		Port:                   "22",
		ListenAddress:          "0.0.0.0",
		PasswordAuthentication: "yes",
		PubkeyAuthentication:   "yes",
		PermitRootLogin:        "yes",
		UseDNS:                 "no",
	}

	// 检查 sshd 是否安装
	serviceName := detectSSHService()
	if serviceName == "" {
		info.IsExist = false
		info.Message = "SSH service not found"
		return info, nil
	}
	info.IsExist = true

	// 检查服务状态
	active, _ := cmd.ExecWithOutput("systemctl", "is-active", serviceName)
	info.IsActive = strings.TrimSpace(active) == "active"

	enabled, _ := cmd.ExecWithOutput("systemctl", "is-enabled", serviceName)
	info.AutoStart = strings.TrimSpace(enabled) == "enabled"

	// 读取配置
	file, err := os.Open(sshdConfigPath)
	if err != nil {
		info.Message = "Cannot read sshd_config: " + err.Error()
		return info, nil
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		switch key {
		case "Port":
			info.Port = val
		case "ListenAddress":
			info.ListenAddress = val
		case "PasswordAuthentication":
			info.PasswordAuthentication = val
		case "PubkeyAuthentication":
			info.PubkeyAuthentication = val
		case "PermitRootLogin":
			info.PermitRootLogin = val
		case "UseDNS":
			info.UseDNS = val
		}
	}
	return info, nil
}

func (s *SSHManageService) OperateSSH(operation string) error {
	serviceName := detectSSHService()
	if serviceName == "" {
		return fmt.Errorf("SSH service not found")
	}

	var args []string
	switch operation {
	case "start":
		args = []string{"systemctl", "start", serviceName}
	case "stop":
		args = []string{"systemctl", "stop", serviceName}
	case "restart":
		args = []string{"systemctl", "restart", serviceName}
	case "enable":
		args = []string{"systemctl", "enable", serviceName}
	case "disable":
		args = []string{"systemctl", "disable", serviceName}
	default:
		return fmt.Errorf("unsupported operation: %s", operation)
	}

	_, err := cmd.ExecWithOutput(args[0], args[1:]...)
	return err
}

func (s *SSHManageService) UpdateSSHConfig(key, value string) error {
	// 允许修改的配置项白名单
	allowedKeys := map[string]bool{
		"Port": true, "ListenAddress": true,
		"PasswordAuthentication": true, "PubkeyAuthentication": true,
		"PermitRootLogin": true, "UseDNS": true,
	}
	if !allowedKeys[key] {
		return fmt.Errorf("key %s is not allowed to update", key)
	}

	content, err := os.ReadFile(sshdConfigPath)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	found := false
	pattern := regexp.MustCompile(`^#?\s*` + regexp.QuoteMeta(key) + `\s+`)

	for i, line := range lines {
		if pattern.MatchString(line) {
			lines[i] = key + " " + value
			found = true
			break
		}
	}
	if !found {
		lines = append(lines, key+" "+value)
	}

	if err := os.WriteFile(sshdConfigPath, []byte(strings.Join(lines, "\n")), 0644); err != nil {
		return err
	}

	// 测试配置
	if _, err := cmd.ExecWithOutput("sshd", "-t"); err != nil {
		// 回滚
		os.WriteFile(sshdConfigPath, content, 0644)
		return fmt.Errorf("sshd config test failed: %v", err)
	}

	global.LOG.Infof("SSH config updated: %s = %s", key, value)
	return nil
}

func (s *SSHManageService) LoadSSHLog(req dto.SSHLogSearch) (int64, []dto.SSHLogEntry, error) {
	// 从 /var/log/auth.log (Debian) 或 journalctl 读取 SSH 日志
	var lines []string

	// 先尝试 journalctl
	output, err := cmd.ExecWithOutput("journalctl", "-u", "ssh", "-u", "sshd", "--no-pager", "-n", "500", "--output=short-iso")
	if err == nil && output != "" {
		lines = strings.Split(strings.TrimSpace(output), "\n")
	} else {
		// 回退到 auth.log
		content, err := os.ReadFile("/var/log/auth.log")
		if err != nil {
			return 0, nil, nil // 无日志可读，返回空
		}
		allLines := strings.Split(string(content), "\n")
		for _, l := range allLines {
			if strings.Contains(l, "sshd") {
				lines = append(lines, l)
			}
		}
	}

	// 解析日志行
	var entries []dto.SSHLogEntry
	acceptedRe := regexp.MustCompile(`Accepted\s+\w+\s+for\s+(\S+)\s+from\s+(\S+)\s+port\s+(\S+)`)
	failedRe := regexp.MustCompile(`Failed\s+\w+\s+for\s+(?:invalid user\s+)?(\S+)\s+from\s+(\S+)\s+port\s+(\S+)`)

	for _, line := range lines {
		if line == "" {
			continue
		}
		entry := dto.SSHLogEntry{}

		if m := acceptedRe.FindStringSubmatch(line); m != nil {
			entry.Status = "success"
			entry.User = m[1]
			entry.IP = m[2]
			entry.Port = m[3]
		} else if m := failedRe.FindStringSubmatch(line); m != nil {
			entry.Status = "failed"
			entry.User = m[1]
			entry.IP = m[2]
			entry.Port = m[3]
		} else {
			continue
		}

		// 过滤
		if req.Status != "" && req.Status != "all" && entry.Status != req.Status {
			continue
		}
		if req.Info != "" && !strings.Contains(line, req.Info) {
			continue
		}

		// 提取日期部分
		if len(line) > 20 {
			entry.Date = line[:19]
		}
		entry.Message = line

		entries = append(entries, entry)
	}

	// 倒序排列（最新在前）
	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}

	// 分页
	total := int64(len(entries))
	start := (req.Page - 1) * req.PageSize
	end := start + req.PageSize
	if start > int(total) {
		return total, nil, nil
	}
	if end > int(total) {
		end = int(total)
	}

	return total, entries[start:end], nil
}

func detectSSHService() string {
	for _, name := range []string{"sshd", "ssh"} {
		output, _ := cmd.ExecWithOutput("systemctl", "status", name)
		if output != "" && !strings.Contains(output, "could not be found") {
			return name
		}
	}
	return ""
}
