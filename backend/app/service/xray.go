package service

import (
	b64 "encoding/base64"
	"bytes"
	"encoding/json"
	"fmt"
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
	// 内部：定时流量同步 & 过期检查
	SyncTraffic()
	CheckExpiredUsers()
}

func NewIXrayService() IXrayService { return &XrayService{} }

type XrayService struct {
	mu     sync.Mutex // 保护 config 写入和 reload
	syncMu sync.Mutex // 保护流量同步，防止并发
}

// 安装状态（进程级全局）
var (
	installLog     strings.Builder
	installRunning bool
	installMu      sync.Mutex
)

var (
	xrayNodeRepo = repo.NewIXrayNodeRepo()
	xrayUserRepo = repo.NewIXrayUserRepo()
)

// ==================== 状态 & 安装 ====================

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
		// 安装完成后写入初始配置
		if err := s.initConfig(); err != nil {
			global.LOG.Warnf("xray: init config after install: %v", err)
		}
	}()
	return nil
}

func (s *XrayService) GetInstallLog() string {
	installMu.Lock()
	defer installMu.Unlock()
	return installLog.String()
}

// initConfig 写入包含 Stats API 的基础配置（首次安装时调用）
func (s *XrayService) initConfig() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.reloadConfig()
}

func (s *XrayService) GetStatus() dto.XrayStatusResponse {
	resp := dto.XrayStatusResponse{
		Installed:  s.IsInstalled(),
		ConfigPath: xrayConfigPath,
		BinPath:    xrayBin,
	}
	cmd := exec.Command("systemctl", "is-active", "xray")
	out, _ := cmd.Output()
	resp.Running = strings.TrimSpace(string(out)) == "active"

	verCmd := exec.Command(xrayBin, "version")
	verOut, _ := verCmd.Output()
	if len(verOut) > 0 {
		lines := strings.Split(string(verOut), "\n")
		if len(lines) > 0 {
			resp.Version = strings.TrimSpace(lines[0])
		}
	}
	return resp
}

// ==================== 节点管理 ====================

func (s *XrayService) ListNodes() ([]dto.XrayNodeResponse, error) {
	nodes, err := xrayNodeRepo.GetList()
	if err != nil {
		return nil, err
	}
	var result []dto.XrayNodeResponse
	for _, n := range nodes {
		count, _ := xrayUserRepo.Count(repo.WithXrayNodeID(n.ID))
		result = append(result, dto.XrayNodeResponse{
			ID:                 n.ID,
			Name:               n.Name,
			Protocol:           n.Protocol,
			Port:               n.Port,
			Transport:          n.Transport,
			Security:           n.Security,
			Domain:             n.Domain,
			RealityPublicKey:   n.RealityPublicKey,
			RealityShortIds:    n.RealityShortIds,
			RealityServerNames: n.RealityServerNames,
			Path:               n.Path,
			ServiceName:        n.ServiceName,
			Remark:             n.Remark,
			Enabled:            n.Enabled,
			UserCount:          count,
			CreatedAt:          n.CreatedAt,
		})
	}
	return result, nil
}

func (s *XrayService) CreateNode(req dto.XrayNodeCreate) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	node := &model.XrayNode{
		Name:               req.Name,
		Protocol:           req.Protocol,
		Port:               req.Port,
		Transport:          req.Transport,
		Security:           req.Security,
		Domain:             req.Domain,
		TLSCert:            req.TLSCert,
		TLSKey:             req.TLSKey,
		RealityPrivateKey:  req.RealityPrivateKey,
		RealityPublicKey:   req.RealityPublicKey,
		RealityShortIds:    req.RealityShortIds,
		RealityServerNames: req.RealityServerNames,
		Path:               req.Path,
		ServiceName:        req.ServiceName,
		Remark:             req.Remark,
		Enabled:            true,
	}
	if node.Path == "" {
		node.Path = "/"
	}
	if err := xrayNodeRepo.Create(node); err != nil {
		return err
	}
	return s.reloadConfig()
}

