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
}

type SystemdServiceImpl struct{}

func NewISystemdService() ISystemdService { return &SystemdServiceImpl{} }

func (s *SystemdServiceImpl) List(showAll bool) ([]dto.SystemdServiceInfo, error) {
	args := []string{"list-units", "--type=service", "--no-pager", "--no-legend", "--plain"}
	if showAll {
		args = append(args, "--all")
	}

	out, err := exec.Command("systemctl", args...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("systemctl list-units failed: %s", strings.TrimSpace(string(out)))
	}

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

		enabled := s.isEnabled(name)

		services = append(services, dto.SystemdServiceInfo{
			Name:        name,
			Description: desc,
			LoadState:   fields[1],
			ActiveState: fields[2],
			SubState:    fields[3],
			Enabled:     enabled,
			IsPanel:     strings.HasPrefix(name, panelServicePrefix),
		})
	}

	return services, nil
}

func (s *SystemdServiceImpl) GetDetail(name string) (*dto.SystemdServiceDetail, error) {
	detail := &dto.SystemdServiceDetail{Name: name}
	detail.IsPanel = strings.HasPrefix(name, panelServicePrefix)

	props := s.getProperties(name, "Description", "LoadState", "ActiveState", "SubState",
		"MainPID", "ExecStart", "WorkingDirectory", "User",
		"Restart", "RestartUSec", "Environment",
		"MemoryCurrent", "CPUUsageNSec", "ActiveEnterTimestamp", "FragmentPath")

	detail.Description = props["Description"]
	detail.LoadState = props["LoadState"]
	detail.ActiveState = props["ActiveState"]
	detail.SubState = props["SubState"]
	detail.MainPID, _ = strconv.Atoi(props["MainPID"])
	detail.User = props["User"]
	detail.Restart = props["Restart"]
	detail.UnitFile = props["FragmentPath"]
	detail.StartedAt = props["ActiveEnterTimestamp"]

	detail.ExecStart = parseExecStartValue(props["ExecStart"])
	detail.WorkingDir = props["WorkingDirectory"]
	detail.Environment = props["Environment"]

	if props["RestartUSec"] != "" && props["RestartUSec"] != "0" {
		usec, _ := strconv.ParseInt(props["RestartUSec"], 10, 64)
		if usec > 0 {
			detail.RestartSec = fmt.Sprintf("%ds", usec/1000000)
		}
	}

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

	if detail.UnitFile != "" {
		if content, err := os.ReadFile(detail.UnitFile); err == nil {
			detail.UnitContent = string(content)
		}
	}

	return detail, nil
}

func (s *SystemdServiceImpl) Create(req dto.SystemdServiceCreate) error {
	name := req.Name
	if !strings.HasPrefix(name, panelServicePrefix) {
		name = panelServicePrefix + name
	}

	unitFile := filepath.Join(systemdUnitDir, name+".service")
	if _, err := os.Stat(unitFile); err == nil {
		return fmt.Errorf("service %s already exists", name)
	}

	content := buildUnitFile(name, req.Description, req.ExecStart, req.WorkingDir,
		req.User, req.Restart, req.RestartSec, req.Environment, req.AfterTarget)

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

func (s *SystemdServiceImpl) Update(req dto.SystemdServiceUpdate) error {
	name := req.Name
	unitFile := filepath.Join(systemdUnitDir, name+".service")
	if _, err := os.Stat(unitFile); os.IsNotExist(err) {
		return fmt.Errorf("service %s unit file not found", name)
	}

	content := buildUnitFile(name, req.Description, req.ExecStart, req.WorkingDir,
		req.User, req.Restart, req.RestartSec, req.Environment, req.AfterTarget)

	if err := os.WriteFile(unitFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("write unit file: %w", err)
	}

	out, err := exec.Command("systemctl", "daemon-reload").CombinedOutput()
	if err != nil {
		return fmt.Errorf("daemon-reload failed: %s", strings.TrimSpace(string(out)))
	}

	if s.isActive(name) {
		_ = exec.Command("systemctl", "restart", name).Run()
	}

	return nil
}

func (s *SystemdServiceImpl) Delete(req dto.SystemdServiceDelete) error {
	name := req.Name
	if !strings.HasPrefix(name, panelServicePrefix) {
		return fmt.Errorf("can only delete panel-created services (xp-* prefix)")
	}

	_ = exec.Command("systemctl", "stop", name).Run()
	_ = exec.Command("systemctl", "disable", name).Run()

	unitFile := filepath.Join(systemdUnitDir, name+".service")
	if err := os.Remove(unitFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove unit file: %w", err)
	}

	_ = exec.Command("systemctl", "daemon-reload").Run()
	_ = exec.Command("systemctl", "reset-failed", name).Run()
	return nil
}

func (s *SystemdServiceImpl) Operate(req dto.SystemdServiceOperate) error {
	out, err := exec.Command("systemctl", req.Operation, req.Name).CombinedOutput()
	if err != nil {
		return fmt.Errorf("systemctl %s %s failed: %s", req.Operation, req.Name, strings.TrimSpace(string(out)))
	}
	return nil
}

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

// --- helpers ---

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
		return m[1]
	}
	return raw
}

func buildUnitFile(name, desc, execStart, workDir, user, restart string, restartSec int, env, afterTarget string) string {
	if desc == "" {
		desc = name + " managed by X-Panel"
	}
	if restart == "" {
		restart = "on-failure"
	}
	if afterTarget == "" {
		afterTarget = "network.target"
	}

	var sb strings.Builder
	sb.WriteString("[Unit]\n")
	sb.WriteString(fmt.Sprintf("Description=%s\n", desc))
	sb.WriteString(fmt.Sprintf("After=%s\n\n", afterTarget))

	sb.WriteString("[Service]\n")
	sb.WriteString("Type=simple\n")
	sb.WriteString(fmt.Sprintf("ExecStart=%s\n", execStart))
	if workDir != "" {
		sb.WriteString(fmt.Sprintf("WorkingDirectory=%s\n", workDir))
	}
	if user != "" {
		sb.WriteString(fmt.Sprintf("User=%s\n", user))
	}
	if env != "" {
		for _, e := range strings.Split(env, "\n") {
			e = strings.TrimSpace(e)
			if e != "" {
				sb.WriteString(fmt.Sprintf("Environment=%s\n", e))
			}
		}
	}
	sb.WriteString(fmt.Sprintf("Restart=%s\n", restart))
	if restartSec > 0 {
		sb.WriteString(fmt.Sprintf("RestartSec=%d\n", restartSec))
	}
	sb.WriteString("StandardOutput=journal\n")
	sb.WriteString("StandardError=journal\n\n")

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
