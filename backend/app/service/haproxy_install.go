package service

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"xpanel/app/dto"
	"xpanel/app/repo"
	"xpanel/buserr"
	"xpanel/constant"
	"xpanel/global"
	"xpanel/utils/cmd"
	haproxyutil "xpanel/utils/haproxy"
)

const (
	haproxyBinary        = "haproxy"
	haproxyConfigPath    = "/etc/haproxy/haproxy.cfg"
	haproxyServiceName   = "haproxy"
	haproxyRsyslogConf   = "/etc/rsyslog.d/49-xpanel-haproxy.conf"
	haproxyCombinedPEMDir = "/opt/xpanel/haproxy/certs"
	haproxyBackupDir     = "/opt/xpanel/haproxy/backups"
)

type IHAProxyInstallService interface {
	GetStatus() (*dto.HAProxyStatus, error)
	Install(req dto.HAProxyInstallReq) error
	GetProgress() *dto.HAProxyInstallProgress
	Uninstall() error
	Operate(req dto.HAProxyOperateReq) error
	CheckUpdate() (*dto.HAProxyCheckUpdateResp, error)
	Upgrade(req dto.HAProxyUpgradeReq) error
}

type HAProxyInstallService struct {
	mu       sync.Mutex
	progress *dto.HAProxyInstallProgress
}

var haproxyInstallSingleton = &HAProxyInstallService{}

func NewIHAProxyInstallService() IHAProxyInstallService {
	return haproxyInstallSingleton
}

func (s *HAProxyInstallService) GetStatus() (*dto.HAProxyStatus, error) {
	status := &dto.HAProxyStatus{
		ConfigPath: haproxyConfigPath,
		SocketPath: haproxyutil.DefaultSocketPath,
	}
	if !isHAProxyInstalled() {
		applyHAProxyStatsSettings(status)
		return status, nil
	}
	status.IsInstalled = true
	if out, err := cmd.ExecWithOutput(haproxyBinary, "-v"); err == nil {
		status.Version = haproxyutil.ParseVersion(out)
	}
	out, _ := cmd.ExecWithOutput("systemctl", "is-active", haproxyServiceName)
	status.IsRunning = strings.TrimSpace(out) == "active"

	// socket 可达
	sock := haproxyutil.NewSocket(haproxyutil.DefaultSocketPath)
	status.SocketReady = status.IsRunning && sock.Ping()

	// 开机自启
	if o, err := cmd.ExecWithOutput("systemctl", "is-enabled", haproxyServiceName); err == nil {
		status.AutoStart = strings.TrimSpace(o) == "enabled"
	}

	applyHAProxyStatsSettings(status)
	return status, nil
}

func applyHAProxyStatsSettings(status *dto.HAProxyStatus) {
	s := repo.NewISettingRepo()
	getv := func(key, def string) string {
		v, err := s.Get(repo.WithByKey(key))
		if err != nil || v.Value == "" {
			return def
		}
		return v.Value
	}
	status.StatsEnable = getv("HAProxyStatsEnable", "enable") == "enable"
	status.StatsBind = getv("HAProxyStatsBind", "127.0.0.1:9999")
	status.StatsURI = getv("HAProxyStatsURI", "/stats")
	status.StatsUser = getv("HAProxyStatsUser", "")
}

func (s *HAProxyInstallService) Install(req dto.HAProxyInstallReq) error {
	if isHAProxyInstalled() {
		return buserr.New(constant.ErrHAProxyAlreadyInstalled)
	}
	go s.doInstall()
	return nil
}

func (s *HAProxyInstallService) GetProgress() *dto.HAProxyInstallProgress {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.progress == nil {
		return &dto.HAProxyInstallProgress{Phase: "idle", Message: "未在安装", Percent: 0}
	}
	cp := *s.progress
	return &cp
}

func (s *HAProxyInstallService) setProgress(phase, message string, percent int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.progress = &dto.HAProxyInstallProgress{Phase: phase, Message: message, Percent: percent}
	global.LOG.Infof("[haproxy-install] [%s] %s (%d%%)", phase, message, percent)
}

