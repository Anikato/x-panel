package service

import (
	"fmt"
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
	"xpanel/utils/cmd"
	haproxyutil "xpanel/utils/haproxy"
)

type IHAProxyService interface {
	// LB
	SearchLB(req dto.HAProxyLBSearch) (int64, []dto.HAProxyLBInfo, error)
	ListLB() ([]dto.HAProxyLBInfo, error)
	CreateLB(req dto.HAProxyLBCreate, operator string) error
	UpdateLB(req dto.HAProxyLBUpdate, operator string) error
	DeleteLB(id uint, operator string) error
	ToggleLB(req dto.HAProxyLBToggle, operator string) error

	// Backend
	SearchBackend(req dto.HAProxyBackendSearch) (int64, []dto.HAProxyBackendInfo, error)
	ListBackend() ([]dto.HAProxyBackendInfo, error)
	GetBackend(id uint) (*dto.HAProxyBackendInfo, error)
	CreateBackend(req dto.HAProxyBackendCreate, operator string) error
	UpdateBackend(req dto.HAProxyBackendUpdate, operator string) error
	DeleteBackend(id uint, operator string) error

	// Server
	CreateServer(req dto.HAProxyServerCreate, operator string) error
	UpdateServer(req dto.HAProxyServerUpdate, operator string) error
	DeleteServer(id uint, operator string) error

	// ACL
	ListACL(lbID uint) ([]dto.HAProxyACLInfo, error)
	CreateACL(req dto.HAProxyACLCreate, operator string) error
	UpdateACL(req dto.HAProxyACLUpdate, operator string) error
	DeleteACL(id uint, operator string) error

	// 配置/版本
	GetRawConfig() (string, error)
	SaveRawConfig(content, operator string) error
	TestConfig(content string) (*dto.HAProxyConfigTestResp, error)
	PreviewConfig() (string, error)
	RebuildConfig(operator string) error
	ListConfigVersions(limit int) ([]dto.HAProxyConfigVersionInfo, error)
	GetConfigVersion(id uint) (string, error)
	RollbackToVersion(id uint, operator string) error

	// ApplyChange 供内部调用
	ApplyChange(reason, operator string) error
}

type HAProxyService struct{}

func NewIHAProxyService() IHAProxyService { return &HAProxyService{} }

// --- LB ---

func (s *HAProxyService) SearchLB(req dto.HAProxyLBSearch) (int64, []dto.HAProxyLBInfo, error) {
	total, items, err := repo.NewIHAProxyLBRepo().Page(req.Page, req.PageSize, repo.WithLikeName(req.Info))
	if err != nil {
		return 0, nil, err
	}
	backends, _ := repo.NewIHAProxyBackendRepo().GetList()
	beNameMap := make(map[uint]string)
	for _, b := range backends {
		beNameMap[b.ID] = b.Name
	}
	certNameMap := s.buildCertNameMap()

	// mode 过滤
	filtered := make([]model.HAProxyLB, 0, len(items))
	for _, it := range items {
		if req.Mode != "" && it.Mode != req.Mode {
			continue
		}
		filtered = append(filtered, it)
	}

	infos := make([]dto.HAProxyLBInfo, 0, len(filtered))
	for _, it := range filtered {
		infos = append(infos, s.lbToInfo(it, beNameMap, certNameMap))
	}
	// 由于 mode 过滤后 total 不准，此处修正（简单重新计数）
	if req.Mode != "" {
		all, _ := repo.NewIHAProxyLBRepo().GetList(repo.WithLikeName(req.Info))
		total = 0
		for _, it := range all {
			if it.Mode == req.Mode {
				total++
			}
		}
	}
	return total, infos, nil
}

func (s *HAProxyService) ListLB() ([]dto.HAProxyLBInfo, error) {
	items, err := repo.NewIHAProxyLBRepo().GetList()
	if err != nil {
		return nil, err
	}
	backends, _ := repo.NewIHAProxyBackendRepo().GetList()
	beNameMap := make(map[uint]string)
	for _, b := range backends {
		beNameMap[b.ID] = b.Name
	}
	certNameMap := s.buildCertNameMap()
	infos := make([]dto.HAProxyLBInfo, 0, len(items))
	for _, it := range items {
		infos = append(infos, s.lbToInfo(it, beNameMap, certNameMap))
	}
	return infos, nil
}

