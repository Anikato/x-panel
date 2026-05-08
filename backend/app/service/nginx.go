package service

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"xpanel/app/dto"
	"xpanel/app/repo"
	"xpanel/buserr"
	"xpanel/constant"
	"xpanel/global"
	"xpanel/utils/cmd"
)

type INginxService interface {
	GetStatus() (*dto.NginxStatus, error)
	Operate(req dto.NginxOperateReq) error
	TestConfig() (*dto.NginxConfigTestResult, error)
	TestConfigDetail() (*dto.NginxConfigTestDetail, error)
	GetIncludeTree() (*dto.NginxIncludeNode, error)
	SetAutoStart(enable bool) error
}

type NginxService struct{}

func NewINginxService() INginxService { return &NginxService{} }

func (s *NginxService) GetStatus() (*dto.NginxStatus, error) {
	nc := global.CONF.Nginx
	status := &dto.NginxStatus{
		InstallDir:       nc.InstallDir,
		IsInstalled:      nc.IsInstalled(),
		SystemMode:       nc.IsSystemMode(),
		HasBothInstalled: nc.HasBothInstalled(),
	}

	websiteRepo := repo.NewIWebsiteRepo()
	status.WebsiteCount, _ = websiteRepo.Count()

	if !status.IsInstalled {
		return status, nil
	}

	status.Version = nc.GetVersion()

	pid, running := s.readPID()
	status.IsRunning = running
	status.PID = pid

	if running && pid > 0 {
		status.StartedAt = getProcessStartTime(pid)
	}

	testResult, _ := s.TestConfig()
	if testResult != nil {
		status.ConfigOK = testResult.Success
	}

	status.AutoStart = s.isAutoStartEnabled()

	return status, nil
}

func (s *NginxService) Operate(req dto.NginxOperateReq) error {
	nc := global.CONF.Nginx
	if !nc.IsInstalled() {
		return buserr.New(constant.ErrNginxNotInstalled)
	}

	switch req.Operation {
	case "start":
		return s.start()
	case "stop":
		return s.stop()
	case "quit":
		return s.signalOrSystemctl("quit")
	case "reload":
		return s.reload()
	case "reopen":
		return s.signalOrSystemctl("reopen")
	default:
		return fmt.Errorf("unsupported operation: %s", req.Operation)
	}
}

func (s *NginxService) TestConfig() (*dto.NginxConfigTestResult, error) {
	nc := global.CONF.Nginx
	if !nc.IsInstalled() {
		return nil, buserr.New(constant.ErrNginxNotInstalled)
	}

	var output string
	var err error
	if nc.IsSystemMode() {
		output, err = cmd.ExecWithOutput(nc.GetBinary(), "-t")
	} else {
		output, err = cmd.ExecWithOutput(nc.GetBinary(), "-p", nc.InstallDir, "-t")
	}
	if err != nil {
		errMsg := output
		if errMsg == "" {
			errMsg = err.Error()
		}
		return &dto.NginxConfigTestResult{Success: false, Output: errMsg}, nil
	}
	return &dto.NginxConfigTestResult{Success: true, Output: output}, nil
}

func (s *NginxService) TestConfigDetail() (*dto.NginxConfigTestDetail, error) {
	result, err := s.TestConfig()
	if err != nil {
		return nil, err
	}
	return &dto.NginxConfigTestDetail{
		Success: result.Success,
		Output:  result.Output,
		Issues:  parseNginxConfigIssues(result.Output),
	}, nil
}

var nginxIssueRe = regexp.MustCompile(`nginx:\s+\[(\w+)\]\s+(.+?)\s+in\s+(.+?):(\d+)`)

func parseNginxConfigIssues(output string) []dto.NginxConfigIssue {
	var issues []dto.NginxConfigIssue
	for _, line := range strings.Split(output, "\n") {
		m := nginxIssueRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		lineNo, _ := strconv.Atoi(m[4])
		issues = append(issues, dto.NginxConfigIssue{
			Level:   m[1],
			Message: strings.TrimSpace(m[2]),
			File:    m[3],
			Line:    lineNo,
		})
	}
	return issues
}