func (s *HAProxyInstallService) doInstall() {
	s.setProgress("download", "正在更新软件包索引...", 5)
	if out, err := exec.Command("apt-get", "update", "-qq").CombinedOutput(); err != nil {
		s.setProgress("error", fmt.Sprintf("apt-get update 失败: %s", strings.TrimSpace(string(out))), 0)
		return
	}
	s.setProgress("download", "软件包索引已更新", 20)

	s.setProgress("install", "正在安装 HAProxy 和 socat...", 30)
	env := append(os.Environ(), "DEBIAN_FRONTEND=noninteractive")
	installCmd := exec.Command("apt-get", "install", "-y", "haproxy", "socat")
	installCmd.Env = env
	if out, err := installCmd.CombinedOutput(); err != nil {
		s.setProgress("error", fmt.Sprintf("安装失败: %s", strings.TrimSpace(string(out))), 0)
		return
	}
	s.setProgress("install", "HAProxy 已安装", 55)

	s.setProgress("install", "初始化面板专属目录...", 60)
	_ = os.MkdirAll(haproxyCombinedPEMDir, 0750)
	_ = os.MkdirAll(haproxyBackupDir, 0750)

	s.setProgress("install", "写入默认配置...", 70)
	if err := s.seedInitialSettings(); err != nil {
		s.setProgress("error", fmt.Sprintf("初始化设置失败: %v", err), 0)
		return
	}
	if err := writeInitialHAProxyConfig(); err != nil {
		s.setProgress("error", fmt.Sprintf("写入配置失败: %v", err), 0)
		return
	}
	if out, err := exec.Command(haproxyBinary, "-c", "-f", haproxyConfigPath).CombinedOutput(); err != nil {
		s.setProgress("error", fmt.Sprintf("配置校验失败: %s", strings.TrimSpace(string(out))), 0)
		return
	}

	s.setProgress("install", "配置 rsyslog...", 80)
	_ = writeHAProxyRsyslog()
	_, _ = cmd.ExecWithOutput("systemctl", "restart", "rsyslog")

	s.setProgress("install", "启动 HAProxy 服务...", 90)
	if _, err := cmd.ExecWithOutput("systemctl", "enable", haproxyServiceName); err != nil {
		global.LOG.Warnf("enable haproxy failed: %v", err)
	}
	if _, err := cmd.ExecWithOutput("systemctl", "restart", haproxyServiceName); err != nil {
		s.setProgress("error", fmt.Sprintf("启动失败: %v", err), 0)
		return
	}
	time.Sleep(1 * time.Second)

	ver := ""
	if out, err := cmd.ExecWithOutput(haproxyBinary, "-v"); err == nil {
		ver = haproxyutil.ParseVersion(out)
	}
	s.setProgress("done", fmt.Sprintf("HAProxy %s 安装成功", ver), 100)
	global.LOG.Infof("HAProxy installed via apt, version: %s", ver)
}

func (s *HAProxyInstallService) Uninstall() error {
	if !isHAProxyInstalled() {
		return buserr.New(constant.ErrHAProxyNotInstalled)
	}
	_, _ = cmd.ExecWithOutput("systemctl", "stop", haproxyServiceName)
	_, _ = cmd.ExecWithOutput("systemctl", "disable", haproxyServiceName)

	out, err := exec.Command("apt-get", "remove", "--purge", "-y", "haproxy").CombinedOutput()
	if err != nil {
		return buserr.WithDetail(constant.ErrInternalServer,
			fmt.Sprintf("apt-get remove failed: %s", strings.TrimSpace(string(out))), err)
	}
	_ = exec.Command("apt-get", "autoremove", "-y").Run()

	_ = os.Remove(haproxyRsyslogConf)
	_, _ = cmd.ExecWithOutput("systemctl", "restart", "rsyslog")

	// 清理面板端设置（保留 DB 中业务数据，避免重新安装时丢失配置）
	settingRepo := repo.NewISettingRepo()
	for _, k := range []string{"HAProxyStatsPass"} {
		_ = settingRepo.Delete(repo.WithByKey(k))
	}
	global.LOG.Info("HAProxy uninstalled via apt")
	return nil
}

func (s *HAProxyInstallService) Operate(req dto.HAProxyOperateReq) error {
	op := req.Operation
	if op != "start" && op != "stop" && op != "restart" && op != "reload" {
		return fmt.Errorf("unsupported operation: %s", op)
	}
	out, err := cmd.ExecWithOutput("systemctl", op, haproxyServiceName)
	if err != nil {
		return buserr.WithDetail(constant.ErrInternalServer, strings.TrimSpace(out), err)
	}
	return nil
}

func (s *HAProxyInstallService) CheckUpdate() (*dto.HAProxyCheckUpdateResp, error) {
	if !isHAProxyInstalled() {
		return nil, buserr.New(constant.ErrHAProxyNotInstalled)
	}
	cur := ""
	if out, err := cmd.ExecWithOutput(haproxyBinary, "-v"); err == nil {
		cur = haproxyutil.ParseVersion(out)
	}
	_, _ = cmd.ExecWithOutput("apt-get", "update", "-qq")
	out, err := cmd.ExecWithOutput("apt-cache", "policy", "haproxy")
	if err != nil {
		return &dto.HAProxyCheckUpdateResp{CurrentVersion: cur}, nil
	}
	available := ""
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Candidate:") {
			v := strings.TrimSpace(strings.TrimPrefix(line, "Candidate:"))
			available = v
			break
		}
	}
	// apt 版本格式如 "2.4.22-0ubuntu0.22.04.3"，简化成主版本比较
	avShort := shortenDebianVersion(available)
	curShort := shortenDebianVersion(cur)
	hasUpdate := avShort != "" && curShort != "" && avShort != curShort
	return &dto.HAProxyCheckUpdateResp{
		CurrentVersion:   cur,
		AvailableVersion: available,
		HasUpdate:        hasUpdate,
	}, nil
}

