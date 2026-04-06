package service

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"xpanel/app/dto"
	"xpanel/app/repo"
	"xpanel/global"
	"xpanel/utils/cmd"

	gostutil "xpanel/utils/gost"
)

const (
	gostInstallDir  = "/opt/xpanel/gost"
	gostBinaryPath  = "/opt/xpanel/gost/gost"
	gostConfigPath  = "/opt/xpanel/gost/gost.yaml"
	gostServiceName = "xpanel-gost"
	gostDefaultAPI  = "127.0.0.1:18080"
	gostGitHubRepo  = "go-gost/gost"
)

type IGostInstallService interface {
	GetStatus() (*dto.GostStatus, error)
	Install(req dto.GostInstallReq) error
	GetProgress() *dto.GostInstallProgress
	Uninstall() error
	Operate(req dto.GostOperateReq) error
	CheckUpdate() (*dto.GostCheckUpdateResp, error)
	Upgrade(req dto.GostUpgradeReq) error
}

type GostInstallService struct {
	mu       sync.Mutex
	progress *dto.GostInstallProgress
}

var gostInstallSingleton = &GostInstallService{}

func NewIGostInstallService() IGostInstallService {
	return gostInstallSingleton
}

func (s *GostInstallService) GetStatus() (*dto.GostStatus, error) {
	status := &dto.GostStatus{}

	if _, err := os.Stat(gostBinaryPath); err != nil {
		return status, nil
	}
	status.IsInstalled = true

	out, err := exec.Command(gostBinaryPath, "-V").CombinedOutput()
	if err == nil {
		ver := strings.TrimSpace(string(out))
		if idx := strings.Index(ver, "gost"); idx >= 0 {
			parts := strings.Fields(ver[idx:])
			if len(parts) >= 2 {
				status.Version = parts[1]
			}
		}
		if status.Version == "" {
			status.Version = ver
		}
	}

	output, err := cmd.ExecWithOutput("systemctl", "is-active", gostServiceName)
	status.IsRunning = err == nil && strings.TrimSpace(output) == "active"

	apiAddr := getGostAPISetting()
	user, pass := getGostAPIAuth()
	client := gostutil.NewClient(apiAddr, user, pass)
	status.APIReady = client.Ping()

	return status, nil
}

func (s *GostInstallService) Install(req dto.GostInstallReq) error {
	if _, err := os.Stat(gostBinaryPath); err == nil {
		return fmt.Errorf("GOST is already installed")
	}
	go s.doInstall(req.Version)
	return nil
}

func (s *GostInstallService) GetProgress() *dto.GostInstallProgress {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.progress == nil {
		return &dto.GostInstallProgress{Phase: "idle", Message: "未在安装", Percent: 0}
	}
	cp := *s.progress
	return &cp
}

func (s *GostInstallService) Uninstall() error {
	if _, err := os.Stat(gostBinaryPath); err != nil {
		return fmt.Errorf("GOST is not installed")
	}

	cmd.ExecWithOutput("systemctl", "stop", gostServiceName)
	cmd.ExecWithOutput("systemctl", "disable", gostServiceName)
	os.Remove("/etc/systemd/system/" + gostServiceName + ".service")
	cmd.ExecWithOutput("systemctl", "daemon-reload")

	if err := os.RemoveAll(gostInstallDir); err != nil {
		return fmt.Errorf("failed to remove %s: %v", gostInstallDir, err)
	}

	settingRepo := repo.NewISettingRepo()
	settingRepo.Delete(repo.WithByKey("GostAPIAddr"))
	settingRepo.Delete(repo.WithByKey("GostAPIUser"))
	settingRepo.Delete(repo.WithByKey("GostAPIPass"))

	global.LOG.Info("GOST uninstalled")
	return nil
}

func (s *GostInstallService) Operate(req dto.GostOperateReq) error {
	switch req.Operation {
	case "start":
		_, err := cmd.ExecWithOutput("systemctl", "start", gostServiceName)
		if err != nil {
			return fmt.Errorf("start failed: %v", err)
		}
		time.Sleep(1 * time.Second)
		go syncAllToGost()
	case "stop":
		_, err := cmd.ExecWithOutput("systemctl", "stop", gostServiceName)
		return err
	case "restart":
		_, err := cmd.ExecWithOutput("systemctl", "restart", gostServiceName)
		if err != nil {
			return fmt.Errorf("restart failed: %v", err)
		}
		time.Sleep(1 * time.Second)
		go syncAllToGost()
	default:
		return fmt.Errorf("unsupported operation: %s", req.Operation)
	}
	return nil
}

