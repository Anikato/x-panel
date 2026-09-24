package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
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

type gostRuntime interface {
	Ping() bool
	CreateService(gostutil.ServiceConfig) error
	UpdateService(name string, cfg gostutil.ServiceConfig) error
	DeleteService(name string) error
	SaveConfig() error
}

var newGostRuntime = func() gostRuntime { return newGostClient() }

type gostChainRuntime interface {
	Ping() bool
	CreateChain(gostutil.ChainConfig) error
	UpdateChain(name string, chain gostutil.ChainConfig) error
	DeleteChain(name string) error
	SaveConfig() error
}

var newGostChainRuntime = func() gostChainRuntime { return newGostClient() }

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
	if err := s.pushServiceToGost(svc); err != nil {
		_ = s.serviceRepo.Delete(repo.WithByID(svc.ID))
		return err
	}
	return nil
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
	if err := s.syncUpdatedGostService(oldName, existing, updated); err != nil {
		_ = s.serviceRepo.Update(req.ID, gostServiceSnapshot(existing))
		_ = s.pushServiceToGost(existing)
		return err
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
	client := newGostRuntime()
	restore := func() error {
		item := existing
		return s.serviceRepo.Create(&item)
	}
	if !client.Ping() {
		return combineRuntimeError(fmt.Errorf("gost api unreachable"), restore())
	}
	if err := s.deleteServiceFromGost(client, existing.Name, existing.Type); err != nil {
		_ = s.restoreGostServices(client, existing, existing.Name)
		return combineRuntimeError(err, restore())
	}
	if err := client.SaveConfig(); err != nil {
		_ = s.restoreGostServices(client, existing, existing.Name)
		return combineRuntimeError(err, restore())
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

	updated := existing
	updated.Enabled = req.Enabled
	if err := s.syncUpdatedGostService(existing.Name, existing, updated); err != nil {
		_ = s.serviceRepo.Update(req.ID, map[string]interface{}{"enabled": existing.Enabled})
		return err
	}
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
			Hops:     sanitizeGostHops(item.Hops),
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
	if err := s.pushChainToGost(chain); err != nil {
		_ = s.chainRepo.Delete(repo.WithByID(chain.ID))
		return err
	}
	return nil
}

func (s *GostService) UpdateChain(req dto.GostChainUpdate) error {
	existing, err := s.chainRepo.Get(repo.WithByID(req.ID))
	if err != nil {
		return buserr.New(constant.ErrRecordNotFound)
	}
	oldName := existing.Name
	mergedHops, err := mergeGostHopSecrets(existing.Hops, req.Hops)
	if err != nil {
		return buserr.WithDetail(constant.ErrInvalidParams, err.Error(), err)
	}

	if err := s.chainRepo.Update(req.ID, map[string]interface{}{
		"name":   req.Name,
		"hops":   mergedHops,
		"remark": req.Remark,
	}); err != nil {
		return err
	}
	restoreRecord := func() error {
		return s.chainRepo.Update(existing.ID, map[string]interface{}{
			"name":   existing.Name,
			"hops":   existing.Hops,
			"remark": existing.Remark,
		})
	}

	client := newGostChainRuntime()
	if !client.Ping() {
		return combineRuntimeError(fmt.Errorf("gost api unreachable"), restoreRecord())
	}
	updated, err := s.chainRepo.Get(repo.WithByID(req.ID))
	if err != nil {
		return combineRuntimeError(err, restoreRecord())
	}
	cfg := s.buildChainConfig(updated)
	oldCfg := s.buildChainConfig(existing)
	if oldName != req.Name {
		if err := client.DeleteChain(oldName); err != nil {
			return combineRuntimeError(err, restoreRecord())
		}
		if err := client.CreateChain(cfg); err != nil {
			_ = client.CreateChain(oldCfg)
			_ = client.SaveConfig()
			return combineRuntimeError(err, restoreRecord())
		}
	} else if err := client.UpdateChain(updated.Name, cfg); err != nil {
		_ = client.UpdateChain(oldName, oldCfg)
		_ = client.SaveConfig()
		return combineRuntimeError(err, restoreRecord())
	}
	if err := client.SaveConfig(); err != nil {
		if oldName != req.Name {
			_ = client.DeleteChain(req.Name)
			_ = client.CreateChain(oldCfg)
		} else {
			_ = client.UpdateChain(oldName, oldCfg)
		}
		_ = client.SaveConfig()
		return combineRuntimeError(err, restoreRecord())
	}
	return nil
}

func sanitizeGostHops(raw string) string {
	var value any
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return ""
	}
	sanitizeJSONSecrets(value)
	data, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return string(data)
}

