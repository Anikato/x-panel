package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"xpanel/app/dto"
	"xpanel/app/model"
	"xpanel/global"
)

type certificateConsumerTargets struct {
	Nginx   bool
	HAProxy bool
	GOST    bool
}

type certificateConsumerRefreshActions struct {
	ReloadNginx   func() error
	VerifyNginx   func() error
	ReloadHAProxy func() error
	ReloadGOST    func() error
}

func refreshUpdatedCertificateConsumers(certIDs []uint) error {
	targets, err := findCertificateConsumerTargets(certIDs)
	if err != nil {
		return err
	}
	return refreshCertificateConsumers(targets, certificateConsumerRefreshActions{
		ReloadNginx: reloadNginxGlobal,
		VerifyNginx: func() error {
			return verifyUpdatedNginxCertificateConsumers(certIDs)
		},
		ReloadHAProxy: func() error {
			if !isHAProxyInstalled() {
				return nil
			}
			return NewIHAProxyService().ApplyChange("证书同步更新", "certificate-sync")
		},
		ReloadGOST: func() error {
			if _, err := os.Stat(gostBinaryPath); os.IsNotExist(err) {
				return nil
			}
			return NewIGostService().SyncAll()
		},
	})
}

func verifyUpdatedNginxCertificateConsumers(certIDs []uint) error {
	if len(certIDs) == 0 {
		return nil
	}
	var sites []model.Website
	if err := global.DB.Where("certificate_id IN ? AND ssl_enable = ? AND status = ?", certIDs, true, "running").
		Find(&sites).Error; err != nil {
		return err
	}
	if len(sites) == 0 {
		return nil
	}
	websiteService := NewIWebsiteService().(*WebsiteService)
	ops := websiteService.effectiveCertificateHealthOps()
	semaphore := make(chan struct{}, 5)
	errorsBySite := make(chan error, len(sites))
	var waitGroup sync.WaitGroup
	for _, site := range sites {
		site := site
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			if err := websiteService.verifyWebsiteLocalCertificate(context.Background(), site, ops); err != nil {
				errorsBySite <- fmt.Errorf("网站 %s: %w", site.PrimaryDomain, err)
			}
		}()
	}
	waitGroup.Wait()
	close(errorsBySite)
	var errs []error
	for err := range errorsBySite {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

func findCertificateConsumerTargets(certIDs []uint) (certificateConsumerTargets, error) {
	grouped, err := listCertificateConsumers(certIDs)
	if err != nil {
		return certificateConsumerTargets{}, err
	}
	var targets certificateConsumerTargets
	nginxRunning := false
	if global.CONF.Nginx.IsInstalled() {
		_, nginxRunning = (&NginxService{}).readPID()
	}
	for _, id := range certIDs {
		for _, consumer := range grouped[id] {
			if !consumer.Active {
				continue
			}
			switch consumer.Kind {
			case dto.CertificateConsumerWebsite, dto.CertificateConsumerNginx:
				if nginxRunning {
					targets.Nginx = true
				}
			case dto.CertificateConsumerHAProxy:
				targets.HAProxy = true
			case dto.CertificateConsumerGOST:
				targets.GOST = true
			}
		}
	}
	return targets, nil
}

func listCertificateConsumers(certIDs []uint) (map[uint][]dto.CertificateConsumer, error) {
	grouped := make(map[uint][]dto.CertificateConsumer, len(certIDs))
	if len(certIDs) == 0 || global.DB == nil {
		return grouped, nil
	}
	wanted := make(map[uint]struct{}, len(certIDs))
	for _, id := range certIDs {
		if id > 0 {
			wanted[id] = struct{}{}
		}
	}
	if len(wanted) == 0 {
		return grouped, nil
	}

	var certs []model.Certificate
	if err := global.DB.Where("id IN ?", certIDs).Find(&certs).Error; err != nil {
		return nil, err
	}
	var sites []model.Website
	if err := global.DB.Find(&sites).Error; err != nil {
		return nil, err
	}
	for _, site := range sites {
		if _, ok := wanted[site.CertificateID]; !ok {
			continue
		}
		addCertificateConsumer(grouped, site.CertificateID, dto.CertificateConsumer{
			Kind:   dto.CertificateConsumerWebsite,
			ID:     site.ID,
			Name:   site.PrimaryDomain,
			Detail: site.Alias,
			Active: site.Status == "running" && site.ConfigMode != "source",
		})
	}

	var lbs []model.HAProxyLB
	if err := global.DB.Where("certificate_id IN ? AND enable_ssl = ?", certIDs, true).Find(&lbs).Error; err != nil {
		return nil, err
	}
	for _, lb := range lbs {
		detail := "已启用"
		if !lb.Enabled {
			detail = "未启用"
		}
		addCertificateConsumer(grouped, lb.CertificateID, dto.CertificateConsumer{
			Kind: dto.CertificateConsumerHAProxy, ID: lb.ID, Name: lb.Name, Detail: detail, Active: lb.Enabled,
		})
	}

	var services []model.GostService
	if err := global.DB.Where("certificate_id IN ? AND custom_cert_path = '' AND custom_key_path = ''", certIDs).Find(&services).Error; err != nil {
		return nil, err
	}
	for _, svc := range services {
		detail := "已启用"
		if !svc.Enabled {
			detail = "未启用"
		}
		addCertificateConsumer(grouped, svc.CertificateID, dto.CertificateConsumer{
			Kind: dto.CertificateConsumerGOST, ID: svc.ID, Name: svc.Name, Detail: detail, Active: svc.Enabled,
		})
	}

	if raw, err := repoSettingValue("PanelSSLCertificateID"); err == nil {
		if id, convErr := strconv.ParseUint(raw, 10, 64); convErr == nil {
			if _, ok := wanted[uint(id)]; ok {
				addCertificateConsumer(grouped, uint(id), dto.CertificateConsumer{
					Kind: dto.CertificateConsumerPanel, Name: "面板 HTTPS", Active: true,
				})
			}
		}
	}

	sslDir := NewICertificateService().GetSSLDir()
	for _, ref := range nginxCertificateReferences(sslDir, certs, sites) {
		if _, ok := wanted[ref.certID]; !ok {
			continue
		}
		if ref.site != nil {
			addCertificateConsumer(grouped, ref.certID, dto.CertificateConsumer{
				Kind: dto.CertificateConsumerWebsite, ID: ref.site.ID, Name: ref.site.PrimaryDomain, Detail: ref.path, Active: true,
			})
			continue
		}
		addCertificateConsumer(grouped, ref.certID, dto.CertificateConsumer{
			Kind: dto.CertificateConsumerNginx, Name: filepath.Base(ref.configFile), Detail: ref.path, Active: true,
		})
	}
	return grouped, nil
}

type nginxCertRef struct {
	certID     uint
	path       string
	configFile string
	site       *model.Website
}

func nginxCertificateReferences(sslDir string, certs []model.Certificate, sites []model.Website) []nginxCertRef {
	if !global.CONF.Nginx.IsInstalled() || len(certs) == 0 {
		return nil
	}
	var refs []nginxCertRef
	for _, file := range nginxLoadedConfigFiles() {
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		site := websiteForConfigFile(sites, file)
		for _, certPath := range sslCertificatePaths(string(data)) {
			certID := certificateIDForPath(sslDir, certs, certPath)
			if certID == 0 {
				continue
			}
			refs = append(refs, nginxCertRef{certID: certID, path: certPath, configFile: file, site: site})
		}
	}
	return refs
}

func nginxLoadedConfigFiles() []string {
	root := global.CONF.Nginx.GetMainConf()
	seen := map[string]bool{}
	var files []string
	var walk func(path, prefix string, depth int)
	walk = func(path, prefix string, depth int) {
		path = filepath.Clean(path)
		if path == "" || depth > 8 || seen[path] {
			return
		}
		seen[path] = true
		data, err := os.ReadFile(path)
		if err != nil {
			return
		}
		files = append(files, path)
		for _, inc := range parseIncludeLines(string(data), prefix) {
			matches, err := filepath.Glob(inc)
			if err != nil || len(matches) == 0 {
				continue
			}
			for _, match := range matches {
				info, err := os.Stat(match)
				if err == nil && !info.IsDir() {
					walk(match, prefix, depth+1)
				}
			}
		}
	}
	walk(root, filepath.Dir(root), 0)
	return files
}

var sslCertificateDirectiveRe = regexp.MustCompile(`(?i)^\s*ssl_certificate\s+(\S+)`)

func sslCertificatePaths(content string) []string {
	var paths []string
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		match := sslCertificateDirectiveRe.FindStringSubmatch(trimmed)
		if match == nil {
			continue
		}
		raw := strings.Trim(match[1], `"'`)
		raw = strings.TrimSuffix(raw, ";")
		if raw == "" || strings.Contains(raw, "$") {
			continue
		}
		paths = append(paths, filepath.Clean(raw))
	}
	return paths
}