func (s *HAProxyService) lbToInfo(lb model.HAProxyLB, beNameMap map[uint]string, certNameMap map[uint]string) dto.HAProxyLBInfo {
	info := dto.HAProxyLBInfo{
		ID: lb.ID, Name: lb.Name, Mode: lb.Mode, Enabled: lb.Enabled,
		BindAddr: lb.BindAddr, BindPort: lb.BindPort, EnableSSL: lb.EnableSSL,
		CertificateID: lb.CertificateID, SSLRedirect: lb.SSLRedirect,
		DefaultBackendID: lb.DefaultBackendID, XForwardedFor: lb.XForwardedFor,
		MaxConn: lb.MaxConn, TimeoutConnect: lb.TimeoutConnect,
		TimeoutClient: lb.TimeoutClient, TimeoutServer: lb.TimeoutServer,
		Remark: lb.Remark,
	}
	if n, ok := beNameMap[lb.DefaultBackendID]; ok {
		info.DefaultBackend = n
	}
	if n, ok := certNameMap[lb.CertificateID]; ok {
		info.CertDomain = n
	}
	return info
}

func (s *HAProxyService) buildCertNameMap() map[uint]string {
	certRepo := repo.NewICertificateRepo()
	certs, _ := certRepo.GetList()
	m := make(map[uint]string, len(certs))
	for _, c := range certs {
		m[c.ID] = c.PrimaryDomain
	}
	return m
}

func (s *HAProxyService) CreateLB(req dto.HAProxyLBCreate, operator string) error {
	if err := s.validateLB(&req, 0); err != nil {
		return err
	}
	item := model.HAProxyLB{
		Name: req.Name, Mode: req.Mode, Enabled: true,
		BindAddr: defaultBindAddr(req.BindAddr), BindPort: req.BindPort,
		EnableSSL: req.EnableSSL && req.Mode == "http",
		CertificateID:    req.CertificateID,
		SSLRedirect:      req.SSLRedirect,
		DefaultBackendID: req.DefaultBackendID,
		XForwardedFor:    req.XForwardedFor,
		MaxConn:          req.MaxConn,
		TimeoutConnect:   withDefault(req.TimeoutConnect, 5),
		TimeoutClient:    withDefault(req.TimeoutClient, 30),
		TimeoutServer:    withDefault(req.TimeoutServer, 30),
		Remark:           req.Remark,
	}
	if err := repo.NewIHAProxyLBRepo().Create(&item); err != nil {
		return err
	}
	return s.ApplyChange(fmt.Sprintf("创建 LB: %s", item.Name), operator)
}

func (s *HAProxyService) UpdateLB(req dto.HAProxyLBUpdate, operator string) error {
	old, err := repo.NewIHAProxyLBRepo().Get(repo.WithByID(req.ID))
	if err != nil {
		return buserr.New(constant.ErrRecordNotFound)
	}
	if err := s.validateLB(&req.HAProxyLBCreate, req.ID); err != nil {
		return err
	}
	updates := map[string]interface{}{
		"name": req.Name, "mode": req.Mode,
		"bind_addr": defaultBindAddr(req.BindAddr), "bind_port": req.BindPort,
		"enable_ssl":         req.EnableSSL && req.Mode == "http",
		"certificate_id":     req.CertificateID,
		"ssl_redirect":       req.SSLRedirect,
		"default_backend_id": req.DefaultBackendID,
		"x_forwarded_for":    req.XForwardedFor,
		"max_conn":           req.MaxConn,
		"timeout_connect":    withDefault(req.TimeoutConnect, 5),
		"timeout_client":     withDefault(req.TimeoutClient, 30),
		"timeout_server":     withDefault(req.TimeoutServer, 30),
		"remark":             req.Remark,
	}
	if err := repo.NewIHAProxyLBRepo().Update(req.ID, updates); err != nil {
		return err
	}
	return s.ApplyChange(fmt.Sprintf("更新 LB: %s", old.Name), operator)
}

func (s *HAProxyService) DeleteLB(id uint, operator string) error {
	old, err := repo.NewIHAProxyLBRepo().Get(repo.WithByID(id))
	if err != nil {
		return buserr.New(constant.ErrRecordNotFound)
	}
	// 删除所属 ACL
	acls, _ := repo.NewIHAProxyACLRepo().GetListByLB(id)
	for _, a := range acls {
		_ = repo.NewIHAProxyACLRepo().Delete(repo.WithByID(a.ID))
	}
	if err := repo.NewIHAProxyLBRepo().Delete(repo.WithByID(id)); err != nil {
		return err
	}
	return s.ApplyChange(fmt.Sprintf("删除 LB: %s", old.Name), operator)
}

