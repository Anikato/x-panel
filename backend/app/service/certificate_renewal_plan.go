package service

import (
	"time"

	"xpanel/app/dto"
	"xpanel/app/model"
	"xpanel/app/repo"
)

const certificateRenewBefore = 15 * 24 * time.Hour

const (
	renewalManagementLocal  = "local"
	renewalManagementSynced = "synced"
	renewalManagementManual = "manual"

	renewalPlanExpired        = "expired"
	renewalPlanApplying       = "applying"
	renewalPlanSyncPaused     = "sync_paused"
	renewalPlanRenewError     = "renew_error"
	renewalPlanSyncError      = "sync_error"
	renewalPlanRenewDue       = "renew_due"
	renewalPlanExpiringManual = "expiring_manual"
	renewalPlanWaitingSync    = "waiting_sync"
	renewalPlanScheduled      = "scheduled"
	renewalPlanManual         = "manual"
)

func nextLocalAutoRenewAt(cert model.Certificate, now time.Time, location *time.Location) *time.Time {
	if location == nil {
		location = time.Local
	}
	if !cert.AutoRenew || !isRenewableCertificate(cert) ||
		cert.ExpireDate.IsZero() || cert.Pem == "" || cert.PrivateKey == "" {
		return nil
	}

	now = now.In(location)
	windowStart := cert.ExpireDate.In(location).Add(-certificateRenewBefore)
	var next time.Time
	if now.Before(windowStart) {
		next = time.Date(windowStart.Year(), windowStart.Month(), windowStart.Day(), 2, 0, 0, 0, location)
		if next.Before(windowStart) {
			next = next.AddDate(0, 0, 1)
		}
	} else {
		next = time.Date(now.Year(), now.Month(), now.Day(), 2, 0, 0, 0, location)
		if !next.After(now) {
			next = next.AddDate(0, 0, 1)
		}
	}
	return &next
}

func nextCertificateSyncAt(source model.CertSource, now time.Time, location *time.Location) *time.Time {
	if !source.Enabled || source.ResumeRequired || source.SyncInterval <= 0 {
		return nil
	}
	if location == nil {
		location = time.Local
	}

	target := now.In(location)
	if source.LastSyncAt != nil {
		due := source.LastSyncAt.In(location).Add(time.Duration(source.SyncInterval) * time.Minute)
		if due.After(target) {
			target = due
		}
	}

	minuteStart := time.Date(
		target.Year(), target.Month(), target.Day(),
		target.Hour(), target.Minute(), 0, 0, location,
	)
	minuteRemainder := minuteStart.Minute() % 10
	if minuteRemainder == 0 && target.Equal(minuteStart) {
		return &minuteStart
	}

	minutesToAdd := 10 - minuteRemainder
	next := minuteStart.Add(time.Duration(minutesToAdd) * time.Minute)
	return &next
}

func effectiveRenewalMetadata(
	cert model.Certificate,
	now time.Time,
	location *time.Location,
) (bool, *time.Time, *time.Time) {
	switch certificateRenewalManagement(cert) {
	case renewalManagementSynced:
		return cert.UpstreamAutoRenew, cert.LastAutoRenewedAt, cert.UpstreamNextAutoRenewAt
	case renewalManagementManual:
		return false, nil, nil
	default:
		return cert.AutoRenew, cert.LastAutoRenewedAt, nextLocalAutoRenewAt(cert, now, location)
	}
}

func certificateRenewalManagement(cert model.Certificate) string {
	if cert.Type == "synced" || cert.SourceType == "synced" {
		return renewalManagementSynced
	}
	if cert.Type == "upload" || cert.SourceType == "upload" {
		return renewalManagementManual
	}
	return renewalManagementLocal
}

