package service

import (
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
	"xpanel/app/model"
	"xpanel/app/repo"
	"xpanel/buserr"
	"xpanel/constant"
	"xpanel/global"
	"xpanel/utils/cmd"
)

type INginxInstallService interface {
	Install(req dto.NginxInstallReq) error
	GetProgress() *dto.NginxInstallProgress
	Uninstall(req dto.NginxUninstallReq) error
	ListVersions() ([]dto.NginxVersionInfo, error)
	CheckUpdate() (*dto.NginxUpdateInfo, error)
	Upgrade(req dto.NginxUpgradeReq) error
}

type NginxInstallService struct {
	mu          sync.Mutex
	progress    *dto.NginxInstallProgress
	websiteRepo repo.IWebsiteRepo
}

func NewINginxInstallService() INginxInstallService {
	return &NginxInstallService{
		websiteRepo: repo.NewIWebsiteRepo(),
	}
}

// Install 安装 Nginx，根据 Method 选择安装方式
func (s *NginxInstallService) Install(req dto.NginxInstallReq) error {
	if global.CONF.Nginx.IsInstalled() {
		return buserr.New(constant.ErrNginxAlreadyInstalled)
	}

	method := strings.ToLower(req.Method)
	if method == "" {
		method = "apt"
	}

	switch method {
	case "apt":
		go s.doInstallApt()
	case "precompiled":
		if req.Version == "" {
			return fmt.Errorf("version is required for precompiled install")
		}
		installDir := global.CONF.Nginx.InstallDir
		go s.doInstall(req.Version, installDir)
	default:
		return fmt.Errorf("unsupported install method: %s", method)
	}
	return nil
}

// GetProgress 返回当前安装进度
func (s *NginxInstallService) GetProgress() *dto.NginxInstallProgress {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.progress == nil {
		return &dto.NginxInstallProgress{Phase: "idle", Message: "未在安装", Percent: 0}
	}
	cp := *s.progress
	return &cp
}

// doInstallApt 通过 apt 安装 Nginx（在 goroutine 中运行）
func (s *NginxInstallService) doInstallApt() {
	s.setProgress("download", "正在更新软件包索引...", 5)

	// apt-get update
	updateCmd := exec.Command("apt-get", "update", "-qq")
	if output, err := updateCmd.CombinedOutput(); err != nil {
		s.setProgress("error", fmt.Sprintf("apt-get update 失败: %s", strings.TrimSpace(string(output))), 0)
		return
	}
	s.setProgress("download", "软件包索引已更新", 20)

	// apt-get install -y nginx
	s.setProgress("install", "正在安装 Nginx...", 30)
	installCmd := exec.Command("apt-get", "install", "-y", "nginx")
	output, err := installCmd.CombinedOutput()
	if err != nil {
		s.setProgress("error", fmt.Sprintf("安装失败: %s", strings.TrimSpace(string(output))), 0)
		return
	}
	s.setProgress("install", "Nginx 已安装", 70)

	// 启用并启动服务
	s.setProgress("install", "正在启动 Nginx 服务...", 80)
	if _, err := cmd.ExecWithOutput("systemctl", "enable", "nginx"); err != nil {
		global.LOG.Warnf("Enable nginx autostart failed: %v", err)
	}
	if _, err := cmd.ExecWithOutput("systemctl", "start", "nginx"); err != nil {
		global.LOG.Warnf("Start nginx failed: %v", err)
	}
	s.setProgress("install", "Nginx 服务已启动", 90)

	// 重新检测 nginx 配置
	global.CONF.Nginx.DetectNginx()

	ver := global.CONF.Nginx.GetVersion()
	s.setProgress("done", fmt.Sprintf("Nginx %s 安装成功（apt）", ver), 100)
	global.LOG.Infof("Nginx installed via apt, version: %s", ver)
}

