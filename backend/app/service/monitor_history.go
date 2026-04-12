package service

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"xpanel/app/dto"
	"xpanel/app/model"
	"xpanel/app/repo"
	"xpanel/global"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/net"
)

type IMonitorHistoryService interface {
	Run()
	LoadMonitorData(req dto.MonitorSearch) ([]dto.MonitorData, error)
	LoadSetting() (*dto.MonitorSetting, error)
	UpdateSetting(key, value string) error
	CleanData() error

	saveIODataToDB(ctx context.Context, interval float64)
	saveNetDataToDB(ctx context.Context, interval float64)
}

type MonitorHistoryService struct {
	DiskIO chan []disk.IOCountersStat
	NetIO  chan []net.IOCountersStat
}

var monitorCancel context.CancelFunc

func NewIMonitorHistoryService() IMonitorHistoryService {
	return &MonitorHistoryService{
		DiskIO: make(chan []disk.IOCountersStat, 2),
		NetIO:  make(chan []net.IOCountersStat, 2),
	}
}

func (m *MonitorHistoryService) Run() {
	if global.MonitorDB == nil {
		return
	}

	var item model.MonitorBase
	totalPercent, _ := cpu.Percent(3*time.Second, false)
	if len(totalPercent) == 1 {
		item.Cpu = totalPercent[0]
	}

	loadInfo, _ := load.Avg()
	if loadInfo != nil {
		item.CpuLoad1 = loadInfo.Load1
		item.CpuLoad5 = loadInfo.Load5
		item.CpuLoad15 = loadInfo.Load15
		cores, _ := cpu.Counts(true)
		if cores > 0 {
			item.LoadUsage = (loadInfo.Load1 / float64(cores)) * 100
			if item.LoadUsage > 100 {
				item.LoadUsage = 100
			}
		}
	}

	memInfo, _ := mem.VirtualMemory()
	if memInfo != nil {
		item.Memory = memInfo.UsedPercent
	}

	if err := global.MonitorDB.Create(&item).Error; err != nil {
		global.LOG.Errorf("Insert monitor base data failed: %v", err)
	}

	m.loadDiskIO()
	m.loadNetIO()

	m.cleanExpiredData()
}

func (m *MonitorHistoryService) loadDiskIO() {
	ioStat, _ := disk.IOCounters()
	var diskIOList []disk.IOCountersStat
	var ioStatAll disk.IOCountersStat
	ioStatAll.Name = "all"
	for _, io := range ioStat {
		ioStatAll.ReadBytes += io.ReadBytes
		ioStatAll.WriteBytes += io.WriteBytes
		diskIOList = append(diskIOList, io)
	}
	diskIOList = append(diskIOList, ioStatAll)
	m.DiskIO <- diskIOList
}

func (m *MonitorHistoryService) loadNetIO() {
	netStat, _ := net.IOCounters(true)
	netStatAll, _ := net.IOCounters(false)
	var netList []net.IOCountersStat
	netList = append(netList, netStat...)
	netList = append(netList, netStatAll...)
	m.NetIO <- netList
}

func (m *MonitorHistoryService) saveIODataToDB(ctx context.Context, interval float64) {
	defer func() {
		recover()
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case ioStat := <-m.DiskIO:
			select {
			case <-ctx.Done():
				return
			case ioStat2 := <-m.DiskIO:
				var ioList []model.MonitorIO
				for _, io2 := range ioStat2 {
					for _, io1 := range ioStat {
						if io2.Name == io1.Name {
							var itemIO model.MonitorIO
							itemIO.Name = io1.Name
							if io2.ReadBytes > io1.ReadBytes {
								itemIO.Read = uint64(float64(io2.ReadBytes-io1.ReadBytes) / interval)
							}
							if io2.WriteBytes > io1.WriteBytes {
								itemIO.Write = uint64(float64(io2.WriteBytes-io1.WriteBytes) / interval)
							}
							ioList = append(ioList, itemIO)
							break
						}
					}
				}
				if len(ioList) > 0 {
					global.MonitorDB.CreateInBatches(ioList, len(ioList))
				}
				m.DiskIO <- ioStat2
			}
		}
	}
}

func (m *MonitorHistoryService) saveNetDataToDB(ctx context.Context, interval float64) {
	defer func() {
		recover()
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case netStat := <-m.NetIO:
			select {
			case <-ctx.Done():
				return
			case netStat2 := <-m.NetIO:
				var netList []model.MonitorNetwork
				for _, net2 := range netStat2 {
					for _, net1 := range netStat {
						if net2.Name == net1.Name {
							var itemNet model.MonitorNetwork
							itemNet.Name = net1.Name
							if net2.BytesSent > net1.BytesSent {
								itemNet.Up = float64(net2.BytesSent-net1.BytesSent) / 1024 / interval
							}
							if net2.BytesRecv > net1.BytesRecv {
								itemNet.Down = float64(net2.BytesRecv-net1.BytesRecv) / 1024 / interval
							}
							netList = append(netList, itemNet)
							break
						}
					}
				}
				if len(netList) > 0 {
					global.MonitorDB.CreateInBatches(netList, len(netList))
				}
				m.NetIO <- netStat2
			}
		}
	}
}

func (m *MonitorHistoryService) cleanExpiredData() {
	settingRepo := NewISettingService()
	daysStr, err := settingRepo.GetValueByKey("MonitorStoreDays")
	if err != nil {
		return
	}
	days, _ := strconv.Atoi(daysStr)
	if days <= 0 {
		days = 7
	}
	cutoff := time.Now().AddDate(0, 0, -days)
	global.MonitorDB.Where("created_at < ?", cutoff).Delete(&model.MonitorBase{})
	global.MonitorDB.Where("created_at < ?", cutoff).Delete(&model.MonitorIO{})
	global.MonitorDB.Where("created_at < ?", cutoff).Delete(&model.MonitorNetwork{})
}

