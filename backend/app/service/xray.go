package service

import (
	b64 "encoding/base64"
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"xpanel/app/dto"
	"xpanel/app/model"
	"xpanel/app/repo"
	"xpanel/global"

	"github.com/google/uuid"
)

const (
	xrayBin        = "/data/xray/bin/xray"
	xrayConfigPath = "/data/xray/etc/config.json"
	xrayAPIAddr    = "127.0.0.1:10085"
	xrayAPIPort    = 10085
)

type IXrayService interface {
	// 状态 & 安装
	GetStatus() dto.XrayStatusResponse
	IsInstalled() bool
	StartInstall() error
	GetInstallLog() string
	// 服务控制
	ControlService(action string) error
	// 节点管理
	ListNodes() ([]dto.XrayNodeResponse, error)
	CreateNode(req dto.XrayNodeCreate) error
	UpdateNode(req dto.XrayNodeUpdate) error
	DeleteNode(id uint) error
	ToggleNode(id uint) error
	// 用户管理
	SearchUsers(req dto.XrayUserSearch) (int64, []dto.XrayUserResponse, error)
	CreateUser(req dto.XrayUserCreate) error
	UpdateUser(req dto.XrayUserUpdate) error
	DeleteUser(id uint) error
	// 工具
	GenerateRealityKeys() (dto.XrayGenerateKeyResponse, error)
	GetShareLink(userID uint) (dto.XrayShareLinkResponse, error)
	// 流量历史
	GetTrafficHistory(userID uint, days int) ([]dto.XrayTrafficDaily, error)
	SnapshotDailyTraffic()
	// 定时任务
	SyncTraffic()
	CheckExpiredUsers()
}

func NewIXrayService() IXrayService { return &XrayService{} }

type XrayService struct {
	mu     sync.Mutex // 保护配置写入 & reload
	syncMu sync.Mutex // 防止并发流量同步
}

var (
	xrayNodeRepo = repo.NewIXrayNodeRepo()
	xrayUserRepo = repo.NewIXrayUserRepo()

	installLog     strings.Builder
	installRunning bool
	installMu      sync.Mutex
)

// ============================================================
// 状态 & 安装
// ============================================================

func (s *XrayService) IsInstalled() bool {
	_, err := os.Stat(xrayBin)
	return err == nil
}

func (s *XrayService) StartInstall() error {
	installMu.Lock()
	defer installMu.Unlock()
	if installRunning {
		return fmt.Errorf("installation already in progress")
	}
	installRunning = true
	installLog.Reset()

	scriptPath := "/data/X-Panel/xray-install.sh"
	go func() {
		defer func() {
			installMu.Lock()
			installRunning = false
			installMu.Unlock()
		}()
		cmd := exec.Command("bash", scriptPath, "install", "--without-logfiles")
		cmd.Stdout = &installLog
		cmd.Stderr = &installLog
		if err := cmd.Run(); err != nil {
			installMu.Lock()
			installLog.WriteString(fmt.Sprintf("\n[ERROR] install failed: %v\n", err))
			installMu.Unlock()
			return
		}
		installMu.Lock()
		installLog.WriteString("\n[DONE] Xray installed successfully.\n")
		installMu.Unlock()
		s.mu.Lock()
		_ = s.reloadConfig()
		s.mu.Unlock()
	}()
	return nil
}

func (s *XrayService) GetInstallLog() string {
	installMu.Lock()
	defer installMu.Unlock()
	return installLog.String()
}

func (s *XrayService) GetStatus() dto.XrayStatusResponse {
	resp := dto.XrayStatusResponse{
		Installed:  s.IsInstalled(),
		ConfigPath: xrayConfigPath,
		BinPath:    xrayBin,
	}
	out, _ := exec.Command("systemctl", "is-active", "xray").Output()
	resp.Running = strings.TrimSpace(string(out)) == "active"

	out2, _ := exec.Command("systemctl", "is-enabled", "xray").Output()
	resp.EnabledOnBoot = strings.TrimSpace(string(out2)) == "enabled"

	if resp.Running {
		verOut, _ := exec.Command(xrayBin, "version").Output()
		if len(verOut) > 0 {
			lines := strings.Split(string(verOut), "\n")
			if len(lines) > 0 {
				resp.Version = strings.TrimSpace(lines[0])
			}
		}
	}
	return resp
}

