package service

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	"xpanel/app/dto"
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

// GetStatus 获取 Nginx 运行状态
func (s *NginxService) GetStatus() (*dto.NginxStatus, error) {
	nc := global.CONF.Nginx
	status := &dto.NginxStatus{
		InstallDir:  nc.InstallDir,
		IsInstalled: nc.IsInstalled(),
	}

	if !status.IsInstalled {
		return status, nil
	}

	// 获取版本
	version, err := cmd.ExecWithOutput(nc.GetBinary(), "-p", nc.InstallDir, "-v")
	if err == nil {
		// nginx -v 输出到 stderr: "nginx version: nginx/1.26.2"
		status.Version = parseNginxVersion(version)
	} else {
		// nginx -v 在某些版本输出到 stderr，ExecWithOutput 会捕获 stderr
		status.Version = parseNginxVersion(err.Error())
	}

	// 检查进程是否运行
	pid, running := s.readPID()
	status.IsRunning = running
	status.PID = pid

	// 获取启动时间（通过 /proc/PID/stat）
	if running && pid > 0 {
		status.StartedAt = getProcessStartTime(pid)
	}

	// 配置测试
	testResult, _ := s.TestConfig()
	if testResult != nil {
		status.ConfigOK = testResult.Success
	}

	// 开机自启状态
	status.AutoStart = s.isAutoStartEnabled()

	return status, nil
}

// Operate 执行 Nginx 操作（start/stop/reload/reopen/quit）
func (s *NginxService) Operate(req dto.NginxOperateReq) error {
	nc := global.CONF.Nginx
	if !nc.IsInstalled() {
		return buserr.New(constant.ErrNginxNotInstalled)
	}

	nginxBin := nc.GetBinary()

	switch req.Operation {
	case "start":
		return s.start(nginxBin)
	case "stop":
		return s.signal(nginxBin, "stop")
	case "quit":
		return s.signal(nginxBin, "quit")
	case "reload":
		return s.reload(nginxBin)
	case "reopen":
		return s.signal(nginxBin, "reopen")
	default:
		return fmt.Errorf("unsupported operation: %s", req.Operation)
	}
}

// TestConfig 执行 nginx -t 配置测试
func (s *NginxService) TestConfig() (*dto.NginxConfigTestResult, error) {
	nc := global.CONF.Nginx
	if !nc.IsInstalled() {
		return nil, buserr.New(constant.ErrNginxNotInstalled)
	}

	output, err := cmd.ExecWithOutput(nc.GetBinary(), "-p", nc.InstallDir, "-t")
	if err != nil {
		// nginx -t 的输出通常在 stderr
		errMsg := output
		if errMsg == "" {
			errMsg = err.Error()
		}
		return &dto.NginxConfigTestResult{
			Success: false,
			Output:  errMsg,
		}, nil
	}

	return &dto.NginxConfigTestResult{
		Success: true,
		Output:  output,
	}, nil
}

// start 启动 Nginx
func (s *NginxService) start(nginxBin string) error {
	_, running := s.readPID()
	if running {
		return buserr.New(constant.ErrNginxAlreadyRunning)
	}

	// 先检查配置
	testResult, err := s.TestConfig()
	if err != nil {
		return err
	}
	if !testResult.Success {
		return buserr.WithDetail(constant.ErrNginxConfigTest, testResult.Output, nil)
	}

	installDir := global.CONF.Nginx.InstallDir
	output, err := cmd.ExecWithOutput(nginxBin, "-p", installDir)
	if err != nil {
		return buserr.WithDetail(constant.ErrInternalServer,
			fmt.Sprintf("start failed: %s %v", output, err), err)
	}
	global.LOG.Info("Nginx started")
	return nil
}

// reload 安全重载：先 nginx -t 测试，通过后再 -s reload
func (s *NginxService) reload(nginxBin string) error {
	_, running := s.readPID()
	if !running {
		return buserr.New(constant.ErrNginxNotRunning)
	}

	// 先测试配置
	testResult, err := s.TestConfig()
	if err != nil {
		return err
	}
	if !testResult.Success {
		return buserr.WithDetail(constant.ErrNginxConfigTest, testResult.Output, nil)
	}

	installDir := global.CONF.Nginx.InstallDir
	output, err := cmd.ExecWithOutput(nginxBin, "-p", installDir, "-s", "reload")
	if err != nil {
		return buserr.WithDetail(constant.ErrInternalServer,
			fmt.Sprintf("reload failed: %s %v", output, err), err)
	}
	global.LOG.Info("Nginx reloaded")
	return nil
}

// signal 发送信号给 Nginx（stop/quit/reopen）
func (s *NginxService) signal(nginxBin, sig string) error {
	_, running := s.readPID()
	if !running {
		return buserr.New(constant.ErrNginxNotRunning)
	}

	installDir := global.CONF.Nginx.InstallDir
	output, err := cmd.ExecWithOutput(nginxBin, "-p", installDir, "-s", sig)
	if err != nil {
		return buserr.WithDetail(constant.ErrInternalServer,
			fmt.Sprintf("signal %s failed: %s %v", sig, output, err), err)
	}
	global.LOG.Infof("Nginx signal sent: %s", sig)
	return nil
}

// readPID 从 pid 文件中读取 PID 并检查进程是否存活
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

	// 在 Unix 上，通过发送 signal 0 检查进程是否存活
	if err := syscall.Kill(pid, 0); err != nil {
		return pid, false
	}
	return pid, true
}

// parseNginxVersion 从 nginx -v 输出中解析版本号
func parseNginxVersion(output string) string {
	// 输出格式: "nginx version: nginx/1.26.2"
	if idx := strings.Index(output, "nginx/"); idx >= 0 {
		ver := output[idx+len("nginx/"):]
		ver = strings.TrimSpace(ver)
		// 截取到第一个空格或换行
		if spIdx := strings.IndexAny(ver, " \n\r"); spIdx >= 0 {
			ver = ver[:spIdx]
		}
		return ver
	}
	return ""
}

// getProcessStartTime 通过 /proc 获取进程启动时间
func getProcessStartTime(pid int) time.Time {
	procPath := fmt.Sprintf("/proc/%d", pid)
	info, err := os.Stat(procPath)
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}

const nginxServiceName = "xpanel-nginx"
const nginxServicePath = "/etc/systemd/system/xpanel-nginx.service"

// SetAutoStart 设置 Nginx 开机自启
func (s *NginxService) SetAutoStart(enable bool) error {
	nc := global.CONF.Nginx
	if !nc.IsInstalled() {
		return buserr.New(constant.ErrNginxNotInstalled)
	}

	if err := EnsureNginxServiceFile(nc.InstallDir); err != nil {
		return buserr.WithDetail(constant.ErrInternalServer, "create service file: "+err.Error(), err)
	}

	var args []string
	if enable {
		args = []string{"systemctl", "enable", nginxServiceName}
	} else {
		args = []string{"systemctl", "disable", nginxServiceName}
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
	output, _ := cmd.ExecWithOutput("systemctl", "is-enabled", nginxServiceName)
	return strings.TrimSpace(output) == "enabled"
}

// EnsureNginxServiceFile 确保 systemd service 文件存在
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
