package service

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"xpanel/app/dto"
	"xpanel/buserr"
	"xpanel/constant"
	"xpanel/global"
	"xpanel/utils/cmd"
)

type INginxInstallService interface {
	Install(req dto.NginxInstallReq) error
	GetProgress() *dto.NginxInstallProgress
	Uninstall() error
	CheckDeps() ([]string, error)
}

type NginxInstallService struct {
	mu       sync.Mutex
	progress *dto.NginxInstallProgress
}

func NewINginxInstallService() INginxInstallService {
	return &NginxInstallService{}
}

// Install 从源码编译安装 Nginx
func (s *NginxInstallService) Install(req dto.NginxInstallReq) error {
	installDir := global.CONF.Nginx.InstallDir
	if global.CONF.Nginx.IsInstalled() {
		return fmt.Errorf("nginx is already installed at %s", installDir)
	}

	// 检查编译依赖
	missing, err := s.CheckDeps()
	if err != nil {
		return err
	}
	if len(missing) > 0 {
		return buserr.WithDetail(constant.ErrNginxBuildDeps,
			strings.Join(missing, ", "), nil)
	}

	// 异步执行编译安装
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
		_ = cmd.Exec(nginxBin, "-s", "quit")
		time.Sleep(2 * time.Second)
	}

	if err := os.RemoveAll(installDir); err != nil {
		return buserr.WithDetail(constant.ErrInternalServer,
			fmt.Sprintf("failed to remove %s: %v", installDir, err), err)
	}

	global.LOG.Infof("Nginx uninstalled from %s", installDir)
	return nil
}