func (s *NginxService) GetIncludeTree() (*dto.NginxIncludeNode, error) {
	nc := global.CONF.Nginx
	if !nc.IsInstalled() {
		return nil, buserr.New(constant.ErrNginxNotInstalled)
	}
	visited := make(map[string]bool)
	node := s.buildIncludeNode(nc.GetMainConf(), visited, 0)
	return &node, nil
}

func (s *NginxService) buildIncludeNode(path string, visited map[string]bool, depth int) dto.NginxIncludeNode {
	path = filepath.Clean(path)
	node := dto.NginxIncludeNode{Path: path}
	if depth > 8 || visited[path] {
		return node
	}
	visited[path] = true
	data, err := os.ReadFile(path)
	if err != nil {
		node.Exists = false
		return node
	}
	node.Exists = true
	baseDir := filepath.Dir(path)
	for _, inc := range parseIncludeLines(string(data), baseDir) {
		matches, err := filepath.Glob(inc)
		if err != nil || len(matches) == 0 {
			node.Children = append(node.Children, dto.NginxIncludeNode{Path: inc, Exists: false})
			continue
		}
		for _, match := range matches {
			if info, err := os.Stat(match); err == nil && !info.IsDir() {
				node.Children = append(node.Children, s.buildIncludeNode(match, visited, depth+1))
			}
		}
	}
	return node
}

var nginxIncludeRe = regexp.MustCompile(`(?i)^\s*include\s+([^;]+);`)

func parseIncludeLines(content, baseDir string) []string {
	var includes []string
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") {
			continue
		}
		m := nginxIncludeRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		p := strings.Trim(m[1], `"'`)
		if !filepath.IsAbs(p) {
			p = filepath.Join(baseDir, p)
		}
		includes = append(includes, filepath.Clean(p))
	}
	return includes
}

func (s *NginxService) start() error {
	_, running := s.readPID()
	if running {
		return buserr.New(constant.ErrNginxAlreadyRunning)
	}

	testResult, err := s.TestConfig()
	if err != nil {
		return err
	}
	if !testResult.Success {
		return buserr.WithDetail(constant.ErrNginxConfigTest, testResult.Output, nil)
	}

	nc := global.CONF.Nginx
	if nc.IsSystemMode() {
		output, err := cmd.ExecWithOutput("systemctl", "start", "nginx")
		if err != nil {
			return buserr.WithDetail(constant.ErrInternalServer,
				fmt.Sprintf("start failed: %s %v", output, err), err)
		}
	} else {
		output, err := cmd.ExecWithOutput(nc.GetBinary(), "-p", nc.InstallDir)
		if err != nil {
			return buserr.WithDetail(constant.ErrInternalServer,
				fmt.Sprintf("start failed: %s %v", output, err), err)
		}
	}
	global.LOG.Info("Nginx started")
	return nil
}

func (s *NginxService) stop() error {
	nc := global.CONF.Nginx
	if nc.IsSystemMode() {
		output, err := cmd.ExecWithOutput("systemctl", "stop", "nginx")
		if err != nil {
			return buserr.WithDetail(constant.ErrInternalServer,
				fmt.Sprintf("stop failed: %s %v", output, err), err)
		}
		global.LOG.Info("Nginx stopped via systemctl")
		return nil
	}
	return s.signalOrSystemctl("stop")
}

func (s *NginxService) reload() error {
	_, running := s.readPID()
	if !running {
		return buserr.New(constant.ErrNginxNotRunning)
	}

	testResult, err := s.TestConfig()
	if err != nil {
		return err
	}
	if !testResult.Success {
		return buserr.WithDetail(constant.ErrNginxConfigTest, testResult.Output, nil)
	}

	nc := global.CONF.Nginx
	if nc.IsSystemMode() {
		output, err := cmd.ExecWithOutput("systemctl", "reload", "nginx")
		if err != nil {
			return buserr.WithDetail(constant.ErrInternalServer,
				fmt.Sprintf("reload failed: %s %v", output, err), err)
		}
	} else {
		output, err := cmd.ExecWithOutput(nc.GetBinary(), "-p", nc.InstallDir, "-s", "reload")
		if err != nil {
			return buserr.WithDetail(constant.ErrInternalServer,
				fmt.Sprintf("reload failed: %s %v", output, err), err)
		}
	}
	global.LOG.Info("Nginx reloaded")
	return nil
}