func (m *MonitorHistoryService) LoadMonitorData(req dto.MonitorSearch) ([]dto.MonitorData, error) {
	if global.MonitorDB == nil {
		return nil, fmt.Errorf("monitor database not initialized")
	}

	var data []dto.MonitorData

	if req.Param == "all" || req.Param == "cpu" || req.Param == "memory" || req.Param == "load" {
		var bases []model.MonitorBase
		global.MonitorDB.Where("created_at >= ? AND created_at <= ?", req.StartTime, req.EndTime).
			Order("created_at ASC").Find(&bases)

		var itemData dto.MonitorData
		itemData.Param = "base"
		for _, base := range bases {
			itemData.Date = append(itemData.Date, base.CreatedAt)
			itemData.Value = append(itemData.Value, base)
		}
		data = append(data, itemData)
	}

	if req.Param == "all" || req.Param == "io" {
		ioName := req.IO
		if ioName == "" {
			ioName = "all"
		}
		var ios []model.MonitorIO
		query := global.MonitorDB.Where("created_at >= ? AND created_at <= ?", req.StartTime, req.EndTime)
		if ioName != "all" {
			query = query.Where("name = ?", ioName)
		} else {
			query = query.Where("name = ?", "all")
		}
		query.Order("created_at ASC").Find(&ios)

		var itemData dto.MonitorData
		itemData.Param = "io"
		for _, io := range ios {
			itemData.Date = append(itemData.Date, io.CreatedAt)
			itemData.Value = append(itemData.Value, io)
		}
		data = append(data, itemData)
	}

	if req.Param == "all" || req.Param == "network" {
		netName := req.Network
		if netName == "" {
			netName = "all"
		}
		var nets []model.MonitorNetwork
		query := global.MonitorDB.Where("created_at >= ? AND created_at <= ?", req.StartTime, req.EndTime)
		if netName != "all" {
			query = query.Where("name = ?", netName)
		} else {
			query = query.Where("name = ?", "all")
		}
		query.Order("created_at ASC").Find(&nets)

		var itemData dto.MonitorData
		itemData.Param = "network"
		for _, n := range nets {
			itemData.Date = append(itemData.Date, n.CreatedAt)
			itemData.Value = append(itemData.Value, n)
		}
		data = append(data, itemData)
	}

	return data, nil
}

func (m *MonitorHistoryService) LoadSetting() (*dto.MonitorSetting, error) {
	settingService := NewISettingService()
	var setting dto.MonitorSetting

	val, _ := settingService.GetValueByKey("MonitorStatus")
	setting.MonitorStatus = val
	val, _ = settingService.GetValueByKey("MonitorInterval")
	setting.MonitorInterval = val
	val, _ = settingService.GetValueByKey("MonitorStoreDays")
	setting.MonitorStoreDays = val
	val, _ = settingService.GetValueByKey("DefaultNetwork")
	setting.DefaultNetwork = val
	val, _ = settingService.GetValueByKey("DefaultIO")
	setting.DefaultIO = val

	return &setting, nil
}

func (m *MonitorHistoryService) UpdateSetting(key, value string) error {
	settingRepo := NewISettingService()

	switch key {
	case "MonitorStatus":
		if value == "enable" && global.MonitorCronID == 0 {
			intervalStr, _ := settingRepo.GetValueByKey("MonitorInterval")
			if err := StartMonitorCollector(false, intervalStr); err != nil {
				return err
			}
		}
		if value == "disable" && global.MonitorCronID != 0 {
			if monitorCancel != nil {
				monitorCancel()
			}
			global.CRON.Remove(global.MonitorCronID)
			global.MonitorCronID = 0
		}
	case "MonitorInterval":
		statusStr, _ := settingRepo.GetValueByKey("MonitorStatus")
		if statusStr == "enable" && global.MonitorCronID != 0 {
			if err := StartMonitorCollector(true, value); err != nil {
				return err
			}
		}
	}

	return repo.NewISettingRepo().Update(key, value)
}

func (m *MonitorHistoryService) CleanData() error {
	if global.MonitorDB == nil {
		return fmt.Errorf("monitor database not initialized")
	}
	global.MonitorDB.Exec("DELETE FROM monitor_bases")
	global.MonitorDB.Exec("DELETE FROM monitor_ios")
	global.MonitorDB.Exec("DELETE FROM monitor_networks")
	return nil
}

// StartMonitorCollector 启动监控采集器
func StartMonitorCollector(removeBefore bool, interval string) error {
	if global.MonitorDB == nil {
		return fmt.Errorf("monitor database not initialized")
	}

	if removeBefore && global.MonitorCronID != 0 {
		if monitorCancel != nil {
			monitorCancel()
		}
		global.CRON.Remove(global.MonitorCronID)
		global.MonitorCronID = 0
	}

	intervalSec, err := strconv.Atoi(interval)
	if err != nil || intervalSec <= 0 {
		intervalSec = 300
	}

	svc := NewIMonitorHistoryService()
	ctx, cancel := context.WithCancel(context.Background())
	monitorCancel = cancel

	svc.Run()

	go svc.saveIODataToDB(ctx, float64(intervalSec))
	go svc.saveNetDataToDB(ctx, float64(intervalSec))

	cronID, err := global.CRON.AddJob(fmt.Sprintf("@every %ds", intervalSec), svc)
	if err != nil {
		cancel()
		return fmt.Errorf("register monitor cron failed: %v", err)
	}
	global.MonitorCronID = cronID

	global.LOG.Infof("Monitor collector started (interval: %ds)", intervalSec)
	return nil
}
