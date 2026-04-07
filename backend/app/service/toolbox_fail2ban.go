package service

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"xpanel/app/dto"
	"xpanel/buserr"
	"xpanel/utils/iplocation"
)

type IFail2banService interface {
	GetStatus() (*dto.ServiceStatus, error)
	Install() error
	Uninstall() error
	Operate(req dto.ServiceOperate) error

	ListJails() ([]dto.Fail2banJail, error)
	UpdateJail(req dto.Fail2banJailUpdate) error
	SetSSHJail(req dto.Fail2banSSHConfig) error

	ListBanned() ([]dto.Fail2banBannedIP, error)
	Unban(req dto.Fail2banUnbanReq) error

	GetLogs(lines int) (string, error)
}

type Fail2banService struct{}

func NewIFail2banService() IFail2banService { return &Fail2banService{} }

const (
	f2bService   = "fail2ban"
	f2bClient    = "fail2ban-client"
	f2bJailLocal = "/etc/fail2ban/jail.local"
	f2bJailD     = "/etc/fail2ban/jail.d"
	f2bLogFile   = "/var/log/fail2ban.log"
)

func (s *Fail2banService) GetStatus() (*dto.ServiceStatus, error) {
	st := &dto.ServiceStatus{}

	if _, err := exec.LookPath(f2bClient); err != nil {
		return st, nil
	}
	st.IsInstalled = true

	if out, err := exec.Command("systemctl", "is-active", f2bService).Output(); err == nil {
		st.IsRunning = strings.TrimSpace(string(out)) == "active"
	}
	if out, err := exec.Command("systemctl", "is-enabled", f2bService).Output(); err == nil {
		enabled := strings.TrimSpace(string(out))
		st.AutoStart = enabled == "enabled" || enabled == "enabled-runtime"
	}
	if out, err := exec.Command(f2bClient, "version").Output(); err == nil {
		st.Version = strings.TrimSpace(string(out))
	}
	return st, nil
}

func (s *Fail2banService) Install() error {
	out, err := exec.Command("apt-get", "install", "-y", "fail2ban").CombinedOutput()
	if err != nil {
		return buserr.WithDetail("ErrFail2banInstall",
			fmt.Sprintf("apt-get install fail2ban failed: %s", strings.TrimSpace(string(out))), err)
	}
	s.ensureJailLocal()
	_ = exec.Command("systemctl", "enable", f2bService).Run()
	_ = exec.Command("systemctl", "start", f2bService).Run()
	return nil
}

func (s *Fail2banService) Uninstall() error {
	_ = exec.Command("systemctl", "stop", f2bService).Run()
	out, err := exec.Command("apt-get", "remove", "-y", "fail2ban").CombinedOutput()
	if err != nil {
		return buserr.WithDetail("ErrFail2banUninstall",
			fmt.Sprintf("apt-get remove fail2ban failed: %s", strings.TrimSpace(string(out))), err)
	}
	return nil
}

func (s *Fail2banService) Operate(req dto.ServiceOperate) error {
	var cmd *exec.Cmd
	switch req.Operation {
	case "start", "stop", "restart":
		cmd = exec.Command("systemctl", req.Operation, f2bService)
	case "enable":
		cmd = exec.Command("systemctl", "enable", f2bService)
	case "disable":
		cmd = exec.Command("systemctl", "disable", f2bService)
	default:
		return fmt.Errorf("unsupported operation: %s", req.Operation)
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("fail2ban %s failed: %s", req.Operation, strings.TrimSpace(string(out)))
	}
	return nil
}

func (s *Fail2banService) ListJails() ([]dto.Fail2banJail, error) {
	out, err := exec.Command(f2bClient, "status").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("fail2ban-client status failed: %s", strings.TrimSpace(string(out)))
	}

	jailNames := parseJailList(string(out))
	jailConf := s.readAllJailConfig()

	var jails []dto.Fail2banJail
	for _, name := range jailNames {
		jail := dto.Fail2banJail{Name: name, Enabled: true}

		if conf, ok := jailConf[name]; ok {
			jail.Port = conf["port"]
			jail.Filter = conf["filter"]
			jail.LogPath = conf["logpath"]
			jail.MaxRetry, _ = strconv.Atoi(conf["maxretry"])
			jail.FindTime = conf["findtime"]
			jail.BanTime = conf["bantime"]
			jail.Action = conf["action"]
		}

		bannedIPs := s.getBannedIPs(name)
		jail.BannedIPs = bannedIPs
		jail.BannedCount = len(bannedIPs)

		jails = append(jails, jail)
	}

	for name, conf := range jailConf {
		found := false
		for _, j := range jails {
			if j.Name == name {
				found = true
				break
			}
		}
		if !found {
			jail := dto.Fail2banJail{
				Name:     name,
				Enabled:  strings.ToLower(conf["enabled"]) == "true",
				Port:     conf["port"],
				Filter:   conf["filter"],
				LogPath:  conf["logpath"],
				MaxRetry: 0,
				FindTime: conf["findtime"],
				BanTime:  conf["bantime"],
				Action:   conf["action"],
			}
			jail.MaxRetry, _ = strconv.Atoi(conf["maxretry"])
			jails = append(jails, jail)
		}
	}

	return jails, nil
}

