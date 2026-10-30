package migration

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"xpanel/app/model"
	"xpanel/utils/websitecontent"

	"gorm.io/gorm"
)

func repairManagedWebsitePermissions(db *gorm.DB, policy websitecontent.Policy) error {
	var sites []model.Website
	if err := db.Where("type = ? AND config_mode <> ? AND site_dir <> ?", "static", "source", "").Find(&sites).Error; err != nil {
		return err
	}
	for _, site := range sites {
		root := strings.TrimSpace(site.SiteDir)
		if root == "" {
			continue
		}
		if shouldRepairWebsitePath(root, true, policy.UID, 0o700) {
			if err := websitecontent.ApplyDirectory(root, policy); err != nil {
				return err
			}
		}
		indexPath := filepath.Join(root, "index.html")
		if shouldRepairWebsitePath(indexPath, false, policy.UID, 0o600) {
			if err := websitecontent.ApplyFile(indexPath, policy); err != nil {
				return err
			}
		}
	}
	return nil
}

func shouldRepairWebsitePath(path string, directory bool, uid int, mode fs.FileMode) bool {
	info, err := os.Lstat(path)
	if err != nil || info.Mode().Perm() != mode || info.IsDir() != directory {
		return false
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && int(stat.Uid) == uid
}