func certificateIDForPath(sslDir string, certs []model.Certificate, certPath string) uint {
	dir := filepath.Clean(filepath.Dir(certPath))
	var legacyID uint
	legacyCount := 0
	for _, cert := range certs {
		if filepath.Clean(certDirPath(sslDir, cert)) == dir {
			return cert.ID
		}
		legacy := filepath.Clean(filepath.Join(sslDir, "certs", safeDomainDir(cert.PrimaryDomain)))
		if legacy == dir {
			legacyCount++
			legacyID = cert.ID
		}
	}
	if legacyCount == 1 {
		return legacyID
	}
	return 0
}

func websiteForConfigFile(sites []model.Website, configFile string) *model.Website {
	configFile = filepath.Clean(configFile)
	for i := range sites {
		site := &sites[i]
		if site.NginxConfPath != "" && filepath.Clean(site.NginxConfPath) == configFile {
			return site
		}
		if filepath.Clean(GetSiteConfPath(site.Alias)) == configFile {
			return site
		}
	}
	return nil
}

func addCertificateConsumer(grouped map[uint][]dto.CertificateConsumer, certID uint, consumer dto.CertificateConsumer) {
	if certID == 0 {
		return
	}
	for _, existing := range grouped[certID] {
		if existing.Kind == consumer.Kind && existing.ID == consumer.ID && existing.Name == consumer.Name && existing.Detail == consumer.Detail {
			return
		}
		if consumer.ID != 0 && existing.Kind == consumer.Kind && existing.ID == consumer.ID {
			return
		}
	}
	grouped[certID] = append(grouped[certID], consumer)
}

