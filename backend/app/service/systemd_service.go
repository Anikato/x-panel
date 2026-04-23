package service

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"xpanel/app/dto"
)

const (
	panelServicePrefix = "xp-"
	systemdUnitDir     = "/etc/systemd/system"
)

type ISystemdService interface {
	List(showAll bool) ([]dto.SystemdServiceInfo, error)
	GetDetail(name string) (*dto.SystemdServiceDetail, error)
	Create(req dto.SystemdServiceCreate) error
	Update(req dto.SystemdServiceUpdate) error
	Delete(req dto.SystemdServiceDelete) error
	Operate(req dto.SystemdServiceOperate) error
	GetLogs(req dto.SystemdServiceLogReq) (string, error)
	GetUnitContent(name string) (string, error)
	SaveUnitContent(name, content string) error
}

type SystemdServiceImpl struct{}

func NewISystemdService() ISystemdService { return &SystemdServiceImpl{} }

// ----- List -----

func (s *SystemdServiceImpl) List(showAll bool) ([]dto.SystemdServiceInfo, error) {
	args := []string{"list-units", "--type=service", "--no-pager", "--no-legend", "--plain"}
	if showAll {
		args = append(args, "--all")
	}
	out, err := exec.Command("systemctl", args...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("systemctl list-units failed: %s", strings.TrimSpace(string(out)))
	}

	// 一次性获取所有 enabled 状态，避免串行调用 is-enabled
	enabledSet := s.listEnabledServices()

	var services []dto.SystemdServiceInfo
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		name := strings.TrimSuffix(fields[0], ".service")
		desc := ""
		if len(fields) >= 5 {
			desc = strings.Join(fields[4:], " ")
		}
		_, enabled := enabledSet[name]

		info := dto.SystemdServiceInfo{
			Name:        name,
			Description: desc,
			LoadState:   fields[1],
			ActiveState: fields[2],
			SubState:    fields[3],
			Enabled:     enabled,
			IsPanel:     strings.HasPrefix(name, panelServicePrefix),
		}

		// 批量 show 运行时数据（只对 active 服务）
		if fields[2] == "active" {
			props := s.getProperties(name, "MainPID", "MemoryCurrent", "NRestarts")
			info.MainPID, _ = strconv.Atoi(props["MainPID"])
			if mem := props["MemoryCurrent"]; mem != "" && mem != "[not set]" {
				if bytes, err := strconv.ParseInt(mem, 10, 64); err == nil && bytes > 0 {
					info.MemoryCurrent = formatBytes(bytes)
				}
			}
			info.RestartCount, _ = strconv.Atoi(props["NRestarts"])
		}

		services = append(services, info)
	}
	return services, nil
}

// listEnabledServices 一次性获取所有 enabled 服务名集合
func (s *SystemdServiceImpl) listEnabledServices() map[string]struct{} {
	out, err := exec.Command("systemctl", "list-unit-files", "--type=service",
		"--no-pager", "--no-legend", "--plain").CombinedOutput()
	enabled := make(map[string]struct{})
	if err != nil {
		return enabled
	}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		state := fields[1]
		if state == "enabled" || state == "enabled-runtime" {
			name := strings.TrimSuffix(fields[0], ".service")
			enabled[name] = struct{}{}
		}
	}
	return enabled
}

// ----- GetDetail -----

