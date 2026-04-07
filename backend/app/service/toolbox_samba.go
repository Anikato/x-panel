package service

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"xpanel/app/dto"
	"xpanel/buserr"
	"xpanel/utils/samba"
)

const smbConfPath = "/etc/samba/smb.conf"

type ISambaService interface {
	GetStatus() (*dto.ServiceStatus, error)
	Install() error
	Uninstall() error
	Operate(req dto.ServiceOperate) error

	ListShares() ([]dto.SambaShare, error)
	CreateShare(req dto.SambaShareCreate) error
	UpdateShare(req dto.SambaShareUpdate) error
	DeleteShare(req dto.SambaShareDelete) error

	ListUsers() ([]dto.SambaUser, error)
	CreateUser(req dto.SambaUserCreate) error
	DeleteUser(req dto.SambaUserDelete) error
	UpdatePassword(req dto.SambaPasswordUpdate) error
	ToggleUser(req dto.SambaUserToggle) error

	GetGlobalConfig() (*dto.SambaGlobalConfig, error)
	UpdateGlobalConfig(req dto.SambaGlobalConfig) error

	GetConnections() (*dto.SambaConnections, error)
}

type SambaService struct{}

func NewISambaService() ISambaService { return &SambaService{} }

// ====== Service Management ======

func (s *SambaService) GetStatus() (*dto.ServiceStatus, error) {
	st := &dto.ServiceStatus{}

	out, err := exec.Command("dpkg", "-l", "samba").CombinedOutput()
	if err != nil || !strings.Contains(string(out), "ii  samba") {
		return st, nil
	}
	st.IsInstalled = true

	if out, err := exec.Command("systemctl", "is-active", "smbd").Output(); err == nil {
		st.IsRunning = strings.TrimSpace(string(out)) == "active"
	}

	if out, err := exec.Command("smbd", "--version").Output(); err == nil {
		st.Version = strings.TrimSpace(string(out))
	}

	if out, err := exec.Command("systemctl", "is-enabled", "smbd").Output(); err == nil {
		st.AutoStart = strings.TrimSpace(string(out)) == "enabled"
	}

	return st, nil
}

func (s *SambaService) Install() error {
	out, err := exec.Command("apt", "install", "-y", "samba").CombinedOutput()
	if err != nil {
		return buserr.WithDetail("ErrSambaInstall", string(out), err)
	}
	return nil
}

func (s *SambaService) Uninstall() error {
	out, err := exec.Command("apt", "remove", "-y", "samba").CombinedOutput()
	if err != nil {
		return buserr.WithDetail("ErrSambaUninstall", string(out), err)
	}
	return nil
}

func (s *SambaService) Operate(req dto.ServiceOperate) error {
	switch req.Operation {
	case "start", "stop", "restart":
		out, err := exec.Command("systemctl", req.Operation, "smbd").CombinedOutput()
		if err != nil {
			return fmt.Errorf("%s smbd failed: %s", req.Operation, strings.TrimSpace(string(out)))
		}
		if req.Operation == "start" || req.Operation == "restart" {
			_ = exec.Command("systemctl", req.Operation, "nmbd").Run()
		}
	case "enable":
		_ = exec.Command("systemctl", "enable", "smbd").Run()
		_ = exec.Command("systemctl", "enable", "nmbd").Run()
	case "disable":
		_ = exec.Command("systemctl", "disable", "smbd").Run()
		_ = exec.Command("systemctl", "disable", "nmbd").Run()
	default:
		return fmt.Errorf("unsupported operation: %s", req.Operation)
	}
	return nil
}

// ====== Share Management ======

func (s *SambaService) ListShares() ([]dto.SambaShare, error) {
	cfg, err := samba.Parse(smbConfPath)
	if err != nil {
		return nil, buserr.WithErr("ErrSambaReadConf", err)
	}

	var shares []dto.SambaShare
	for _, sec := range cfg.GetShares() {
		share := dto.SambaShare{
			Name:       sec.Name,
			Path:       sec.Params["path"],
			Comment:    sec.Params["comment"],
			Writable:   paramBool(sec.Params, "writable", false) || !paramBool(sec.Params, "read only", true),
			GuestOK:    paramBool(sec.Params, "guest ok", false),
			Browseable: paramBool(sec.Params, "browseable", true),
			ValidUsers: sec.Params["valid users"],
		}
		shares = append(shares, share)
	}
	return shares, nil
}

