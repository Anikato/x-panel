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
	"time"

	"xpanel/app/dto"
	"xpanel/app/version"
	"xpanel/buserr"
	"xpanel/constant"
	"xpanel/global"
)

type IUpgradeService interface {
	GetCurrentVersion() *dto.VersionInfo
	CheckUpdate(req dto.UpgradeCheckReq) (*dto.UpgradeInfo, error)
	DoUpgrade(req dto.UpgradeReq) error
	GetUpgradeLog() (string, error)
}

type UpgradeService struct{}

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
		// 默认从面板设置中读取
		val, _ := settingRepo.GetValueByKey("UpgradeURL")
		if val != "" {
			releaseURL = val
		}
	}
	if releaseURL == "" {
		return nil, buserr.WithDetail(constant.ErrInvalidParams, "release URL not configured", nil)
	}

	// 请求远端版本信息
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
	hasUpdate := remoteInfo.Version != currentVer && remoteInfo.Version != ""

	// 构建下载 URL
	arch := runtime.GOARCH
	downloadURL := ""
	if hasUpdate {
		downloadURL = fmt.Sprintf("%s/xpanel-%s-linux-%s.tar.gz", releaseURL, remoteInfo.Version, arch)
	}

	return &dto.UpgradeInfo{
		CurrentVersion: currentVer,
		LatestVersion:  remoteInfo.Version,
		ReleaseNote:    remoteInfo.ReleaseNote,
		HasUpdate:      hasUpdate,
		DownloadURL:    downloadURL,
		PublishDate:    remoteInfo.PublishDate,
	}, nil
}

// DoUpgrade 执行升级
func (s *UpgradeService) DoUpgrade(req dto.UpgradeReq) error {
	if req.DownloadURL == "" {
		return buserr.WithDetail(constant.ErrInvalidParams, "download URL is required", nil)
	}

	global.LOG.Infof("Starting upgrade from %s, download: %s", version.Version, req.DownloadURL)
	logFile := s.getLogPath()

	// 在后台执行升级
	go s.doUpgradeAsync(req.DownloadURL, req.Version, logFile)

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

func (s *UpgradeService) doUpgradeAsync(downloadURL, newVersion, logFile string) {
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

	// 4. 解压
	writeLog("正在解压...")
	extractDir := filepath.Join(tmpDir, "extract")
	os.MkdirAll(extractDir, 0755)
	cmd := exec.Command("tar", "-xzf", tarball, "-C", extractDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		writeLog("错误：解压失败: %s", string(output))
		return
	}

	// 5. 查找新的二进制文件
	newBinary := filepath.Join(extractDir, "xpanel")
	if _, err := os.Stat(newBinary); os.IsNotExist(err) {
		writeLog("错误：解压目录中未找到 xpanel 二进制文件")
		return
	}
	writeLog("新版本二进制已就绪")

	// 6. 备份当前二进制
	backupPath := execPath + ".bak"
	writeLog("备份当前版本: %s", backupPath)
	if err := copyFile(execPath, backupPath); err != nil {
		writeLog("错误：备份失败: %v", err)
		return
	}

	// 7. 替换二进制
	writeLog("替换二进制文件...")
	if err := copyFile(newBinary, execPath); err != nil {
		writeLog("错误：替换失败: %v，正在回滚...", err)
		copyFile(backupPath, execPath)
		return
	}
	os.Chmod(execPath, 0755)
	writeLog("二进制替换完成")

	// 8. 重启服务
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

// downloadFile 下载文件
func downloadFile(url, dst string) error {
	client := &http.Client{Timeout: 10 * time.Minute}
	resp, err := client.Get(url)
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
