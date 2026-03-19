package service

import (
	"net"
	"strings"
	"sync"
	"time"

	"xpanel/app/dto"
	"xpanel/app/model"
	"xpanel/app/repo"
	"xpanel/global"

	gnet "github.com/shirou/gopsutil/v4/net"
	"gorm.io/gorm"
)

type ITrafficService interface {
	ListConfigs() ([]dto.TrafficConfigDTO, error)
	CreateConfig(req dto.TrafficConfigCreate) error
	DeleteConfig(interfaceName string) error
	ListInterfaces() ([]dto.InterfaceInfo, error)
	GetStats(req dto.TrafficStatsRequest) (*dto.TrafficStatsResponse, error)
	GetSummary() ([]dto.TrafficSummaryItem, error)
	GetRealtime() ([]dto.TrafficRealtimeItem, error)
	StartCollector()
	CollectOnce()
	CleanOldRecords()
}

type TrafficService struct {
	repo repo.ITrafficRepo
}

var (
	trafficLastIO   map[string]gnet.IOCountersStat
	trafficLastTime time.Time
	trafficMu       sync.Mutex
)

func NewITrafficService() ITrafficService {
	return &TrafficService{repo: repo.NewITrafficRepo()}
}

func (s *TrafficService) ListConfigs() ([]dto.TrafficConfigDTO, error) {
	configs, err := s.repo.ListConfigs()
	if err != nil {
		return nil, err
	}
	var result []dto.TrafficConfigDTO
	for _, c := range configs {
		result = append(result, dto.TrafficConfigDTO{
			ID:            c.ID,
			InterfaceName: c.InterfaceName,
			MonthlyLimit:  c.MonthlyLimit,
			ResetDay:      c.ResetDay,
			Enabled:       c.Enabled,
		})
	}
	return result, nil
}

func (s *TrafficService) CreateConfig(req dto.TrafficConfigCreate) error {
	existing, err := s.repo.GetConfig(req.InterfaceName)
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	if existing.ID > 0 {
		existing.MonthlyLimit = req.MonthlyLimit
		existing.ResetDay = req.ResetDay
		existing.Enabled = req.Enabled
		return s.repo.SaveConfig(&existing)
	}
	item := &model.TrafficConfig{
		InterfaceName: req.InterfaceName,
		MonthlyLimit:  req.MonthlyLimit,
		ResetDay:      req.ResetDay,
		Enabled:       req.Enabled,
	}
	if err := s.repo.SaveConfig(item); err != nil {
		return err
	}
	// Immediately take a snapshot so the next collect cycle can compute a delta
	go s.CollectOnce()
	return nil
}

func (s *TrafficService) DeleteConfig(interfaceName string) error {
	if err := s.repo.DeleteConfig(interfaceName); err != nil {
		return err
	}
	_ = s.repo.DeleteSnapshot(interfaceName)
	return nil
}

func (s *TrafficService) ListInterfaces() ([]dto.InterfaceInfo, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	var result []dto.InterfaceInfo
	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		info := dto.InterfaceInfo{
			Name: iface.Name,
			MAC:  iface.HardwareAddr.String(),
		}
		if iface.Flags&net.FlagUp != 0 {
			info.Status = "up"
		} else {
			info.Status = "down"
		}
		addrs, err := iface.Addrs()
		if err == nil {
			for _, addr := range addrs {
				ip := addr.String()
				if strings.Contains(ip, ":") {
					info.IPv6 = append(info.IPv6, ip)
				} else {
					info.IPv4 = append(info.IPv4, ip)
				}
			}
		}
		result = append(result, info)
	}
	return result, nil
}

