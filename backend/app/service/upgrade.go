package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"xpanel/app/dto"
	"xpanel/app/version"
	"xpanel/buserr"
	"xpanel/constant"
	"xpanel/global"
)

const (
	// DefaultGitHubRepo 默认 GitHub 仓库（更新源）
	DefaultGitHubRepo = "Anikato/x-panel"

	// GitHubAPIBase GitHub API 基础地址
	GitHubAPIBase = "https://api.github.com"
)

type IUpgradeService interface {
	GetCurrentVersion() *dto.VersionInfo
	CheckUpdate(req dto.UpgradeCheckReq) (*dto.UpgradeInfo, error)
	DoUpgrade(req dto.UpgradeReq) error
	GetUpgradeLog() (string, error)
}

type UpgradeService struct{}

// 升级互斥锁，防止并发升级
var upgradeMu sync.Mutex
var upgrading bool

func NewIUpgradeService() IUpgradeService {
	return &UpgradeService{}
}

// GetCurrentVersion 获取当前版本信息
func (s *UpgradeService) GetCurrentVersion() *dto.VersionInfo {
	v := version.Get()
	return &dto.VersionInfo{
		Version:    v.Version,
		CommitHash: v.CommitHash,
		BuildTime:  v.BuildTime,
		GoVersion:  v.GoVersion,
	}
}

// CheckUpdate 检查是否有可用更新
func (s *UpgradeService) CheckUpdate(req dto.UpgradeCheckReq) (*dto.UpgradeInfo, error) {
	releaseURL := req.ReleaseURL
	if releaseURL == "" {
		// 从面板设置中读取自定义更新源
		val, _ := settingRepo.GetValueByKey("UpgradeURL")
		if val != "" {
			releaseURL = val
		}
	}

	// 根据 URL 类型选择不同的检查方式
	if releaseURL != "" && !isGitHubURL(releaseURL) {
		// 自建服务器模式（兼容旧版 version.json）
		return s.checkUpdateFromCustomServer(releaseURL)
	}

	// 默认使用 GitHub Releases API
	return s.checkUpdateFromGitHub(releaseURL)
}

// checkUpdateFromGitHub 从 GitHub Releases API 检查更新
func (s *UpgradeService) checkUpdateFromGitHub(repoURL string) (*dto.UpgradeInfo, error) {
	repo := DefaultGitHubRepo
	if repoURL != "" {
		// 从 GitHub URL 中提取 owner/repo
		extracted := extractGitHubRepo(repoURL)
		if extracted != "" {
			repo = extracted
		}
	}

	apiURL := fmt.Sprintf("%s/repos/%s/releases/latest", GitHubAPIBase, repo)

	client := &http.Client{Timeout: 15 * time.Second}
	req, _ := http.NewRequest("GET", apiURL, nil)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "X-Panel/"+version.Version)

	resp, err := client.Do(req)
	if err != nil {
		return nil, buserr.WithDetail(constant.ErrInternalServer, "failed to check update: "+err.Error(), err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		// 没有任何 Release
		return &dto.UpgradeInfo{
			CurrentVersion: version.Version,
			HasUpdate:      false,
		}, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, buserr.WithDetail(constant.ErrInternalServer,
			fmt.Sprintf("GitHub API returned %d", resp.StatusCode), nil)
	}

	var release dto.GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, buserr.WithDetail(constant.ErrInternalServer, "failed to parse GitHub release", err)
	}

	latestVersion := release.TagName
	currentVer := version.Version
	hasUpdate := compareVersions(latestVersion, currentVer) > 0

	// 查找当前架构对应的下载文件
	arch := runtime.GOARCH
	downloadURL := ""
	checksumURL := ""
	for _, asset := range release.Assets {
		if strings.Contains(asset.Name, "linux-"+arch) {
			if strings.HasSuffix(asset.Name, ".tar.gz") && !strings.HasSuffix(asset.Name, ".sha256") {
				downloadURL = asset.BrowserDownloadURL
			}
			if strings.HasSuffix(asset.Name, ".sha256") {
				checksumURL = asset.BrowserDownloadURL
			}
		}
	}

	// 解析发布日期
	publishDate := ""
	if release.PublishedAt != "" {
		if t, err := time.Parse(time.RFC3339, release.PublishedAt); err == nil {
			publishDate = t.Format("2006-01-02")
		}
	}

	return &dto.UpgradeInfo{
		CurrentVersion: currentVer,
		LatestVersion:  latestVersion,
		ReleaseNote:    release.Body,
		HasUpdate:      hasUpdate,
		DownloadURL:    downloadURL,
		ChecksumURL:    checksumURL,
		PublishDate:    publishDate,
	}, nil
}