func (s *SambaService) CreateShare(req dto.SambaShareCreate) error {
	cfg, err := samba.Parse(smbConfPath)
	if err != nil {
		return buserr.WithErr("ErrSambaReadConf", err)
	}

	if cfg.GetSection(req.Name) != nil {
		return buserr.WithName("ErrSambaShareExist", req.Name)
	}

	if req.CreateDir {
		if err := os.MkdirAll(req.Path, 0777); err != nil {
			return fmt.Errorf("create directory failed: %v", err)
		}
	}

	sec := samba.NewShareSection(req.Name, req.Path, req.Comment, req.Writable, req.GuestOK, req.ValidUsers)
	cfg.AddSection(sec)

	return s.safeWriteConfig(cfg)
}

func (s *SambaService) UpdateShare(req dto.SambaShareUpdate) error {
	cfg, err := samba.Parse(smbConfPath)
	if err != nil {
		return buserr.WithErr("ErrSambaReadConf", err)
	}

	cfg.RemoveSection(req.OrigName)

	sec := samba.NewShareSection(req.Name, req.Path, req.Comment, req.Writable, req.GuestOK, req.ValidUsers)
	cfg.AddSection(sec)

	return s.safeWriteConfig(cfg)
}

func (s *SambaService) DeleteShare(req dto.SambaShareDelete) error {
	cfg, err := samba.Parse(smbConfPath)
	if err != nil {
		return buserr.WithErr("ErrSambaReadConf", err)
	}

	cfg.RemoveSection(req.Name)
	return s.safeWriteConfig(cfg)
}

// ====== User Management ======

func (s *SambaService) ListUsers() ([]dto.SambaUser, error) {
	out, err := exec.Command("pdbedit", "-L", "-v").CombinedOutput()
	if err != nil {
		return nil, buserr.WithErr("ErrSambaListUsers", err)
	}

	var users []dto.SambaUser
	var current *dto.SambaUser
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "Unix username:") {
			if current != nil {
				users = append(users, *current)
			}
			current = &dto.SambaUser{
				Username: strings.TrimSpace(strings.TrimPrefix(line, "Unix username:")),
			}
		}
		if current != nil && strings.HasPrefix(line, "Account Flags:") {
			current.Flags = strings.TrimSpace(strings.TrimPrefix(line, "Account Flags:"))
		}
	}
	if current != nil {
		users = append(users, *current)
	}
	return users, nil
}

func (s *SambaService) CreateUser(req dto.SambaUserCreate) error {
	if _, err := exec.Command("id", req.Username).Output(); err != nil {
		out, err := exec.Command("useradd", "-M", "-s", "/usr/sbin/nologin", req.Username).CombinedOutput()
		if err != nil {
			return fmt.Errorf("create system user failed: %s", strings.TrimSpace(string(out)))
		}
	}

	cmd := exec.Command("smbpasswd", "-a", "-s", req.Username)
	cmd.Stdin = strings.NewReader(req.Password + "\n" + req.Password + "\n")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return buserr.WithDetail("ErrSambaCreateUser", string(out), err)
	}
	return nil
}

func (s *SambaService) DeleteUser(req dto.SambaUserDelete) error {
	out, err := exec.Command("smbpasswd", "-x", req.Username).CombinedOutput()
	if err != nil {
		return buserr.WithDetail("ErrSambaDeleteUser", string(out), err)
	}
	return nil
}

func (s *SambaService) UpdatePassword(req dto.SambaPasswordUpdate) error {
	cmd := exec.Command("smbpasswd", "-s", req.Username)
	cmd.Stdin = strings.NewReader(req.Password + "\n" + req.Password + "\n")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return buserr.WithDetail("ErrSambaUpdatePassword", string(out), err)
	}
	return nil
}

func (s *SambaService) ToggleUser(req dto.SambaUserToggle) error {
	flag := "-d"
	if req.Enabled {
		flag = "-e"
	}
	out, err := exec.Command("smbpasswd", flag, req.Username).CombinedOutput()
	if err != nil {
		return buserr.WithDetail("ErrSambaToggleUser", string(out), err)
	}
	return nil
}

// ====== Global Config ======

func (s *SambaService) GetGlobalConfig() (*dto.SambaGlobalConfig, error) {
	cfg, err := samba.Parse(smbConfPath)
	if err != nil {
		return nil, buserr.WithErr("ErrSambaReadConf", err)
	}

	global := cfg.GetGlobal()
	if global == nil {
		return &dto.SambaGlobalConfig{}, nil
	}

	return &dto.SambaGlobalConfig{
		Workgroup:  global.Params["workgroup"],
		ServerName: global.Params["server string"],
		Security:   global.Params["security"],
		MapToGuest: global.Params["map to guest"],
		LogLevel:   global.Params["log level"],
		MaxLogSize: global.Params["max log size"],
		Interfaces: global.Params["interfaces"],
	}, nil
}