func mergeGostHopSecrets(existingRaw, updateRaw string) (string, error) {
	var existing any
	if err := json.Unmarshal([]byte(existingRaw), &existing); err != nil {
		return "", fmt.Errorf("decode existing GOST hops: %w", err)
	}
	var update any
	if err := json.Unmarshal([]byte(updateRaw), &update); err != nil {
		return "", fmt.Errorf("decode updated GOST hops: %w", err)
	}
	if err := mergeJSONSecrets(existing, update); err != nil {
		return "", err
	}
	data, err := json.Marshal(update)
	if err != nil {
		return "", fmt.Errorf("encode updated GOST hops: %w", err)
	}
	return string(data), nil
}

func sanitizeJSONSecrets(value any) {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			if isJSONSecretKey(key) {
				typed[key] = ""
				continue
			}
			sanitizeJSONSecrets(child)
		}
	case []any:
		for _, child := range typed {
			sanitizeJSONSecrets(child)
		}
	}
}

func mergeJSONSecrets(existing, update any) error {
	switch updated := update.(type) {
	case map[string]any:
		original, _ := existing.(map[string]any)
		for key, updatedChild := range updated {
			originalChild := original[key]
			if isJSONSecretKey(key) {
				if text, ok := updatedChild.(string); ok && text == "" && originalChild != nil {
					updated[key] = originalChild
				}
				continue
			}
			if err := mergeJSONSecrets(originalChild, updatedChild); err != nil {
				return err
			}
		}
	case []any:
		original, _ := existing.([]any)
		if len(original) != len(updated) && containsBlankJSONSecret(updated) {
			return errors.New("GOST chain structure changed; re-enter credentials for affected nodes")
		}
		for index, updatedChild := range updated {
			if index < len(original) {
				if !equalNonSecretJSON(original[index], updatedChild) &&
					containsBlankJSONSecret(updatedChild) {
					return errors.New("GOST chain node order or identity changed; re-enter credentials for affected nodes")
				}
				if err := mergeJSONSecrets(original[index], updatedChild); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func containsBlankJSONSecret(value any) bool {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			if isJSONSecretKey(key) {
				if text, ok := child.(string); ok && text == "" {
					return true
				}
				continue
			}
			if containsBlankJSONSecret(child) {
				return true
			}
		}
	case []any:
		for _, child := range typed {
			if containsBlankJSONSecret(child) {
				return true
			}
		}
	}
	return false
}

func equalNonSecretJSON(left, right any) bool {
	switch leftValue := left.(type) {
	case map[string]any:
		rightValue, ok := right.(map[string]any)
		if !ok {
			return false
		}
		for key, leftChild := range leftValue {
			if isJSONSecretKey(key) {
				continue
			}
			rightChild, exists := rightValue[key]
			if !exists || !equalNonSecretJSON(leftChild, rightChild) {
				return false
			}
		}
		for key := range rightValue {
			if isJSONSecretKey(key) {
				continue
			}
			if _, exists := leftValue[key]; !exists {
				return false
			}
		}
		return true
	case []any:
		rightValue, ok := right.([]any)
		if !ok || len(leftValue) != len(rightValue) {
			return false
		}
		for index := range leftValue {
			if !equalNonSecretJSON(leftValue[index], rightValue[index]) {
				return false
			}
		}
		return true
	default:
		return fmt.Sprint(left) == fmt.Sprint(right)
	}
}

func isJSONSecretKey(key string) bool {
	normalized := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(key, "_", ""), "-", ""))
	switch normalized {
	case "password", "passphrase", "token", "secret", "credential",
		"authorization", "privatekey", "accesskey", "authpass":
		return true
	default:
		return false
	}
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
	client := newGostChainRuntime()
	restore := func() error {
		item := existing
		return s.chainRepo.Create(&item)
	}
	if !client.Ping() {
		return combineRuntimeError(fmt.Errorf("gost api unreachable"), restore())
	}
	if err := client.DeleteChain(existing.Name); err != nil {
		return combineRuntimeError(err, restore())
	}
	if err := client.SaveConfig(); err != nil {
		_ = client.CreateChain(s.buildChainConfig(existing))
		_ = client.SaveConfig()
		return combineRuntimeError(err, restore())
	}
	return nil
}

// --- Sync ---