func (s *NginxService) signalOrSystemctl(sig string) error {
	_, running := s.readPID()
	if !running {
		return buserr.New(constant.ErrNginxNotRunning)
	}

	nc := global.CONF.Nginx
	var output string
	var err error
	if nc.IsSystemMode() {
		output, err = cmd.ExecWithOutput(nc.GetBinary(), "-s", sig)
	} else {
		output, err = cmd.ExecWithOutput(nc.GetBinary(), "-p", nc.InstallDir, "-s", sig)
	}
	if err != nil {
		return buserr.WithDetail(constant.ErrInternalServer,
			fmt.Sprintf("signal %s failed: %s %v", sig, output, err), err)
	}
	global.LOG.Infof("Nginx signal sent: %s", sig)
	return nil
}

func (s *NginxService) readPID() (int, bool) {
	pidPath := global.CONF.Nginx.GetPidPath()
	data, err := os.ReadFile(pidPath)
	if err != nil {
		return 0, false
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid <= 0 {
		return 0, false
	}

	if err := syscall.Kill(pid, 0); err != nil {
		return pid, false
	}
	return pid, true
}

func getProcessStartTime(pid int) time.Time {
	procPath := fmt.Sprintf("/proc/%d", pid)
	info, err := os.Stat(procPath)
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}

func (s *NginxService) SetAutoStart(enable bool) error {
	nc := global.CONF.Nginx
	if !nc.IsInstalled() {
		return buserr.New(constant.ErrNginxNotInstalled)
	}

	serviceName := s.getServiceName()

	if !nc.IsSystemMode() {
		if err := EnsureNginxServiceFile(nc.InstallDir); err != nil {
			return buserr.WithDetail(constant.ErrInternalServer, "create service file: "+err.Error(), err)
		}
	}

	var args []string
	if enable {
		args = []string{"systemctl", "enable", serviceName}
	} else {
		args = []string{"systemctl", "disable", serviceName}
	}
	output, err := cmd.ExecWithOutput(args[0], args[1:]...)
	if err != nil {
		return buserr.WithDetail(constant.ErrInternalServer,
			fmt.Sprintf("systemctl failed: %s %v", output, err), err)
	}
	global.LOG.Infof("Nginx autostart set to %v", enable)
	return nil
}

func (s *NginxService) isAutoStartEnabled() bool {
	serviceName := s.getServiceName()
	output, _ := cmd.ExecWithOutput("systemctl", "is-enabled", serviceName)
	return strings.TrimSpace(output) == "enabled"
}

func (s *NginxService) getServiceName() string {
	if global.CONF.Nginx.IsSystemMode() {
		return "nginx"
	}
	return "xpanel-nginx"
}

const nginxServicePath = "/etc/systemd/system/xpanel-nginx.service"

func EnsureNginxServiceFile(installDir string) error {
	if _, err := os.Stat(nginxServicePath); err == nil {
		return nil
	}
	content := fmt.Sprintf(`[Unit]
Description=Nginx HTTP Server (X-Panel managed)
After=network.target

[Service]
Type=forking
PIDFile=%s/logs/nginx.pid
ExecStart=%s/sbin/nginx -p %s
ExecReload=%s/sbin/nginx -p %s -s reload
ExecStop=%s/sbin/nginx -p %s -s quit

[Install]
WantedBy=multi-user.target
`, installDir, installDir, installDir, installDir, installDir, installDir, installDir)

	if err := os.WriteFile(nginxServicePath, []byte(content), 0644); err != nil {
		return err
	}
	_, _ = cmd.ExecWithOutput("systemctl", "daemon-reload")
	return nil
}