func (s *Fail2banService) UpdateJail(req dto.Fail2banJailUpdate) error {
	s.ensureJailLocal()

	content, err := os.ReadFile(f2bJailLocal)
	if err != nil {
		return err
	}

	newSection := fmt.Sprintf("[%s]\nenabled = %v\n", req.Name, req.Enabled)
	if req.Port != "" {
		newSection += fmt.Sprintf("port = %s\n", req.Port)
	}
	if req.MaxRetry > 0 {
		newSection += fmt.Sprintf("maxretry = %d\n", req.MaxRetry)
	}
	if req.FindTime != "" {
		newSection += fmt.Sprintf("findtime = %s\n", req.FindTime)
	}
	if req.BanTime != "" {
		newSection += fmt.Sprintf("bantime = %s\n", req.BanTime)
	}
	if req.Action != "" {
		newSection += fmt.Sprintf("action = %s\n", req.Action)
	}

	updated := replaceOrAppendSection(string(content), req.Name, newSection)
	return s.safeWriteConfig([]byte(updated))
}

func (s *Fail2banService) SetSSHJail(req dto.Fail2banSSHConfig) error {
	s.ensureJailLocal()

	content, err := os.ReadFile(f2bJailLocal)
	if err != nil {
		return err
	}

	port := req.Port
	if port == "" {
		port = detectSSHPort()
	}
	backend := detectBackend()

	newSection := fmt.Sprintf(`[sshd]
enabled = %v
port = %s
filter = sshd
backend = %s
maxretry = %d
findtime = %s
bantime = %s
`, req.Enabled, port, backend, req.MaxRetry, req.FindTime, req.BanTime)

	updated := replaceOrAppendSection(string(content), "sshd", newSection)
	return s.safeWriteConfig([]byte(updated))
}

func (s *Fail2banService) ListBanned() ([]dto.Fail2banBannedIP, error) {
	out, err := exec.Command(f2bClient, "status").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("fail2ban-client status failed: %s", strings.TrimSpace(string(out)))
	}

	jailNames := parseJailList(string(out))
	var result []dto.Fail2banBannedIP

	ipSvc := iplocation.GetService()
	for _, jail := range jailNames {
		ips := s.getBannedIPs(jail)
		for _, ip := range ips {
			geo := ipSvc.Lookup(ip)
			result = append(result, dto.Fail2banBannedIP{
				IP: ip, Jail: jail,
				Country: geo.Country, CountryCode: geo.CountryCode,
				City: geo.City, Region: geo.Region,
			})
		}
	}
	return result, nil
}

