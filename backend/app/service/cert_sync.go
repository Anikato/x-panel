package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"xpanel/app/dto"
	"xpanel/app/model"
	"xpanel/app/repo"
	"xpanel/buserr"
	"xpanel/constant"
	"xpanel/global"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ======================= 证书源 CRUD =======================

type ICertSourceService interface {
	GetList() ([]dto.CertSourceInfo, error)
	Create(req dto.CertSourceCreate) error
	Update(req dto.CertSourceUpdate) error
	Delete(id uint) error
	Sync(id uint) error
	SyncAll()
	TestConnection(id uint) error
}

type CertSourceService struct {
	sourceRepo  repo.ICertSourceRepo
	logRepo     repo.ICertSyncLogRepo
	certRepo    repo.ICertificateRepo
	settingRepo repo.ISettingRepo
}

func NewICertSourceService() ICertSourceService {
	return &CertSourceService{
		sourceRepo:  repo.NewICertSourceRepo(),
		logRepo:     repo.NewICertSyncLogRepo(),
		certRepo:    repo.NewICertificateRepo(),
		settingRepo: repo.NewISettingRepo(),
	}
}

func (s *CertSourceService) GetList() ([]dto.CertSourceInfo, error) {
	sources, err := s.sourceRepo.GetList()
	if err != nil {
		return nil, err
	}
	var items []dto.CertSourceInfo
	for _, src := range sources {
		items = append(items, dto.CertSourceInfo{
			ID:              src.ID,
			Name:            src.Name,
			ServerAddr:      src.ServerAddr,
			TLSFingerprint:  src.TLSFingerprint,
			SyncInterval:    src.SyncInterval,
			SyncStrategy:    normalizeSyncStrategy(src.SyncStrategy),
			PostSyncCommand: src.PostSyncCommand,
			Enabled:         src.Enabled,
			ResumeRequired:  src.ResumeRequired,
			LastSyncAt:      src.LastSyncAt,
			LastSyncStatus:  src.LastSyncStatus,
			LastSyncMessage: src.LastSyncMessage,
			CreatedAt:       src.CreatedAt,
		})
	}
	return items, nil
}

func (s *CertSourceService) Create(req dto.CertSourceCreate) error {
	serverAddr, err := normalizeCertSourceServerAddr(req.ServerAddr)
	if err != nil {
		return err
	}
	source := model.CertSource{
		Name:            req.Name,
		ServerAddr:      serverAddr,
		Token:           req.Token,
		TLSFingerprint:  req.TLSFingerprint,
		SyncInterval:    req.SyncInterval,
		SyncStrategy:    normalizeSyncStrategy(req.SyncStrategy),
		PostSyncCommand: req.PostSyncCommand,
		Enabled:         req.Enabled,
	}
	return s.sourceRepo.Create(&source)
}

func (s *CertSourceService) Update(req dto.CertSourceUpdate) error {
	source, err := s.sourceRepo.Get(repo.WithByID(req.ID))
	if err != nil {
		return buserr.New(constant.ErrRecordNotFound)
	}
	serverAddr, err := normalizeCertSourceServerAddr(req.ServerAddr)
	if err != nil {
		return err
	}
	source.Name = req.Name
	source.ServerAddr = serverAddr
	if req.Token != "" {
		source.Token = req.Token
	}
	source.SyncInterval = req.SyncInterval
	source.TLSFingerprint = req.TLSFingerprint
	source.SyncStrategy = normalizeSyncStrategy(req.SyncStrategy)
	source.PostSyncCommand = req.PostSyncCommand
	source.Enabled = req.Enabled
	if req.Enabled {
		source.ResumeRequired = false
	}
	return s.sourceRepo.Save(&source)
}

func (s *CertSourceService) Delete(id uint) error {
	s.logRepo.DeleteBySourceID(id)
	return s.sourceRepo.Delete(repo.WithByID(id))
}

func (s *CertSourceService) TestConnection(id uint) error {
	if err := certificateSyncMigrationReady(); err != nil {
		return err
	}
	source, err := s.sourceRepo.Get(repo.WithByID(id))
	if err != nil {
		return buserr.New(constant.ErrRecordNotFound)
	}
	_, err = s.fetchRemoteCerts(source)
	return err
}

// ======================= 同步逻辑 =======================

