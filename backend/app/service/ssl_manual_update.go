package service

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"xpanel/app/dto"
	"xpanel/app/model"
	"xpanel/app/repo"
	"xpanel/buserr"
	"xpanel/constant"
	"xpanel/global"

	"gorm.io/gorm"
)

type manualCertificatePair struct {
	pem string
	key string
}

func (pair manualCertificatePair) parse() (*x509.Certificate, *certParsed, error) {
	keyPair, err := tls.X509KeyPair([]byte(pair.pem), []byte(pair.key))
	if err != nil {
		return nil, nil, err
	}
	leaf, err := x509.ParseCertificate(keyPair.Certificate[0])
	if err != nil {
		return nil, nil, err
	}
	parsed, err := parseCertPEM(pair.pem)
	return leaf, parsed, err
}

func (pair manualCertificatePair) equals(other manualCertificatePair) bool {
	return strings.TrimSpace(pair.pem) == strings.TrimSpace(other.pem) &&
		strings.TrimSpace(pair.key) == strings.TrimSpace(other.key)
}

func isManualUploadedCertificate(cert model.Certificate) bool {
	return cert.Type == "upload" && cert.SourceType != "synced"
}

func (s *CertificateService) UpdateManual(req dto.ManualCertificateUpdate) (*dto.ManualCertificateUpdateResult, error) {
	return s.changeManualCertificate(req.ID, manualCertificatePair{pem: req.Certificate, key: req.PrivateKey}, time.Time{})
}

func (s *CertificateService) RestoreManual(req dto.ManualCertificateRestore) (*dto.ManualCertificateUpdateResult, error) {
	if req.SavedAt.IsZero() {
		return nil, buserr.New(constant.ErrInvalidParams)
	}
	return s.changeManualCertificate(req.ID, manualCertificatePair{}, req.SavedAt)
}

func (s *CertificateService) changeManualCertificate(id uint, next manualCertificatePair, restoreAt time.Time) (*dto.ManualCertificateUpdateResult, error) {
	release, err := acquireCertificateRenewal(id)
	if err != nil {
		return nil, err
	}
	defer release()

	cert, err := s.certRepo.Get(repo.WithByID(id))
	if err != nil {
		return nil, buserr.New(constant.ErrRecordNotFound)
	}
	if !isManualUploadedCertificate(cert) {
		return nil, buserr.WithDetail(constant.ErrInvalidParams, "此操作仅适用于手动上传的证书", nil)
	}
	sslDir := s.GetSSLDir()
	if err := s.recoverManualCertificate(sslDir, cert); err != nil {
		return nil, err
	}
	restoring := !restoreAt.IsZero()
	if restoring {
		if cert.BackupSavedAt == nil || cert.BackupPEM == "" || cert.BackupPrivateKey == "" {
			return nil, buserr.WithDetail(constant.ErrInvalidParams, "没有可恢复的证书备份", nil)
		}
		if !cert.BackupSavedAt.Equal(restoreAt) {
			return nil, buserr.WithDetail(constant.ErrInvalidParams, "证书备份已变更，请刷新详情后再试", nil)
		}
		next = manualCertificatePair{pem: cert.BackupPEM, key: cert.BackupPrivateKey}
	}
	leaf, parsed, err := next.parse()
	if err != nil {
		return nil, buserr.WithDetail(constant.ErrInvalidParams, "证书或私钥无效，或两者不匹配", err)
	}
	now := time.Now()
	if !restoring && (now.Before(leaf.NotBefore) || !now.Before(leaf.NotAfter)) {
		return nil, buserr.WithDetail(constant.ErrInvalidParams, "新证书尚未生效或已过期", nil)
	}
	if err := validateManualCertificateWebsiteDomains(id, leaf); err != nil {
		return nil, err
	}

	onDisk := readManualCertificateFiles(sslDir, cert)
	fromRecord := manualCertificatePair{pem: cert.Pem, key: cert.PrivateKey}
	if next.equals(onDisk) && next.equals(fromRecord) {
		return s.activateManualCertificate(id), nil
	}
	previous := onDisk
	_, metadata, err := previous.parse()
	if err != nil {
		previous = fromRecord
		_, metadata, err = previous.parse()
		if err != nil {
			return nil, buserr.WithDetail(constant.ErrInvalidParams, "现有证书或私钥无有效副本，无法备份", nil)
		}
	}
	if err := s.writeManualCertificate(sslDir, cert, next, parsed, previous, metadata.expireDate); err != nil {
		return nil, err
	}
	return s.activateManualCertificate(id), nil
}

func readManualCertificateFiles(sslDir string, cert model.Certificate) manualCertificatePair {
	certPath, keyPath := existingCertFilePaths(sslDir, cert)
	certPEM, _ := os.ReadFile(certPath)
	keyPEM, _ := os.ReadFile(keyPath)
	return manualCertificatePair{pem: string(certPEM), key: string(keyPEM)}
}