func (s *TrafficService) GetStats(req dto.TrafficStatsRequest) (*dto.TrafficStatsResponse, error) {
	start, err := parseFlexTime(req.StartTime)
	if err != nil {
		return nil, err
	}
	end, err := parseFlexTime(req.EndTime)
	if err != nil {
		return nil, err
	}
	end = end.Add(24 * time.Hour)

	records, err := s.repo.ListHourly(req.InterfaceName, start, end)
	if err != nil {
		return nil, err
	}

	groupBy := req.GroupBy
	if groupBy == "" {
		groupBy = "day"
	}

	resp := &dto.TrafficStatsResponse{InterfaceName: req.InterfaceName}

	if groupBy == "hour" {
		for _, r := range records {
			resp.Items = append(resp.Items, dto.TrafficStatsItem{
				Timestamp: r.Timestamp.Format("2006-01-02 15:04"),
				BytesSent: r.BytesSent,
				BytesRecv: r.BytesRecv,
			})
			resp.TotalSent += r.BytesSent
			resp.TotalRecv += r.BytesRecv
		}
	} else {
		dayMap := make(map[string]*dto.TrafficStatsItem)
		var dayOrder []string
		for _, r := range records {
			day := r.Timestamp.Format("2006-01-02")
			if _, ok := dayMap[day]; !ok {
				dayMap[day] = &dto.TrafficStatsItem{Timestamp: day}
				dayOrder = append(dayOrder, day)
			}
			dayMap[day].BytesSent += r.BytesSent
			dayMap[day].BytesRecv += r.BytesRecv
			resp.TotalSent += r.BytesSent
			resp.TotalRecv += r.BytesRecv
		}
		for _, day := range dayOrder {
			resp.Items = append(resp.Items, *dayMap[day])
		}
	}
	return resp, nil
}

func (s *TrafficService) GetSummary() ([]dto.TrafficSummaryItem, error) {
	configs, err := s.repo.ListConfigs()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	var result []dto.TrafficSummaryItem
	for _, c := range configs {
		periodStart, periodEnd := calcBillingPeriod(now, c.ResetDay)
		sent, recv, err := s.repo.SumTraffic(c.InterfaceName, periodStart, periodEnd)
		if err != nil {
			continue
		}
		total := sent + recv
		var pct float64
		if c.MonthlyLimit > 0 {
			pct = float64(total) / float64(c.MonthlyLimit) * 100
			if pct > 100 {
				pct = 100
			}
		}
		result = append(result, dto.TrafficSummaryItem{
			InterfaceName: c.InterfaceName,
			MonthlyLimit:  c.MonthlyLimit,
			ResetDay:      c.ResetDay,
			PeriodStart:   periodStart,
			PeriodEnd:     periodEnd,
			TotalSent:     sent,
			TotalRecv:     recv,
			TotalUsed:     total,
			UsedPercent:   pct,
			Enabled:       c.Enabled,
		})
	}
	return result, nil
}

// GetRealtime returns live upload/download speeds for all monitored interfaces.
func (s *TrafficService) GetRealtime() ([]dto.TrafficRealtimeItem, error) {
	netIO, err := gnet.IOCounters(true)
	if err != nil {
		return nil, err
	}

	trafficMu.Lock()
	defer trafficMu.Unlock()

	now := time.Now()
	elapsed := now.Sub(trafficLastTime).Seconds()
	var result []dto.TrafficRealtimeItem

	for _, nic := range netIO {
		if nic.Name == "lo" {
			continue
		}
		item := dto.TrafficRealtimeItem{
			Name:      nic.Name,
			BytesSent: nic.BytesSent,
			BytesRecv: nic.BytesRecv,
		}
		if elapsed > 0 && elapsed < 30 && trafficLastIO != nil {
			if prev, ok := trafficLastIO[nic.Name]; ok {
				item.SpeedUp = float64(nic.BytesSent-prev.BytesSent) / elapsed
				item.SpeedDown = float64(nic.BytesRecv-prev.BytesRecv) / elapsed
				if item.SpeedUp < 0 {
					item.SpeedUp = 0
				}
				if item.SpeedDown < 0 {
					item.SpeedDown = 0
				}
			}
		}
		result = append(result, item)
	}

	// Update last state
	trafficLastIO = make(map[string]gnet.IOCountersStat)
	for _, nic := range netIO {
		trafficLastIO[nic.Name] = nic
	}
	trafficLastTime = now

	return result, nil
}

