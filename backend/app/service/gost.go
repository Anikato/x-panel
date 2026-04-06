package service

import (
	"encoding/json"
	"fmt"
	"net"
	"path/filepath"
	"strconv"
	"strings"

	"xpanel/app/dto"
	"xpanel/app/model"
	"xpanel/app/repo"
	"xpanel/buserr"
	"xpanel/constant"
	"xpanel/global"

	gostutil "xpanel/utils/gost"
)

type IGostService interface {
	// Service CRUD
	SearchService(req dto.GostServiceSearch) (int64, []dto.GostServiceInfo, error)
	CreateService(req dto.GostServiceCreate) error
	UpdateService(req dto.GostServiceUpdate) error
	DeleteService(id uint) error
	ToggleService(req dto.GostServiceToggle) error

	// Chain CRUD
	SearchChain(req dto.GostChainSearch) (int64, []dto.GostChainInfo, error)
	CreateChain(req dto.GostChainCreate) error
	UpdateChain(req dto.GostChainUpdate) error
	DeleteChain(id uint) error

	// Sync
	SyncAll() error
}

type GostService struct {
	serviceRepo repo.IGostServiceRepo
	chainRepo   repo.IGostChainRepo
}

func NewIGostService() IGostService {
	return &GostService{
		serviceRepo: repo.NewIGostServiceRepo(),
		chainRepo:   repo.NewIGostChainRepo(),
	}
}

// --- Service CRUD ---

func (s *GostService) SearchService(req dto.GostServiceSearch) (int64, []dto.GostServiceInfo, error) {
	var opts []repo.DBOption
	if req.Info != "" {
		opts = append(opts, repo.WithLikeName(req.Info))
	}
	if req.Type != "" {
		opts = append(opts, repo.WithByGostType(req.Type))
	}
	total, items, err := s.serviceRepo.Page(req.Page, req.PageSize, opts...)
	if err != nil {
		return 0, nil, err
	}

	statsMap := make(map[string]*gostutil.Stats)
	client := newGostClient()
	if client.Ping() {
		if sm, err := client.GetServiceStats(); err == nil {
			statsMap = sm
		}
	}

	chainMap := make(map[uint]string)
	certMap := make(map[uint]string)
	var infos []dto.GostServiceInfo
	for _, item := range items {
		info := dto.GostServiceInfo{
			ID:             item.ID,
			Name:           item.Name,
			Type:           item.Type,
			ListenAddr:     item.ListenAddr,
			TargetAddr:     item.TargetAddr,
			ListenerType:   item.ListenerType,
			AuthUser:       item.AuthUser,
			ChainID:        item.ChainID,
			CertificateID:  item.CertificateID,
			CustomCertPath: item.CustomCertPath,
			CustomKeyPath:  item.CustomKeyPath,
			EnableStats:    item.EnableStats,
			Enabled:        item.Enabled,
			Remark:         item.Remark,
		}
		if item.ChainID > 0 {
			if name, ok := chainMap[item.ChainID]; ok {
				info.ChainName = name
			} else {
				chain, err := s.chainRepo.Get(repo.WithByID(item.ChainID))
				if err == nil {
					chainMap[item.ChainID] = chain.Name
					info.ChainName = chain.Name
				}
			}
		}
		if item.CertificateID > 0 {
			if domain, ok := certMap[item.CertificateID]; ok {
				info.CertDomain = domain
			} else {
				certRepo := repo.NewICertificateRepo()
				cert, err := certRepo.Get(repo.WithByID(item.CertificateID))
				if err == nil {
					certMap[item.CertificateID] = cert.PrimaryDomain
					info.CertDomain = cert.PrimaryDomain
				}
			}
		}
		if item.Type == "tcp_udp_forward" {
			mergeStats(&info, statsMap[item.Name+"-tcp"])
			mergeStats(&info, statsMap[item.Name+"-udp"])
		} else {
			mergeStats(&info, statsMap[item.Name])
		}
		infos = append(infos, info)
	}
	return total, infos, nil
}

func mergeStats(info *dto.GostServiceInfo, stats *gostutil.Stats) {
	if stats == nil {
		return
	}
	info.TotalConns += stats.TotalConns
	info.CurrentConns += stats.CurrentConns
	info.InputBytes += stats.InputBytes
	info.OutputBytes += stats.OutputBytes
	info.TotalErrs += stats.TotalErrs
}