func (s *HAProxyInstallService) Upgrade(req dto.HAProxyUpgradeReq) error {
	if !isHAProxyInstalled() {
		return buserr.New(constant.ErrHAProxyNotInstalled)
	}
	go s.doUpgrade()
	return nil
}

func (s *HAProxyInstallService) doUpgrade() {
	s.setProgress("download", "正在更新软件包索引...", 5)
	_, _ = cmd.ExecWithOutput("apt-get", "update", "-qq")
	s.setProgress("install", "正在升级 HAProxy...", 40)
	out, err := exec.Command("apt-get", "install", "--only-upgrade", "-y", "haproxy").CombinedOutput()
	if err != nil {
		s.setProgress("error", fmt.Sprintf("升级失败: %s", strings.TrimSpace(string(out))), 0)
		return
	}
	s.setProgress("install", "重载 HAProxy...", 85)
	_, _ = cmd.ExecWithOutput("systemctl", "reload", haproxyServiceName)
	ver := ""
	if out, err := cmd.ExecWithOutput(haproxyBinary, "-v"); err == nil {
		ver = haproxyutil.ParseVersion(out)
	}
	s.setProgress("done", fmt.Sprintf("HAProxy 已升级到 %s", ver), 100)
}

// --- 辅助 ---

func isHAProxyInstalled() bool {
	if _, err := exec.LookPath(haproxyBinary); err == nil {
		return true
	}
	if _, err := os.Stat("/usr/sbin/haproxy"); err == nil {
		return true
	}
	return false
}

func writeInitialHAProxyConfig() error {
	// 如果已存在面板管理的配置，则不覆盖
	if data, err := os.ReadFile(haproxyConfigPath); err == nil {
		if strings.Contains(string(data), "Generated by X-Panel") {
			return nil
		}
		// 备份原始配置
		_ = os.MkdirAll(haproxyBackupDir, 0750)
		_ = os.WriteFile(
			fmt.Sprintf("%s/haproxy.cfg.pre-xpanel.%d", haproxyBackupDir, time.Now().Unix()),
			data, 0640,
		)
	}

	user, pass := getHAProxyStatsAuth()
	cfg := haproxyutil.Build(haproxyutil.BuilderInput{
		Settings: haproxyutil.Settings{
			GlobalLog:   "127.0.0.1 local0",
			SocketPath:  haproxyutil.DefaultSocketPath,
			StatsEnable: true,
			StatsBind:   "127.0.0.1:9999",
			StatsURI:    "/stats",
			StatsUser:   user,
			StatsPass:   pass,
			MaxConn:     50000,
		},
	})
	return os.WriteFile(haproxyConfigPath, []byte(cfg), 0640)
}

func writeHAProxyRsyslog() error {
	content := `# Generated by X-Panel
if $programname == 'haproxy' then /var/log/haproxy.log
& stop
`
	return os.WriteFile(haproxyRsyslogConf, []byte(content), 0644)
}

func (s *HAProxyInstallService) seedInitialSettings() error {
	sr := repo.NewISettingRepo()
	ensure := func(key, def string) {
		if v, err := sr.Get(repo.WithByKey(key)); err != nil || v.Value == "" {
			_ = sr.CreateOrUpdate(key, def)
		}
	}
	ensure("HAProxyStatsEnable", "enable")
	ensure("HAProxyStatsBind", "127.0.0.1:9999")
	ensure("HAProxyStatsURI", "/stats")
	ensure("HAProxyStatsUser", "xpanel")
	if v, err := sr.Get(repo.WithByKey("HAProxyStatsPass")); err != nil || v.Value == "" {
		_ = sr.CreateOrUpdate("HAProxyStatsPass", randHex(12))
	}
	return nil
}

func getHAProxyStatsAuth() (string, string) {
	sr := repo.NewISettingRepo()
	u, _ := sr.Get(repo.WithByKey("HAProxyStatsUser"))
	p, _ := sr.Get(repo.WithByKey("HAProxyStatsPass"))
	if u.Value == "" {
		u.Value = "xpanel"
	}
	if p.Value == "" {
		p.Value = randHex(12)
		_ = sr.CreateOrUpdate("HAProxyStatsPass", p.Value)
	}
	return u.Value, p.Value
}

func randHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)[:n]
}

func shortenDebianVersion(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	if idx := strings.Index(v, "-"); idx > 0 {
		v = v[:idx]
	}
	return v
}