func (s *CertSourceService) Sync(id uint) error {
	if err := certificateSyncMigrationReady(); err != nil {
		return err
	}
	source, err := s.sourceRepo.Get(repo.WithByID(id))
	if err != nil {
		return buserr.New(constant.ErrRecordNotFound)
	}
	if !source.Enabled || source.ResumeRequired {
		return fmt.Errorf("证书源已暂停，请编辑并启用后再同步")
	}
	return s.syncFromSource(source)
}

func (s *CertSourceService) SyncAll() {
	if err := certificateSyncMigrationReady(); err != nil {
		global.LOG.Errorf("[cert-sync] Synchronization disabled: %v", err)
		return
	}
	sources, err := s.sourceRepo.GetList()
	if err != nil {
		global.LOG.Warnf("[cert-sync] Failed to list sources: %v", err)
		return
	}
	now := time.Now()
	for _, src := range sources {
		if !src.Enabled || src.ResumeRequired || src.SyncInterval <= 0 {
			continue
		}
		if src.LastSyncAt != nil {
			nextSync := src.LastSyncAt.Add(time.Duration(src.SyncInterval) * time.Minute)
			if now.Before(nextSync) {
				continue
			}
		}
		global.LOG.Infof("[cert-sync] Auto syncing from source: %s", src.Name)
		if err := s.syncFromSource(src); err != nil {
			global.LOG.Errorf("[cert-sync] Sync from %s failed: %v", src.Name, err)
		}
	}
}

func certificateSyncMigrationReady() error {
	var count int64
	if err := global.DB.Model(&model.Setting{}).
		Where("`key` = ? AND value = ?", model.CertificateLineageMigrationKey, "done").
		Count(&count).Error; err != nil {
		return fmt.Errorf("证书同步安全迁移状态无法确认，已拒绝同步: %w", err)
	}
	if count == 0 {
		return fmt.Errorf("证书同步安全迁移尚未完成，已拒绝同步")
	}
	return nil
}