func (s *XrayService) UpdateNode(req dto.XrayNodeUpdate) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	node, err := xrayNodeRepo.Get(repo.WithByID(req.ID))
	if err != nil {
		return err
	}
	node.Name = req.Name
	node.Transport = req.Transport
	node.Security = req.Security
	node.Domain = req.Domain
	node.TLSCert = req.TLSCert
	node.TLSKey = req.TLSKey
	node.RealityPrivateKey = req.RealityPrivateKey
	node.RealityPublicKey = req.RealityPublicKey
	node.RealityShortIds = req.RealityShortIds
	node.RealityServerNames = req.RealityServerNames
	node.Path = req.Path
	node.ServiceName = req.ServiceName
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

// ==================== 用户管理 ====================

func (s *XrayService) SearchUsers(req dto.XrayUserSearch) (int64, []dto.XrayUserResponse, error) {
	opts := []repo.DBOption{}
	if req.NodeID > 0 {
		opts = append(opts, repo.WithXrayNodeID(req.NodeID))
	}
	total, users, err := xrayUserRepo.Page(req.Page, req.PageSize, opts...)
	if err != nil {
		return 0, nil, err
	}

	// 批量查节点名称
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
	email := fmt.Sprintf("%s@xpanel", strings.ReplaceAll(uid, "-", "")[:8])

	user := &model.XrayUser{
		NodeID:   req.NodeID,
		Name:     req.Name,
		UUID:     uid,
		Email:    email,
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

// ==================== 工具 ====================

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

	// 使用节点配置的域名，若未配置则提示管理员手动填写
	host := node.Domain
	if host == "" {
		host = "YOUR_SERVER_ADDRESS"
	}
	link := buildShareLink(node, user, host)
	return dto.XrayShareLinkResponse{Link: link}, nil
}

// ==================== 配置生成 ====================

// reloadConfig 重新生成 config.json 并 reload xray（调用前需持有锁）
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

// reloadXray 通知 xray 重载配置
func reloadXray() error {
	cmd := exec.Command("systemctl", "reload", "xray")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		// reload 失败尝试 restart
		restartCmd := exec.Command("systemctl", "restart", "xray")
		if rerr := restartCmd.Run(); rerr != nil {
			return fmt.Errorf("reload xray failed: %s, restart also failed: %v", stderr.String(), rerr)
		}
	}
	return nil
}