func (s *TrafficService) StartCollector() {
	// Run immediately on startup to capture the initial snapshot
	go s.CollectOnce()

	_, err := global.CRON.AddFunc("@every 5m", func() {
		s.CollectOnce()
	})
	if err != nil {
		global.LOG.Errorf("Failed to register traffic collector cron: %v", err)
	}

	_, err = global.CRON.AddFunc("0 3 1 * *", func() {
		s.CleanOldRecords()
	})
	if err != nil {
		global.LOG.Errorf("Failed to register traffic cleanup cron: %v", err)
	}

	global.LOG.Info("Traffic collector started (interval: 5m)")
}

func (s *TrafficService) CollectOnce() {
	configs, err := s.repo.ListConfigs()
	if err != nil {
		global.LOG.Errorf("Traffic collect: failed to list configs: %v", err)
		return
	}
	if len(configs) == 0 {
		return
	}

	enabledMap := make(map[string]bool)
	for _, c := range configs {
		if c.Enabled {
			enabledMap[c.InterfaceName] = true
		}
	}
	if len(enabledMap) == 0 {
		return
	}

	netIO, err := gnet.IOCounters(true)
	if err != nil {
		global.LOG.Errorf("Traffic collect: failed to read IO counters: %v", err)
		return
	}

	now := time.Now()
	for _, nic := range netIO {
		if !enabledMap[nic.Name] {
			continue
		}

		snapshot, err := s.repo.GetSnapshot(nic.Name)
		if err != nil && err != gorm.ErrRecordNotFound {
			global.LOG.Errorf("Traffic collect: failed to get snapshot for %s: %v", nic.Name, err)
			continue
		}

		var deltaSent, deltaRecv uint64
		if err == gorm.ErrRecordNotFound {
			deltaSent = 0
			deltaRecv = 0
		} else {
			if nic.BytesSent >= snapshot.BytesSent {
				deltaSent = nic.BytesSent - snapshot.BytesSent
			} else {
				deltaSent = nic.BytesSent
			}
			if nic.BytesRecv >= snapshot.BytesRecv {
				deltaRecv = nic.BytesRecv - snapshot.BytesRecv
			} else {
				deltaRecv = nic.BytesRecv
			}
		}

		if deltaSent > 0 || deltaRecv > 0 {
			if err := s.repo.UpsertHourly(nic.Name, now, deltaSent, deltaRecv); err != nil {
				global.LOG.Errorf("Traffic collect: failed to upsert hourly for %s: %v", nic.Name, err)
			}
		}

		newSnapshot := &model.TrafficSnapshot{
			InterfaceName: nic.Name,
			BytesSent:     nic.BytesSent,
			BytesRecv:     nic.BytesRecv,
			SampledAt:     now,
		}
		if err := s.repo.SaveSnapshot(newSnapshot); err != nil {
			global.LOG.Errorf("Traffic collect: failed to save snapshot for %s: %v", nic.Name, err)
		}
	}
}

func (s *TrafficService) CleanOldRecords() {
	cutoff := time.Now().AddDate(-1, 0, 0)
	if err := s.repo.DeleteHourlyBefore(cutoff); err != nil {
		global.LOG.Errorf("Traffic cleanup: failed to delete old records: %v", err)
	} else {
		global.LOG.Info("Traffic cleanup: deleted records older than 12 months")
	}
}

func calcBillingPeriod(now time.Time, resetDay int) (time.Time, time.Time) {
	if resetDay < 1 {
		resetDay = 1
	}
	if resetDay > 28 {
		resetDay = 28
	}

	year, month, day := now.Date()
	loc := now.Location()

	var periodStart time.Time
	if day >= resetDay {
		periodStart = time.Date(year, month, resetDay, 0, 0, 0, 0, loc)
	} else {
		periodStart = time.Date(year, month-1, resetDay, 0, 0, 0, 0, loc)
	}
	periodEnd := periodStart.AddDate(0, 1, 0)
	return periodStart, periodEnd
}

func parseFlexTime(s string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339, s)
	if err == nil {
		return t, nil
	}
	t, err = time.ParseInLocation("2006-01-02", s, time.Now().Location())
	if err == nil {
		return t, nil
	}
	return time.Time{}, err
}