func (s *HAProxyService) ToggleLB(req dto.HAProxyLBToggle, operator string) error {
	old, err := repo.NewIHAProxyLBRepo().Get(repo.WithByID(req.ID))
	if err != nil {
		return buserr.New(constant.ErrRecordNotFound)
	}
	if err := repo.NewIHAProxyLBRepo().Update(req.ID, map[string]interface{}{"enabled": req.Enabled}); err != nil {
		return err
	}
	state := "启用"
	if !req.Enabled {
		state = "禁用"
	}
	return s.ApplyChange(fmt.Sprintf("%s LB: %s", state, old.Name), operator)
}

func (s *HAProxyService) validateLB(req *dto.HAProxyLBCreate, excludeID uint) error {
	if req.Name == "" {
		return buserr.New(constant.ErrInvalidParams)
	}
	if !isValidHAProxyName(req.Name) {
		return buserr.WithDetail(constant.ErrInvalidParams, "name must match [a-zA-Z0-9_.-]", nil)
	}
	existing, _ := repo.NewIHAProxyLBRepo().GetList(repo.WithByName(req.Name))
	for _, e := range existing {
		if e.ID != excludeID {
			return buserr.New(constant.ErrHAProxyNameExist)
		}
	}
	cnt, _ := repo.NewIHAProxyLBRepo().CountByBindPort(req.BindPort, excludeID)
	if cnt > 0 {
		return buserr.WithName(constant.ErrHAProxyPortInUse, fmt.Sprintf("%d", req.BindPort))
	}
	if req.Mode == "http" && req.EnableSSL && req.CertificateID == 0 {
		return buserr.WithDetail(constant.ErrInvalidParams, "enableSSL requires certificateID", nil)
	}
	return nil
}

// --- Backend ---

func (s *HAProxyService) SearchBackend(req dto.HAProxyBackendSearch) (int64, []dto.HAProxyBackendInfo, error) {
	total, items, err := repo.NewIHAProxyBackendRepo().Page(req.Page, req.PageSize, repo.WithLikeName(req.Info))
	if err != nil {
		return 0, nil, err
	}
	filtered := make([]model.HAProxyBackend, 0, len(items))
	for _, it := range items {
		if req.Mode != "" && it.Mode != req.Mode {
			continue
		}
		filtered = append(filtered, it)
	}
	infos := make([]dto.HAProxyBackendInfo, 0, len(filtered))
	for _, it := range filtered {
		infos = append(infos, s.backendToInfo(it, false))
	}
	if req.Mode != "" {
		all, _ := repo.NewIHAProxyBackendRepo().GetList(repo.WithLikeName(req.Info))
		total = 0
		for _, it := range all {
			if it.Mode == req.Mode {
				total++
			}
		}
	}
	return total, infos, nil
}

func (s *HAProxyService) ListBackend() ([]dto.HAProxyBackendInfo, error) {
	items, err := repo.NewIHAProxyBackendRepo().GetList()
	if err != nil {
		return nil, err
	}
	infos := make([]dto.HAProxyBackendInfo, 0, len(items))
	for _, it := range items {
		infos = append(infos, s.backendToInfo(it, false))
	}
	return infos, nil
}

func (s *HAProxyService) GetBackend(id uint) (*dto.HAProxyBackendInfo, error) {
	item, err := repo.NewIHAProxyBackendRepo().Get(repo.WithByID(id))
	if err != nil {
		return nil, buserr.New(constant.ErrRecordNotFound)
	}
	info := s.backendToInfo(item, true)
	return &info, nil
}