func (s *CertSourceService) syncFromSource(source model.CertSource) error {
	remoteCerts, err := s.fetchRemoteCerts(source)
	if err != nil {
		now := time.Now()
		errMsg := err.Error()
		s.sourceRepo.Update(source.ID, map[string]interface{}{
			"last_sync_at":      &now,
			"last_sync_status":  "error",
			"last_sync_message": errMsg,
		})
		s.logRepo.Create(&model.CertSyncLog{
			SourceID:   source.ID,
			SourceName: source.Name,
			Domain:     "*",
			Status:     "error",
			Message:    "连接失败: " + errMsg,
		})
		return err
	}

	sslDir := s.getSSLDir()
	var updatedCount, newCount, skippedCount, errorCount int
	certIDsToRefresh := make(map[uint]struct{})

	for _, remote := range remoteCerts {
		remote = normalizeRemoteCertItem(remote)
		if remote.Pem == "" || remote.PrivateKey == "" {
			continue
		}

		logEntry := model.CertSyncLog{
			SourceID:   source.ID,
			SourceName: source.Name,
			Domain:     remote.PrimaryDomain,
		}

		localCert, localFound, localErr := s.findLocalCert(remote, source.ID)
		if localErr != nil {
			logEntry.Status = "error"
			logEntry.Message = localErr.Error()
			s.logRepo.Create(&logEntry)
			errorCount++
			continue
		}

		if localFound {
			// 本地已存在该域名的证书 — 逐证书对比
			if localCert.LineageUID != remote.LineageUID || localCert.SourceID != source.ID || localCert.SourceName != source.Name {
				identityUpdates := map[string]interface{}{
					"lineage_uid": remote.LineageUID,
					"source_id":   source.ID,
					"source_name": source.Name,
					"source_type": "synced",
					"type":        "synced",
					"provider":    "manual",
					"auto_renew":  false,
				}
				if err := s.certRepo.Update(localCert.ID, identityUpdates); err != nil {
					logEntry.Status = "error"
					logEntry.Message = "绑定上游证书身份失败: " + err.Error()
					s.logRepo.Create(&logEntry)
					errorCount++
					continue
				}
				localCert.LineageUID = remote.LineageUID
				localCert.SourceID = source.ID
				localCert.SourceName = source.Name
				localCert.SourceType = "synced"
				localCert.Type = "synced"
				localCert.Provider = "manual"
				localCert.AutoRenew = false
			}
			if localCert.ExpireDate.After(remote.ExpireDate) {
				logEntry.Status = "skipped"
				logEntry.Message = fmt.Sprintf("本地到期更晚 (%s > %s)，保留本地版本",
					localCert.ExpireDate.Format("2006-01-02"), remote.ExpireDate.Format("2006-01-02"))
				logEntry.CertificateID = localCert.ID
				s.logRepo.Create(&logEntry)
				skippedCount++
				if source.LastSyncStatus == "warning" {
					certIDsToRefresh[localCert.ID] = struct{}{}
				}
				continue
			}
			if localCert.ExpireDate.Equal(remote.ExpireDate) && localCert.Pem == remote.Pem {
				logEntry.Status = "skipped"
				logEntry.Message = "证书内容无变化"
				logEntry.CertificateID = localCert.ID
				s.logRepo.Create(&logEntry)
				skippedCount++
				if source.LastSyncStatus == "warning" {
					certIDsToRefresh[localCert.ID] = struct{}{}
				}
				continue
			}

			// 远程到期更晚，或到期相同但内容不同（如换了密钥）→ 更新
			localCert.Pem = remote.Pem
			localCert.PrivateKey = remote.PrivateKey
			localCert.PrimaryDomain = remote.PrimaryDomain
			localCert.ExpireDate = remote.ExpireDate
			localCert.StartDate = remote.StartDate
			localCert.Domains = remote.Domains
			localCert.Issuer = remote.Issuer
			localCert.Subject = remote.Subject
			localCert.SerialNumber = remote.SerialNumber
			localCert.Fingerprint = remote.Fingerprint
			localCert.DNSNames = remote.DNSNames
			localCert.LineageUID = remote.LineageUID
			applySyncedCertificateMetadata(&localCert, source.ID, source.Name)
			localCert.Status = "applied"
			localCert.Message = fmt.Sprintf("从 %s 同步 (%s)", source.Name, remote.ExpireDate.Format("2006-01-02"))
			fileTx, err := prepareSyncedCertFileTransaction(sslDir, localCert, true)
			if err != nil {
				logEntry.Status = "error"
				logEntry.Message = "准备本地证书失败: " + err.Error()
				s.logRepo.Create(&logEntry)
				errorCount++
				continue
			}
			if err := fileTx.Commit(); err != nil {
				logEntry.Status = "error"
				logEntry.Message = "写入本地证书失败: " + err.Error()
				s.logRepo.Create(&logEntry)
				errorCount++
				continue
			}
			if err := s.certRepo.Save(&localCert); err != nil {
				if rollbackErr := fileTx.Rollback(); rollbackErr != nil {
					err = errors.Join(err, fmt.Errorf("恢复旧证书文件失败: %w", rollbackErr))
				}
				logEntry.Status = "error"
				logEntry.Message = "更新本地证书失败: " + err.Error()
				s.logRepo.Create(&logEntry)
				errorCount++
				continue
			}
			if err := fileTx.Finalize(); err != nil {
				global.LOG.Warnf("[cert-sync] Clean certificate backups failed for %s: %v", remote.PrimaryDomain, err)
			}
			logEntry.Status = "success"
			logEntry.Message = fmt.Sprintf("证书已更新 (新到期 %s)", remote.ExpireDate.Format("2006-01-02"))
			logEntry.CertificateID = localCert.ID
			certIDsToRefresh[localCert.ID] = struct{}{}
			s.logRepo.Create(&logEntry)
			updatedCount++
		} else {
			// 本地不存在 → 新建
			newCert := model.Certificate{
				LineageUID:    remote.LineageUID,
				PrimaryDomain: remote.PrimaryDomain,
				Domains:       remote.Domains,
				Provider:      "manual",
				Type:          "synced",
				KeyType:       remote.KeyType,
				Pem:           remote.Pem,
				PrivateKey:    remote.PrivateKey,
				ExpireDate:    remote.ExpireDate,
				StartDate:     remote.StartDate,
				Issuer:        remote.Issuer,
				Subject:       remote.Subject,
				SerialNumber:  remote.SerialNumber,
				Fingerprint:   remote.Fingerprint,
				DNSNames:      remote.DNSNames,
				Status:        "applied",
				Description:   fmt.Sprintf("从 %s 同步", source.Name),
			}
			applySyncedCertificateMetadata(&newCert, source.ID, source.Name)
			if err := s.certRepo.Create(&newCert); err != nil {
				logEntry.Status = "error"
				logEntry.Message = "创建本地证书失败: " + err.Error()
				s.logRepo.Create(&logEntry)
				errorCount++
				continue
			}
			if err := saveSyncedCertFilesAtomic(sslDir, newCert, false); err != nil {
				s.certRepo.Delete(repo.WithByID(newCert.ID))
				logEntry.Status = "error"
				logEntry.Message = "写入本地证书失败: " + err.Error()
				s.logRepo.Create(&logEntry)
				errorCount++
				continue
			}
			logEntry.Status = "success"
			logEntry.Message = fmt.Sprintf("新证书已创建 (到期 %s)", remote.ExpireDate.Format("2006-01-02"))
			logEntry.CertificateID = newCert.ID
			certIDsToRefresh[newCert.ID] = struct{}{}
			s.logRepo.Create(&logEntry)
			newCount++
		}
	}

	certIDs := make([]uint, 0, len(certIDsToRefresh))
	for id := range certIDsToRefresh {
		certIDs = append(certIDs, id)
	}
	postActionErr := runCertificateSyncPostActions(certIDs, source.PostSyncCommand, refreshUpdatedCertificateConsumers, func(command string) error {
		global.LOG.Infof("[cert-sync] Running post-sync command: %s", command)
		out, err := exec.Command("bash", "-c", command).CombinedOutput()
		if err != nil {
			return fmt.Errorf("%w, output: %s", err, strings.TrimSpace(string(out)))
		}
		return nil
	})
	if postActionErr != nil {
		global.LOG.Warnf("[cert-sync] Certificate consumer refresh warning: %v", postActionErr)
		s.logRepo.Create(&model.CertSyncLog{
			SourceID: source.ID, SourceName: source.Name, Domain: "*", Status: "warning",
			Message: "证书已保存，但服务刷新失败: " + postActionErr.Error(),
		})
	}

	now := time.Now()
	msg := fmt.Sprintf("新增 %d, 更新 %d, 跳过 %d, 失败 %d", newCount, updatedCount, skippedCount, errorCount)
	status := certificateSyncStatus(errorCount, newCount, updatedCount, postActionErr)
	if postActionErr != nil {
		msg += "; 服务刷新待重试: " + postActionErr.Error()
	}
	s.sourceRepo.Update(source.ID, map[string]interface{}{
		"last_sync_at":      &now,
		"last_sync_status":  status,
		"last_sync_message": msg,
	})

	global.LOG.Infof("[cert-sync] Source %s: %s", source.Name, msg)
	return nil
}