func repoSettingValue(key string) (string, error) {
	var setting model.Setting
	err := global.DB.Where("`key` = ?", key).First(&setting).Error
	if err != nil {
		return "", err
	}
	return setting.Value, nil
}

func refreshCertificateConsumers(targets certificateConsumerTargets, actions certificateConsumerRefreshActions) error {
	var errs []error
	if targets.Nginx && actions.ReloadNginx != nil {
		if err := actions.ReloadNginx(); err != nil {
			errs = append(errs, fmt.Errorf("Nginx reload: %w", err))
		} else if actions.VerifyNginx != nil {
			if err := actions.VerifyNginx(); err != nil {
				errs = append(errs, fmt.Errorf("Nginx 证书验证: %w", err))
			}
		}
	}
	if targets.HAProxy && actions.ReloadHAProxy != nil {
		if err := actions.ReloadHAProxy(); err != nil {
			errs = append(errs, fmt.Errorf("HAProxy reload: %w", err))
		}
	}
	if targets.GOST && actions.ReloadGOST != nil {
		if err := actions.ReloadGOST(); err != nil {
			errs = append(errs, fmt.Errorf("GOST sync: %w", err))
		}
	}
	return errors.Join(errs...)
}

func runCertificateSyncPostActions(certIDs []uint, postCommand string, refresh func([]uint) error, runCommand func(string) error) error {
	if len(certIDs) == 0 {
		return nil
	}
	var errs []error
	if refresh != nil {
		if err := refresh(certIDs); err != nil {
			errs = append(errs, err)
		}
	}
	if postCommand != "" && runCommand != nil {
		if err := runCommand(postCommand); err != nil {
			errs = append(errs, fmt.Errorf("同步后命令: %w", err))
		}
	}
	return errors.Join(errs...)
}