func (s *HAProxyService) backendToInfo(be model.HAProxyBackend, includeServers bool) dto.HAProxyBackendInfo {
	servers, _ := repo.NewIHAProxyServerRepo().GetListByBackend(be.ID)
	aclRefCount, _ := repo.NewIHAProxyACLRepo().CountByBackendID(be.ID)
	defRef, _ := repo.CountHAProxyLBByDefaultBackend(be.ID)

	info := dto.HAProxyBackendInfo{
		ID: be.ID, Name: be.Name, Mode: be.Mode, Balance: be.Balance,
		StickyType: be.StickyType, StickyName: be.StickyName,
		HealthType: be.HealthType, HealthPath: be.HealthPath,
		HealthMethod: be.HealthMethod, HealthHost: be.HealthHost,
		HealthExpect: be.HealthExpect, HealthInter: be.HealthInter,
		HealthRise: be.HealthRise, HealthFall: be.HealthFall,
		Remark:      be.Remark,
		ServerCount: len(servers),
		RefCount:    aclRefCount + defRef,
	}
	if includeServers {
		for _, srv := range servers {
			info.Servers = append(info.Servers, dto.HAProxyServerInfo{
				ID: srv.ID, BackendID: srv.BackendID, Name: srv.Name,
				Address: srv.Address, Port: srv.Port, Weight: srv.Weight,
				MaxConn: srv.MaxConn, Backup: srv.Backup, Disabled: srv.Disabled,
				SSL: srv.SSL, SSLVerify: srv.SSLVerify,
			})
		}
	}
	return info
}

func (s *HAProxyService) CreateBackend(req dto.HAProxyBackendCreate, operator string) error {
	if err := s.validateBackendName(req.Name, 0); err != nil {
		return err
	}
	be := model.HAProxyBackend{
		Name: req.Name, Mode: req.Mode, Balance: req.Balance,
		StickyType: req.StickyType, StickyName: req.StickyName,
		HealthType: defaultStr(req.HealthType, "tcp"),
		HealthPath: req.HealthPath, HealthMethod: req.HealthMethod,
		HealthHost: req.HealthHost, HealthExpect: req.HealthExpect,
		HealthInter: withDefault(req.HealthInter, 2000),
		HealthRise:  withDefault(req.HealthRise, 2),
		HealthFall:  withDefault(req.HealthFall, 3),
		Remark:      req.Remark,
	}
	if err := repo.NewIHAProxyBackendRepo().Create(&be); err != nil {
		return err
	}
	for _, srv := range req.Servers {
		if err := s.validateServer(&srv); err != nil {
			continue
		}
		s := model.HAProxyServer{
			BackendID: be.ID, Name: srv.Name,
			Address: srv.Address, Port: srv.Port,
			Weight: withDefault(srv.Weight, 100), MaxConn: srv.MaxConn,
			Backup: srv.Backup, Disabled: srv.Disabled,
			SSL: srv.SSL, SSLVerify: srv.SSLVerify,
		}
		_ = repo.NewIHAProxyServerRepo().Create(&s)
	}
	return s.ApplyChange(fmt.Sprintf("创建 Backend: %s", be.Name), operator)
}

func (s *HAProxyService) UpdateBackend(req dto.HAProxyBackendUpdate, operator string) error {
	old, err := repo.NewIHAProxyBackendRepo().Get(repo.WithByID(req.ID))
	if err != nil {
		return buserr.New(constant.ErrRecordNotFound)
	}
	if err := s.validateBackendName(req.Name, req.ID); err != nil {
		return err
	}
	updates := map[string]interface{}{
		"name": req.Name, "mode": req.Mode, "balance": req.Balance,
		"sticky_type": req.StickyType, "sticky_name": req.StickyName,
		"health_type": defaultStr(req.HealthType, "tcp"),
		"health_path": req.HealthPath, "health_method": req.HealthMethod,
		"health_host": req.HealthHost, "health_expect": req.HealthExpect,
		"health_inter": withDefault(req.HealthInter, 2000),
		"health_rise":  withDefault(req.HealthRise, 2),
		"health_fall":  withDefault(req.HealthFall, 3),
		"remark":       req.Remark,
	}
	if err := repo.NewIHAProxyBackendRepo().Update(req.ID, updates); err != nil {
		return err
	}
	return s.ApplyChange(fmt.Sprintf("更新 Backend: %s", old.Name), operator)
}

func (s *HAProxyService) DeleteBackend(id uint, operator string) error {
	old, err := repo.NewIHAProxyBackendRepo().Get(repo.WithByID(id))
	if err != nil {
		return buserr.New(constant.ErrRecordNotFound)
	}
	aclCnt, _ := repo.NewIHAProxyACLRepo().CountByBackendID(id)
	defRef, _ := repo.CountHAProxyLBByDefaultBackend(id)
	if aclCnt+defRef > 0 {
		return buserr.New(constant.ErrHAProxyBackendHasRefs)
	}
	servers, _ := repo.NewIHAProxyServerRepo().GetListByBackend(id)
	for _, srv := range servers {
		_ = repo.NewIHAProxyServerRepo().Delete(repo.WithByID(srv.ID))
	}
	if err := repo.NewIHAProxyBackendRepo().Delete(repo.WithByID(id)); err != nil {
		return err
	}
	return s.ApplyChange(fmt.Sprintf("删除 Backend: %s", old.Name), operator)
}