func (s *GostService) CreateService(req dto.GostServiceCreate) error {
	if _, err := s.serviceRepo.Get(repo.WithByName(req.Name)); err == nil {
		return buserr.New(constant.ErrGostNameExist)
	}
	if err := validateListenAddr(req.ListenAddr); err != nil {
		return err
	}
	if err := validateTargetAddr(req.TargetAddr); err != nil {
		return err
	}
	svc := model.GostService{
		Name:           req.Name,
		Type:           req.Type,
		ListenAddr:     normalizeListenAddr(req.ListenAddr),
		TargetAddr:     req.TargetAddr,
		ListenerType:   req.ListenerType,
		AuthUser:       req.AuthUser,
		AuthPass:       req.AuthPass,
		ChainID:        req.ChainID,
		CertificateID:  req.CertificateID,
		CustomCertPath: req.CustomCertPath,
		CustomKeyPath:  req.CustomKeyPath,
		EnableStats:    req.EnableStats,
		Enabled:        true,
		Remark:         req.Remark,
	}
	if err := s.serviceRepo.Create(&svc); err != nil {
		return err
	}
	return s.pushServiceToGost(svc)
}

func normalizeListenAddr(addr string) string {
	if addr == "" {
		return addr
	}
	if addr[0] != ':' && !strings.Contains(addr, ":") {
		return ":" + addr
	}
	return addr
}

func validateListenAddr(addr string) error {
	normalized := normalizeListenAddr(addr)
	_, portStr, err := net.SplitHostPort(normalized)
	if err != nil {
		return fmt.Errorf("监听地址格式不正确，应为 :端口 或 IP:端口")
	}
	port, err := strconv.Atoi(portStr)
	if err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("端口号必须在 1-65535 之间")
	}
	return nil
}

func validateTargetAddr(addr string) error {
	if addr == "" {
		return nil
	}
	_, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("目标地址格式不正确，应为 IP:端口")
	}
	port, err := strconv.Atoi(portStr)
	if err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("目标端口号必须在 1-65535 之间")
	}
	return nil
}

func (s *GostService) UpdateService(req dto.GostServiceUpdate) error {
	existing, err := s.serviceRepo.Get(repo.WithByID(req.ID))
	if err != nil {
		return buserr.New(constant.ErrRecordNotFound)
	}
	oldName := existing.Name

	if err := validateListenAddr(req.ListenAddr); err != nil {
		return err
	}
	if err := validateTargetAddr(req.TargetAddr); err != nil {
		return err
	}

	updates := map[string]interface{}{
		"name":             req.Name,
		"type":             req.Type,
		"listen_addr":      normalizeListenAddr(req.ListenAddr),
		"target_addr":      req.TargetAddr,
		"listener_type":    req.ListenerType,
		"auth_user":        req.AuthUser,
		"chain_id":         req.ChainID,
		"certificate_id":   req.CertificateID,
		"custom_cert_path": req.CustomCertPath,
		"custom_key_path":  req.CustomKeyPath,
		"enable_stats":     req.EnableStats,
		"remark":           req.Remark,
	}
	if req.AuthPass != "" {
		updates["auth_pass"] = req.AuthPass
	}
	if err := s.serviceRepo.Update(req.ID, updates); err != nil {
		return err
	}

	updated, _ := s.serviceRepo.Get(repo.WithByID(req.ID))
	client := newGostClient()
	if !client.Ping() {
		return nil
	}
	if updated.Enabled {
		s.deleteServiceFromGost(client, oldName, existing.Type)
		for _, cfg := range s.buildServiceConfigs(updated) {
			client.CreateService(cfg)
		}
		client.SaveConfig()
	}
	return nil
}

func (s *GostService) DeleteService(id uint) error {
	existing, err := s.serviceRepo.Get(repo.WithByID(id))
	if err != nil {
		return buserr.New(constant.ErrRecordNotFound)
	}
	if err := s.serviceRepo.Delete(repo.WithByID(id)); err != nil {
		return err
	}

	client := newGostClient()
	if client.Ping() {
		s.deleteServiceFromGost(client, existing.Name, existing.Type)
		client.SaveConfig()
	}
	return nil
}

func (s *GostService) ToggleService(req dto.GostServiceToggle) error {
	existing, err := s.serviceRepo.Get(repo.WithByID(req.ID))
	if err != nil {
		return buserr.New(constant.ErrRecordNotFound)
	}
	if err := s.serviceRepo.Update(req.ID, map[string]interface{}{"enabled": req.Enabled}); err != nil {
		return err
	}

	client := newGostClient()
	if !client.Ping() {
		return nil
	}
	if req.Enabled {
		existing.Enabled = true
		for _, cfg := range s.buildServiceConfigs(existing) {
			client.CreateService(cfg)
		}
	} else {
		s.deleteServiceFromGost(client, existing.Name, existing.Type)
	}
	client.SaveConfig()
	return nil
}

