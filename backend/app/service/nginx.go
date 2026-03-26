package service

import (
	"fmt"
	"os"
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
