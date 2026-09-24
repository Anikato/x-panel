package service

import (
	"time"

	"xpanel/app/model"
	"xpanel/app/repo"
	"xpanel/global"
)

func replaceUnusableWebsiteCertificates(now time.Time) {
	if global.DB == nil {
		return
	}
	var sites []model.Website
	if err := global.DB.Where("certificate_id > 0 AND nginx_conf_path = '' AND config_mode <> ? AND skip_cert_adapt = ?", "source", false).Find(&sites).Error; err != nil {
		logCertificateReplace("读取网站失败: %v", err)
		return
	}
	var certs []model.Certificate
	if err := global.DB.Find(&certs).Error; err != nil {
		logCertificateReplace("读取证书失败: %v", err)
		return
	}
	byID := make(map[uint]model.Certificate, len(certs))
	for _, cert := range certs {
		byID[cert.ID] = cert
	}
	sslDir := NewICertificateService().GetSSLDir()
	websites := NewIWebsiteService().(*WebsiteService)
	for _, site := range sites {
		domains := websiteDomains(site)
		if len(domains) == 0 {
			continue
		}
		current, ok := byID[site.CertificateID]
		if !ok {
			continue
		}
		currentPath, currentKey := existingCertFilePaths(sslDir, current)
		if !certificateNeedsReplacement(inspectCertificateFiles(currentPath, currentKey, domains, now).Status) {
			continue
		}
		replacement := bestReplacementCertificate(certs, site.CertificateID, domains, sslDir, now)
		if replacement == nil {
			logCertificateReplace("网站 %s 的证书已失效，证书库里没有可覆盖其域名的替换证书", site.PrimaryDomain)
			continue
		}
		if err := websites.replaceWebsiteCertificate(site, replacement.ID); err != nil {
			logCertificateReplace("网站 %s 换成证书 %s 失败: %v", site.PrimaryDomain, replacement.PrimaryDomain, err)
			continue
		}
		logCertificateReplace("网站 %s 的证书已失效，已换成 %s (#%d)", site.PrimaryDomain, replacement.PrimaryDomain, replacement.ID)
	}
}

func certificateNeedsReplacement(status string) bool {
	switch status {
	case "expired", "unreadable", "key_mismatch", "domain_mismatch", "not_yet_valid":
		return true
	default:
		return false
	}
}

func bestReplacementCertificate(certs []model.Certificate, currentID uint, domains []string, sslDir string, now time.Time) *model.Certificate {
	var best *model.Certificate
	var bestExpiry time.Time
	for i := range certs {
		cert := &certs[i]
		if cert.ID == 0 || cert.ID == currentID {
			continue
		}
		certPath, keyPath := existingCertFilePaths(sslDir, *cert)
		snapshot := inspectCertificateFiles(certPath, keyPath, domains, now)
		if snapshot.Status != "valid" && snapshot.Status != "expiring" {
			continue
		}
		if best == nil || snapshot.NotAfter.After(bestExpiry) || (snapshot.NotAfter.Equal(bestExpiry) && cert.ID > best.ID) {
			best = cert
			bestExpiry = snapshot.NotAfter
		}
	}
	return best
}

func (s *WebsiteService) replaceWebsiteCertificate(site model.Website, certID uint) error {
	previous := site.CertificateID
	site.CertificateID = certID
	site.SSLEnable = true
	if s.websiteRepo == nil {
		s.websiteRepo = repo.NewIWebsiteRepo()
	}
	if err := s.websiteRepo.Save(&site); err != nil {
		return err
	}
	if site.Status != "running" {
		return nil
	}
	if err := s.applyConfig(site); err != nil {
		site.CertificateID = previous
		_ = s.websiteRepo.Save(&site)
		if previous > 0 {
			_ = s.applyConfig(site)
		}
		return err
	}
	return nil
}

func logCertificateReplace(format string, args ...interface{}) {
	if global.LOG != nil {
		global.LOG.Infof("[cert-replace] "+format, args...)
	}
}