// --- Chain CRUD ---

func (s *GostService) SearchChain(req dto.GostChainSearch) (int64, []dto.GostChainInfo, error) {
	var opts []repo.DBOption
	if req.Info != "" {
		opts = append(opts, repo.WithLikeName(req.Info))
	}
	total, items, err := s.chainRepo.Page(req.Page, req.PageSize, opts...)
	if err != nil {
		return 0, nil, err
	}

	var infos []dto.GostChainInfo
	for _, item := range items {
		refCount, _ := s.serviceRepo.CountByChainID(item.ID)
		hopCount := 0
		var hops []interface{}
		if json.Unmarshal([]byte(item.Hops), &hops) == nil {
			hopCount = len(hops)
		}
		infos = append(infos, dto.GostChainInfo{
			ID:       item.ID,
			Name:     item.Name,
			Hops:     item.Hops,
			HopCount: hopCount,
			RefCount: refCount,
			Remark:   item.Remark,
		})
	}
	return total, infos, nil
}

func (s *GostService) CreateChain(req dto.GostChainCreate) error {
	chain := model.GostChain{
		Name:   req.Name,
		Hops:   req.Hops,
		Remark: req.Remark,
	}
	if err := s.chainRepo.Create(&chain); err != nil {
		return err
	}
	return s.pushChainToGost(chain)
}

func (s *GostService) UpdateChain(req dto.GostChainUpdate) error {
	existing, err := s.chainRepo.Get(repo.WithByID(req.ID))
	if err != nil {
		return buserr.New(constant.ErrRecordNotFound)
	}
	oldName := existing.Name

	if err := s.chainRepo.Update(req.ID, map[string]interface{}{
		"name":   req.Name,
		"hops":   req.Hops,
		"remark": req.Remark,
	}); err != nil {
		return err
	}

	client := newGostClient()
	if !client.Ping() {
		return nil
	}
	updated, _ := s.chainRepo.Get(repo.WithByID(req.ID))
	cfg := s.buildChainConfig(updated)
	if oldName != req.Name {
		client.DeleteChain(oldName)
		client.CreateChain(cfg)
	} else {
		client.UpdateChain(updated.Name, cfg)
	}
	client.SaveConfig()
	return nil
}

func (s *GostService) DeleteChain(id uint) error {
	refCount, _ := s.serviceRepo.CountByChainID(id)
	if refCount > 0 {
		return fmt.Errorf("chain is referenced by %d service(s), cannot delete", refCount)
	}
	existing, err := s.chainRepo.Get(repo.WithByID(id))
	if err != nil {
		return buserr.New(constant.ErrRecordNotFound)
	}
	if err := s.chainRepo.Delete(repo.WithByID(id)); err != nil {
		return err
	}

	client := newGostClient()
	if client.Ping() {
		client.DeleteChain(existing.Name)
		client.SaveConfig()
	}
	return nil
}

// --- Sync ---

func (s *GostService) SyncAll() error {
	client := newGostClient()
	if !client.Ping() {
		global.LOG.Warn("GOST API not reachable, skip sync")
		return nil
	}

	chains, err := s.chainRepo.GetList()
	if err != nil {
		return err
	}
	for _, chain := range chains {
		cfg := s.buildChainConfig(chain)
		if err := client.CreateChain(cfg); err != nil {
			client.UpdateChain(chain.Name, cfg)
		}
	}

	services, err := s.serviceRepo.GetList()
	if err != nil {
		return err
	}
	for _, svc := range services {
		if !svc.Enabled {
			continue
		}
		for _, cfg := range s.buildServiceConfigs(svc) {
			if err := client.CreateService(cfg); err != nil {
				client.UpdateService(cfg.Name, cfg)
			}
		}
	}

	client.SaveConfig()
	global.LOG.Infof("Synced %d chains and %d services to GOST", len(chains), len(services))
	return nil
}

// --- config builders ---

func (s *GostService) pushServiceToGost(svc model.GostService) error {
	client := newGostClient()
	if !client.Ping() {
		return nil
	}
	for _, cfg := range s.buildServiceConfigs(svc) {
		if err := client.CreateService(cfg); err != nil {
			return err
		}
	}
	return client.SaveConfig()
}

