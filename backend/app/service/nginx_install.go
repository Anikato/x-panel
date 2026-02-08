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
	"xpanel/buserr"
	"xpanel/constant"
	"xpanel/global"
)

type INginxInstallService interface {
	Install(req dto.NginxInstallReq) error
	GetProgress() *dto.NginxInstallProgress
	Uninstall() error
	ListVersions() ([]dto.NginxVersionInfo, error)
}

type NginxInstallService struct {
	mu       sync.Mutex
	progress *dto.NginxInstallProgress
}

func NewINginxInstallService() INginxInstallService {
	return &NginxInstallService{}
}

// Install 从 GitHub Release 下载预编译 Nginx 并安装
func (s *NginxInstallService) Install(req dto.NginxInstallReq) error {
	installDir := global.CONF.Nginx.InstallDir
	if global.CONF.Nginx.IsInstalled() {
		return fmt.Errorf("nginx is already installed at %s", installDir)
	}

	// 异步执行下载安装
	go s.doInstall(req.Version, installDir)
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

// Uninstall 卸载 Nginx（删除安装目录）
func (s *NginxInstallService) Uninstall() error {
	installDir := global.CONF.Nginx.InstallDir
	if !global.CONF.Nginx.IsInstalled() {
		return buserr.New(constant.ErrNginxNotInstalled)
	}

	// 先确保 Nginx 已停止
	nginxBin := global.CONF.Nginx.GetBinary()
	pidPath := global.CONF.Nginx.GetPidPath()
	if _, err := os.Stat(pidPath); err == nil {
		_ = exec.Command(nginxBin, "-p", installDir, "-s", "quit").Run()
		time.Sleep(2 * time.Second)
	}

	if err := os.RemoveAll(installDir); err != nil {
		return buserr.WithDetail(constant.ErrInternalServer,
			fmt.Sprintf("failed to remove %s: %v", installDir, err), err)
	}

	global.LOG.Infof("Nginx uninstalled from %s", installDir)
	return nil
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
		filepath.Join(installDir, "conf", "ssl"),
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

	// Step 7: 完成
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