// buildXrayConfig 从节点和用户列表构建完整的 Xray 配置
func buildXrayConfig(nodes []model.XrayNode, users []model.XrayUser) map[string]interface{} {
	// 按节点 ID 分组用户
	usersByNode := map[uint][]model.XrayUser{}
	for _, u := range users {
		if u.Enabled {
			usersByNode[u.NodeID] = append(usersByNode[u.NodeID], u)
		}
	}

	// 构建入站列表
	inbounds := []map[string]interface{}{
		// Stats API inbound
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
		inbound := buildInbound(node, usersByNode[node.ID])
		if inbound != nil {
			inbounds = append(inbounds, inbound)
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
			"levels": map[string]interface{}{
				"0": map[string]interface{}{},
			},
			"system": map[string]interface{}{
				"statsUserUplink":   true,
				"statsUserDownlink": true,
			},
		},
		"routing": map[string]interface{}{
			"rules": []map[string]interface{}{
				{
					"inboundTag":  []string{"api"},
					"outboundTag": "api",
					"type":        "field",
				},
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
	// 构建 clients
	var clients []map[string]interface{}
	for _, u := range users {
		client := map[string]interface{}{
			"id":    u.UUID,
			"email": u.Email,
			"level": u.Level,
		}
		if node.Protocol == "vmess" {
			client["alterId"] = 0
		}
		clients = append(clients, client)
	}

	// 构建 settings
	settings := map[string]interface{}{}
	switch node.Protocol {
	case "vless":
		settings["clients"] = clients
		settings["decryption"] = "none"
	case "vmess":
		settings["clients"] = clients
	case "trojan":
		settings["clients"] = clients
	}

	// 构建 streamSettings
	streamSettings := buildStreamSettings(node)

	inbound := map[string]interface{}{
		"listen":         "0.0.0.0",
		"port":           node.Port,
		"protocol":       node.Protocol,
		"settings":       settings,
		"streamSettings": streamSettings,
		"tag":            fmt.Sprintf("inbound-%d", node.ID),
		"sniffing": map[string]interface{}{
			"enabled":      true,
			"destOverride": []string{"http", "tls"},
		},
	}
	return inbound
}

func buildStreamSettings(node model.XrayNode) map[string]interface{} {
	ss := map[string]interface{}{}

	switch node.Transport {
	case "tcp":
		ss["network"] = "tcp"
	case "ws":
		ss["network"] = "ws"
		ss["wsSettings"] = map[string]interface{}{
			"path": node.Path,
		}
	case "grpc":
		ss["network"] = "grpc"
		ss["grpcSettings"] = map[string]interface{}{
			"serviceName": node.ServiceName,
		}
	}

	switch node.Security {
	case "tls":
		ss["security"] = "tls"
		tlsSettings := map[string]interface{}{}
		if node.Domain != "" {
			tlsSettings["serverName"] = node.Domain
		}
		if node.TLSCert != "" && node.TLSKey != "" {
			tlsSettings["certificates"] = []map[string]interface{}{
				{
					"certificateFile": node.TLSCert,
					"keyFile":         node.TLSKey,
				},
			}
		}
		ss["tlsSettings"] = tlsSettings
	case "reality":
		ss["security"] = "reality"
		var shortIds []string
		if node.RealityShortIds != "" {
			_ = json.Unmarshal([]byte(node.RealityShortIds), &shortIds)
		}
		var serverNames []string
		if node.RealityServerNames != "" {
			_ = json.Unmarshal([]byte(node.RealityServerNames), &serverNames)
		}
		dest := "www.apple.com:443"
		if len(serverNames) > 0 {
			dest = serverNames[0] + ":443"
		}
		ss["realitySettings"] = map[string]interface{}{
			"show":        false,
			"dest":        dest,
			"xver":        0,
			"serverNames": serverNames,
			"privateKey":  node.RealityPrivateKey,
			"shortIds":    shortIds,
		}
	default:
		ss["security"] = "none"
	}

	return ss
}

// ==================== 流量同步 ====================

// SyncTraffic 从 Xray Stats API 拉取流量数据并累加到 DB
func (s *XrayService) SyncTraffic() {
	// 防止并发执行（例如定时任务积压时）
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

	// 汇总每个 email 的流量
	uploads := map[string]int64{}
	downloads := map[string]int64{}
	for _, stat := range result.Stat {
		// name 格式: user>>>email>>>traffic>>>uplink / downlink
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

	// 汇总所有 email 的集合
	allEmails := map[string]bool{}
	for e := range uploads {
		allEmails[e] = true
	}
	for e := range downloads {
		allEmails[e] = true
	}

	// 更新数据库（使用原生 SQL 原子累加）
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

// CheckExpiredUsers 检查并禁用已到期用户
func (s *XrayService) CheckExpiredUsers() {
	now := time.Now()
	var expiredUsers []model.XrayUser
	global.DB.Model(&model.XrayUser{}).
		Where("expire_at IS NOT NULL AND expire_at < ? AND enabled = ?", now, true).
		Find(&expiredUsers)

	if len(expiredUsers) == 0 {
		return
	}

	global.LOG.Infof("Found %d expired xray users, disabling...", len(expiredUsers))
	global.DB.Model(&model.XrayUser{}).
		Where("expire_at IS NOT NULL AND expire_at < ? AND enabled = ?", now, true).
		Update("enabled", false)

	s.mu.Lock()
	defer s.mu.Unlock()
	_ = s.reloadConfig()
}

// SnapshotDailyTraffic 每日零点快照当前累计流量（供历史图表使用）
func (s *XrayService) SnapshotDailyTraffic() {
	today := time.Now().Format("2006-01-02")
	var users []model.XrayUser
	if err := global.DB.Find(&users).Error; err != nil {
		return
	}
	for _, u := range users {
		// upsert：同一天只保留一条
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

// GetTrafficHistory 返回某用户最近 N 天的每日流量增量
func (s *XrayService) GetTrafficHistory(userID uint, days int) ([]dto.XrayTrafficDaily, error) {
	if days <= 0 || days > 90 {
		days = 30
	}
	var rows []model.XrayTrafficDaily
	err := global.DB.Where("user_id = ?", userID).
		Order("date DESC").
		Limit(days).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}

	// 转换为增量（每天相比前一天的新增流量）
	result := make([]dto.XrayTrafficDaily, len(rows))
	for i, r := range rows {
		upload, download := r.Upload, r.Download
		if i+1 < len(rows) {
			prev := rows[i+1]
			if r.Upload >= prev.Upload {
				upload = r.Upload - prev.Upload
			}
			if r.Download >= prev.Download {
				download = r.Download - prev.Download
			}
		}
		result[i] = dto.XrayTrafficDaily{
			Date:     r.Date,
			Upload:   upload,
			Download: download,
		}
	}
	// 按日期正序返回
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}
	return result, nil
}

// ==================== 工具函数 ====================

func buildShareLink(node model.XrayNode, user model.XrayUser, serverIP string) string {
	host := serverIP
	if node.Domain != "" {
		host = node.Domain
	}

	switch node.Protocol {
	case "vless":
		params := fmt.Sprintf("type=%s&security=%s", node.Transport, node.Security)
		if node.Transport == "ws" {
			params += fmt.Sprintf("&path=%s", node.Path)
		}
		if node.Transport == "grpc" {
			params += fmt.Sprintf("&serviceName=%s&mode=gun", node.ServiceName)
		}
		if node.Security == "reality" {
			params += fmt.Sprintf("&pbk=%s&fp=chrome", node.RealityPublicKey)
			var sids []string
			if node.RealityShortIds != "" {
				_ = json.Unmarshal([]byte(node.RealityShortIds), &sids)
			}
			if len(sids) > 0 {
				params += fmt.Sprintf("&sid=%s", sids[0])
			}
			var sns []string
			if node.RealityServerNames != "" {
				_ = json.Unmarshal([]byte(node.RealityServerNames), &sns)
			}
			if len(sns) > 0 {
				params += fmt.Sprintf("&sni=%s", sns[0])
			}
		}
		if node.Security == "tls" && node.Domain != "" {
			params += fmt.Sprintf("&sni=%s", node.Domain)
		}
		return fmt.Sprintf("vless://%s@%s:%d?%s#%s",
			user.UUID, host, node.Port, params, user.Name)

	case "vmess":
		v := map[string]interface{}{
			"v":    "2",
			"ps":   user.Name,
			"add":  host,
			"port": fmt.Sprintf("%d", node.Port),
			"id":   user.UUID,
			"aid":  "0",
			"net":  node.Transport,
			"type": "none",
			"tls":  node.Security,
		}
		if node.Transport == "ws" {
			v["path"] = node.Path
		}
		data, _ := json.Marshal(v)
		return "vmess://" + encodeBase64(string(data))

	case "trojan":
		params := fmt.Sprintf("type=%s", node.Transport)
		if node.Security == "tls" && node.Domain != "" {
			params += fmt.Sprintf("&sni=%s", node.Domain)
		}
		return fmt.Sprintf("trojan://%s@%s:%d?%s#%s",
			user.UUID, host, node.Port, params, user.Name)
	}
	return ""
}

func encodeBase64(s string) string {
	return b64.StdEncoding.EncodeToString([]byte(s))
}
