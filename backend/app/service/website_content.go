package service

import (
	"path/filepath"
	"strings"

	"xpanel/app/repo"
	"xpanel/global"
	"xpanel/utils/websitecontent"
)

func managedWebsiteContentPolicy(targetPath string) (websitecontent.Policy, bool) {
	if global.DB == nil {
		return websitecontent.Policy{}, false
	}
	target, err := filepath.EvalSymlinks(filepath.Clean(targetPath))
	if err != nil {
		return websitecontent.Policy{}, false
	}
	sites, err := repo.NewIWebsiteRepo().GetList()
	if err != nil {
		return websitecontent.Policy{}, false
	}
	for _, site := range sites {
		if site.Type != "static" || site.ConfigMode == "source" || strings.TrimSpace(site.SiteDir) == "" {
			continue
		}
		root, err := filepath.EvalSymlinks(filepath.Clean(site.SiteDir))
		if err == nil && websitecontent.Contains(root, target) {
			return websitecontent.DefaultPolicy(), true
		}
	}
	return websitecontent.Policy{}, false
}
