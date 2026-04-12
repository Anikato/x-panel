package global

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

type ipTracker interface {
	IncrementFail(ip string)
	NeedCaptcha(ip string) bool
	Clear(ip string)
}

var (
	DB        *gorm.DB
	MonitorDB *gorm.DB
	LOG       *logrus.Logger
	CONF      ServerConfig
	Vp        *viper.Viper
	I18n      *i18n.Localizer
	IPTracker ipTracker
	CRON      *cron.Cron

	MonitorCronID cron.EntryID
)

// ServerConfig 服务器配置结构
type ServerConfig struct {
	System SystemConfig `mapstructure:"system"`
	Log    LogConfig    `mapstructure:"log"`
	Nginx  NginxConfig  `mapstructure:"nginx"`
}

// GetDefaultSSLDir 返回默认 SSL 证书目录（独立于 Nginx，不会因 Nginx 卸载而丢失）
func (c ServerConfig) GetDefaultSSLDir() string {
	return filepath.Join(c.System.DataDir, "ssl")
}

// SystemConfig 系统配置
type SystemConfig struct {
	Port           string    `mapstructure:"port"`
	Mode           string    `mapstructure:"mode"`
	DataDir        string    `mapstructure:"data_dir"`
	DbPath         string    `mapstructure:"db_path"`
	JwtSecret      string    `mapstructure:"jwt_secret"`
	SessionTimeout int       `mapstructure:"session_timeout"`
	SSL            SSLConfig `mapstructure:"ssl"`
}

// SSLConfig 面板 SSL/TLS 配置
type SSLConfig struct {
	Enable   bool   `mapstructure:"enable"`
	CertPath string `mapstructure:"cert_path"`
	KeyPath  string `mapstructure:"key_path"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level    string `mapstructure:"level"`
	Path     string `mapstructure:"path"`
	MaxSize  int    `mapstructure:"max_size"`
	MaxAge   int    `mapstructure:"max_age"`
	Compress bool   `mapstructure:"compress"`
}

// NginxConfig Nginx 配置 - 同时支持自包含安装和 apt 系统包
type NginxConfig struct {
	InstallDir string `mapstructure:"install_dir"`
	Version    string `mapstructure:"version"`
	BuildRepo  string `mapstructure:"build_repo"`
	Mode       string `mapstructure:"mode"` // "auto"(default) / "system" / "prefix"
	// 运行时检测结果（不从配置文件读取）
	systemMode    bool
	prefixExist   bool
	systemExist   bool
	systemBinary  string
	systemConfDir string
}

// DetectNginx 检测 nginx 安装模式（启动时调用一次）
//
// 优先级规则：
//   - mode=system  → 强制系统包模式（忽略自包含安装）
//   - mode=prefix  → 强制自包含模式（忽略系统 nginx）
//   - mode=auto/空 → 系统 nginx 优先，回退到自包含安装
func (c *NginxConfig) DetectNginx() {
	// 重置检测状态
	c.prefixExist = false
	c.systemExist = false
	c.systemBinary = ""
	c.systemConfDir = ""

	// 探测两种安装是否存在
	if c.InstallDir != "" {
		prefixBin := filepath.Join(c.InstallDir, "sbin", "nginx")
		if _, err := os.Stat(prefixBin); err == nil {
			c.prefixExist = true
		}
	}
	binPath, err := exec.LookPath("nginx")
	if err == nil {
		c.systemExist = true
		c.systemBinary = binPath
		if _, err := os.Stat("/etc/nginx/nginx.conf"); err == nil {
			c.systemConfDir = "/etc/nginx"
		}
	}

	// 根据 mode 决定使用哪种
	switch strings.ToLower(c.Mode) {
	case "system":
		c.systemMode = true
	case "prefix":
		c.systemMode = false
	default: // "auto" or empty — 系统 nginx 优先
		if c.systemExist {
			c.systemMode = true
		} else {
			c.systemMode = false
		}
	}
}

// HasBothInstalled 两种 nginx 是否同时存在
func (c NginxConfig) HasBothInstalled() bool {
	return c.prefixExist && c.systemExist
}

func (c NginxConfig) HasSystemInstalled() bool { return c.systemExist }
func (c NginxConfig) HasPrefixInstalled() bool  { return c.prefixExist }

// IsSystemMode 是否使用系统包管理器安装的 nginx
func (c NginxConfig) IsSystemMode() bool {
	return c.systemMode
}

// GetBinary 返回 Nginx 二进制路径
func (c NginxConfig) GetBinary() string {
	if c.systemMode && c.systemBinary != "" {
		return c.systemBinary
	}
	return filepath.Join(c.InstallDir, "sbin", "nginx")
}

// GetMainConf 返回主配置文件路径
func (c NginxConfig) GetMainConf() string {
	if c.systemMode {
		return "/etc/nginx/nginx.conf"
	}
	return filepath.Join(c.InstallDir, "conf", "nginx.conf")
}

// GetConfDir 返回配置目录路径
func (c NginxConfig) GetConfDir() string {
	if c.systemMode {
		return "/etc/nginx"
	}
	return filepath.Join(c.InstallDir, "conf")
}

// GetSitesDir 返回站点配置目录路径（sites-enabled for system, conf.d for prefix）
func (c NginxConfig) GetSitesDir() string {
	if c.systemMode {
		return "/etc/nginx/sites-enabled"
	}
	return filepath.Join(c.InstallDir, "conf", "conf.d")
}

// GetSitesAvailableDir 返回 sites-available 目录路径（系统模式专用）
func (c NginxConfig) GetSitesAvailableDir() string {
	if c.systemMode {
		return "/etc/nginx/sites-available"
	}
	return filepath.Join(c.InstallDir, "conf", "conf.d")
}

// GetSSLDir 返回 SSL 证书目录路径（已弃用，请使用 ServerConfig.GetDefaultSSLDir）
// 保留用于向后兼容，现在始终返回独立于 Nginx 的路径
func (c NginxConfig) GetSSLDir() string {
	return CONF.GetDefaultSSLDir()
}

// GetLogDir 返回日志目录路径
func (c NginxConfig) GetLogDir() string {
	if c.systemMode {
		return "/var/log/nginx"
	}
	return filepath.Join(c.InstallDir, "logs")
}

// GetPidPath 返回 PID 文件路径
func (c NginxConfig) GetPidPath() string {
	if c.systemMode {
		// Debian/Ubuntu 默认 PID 路径
		for _, p := range []string{"/run/nginx.pid", "/var/run/nginx.pid"} {
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
		return "/run/nginx.pid"
	}
	return filepath.Join(c.InstallDir, "logs", "nginx.pid")
}

// IsInstalled 检查 Nginx 是否已安装
func (c NginxConfig) IsInstalled() bool {
	if c.systemMode {
		return c.systemBinary != ""
	}
	_, err := os.Stat(c.GetBinary())
	return err == nil
}

// GetVersion 获取 nginx 版本
func (c NginxConfig) GetVersion() string {
	bin := c.GetBinary()
	out, err := exec.Command(bin, "-v").CombinedOutput()
	if err != nil {
		return ""
	}
	s := string(out)
	if idx := strings.Index(s, "nginx/"); idx >= 0 {
		ver := s[idx+len("nginx/"):]
		ver = strings.TrimSpace(ver)
		if spIdx := strings.IndexAny(ver, " \n\r"); spIdx >= 0 {
			ver = ver[:spIdx]
		}
		return ver
	}
	return ""
}