func (s *HAProxyService) validateBackendName(name string, excludeID uint) error {
	if !isValidHAProxyName(name) {
		return buserr.WithDetail(constant.ErrInvalidParams, "name must match [a-zA-Z0-9_.-]", nil)
	}
	existing, _ := repo.NewIHAProxyBackendRepo().GetList(repo.WithByName(name))
	for _, e := range existing {
		if e.ID != excludeID {
			return buserr.New(constant.ErrHAProxyNameExist)
		}
	}
	return nil
}

// --- Server ---

func (s *HAProxyService) CreateServer(req dto.HAProxyServerCreate, operator string) error {
	if req.BackendID == 0 {
		return buserr.New(constant.ErrInvalidParams)
	}
	if _, err := repo.NewIHAProxyBackendRepo().Get(repo.WithByID(req.BackendID)); err != nil {
		return buserr.New(constant.ErrRecordNotFound)
	}
	if err := s.validateServer(&req); err != nil {
		return err
	}
	srv := model.HAProxyServer{
		BackendID: req.BackendID, Name: req.Name,
		Address: req.Address, Port: req.Port,
		Weight: withDefault(req.Weight, 100), MaxConn: req.MaxConn,
		Backup: req.Backup, Disabled: req.Disabled,
		SSL: req.SSL, SSLVerify: req.SSLVerify,
	}
	if err := repo.NewIHAProxyServerRepo().Create(&srv); err != nil {
		return err
	}
	return s.ApplyChange(fmt.Sprintf("新增 Server: %s", srv.Name), operator)
}

func (s *HAProxyService) UpdateServer(req dto.HAProxyServerUpdate, operator string) error {
	old, err := repo.NewIHAProxyServerRepo().Get(repo.WithByID(req.ID))
	if err != nil {
		return buserr.New(constant.ErrRecordNotFound)
	}
	if err := s.validateServer(&req.HAProxyServerCreate); err != nil {
		return err
	}
	updates := map[string]interface{}{
		"name": req.Name, "address": req.Address, "port": req.Port,
		"weight": withDefault(req.Weight, 100), "max_conn": req.MaxConn,
		"backup": req.Backup, "disabled": req.Disabled,
		"ssl": req.SSL, "ssl_verify": req.SSLVerify,
	}
	if err := repo.NewIHAProxyServerRepo().Update(req.ID, updates); err != nil {
		return err
	}
	return s.ApplyChange(fmt.Sprintf("更新 Server: %s", old.Name), operator)
}

func (s *HAProxyService) DeleteServer(id uint, operator string) error {
	old, err := repo.NewIHAProxyServerRepo().Get(repo.WithByID(id))
	if err != nil {
		return buserr.New(constant.ErrRecordNotFound)
	}
	if err := repo.NewIHAProxyServerRepo().Delete(repo.WithByID(id)); err != nil {
		return err
	}
	return s.ApplyChange(fmt.Sprintf("删除 Server: %s", old.Name), operator)
}

func (s *HAProxyService) validateServer(req *dto.HAProxyServerCreate) error {
	if !isValidHAProxyName(req.Name) {
		return buserr.WithDetail(constant.ErrInvalidParams, "server name must match [a-zA-Z0-9_.-]", nil)
	}
	if req.Address == "" || req.Port == 0 {
		return buserr.New(constant.ErrInvalidParams)
	}
	return nil
}

// --- ACL ---

func (s *HAProxyService) ListACL(lbID uint) ([]dto.HAProxyACLInfo, error) {
	items, err := repo.NewIHAProxyACLRepo().GetListByLB(lbID)
	if err != nil {
		return nil, err
	}
	bes, _ := repo.NewIHAProxyBackendRepo().GetList()
	beMap := make(map[uint]string)
	for _, b := range bes {
		beMap[b.ID] = b.Name
	}
	infos := make([]dto.HAProxyACLInfo, 0, len(items))
	for _, it := range items {
		infos = append(infos, dto.HAProxyACLInfo{
			ID: it.ID, LBID: it.LBID, Priority: it.Priority,
			MatchType: it.MatchType, MatchHeader: it.MatchHeader,
			MatchValue: it.MatchValue, TargetBackendID: it.TargetBackendID,
			TargetBackend: beMap[it.TargetBackendID], Enabled: it.Enabled,
			Remark: it.Remark,
		})
	}
	return infos, nil
}