func (s *SystemdServiceImpl) GetDetail(name string) (*dto.SystemdServiceDetail, error) {
	detail := &dto.SystemdServiceDetail{Name: name}
	detail.IsPanel = strings.HasPrefix(name, panelServicePrefix)

	props := s.getProperties(name,
		"Description", "LoadState", "ActiveState", "SubState",
		"MainPID", "ExecStart", "ExecStartPre", "ExecStopPost",
		"WorkingDirectory", "User", "Type",
		"Restart", "RestartUSec", "RestartSec", "NRestarts",
		"Environment",
		"MemoryCurrent", "CPUUsageNSec", "ActiveEnterTimestamp", "FragmentPath",
		"StandardOutput", "StandardError",
		"TimeoutStartUSec", "TimeoutStopUSec",
	)

	detail.Description = props["Description"]
	detail.LoadState = props["LoadState"]
	detail.ActiveState = props["ActiveState"]
	detail.SubState = props["SubState"]
	detail.MainPID, _ = strconv.Atoi(props["MainPID"])
	detail.User = props["User"]
	detail.Type = props["Type"]
	detail.Restart = props["Restart"]
	detail.UnitFile = props["FragmentPath"]
	detail.StartedAt = props["ActiveEnterTimestamp"]
	detail.RestartCount, _ = strconv.Atoi(props["NRestarts"])

	detail.ExecStart = parseExecStartValue(props["ExecStart"])
	detail.ExecStartPre = parseExecStartValue(props["ExecStartPre"])
	detail.ExecStopPost = parseExecStartValue(props["ExecStopPost"])
	detail.WorkingDir = props["WorkingDirectory"]

	// Environment: systemctl show 返回空格分隔的 KEY=VALUE 列表
	detail.Environment = parseEnvironmentValue(props["Environment"])

	// StdOutput/StdError
	detail.StdOutput = props["StandardOutput"]
	detail.StdError = props["StandardError"]

	// RestartSec: 优先使用 RestartSec 属性，回退到 RestartUSec
	detail.RestartSec = parseTimeSec(props["RestartSec"], props["RestartUSec"])

	// Timeout
	detail.TimeoutStart = parseTimeoutSec(props["TimeoutStartUSec"])
	detail.TimeoutStop = parseTimeoutSec(props["TimeoutStopUSec"])

	if mem := props["MemoryCurrent"]; mem != "" && mem != "[not set]" {
		if bytes, err := strconv.ParseInt(mem, 10, 64); err == nil && bytes > 0 {
			detail.MemoryCurrent = formatBytes(bytes)
		}
	}
	if cpu := props["CPUUsageNSec"]; cpu != "" && cpu != "[not set]" {
		if ns, err := strconv.ParseInt(cpu, 10, 64); err == nil && ns > 0 {
			detail.CPUUsage = fmt.Sprintf("%.2fs", float64(ns)/1e9)
		}
	}

	detail.Enabled = s.isEnabled(name)

	// 读取 Unit 文件内容，同时从中解析 After= 字段
	if detail.UnitFile != "" {
		if content, err := os.ReadFile(detail.UnitFile); err == nil {
			detail.UnitContent = string(content)
			detail.AfterTarget = parseAfterFromUnit(string(content))
		}
	}

	return detail, nil
}

// ----- Create -----

func (s *SystemdServiceImpl) Create(req dto.SystemdServiceCreate) error {
	name := req.Name
	if !strings.HasPrefix(name, panelServicePrefix) {
		name = panelServicePrefix + name
	}
	unitFile := filepath.Join(systemdUnitDir, name+".service")
	if _, err := os.Stat(unitFile); err == nil {
		return fmt.Errorf("service %s already exists", name)
	}

	content := buildUnitFile(buildUnitParams{
		Name:         name,
		Description:  req.Description,
		Type:         req.Type,
		ExecStart:    req.ExecStart,
		ExecStartPre: req.ExecStartPre,
		ExecStopPost: req.ExecStopPost,
		WorkingDir:   req.WorkingDir,
		User:         req.User,
		Restart:      req.Restart,
		RestartSec:   req.RestartSec,
		Environment:  req.Environment,
		AfterTarget:  req.AfterTarget,
		StdOutput:    req.StdOutput,
		StdError:     req.StdError,
		TimeoutStart: req.TimeoutStart,
		TimeoutStop:  req.TimeoutStop,
	})

	if err := os.WriteFile(unitFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("write unit file: %w", err)
	}
	if out, err := exec.Command("systemctl", "daemon-reload").CombinedOutput(); err != nil {
		return fmt.Errorf("daemon-reload failed: %s", strings.TrimSpace(string(out)))
	}
	if req.AutoStart {
		_ = exec.Command("systemctl", "enable", name).Run()
		_ = exec.Command("systemctl", "start", name).Run()
	}
	return nil
}

// ----- Update -----