func (s *CertificateService) writeManualCertificate(sslDir string, cert model.Certificate, next manualCertificatePair, parsed *certParsed, previous manualCertificatePair, previousExpiry time.Time) error {
	id := cert.ID
	cert.Pem, cert.PrivateKey = next.pem, next.key
	tx, err := prepareSyncedCertFileTransaction(sslDir, cert, true)
	if err != nil {
		return err
	}
	// Save the live pair before publishing either file. The existing rollback
	// version stays available until the new pair and metadata are committed.
	if err := s.certRepo.Update(id, map[string]interface{}{
		"pending_pem": previous.pem, "pending_private_key": previous.key,
	}); err != nil {
		return errors.Join(err, tx.Rollback())
	}
	if err := tx.Commit(); err != nil {
		return errors.Join(err, tx.Rollback())
	}

	updates := map[string]interface{}{
		"pem": next.pem, "private_key": next.key, "status": "applied", "message": "",
		"start_date": parsed.startDate, "expire_date": parsed.expireDate,
		"backup_pem": previous.pem, "backup_private_key": previous.key,
		"backup_saved_at": time.Now().UTC(), "backup_expire_date": previousExpiry,
		"pending_pem": "", "pending_private_key": "",
	}
	addParsedMetadataUpdates(updates, parsed)
	canonicalPath, _ := certFilePaths(sslDir, cert)
	if tx.certPath != canonicalPath {
		// The legacy directory uses the original primary domain in its path.
		updates["primary_domain"] = cert.PrimaryDomain
		updates["domains"] = strings.Join(otherDomains(cert.PrimaryDomain, parsed.domains), ",")
	}
	if err := persistCertificateUpdate(s.certRepo.Update, id, updates); err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return errors.Join(err, rollbackErr)
		}
		return errors.Join(err, s.clearPendingManualCertificate(id))
	}
	if err := tx.Finalize(); err != nil && global.LOG != nil {
		global.LOG.Warnf("Clean up certificate %d backup files: %v", id, err)
	}
	return nil
}

func (s *CertificateService) activateManualCertificate(id uint) *dto.ManualCertificateUpdateResult {
	result := &dto.ManualCertificateUpdateResult{}
	if err := s.activateCertificateConsumers(id, nil); err != nil {
		result.ActivationError = err.Error()
	}
	return result
}

func (s *CertificateService) clearPendingManualCertificate(id uint) error {
	return s.certRepo.Update(id, map[string]interface{}{"pending_pem": "", "pending_private_key": ""})
}

// RecoverInterruptedManualCertificateUpdates restores live files if a manual
// update stopped before the active certificate and its backup were committed.
func RecoverInterruptedManualCertificateUpdates() {
	service := NewICertificateService().(*CertificateService)
	certs, err := service.certRepo.GetList(func(db *gorm.DB) *gorm.DB {
		return db.Where("type = ? AND pending_private_key <> ''", "upload")
	})
	if err != nil {
		if global.LOG != nil {
			global.LOG.Warnf("Find interrupted manual certificate updates: %v", err)
		}
		return
	}
	sslDir := service.GetSSLDir()
	for _, cert := range certs {
		if err := service.recoverManualCertificate(sslDir, cert); err != nil && global.LOG != nil {
			global.LOG.Warnf("Recover interrupted manual certificate %d update: %v", cert.ID, err)
		}
	}
}

func (s *CertificateService) recoverManualCertificate(sslDir string, cert model.Certificate) error {
	if cert.PendingPrivateKey == "" {
		return nil
	}
	pending := manualCertificatePair{pem: cert.PendingPEM, key: cert.PendingPrivateKey}
	if !pending.equals(readManualCertificateFiles(sslDir, cert)) {
		cert.Pem, cert.PrivateKey = pending.pem, pending.key
		if err := saveSyncedCertFilesAtomic(sslDir, cert, true); err != nil {
			return err
		}
	}
	return s.clearPendingManualCertificate(cert.ID)
}

func validateManualCertificateWebsiteDomains(id uint, leaf *x509.Certificate) error {
	var sites []model.Website
	if err := global.DB.Where("certificate_id = ? AND ssl_enable = ?", id, true).Find(&sites).Error; err != nil {
		return err
	}
	for _, site := range sites {
		for _, domain := range websiteDomains(site) {
			if err := leaf.VerifyHostname(domain); err != nil {
				return buserr.WithDetail(constant.ErrInvalidParams, fmt.Sprintf("证书未覆盖已绑定的网站域名 %s", domain), err)
			}
		}
	}
	return nil
}