func (s *HAProxyService) CreateACL(req dto.HAProxyACLCreate, operator string) error {
	if _, err := repo.NewIHAProxyLBRepo().Get(repo.WithByID(req.LBID)); err != nil {
		return buserr.New(constant.ErrRecordNotFound)
	}
	if _, err := repo.NewIHAProxyBackendRepo().Get(repo.WithByID(req.TargetBackendID)); err != nil {
		return buserr.New(constant.ErrRecordNotFound)
	}
	item := model.HAProxyACLRule{
		LBID: req.LBID, Priority: withDefault(req.Priority, 100),
		MatchType: req.MatchType, MatchHeader: req.MatchHeader,
		MatchValue: req.MatchValue, TargetBackendID: req.TargetBackendID,
		Enabled: req.Enabled, Remark: req.Remark,
	}
	if err := repo.NewIHAProxyACLRepo().Create(&item); err != nil {
		return err
	}
	return s.ApplyChange("新增 ACL 规则", operator)
}

func (s *HAProxyService) UpdateACL(req dto.HAProxyACLUpdate, operator string) error {
	if _, err := repo.NewIHAProxyACLRepo().Get(repo.WithByID(req.ID)); err != nil {
		return buserr.New(constant.ErrRecordNotFound)
	}
	updates := map[string]interface{}{
		"priority":          withDefault(req.Priority, 100),
		"match_type":        req.MatchType,
		"match_header":      req.MatchHeader,
		"match_value":       req.MatchValue,
		"target_backend_id": req.TargetBackendID,
		"enabled":           req.Enabled,
		"remark":            req.Remark,
	}
	if err := repo.NewIHAProxyACLRepo().Update(req.ID, updates); err != nil {
		return err
	}
	return s.ApplyChange("更新 ACL 规则", operator)
}

func (s *HAProxyService) DeleteACL(id uint, operator string) error {
	if err := repo.NewIHAProxyACLRepo().Delete(repo.WithByID(id)); err != nil {
		return err
	}
	return s.ApplyChange("删除 ACL 规则", operator)
}

// --- Raw Config / 版本管理 ---