func (s *GostInstallService) CheckUpdate() (*dto.GostCheckUpdateResp, error) {
	if _, err := os.Stat(gostBinaryPath); err != nil {
		return nil, fmt.Errorf("GOST is not installed")
	}

	currentVersion := ""
	out, err := exec.Command(gostBinaryPath, "-V").CombinedOutput()
	if err == nil {
		ver := strings.TrimSpace(string(out))
		if idx := strings.Index(ver, "gost"); idx >= 0 {
			parts := strings.Fields(ver[idx:])
			if len(parts) >= 2 {
				currentVersion = parts[1]
			}
		}
	}

	latestVersion, err := s.fetchLatestVersion()
	if err != nil {
		return nil, fmt.Errorf("failed to check latest version: %v", err)
	}

	hasUpdate := false
	if currentVersion != "" && latestVersion != "" {
		hasUpdate = normalizeVersion(latestVersion) != normalizeVersion(currentVersion)
	}

	return &dto.GostCheckUpdateResp{
		CurrentVersion: currentVersion,
		LatestVersion:  latestVersion,
		HasUpdate:      hasUpdate,
		ReleaseURL:     fmt.Sprintf("https://github.com/%s/releases/tag/%s", gostGitHubRepo, latestVersion),
	}, nil
}

func (s *GostInstallService) Upgrade(req dto.GostUpgradeReq) error {
	if _, err := os.Stat(gostBinaryPath); err != nil {
		return fmt.Errorf("GOST is not installed")
	}
	go s.doUpgrade(req.Version)
	return nil
}

func (s *GostInstallService) doUpgrade(version string) {
	s.setProgress("download", fmt.Sprintf("正在下载 GOST %s ...", version), 10)

	arch := runtime.GOARCH
	goos := runtime.GOOS
	assetName := fmt.Sprintf("gost_%s_%s_%s.tar.gz", strings.TrimPrefix(version, "v"), goos, arch)
	downloadURL := fmt.Sprintf("https://github.com/%s/releases/download/%s/%s", gostGitHubRepo, version, assetName)

	tmpDir, err := os.MkdirTemp("", "gost-upgrade-*")
	if err != nil {
		s.setProgress("error", fmt.Sprintf("创建临时目录失败: %v", err), 0)
		return
	}
	defer os.RemoveAll(tmpDir)

	tarballPath := filepath.Join(tmpDir, assetName)
	if err := gostDownloadFile(downloadURL, tarballPath); err != nil {
		s.setProgress("error", fmt.Sprintf("下载失败: %v", err), 0)
		return
	}
	s.setProgress("download", "下载完成", 40)

	s.setProgress("install", "正在解压...", 50)
	extractCmd := exec.Command("tar", "-xzf", tarballPath, "-C", tmpDir)
	if out, err := extractCmd.CombinedOutput(); err != nil {
		s.setProgress("error", fmt.Sprintf("解压失败: %s", strings.TrimSpace(string(out))), 0)
		return
	}

	extractedBin := filepath.Join(tmpDir, "gost")
	if _, err := os.Stat(extractedBin); err != nil {
		s.setProgress("error", "解压后未找到 gost 二进制文件", 0)
		return
	}

	s.setProgress("install", "停止 GOST 服务...", 60)
	cmd.ExecWithOutput("systemctl", "stop", gostServiceName)

	backupPath := gostBinaryPath + ".bak"
	if err := copyFileSimple(gostBinaryPath, backupPath); err != nil {
		global.LOG.Warnf("Failed to backup GOST binary: %v", err)
	}

	s.setProgress("install", "替换二进制文件...", 70)
	if err := copyFileSimple(extractedBin, gostBinaryPath); err != nil {
		s.setProgress("error", fmt.Sprintf("替换失败: %v (正在回滚)", err), 0)
		copyFileSimple(backupPath, gostBinaryPath)
		cmd.ExecWithOutput("systemctl", "start", gostServiceName)
		return
	}
	os.Chmod(gostBinaryPath, 0755)
	os.Remove(backupPath)

	s.setProgress("install", "启动 GOST 服务...", 85)
	cmd.ExecWithOutput("systemctl", "start", gostServiceName)
	time.Sleep(2 * time.Second)
	go syncAllToGost()

	s.setProgress("done", fmt.Sprintf("GOST 已升级到 %s", version), 100)
	global.LOG.Infof("GOST upgraded to %s", version)
}

func normalizeVersion(v string) string {
	return strings.TrimPrefix(strings.TrimSpace(v), "v")
}

func (s *GostInstallService) setProgress(phase, message string, percent int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.progress = &dto.GostInstallProgress{
		Phase:   phase,
		Message: message,
		Percent: percent,
	}
	global.LOG.Infof("[gost-install] [%s] %s (%d%%)", phase, message, percent)
}