func (s *SystemdServiceImpl) Update(req dto.SystemdServiceUpdate) error {
	unitFile := filepath.Join(systemdUnitDir, req.Name+".service")
	if _, err := os.Stat(unitFile); os.IsNotExist(err) {
		// 允许编辑非 xp- 开头的服务，但 unit 文件必须存在
		return fmt.Errorf("unit file not found for service %s", req.Name)
	}

	content := buildUnitFile(buildUnitParams{
		Name:         req.Name,
		Description:  req.Description,
		Type:         req.Type,
		ExecStart:    req.ExecStart,
		ExecStartPre: req.ExecStartPre,
		ExecStopPost: req.ExecStopPost,
		WorkingDir:   req.WorkingDir,
		User:         req.User,
		Restart:      req.Restart,
		RestartSec:   req.RestartSec,
		Environment:  req.Environment,
		AfterTarget:  req.AfterTarget,
		StdOutput:    req.StdOutput,
		StdError:     req.StdError,
		TimeoutStart: req.TimeoutStart,
		TimeoutStop:  req.TimeoutStop,
	})

	if err := os.WriteFile(unitFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("write unit file: %w", err)
	}
	if out, err := exec.Command("systemctl", "daemon-reload").CombinedOutput(); err != nil {
		return fmt.Errorf("daemon-reload failed: %s", strings.TrimSpace(string(out)))
	}
	if s.isActive(req.Name) {
		_ = exec.Command("systemctl", "restart", req.Name).Run()
	}
	return nil
}

// ----- Delete -----

func (s *SystemdServiceImpl) Delete(req dto.SystemdServiceDelete) error {
	if !strings.HasPrefix(req.Name, panelServicePrefix) {
		return fmt.Errorf("can only delete panel-created services (xp-* prefix)")
	}
	_ = exec.Command("systemctl", "stop", req.Name).Run()
	_ = exec.Command("systemctl", "disable", req.Name).Run()
	unitFile := filepath.Join(systemdUnitDir, req.Name+".service")
	if err := os.Remove(unitFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove unit file: %w", err)
	}
	_ = exec.Command("systemctl", "daemon-reload").Run()
	_ = exec.Command("systemctl", "reset-failed", req.Name).Run()
	return nil
}

// ----- Operate -----

func (s *SystemdServiceImpl) Operate(req dto.SystemdServiceOperate) error {
	out, err := exec.Command("systemctl", req.Operation, req.Name).CombinedOutput()
	if err != nil {
		return fmt.Errorf("systemctl %s %s failed: %s", req.Operation, req.Name, strings.TrimSpace(string(out)))
	}
	return nil
}

// ----- GetLogs -----

