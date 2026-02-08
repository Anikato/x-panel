package global

import (
	"os"
	"path/filepath"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

var (
	DB   *gorm.DB
	LOG  *logrus.Logger
	CONF ServerConfig
	Vp   *viper.Viper
	I18n *i18n.Localizer
)

// ServerConfig 服务器配置结构
type ServerConfig struct {
	System SystemConfig `mapstructure:"system"`
	Log    LogConfig    `mapstructure:"log"`
	Nginx  NginxConfig  `mapstructure:"nginx"`
}

// SystemConfig 系统配置
type SystemConfig struct {
	Port           string `mapstructure:"port"`
	Mode           string `mapstructure:"mode"`
	DataDir        string `mapstructure:"data_dir"`
	DbPath         string `mapstructure:"db_path"`
	JwtSecret      string `mapstructure:"jwt_secret"`
	SessionTimeout int    `mapstructure:"session_timeout"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level    string `mapstructure:"level"`
	Path     string `mapstructure:"path"`
	MaxSize  int    `mapstructure:"max_size"`
	MaxAge   int    `mapstructure:"max_age"`
	Compress bool   `mapstructure:"compress"`
}

// NginxConfig Nginx 自包含安装配置
type NginxConfig struct {
	InstallDir string `mapstructure:"install_dir"` // Nginx 安装根目录
	Version    string `mapstructure:"version"`     // 当前 Nginx 版本
}

// GetBinary 返回 Nginx 二进制路径
func (c NginxConfig) GetBinary() string {
	return filepath.Join(c.InstallDir, "sbin", "nginx")
}

// GetMainConf 返回主配置文件路径
func (c NginxConfig) GetMainConf() string {
	return filepath.Join(c.InstallDir, "conf", "nginx.conf")
}

// GetConfDir 返回配置目录路径
func (c NginxConfig) GetConfDir() string {
	return filepath.Join(c.InstallDir, "conf")
}

// GetSitesDir 返回站点配置目录路径
func (c NginxConfig) GetSitesDir() string {
	return filepath.Join(c.InstallDir, "conf", "conf.d")
}

// GetSSLDir 返回 SSL 证书目录路径
func (c NginxConfig) GetSSLDir() string {
	return filepath.Join(c.InstallDir, "conf", "ssl")
}

// GetLogDir 返回日志目录路径
func (c NginxConfig) GetLogDir() string {
	return filepath.Join(c.InstallDir, "logs")
}

// GetPidPath 返回 PID 文件路径
func (c NginxConfig) GetPidPath() string {
	return filepath.Join(c.InstallDir, "logs", "nginx.pid")
}

// IsInstalled 检查 Nginx 是否已安装
func (c NginxConfig) IsInstalled() bool {
	_, err := os.Stat(c.GetBinary())
	return err == nil
}