// ControlService 控制 Xray systemd 服务
func (s *XrayService) ControlService(action string) error {
	var args []string
	switch action {
	case "start", "stop", "restart", "enable", "disable":
		args = []string{action, "xray"}
	default:
		return fmt.Errorf("unknown action: %s", action)
	}
	if out, err := exec.Command("systemctl", args...).CombinedOutput(); err != nil {
		return fmt.Errorf("systemctl %s: %s", action, strings.TrimSpace(string(out)))
	}
	return nil
}

// ============================================================
// 节点管理
// ============================================================

func (s *XrayService) ListNodes() ([]dto.XrayNodeResponse, error) {
	nodes, err := xrayNodeRepo.GetList()
	if err != nil {
		return nil, err
	}
	var result []dto.XrayNodeResponse
	for _, n := range nodes {
		count, _ := xrayUserRepo.Count(repo.WithXrayNodeID(n.ID))
		resp := nodeToResponse(n)
		resp.UserCount = count
		result = append(result, resp)
	}
	return result, nil
}

func (s *XrayService) CreateNode(req dto.XrayNodeCreate) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 规范化 network：tcp 别名为 raw
	if req.Network == "tcp" {
		req.Network = "raw"
	}
	if req.ListenAddr == "" {
		req.ListenAddr = "0.0.0.0"
	}

	netJSON, secJSON, err := marshalNodeSettings(req.Network, req.Security,
		req.RawSettings, req.WSSettings, req.GRPCSettings, req.XHTTPSettings, req.HTTPUpgradeSettings,
		req.TLSSettings, req.RealitySettings)
	if err != nil {
		return err
	}

	sniffDest, _ := json.Marshal(req.SniffDestOverride)
	if len(req.SniffDestOverride) == 0 {
		sniffDest = []byte(`["http","tls"]`)
	}
	fallbacksJSON, _ := json.Marshal(req.Fallbacks)
	if fallbacksJSON == nil {
		fallbacksJSON = []byte("[]")
	}

	node := &model.XrayNode{
		Name:              req.Name,
		Protocol:          req.Protocol,
		ListenAddr:        req.ListenAddr,
		Port:              req.Port,
		Network:           req.Network,
		Security:          req.Security,
		NetworkSettings:   netJSON,
		SecuritySettings:  secJSON,
		Flow:              req.Flow,
		SSMethod:          req.SSMethod,
		SSPassword:        req.SSPassword,
		Fallbacks:         string(fallbacksJSON),
		SniffEnabled:      req.SniffEnabled,
		SniffDestOverride: string(sniffDest),
		SniffMetadataOnly: req.SniffMetadataOnly,
		Remark:            req.Remark,
		Enabled:           true,
	}
	if err := xrayNodeRepo.Create(node); err != nil {
		return err
	}
	return s.reloadConfig()
}

func (s *XrayService) UpdateNode(req dto.XrayNodeUpdate) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if req.Network == "tcp" {
		req.Network = "raw"
	}

	node, err := xrayNodeRepo.Get(repo.WithByID(req.ID))
	if err != nil {
		return err
	}

	netJSON, secJSON, err := marshalNodeSettings(req.Network, req.Security,
		req.RawSettings, req.WSSettings, req.GRPCSettings, req.XHTTPSettings, req.HTTPUpgradeSettings,
		req.TLSSettings, req.RealitySettings)
	if err != nil {
		return err
	}

	sniffDest, _ := json.Marshal(req.SniffDestOverride)
	if len(req.SniffDestOverride) == 0 {
		sniffDest = []byte(`["http","tls"]`)
	}
	fallbacksJSON, _ := json.Marshal(req.Fallbacks)
	if fallbacksJSON == nil {
		fallbacksJSON = []byte("[]")
	}

	node.Name = req.Name
	node.ListenAddr = req.ListenAddr
	if node.ListenAddr == "" {
		node.ListenAddr = "0.0.0.0"
	}
	node.Network = req.Network
	node.Security = req.Security
	node.NetworkSettings = netJSON
	node.SecuritySettings = secJSON
	node.Flow = req.Flow
	node.SSMethod = req.SSMethod
	node.SSPassword = req.SSPassword
	node.Fallbacks = string(fallbacksJSON)
	node.SniffEnabled = req.SniffEnabled
	node.SniffDestOverride = string(sniffDest)
	node.SniffMetadataOnly = req.SniffMetadataOnly
	node.Remark = req.Remark
	node.Enabled = req.Enabled

	if err := xrayNodeRepo.Save(&node); err != nil {
		return err
	}
	return s.reloadConfig()
}

