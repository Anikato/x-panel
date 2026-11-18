package service

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"xpanel/app/dto"
	"xpanel/global"
	"xpanel/utils/cmd"
)

const sshdConfigPath = "/etc/ssh/sshd_config"
const authorizedKeysPath = "/root/.ssh/authorized_keys"

type ISSHManageService interface {
	GetSSHInfo() (*dto.SSHInfo, error)
	OperateSSH(operation string) error
	UpdateSSHConfig(key, value string) error
	LoadSSHLog(req dto.SSHLogSearch) (int64, []dto.SSHLogEntry, error)
	GetSSHDConfig() (string, error)
	SaveSSHDConfig(content string) error
	ListAuthorizedKeys() ([]dto.AuthorizedKey, error)
	AddAuthorizedKey(req dto.AuthorizedKeyCreate) error
	DeleteAuthorizedKey(fingerprint string) error
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

	serviceName := detectSSHService()
	if serviceName == "" {
		info.IsExist = false
		info.Message = "SSH service not found"
		return info, nil
	}
	info.IsExist = true

	active, _ := cmd.ExecWithOutput("systemctl", "is-active", serviceName)
	info.IsActive = strings.TrimSpace(active) == "active"

	enabled, _ := cmd.ExecWithOutput("systemctl", "is-enabled", serviceName)
	enabledStatus := strings.TrimSpace(enabled)
	info.AutoStart = enabledStatus == "enabled" || enabledStatus == "static" ||
		enabledStatus == "indirect" || enabledStatus == "alias"

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
	case "start", "stop", "restart", "enable", "disable":
		args = []string{"systemctl", operation, serviceName}
	default:
		return fmt.Errorf("unsupported operation: %s", operation)
	}

	_, err := cmd.ExecWithOutput(args[0], args[1:]...)
	return err
}

func (s *SSHManageService) UpdateSSHConfig(key, value string) error {
	allowedKeys := map[string]bool{
		"Port": true, "ListenAddress": true,
		"PasswordAuthentication": true, "PubkeyAuthentication": true,
		"PermitRootLogin": true, "UseDNS": true,
	}
	if !allowedKeys[key] {
		return fmt.Errorf("key %s is not allowed to update", key)
	}

	if strings.ContainsAny(value, "\r\n") {
		return fmt.Errorf("SSH directive must be a single line")
	}
	return editSSHDConfig(sshdConfigPath, func(content string) string {
		lines := strings.Split(content, "\n")
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

		return strings.Join(lines, "\n")
	})
}

func (s *SSHManageService) LoadSSHLog(req dto.SSHLogSearch) (int64, []dto.SSHLogEntry, error) {
	var lines []string

	output, err := cmd.ExecWithOutput("journalctl", "-u", "ssh", "-u", "sshd", "--no-pager", "-n", "500", "--output=short-iso")
	if err == nil && output != "" {
		lines = strings.Split(strings.TrimSpace(output), "\n")
	} else {
		content, err := os.ReadFile("/var/log/auth.log")
		if err != nil {
			return 0, nil, nil
		}
		allLines := strings.Split(string(content), "\n")
		for _, l := range allLines {
			if strings.Contains(l, "sshd") {
				lines = append(lines, l)
			}
		}
	}

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

		if req.Status != "" && req.Status != "all" && entry.Status != req.Status {
			continue
		}
		if req.Info != "" && !strings.Contains(line, req.Info) {
			continue
		}

		if len(line) > 20 {
			entry.Date = line[:19]
		}
		entry.Message = line

		entries = append(entries, entry)
	}

	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}

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

func (s *SSHManageService) GetSSHDConfig() (string, error) {
	content, err := os.ReadFile(sshdConfigPath)
	if err != nil {
		return "", fmt.Errorf("read sshd_config: %v", err)
	}
	return string(content), nil
}

func (s *SSHManageService) SaveSSHDConfig(content string) error {
	return editSSHDConfig(sshdConfigPath, func(string) string { return content })
}

var sshConfigMu sync.Mutex

func editSSHDConfig(configPath string, edit func(string) string) error {
	sshConfigMu.Lock()
	defer sshConfigMu.Unlock()

	path, err := filepath.EvalSymlinks(configPath)
	if err != nil {
		return err
	}
	original, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	stage, err := os.MkdirTemp(filepath.Dir(path), ".xpanel-sshd-")
	if err != nil {
		return err
	}
	keepBackup := false
	defer func() {
		if !keepBackup {
			_ = os.RemoveAll(stage)
		}
	}()
	candidate := filepath.Join(stage, "candidate")
	// Keep the original mode, owner, ACLs and host security labels when replacing the file.
	if output, err := cmd.ExecWithOutput("cp", "--preserve=all", "--", path, candidate); err != nil {
		return fmt.Errorf("stage sshd config: %w: %s", err, strings.TrimSpace(output))
	}
	if err := os.WriteFile(candidate, []byte(edit(string(original))), 0600); err != nil {
		return err
	}
	if output, err := cmd.ExecWithOutput("sshd", "-t", "-f", candidate); err != nil {
		return fmt.Errorf("sshd config test failed: %w: %s", err, strings.TrimSpace(output))
	}

	serviceName := detectSSHService()
	if serviceName == "" {
		return fmt.Errorf("SSH service not found")
	}

	backup := filepath.Join(stage, "original")
	if err := os.Link(path, backup); err != nil {
		return err
	}
	if err := os.Rename(candidate, path); err != nil {
		return err
	}
	if applyErr := restartSSHService(serviceName); applyErr != nil {
		if err := os.Rename(backup, path); err != nil {
			keepBackup = true
			return fmt.Errorf("SSH restart failed: %v; could not restore config from %s: %w", applyErr, backup, err)
		}
		if err := restartSSHService(serviceName); err != nil {
			return fmt.Errorf("SSH restart failed: %v; original config restored but recovery restart failed: %w", applyErr, err)
		}
		return fmt.Errorf("SSH restart failed; original config restored and service restarted: %w", applyErr)
	}
	return nil
}