// Uninstall 卸载 Nginx
// Mode 可选: "system" 仅卸载 apt 安装, "prefix" 仅卸载自包含安装, 空则卸载当前活跃模式
func (s *NginxInstallService) Uninstall(req dto.NginxUninstallReq) error {
	if !global.CONF.Nginx.IsInstalled() {
		return buserr.New(constant.ErrNginxNotInstalled)
	}

	nc := global.CONF.Nginx
	uninstallSystem := false
	uninstallPrefix := false

	switch strings.ToLower(req.Mode) {
	case "system":
		if !nc.HasSystemInstalled() {
			return buserr.WithDetail(constant.ErrInternalServer, "system nginx not found", nil)
		}
		uninstallSystem = true
	case "prefix":
		if !nc.HasPrefixInstalled() {
			return buserr.WithDetail(constant.ErrInternalServer, "prefix nginx not found", nil)
		}
		uninstallPrefix = true
	default:
		if nc.IsSystemMode() {
			uninstallSystem = true
		} else {
			uninstallPrefix = true
		}
	}

	isActiveMode := (uninstallSystem && nc.IsSystemMode()) || (uninstallPrefix && !nc.IsSystemMode())

	if isActiveMode {
		siteCount, _ := s.websiteRepo.Count()
		if siteCount > 0 && !req.ForceCleanup {
			return buserr.WithDetail(constant.ErrNginxHasSites,
				fmt.Sprintf("%d websites exist", siteCount), nil)
		}
		if siteCount > 0 && req.ForceCleanup {
			s.cleanupAllSites()
		}
	}

	if uninstallSystem {
		if err := s.uninstallSystemNginx(); err != nil {
			return err
		}
	}
	if uninstallPrefix {
		if err := s.uninstallPrefixNginx(); err != nil {
			return err
		}
	}

	global.CONF.Nginx.DetectNginx()
	return nil
}

func (s *NginxInstallService) uninstallSystemNginx() error {
	if _, err := cmd.ExecWithOutput("systemctl", "stop", "nginx"); err != nil {
		global.LOG.Warnf("Stop system nginx failed: %v", err)
	}
	output, err := exec.Command("apt-get", "remove", "--purge", "-y", "nginx", "nginx-common", "nginx-core").CombinedOutput()
	if err != nil {
		return buserr.WithDetail(constant.ErrInternalServer,
			fmt.Sprintf("apt-get remove failed: %s", strings.TrimSpace(string(output))), err)
	}
	_ = exec.Command("apt-get", "autoremove", "-y").Run()
	global.LOG.Infof("Nginx uninstalled via apt (purge)")
	return nil
}

func (s *NginxInstallService) uninstallPrefixNginx() error {
	installDir := global.CONF.Nginx.InstallDir
	nginxBin := filepath.Join(installDir, "sbin", "nginx")
	pidPath := filepath.Join(installDir, "logs", "nginx.pid")

	if _, err := os.Stat(pidPath); err == nil {
		_ = exec.Command(nginxBin, "-p", installDir, "-s", "quit").Run()
		time.Sleep(2 * time.Second)
	}
	// 清理 systemd 服务
	svcFile := "/etc/systemd/system/xpanel-nginx.service"
	if _, err := os.Stat(svcFile); err == nil {
		_ = exec.Command("systemctl", "stop", "xpanel-nginx").Run()
		_ = exec.Command("systemctl", "disable", "xpanel-nginx").Run()
		os.Remove(svcFile)
		_ = exec.Command("systemctl", "daemon-reload").Run()
	}

	if err := os.RemoveAll(installDir); err != nil {
		return buserr.WithDetail(constant.ErrInternalServer,
			fmt.Sprintf("failed to remove %s: %v", installDir, err), err)
	}
	global.LOG.Infof("Nginx uninstalled from %s (clean)", installDir)
	return nil
}

// cleanupAllSites 清理所有网站的 nginx 配置文件和数据库记录
func (s *NginxInstallService) cleanupAllSites() {
	sites, err := s.websiteRepo.GetList()
	if err != nil {
		global.LOG.Warnf("Failed to list sites for cleanup: %v", err)
		return
	}

	nc := global.CONF.Nginx
	for _, site := range sites {
		if nc.IsSystemMode() {
			os.Remove(filepath.Join(nc.GetSitesDir(), site.Alias+".conf"))
			os.Remove(filepath.Join(nc.GetSitesAvailableDir(), site.Alias+".conf"))
		} else {
			os.Remove(filepath.Join(nc.GetConfDir(), "conf.d", site.Alias+".conf"))
		}
		authFile := filepath.Join(nc.GetConfDir(), "auth", site.Alias+".htpasswd")
		os.Remove(authFile)

		global.LOG.Infof("Cleaned up site config: %s", site.Alias)
	}

	// 清空网站数据库记录
	if err := global.DB.Where("1 = 1").Delete(&model.Website{}).Error; err != nil {
		global.LOG.Warnf("Failed to cleanup website records: %v", err)
	} else {
		global.LOG.Infof("Cleaned up %d website records from database", len(sites))
	}
}