// checkUpdateFromCustomServer 从自建服务器检查更新（兼容旧版）
func (s *UpgradeService) checkUpdateFromCustomServer(releaseURL string) (*dto.UpgradeInfo, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(releaseURL + "/version.json")
	if err != nil {
		return nil, buserr.WithDetail(constant.ErrInternalServer, "failed to check update: "+err.Error(), err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, buserr.WithDetail(constant.ErrInternalServer,
			fmt.Sprintf("update server returned %d", resp.StatusCode), nil)
	}

	var remoteInfo dto.RemoteVersionInfo
	if err := json.NewDecoder(resp.Body).Decode(&remoteInfo); err != nil {
		return nil, buserr.WithDetail(constant.ErrInternalServer, "failed to parse version info", err)
	}

	currentVer := version.Version
	hasUpdate := compareVersions(remoteInfo.Version, currentVer) > 0

	arch := runtime.GOARCH
	downloadURL := ""
	checksumURL := ""
	if hasUpdate {
		downloadURL = fmt.Sprintf("%s/xpanel-%s-linux-%s.tar.gz", releaseURL, remoteInfo.Version, arch)
		checksumURL = fmt.Sprintf("%s/xpanel-%s-linux-%s.tar.gz.sha256", releaseURL, remoteInfo.Version, arch)
	}

	return &dto.UpgradeInfo{
		CurrentVersion: currentVer,
		LatestVersion:  remoteInfo.Version,
		ReleaseNote:    remoteInfo.ReleaseNote,
		HasUpdate:      hasUpdate,
		DownloadURL:    downloadURL,
		ChecksumURL:    checksumURL,
		PublishDate:    remoteInfo.PublishDate,
	}, nil
}

// DoUpgrade 执行升级
func (s *UpgradeService) DoUpgrade(req dto.UpgradeReq) error {
	if req.DownloadURL == "" {
		return buserr.WithDetail(constant.ErrInvalidParams, "download URL is required", nil)
	}

	// 加互斥锁，防止并发升级
	upgradeMu.Lock()
	if upgrading {
		upgradeMu.Unlock()
		return buserr.New(constant.ErrUpgradeInProgress)
	}
	upgrading = true
	upgradeMu.Unlock()

	global.LOG.Infof("Starting upgrade from %s to %s, download: %s", version.Version, req.Version, req.DownloadURL)
	logFile := s.getLogPath()

	// 在后台执行升级
	go func() {
		defer func() {
			upgradeMu.Lock()
			upgrading = false
			upgradeMu.Unlock()
		}()
		s.doUpgradeAsync(req.DownloadURL, req.ChecksumURL, req.Version, logFile)
	}()

	return nil
}

// GetUpgradeLog 获取升级日志
func (s *UpgradeService) GetUpgradeLog() (string, error) {
	logFile := s.getLogPath()
	content, err := os.ReadFile(logFile)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return string(content), nil
}

func (s *UpgradeService) getLogPath() string {
	dataDir := global.CONF.System.DataDir
	return filepath.Join(dataDir, "log", "upgrade.log")
}

func (s *UpgradeService) doUpgradeAsync(downloadURL, checksumURL, newVersion, logFile string) {
	logger := s.openLog(logFile)
	defer logger.Close()

	writeLog := func(format string, args ...interface{}) {
		msg := fmt.Sprintf("[%s] %s\n", time.Now().Format("15:04:05"), fmt.Sprintf(format, args...))
		logger.WriteString(msg)
		global.LOG.Info(strings.TrimSpace(msg))
	}

	writeLog("开始升级到 %s", newVersion)

	// 1. 获取当前二进制路径
	execPath, err := os.Executable()
	if err != nil {
		writeLog("错误：无法获取当前程序路径: %v", err)
		return
	}
	execPath, _ = filepath.EvalSymlinks(execPath)
	writeLog("当前程序路径: %s", execPath)

	// 2. 创建临时目录
	tmpDir, err := os.MkdirTemp("", "xpanel-upgrade-*")
	if err != nil {
		writeLog("错误：创建临时目录失败: %v", err)
		return
	}
	defer os.RemoveAll(tmpDir)

	// 3. 下载新版本
	writeLog("正在下载: %s", downloadURL)
	tarball := filepath.Join(tmpDir, "xpanel-update.tar.gz")
	if err := downloadFile(downloadURL, tarball); err != nil {
		writeLog("错误：下载失败: %v", err)
		return
	}
	writeLog("下载完成")

	// 4. 校验 SHA256（如果有 checksum URL）
	if checksumURL != "" {
		writeLog("正在验证 SHA256 校验和...")
		checksumFile := filepath.Join(tmpDir, "checksum.sha256")
		if err := downloadFile(checksumURL, checksumFile); err != nil {
			writeLog("警告：下载校验文件失败: %v，跳过校验", err)
		} else {
			if err := verifySHA256(tarball, checksumFile); err != nil {
				writeLog("错误：SHA256 校验失败: %v", err)
				return
			}
			writeLog("SHA256 校验通过")
		}
	}

	// 5. 解压
	writeLog("正在解压...")
	extractDir := filepath.Join(tmpDir, "extract")
	os.MkdirAll(extractDir, 0755)
	cmd := exec.Command("tar", "-xzf", tarball, "-C", extractDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		writeLog("错误：解压失败: %s", string(output))
		return
	}

	// 6. 查找新的二进制文件
	newBinary := filepath.Join(extractDir, "xpanel")
	if _, err := os.Stat(newBinary); os.IsNotExist(err) {
		writeLog("错误：解压目录中未找到 xpanel 二进制文件")
		return
	}
	writeLog("新版本二进制已就绪")

	// 7. 备份当前二进制
	backupPath := execPath + ".bak"
	writeLog("备份当前版本: %s", backupPath)
	if err := copyFile(execPath, backupPath); err != nil {
		writeLog("错误：备份失败: %v", err)
		return
	}

	// 8. 原子替换二进制（先复制到同目录，再 rename）
	writeLog("替换二进制文件...")
	tmpBinary := execPath + ".new"
	if err := copyFile(newBinary, tmpBinary); err != nil {
		writeLog("错误：复制新版本失败: %v，正在回滚...", err)
		os.Remove(tmpBinary)
		return
	}
	os.Chmod(tmpBinary, 0755)

	if err := os.Rename(tmpBinary, execPath); err != nil {
		writeLog("错误：原子替换失败: %v，尝试直接复制...", err)
		// 回退到直接复制
		if err2 := copyFile(newBinary, execPath); err2 != nil {
			writeLog("错误：直接复制也失败: %v，正在回滚...", err2)
			copyFile(backupPath, execPath)
			return
		}
		os.Chmod(execPath, 0755)
	}
	writeLog("二进制替换完成")

	// 9. 重启服务
	writeLog("正在重启服务...")
	writeLog("升级完成！新版本: %s", newVersion)

	// 通过 systemd 重启（如果作为 systemd 服务运行）
	restartCmd := exec.Command("systemctl", "restart", "xpanel")
	if err := restartCmd.Start(); err != nil {
		writeLog("systemctl 重启失败: %v，尝试直接重启...", err)
		// 备选：发送信号给自己
		proc, _ := os.FindProcess(os.Getpid())
		if proc != nil {
			proc.Signal(os.Interrupt)
		}
	}
}

// openLog 打开升级日志文件
func (s *UpgradeService) openLog(logFile string) *os.File {
	os.MkdirAll(filepath.Dir(logFile), 0755)
	f, err := os.Create(logFile)
	if err != nil {
		f, _ = os.CreateTemp("", "xpanel-upgrade-*.log")
	}
	return f
}

// --------- 工具函数 ---------

// downloadFile 下载文件
func downloadFile(url, dst string) error {
	client := &http.Client{Timeout: 10 * time.Minute}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "X-Panel/"+version.Version)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

// copyFile 复制文件
func copyFile(src, dst string) error {
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
	if err != nil {
		return err
	}

	info, _ := os.Stat(src)
	if info != nil {
		os.Chmod(dst, info.Mode())
	}
	return nil
}

// verifySHA256 校验文件的 SHA256
func verifySHA256(filePath, checksumFile string) error {
	// 读取预期的 checksum
	checksumData, err := os.ReadFile(checksumFile)
	if err != nil {
		return fmt.Errorf("read checksum file: %w", err)
	}

	// checksum 文件格式: "hash  filename" 或仅 "hash"
	parts := strings.Fields(strings.TrimSpace(string(checksumData)))
	if len(parts) == 0 {
		return fmt.Errorf("empty checksum file")
	}
	expectedHash := strings.ToLower(parts[0])

	// 计算实际的 checksum
	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return fmt.Errorf("hash file: %w", err)
	}
	actualHash := hex.EncodeToString(h.Sum(nil))

	if actualHash != expectedHash {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedHash, actualHash)
	}
	return nil
}