func certificateSyncStatus(errorCount, newCount, updatedCount int, postActionErr error) string {
	if postActionErr != nil {
		return "warning"
	}
	if errorCount > 0 && newCount == 0 && updatedCount == 0 {
		return "error"
	}
	return "success"
}

func (s *CertSourceService) fetchRemoteCerts(source model.CertSource) ([]dto.CertServerItem, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: newCertSourceTLSConfig(),
		},
	}

	url := source.ServerAddr + "/api/v1/cert-server/certs"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("X-Cert-Token", source.Token)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		bodyStr := string(body)
		if resp.StatusCode == 404 {
			if strings.Contains(bodyStr, "nginx") || strings.Contains(bodyStr, "<html") {
				return nil, fmt.Errorf("远程返回 404 (Nginx)：请确认地址是面板直连地址（如 https://IP:面板端口），不要使用经过 Nginx 反向代理的域名；同时确认远程面板已更新到支持证书服务的版本")
			}
			return nil, fmt.Errorf("远程返回 404：该面板可能未启用证书服务或版本不支持，请确认远程面板已更新")
		}
		if resp.StatusCode == 403 {
			return nil, fmt.Errorf("远程面板未启用证书服务功能")
		}
		if resp.StatusCode == 401 {
			return nil, fmt.Errorf("Token 认证失败，请检查 Token 是否正确")
		}
		return nil, fmt.Errorf("远程返回 %d: %s", resp.StatusCode, bodyStr)
	}

	var result struct {
		Code    int                  `json:"code"`
		Message string               `json:"message"`
		Data    []dto.CertServerItem `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}
	if result.Code != 0 {
		return nil, fmt.Errorf("远程返回错误 (code=%d): %s", result.Code, result.Message)
	}
	return result.Data, nil
}

func (s *CertSourceService) findLocalCert(remote dto.CertServerItem, sourceID uint) (model.Certificate, bool, error) {
	if _, err := uuid.Parse(remote.LineageUID); err != nil {
		return model.Certificate{}, false, fmt.Errorf("上游证书缺少有效 lineageUID，请先升级上游面板")
	}
	cert, err := s.certRepo.Get(repo.WithBySourceLineage(sourceID, remote.LineageUID))
	if err == nil {
		return cert, true, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return model.Certificate{}, false, fmt.Errorf("查询同步证书身份失败: %w", err)
	}

	candidates, err := s.certRepo.GetList(repo.WithBySourceID(sourceID))
	if err != nil {
		return model.Certificate{}, false, fmt.Errorf("查询本地证书失败: %w", err)
	}
	synced := syncedCertificateCandidates(candidates)
	if len(synced) == 0 {
		legacyCandidates, err := s.certRepo.GetList(repo.WithBySourceID(0))
		if err != nil {
			return model.Certificate{}, false, fmt.Errorf("查询历史同步证书失败: %w", err)
		}
		synced = syncedCertificateCandidates(legacyCandidates)
	}
	if len(synced) == 0 {
		return model.Certificate{}, false, nil
	}
	cert, err = selectLegacyCertificateWithReferences(remote, synced, s.referencedCertificateIDs(synced))
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return model.Certificate{}, false, nil
		}
		return model.Certificate{}, false, err
	}
	return cert, true, nil
}

func syncedCertificateCandidates(candidates []model.Certificate) []model.Certificate {
	synced := make([]model.Certificate, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.Type == "synced" || candidate.SourceType == "synced" {
			synced = append(synced, candidate)
		}
	}
	return synced
}

func (s *CertSourceService) referencedCertificateIDs(candidates []model.Certificate) map[uint]bool {
	ids := make([]uint, 0, len(candidates))
	for _, candidate := range candidates {
		ids = append(ids, candidate.ID)
	}
	referenced := make(map[uint]bool)
	if len(ids) == 0 {
		return referenced
	}
	for _, target := range []interface{}{&model.Website{}, &model.HAProxyLB{}, &model.GostService{}} {
		var used []uint
		if err := global.DB.Model(target).Where("certificate_id IN ?", ids).Pluck("certificate_id", &used).Error; err == nil {
			for _, id := range used {
				referenced[id] = true
			}
		}
	}
	if raw, err := s.settingRepo.GetValueByKey("PanelSSLCertificateID"); err == nil {
		if id, err := strconv.ParseUint(raw, 10, 64); err == nil {
			referenced[uint(id)] = true
		}
	}
	return referenced
}

func (s *CertSourceService) getSSLDir() string {
	dir, err := s.settingRepo.GetValueByKey("SSLDir")
	if err != nil || dir == "" {
		return global.CONF.GetDefaultSSLDir()
	}
	return dir
}

func applySyncedCertificateMetadata(cert *model.Certificate, sourceID uint, sourceName string) {
	cert.Type = "synced"
	cert.Provider = "manual"
	cert.SourceType = "synced"
	cert.SourceID = sourceID
	cert.SourceName = sourceName
	cert.AutoRenew = false
	cert.AcmeAccountID = 0
	cert.DnsAccountID = 0
	cert.CertURL = ""
}

// ======================= 证书服务端 API =======================

type ICertServerService interface {
	ListCerts() ([]dto.CertServerItem, error)
	GetSetting() (*dto.CertServerSetting, error)
	UpdateSetting(req dto.CertServerSetting) error
}

type CertServerService struct {
	certRepo    repo.ICertificateRepo
	settingRepo repo.ISettingRepo
}

func NewICertServerService() ICertServerService {
	return &CertServerService{
		certRepo:    repo.NewICertificateRepo(),
		settingRepo: repo.NewISettingRepo(),
	}
}

func (s *CertServerService) ListCerts() ([]dto.CertServerItem, error) {
	certs, err := s.certRepo.GetList()
	if err != nil {
		return nil, err
	}
	var items []dto.CertServerItem
	for _, c := range certs {
		if c.Status != "applied" || c.Pem == "" {
			continue
		}
		if (c.Type == "synced" || c.SourceType == "synced") && c.LineageUID == "" {
			continue
		}
		item := dto.CertServerItem{
			LineageUID:    c.LineageUID,
			PrimaryDomain: c.PrimaryDomain,
			Domains:       c.Domains,
			Pem:           c.Pem,
			PrivateKey:    c.PrivateKey,
			ExpireDate:    c.ExpireDate,
			StartDate:     c.StartDate,
			KeyType:       c.KeyType,
			Issuer:        c.Issuer,
			Subject:       c.Subject,
			SerialNumber:  c.SerialNumber,
			Fingerprint:   c.Fingerprint,
			DNSNames:      c.DNSNames,
			SourceType:    c.SourceType,
			SourceName:    c.SourceName,
		}
		items = append(items, normalizeRemoteCertItem(item))
	}
	return items, nil
}

func (s *CertServerService) GetSetting() (*dto.CertServerSetting, error) {
	enabled, _ := s.settingRepo.GetValueByKey("CertServerEnabled")
	token, _ := s.settingRepo.GetValueByKey("CertServerToken")
	return &dto.CertServerSetting{
		Enabled: enabled == "enable",
		Token:   token,
	}, nil
}

func (s *CertServerService) UpdateSetting(req dto.CertServerSetting) error {
	val := "disable"
	if req.Enabled {
		val = "enable"
	}
	if err := s.settingRepo.CreateOrUpdate("CertServerEnabled", val); err != nil {
		return err
	}
	if req.Token != "" {
		if err := s.settingRepo.CreateOrUpdate("CertServerToken", req.Token); err != nil {
			return err
		}
	}
	return nil
}

func normalizeSyncStrategy(strategy string) string {
	switch strategy {
	case "domainIssuerKey", "domainLatest":
		return strategy
	default:
		return "fingerprint"
	}
}

func normalizeRemoteCertItem(item dto.CertServerItem) dto.CertServerItem {
	parsed, err := parseCertPEM(item.Pem)
	if err != nil || parsed == nil {
		return item
	}
	if item.PrimaryDomain == "" {
		item.PrimaryDomain = parsed.primaryDomain
	}
	if item.Domains == "" {
		item.Domains = strings.Join(otherDomains(item.PrimaryDomain, parsed.domains), ",")
	}
	if item.ExpireDate.IsZero() {
		item.ExpireDate = parsed.expireDate
	}
	if item.StartDate.IsZero() {
		item.StartDate = parsed.startDate
	}
	if item.Issuer == "" {
		item.Issuer = parsed.issuer
	}
	if item.Subject == "" {
		item.Subject = parsed.subject
	}
	if item.SerialNumber == "" {
		item.SerialNumber = parsed.serialNumber
	}
	if item.Fingerprint == "" {
		item.Fingerprint = parsed.fingerprint
	}
	if item.DNSNames == "" {
		item.DNSNames = encodeStringList(parsed.dnsNames)
	}
	return item
}

// ======================= 同步日志查询 =======================

type ICertSyncLogService interface {
	SearchWithPage(req dto.SearchCertSyncLogReq) (int64, []dto.CertSyncLogInfo, error)
}

type CertSyncLogService struct {
	logRepo repo.ICertSyncLogRepo
}

func NewICertSyncLogService() ICertSyncLogService {
	return &CertSyncLogService{logRepo: repo.NewICertSyncLogRepo()}
}

func (s *CertSyncLogService) SearchWithPage(req dto.SearchCertSyncLogReq) (int64, []dto.CertSyncLogInfo, error) {
	var opts []repo.DBOption
	if req.SourceID > 0 {
		opts = append(opts, repo.WithBySourceID(req.SourceID))
	}
	total, logs, err := s.logRepo.Page(req.Page, req.PageSize, opts...)
	if err != nil {
		return 0, nil, err
	}
	var items []dto.CertSyncLogInfo
	for _, l := range logs {
		items = append(items, dto.CertSyncLogInfo{
			ID:            l.ID,
			SourceID:      l.SourceID,
			SourceName:    l.SourceName,
			Domain:        l.Domain,
			Status:        l.Status,
			Message:       l.Message,
			CertificateID: l.CertificateID,
			CreatedAt:     l.CreatedAt,
		})
	}
	return total, items, nil
}