// ListVersions 从 GitHub Release 获取可用的 Nginx 预编译版本列表
func (s *NginxInstallService) ListVersions() ([]dto.NginxVersionInfo, error) {
	buildRepo := global.CONF.Nginx.BuildRepo
	if buildRepo == "" {
		buildRepo = "Anikato/nginx-build"
	}

	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/releases?per_page=20", buildRepo)
	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch versions: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	var releases []struct {
		TagName     string `json:"tag_name"`
		Name        string `json:"name"`
		PublishedAt string `json:"published_at"`
		Prerelease  bool   `json:"prerelease"`
		Assets      []struct {
			Name               string `json:"name"`
			Size               int64  `json:"size"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, fmt.Errorf("failed to parse releases: %v", err)
	}

	arch := runtime.GOARCH
	var versions []dto.NginxVersionInfo
	for _, r := range releases {
		if r.Prerelease {
			continue
		}
		// 检查是否有当前架构的资源
		hasArch := false
		for _, a := range r.Assets {
			if strings.Contains(a.Name, arch) && strings.HasSuffix(a.Name, ".tar.gz") &&
				!strings.HasSuffix(a.Name, ".sha256") {
				hasArch = true
				break
			}
		}
		if !hasArch {
			continue
		}

		version := strings.TrimPrefix(r.TagName, "v")
		versions = append(versions, dto.NginxVersionInfo{
			Version:     version,
			Tag:         r.TagName,
			PublishedAt: r.PublishedAt,
		})
	}

	return versions, nil
}

// CheckUpdate 检查 Nginx 是否有可用更新
func (s *NginxInstallService) CheckUpdate() (*dto.NginxUpdateInfo, error) {
	nc := global.CONF.Nginx
	if !nc.IsInstalled() {
		return nil, buserr.New(constant.ErrNginxNotInstalled)
	}

	info := &dto.NginxUpdateInfo{
		CurrentVersion: nc.GetVersion(),
		SystemMode:     nc.IsSystemMode(),
	}

	if nc.IsSystemMode() {
		available := s.checkAptUpdate()
		if available != "" && available != info.CurrentVersion {
			info.HasUpdate = true
			info.AvailableVersion = available
		}
	} else {
		versions, err := s.ListVersions()
		if err == nil && len(versions) > 0 {
			latest := versions[0].Version
			if latest != info.CurrentVersion {
				info.HasUpdate = true
				info.AvailableVersion = latest
			}
		}
	}

	return info, nil
}

// checkAptUpdate 检查 apt 仓库中 nginx 的可用版本
func (s *NginxInstallService) checkAptUpdate() string {
	exec.Command("apt-get", "update", "-qq").Run()
	output, err := exec.Command("apt-cache", "policy", "nginx").CombinedOutput()
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Candidate:") {
			ver := strings.TrimSpace(strings.TrimPrefix(line, "Candidate:"))
			// apt 版本格式如 "1.22.1-1ubuntu1" → 提取主版本
			if idx := strings.Index(ver, "-"); idx > 0 {
				ver = ver[:idx]
			}
			return ver
		}
	}
	return ""
}

// Upgrade 升级 Nginx
func (s *NginxInstallService) Upgrade(req dto.NginxUpgradeReq) error {
	nc := global.CONF.Nginx
	if !nc.IsInstalled() {
		return buserr.New(constant.ErrNginxNotInstalled)
	}

	if nc.IsSystemMode() {
		go s.doUpgradeApt()
	} else {
		if req.Version == "" {
			return fmt.Errorf("version is required for precompiled upgrade")
		}
		go s.doUpgradePrecompiled(req.Version)
	}
	return nil
}

// doUpgradeApt 通过 apt 升级 Nginx
func (s *NginxInstallService) doUpgradeApt() {
	s.setProgress("download", "正在更新软件包索引...", 5)
	exec.Command("apt-get", "update", "-qq").Run()
	s.setProgress("download", "索引已更新", 20)

	s.setProgress("install", "正在升级 Nginx...", 30)
	output, err := exec.Command("apt-get", "install", "--only-upgrade", "-y", "nginx").CombinedOutput()
	if err != nil {
		s.setProgress("error", fmt.Sprintf("升级失败: %s", strings.TrimSpace(string(output))), 0)
		return
	}
	s.setProgress("install", "Nginx 已升级", 80)

	if _, err := cmd.ExecWithOutput("systemctl", "reload", "nginx"); err != nil {
		global.LOG.Warnf("Reload nginx after upgrade failed: %v", err)
	}

	global.CONF.Nginx.DetectNginx()
	ver := global.CONF.Nginx.GetVersion()
	s.setProgress("done", fmt.Sprintf("Nginx 已升级到 %s", ver), 100)
	global.LOG.Infof("Nginx upgraded via apt to %s", ver)
}

// doUpgradePrecompiled 通过预编译包升级 Nginx
func (s *NginxInstallService) doUpgradePrecompiled(version string) {
	installDir := global.CONF.Nginx.InstallDir

	// 先停止当前 nginx
	s.setProgress("install", "正在停止 Nginx...", 5)
	pidPath := global.CONF.Nginx.GetPidPath()
	nginxBin := global.CONF.Nginx.GetBinary()
	if _, err := os.Stat(pidPath); err == nil {
		exec.Command(nginxBin, "-p", installDir, "-s", "quit").Run()
		time.Sleep(2 * time.Second)
	}

	// 复用 doInstall 逻辑（它会覆盖安装目录）
	s.doInstall(version, installDir)
}

// setProgress 线程安全地更新进度
func (s *NginxInstallService) setProgress(phase, message string, percent int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.progress = &dto.NginxInstallProgress{
		Phase:   phase,
		Message: message,
		Percent: percent,
	}
	global.LOG.Infof("[nginx-install] [%s] %s (%d%%)", phase, message, percent)
}

// doInstall 从 GitHub Release 下载预编译 Nginx 并安装（在 goroutine 中运行）
func (s *NginxInstallService) doInstall(version, installDir string) {
	buildRepo := global.CONF.Nginx.BuildRepo
	if buildRepo == "" {
		buildRepo = "Anikato/nginx-build"
	}

	arch := runtime.GOARCH
	tag := "v" + version
	pkgName := fmt.Sprintf("nginx-%s-linux-%s", version, arch)

	// Step 1: 获取 Release 信息
	s.setProgress("download", fmt.Sprintf("正在获取 Nginx %s 版本信息...", version), 5)

	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/releases/tags/%s", buildRepo, tag)
	resp, err := http.Get(apiURL)
	if err != nil {
		s.setProgress("error", fmt.Sprintf("连接 GitHub 失败: %v", err), 0)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		s.setProgress("error", fmt.Sprintf("未找到 Nginx %s 的预编译版本 (HTTP %d)，请确认 nginx-build 仓库已发布该版本", version, resp.StatusCode), 0)
		return
	}

	var release struct {
		Assets []struct {
			Name               string `json:"name"`
			Size               int64  `json:"size"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		s.setProgress("error", fmt.Sprintf("解析 Release 信息失败: %v", err), 0)
		return
	}

	// 查找对应架构的 tar.gz 和 sha256 文件
	var downloadURL, checksumURL string
	for _, a := range release.Assets {
		if a.Name == pkgName+".tar.gz" {
			downloadURL = a.BrowserDownloadURL
		}
		if a.Name == pkgName+".tar.gz.sha256" {
			checksumURL = a.BrowserDownloadURL
		}
	}

	if downloadURL == "" {
		s.setProgress("error", fmt.Sprintf("未找到 %s 架构的预编译包", arch), 0)
		return
	}

	// Step 2: 下载预编译包
	s.setProgress("download", fmt.Sprintf("正在下载 Nginx %s (%s)...", version, arch), 10)

	tmpDir, err := os.MkdirTemp("", "nginx-install-*")
	if err != nil {
		s.setProgress("error", fmt.Sprintf("创建临时目录失败: %v", err), 0)
		return
	}
	defer os.RemoveAll(tmpDir)

	tarballPath := filepath.Join(tmpDir, pkgName+".tar.gz")
	if err := downloadToFile(downloadURL, tarballPath); err != nil {
		s.setProgress("error", fmt.Sprintf("下载失败: %v", err), 0)
		return
	}
	s.setProgress("download", "下载完成", 40)

	// Step 3: 校验 SHA256
	if checksumURL != "" {
		s.setProgress("verify", "正在校验文件完整性...", 45)
		checksumPath := filepath.Join(tmpDir, "checksum.sha256")
		if err := downloadToFile(checksumURL, checksumPath); err == nil {
			checksumData, _ := os.ReadFile(checksumPath)
			expectedHash := strings.Fields(string(checksumData))[0]
			actualHash, _ := computeSHA256(tarballPath)
			if actualHash != "" && expectedHash != actualHash {
				s.setProgress("error", fmt.Sprintf("SHA256 校验失败\n期望: %s\n实际: %s", expectedHash, actualHash), 0)
				return
			}
			s.setProgress("verify", "SHA256 校验通过 ✓", 50)
		} else {
			s.setProgress("verify", "跳过校验（无法下载校验文件）", 50)
		}
	}

	// Step 4: 解压安装
	s.setProgress("install", "正在解压安装...", 55)

	// 创建安装目录
	if err := os.MkdirAll(installDir, 0755); err != nil {
		s.setProgress("error", fmt.Sprintf("创建安装目录失败: %v", err), 0)
		return
	}

	// 解压到安装目录
	if err := extractTarGz(tarballPath, installDir); err != nil {
		s.setProgress("error", fmt.Sprintf("解压失败: %v", err), 0)
		return
	}
	s.setProgress("install", "解压完成", 75)

	// Step 5: 设置权限和目录结构
	s.setProgress("install", "配置目录结构...", 80)

	// 确保 nginx 二进制可执行
	nginxBin := filepath.Join(installDir, "sbin", "nginx")
	if err := os.Chmod(nginxBin, 0755); err != nil {
		s.setProgress("error", fmt.Sprintf("设置可执行权限失败: %v", err), 0)
		return
	}

	// 创建额外目录
	extraDirs := []string{
		filepath.Join(installDir, "conf", "conf.d"),
		filepath.Join(installDir, "temp", "client_body"),
		filepath.Join(installDir, "temp", "proxy"),
		filepath.Join(installDir, "temp", "fastcgi"),
		filepath.Join(installDir, "temp", "uwsgi"),
		filepath.Join(installDir, "temp", "scgi"),
	}
	for _, d := range extraDirs {
		os.MkdirAll(d, 0755)
	}

	// Step 6: 更新配置
	s.setProgress("install", "更新配置...", 90)
	global.CONF.Nginx.Version = version
	if global.Vp != nil {
		global.Vp.Set("nginx.version", version)
		_ = global.Vp.WriteConfig()
	}

	// Step 7: 创建 systemd service 文件并默认启用开机自启
	s.setProgress("install", "创建 systemd 服务...", 95)
	if err := EnsureNginxServiceFile(installDir); err != nil {
		global.LOG.Warnf("Create systemd service failed: %v", err)
	} else {
		if _, err := cmd.ExecWithOutput("systemctl", "enable", "xpanel-nginx"); err != nil {
			global.LOG.Warnf("Enable nginx autostart failed: %v", err)
		}
	}

	// Step 8: 重新检测并完成
	global.CONF.Nginx.DetectNginx()
	s.setProgress("done", fmt.Sprintf("Nginx %s 安装成功", version), 100)
	global.LOG.Infof("Nginx %s installed at %s (pre-compiled)", version, installDir)
}

// downloadToFile 下载 URL 到本地文件
func downloadToFile(url, destPath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	f, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("create file failed: %v", err)
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	return err
}

// extractTarGz 解压 tar.gz 到目标目录
func extractTarGz(tarball, destDir string) error {
	cmd := exec.Command("tar", "-xzf", tarball, "-C", destDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %v", string(output), err)
	}
	return nil
}

// computeSHA256 计算文件的 SHA256 哈希
func computeSHA256(filePath string) (string, error) {
	cmd := exec.Command("sha256sum", filePath)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	fields := strings.Fields(string(output))
	if len(fields) < 1 {
		return "", fmt.Errorf("unexpected sha256sum output")
	}
	return fields[0], nil
}