func (s *Fail2banService) Unban(req dto.Fail2banUnbanReq) error {
	out, err := exec.Command(f2bClient, "set", req.Jail, "unbanip", req.IP).CombinedOutput()
	if err != nil {
		return fmt.Errorf("unban failed: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

func (s *Fail2banService) GetLogs(lines int) (string, error) {
	if lines <= 0 {
		lines = 200
	}
	if lines > 2000 {
		lines = 2000
	}
	out, err := exec.Command("tail", "-n", strconv.Itoa(lines), f2bLogFile).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("read log failed: %s", strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

// --- internal helpers ---

func detectSSHPort() string {
	out, err := exec.Command("ss", "-tlnp").CombinedOutput()
	if err == nil {
		for _, line := range strings.Split(string(out), "\n") {
			if strings.Contains(line, "sshd") {
				parts := strings.Fields(line)
				for _, p := range parts {
					if idx := strings.LastIndex(p, ":"); idx >= 0 {
						port := p[idx+1:]
						if _, err := strconv.Atoi(port); err == nil && port != "0" {
							if port == "22" {
								return "ssh"
							}
							return port
						}
					}
				}
			}
		}
	}
	return "ssh"
}

func detectBackend() string {
	if _, err := os.Stat("/var/log/auth.log"); err == nil {
		return "auto"
	}
	if _, err := exec.LookPath("journalctl"); err == nil {
		return "systemd"
	}
	return "auto"
}

func (s *Fail2banService) ensureJailLocal() {
	if _, err := os.Stat(f2bJailLocal); os.IsNotExist(err) {
		port := detectSSHPort()
		backend := detectBackend()
		defaultContent := fmt.Sprintf(`# Fail2ban local configuration - managed by X-Panel
# Override settings from jail.conf here

[DEFAULT]
bantime = 90d
findtime = 10m
maxretry = 5

[sshd]
enabled = true
port = %s
filter = sshd
backend = %s
maxretry = 5
findtime = 10m
bantime = 90d
`, port, backend)
		_ = os.WriteFile(f2bJailLocal, []byte(defaultContent), 0644)
	}
}

func (s *Fail2banService) safeWriteConfig(data []byte) error {
	backup := f2bJailLocal + ".bak"
	if orig, err := os.ReadFile(f2bJailLocal); err == nil {
		_ = os.WriteFile(backup, orig, 0644)
	}

	if err := os.WriteFile(f2bJailLocal, data, 0644); err != nil {
		return err
	}

	out, err := exec.Command(f2bClient, "reload").CombinedOutput()
	if err != nil {
		if backupData, bErr := os.ReadFile(backup); bErr == nil {
			_ = os.WriteFile(f2bJailLocal, backupData, 0644)
			_ = exec.Command(f2bClient, "reload").Run()
		}
		return fmt.Errorf("fail2ban reload failed: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

func (s *Fail2banService) getBannedIPs(jail string) []string {
	out, err := exec.Command(f2bClient, "status", jail).CombinedOutput()
	if err != nil {
		return nil
	}
	return parseBannedIPs(string(out))
}

func (s *Fail2banService) readAllJailConfig() map[string]map[string]string {
	result := make(map[string]map[string]string)

	for _, path := range []string{f2bJailLocal} {
		s.parseJailFile(path, result)
	}

	entries, err := os.ReadDir(f2bJailD)
	if err == nil {
		for _, e := range entries {
			if !e.IsDir() && (strings.HasSuffix(e.Name(), ".local") || strings.HasSuffix(e.Name(), ".conf")) {
				s.parseJailFile(filepath.Join(f2bJailD, e.Name()), result)
			}
		}
	}
	return result
}

func (s *Fail2banService) parseJailFile(path string, result map[string]map[string]string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	currentSection := ""

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			currentSection = strings.TrimSuffix(strings.TrimPrefix(line, "["), "]")
			if currentSection != "DEFAULT" {
				if _, ok := result[currentSection]; !ok {
					result[currentSection] = make(map[string]string)
				}
			}
			continue
		}
		if currentSection != "" && currentSection != "DEFAULT" {
			if idx := strings.Index(line, "="); idx > 0 {
				key := strings.TrimSpace(line[:idx])
				val := strings.TrimSpace(line[idx+1:])
				if result[currentSection] == nil {
					result[currentSection] = make(map[string]string)
				}
				result[currentSection][key] = val
			}
		}
	}
}

var jailListRegexp = regexp.MustCompile(`Jail list:\s*(.+)`)

func parseJailList(output string) []string {
	m := jailListRegexp.FindStringSubmatch(output)
	if m == nil {
		return nil
	}
	parts := strings.Split(m[1], ",")
	var jails []string
	for _, p := range parts {
		name := strings.TrimSpace(p)
		if name != "" {
			jails = append(jails, name)
		}
	}
	return jails
}

var bannedIPRegexp = regexp.MustCompile(`Banned IP list:\s*(.*)`)

func parseBannedIPs(output string) []string {
	m := bannedIPRegexp.FindStringSubmatch(output)
	if m == nil {
		return nil
	}
	parts := strings.Split(m[1], " ")
	var ips []string
	for _, p := range parts {
		ip := strings.TrimSpace(p)
		if ip != "" {
			ips = append(ips, ip)
		}
	}
	return ips
}

func replaceOrAppendSection(content, sectionName, newSection string) string {
	header := "[" + sectionName + "]"
	lines := strings.Split(content, "\n")
	var result []string
	inSection := false
	replaced := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == header {
			inSection = true
			replaced = true
			result = append(result, strings.TrimRight(newSection, "\n"))
			continue
		}
		if inSection {
			if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
				inSection = false
				result = append(result, line)
			}
			continue
		}
		result = append(result, line)
	}

	if !replaced {
		result = append(result, "")
		result = append(result, strings.TrimRight(newSection, "\n"))
	}

	return strings.Join(result, "\n") + "\n"
}