func (s *XrayService) DeleteNode(id uint) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := xrayUserRepo.Delete(repo.WithXrayNodeID(id)); err != nil {
		return err
	}
	if err := xrayNodeRepo.Delete(repo.WithByID(id)); err != nil {
		return err
	}
	return s.reloadConfig()
}

func (s *XrayService) ToggleNode(id uint) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	node, err := xrayNodeRepo.Get(repo.WithByID(id))
	if err != nil {
		return err
	}
	node.Enabled = !node.Enabled
	if err := xrayNodeRepo.Save(&node); err != nil {
		return err
	}
	return s.reloadConfig()
}

// ============================================================
// 用户管理
// ============================================================

func (s *XrayService) SearchUsers(req dto.XrayUserSearch) (int64, []dto.XrayUserResponse, error) {
	opts := []repo.DBOption{}
	if req.NodeID > 0 {
		opts = append(opts, repo.WithXrayNodeID(req.NodeID))
	}
	total, users, err := xrayUserRepo.Page(req.Page, req.PageSize, opts...)
	if err != nil {
		return 0, nil, err
	}
	nodeNames := map[uint]string{}
	nodes, _ := xrayNodeRepo.GetList()
	for _, n := range nodes {
		nodeNames[n.ID] = n.Name
	}
	var result []dto.XrayUserResponse
	for _, u := range users {
		result = append(result, dto.XrayUserResponse{
			ID:            u.ID,
			NodeID:        u.NodeID,
			NodeName:      nodeNames[u.NodeID],
			Name:          u.Name,
			UUID:          u.UUID,
			Email:         u.Email,
			Flow:          u.Flow,
			Level:         u.Level,
			ExpireAt:      u.ExpireAt,
			Enabled:       u.Enabled,
			Remark:        u.Remark,
			UploadTotal:   u.UploadTotal,
			DownloadTotal: u.DownloadTotal,
			CreatedAt:     u.CreatedAt,
		})
	}
	return total, result, nil
}

func (s *XrayService) CreateUser(req dto.XrayUserCreate) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	uid := req.UUID
	if uid == "" {
		uid = uuid.New().String()
	}
	// email 用 uuid 前8位 + @xpanel 确保唯一
	emailPrefix := strings.ReplaceAll(uid, "-", "")
	if len(emailPrefix) > 8 {
		emailPrefix = emailPrefix[:8]
	}
	email := fmt.Sprintf("%s@xpanel", emailPrefix)

	user := &model.XrayUser{
		NodeID:   req.NodeID,
		Name:     req.Name,
		UUID:     uid,
		Email:    email,
		Flow:     req.Flow,
		Level:    req.Level,
		ExpireAt: req.ExpireAt,
		Remark:   req.Remark,
		Enabled:  true,
	}
	if err := xrayUserRepo.Create(user); err != nil {
		return err
	}
	return s.reloadConfig()
}

func (s *XrayService) UpdateUser(req dto.XrayUserUpdate) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, err := xrayUserRepo.Get(repo.WithByID(req.ID))
	if err != nil {
		return err
	}
	user.Name = req.Name
	user.Flow = req.Flow
	user.Level = req.Level
	user.ExpireAt = req.ExpireAt
	user.Enabled = req.Enabled
	user.Remark = req.Remark
	if err := xrayUserRepo.Save(&user); err != nil {
		return err
	}
	return s.reloadConfig()
}

func (s *XrayService) DeleteUser(id uint) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := xrayUserRepo.Delete(repo.WithByID(id)); err != nil {
		return err
	}
	return s.reloadConfig()
}

// ============================================================
// 工具
// ============================================================