func buildCertificateRenewalPlanItem(
	cert model.Certificate,
	source *model.CertSource,
	now time.Time,
	location *time.Location,
) dto.CertificateRenewalPlanItem {
	managementType := certificateRenewalManagement(cert)
	item := dto.CertificateRenewalPlanItem{
		ID:                cert.ID,
		PrimaryDomain:     cert.PrimaryDomain,
		ManagementType:    managementType,
		SourceID:          cert.SourceID,
		SourceName:        cert.SourceName,
		LastAutoRenewedAt: cert.LastAutoRenewedAt,
	}
	if !cert.ExpireDate.IsZero() {
		expireDate := cert.ExpireDate
		item.ExpireDate = &expireDate
	}

	expired := !cert.ExpireDate.IsZero() && !cert.ExpireDate.After(now)

	switch managementType {
	case renewalManagementLocal:
		item.SourceName = "本机"
		item.AutoRenew = cert.AutoRenew
		item.NextAutoRenewAt = nextLocalAutoRenewAt(cert, now, location)
		switch {
		case expired:
			item.Status = renewalPlanExpired
			item.StatusMessage = "证书已过期"
		case cert.Status == "applying":
			item.Status = renewalPlanApplying
			item.StatusMessage = "证书正在申请或续签"
			item.NextAutoRenewAt = nil
		case cert.Status == "error":
			item.Status = renewalPlanRenewError
			item.StatusMessage = cert.Message
			if item.StatusMessage == "" {
				item.StatusMessage = "最近一次申请或续签失败"
			}
		case item.NextAutoRenewAt == nil:
			item.Status = renewalPlanManual
			item.StatusMessage = "本机自动续签未启用或证书材料不完整"
		case !cert.ExpireDate.After(now.Add(certificateRenewBefore)):
			item.Status = renewalPlanRenewDue
			item.StatusMessage = "已进入自动续签窗口，等待执行或重试"
		default:
			item.Status = renewalPlanScheduled
			item.StatusMessage = "已安排本机自动续签"
		}
	case renewalManagementSynced:
		item.AutoRenew = cert.UpstreamAutoRenew
		item.RenewalMetadataKnown = cert.UpstreamRenewalMetadataKnown
		item.NextAutoRenewAt = cert.UpstreamNextAutoRenewAt
		if source != nil {
			item.SourceID = source.ID
			item.SourceName = source.Name
			item.LastSyncAt = source.LastSyncAt
			item.NextSyncAt = nextCertificateSyncAt(*source, now, location)
		}
		switch {
		case expired:
			item.Status = renewalPlanExpired
			item.StatusMessage = "证书已过期"
		case source == nil:
			item.Status = renewalPlanSyncPaused
			item.StatusMessage = "证书源已不存在"
		case !source.Enabled:
			item.Status = renewalPlanSyncPaused
			item.StatusMessage = "证书源已禁用"
		case source.ResumeRequired:
			item.Status = renewalPlanSyncPaused
			item.StatusMessage = "证书源处于升级保护暂停"
		case source.SyncInterval <= 0:
			item.Status = renewalPlanSyncPaused
			item.StatusMessage = "证书源仅允许手动同步"
		case source.LastSyncStatus == "error":
			item.Status = renewalPlanSyncError
			item.StatusMessage = source.LastSyncMessage
			if item.StatusMessage == "" {
				item.StatusMessage = "最近一次同步失败"
			}
		case !cert.UpstreamRenewalMetadataKnown:
			item.Status = renewalPlanWaitingSync
			item.StatusMessage = "上游版本未提供续签计划，请升级上游面板"
		case !cert.UpstreamAutoRenew:
			item.Status = renewalPlanManual
			item.StatusMessage = "上游未启用自动续签，本机仅按计划同步证书"
		case cert.UpstreamAutoRenew &&
			cert.UpstreamNextAutoRenewAt != nil &&
			cert.UpstreamNextAutoRenewAt.After(now):
			item.Status = renewalPlanScheduled
			item.StatusMessage = "由上游续签，本机按计划同步"
		default:
			item.Status = renewalPlanWaitingSync
			item.StatusMessage = "等待上游更新续签计划，本机按计划同步"
		}
	case renewalManagementManual:
		item.SourceName = "手动导入"
		if expired {
			item.Status = renewalPlanExpired
			item.StatusMessage = "证书已过期"
		} else if !cert.ExpireDate.IsZero() && !cert.ExpireDate.After(now.Add(certificateRenewBefore)) {
			item.Status = renewalPlanExpiringManual
			item.StatusMessage = "证书即将到期，请手动更换"
		} else {
			item.Status = renewalPlanManual
			item.StatusMessage = "手动导入证书需要人工维护"
		}
	}
	return item
}

func (s *CertificateService) SearchRenewalPlan(
	req dto.SearchCertRenewalPlanReq,
) (int64, []dto.CertificateRenewalPlanItem, error) {
	options := []repo.DBOption{
		repo.WithLikeDomain(req.Info),
		repo.WithCertificateManagementType(req.ManagementType),
	}
	total, certificates, err := s.certRepo.Page(req.Page, req.PageSize, options...)
	if err != nil {
		return 0, nil, err
	}

	sources, err := repo.NewICertSourceRepo().GetList()
	if err != nil {
		return 0, nil, err
	}
	sourceByID := make(map[uint]model.CertSource, len(sources))
	for _, source := range sources {
		sourceByID[source.ID] = source
	}

	now := time.Now()
	items := make([]dto.CertificateRenewalPlanItem, 0, len(certificates))
	for _, certificate := range certificates {
		var source *model.CertSource
		if certificateRenewalManagement(certificate) == renewalManagementSynced {
			if found, ok := sourceByID[certificate.SourceID]; ok {
				sourceCopy := found
				source = &sourceCopy
			}
		}
		items = append(items, buildCertificateRenewalPlanItem(certificate, source, now, time.Local))
	}
	return total, items, nil
}
