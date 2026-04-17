package service

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"xpanel/app/dto"
	"xpanel/app/repo"
	"xpanel/buserr"
	"xpanel/constant"
	haproxyutil "xpanel/utils/haproxy"
)

type IHAProxyRuntimeService interface {
	ToggleServerLive(req dto.HAProxyServerToggleLive) error
	SetServerWeightLive(req dto.HAProxyServerWeightLive) error
	GetStats() (*dto.HAProxyStatsInfo, error)
	GetInfo() (*dto.HAProxyRuntimeInfo, error)
	ClearCounters() error
}

type HAProxyRuntimeService struct{}

func NewIHAProxyRuntimeService() IHAProxyRuntimeService { return &HAProxyRuntimeService{} }

func (s *HAProxyRuntimeService) ToggleServerLive(req dto.HAProxyServerToggleLive) error {
	srv, err := repo.NewIHAProxyServerRepo().Get(repo.WithByID(req.ID))
	if err != nil {
		return buserr.New(constant.ErrRecordNotFound)
	}
	be, err := repo.NewIHAProxyBackendRepo().Get(repo.WithByID(srv.BackendID))
	if err != nil {
		return buserr.New(constant.ErrRecordNotFound)
	}
	sock := haproxyutil.NewSocket(haproxyutil.DefaultSocketPath)
	if req.Disable {
		if err := sock.DisableServer(be.Name, srv.Name); err != nil {
			return buserr.WithErr(constant.ErrHAProxySocketFailed, err)
		}
	} else {
		if err := sock.EnableServer(be.Name, srv.Name); err != nil {
			return buserr.WithErr(constant.ErrHAProxySocketFailed, err)
		}
	}
	return repo.NewIHAProxyServerRepo().Update(req.ID, map[string]interface{}{"disabled": req.Disable})
}

func (s *HAProxyRuntimeService) SetServerWeightLive(req dto.HAProxyServerWeightLive) error {
	srv, err := repo.NewIHAProxyServerRepo().Get(repo.WithByID(req.ID))
	if err != nil {
		return buserr.New(constant.ErrRecordNotFound)
	}
	be, err := repo.NewIHAProxyBackendRepo().Get(repo.WithByID(srv.BackendID))
	if err != nil {
		return buserr.New(constant.ErrRecordNotFound)
	}
	sock := haproxyutil.NewSocket(haproxyutil.DefaultSocketPath)
	if err := sock.SetWeight(be.Name, srv.Name, req.Weight); err != nil {
		return buserr.WithErr(constant.ErrHAProxySocketFailed, err)
	}
	return repo.NewIHAProxyServerRepo().Update(req.ID, map[string]interface{}{"weight": req.Weight})
}

// --- Stats（带轻量缓存避免压 socket） ---

type statsCache struct {
	mu        sync.Mutex
	expiresAt time.Time
	data      *dto.HAProxyStatsInfo
}

var haproxyStatsCache = &statsCache{}

func (s *HAProxyRuntimeService) GetStats() (*dto.HAProxyStatsInfo, error) {
	haproxyStatsCache.mu.Lock()
	defer haproxyStatsCache.mu.Unlock()
	if haproxyStatsCache.data != nil && time.Now().Before(haproxyStatsCache.expiresAt) {
		return haproxyStatsCache.data, nil
	}
	sock := haproxyutil.NewSocket(haproxyutil.DefaultSocketPath)
	raw, err := sock.ShowStat()
	if err != nil {
		return nil, buserr.WithErr(constant.ErrHAProxySocketFailed, err)
	}
	rows := haproxyutil.ParseStatCSV(raw)
	result := &dto.HAProxyStatsInfo{}
	backendCount := make(map[string]struct {
		Act, Bck, Tot int
	})
	for _, row := range rows {
		if row.SvName == "FRONTEND" {
			result.Frontends = append(result.Frontends, dto.HAProxyFrontendStat{
				Name: row.PxName, Status: row.Status,
				CurConns: row.Scur, MaxConns: row.Smax, TotalConns: row.Stot,
				BytesIn: row.Bin, BytesOut: row.Bout,
				ReqRate: row.ReqRate, TotalReq: row.ReqTot,
			})
		} else if row.SvName == "BACKEND" {
			c := backendCount[row.PxName]
			result.Backends = append(result.Backends, dto.HAProxyBackendStat{
				Name: row.PxName, Status: row.Status,
				CurConns: row.Scur, TotalConns: row.Stot,
				BytesIn: row.Bin, BytesOut: row.Bout,
				ActServers: c.Act, BckServers: c.Bck, TotalServers: c.Tot,
			})
		} else if row.PxName != "" && row.SvName != "" {
			// server row
			if strings.HasPrefix(row.PxName, "xpanel-stats") {
				continue
			}
			result.Servers = append(result.Servers, dto.HAProxyServerStat{
				Backend: row.PxName, Name: row.SvName, Status: row.Status,
				CurConns: row.Scur, TotalConns: row.Stot,
				BytesIn: row.Bin, BytesOut: row.Bout,
				CheckStatus: row.CheckSt, LastChange: row.Lastchg,
				Weight: row.Weight,
			})
			c := backendCount[row.PxName]
			c.Tot++
			if row.Status == "UP" || strings.HasPrefix(row.Status, "UP") {
				c.Act++
			}
			backendCount[row.PxName] = c
		}
	}
	// 回填 backend 聚合
	for i, b := range result.Backends {
		if c, ok := backendCount[b.Name]; ok {
			result.Backends[i].ActServers = c.Act
			result.Backends[i].BckServers = c.Bck
			result.Backends[i].TotalServers = c.Tot
		}
	}
	haproxyStatsCache.data = result
	haproxyStatsCache.expiresAt = time.Now().Add(2 * time.Second)
	return result, nil
}

func (s *HAProxyRuntimeService) GetInfo() (*dto.HAProxyRuntimeInfo, error) {
	sock := haproxyutil.NewSocket(haproxyutil.DefaultSocketPath)
	raw, err := sock.ShowInfo()
	if err != nil {
		return nil, buserr.WithErr(constant.ErrHAProxySocketFailed, err)
	}
	return &dto.HAProxyRuntimeInfo{Raw: raw}, nil
}

func (s *HAProxyRuntimeService) ClearCounters() error {
	sock := haproxyutil.NewSocket(haproxyutil.DefaultSocketPath)
	if err := sock.ClearCounters(); err != nil {
		return buserr.WithErr(constant.ErrHAProxySocketFailed, err)
	}
	return nil
}

// 避免 linter unused
var _ = fmt.Sprintf