func (s *GostInstallService) doInstall(version string) {
	s.setProgress("download", "正在获取 GOST 版本信息...", 5)

	if version == "" {
		ver, err := s.fetchLatestVersion()
		if err != nil {
			s.setProgress("error", fmt.Sprintf("获取最新版本失败: %v", err), 0)
			return
		}
		version = ver
	}

	arch := runtime.GOARCH
	goos := runtime.GOOS
	assetName := fmt.Sprintf("gost_%s_%s_%s.tar.gz", strings.TrimPrefix(version, "v"), goos, arch)
	downloadURL := fmt.Sprintf("https://github.com/%s/releases/download/%s/%s", gostGitHubRepo, version, assetName)

	s.setProgress("download", fmt.Sprintf("正在下载 GOST %s (%s/%s)...", version, goos, arch), 15)

	tmpDir, err := os.MkdirTemp("", "gost-install-*")
	if err != nil {
		s.setProgress("error", fmt.Sprintf("创建临时目录失败: %v", err), 0)
		return
	}
	defer os.RemoveAll(tmpDir)

	tarballPath := filepath.Join(tmpDir, assetName)
	if err := gostDownloadFile(downloadURL, tarballPath); err != nil {
		s.setProgress("error", fmt.Sprintf("下载失败: %v", err), 0)
		return
	}
	s.setProgress("download", "下载完成", 40)

	s.setProgress("install", "正在解压安装...", 50)
	os.MkdirAll(gostInstallDir, 0755)

	extractCmd := exec.Command("tar", "-xzf", tarballPath, "-C", tmpDir)
	if out, err := extractCmd.CombinedOutput(); err != nil {
		s.setProgress("error", fmt.Sprintf("解压失败: %s", strings.TrimSpace(string(out))), 0)
		return
	}

	extractedBin := filepath.Join(tmpDir, "gost")
	if _, err := os.Stat(extractedBin); err != nil {
		s.setProgress("error", "解压后未找到 gost 二进制文件", 0)
		return
	}

	if err := copyFileSimple(extractedBin, gostBinaryPath); err != nil {
		s.setProgress("error", fmt.Sprintf("安装二进制文件失败: %v", err), 0)
		return
	}
	os.Chmod(gostBinaryPath, 0755)
	s.setProgress("install", "二进制文件已安装", 65)

	s.setProgress("install", "生成配置文件...", 70)
	apiUser := "xpanel"
	apiPass := generateRandomPassword(16)
	if err := s.writeInitialConfig(apiUser, apiPass); err != nil {
		s.setProgress("error", fmt.Sprintf("写入配置文件失败: %v", err), 0)
		return
	}

	settingRepo := repo.NewISettingRepo()
	settingRepo.CreateOrUpdate("GostAPIAddr", gostDefaultAPI)
	settingRepo.CreateOrUpdate("GostAPIUser", apiUser)
	settingRepo.CreateOrUpdate("GostAPIPass", apiPass)

	s.setProgress("install", "创建 systemd 服务...", 80)
	if err := s.writeServiceFile(); err != nil {
		s.setProgress("error", fmt.Sprintf("创建 systemd 服务失败: %v", err), 0)
		return
	}

	cmd.ExecWithOutput("systemctl", "daemon-reload")
	cmd.ExecWithOutput("systemctl", "enable", gostServiceName)
	cmd.ExecWithOutput("systemctl", "start", gostServiceName)
	s.setProgress("install", "GOST 服务已启动", 95)

	time.Sleep(2 * time.Second)
	go syncAllToGost()

	s.setProgress("done", fmt.Sprintf("GOST %s 安装成功", version), 100)
	global.LOG.Infof("GOST %s installed at %s", version, gostInstallDir)
}

func (s *GostInstallService) fetchLatestVersion() (string, error) {
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", gostGitHubRepo)
	resp, err := http.Get(apiURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}
	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}
	return release.TagName, nil
}

func (s *GostInstallService) writeInitialConfig(user, pass string) error {
	config := fmt.Sprintf(`api:
  addr: %s
  auth:
    username: %s
    password: %s
`, gostDefaultAPI, user, pass)
	return os.WriteFile(gostConfigPath, []byte(config), 0600)
}

func (s *GostInstallService) writeServiceFile() error {
	content := fmt.Sprintf(`[Unit]
Description=GOST Tunnel Service (managed by X-Panel)
After=network.target

[Service]
Type=simple
ExecStart=%s -C %s
Restart=on-failure
RestartSec=5
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
`, gostBinaryPath, gostConfigPath)
	return os.WriteFile("/etc/systemd/system/"+gostServiceName+".service", []byte(content), 0644)
}

// --- helpers ---

func getGostAPISetting() string {
	settingRepo := repo.NewISettingRepo()
	s, err := settingRepo.Get(repo.WithByKey("GostAPIAddr"))
	if err != nil || s.Value == "" {
		return gostDefaultAPI
	}
	return s.Value
}

func getGostAPIAuth() (string, string) {
	settingRepo := repo.NewISettingRepo()
	u, _ := settingRepo.Get(repo.WithByKey("GostAPIUser"))
	p, _ := settingRepo.Get(repo.WithByKey("GostAPIPass"))
	return u.Value, p.Value
}

func newGostClient() *gostutil.Client {
	user, pass := getGostAPIAuth()
	return gostutil.NewClient(getGostAPISetting(), user, pass)
}

func generateRandomPassword(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)[:n]
}

func gostDownloadFile(url, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}

func copyFileSimple(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

// syncAllToGost pushes all enabled DB rules to GOST via API (called after start/restart).
func syncAllToGost() {
	time.Sleep(1 * time.Second)
	svc := NewIGostService()
	if err := svc.SyncAll(); err != nil {
		global.LOG.Warnf("Failed to sync GOST config: %v", err)
	}
}
