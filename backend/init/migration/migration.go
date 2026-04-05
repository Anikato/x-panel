package migration

import (
	"io"
	"os"
	"path/filepath"

	"xpanel/app/model"
	"xpanel/global"
)

// Init 执行数据库自动迁移
func Init() {
	if err := global.DB.AutoMigrate(
		&model.Setting{},
		&model.LoginLog{},
		&model.OperationLog{},
		&model.Host{},
		&model.Command{},
		&model.Group{},
		&model.AcmeAccount{},
		&model.DnsAccount{},
		&model.Certificate{},
		&model.Website{},
		&model.Cronjob{},
		&model.CronjobRecord{},
		&model.DatabaseServer{},
		&model.DatabaseInstance{},
		&model.BackupAccount{},
		&model.BackupRecord{},
		&model.Node{},
		&model.TrafficConfig{},
		&model.TrafficHourly{},
		&model.TrafficSnapshot{},
	); err != nil {
		panic("Failed to auto-migrate database: " + err.Error())
	}

	runOnceDataMigrations()

	initDefaultSettings()
	ensureSSLDir()
	global.LOG.Info("Database migration completed")
}

func runOnceDataMigrations() {
	migrated := func(key string) bool {
		var count int64
		global.DB.Model(&model.Setting{}).Where("`key` = ?", key).Count(&count)
		return count > 0
	}
	markDone := func(key string) {
		global.DB.Create(&model.Setting{Key: key, Value: "done"})
	}

	if !migrated("_mig_website_perf_defaults") {
		global.DB.Exec("UPDATE websites SET gzip_enable = 1, security_headers = 1 WHERE gzip_enable = 0 AND security_headers = 0")
		markDone("_mig_website_perf_defaults")
		global.LOG.Info("Migration: enabled gzip & security headers for existing websites")
	}

	if !migrated("_mig_ssl_dir_independent") {
		migrateSSLCertsToIndependentDir()
		markDone("_mig_ssl_dir_independent")
	}
}

// migrateSSLCertsToIndependentDir 将旧 Nginx 目录下的证书迁移到独立 SSL 目录
func migrateSSLCertsToIndependentDir() {
	newDir := global.CONF.GetDefaultSSLDir()
	newCertsDir := filepath.Join(newDir, "certs")
	os.MkdirAll(newCertsDir, 0755)

	oldDirs := []string{
		"/etc/nginx/ssl/certs",
	}
	installDir := global.CONF.Nginx.InstallDir
	if installDir != "" {
		oldDirs = append(oldDirs, filepath.Join(installDir, "conf", "ssl", "certs"))
	}

	for _, oldCertsDir := range oldDirs {
		if oldCertsDir == newCertsDir {
			continue
		}
		entries, err := os.ReadDir(oldCertsDir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			srcDir := filepath.Join(oldCertsDir, entry.Name())
			dstDir := filepath.Join(newCertsDir, entry.Name())
			if _, err := os.Stat(dstDir); err == nil {
				continue
			}
			if err := copyDir(srcDir, dstDir); err != nil {
				global.LOG.Warnf("Migrate cert %s failed: %v", entry.Name(), err)
				continue
			}
			global.LOG.Infof("Migrated cert directory: %s -> %s", srcDir, dstDir)
		}
	}
}

func copyDir(src, dst string) error {
	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())
		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}
	return nil
}

func copyFile(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

// ensureSSLDir 确保 SSL 证书独立目录存在
func ensureSSLDir() {
	sslDir := global.CONF.GetDefaultSSLDir()
	dirs := []string{
		sslDir,
		sslDir + "/certs",
		sslDir + "/logs",
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			global.LOG.Warnf("Failed to create SSL dir %s: %v", d, err)
		}
	}
}

func initDefaultSettings() {
	defaults := []model.Setting{
		{Key: "UserName", Value: "admin"},
		{Key: "Password", Value: ""},
		{Key: "Language", Value: "zh"},
		{Key: "SessionTimeout", Value: "86400"},
		{Key: "PanelName", Value: "X-Panel"},
		{Key: "Theme", Value: "auto"},
		{Key: "SecurityEntrance", Value: ""},
		{Key: "MFAStatus", Value: "Disable"},
		{Key: "MFASecret", Value: ""},
		{Key: "SSLDir", Value: ""},
		{Key: "UpgradeURL", Value: ""},
		{Key: "GitHubToken", Value: ""},
		{Key: "AgentToken", Value: ""},
	}

	for _, s := range defaults {
		var count int64
		global.DB.Model(&model.Setting{}).Where("`key` = ?", s.Key).Count(&count)
		if count == 0 {
			global.DB.Create(&s)
		}
	}
}