func (s *GostService) SyncAll() error {
	client := newGostRuntime()
	if !client.Ping() {
		return fmt.Errorf("GOST API not reachable")
	}
	chainClient := newGostChainRuntime()

	chains, err := s.chainRepo.GetList()
	if err != nil {
		return err
	}
	var syncErr error
	for _, chain := range chains {
		cfg := s.buildChainConfig(chain)
		if err := chainClient.CreateChain(cfg); err != nil {
			if uerr := chainClient.UpdateChain(chain.Name, cfg); uerr != nil {
				syncErr = errors.Join(syncErr, fmt.Errorf("chain %s: %w", chain.Name, uerr))
			}
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
				if uerr := client.UpdateService(cfg.Name, cfg); uerr != nil {
					syncErr = errors.Join(syncErr, fmt.Errorf("service %s: %w", cfg.Name, uerr))
				}
			}
		}
	}

	if err := chainClient.SaveConfig(); err != nil {
		syncErr = errors.Join(syncErr, err)
	}
	if err := client.SaveConfig(); err != nil {
		syncErr = errors.Join(syncErr, err)
	}
	if syncErr != nil {
		return syncErr
	}
	global.LOG.Infof("Synced %d chains and %d services to GOST", len(chains), len(services))
	return nil
}

// --- config builders ---

func (s *GostService) pushServiceToGost(svc model.GostService) error {
	client := newGostRuntime()
	if !client.Ping() {
		return fmt.Errorf("gost api unreachable")
	}
	for _, cfg := range s.buildServiceConfigs(svc) {
		if err := client.CreateService(cfg); err != nil {
			return err
		}
	}
	return client.SaveConfig()
}

func (s *GostService) syncUpdatedGostService(oldName string, previous, updated model.GostService) error {
	client := newGostRuntime()
	if !client.Ping() {
		return fmt.Errorf("gost api unreachable")
	}

	oldNames := gostRuntimeNames(oldName, previous.Type)
	newCfgs := []gostutil.ServiceConfig{}
	if updated.Enabled {
		newCfgs = s.buildServiceConfigs(updated)
	}
	created := []string{}
	for _, cfg := range newCfgs {
		if err := client.CreateService(cfg); err != nil {
			for _, name := range created {
				_ = client.DeleteService(name)
			}
			_ = s.restoreGostServices(client, previous, oldName)
			return err
		}
		created = append(created, cfg.Name)
	}

	newNameSet := map[string]struct{}{}
	for _, cfg := range newCfgs {
		newNameSet[cfg.Name] = struct{}{}
	}
	if previous.Enabled {
		for _, name := range oldNames {
			if _, keep := newNameSet[name]; keep {
				continue
			}
			_ = client.DeleteService(name)
		}
	} else if !updated.Enabled {
		for _, name := range oldNames {
			_ = client.DeleteService(name)
		}
	}

	if err := client.SaveConfig(); err != nil {
		for _, name := range created {
			_ = client.DeleteService(name)
		}
		if rerr := s.restoreGostServices(client, previous, oldName); rerr != nil {
			return fmt.Errorf("%v; restore: %w", err, rerr)
		}
		return err
	}
	return nil
}

func gostRuntimeNames(name, svcType string) []string {
	if svcType == "tcp_udp_forward" {
		return []string{name + "-tcp", name + "-udp"}
	}
	return []string{name}
}

func (s *GostService) restoreGostServices(client gostRuntime, previous model.GostService, oldName string) error {
	if !previous.Enabled {
		return client.SaveConfig()
	}
	prev := previous
	prev.Name = oldName
	for _, cfg := range s.buildServiceConfigs(prev) {
		if err := client.CreateService(cfg); err != nil {
			return err
		}
	}
	return client.SaveConfig()
}

func gostServiceSnapshot(item model.GostService) map[string]interface{} {
	return map[string]interface{}{
		"name":             item.Name,
		"type":             item.Type,
		"listen_addr":      item.ListenAddr,
		"target_addr":      item.TargetAddr,
		"listener_type":    item.ListenerType,
		"auth_user":        item.AuthUser,
		"auth_pass":        item.AuthPass,
		"chain_id":         item.ChainID,
		"certificate_id":   item.CertificateID,
		"custom_cert_path": item.CustomCertPath,
		"custom_key_path":  item.CustomKeyPath,
		"enable_stats":     item.EnableStats,
		"remark":           item.Remark,
		"enabled":          item.Enabled,
	}
}

func (s *GostService) deleteServiceFromGost(client gostRuntime, name, svcType string) error {
	var first error
	for _, runtimeName := range gostRuntimeNames(name, svcType) {
		if err := client.DeleteService(runtimeName); err != nil && first == nil {
			first = err
		}
	}
	return first
}

func (s *GostService) pushChainToGost(chain model.GostChain) error {
	client := newGostChainRuntime()
	if !client.Ping() {
		return fmt.Errorf("gost api unreachable")
	}
	cfg := s.buildChainConfig(chain)
	if err := client.CreateChain(cfg); err != nil {
		return err
	}
	return client.SaveConfig()
}

func combineRuntimeError(action, restore error) error {
	if restore == nil {
		return action
	}
	return fmt.Errorf("%v; restore: %w", action, restore)
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
		return existingCertFilePaths(sslDir, cert)
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