// ============================================================
// Authorized Keys 管理
// ============================================================

func (s *SSHManageService) ListAuthorizedKeys() ([]dto.AuthorizedKey, error) {
	content, err := os.ReadFile(authorizedKeysPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []dto.AuthorizedKey{}, nil
		}
		return nil, fmt.Errorf("read authorized_keys: %v", err)
	}

	var keys []dto.AuthorizedKey
	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key := parseAuthorizedKeyLine(line)
		if key.KeyType != "" {
			keys = append(keys, key)
		}
	}
	return keys, nil
}

func (s *SSHManageService) AddAuthorizedKey(req dto.AuthorizedKeyCreate) error {
	keyLine := strings.TrimSpace(req.Key)
	if keyLine == "" {
		return fmt.Errorf("key content is empty")
	}

	parts := strings.Fields(keyLine)
	if len(parts) < 2 {
		return fmt.Errorf("invalid SSH public key format")
	}

	dir := filepath.Dir(authorizedKeysPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create .ssh dir: %v", err)
	}

	existing, _ := os.ReadFile(authorizedKeysPath)
	for _, line := range strings.Split(string(existing), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		existParts := strings.Fields(line)
		if len(existParts) >= 2 && existParts[1] == parts[1] {
			return fmt.Errorf("this key already exists")
		}
	}

	f, err := os.OpenFile(authorizedKeysPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("open authorized_keys: %v", err)
	}
	defer f.Close()

	content := keyLine
	if !strings.HasSuffix(string(existing), "\n") && len(existing) > 0 {
		content = "\n" + content
	}
	content += "\n"

	if _, err := f.WriteString(content); err != nil {
		return fmt.Errorf("write key: %v", err)
	}

	global.LOG.Infof("SSH authorized key added: %s", req.Name)
	return nil
}

func (s *SSHManageService) DeleteAuthorizedKey(fingerprint string) error {
	content, err := os.ReadFile(authorizedKeysPath)
	if err != nil {
		return fmt.Errorf("read authorized_keys: %v", err)
	}

	lines := strings.Split(string(content), "\n")
	var newLines []string
	found := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			newLines = append(newLines, line)
			continue
		}
		key := parseAuthorizedKeyLine(trimmed)
		if key.Fingerprint == fingerprint {
			found = true
			continue
		}
		newLines = append(newLines, line)
	}

	if !found {
		return fmt.Errorf("key not found")
	}

	if err := os.WriteFile(authorizedKeysPath, []byte(strings.Join(newLines, "\n")), 0600); err != nil {
		return fmt.Errorf("write authorized_keys: %v", err)
	}

	global.LOG.Infof("SSH authorized key deleted: %s", fingerprint)
	return nil
}

// ============================================================
// 辅助函数
// ============================================================

func parseAuthorizedKeyLine(line string) dto.AuthorizedKey {
	parts := strings.Fields(line)
	if len(parts) < 2 {
		return dto.AuthorizedKey{}
	}

	key := dto.AuthorizedKey{
		KeyType: parts[0],
		Key:     parts[1],
	}
	if len(parts) >= 3 {
		key.Name = strings.Join(parts[2:], " ")
	}

	// 生成简单指纹用于标识（取 base64 前 16 字符）
	if len(parts[1]) > 16 {
		key.Fingerprint = parts[1][:16]
	} else {
		key.Fingerprint = parts[1]
	}

	return key
}

func detectSSHService() string {
	// Debian/Ubuntu 用 ssh，RHEL/CentOS 用 sshd
	for _, name := range []string{"ssh", "sshd"} {
		output, err := cmd.ExecWithOutput("systemctl", "show", "--property=LoadState", "--value", name)
		if err == nil && strings.TrimSpace(output) == "loaded" {
			return name
		}
	}
	return ""
}

func restartSSHService(serviceName string) error {
	output, err := cmd.ExecWithOutput("systemctl", "restart", serviceName)
	if err != nil {
		return fmt.Errorf("restart %s: %w: %s", serviceName, err, strings.TrimSpace(output))
	}
	return nil
}