func (s *SystemdServiceImpl) GetLogs(req dto.SystemdServiceLogReq) (string, error) {
	lines := req.Lines
	if lines <= 0 {
		lines = 100
	}
	if lines > 2000 {
		lines = 2000
	}
	out, err := exec.Command("journalctl", "-u", req.Name, "-n", strconv.Itoa(lines),
		"--no-pager", "--output=short-iso").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("journalctl failed: %s", strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

// ----- GetUnitContent / SaveUnitContent -----

func (s *SystemdServiceImpl) GetUnitContent(name string) (string, error) {
	// 先从 systemctl show 获取 FragmentPath
	props := s.getProperties(name, "FragmentPath")
	unitFile := props["FragmentPath"]
	if unitFile == "" {
		// 回退：尝试标准路径
		unitFile = filepath.Join(systemdUnitDir, name+".service")
	}
	data, err := os.ReadFile(unitFile)
	if err != nil {
		return "", fmt.Errorf("read unit file %s: %w", unitFile, err)
	}
	return string(data), nil
}

func (s *SystemdServiceImpl) SaveUnitContent(name, content string) error {
	// 获取 unit 文件路径
	props := s.getProperties(name, "FragmentPath")
	unitFile := props["FragmentPath"]
	if unitFile == "" {
		unitFile = filepath.Join(systemdUnitDir, name+".service")
	}

	// 安全检查：不允许写到 systemdUnitDir 以外
	absUnit, _ := filepath.Abs(unitFile)
	absDir, _ := filepath.Abs(systemdUnitDir)
	if !strings.HasPrefix(absUnit, absDir+"/") {
		return fmt.Errorf("invalid unit file path: %s", unitFile)
	}

	backup, _ := os.ReadFile(unitFile)
	if err := os.WriteFile(unitFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("write unit file: %w", err)
	}

	if out, err := exec.Command("systemctl", "daemon-reload").CombinedOutput(); err != nil {
		// 回滚
		if backup != nil {
			_ = os.WriteFile(unitFile, backup, 0644)
			_ = exec.Command("systemctl", "daemon-reload").Run()
		}
		return fmt.Errorf("daemon-reload failed: %s", strings.TrimSpace(string(out)))
	}

	if s.isActive(name) {
		_ = exec.Command("systemctl", "restart", name).Run()
	}
	return nil
}

// ----- helpers -----

func (s *SystemdServiceImpl) isEnabled(name string) bool {
	out, err := exec.Command("systemctl", "is-enabled", name).Output()
	if err != nil {
		return false
	}
	state := strings.TrimSpace(string(out))
	return state == "enabled" || state == "enabled-runtime"
}

func (s *SystemdServiceImpl) isActive(name string) bool {
	out, _ := exec.Command("systemctl", "is-active", name).Output()
	return strings.TrimSpace(string(out)) == "active"
}

func (s *SystemdServiceImpl) getProperties(name string, props ...string) map[string]string {
	args := []string{"show", name, "--no-pager"}
	for _, p := range props {
		args = append(args, "-p", p)
	}
	out, err := exec.Command("systemctl", args...).CombinedOutput()
	if err != nil {
		return map[string]string{}
	}
	result := make(map[string]string)
	for _, line := range strings.Split(string(out), "\n") {
		if idx := strings.Index(line, "="); idx > 0 {
			key := strings.TrimSpace(line[:idx])
			val := strings.TrimSpace(line[idx+1:])
			result[key] = val
		}
	}
	return result
}

var execStartPathRe = regexp.MustCompile(`path=([^;]+)`)

func parseExecStartValue(raw string) string {
	if raw == "" || raw == "[not set]" {
		return ""
	}
	if m := execStartPathRe.FindStringSubmatch(raw); len(m) > 1 {
		return strings.TrimSpace(m[1])
	}
	return raw
}

// parseEnvironmentValue: systemctl show 返回格式 "KEY=VAL KEY2=VAL2"（空格分隔）
// 转换为换行分隔，方便前端展示
func parseEnvironmentValue(raw string) string {
	if raw == "" || raw == "[not set]" {
		return ""
	}
	// systemd 用空格分隔多个 Environment 条目，每个条目可能被引号括起
	// 简单处理：尝试按空格分割，但合并引号内的空格
	var result []string
	current := ""
	inQuote := false
	for _, ch := range raw {
		switch ch {
		case '"':
			inQuote = !inQuote
		case ' ':
			if !inQuote && current != "" {
				result = append(result, current)
				current = ""
			} else if inQuote {
				current += string(ch)
			}
		default:
			current += string(ch)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return strings.Join(result, "\n")
}

// parseAfterFromUnit: 从 Unit 文件内容中提取 After= 字段
func parseAfterFromUnit(content string) string {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "After=") {
			return strings.TrimPrefix(line, "After=")
		}
	}
	return ""
}

// parseTimeSec: 解析 RestartSec（可能是 "5s" 格式或 "5000000" 微秒格式）
func parseTimeSec(restartSec, restartUSec string) int {
	// 优先使用 RestartSec 属性（格式如 "5s", "5min", "0"）
	if restartSec != "" && restartSec != "[not set]" {
		// 去掉单位后缀
		if strings.HasSuffix(restartSec, "ms") {
			v, _ := strconv.Atoi(strings.TrimSuffix(restartSec, "ms"))
			return v / 1000
		}
		if strings.HasSuffix(restartSec, "min") {
			v, _ := strconv.Atoi(strings.TrimSuffix(restartSec, "min"))
			return v * 60
		}
		if strings.HasSuffix(restartSec, "s") {
			v, _ := strconv.Atoi(strings.TrimSuffix(restartSec, "s"))
			return v
		}
		if v, err := strconv.Atoi(restartSec); err == nil {
			return v
		}
	}
	// 回退到 RestartUSec（微秒）
	if restartUSec != "" && restartUSec != "[not set]" && restartUSec != "0" {
		usec, _ := strconv.ParseInt(restartUSec, 10, 64)
		if usec > 0 {
			return int(usec / 1_000_000)
		}
	}
	return 0
}

// parseTimeoutSec: 解析 TimeoutStartUSec/TimeoutStopUSec（微秒）→ 秒
func parseTimeoutSec(usecStr string) int {
	if usecStr == "" || usecStr == "[not set]" || usecStr == "infinity" {
		return 0
	}
	// 可能是 "90s", "1min 30s" 等格式
	if strings.Contains(usecStr, "min") || strings.Contains(usecStr, "s") {
		return 0 // 复杂格式忽略
	}
	usec, _ := strconv.ParseInt(usecStr, 10, 64)
	if usec <= 0 {
		return 0
	}
	return int(usec / 1_000_000)
}

// ----- buildUnitFile -----

type buildUnitParams struct {
	Name         string
	Description  string
	Type         string
	ExecStart    string
	ExecStartPre string
	ExecStopPost string
	WorkingDir   string
	User         string
	Restart      string
	RestartSec   int
	Environment  string
	AfterTarget  string
	StdOutput    string
	StdError     string
	TimeoutStart int
	TimeoutStop  int
}

func buildUnitFile(p buildUnitParams) string {
	if p.Description == "" {
		p.Description = p.Name + " managed by X-Panel"
	}
	if p.Restart == "" {
		p.Restart = "on-failure"
	}
	if p.AfterTarget == "" {
		p.AfterTarget = "network.target"
	}
	if p.Type == "" {
		p.Type = "simple"
	}
	if p.StdOutput == "" {
		p.StdOutput = "journal"
	}
	if p.StdError == "" {
		p.StdError = "journal"
	}

	var sb strings.Builder
	sb.WriteString("[Unit]\n")
	sb.WriteString(fmt.Sprintf("Description=%s\n", p.Description))
	sb.WriteString(fmt.Sprintf("After=%s\n\n", p.AfterTarget))

	sb.WriteString("[Service]\n")
	sb.WriteString(fmt.Sprintf("Type=%s\n", p.Type))
	if p.ExecStartPre != "" {
		sb.WriteString(fmt.Sprintf("ExecStartPre=%s\n", p.ExecStartPre))
	}
	sb.WriteString(fmt.Sprintf("ExecStart=%s\n", p.ExecStart))
	if p.ExecStopPost != "" {
		sb.WriteString(fmt.Sprintf("ExecStopPost=%s\n", p.ExecStopPost))
	}
	if p.WorkingDir != "" {
		sb.WriteString(fmt.Sprintf("WorkingDirectory=%s\n", p.WorkingDir))
	}
	if p.User != "" {
		sb.WriteString(fmt.Sprintf("User=%s\n", p.User))
	}
	// Environment: 每行一个 KEY=VALUE 条目
	if p.Environment != "" {
		for _, e := range strings.Split(p.Environment, "\n") {
			e = strings.TrimSpace(e)
			if e != "" {
				sb.WriteString(fmt.Sprintf("Environment=%s\n", e))
			}
		}
	}
	sb.WriteString(fmt.Sprintf("Restart=%s\n", p.Restart))
	if p.RestartSec > 0 {
		sb.WriteString(fmt.Sprintf("RestartSec=%d\n", p.RestartSec))
	}
	if p.TimeoutStart > 0 {
		sb.WriteString(fmt.Sprintf("TimeoutStartSec=%d\n", p.TimeoutStart))
	}
	if p.TimeoutStop > 0 {
		sb.WriteString(fmt.Sprintf("TimeoutStopSec=%d\n", p.TimeoutStop))
	}
	sb.WriteString(fmt.Sprintf("StandardOutput=%s\n", p.StdOutput))
	sb.WriteString(fmt.Sprintf("StandardError=%s\n\n", p.StdError))

	sb.WriteString("[Install]\n")
	sb.WriteString("WantedBy=multi-user.target\n")

	return sb.String()
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %s", float64(b)/float64(div), []string{"KB", "MB", "GB", "TB"}[exp])
}