func (s *XrayService) GenerateRealityKeys() (dto.XrayGenerateKeyResponse, error) {
	out, err := exec.Command(xrayBin, "x25519").Output()
	if err != nil {
		return dto.XrayGenerateKeyResponse{}, fmt.Errorf("generate keys failed: %w", err)
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	resp := dto.XrayGenerateKeyResponse{}
	for _, line := range lines {
		if strings.HasPrefix(line, "Private key:") {
			resp.PrivateKey = strings.TrimSpace(strings.TrimPrefix(line, "Private key:"))
		} else if strings.HasPrefix(line, "Public key:") {
			resp.PublicKey = strings.TrimSpace(strings.TrimPrefix(line, "Public key:"))
		}
	}
	return resp, nil
}

func (s *XrayService) GetShareLink(userID uint) (dto.XrayShareLinkResponse, error) {
	user, err := xrayUserRepo.Get(repo.WithByID(userID))
	if err != nil {
		return dto.XrayShareLinkResponse{}, err
	}
	node, err := xrayNodeRepo.Get(repo.WithByID(user.NodeID))
	if err != nil {
		return dto.XrayShareLinkResponse{}, err
	}

	// 节点域名作为连接地址，未填则提示用户手动替换
	host := nodeHost(node)
	if host == "" {
		host = "YOUR_SERVER_DOMAIN_OR_IP"
	}
	link := buildShareLink(node, user, host)
	return dto.XrayShareLinkResponse{Link: link}, nil
}

// ============================================================
// 配置生成 & reload
// ============================================================

func (s *XrayService) reloadConfig() error {
	nodes, err := xrayNodeRepo.GetList()
	if err != nil {
		return err
	}
	users, err := xrayUserRepo.GetList()
	if err != nil {
		return err
	}
	cfg := buildXrayConfig(nodes, users)
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(xrayConfigPath, data, 0644); err != nil {
		return err
	}
	return reloadXray()
}

func reloadXray() error {
	cmd := exec.Command("systemctl", "reload", "xray")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		restartCmd := exec.Command("systemctl", "restart", "xray")
		if rerr := restartCmd.Run(); rerr != nil {
			return fmt.Errorf("reload xray failed: %s; restart also failed: %v", stderr.String(), rerr)
		}
	}
	return nil
}

// buildXrayConfig 从数据库生成完整 Xray config.json
func buildXrayConfig(nodes []model.XrayNode, users []model.XrayUser) map[string]interface{} {
	usersByNode := map[uint][]model.XrayUser{}
	for _, u := range users {
		if u.Enabled {
			usersByNode[u.NodeID] = append(usersByNode[u.NodeID], u)
		}
	}

	inbounds := []map[string]interface{}{
		// Stats API dokodemo-door（固定 127.0.0.1:10085）
		{
			"listen":   "127.0.0.1",
			"port":     xrayAPIPort,
			"protocol": "dokodemo-door",
			"settings": map[string]interface{}{"address": "127.0.0.1"},
			"tag":      "api",
		},
	}

	for _, node := range nodes {
		if !node.Enabled {
			continue
		}
		if ib := buildInbound(node, usersByNode[node.ID]); ib != nil {
			inbounds = append(inbounds, ib)
		}
	}

	return map[string]interface{}{
		"log": map[string]interface{}{
			"loglevel": "warning",
			"access":   "/data/xray/log/access.log",
			"error":    "/data/xray/log/error.log",
		},
		"stats": map[string]interface{}{},
		"api": map[string]interface{}{
			"tag":      "api",
			"services": []string{"StatsService"},
		},
		"policy": map[string]interface{}{
			"levels": map[string]interface{}{"0": map[string]interface{}{}},
			"system": map[string]interface{}{
				"statsUserUplink":   true,
				"statsUserDownlink": true,
			},
		},
		"routing": map[string]interface{}{
			"rules": []map[string]interface{}{
				{"inboundTag": []string{"api"}, "outboundTag": "api", "type": "field"},
			},
		},
		"inbounds": inbounds,
		"outbounds": []map[string]interface{}{
			{"protocol": "freedom", "tag": "direct"},
			{"protocol": "blackhole", "tag": "blocked"},
		},
	}
}

func buildInbound(node model.XrayNode, users []model.XrayUser) map[string]interface{} {
	settings := buildProtocolSettings(node, users)
	streamSettings := buildStreamSettings(node)
	sniffing := buildSniffing(node)

	return map[string]interface{}{
		"listen":         node.ListenAddr,
		"port":           node.Port,
		"protocol":       node.Protocol,
		"settings":       settings,
		"streamSettings": streamSettings,
		"sniffing":       sniffing,
		"tag":            fmt.Sprintf("inbound-%d", node.ID),
	}
}