// compareVersions 语义化版本比较
// 返回值: >0 表示 v1 > v2, <0 表示 v1 < v2, 0 表示相等
func compareVersions(v1, v2 string) int {
	// 去除 "v" 前缀
	v1 = strings.TrimPrefix(v1, "v")
	v2 = strings.TrimPrefix(v2, "v")

	// 处理 dev 版本
	if v1 == "dev" && v2 == "dev" {
		return 0
	}
	if v1 == "dev" {
		return -1 // dev 视为最低版本
	}
	if v2 == "dev" {
		return 1 // 任何版本都比 dev 高
	}

	// 分离主版本号和预发布标识
	// 例如: "1.2.3-beta.1" → main="1.2.3", pre="beta.1"
	v1Main, v1Pre := splitPrerelease(v1)
	v2Main, v2Pre := splitPrerelease(v2)

	// 比较主版本号
	v1Parts := strings.Split(v1Main, ".")
	v2Parts := strings.Split(v2Main, ".")

	maxLen := len(v1Parts)
	if len(v2Parts) > maxLen {
		maxLen = len(v2Parts)
	}

	for i := 0; i < maxLen; i++ {
		var n1, n2 int
		if i < len(v1Parts) {
			n1, _ = strconv.Atoi(v1Parts[i])
		}
		if i < len(v2Parts) {
			n2, _ = strconv.Atoi(v2Parts[i])
		}
		if n1 != n2 {
			return n1 - n2
		}
	}

	// 主版本号相同，比较预发布标识
	// 没有预发布标识的版本优先级更高 (1.0.0 > 1.0.0-beta)
	if v1Pre == "" && v2Pre == "" {
		return 0
	}
	if v1Pre == "" {
		return 1
	}
	if v2Pre == "" {
		return -1
	}

	// 两个都有预发布标识，字符串比较
	if v1Pre < v2Pre {
		return -1
	}
	if v1Pre > v2Pre {
		return 1
	}
	return 0
}