func (s *HAProxyService) GetRawConfig() (string, error) {
	if !isHAProxyInstalled() {
		return "", buserr.New(constant.ErrHAProxyNotInstalled)
	}
	data, err := os.ReadFile(haproxyConfigPath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (s *HAProxyService) SaveRawConfig(content, operator string) error {
	if !isHAProxyInstalled() {
		return buserr.New(constant.ErrHAProxyNotInstalled)
	}
	// 1. 校验
	if resp, err := s.TestConfig(content); err != nil || !resp.Valid {
		detail := ""
		if resp != nil {
			detail = resp.Output
		}
		if err != nil {
			detail = err.Error()
		}
		return buserr.WithDetail(constant.ErrHAProxyCheckFailed, detail, err)
	}
	// 2. 备份
	if err := backupHAProxyConfig("manual-save"); err != nil {
		global.LOG.Warnf("backup haproxy config failed: %v", err)
	}
	// 3. 写入
	if err := os.WriteFile(haproxyConfigPath, []byte(content), 0640); err != nil {
		return err
	}
	// 4. reload
	if out, err := cmd.ExecWithOutput("systemctl", "reload", haproxyServiceName); err != nil {
		rollbackHAProxyConfig()
		return buserr.WithDetail(constant.ErrHAProxyReloadFailed, strings.TrimSpace(out), err)
	}
	// 5. 记录版本
	_ = repo.NewIHAProxyConfigVersionRepo().Create(&model.HAProxyConfigVersion{
		Content: content, Reason: "手动编辑原始配置",
		Success: true, Operator: operator,
	})
	_ = repo.NewIHAProxyConfigVersionRepo().PruneOld(50)
	return nil
}

func (s *HAProxyService) TestConfig(content string) (*dto.HAProxyConfigTestResp, error) {
	tmp, err := os.CreateTemp("", "haproxy-*.cfg")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.WriteString(content); err != nil {
		tmp.Close()
		return nil, err
	}
	tmp.Close()
	out, err := exec.Command(haproxyBinary, "-c", "-f", tmp.Name()).CombinedOutput()
	resp := &dto.HAProxyConfigTestResp{Output: string(out)}
	if err == nil {
		resp.Valid = true
	}
	return resp, nil
}

func (s *HAProxyService) PreviewConfig() (string, error) {
	return s.buildConfig()
}

func (s *HAProxyService) RebuildConfig(operator string) error {
	return s.ApplyChange("手动重建配置", operator)
}

func (s *HAProxyService) ListConfigVersions(limit int) ([]dto.HAProxyConfigVersionInfo, error) {
	items, err := repo.NewIHAProxyConfigVersionRepo().List(limit)
	if err != nil {
		return nil, err
	}
	infos := make([]dto.HAProxyConfigVersionInfo, 0, len(items))
	for _, it := range items {
		infos = append(infos, dto.HAProxyConfigVersionInfo{
			ID: it.ID,
			Version: it.CreatedAt.Format("20060102-150405"),
			Reason: it.Reason, Operator: it.Operator,
			Success: it.Success,
			CreatedAt: it.CreatedAt.Format(time.RFC3339),
		})
	}
	return infos, nil
}

func (s *HAProxyService) GetConfigVersion(id uint) (string, error) {
	v, err := repo.NewIHAProxyConfigVersionRepo().Get(id)
	if err != nil {
		return "", buserr.New(constant.ErrRecordNotFound)
	}
	return v.Content, nil
}

func (s *HAProxyService) RollbackToVersion(id uint, operator string) error {
	v, err := repo.NewIHAProxyConfigVersionRepo().Get(id)
	if err != nil {
		return buserr.New(constant.ErrRecordNotFound)
	}
	return s.SaveRawConfig(v.Content, operator+" (rollback #"+fmt.Sprintf("%d", id)+")")
}

// --- ApplyChange：核心三段式（生成 → 校验 → 备份 → reload → 失败回滚） ---

func (s *HAProxyService) ApplyChange(reason, operator string) error {
	if !isHAProxyInstalled() {
		return buserr.New(constant.ErrHAProxyNotInstalled)
	}
	content, err := s.buildConfig()
	if err != nil {
		return err
	}
	// 校验
	resp, err := s.TestConfig(content)
	if err != nil {
		return err
	}
	if !resp.Valid {
		_ = repo.NewIHAProxyConfigVersionRepo().Create(&model.HAProxyConfigVersion{
			Content: content, Reason: reason + " (校验失败)",
			Success: false, Operator: operator,
		})
		return buserr.WithDetail(constant.ErrHAProxyCheckFailed, resp.Output, nil)
	}

	if err := backupHAProxyConfig(reason); err != nil {
		global.LOG.Warnf("backup haproxy config failed: %v", err)
	}
	if err := os.WriteFile(haproxyConfigPath, []byte(content), 0640); err != nil {
		return err
	}
	// reload
	if out, err := cmd.ExecWithOutput("systemctl", "reload", haproxyServiceName); err != nil {
		rollbackHAProxyConfig()
		_ = repo.NewIHAProxyConfigVersionRepo().Create(&model.HAProxyConfigVersion{
			Content: content, Reason: reason + " (reload失败)",
			Success: false, Operator: operator,
		})
		return buserr.WithDetail(constant.ErrHAProxyReloadFailed, strings.TrimSpace(out), err)
	}
	_ = repo.NewIHAProxyConfigVersionRepo().Create(&model.HAProxyConfigVersion{
		Content: content, Reason: reason,
		Success: true, Operator: operator,
	})
	_ = repo.NewIHAProxyConfigVersionRepo().PruneOld(50)
	return nil
}

func (s *HAProxyService) buildConfig() (string, error) {
	lbs, err := repo.NewIHAProxyLBRepo().GetList()
	if err != nil {
		return "", err
	}
	bes, err := repo.NewIHAProxyBackendRepo().GetList()
	if err != nil {
		return "", err
	}
	serverMap := make(map[uint][]model.HAProxyServer)
	for _, be := range bes {
		srvs, _ := repo.NewIHAProxyServerRepo().GetListByBackend(be.ID)
		serverMap[be.ID] = srvs
	}
	aclMap := make(map[uint][]model.HAProxyACLRule)
	for _, lb := range lbs {
		as, _ := repo.NewIHAProxyACLRepo().GetListByLB(lb.ID)
		aclMap[lb.ID] = as
	}
	user, pass := getHAProxyStatsAuth()
	settings := haproxyutil.Settings{
		GlobalLog:   readHAProxySetting("HAProxyGlobalLog", "127.0.0.1 local0"),
		SocketPath:  haproxyutil.DefaultSocketPath,
		StatsEnable: readHAProxySetting("HAProxyStatsEnable", "enable") == "enable",
		StatsBind:   readHAProxySetting("HAProxyStatsBind", "127.0.0.1:9999"),
		StatsURI:    readHAProxySetting("HAProxyStatsURI", "/stats"),
		StatsUser:   user,
		StatsPass:   pass,
		MaxConn:     50000,
	}
	return haproxyutil.Build(haproxyutil.BuilderInput{
		Settings: settings,
		LBs:      lbs,
		Backends: bes,
		Servers:  serverMap,
		ACLs:     aclMap,
		CertPathFor: func(lb model.HAProxyLB) string {
			if !lb.EnableSSL || lb.CertificateID == 0 {
				return ""
			}
			return ensureCombinedPEM(lb.CertificateID)
		},
	}), nil
}

// ensureCombinedPEM 将 X-Panel SSL 证书的 fullchain+privkey 合并成 HAProxy 所需的单 PEM
func ensureCombinedPEM(certID uint) string {
	cert, err := repo.NewICertificateRepo().Get(repo.WithByID(certID))
	if err != nil {
		return ""
	}
	sslDir := NewICertificateService().GetSSLDir()
	domainDir := safeDomainDir(cert.PrimaryDomain)
	fullchain := filepath.Join(sslDir, "certs", domainDir, "fullchain.pem")
	privkey := filepath.Join(sslDir, "certs", domainDir, "privkey.pem")

	chainData, err := os.ReadFile(fullchain)
	if err != nil {
		global.LOG.Warnf("haproxy merge pem: read fullchain failed: %v", err)
		return ""
	}
	keyData, err := os.ReadFile(privkey)
	if err != nil {
		global.LOG.Warnf("haproxy merge pem: read privkey failed: %v", err)
		return ""
	}
	_ = os.MkdirAll(haproxyCombinedPEMDir, 0750)
	outPath := filepath.Join(haproxyCombinedPEMDir, fmt.Sprintf("cert-%d.pem", certID))

	var combined strings.Builder
	combined.Write(chainData)
	if len(chainData) > 0 && chainData[len(chainData)-1] != '\n' {
		combined.WriteByte('\n')
	}
	combined.Write(keyData)

	if err := os.WriteFile(outPath, []byte(combined.String()), 0640); err != nil {
		global.LOG.Warnf("haproxy merge pem: write %s failed: %v", outPath, err)
		return ""
	}
	return outPath
}

// --- 辅助 ---

func backupHAProxyConfig(reason string) error {
	_ = os.MkdirAll(haproxyBackupDir, 0750)
	src, err := os.ReadFile(haproxyConfigPath)
	if err != nil {
		return err
	}
	name := fmt.Sprintf("haproxy.cfg.%d.bak", time.Now().Unix())
	_ = os.WriteFile(filepath.Join(haproxyBackupDir, "latest.bak"), src, 0640)
	return os.WriteFile(filepath.Join(haproxyBackupDir, name), src, 0640)
}

func rollbackHAProxyConfig() {
	bak := filepath.Join(haproxyBackupDir, "latest.bak")
	data, err := os.ReadFile(bak)
	if err != nil {
		return
	}
	_ = os.WriteFile(haproxyConfigPath, data, 0640)
	_, _ = cmd.ExecWithOutput("systemctl", "reload", haproxyServiceName)
}

func readHAProxySetting(key, def string) string {
	v, err := repo.NewISettingRepo().Get(repo.WithByKey(key))
	if err != nil || v.Value == "" {
		return def
	}
	return v.Value
}

func isValidHAProxyName(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '.' || c == '-') {
			return false
		}
	}
	return true
}

func defaultBindAddr(v string) string {
	if v == "" {
		return "0.0.0.0"
	}
	return v
}

func defaultStr(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

func withDefault(v, def int) int {
	if v <= 0 {
		return def
	}
	return v
}