// CheckDeps 检查编译 Nginx 所需的系统依赖
func (s *NginxInstallService) CheckDeps() ([]string, error) {
	required := map[string]string{
		"gcc":  "gcc",
		"make": "make",
	}

	// 检查开发库头文件
	libs := map[string]string{
		"pcre":   "libpcre3-dev or pcre-devel",
		"zlib":   "zlib1g-dev or zlib-devel",
		"openssl": "libssl-dev or openssl-devel",
	}

	var missing []string

	// 检查基本工具
	for bin, pkg := range required {
		if _, err := exec.LookPath(bin); err != nil {
			missing = append(missing, pkg)
		}
	}

	// 检查开发库（通过 pkg-config 或头文件路径）
	for lib, pkg := range libs {
		found := false

		// 方式1: pkg-config
		if err := cmd.Exec("pkg-config", "--exists", lib); err == nil {
			found = true
		}

		// 方式2: 常见头文件路径
		if !found {
			headerPaths := getHeaderPaths(lib)
			for _, p := range headerPaths {
				if _, err := os.Stat(p); err == nil {
					found = true
					break
				}
			}
		}

		if !found {
			missing = append(missing, pkg)
		}
	}

	return missing, nil
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

// doInstall 实际执行编译安装（在 goroutine 中运行）
func (s *NginxInstallService) doInstall(version, installDir string) {
	s.setProgress("download", fmt.Sprintf("正在下载 nginx-%s 源码...", version), 5)

	// 创建临时构建目录
	buildDir := filepath.Join(os.TempDir(), fmt.Sprintf("nginx-build-%d", time.Now().Unix()))
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		s.setProgress("error", fmt.Sprintf("创建构建目录失败: %v", err), 0)
		return
	}
	defer os.RemoveAll(buildDir)

	tarball := fmt.Sprintf("nginx-%s.tar.gz", version)
	tarballPath := filepath.Join(buildDir, tarball)
	downloadURL := fmt.Sprintf("http://nginx.org/download/%s", tarball)

	// 下载源码
	output, err := cmd.ExecWithTimeoutAndOutput(5*time.Minute,
		"wget", "-q", "-O", tarballPath, downloadURL)
	if err != nil {
		s.setProgress("error", fmt.Sprintf("下载失败: %s %v", output, err), 0)
		return
	}
	s.setProgress("download", "源码下载完成", 15)

	// 解压
	s.setProgress("download", "正在解压源码...", 18)
	output, err = cmd.ExecWithTimeoutAndOutput(1*time.Minute,
		"tar", "-xzf", tarballPath, "-C", buildDir)
	if err != nil {
		s.setProgress("error", fmt.Sprintf("解压失败: %s %v", output, err), 0)
		return
	}

	srcDir := filepath.Join(buildDir, fmt.Sprintf("nginx-%s", version))
	s.setProgress("configure", "正在配置编译选项...", 20)

	// configure
	configureArgs := []string{
		"./configure",
		"--prefix=" + installDir,
		"--sbin-path=" + filepath.Join(installDir, "sbin", "nginx"),
		"--conf-path=" + filepath.Join(installDir, "conf", "nginx.conf"),
		"--pid-path=" + filepath.Join(installDir, "logs", "nginx.pid"),
		"--error-log-path=" + filepath.Join(installDir, "logs", "error.log"),
		"--http-log-path=" + filepath.Join(installDir, "logs", "access.log"),
		"--with-http_ssl_module",
		"--with-http_v2_module",
		"--with-http_realip_module",
		"--with-http_gzip_static_module",
		"--with-http_stub_status_module",
		"--with-stream",
		"--with-stream_ssl_module",
		"--with-pcre",
	}

	configCmd := exec.Command("sh", "-c", strings.Join(configureArgs, " "))
	configCmd.Dir = srcDir
	configOutput, err := configCmd.CombinedOutput()
	if err != nil {
		s.setProgress("error", fmt.Sprintf("configure 失败: %s", string(configOutput)), 0)
		return
	}
	s.setProgress("configure", "配置完成", 30)

	// 获取 CPU 核数用于并行编译
	nproc := "1"
	if out, err := cmd.ExecWithOutput("nproc"); err == nil {
		nproc = strings.TrimSpace(out)
	}

	// make
	s.setProgress("compile", fmt.Sprintf("正在编译（使用 %s 核心）...", nproc), 35)
	makeCmd := exec.Command("make", "-j"+nproc)
	makeCmd.Dir = srcDir
	makeOutput, err := makeCmd.CombinedOutput()
	if err != nil {
		s.setProgress("error", fmt.Sprintf("编译失败: %s", string(makeOutput)), 0)
		return
	}
	s.setProgress("compile", "编译完成", 80)

	// make install
	s.setProgress("install", "正在安装...", 85)
	installCmd := exec.Command("make", "install")
	installCmd.Dir = srcDir
	installOutput, err := installCmd.CombinedOutput()
	if err != nil {
		s.setProgress("error", fmt.Sprintf("安装失败: %s", string(installOutput)), 0)
		return
	}
	s.setProgress("install", "安装完成", 90)

	// 创建额外目录
	s.setProgress("install", "创建目录结构...", 92)
	extraDirs := []string{
		filepath.Join(installDir, "conf", "conf.d"),
		filepath.Join(installDir, "conf", "ssl"),
		filepath.Join(installDir, "temp", "client_body"),
		filepath.Join(installDir, "temp", "proxy"),
		filepath.Join(installDir, "temp", "fastcgi"),
	}
	for _, d := range extraDirs {
		os.MkdirAll(d, 0755)
	}

	// 更新配置中的版本号
	global.CONF.Nginx.Version = version
	if global.Vp != nil {
		global.Vp.Set("nginx.version", version)
		_ = global.Vp.WriteConfig()
	}

	s.setProgress("done", fmt.Sprintf("Nginx %s 安装成功", version), 100)
	global.LOG.Infof("Nginx %s installed at %s", version, installDir)
}

// getHeaderPaths 返回给定库的常见头文件路径
func getHeaderPaths(lib string) []string {
	switch lib {
	case "pcre":
		return []string{
			"/usr/include/pcre.h",
			"/usr/local/include/pcre.h",
		}
	case "zlib":
		return []string{
			"/usr/include/zlib.h",
			"/usr/local/include/zlib.h",
		}
	case "openssl":
		return []string{
			"/usr/include/openssl/ssl.h",
			"/usr/local/include/openssl/ssl.h",
		}
	default:
		return nil
	}
}