// splitPrerelease 分离版本号和预发布标识
func splitPrerelease(v string) (main, pre string) {
	idx := strings.Index(v, "-")
	if idx < 0 {
		return v, ""
	}
	return v[:idx], v[idx+1:]
}

// isGitHubURL 判断是否为 GitHub URL
func isGitHubURL(url string) bool {
	return strings.Contains(url, "github.com") || strings.Contains(url, "api.github.com")
}

// extractGitHubRepo 从 GitHub URL 提取 owner/repo
// 支持格式:
//   - https://github.com/owner/repo
//   - https://api.github.com/repos/owner/repo
//   - owner/repo
func extractGitHubRepo(url string) string {
	// 直接是 owner/repo 格式
	if !strings.Contains(url, "/") {
		return ""
	}

	// 去除协议前缀
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimSuffix(url, "/")

	// api.github.com/repos/owner/repo
	if strings.HasPrefix(url, "api.github.com/repos/") {
		parts := strings.Split(strings.TrimPrefix(url, "api.github.com/repos/"), "/")
		if len(parts) >= 2 {
			return parts[0] + "/" + parts[1]
		}
	}

	// github.com/owner/repo
	if strings.HasPrefix(url, "github.com/") {
		parts := strings.Split(strings.TrimPrefix(url, "github.com/"), "/")
		if len(parts) >= 2 {
			return parts[0] + "/" + parts[1]
		}
	}

	return ""
}