func (s *SambaService) UpdateGlobalConfig(req dto.SambaGlobalConfig) error {
	cfg, err := samba.Parse(smbConfPath)
	if err != nil {
		return buserr.WithErr("ErrSambaReadConf", err)
	}

	global := cfg.GetGlobal()
	if global == nil {
		global = &samba.Section{
			Name:   "global",
			Params: make(map[string]string),
		}
		sec := []*samba.Section{global}
		cfg.Sections = append(sec, cfg.Sections...)
	}

	setParam(global, "workgroup", req.Workgroup)
	setParam(global, "server string", req.ServerName)
	setParam(global, "security", req.Security)
	setParam(global, "map to guest", req.MapToGuest)
	setParam(global, "log level", req.LogLevel)
	setParam(global, "max log size", req.MaxLogSize)
	setParam(global, "interfaces", req.Interfaces)

	return s.safeWriteConfig(cfg)
}

// ====== Connections ======

func (s *SambaService) GetConnections() (*dto.SambaConnections, error) {
	result := &dto.SambaConnections{}

	if out, err := exec.Command("smbstatus", "-p", "--no-header", "-j").CombinedOutput(); err == nil {
		result.Processes = parseSmbstatusProcesses(string(out))
	} else {
		if out, err := exec.Command("smbstatus", "-p", "--no-header").CombinedOutput(); err == nil {
			result.Processes = parseSmbstatusProcessesText(string(out))
		}
	}

	if out, err := exec.Command("smbstatus", "-S", "--no-header").CombinedOutput(); err == nil {
		result.Shares = parseSmbstatusShares(string(out))
	}

	return result, nil
}

// ====== Helpers ======

func (s *SambaService) safeWriteConfig(cfg *samba.Config) error {
	backup := smbConfPath + ".bak"
	if data, err := os.ReadFile(smbConfPath); err == nil {
		_ = os.WriteFile(backup, data, 0644)
	}

	if err := cfg.Write(smbConfPath); err != nil {
		return err
	}

	out, err := exec.Command("testparm", "-s", smbConfPath).CombinedOutput()
	if err != nil {
		if bak, readErr := os.ReadFile(backup); readErr == nil {
			_ = os.WriteFile(smbConfPath, bak, 0644)
		}
		return buserr.WithDetail("ErrSambaConfigTest", string(out), err)
	}

	_ = exec.Command("systemctl", "reload", "smbd").Run()
	return nil
}

func paramBool(params map[string]string, key string, def bool) bool {
	val, ok := params[key]
	if !ok {
		return def
	}
	val = strings.ToLower(strings.TrimSpace(val))
	return val == "yes" || val == "true" || val == "1"
}

func setParam(sec *samba.Section, key, value string) {
	if value != "" {
		sec.Params[key] = value
		for i, l := range sec.Lines {
			if l.Type == "param" && l.Key == key {
				sec.Lines[i].Value = value
				return
			}
		}
		sec.Lines = append(sec.Lines, samba.Line{Type: "param", Key: key, Value: value})
	}
}

func parseSmbstatusProcesses(output string) []dto.SambaConnection {
	var conns []dto.SambaConnection
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "-") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 4 {
			conn := dto.SambaConnection{
				PID:      fields[0],
				Username: fields[1],
				Group:    fields[2],
				Machine:  fields[3],
			}
			if len(fields) >= 5 {
				conn.Protocol = fields[4]
			}
			if len(fields) >= 6 {
				conn.Encryption = fields[5]
			}
			conns = append(conns, conn)
		}
	}
	return conns
}

func parseSmbstatusProcessesText(output string) []dto.SambaConnection {
	return parseSmbstatusProcesses(output)
}

func parseSmbstatusShares(output string) []dto.SambaShareUsage {
	var shares []dto.SambaShareUsage
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "-") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 4 {
			share := dto.SambaShareUsage{
				Service: fields[0],
				PID:     fields[1],
				Machine: fields[2],
			}
			if len(fields) >= 5 {
				share.ConnectedAt = fields[3] + " " + fields[4]
			}
			shares = append(shares, share)
		}
	}
	return shares
}
