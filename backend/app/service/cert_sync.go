package service

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"xpanel/app/dto"
	"xpanel/app/model"
	"xpanel/app/repo"
	"xpanel/buserr"
	"xpanel/constant"
	"xpanel/global"
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
			SyncInterval:    src.SyncInterval,
			PostSyncCommand: src.PostSyncCommand,
			Enabled:         src.Enabled,
			LastSyncAt:      src.LastSyncAt,
			LastSyncStatus:  src.LastSyncStatus,
			LastSyncMessage: src.LastSyncMessage,
			CreatedAt:       src.CreatedAt,
		})
	}
	return items, nil
}

func (s *CertSourceService) Create(req dto.CertSourceCreate) error {
	source := model.CertSource{
		Name:            req.Name,
		ServerAddr:      strings.TrimRight(req.ServerAddr, "/"),
		Token:           req.Token,
		SyncInterval:    req.SyncInterval,
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
	source.Name = req.Name
	source.ServerAddr = strings.TrimRight(req.ServerAddr, "/")
	if req.Token != "" {
		source.Token = req.Token
	}
	source.SyncInterval = req.SyncInterval
	source.PostSyncCommand = req.PostSyncCommand
	source.Enabled = req.Enabled
	return s.sourceRepo.Save(&source)
}

func (s *CertSourceService) Delete(id uint) error {
	s.logRepo.DeleteBySourceID(id)
	return s.sourceRepo.Delete(repo.WithByID(id))
}

func (s *CertSourceService) TestConnection(id uint) error {
	source, err := s.sourceRepo.Get(repo.WithByID(id))
	if err != nil {
		return buserr.New(constant.ErrRecordNotFound)
	}
	_, err = s.fetchRemoteCerts(source)
	return err
}

// ======================= 同步逻辑 =======================

func (s *CertSourceService) Sync(id uint) error {
	source, err := s.sourceRepo.Get(repo.WithByID(id))
	if err != nil {
		return buserr.New(constant.ErrRecordNotFound)
	}
	return s.syncFromSource(source)
}

func (s *CertSourceService) SyncAll() {
	sources, err := s.sourceRepo.GetList()
	if err != nil {
		global.LOG.Warnf("[cert-sync] Failed to list sources: %v", err)
		return
	}
	now := time.Now()
	for _, src := range sources {
		if !src.Enabled || src.SyncInterval <= 0 {
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

func (s *CertSourceService) syncFromSource(source model.CertSource) error {
	remoteCerts, err := s.fetchRemoteCerts(source)
	if err != nil {
		now := time.Now()
		s.sourceRepo.Update(source.ID, map[string]interface{}{
			"last_sync_at":      &now,
			"last_sync_status":  "error",
			"last_sync_message": err.Error(),
		})
		return err
	}

	sslDir := s.getSSLDir()
	var updatedCount, newCount, skippedCount, errorCount int
	var needReload bool

	for _, remote := range remoteCerts {
		if remote.Pem == "" || remote.PrivateKey == "" {
			continue
		}

		logEntry := model.CertSyncLog{
			SourceID:   source.ID,
			SourceName: source.Name,
			Domain:     remote.PrimaryDomain,
		}

		localCert, localErr := s.findLocalCertByDomain(remote.PrimaryDomain)

		if localErr == nil && localCert.ID > 0 {
			// 本地已存在该域名的证书 — 逐证书对比
			if localCert.ExpireDate.After(remote.ExpireDate) {
				logEntry.Status = "skipped"
				logEntry.Message = fmt.Sprintf("本地到期更晚 (%s > %s)，保留本地版本",
					localCert.ExpireDate.Format("2006-01-02"), remote.ExpireDate.Format("2006-01-02"))
				logEntry.CertificateID = localCert.ID
				s.logRepo.Create(&logEntry)
				skippedCount++
				continue
			}
			if localCert.ExpireDate.Equal(remote.ExpireDate) && localCert.Pem == remote.Pem {
				logEntry.Status = "skipped"
				logEntry.Message = "证书内容无变化"
				logEntry.CertificateID = localCert.ID
				s.logRepo.Create(&logEntry)
				skippedCount++
				continue
			}

			// 远程到期更晚，或到期相同但内容不同（如换了密钥）→ 更新
			localCert.Pem = remote.Pem
			localCert.PrivateKey = remote.PrivateKey
			localCert.ExpireDate = remote.ExpireDate
			localCert.StartDate = remote.StartDate
			localCert.Domains = remote.Domains
			localCert.Status = "applied"
			localCert.Message = fmt.Sprintf("从 %s 同步 (%s)", source.Name, remote.ExpireDate.Format("2006-01-02"))
			if err := s.certRepo.Save(&localCert); err != nil {
				logEntry.Status = "error"
				logEntry.Message = "更新本地证书失败: " + err.Error()
				s.logRepo.Create(&logEntry)
				errorCount++
				continue
			}
			if err := s.saveSyncedCertFiles(sslDir, localCert); err != nil {
				global.LOG.Warnf("[cert-sync] Save cert files failed for %s: %v", remote.PrimaryDomain, err)
			}
			logEntry.Status = "success"
			logEntry.Message = fmt.Sprintf("证书已更新 (新到期 %s)", remote.ExpireDate.Format("2006-01-02"))
			logEntry.CertificateID = localCert.ID
			needReload = true
			s.logRepo.Create(&logEntry)
			updatedCount++
		} else {
			// 本地不存在 → 新建
			newCert := model.Certificate{
				PrimaryDomain: remote.PrimaryDomain,
				Domains:       remote.Domains,
				Provider:      "manual",
				Type:          "synced",
				KeyType:       remote.KeyType,
				Pem:           remote.Pem,
				PrivateKey:    remote.PrivateKey,
				ExpireDate:    remote.ExpireDate,
				StartDate:     remote.StartDate,
				AutoRenew:     false,
				Status:        "applied",
				Description:   fmt.Sprintf("从 %s 同步", source.Name),
			}
			if err := s.certRepo.Create(&newCert); err != nil {
				logEntry.Status = "error"
				logEntry.Message = "创建本地证书失败: " + err.Error()
				s.logRepo.Create(&logEntry)
				errorCount++
				continue
			}
			if err := s.saveSyncedCertFiles(sslDir, newCert); err != nil {
				global.LOG.Warnf("[cert-sync] Save cert files failed for %s: %v", remote.PrimaryDomain, err)
			}
			logEntry.Status = "success"
			logEntry.Message = fmt.Sprintf("新证书已创建 (到期 %s)", remote.ExpireDate.Format("2006-01-02"))
			logEntry.CertificateID = newCert.ID
			needReload = true
			s.logRepo.Create(&logEntry)
			newCount++
		}
	}

	// 仅在有实际变更时才执行同步后动作
	if needReload && source.PostSyncCommand != "" {
		global.LOG.Infof("[cert-sync] Running post-sync command: %s", source.PostSyncCommand)
		out, err := exec.Command("bash", "-c", source.PostSyncCommand).CombinedOutput()
		if err != nil {
			global.LOG.Warnf("[cert-sync] Post-sync command failed: %v, output: %s", err, string(out))
		}
	} else if needReload {
		if global.CONF.Nginx.IsInstalled() {
			if err := reloadNginxGlobal(); err != nil {
				global.LOG.Warnf("[cert-sync] Nginx reload failed: %v", err)
			} else {
				global.LOG.Info("[cert-sync] Nginx reloaded after cert sync")
			}
		}
	}

	now := time.Now()
	msg := fmt.Sprintf("新增 %d, 更新 %d, 跳过 %d, 失败 %d", newCount, updatedCount, skippedCount, errorCount)
	status := "success"
	if errorCount > 0 && newCount == 0 && updatedCount == 0 {
		status = "error"
	}
	s.sourceRepo.Update(source.ID, map[string]interface{}{
		"last_sync_at":      &now,
		"last_sync_status":  status,
		"last_sync_message": msg,
	})

	global.LOG.Infof("[cert-sync] Source %s: %s", source.Name, msg)
	return nil
}

func (s *CertSourceService) fetchRemoteCerts(source model.CertSource) ([]dto.CertServerItem, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
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
		Code int                  `json:"code"`
		Data []dto.CertServerItem `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if result.Code != 200 {
		return nil, fmt.Errorf("server returned code %d", result.Code)
	}
	return result.Data, nil
}

func (s *CertSourceService) findLocalCertByDomain(domain string) (model.Certificate, error) {
	return s.certRepo.Get(repo.WithByPrimaryDomain(domain))
}

func (s *CertSourceService) getSSLDir() string {
	dir, err := s.settingRepo.GetValueByKey("SSLDir")
	if err != nil || dir == "" {
		return global.CONF.GetDefaultSSLDir()
	}
	return dir
}

func (s *CertSourceService) saveSyncedCertFiles(sslDir string, cert model.Certificate) error {
	certDir := filepath.Join(sslDir, "certs", safeDomainDir(cert.PrimaryDomain))
	if err := os.MkdirAll(certDir, 0755); err != nil {
		return fmt.Errorf("create cert dir: %w", err)
	}
	certPath := filepath.Join(certDir, "fullchain.pem")
	if err := os.WriteFile(certPath, []byte(cert.Pem), 0644); err != nil {
		return fmt.Errorf("write fullchain.pem: %w", err)
	}
	keyPath := filepath.Join(certDir, "privkey.pem")
	if err := os.WriteFile(keyPath, []byte(cert.PrivateKey), 0600); err != nil {
		return fmt.Errorf("write privkey.pem: %w", err)
	}
	ensureCertPermissions(certDir, certPath, keyPath)
	return nil
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
		items = append(items, dto.CertServerItem{
			PrimaryDomain: c.PrimaryDomain,
			Domains:       c.Domains,
			Pem:           c.Pem,
			PrivateKey:    c.PrivateKey,
			ExpireDate:    c.ExpireDate,
			StartDate:     c.StartDate,
			KeyType:       c.KeyType,
		})
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