func (s *GostService) deleteServiceFromGost(client *gostutil.Client, name, svcType string) {
	if svcType == "tcp_udp_forward" {
		client.DeleteService(name + "-tcp")
		client.DeleteService(name + "-udp")
	} else {
		client.DeleteService(name)
	}
}

func (s *GostService) pushChainToGost(chain model.GostChain) error {
	client := newGostClient()
	if !client.Ping() {
		return nil
	}
	cfg := s.buildChainConfig(chain)
	if err := client.CreateChain(cfg); err != nil {
		return err
	}
	return client.SaveConfig()
}

func (s *GostService) buildServiceConfigs(svc model.GostService) []gostutil.ServiceConfig {
	if svc.Type == "tcp_udp_forward" {
		tcpSvc := svc
		tcpSvc.Type = "tcp_forward"
		tcpSvc.Name = svc.Name + "-tcp"
		udpSvc := svc
		udpSvc.Type = "udp_forward"
		udpSvc.Name = svc.Name + "-udp"
		return []gostutil.ServiceConfig{
			s.buildSingleServiceConfig(tcpSvc),
			s.buildSingleServiceConfig(udpSvc),
		}
	}
	return []gostutil.ServiceConfig{s.buildSingleServiceConfig(svc)}
}

func (s *GostService) buildSingleServiceConfig(svc model.GostService) gostutil.ServiceConfig {
	cfg := gostutil.ServiceConfig{
		Name: svc.Name,
		Addr: svc.ListenAddr,
	}

	if svc.EnableStats {
		cfg.Metadata = map[string]string{"enableStats": "true"}
	}

	switch svc.Type {
	case "tcp_forward", "udp_forward":
		proto := "tcp"
		if svc.Type == "udp_forward" {
			proto = "udp"
		}
		cfg.Handler = gostutil.HandlerConfig{Type: proto}
		cfg.Listener = gostutil.ListenerConfig{Type: proto}
		if svc.TargetAddr != "" {
			cfg.Forwarder = &gostutil.ForwarderConfig{
				Nodes: []gostutil.ForwarderNode{{Name: "target-0", Addr: svc.TargetAddr}},
			}
		}
		if svc.ChainID > 0 {
			chain, err := s.chainRepo.Get(repo.WithByID(svc.ChainID))
			if err == nil {
				cfg.Handler.Chain = chain.Name
			}
		}

	case "relay_server":
		cfg.Handler = gostutil.HandlerConfig{
			Type: "relay",
			Metadata: map[string]string{
				"bind": "true",
				"udp":  "true",
			},
		}
		if svc.AuthUser != "" {
			cfg.Handler.Auth = &gostutil.AuthConfig{
				Username: svc.AuthUser,
				Password: svc.AuthPass,
			}
		}
		cfg.Listener = gostutil.ListenerConfig{Type: svc.ListenerType}
	}

	if svc.ListenerType == "tls" || svc.ListenerType == "wss" {
		certFile, keyFile := s.resolveServiceCert(svc)
		if certFile != "" && keyFile != "" {
			cfg.Listener.TLS = &gostutil.TLSConfig{
				CertFile: certFile,
				KeyFile:  keyFile,
			}
		}
	}

	return cfg
}

func (s *GostService) resolveServiceCert(svc model.GostService) (certFile, keyFile string) {
	if svc.CustomCertPath != "" && svc.CustomKeyPath != "" {
		return svc.CustomCertPath, svc.CustomKeyPath
	}
	if svc.CertificateID > 0 {
		certRepo := repo.NewICertificateRepo()
		cert, err := certRepo.Get(repo.WithByID(svc.CertificateID))
		if err != nil {
			global.LOG.Warnf("Failed to load certificate %d for GOST service %s: %v", svc.CertificateID, svc.Name, err)
			return "", ""
		}
		sslSvc := NewICertificateService()
		sslDir := sslSvc.GetSSLDir()
		return filepath.Join(sslDir, "certs", cert.PrimaryDomain, "fullchain.pem"),
			filepath.Join(sslDir, "certs", cert.PrimaryDomain, "privkey.pem")
	}
	return "", ""
}

func (s *GostService) buildChainConfig(chain model.GostChain) gostutil.ChainConfig {
	cfg := gostutil.ChainConfig{Name: chain.Name}
	var hops []gostutil.HopItem
	if err := json.Unmarshal([]byte(chain.Hops), &hops); err != nil {
		global.LOG.Warnf("Failed to parse chain hops for %s: %v", chain.Name, err)
		return cfg
	}
	cfg.Hops = hops
	return cfg
}
