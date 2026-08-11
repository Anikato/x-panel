package service

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"time"

	"xpanel/app/dto"
	"xpanel/app/model"
	"xpanel/app/repo"
	"xpanel/global"
)

const maxCertificateBatchDelete = 500

var removeCertificateDirectory = os.RemoveAll

func (s *CertificateService) BatchDelete(ids []uint) (*dto.CertificateDeleteResult, error) {
	ids = normalizeCertificateIDs(ids)
	if len(ids) == 0 || len(ids) > maxCertificateBatchDelete {
		return nil, fmt.Errorf("certificate ids must contain between 1 and %d values", maxCertificateBatchDelete)
	}
	certs, err := repo.GetCertificatesByIDs(ids)
	if err != nil {
		return nil, err
	}
	return s.deleteCertificates(ids, certs)
}

func (s *CertificateService) CleanupExpired(now time.Time) (*dto.CertificateDeleteResult, error) {
	certs, err := repo.GetExpiredCertificatesBefore(now)
	if err != nil {
		return nil, err
	}
	if len(certs) == 0 {
		return &dto.CertificateDeleteResult{
			Skipped: []dto.CertificateDeleteIssue{},
			Failed:  []dto.CertificateDeleteIssue{},
		}, nil
	}
	ids := make([]uint, 0, len(certs))
	for _, cert := range certs {
		ids = append(ids, cert.ID)
	}
	return s.deleteCertificates(ids, certs)
}

func normalizeCertificateIDs(ids []uint) []uint {
	seen := make(map[uint]struct{}, len(ids))
	out := make([]uint, 0, len(ids))
	for _, id := range ids {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func (s *CertificateService) deleteCertificates(requested []uint, certs []model.Certificate) (*dto.CertificateDeleteResult, error) {
	result := &dto.CertificateDeleteResult{
		Skipped: []dto.CertificateDeleteIssue{},
		Failed:  []dto.CertificateDeleteIssue{},
	}
	references, err := loadCertificateDeleteReferences(requested)
	if err != nil {
		return nil, err
	}
	byID := make(map[uint]model.Certificate, len(certs))
	for _, cert := range certs {
		byID[cert.ID] = cert
	}
	sslDir := s.GetSSLDir()
	for _, id := range requested {
		cert, ok := byID[id]
		if !ok {
			result.Failed = append(result.Failed, certificateDeleteIssue(id, "", "证书不存在"))
			continue
		}
		if cert.Status == "applying" {
			result.Skipped = append(result.Skipped, certificateDeleteIssue(id, cert.PrimaryDomain, "证书正在申请或续签"))
			continue
		}
		if reason := references[id]; reason != "" {
			result.Skipped = append(result.Skipped, certificateDeleteIssue(id, cert.PrimaryDomain, reason))
			continue
		}
		if err := removeCertificateDirectory(certDirPath(sslDir, cert)); err != nil {
			result.Failed = append(result.Failed, certificateDeleteIssue(id, cert.PrimaryDomain, "删除证书目录失败: "+err.Error()))
			continue
		}
		if err := s.certRepo.Delete(repo.WithByID(id)); err != nil {
			result.Failed = append(result.Failed, certificateDeleteIssue(id, cert.PrimaryDomain, "删除证书记录失败: "+err.Error()))
			continue
		}
		result.DeletedCount++
	}
	return result, nil
}

func certificateDeleteIssue(id uint, domain, reason string) dto.CertificateDeleteIssue {
	return dto.CertificateDeleteIssue{ID: id, Domain: domain, Reason: reason}
}

func loadCertificateDeleteReferences(ids []uint) (map[uint]string, error) {
	refs := make(map[uint]string)
	queries := []struct {
		model  any
		reason string
	}{
		{&model.Website{}, "正在被网站使用"},
		{&model.HAProxyLB{}, "正在被 HAProxy 使用"},
		{&model.GostService{}, "正在被 GOST 使用"},
	}
	for _, query := range queries {
		var referenced []uint
		if err := global.DB.Model(query.model).
			Where("certificate_id IN ?", ids).
			Distinct("certificate_id").
			Pluck("certificate_id", &referenced).Error; err != nil {
			return nil, err
		}
		for _, id := range referenced {
			if refs[id] == "" {
				refs[id] = query.reason
			}
		}
	}
	var panelValues []string
	if err := global.DB.Model(&model.Setting{}).
		Where("`key` = ?", "PanelSSLCertificateID").
		Pluck("value", &panelValues).Error; err != nil {
		return nil, err
	}
	if len(panelValues) > 0 {
		if value, err := strconv.ParseUint(panelValues[0], 10, 64); err == nil && value > 0 {
			id := uint(value)
			for _, candidate := range ids {
				if id == candidate && refs[id] == "" {
					refs[id] = "正在被面板 HTTPS 使用"
				}
			}
		}
	}
	return refs, nil
}