func buildProtocolSettings(node model.XrayNode, users []model.XrayUser) map[string]interface{} {
	var clients []map[string]interface{}
	for _, u := range users {
		flow := u.Flow
		if flow == "" {
			flow = node.Flow
		}
		client := map[string]interface{}{
			"id":    u.UUID,
			"email": u.Email,
			"level": u.Level,
		}
		if flow != "" {
			client["flow"] = flow
		}
		clients = append(clients, client)
	}

	// 解析 fallbacks
	var fallbacks []dto.XrayFallback
	if node.Fallbacks != "" && node.Fallbacks != "[]" {
		_ = json.Unmarshal([]byte(node.Fallbacks), &fallbacks)
	}

	switch node.Protocol {
	case "vless":
		settings := map[string]interface{}{
			"clients":    clients,
			"decryption": "none",
		}
		if len(fallbacks) > 0 {
			var fbList []map[string]interface{}
			for _, fb := range fallbacks {
				item := map[string]interface{}{"dest": fb.Dest}
				if fb.Path != "" {
					item["path"] = fb.Path
				}
				if fb.ALPN != "" {
					item["alpn"] = fb.ALPN
				}
				fbList = append(fbList, item)
			}
			settings["fallbacks"] = fbList
		}
		return settings

	case "vmess":
		for i := range clients {
			clients[i]["alterId"] = 0
		}
		return map[string]interface{}{"clients": clients}

	case "trojan":
		// Trojan 客户端使用 password 字段
		var trojanClients []map[string]interface{}
		for _, u := range users {
			client := map[string]interface{}{
				"password": u.UUID, // 存 UUID 字段但用作 password
				"email":    u.Email,
				"level":    u.Level,
			}
			trojanClients = append(trojanClients, client)
		}
		settings := map[string]interface{}{"clients": trojanClients}
		if len(fallbacks) > 0 {
			var fbList []map[string]interface{}
			for _, fb := range fallbacks {
				item := map[string]interface{}{"dest": fb.Dest}
				if fb.Path != "" {
					item["path"] = fb.Path
				}
				if fb.ALPN != "" {
					item["alpn"] = fb.ALPN
				}
				fbList = append(fbList, item)
			}
			settings["fallbacks"] = fbList
		}
		return settings

	case "shadowsocks":
		method := node.SSMethod
		if method == "" {
			method = "aes-256-gcm"
		}
		password := node.SSPassword
		if password == "" && len(users) > 0 {
			password = users[0].UUID
		}
		return map[string]interface{}{
			"method":   method,
			"password": password,
		}
	}
	return map[string]interface{}{"clients": clients}
}

func buildStreamSettings(node model.XrayNode) map[string]interface{} {
	ss := map[string]interface{}{
		"network":  node.Network,
		"security": node.Security,
	}

	// 传输方式配置
	netKey := node.Network + "Settings"
	if node.Network == "raw" {
		netKey = "rawSettings"
	}
	if node.NetworkSettings != "" && node.NetworkSettings != "{}" {
		var raw interface{}
		if err := json.Unmarshal([]byte(node.NetworkSettings), &raw); err == nil {
			ss[netKey] = raw
		}
	}

	// 安全配置
	switch node.Security {
	case "tls":
		if node.SecuritySettings != "" && node.SecuritySettings != "{}" {
			var tls dto.XrayTLSSettings
			if err := json.Unmarshal([]byte(node.SecuritySettings), &tls); err == nil {
				tlsCfg := map[string]interface{}{}
				if tls.ServerName != "" {
					tlsCfg["serverName"] = tls.ServerName
				}
				if tls.CertFile != "" && tls.KeyFile != "" {
					tlsCfg["certificates"] = []map[string]interface{}{
						{"certificateFile": tls.CertFile, "keyFile": tls.KeyFile},
					}
				}
				if len(tls.ALPN) > 0 {
					tlsCfg["alpn"] = tls.ALPN
				}
				if tls.Fingerprint != "" {
					tlsCfg["fingerprint"] = tls.Fingerprint
				}
				if tls.MinVersion != "" {
					tlsCfg["minVersion"] = tls.MinVersion
				}
				if tls.RejectUnknownSni {
					tlsCfg["rejectUnknownSni"] = true
				}
				ss["tlsSettings"] = tlsCfg
			}
		}
	case "reality":
		if node.SecuritySettings != "" && node.SecuritySettings != "{}" {
			var r dto.XrayRealitySettings
			if err := json.Unmarshal([]byte(node.SecuritySettings), &r); err == nil {
				realityCfg := map[string]interface{}{
					"show":        r.Show,
					"dest":        r.Dest,
					"xver":        r.Xver,
					"serverNames": r.ServerNames,
					"privateKey":  r.PrivateKey,
					"shortIds":    r.ShortIds,
				}
				if r.Fingerprint != "" {
					realityCfg["fingerprint"] = r.Fingerprint
				}
				if r.SpiderX != "" {
					realityCfg["spiderX"] = r.SpiderX
				}
				ss["realitySettings"] = realityCfg
			}
		}
	}

	return ss
}

func buildSniffing(node model.XrayNode) map[string]interface{} {
	var destOverride []string
	if node.SniffDestOverride != "" {
		_ = json.Unmarshal([]byte(node.SniffDestOverride), &destOverride)
	}
	if len(destOverride) == 0 {
		destOverride = []string{"http", "tls"}
	}
	return map[string]interface{}{
		"enabled":      node.SniffEnabled,
		"destOverride": destOverride,
		"metadataOnly": node.SniffMetadataOnly,
	}
}

// ============================================================
// 流量同步 & 过期检查
// ============================================================

func (s *XrayService) SyncTraffic() {
	if !s.syncMu.TryLock() {
		return
	}
	defer s.syncMu.Unlock()

	out, err := exec.Command(xrayBin, "api", "statsquery",
		"--server="+xrayAPIAddr, "-reset", "true").Output()
	if err != nil {
		global.LOG.Debugf("xray stats query failed: %v", err)
		return
	}

	var result struct {
		Stat []struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		} `json:"stat"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return
	}

	uploads := map[string]int64{}
	downloads := map[string]int64{}
	for _, stat := range result.Stat {
		parts := strings.Split(stat.Name, ">>>")
		if len(parts) != 4 || parts[0] != "user" || parts[2] != "traffic" {
			continue
		}
		email := parts[1]
		var val int64
		fmt.Sscanf(stat.Value, "%d", &val)
		switch parts[3] {
		case "uplink":
			uploads[email] += val
		case "downlink":
			downloads[email] += val
		}
	}

	allEmails := map[string]bool{}
	for e := range uploads {
		allEmails[e] = true
	}
	for e := range downloads {
		allEmails[e] = true
	}
	for email := range allEmails {
		up := uploads[email]
		dl := downloads[email]
		if up == 0 && dl == 0 {
			continue
		}
		global.DB.Exec(
			"UPDATE xray_users SET upload_total = upload_total + ?, download_total = download_total + ? WHERE email = ?",
			up, dl, email,
		)
	}
}

func (s *XrayService) CheckExpiredUsers() {
	now := time.Now()
	var count int64
	global.DB.Model(&model.XrayUser{}).
		Where("expire_at IS NOT NULL AND expire_at < ? AND enabled = ?", now, true).
		Count(&count)
	if count == 0 {
		return
	}
	global.LOG.Infof("xray: disabling %d expired users", count)
	global.DB.Model(&model.XrayUser{}).
		Where("expire_at IS NOT NULL AND expire_at < ? AND enabled = ?", now, true).
		Update("enabled", false)
	s.mu.Lock()
	defer s.mu.Unlock()
	_ = s.reloadConfig()
}

func (s *XrayService) SnapshotDailyTraffic() {
	today := time.Now().Format("2006-01-02")
	var users []model.XrayUser
	if err := global.DB.Find(&users).Error; err != nil {
		return
	}
	for _, u := range users {
		global.DB.Exec(`
			INSERT INTO xray_traffic_dailies (user_id, date, upload, download, created_at, updated_at)
			VALUES (?, ?, ?, ?, datetime('now'), datetime('now'))
			ON CONFLICT(user_id, date) DO UPDATE SET
				upload = excluded.upload,
				download = excluded.download,
				updated_at = datetime('now')
		`, u.ID, today, u.UploadTotal, u.DownloadTotal)
	}
}

func (s *XrayService) GetTrafficHistory(userID uint, days int) ([]dto.XrayTrafficDaily, error) {
	if days <= 0 || days > 90 {
		days = 30
	}
	var rows []model.XrayTrafficDaily
	err := global.DB.Where("user_id = ?", userID).
		Order("date DESC").Limit(days).Find(&rows).Error
	if err != nil {
		return nil, err
	}
	result := make([]dto.XrayTrafficDaily, len(rows))
	for i, r := range rows {
		up, dl := r.Upload, r.Download
		if i+1 < len(rows) {
			prev := rows[i+1]
			if r.Upload >= prev.Upload {
				up = r.Upload - prev.Upload
			}
			if r.Download >= prev.Download {
				dl = r.Download - prev.Download
			}
		}
		result[i] = dto.XrayTrafficDaily{Date: r.Date, Upload: up, Download: dl}
	}
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}
	return result, nil
}

// ============================================================
// 辅助函数
// ============================================================

// marshalNodeSettings 序列化网络/安全子配置为 JSON 字符串
func marshalNodeSettings(
	network, security string,
	raw *dto.XrayRawSettings,
	ws *dto.XrayWSSettings,
	grpc *dto.XrayGRPCSettings,
	xhttp *dto.XrayXHTTPSettings,
	httpUpgrade *dto.XrayHTTPUpgradeSettings,
	tls *dto.XrayTLSSettings,
	reality *dto.XrayRealitySettings,
) (netJSON, secJSON string, err error) {
	// 网络配置
	var netObj interface{}
	switch network {
	case "raw", "tcp":
		if raw != nil {
			netObj = raw
		}
	case "ws":
		if ws != nil {
			netObj = ws
		}
	case "grpc":
		if grpc != nil {
			netObj = grpc
		}
	case "xhttp":
		if xhttp != nil {
			netObj = xhttp
		}
	case "httpupgrade":
		if httpUpgrade != nil {
			netObj = httpUpgrade
		}
	}
	if netObj != nil {
		b, e := json.Marshal(netObj)
		if e != nil {
			return "", "", e
		}
		netJSON = string(b)
	} else {
		netJSON = "{}"
	}

	// 安全配置
	var secObj interface{}
	switch security {
	case "tls":
		if tls != nil {
			secObj = tls
		}
	case "reality":
		if reality != nil {
			secObj = reality
		}
	}
	if secObj != nil {
		b, e := json.Marshal(secObj)
		if e != nil {
			return "", "", e
		}
		secJSON = string(b)
	} else {
		secJSON = "{}"
	}
	return netJSON, secJSON, nil
}

func nodeToResponse(n model.XrayNode) dto.XrayNodeResponse {
	resp := dto.XrayNodeResponse{
		ID:                n.ID,
		Name:              n.Name,
		Protocol:          n.Protocol,
		ListenAddr:        n.ListenAddr,
		Port:              n.Port,
		Network:           n.Network,
		Security:          n.Security,
		Flow:              n.Flow,
		SSMethod:          n.SSMethod,
		SSPassword:        n.SSPassword,
		SniffEnabled:      n.SniffEnabled,
		SniffMetadataOnly: n.SniffMetadataOnly,
		Remark:            n.Remark,
		Enabled:           n.Enabled,
		CreatedAt:         n.CreatedAt,
	}
	// sniffDestOverride
	if n.SniffDestOverride != "" {
		_ = json.Unmarshal([]byte(n.SniffDestOverride), &resp.SniffDestOverride)
	}
	// fallbacks
	if n.Fallbacks != "" && n.Fallbacks != "[]" {
		_ = json.Unmarshal([]byte(n.Fallbacks), &resp.Fallbacks)
	}
	if resp.Fallbacks == nil {
		resp.Fallbacks = []dto.XrayFallback{}
	}
	// 网络子配置
	if n.NetworkSettings != "" && n.NetworkSettings != "{}" {
		switch n.Network {
		case "raw", "tcp":
			var v dto.XrayRawSettings
			if json.Unmarshal([]byte(n.NetworkSettings), &v) == nil { resp.RawSettings = &v }
		case "ws":
			var v dto.XrayWSSettings
			if json.Unmarshal([]byte(n.NetworkSettings), &v) == nil { resp.WSSettings = &v }
		case "grpc":
			var v dto.XrayGRPCSettings
			if json.Unmarshal([]byte(n.NetworkSettings), &v) == nil { resp.GRPCSettings = &v }
		case "xhttp":
			var v dto.XrayXHTTPSettings
			if json.Unmarshal([]byte(n.NetworkSettings), &v) == nil { resp.XHTTPSettings = &v }
		case "httpupgrade":
			var v dto.XrayHTTPUpgradeSettings
			if json.Unmarshal([]byte(n.NetworkSettings), &v) == nil { resp.HTTPUpgradeSettings = &v }
		}
	}
	// 安全子配置
	if n.SecuritySettings != "" && n.SecuritySettings != "{}" {
		switch n.Security {
		case "tls":
			var v dto.XrayTLSSettings
			if json.Unmarshal([]byte(n.SecuritySettings), &v) == nil { resp.TLSSettings = &v }
		case "reality":
			var v dto.XrayRealitySettings
			if json.Unmarshal([]byte(n.SecuritySettings), &v) == nil { resp.RealitySettings = &v }
		}
	}
	return resp
}

// nodeHost 从节点提取连接域名（从 TLS ServerName 或 Reality ServerNames 中取）
func nodeHost(n model.XrayNode) string {
	switch n.Security {
	case "tls":
		var tls dto.XrayTLSSettings
		if json.Unmarshal([]byte(n.SecuritySettings), &tls) == nil && tls.ServerName != "" {
			return tls.ServerName
		}
	case "reality":
		var r dto.XrayRealitySettings
		if json.Unmarshal([]byte(n.SecuritySettings), &r) == nil && len(r.ServerNames) > 0 {
			return r.ServerNames[0]
		}
	}
	return ""
}

// buildShareLink 生成 VLESS/VMess/Trojan 分享 URI
func buildShareLink(node model.XrayNode, user model.XrayUser, host string) string {
	userFlow := user.Flow
	if userFlow == "" {
		userFlow = node.Flow
	}

	switch node.Protocol {
	case "vless":
		params := url.Values{}
		params.Set("type", node.Network)
		params.Set("security", node.Security)
		// flow
		if userFlow != "" {
			params.Set("flow", userFlow)
		}
		// 传输参数
		switch node.Network {
		case "ws":
			var ws dto.XrayWSSettings
			if json.Unmarshal([]byte(node.NetworkSettings), &ws) == nil {
				if ws.Path != "" {
					params.Set("path", ws.Path)
				}
				if ws.Host != "" {
					params.Set("host", ws.Host)
				}
			}
		case "grpc":
			var g dto.XrayGRPCSettings
			if json.Unmarshal([]byte(node.NetworkSettings), &g) == nil {
				params.Set("serviceName", g.ServiceName)
				params.Set("mode", "gun")
			}
		case "xhttp":
			var x dto.XrayXHTTPSettings
			if json.Unmarshal([]byte(node.NetworkSettings), &x) == nil {
				if x.Path != "" {
					params.Set("path", x.Path)
				}
				if x.Host != "" {
					params.Set("host", x.Host)
				}
				if x.Mode != "" {
					params.Set("mode", x.Mode)
				}
			}
		case "httpupgrade":
			var h dto.XrayHTTPUpgradeSettings
			if json.Unmarshal([]byte(node.NetworkSettings), &h) == nil {
				if h.Path != "" {
					params.Set("path", h.Path)
				}
				if h.Host != "" {
					params.Set("host", h.Host)
				}
			}
		}
		// 安全参数
		switch node.Security {
		case "reality":
			var r dto.XrayRealitySettings
			if json.Unmarshal([]byte(node.SecuritySettings), &r) == nil {
				params.Set("pbk", r.PublicKey)
				if r.Fingerprint != "" {
					params.Set("fp", r.Fingerprint)
				} else {
					params.Set("fp", "chrome")
				}
				if len(r.ShortIds) > 0 {
					params.Set("sid", r.ShortIds[0])
				}
				if len(r.ServerNames) > 0 {
					params.Set("sni", r.ServerNames[0])
				}
			}
		case "tls":
			var t dto.XrayTLSSettings
			if json.Unmarshal([]byte(node.SecuritySettings), &t) == nil {
				if t.ServerName != "" {
					params.Set("sni", t.ServerName)
				}
				if t.Fingerprint != "" {
					params.Set("fp", t.Fingerprint)
				}
				if len(t.ALPN) > 0 {
					params.Set("alpn", strings.Join(t.ALPN, ","))
				}
			}
		}
		name := url.PathEscape(user.Name)
		return fmt.Sprintf("vless://%s@%s:%d?%s#%s", user.UUID, host, node.Port, params.Encode(), name)

	case "vmess":
		v := map[string]interface{}{
			"v":    "2",
			"ps":   user.Name,
			"add":  host,
			"port": fmt.Sprintf("%d", node.Port),
			"id":   user.UUID,
			"aid":  "0",
			"scy":  "auto",
			"net":  node.Network,
			"tls":  node.Security,
		}
		if node.Network == "ws" {
			var ws dto.XrayWSSettings
			if json.Unmarshal([]byte(node.NetworkSettings), &ws) == nil {
				v["path"] = ws.Path
				v["host"] = ws.Host
			}
		}
		if node.Security == "tls" {
			var t dto.XrayTLSSettings
			if json.Unmarshal([]byte(node.SecuritySettings), &t) == nil {
				v["sni"] = t.ServerName
			}
		}
		data, _ := json.Marshal(v)
		return "vmess://" + b64.StdEncoding.EncodeToString(data)

	case "trojan":
		params := url.Values{}
		params.Set("type", node.Network)
		if node.Security == "tls" {
			var t dto.XrayTLSSettings
			if json.Unmarshal([]byte(node.SecuritySettings), &t) == nil {
				if t.ServerName != "" {
					params.Set("sni", t.ServerName)
				}
			}
		}
		name := url.PathEscape(user.Name)
		return fmt.Sprintf("trojan://%s@%s:%d?%s#%s", user.UUID, host, node.Port, params.Encode(), name)
	}
	return ""
}

func encodeBase64(s string) string {
	return b64.StdEncoding.EncodeToString([]byte(s))
}
