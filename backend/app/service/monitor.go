package service

import (
	"runtime"
	"sort"
	"sync"
	"time"

	"xpanel/app/dto"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	hostUtil "github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/process"
)

type IMonitorService interface {
	GetCurrentStats() (*dto.SystemStats, error)
}

type MonitorService struct{}

func NewIMonitorService() IMonitorService { return &MonitorService{} }

// 上一次网络采样，用于计算速率
var (
	lastNetIO   []net.IOCountersStat
	lastNetTime time.Time
	netMu       sync.Mutex
)

func (s *MonitorService) GetCurrentStats() (*dto.SystemStats, error) {
	stats := &dto.SystemStats{}

	// 系统基本信息
	hostInfo, err := hostUtil.Info()
	if err == nil {
		stats.Host = dto.SystemHostInfo{
			Hostname:        hostInfo.Hostname,
			OS:              hostInfo.OS,
			Platform:        hostInfo.Platform,
			PlatformVersion: hostInfo.PlatformVersion,
			KernelVersion:   hostInfo.KernelVersion,
			KernelArch:      hostInfo.KernelArch,
		}
	}

	// CPU
	cpuInfo, _ := cpu.Info()
	cpuPercent, _ := cpu.Percent(0, false)
	perCPU, _ := cpu.Percent(0, true)
	physicalCores, _ := cpu.Counts(false)
	logicalCores, _ := cpu.Counts(true)

	stats.CPU = dto.CPUStats{
		Cores:        physicalCores,
		LogicalCores: logicalCores,
		PerCPU:       perCPU,
	}
	if len(cpuInfo) > 0 {
		stats.CPU.ModelName = cpuInfo[0].ModelName
	}
	if len(cpuPercent) > 0 {
		stats.CPU.UsagePercent = cpuPercent[0]
	}

	// 内存
	memStat, err := mem.VirtualMemory()
	if err == nil {
		stats.Memory = dto.MemoryStats{
			Total:       memStat.Total,
			Used:        memStat.Used,
			Available:   memStat.Available,
			UsedPercent: memStat.UsedPercent,
		}
	}
	swapStat, err := mem.SwapMemory()
	if err == nil {
		stats.Memory.SwapTotal = swapStat.Total
		stats.Memory.SwapUsed = swapStat.Used
		stats.Memory.SwapPercent = swapStat.UsedPercent
	}

	// 负载（Linux/macOS）
	loadStat, err := load.Avg()
	if err == nil {
		stats.Load = dto.LoadStats{
			Load1:  loadStat.Load1,
			Load5:  loadStat.Load5,
			Load15: loadStat.Load15,
		}
	}

	// 磁盘（含 inode）
	partitions, err := disk.Partitions(false)
	if err == nil {
		for _, p := range partitions {
			if shouldSkipPartition(p) {
				continue
			}
			usage, err := disk.Usage(p.Mountpoint)
			if err != nil || usage.Total == 0 {
				continue
			}
			stats.Disks = append(stats.Disks, dto.DiskStats{
				Device:        p.Device,
				MountPoint:    p.Mountpoint,
				FSType:        p.Fstype,
				Total:         usage.Total,
				Used:          usage.Used,
				Free:          usage.Free,
				UsedPercent:   usage.UsedPercent,
				InodesTotal:   usage.InodesTotal,
				InodesUsed:    usage.InodesUsed,
				InodesFree:    usage.InodesFree,
				InodesPercent: usage.InodesUsedPercent,
			})
		}
	}

	// 网络（累计 + 每网卡速率）
	netIO, err := net.IOCounters(true) // per-nic
	if err == nil {
		var totalSent, totalRecv, totalPktSent, totalPktRecv uint64
		netMu.Lock()
		elapsed := time.Since(lastNetTime).Seconds()
		for _, nic := range netIO {
			if nic.Name == "lo" {
				continue
			}
			totalSent += nic.BytesSent
			totalRecv += nic.BytesRecv
			totalPktSent += nic.PacketsSent
			totalPktRecv += nic.PacketsRecv

			nio := dto.NetIOStats{
				Name:      nic.Name,
				BytesSent: nic.BytesSent,
				BytesRecv: nic.BytesRecv,
			}
			// 计算速率
			if elapsed > 0 && lastNetIO != nil {
				for _, prev := range lastNetIO {
					if prev.Name == nic.Name {
						nio.SpeedUp = float64(nic.BytesSent-prev.BytesSent) / elapsed
						nio.SpeedDown = float64(nic.BytesRecv-prev.BytesRecv) / elapsed
						break
					}
				}
			}
			stats.NetIO = append(stats.NetIO, nio)
		}
		lastNetIO = netIO
		lastNetTime = time.Now()
		netMu.Unlock()

		stats.Network = dto.NetworkStats{
			BytesSent:   totalSent,
			BytesRecv:   totalRecv,
			PacketsSent: totalPktSent,
			PacketsRecv: totalPktRecv,
		}
	}

	// Top 进程（CPU Top 5）
	stats.TopProcess = getTopProcesses(5)

	// Uptime
	uptime, _ := hostUtil.Uptime()
	stats.Uptime = uptime

	return stats, nil
}

// getTopProcesses 获取 CPU 占用 Top N 的进程
func getTopProcesses(n int) []dto.ProcessBrief {
	procs, err := process.Processes()
	if err != nil {
		return nil
	}

	var briefs []dto.ProcessBrief
	for _, p := range procs {
		cpuPct, err := p.CPUPercent()
		if err != nil {
			continue
		}
		name, _ := p.Name()
		memPct, _ := p.MemoryPercent()
		memInfo, _ := p.MemoryInfo()
		var rss uint64
		if memInfo != nil {
			rss = memInfo.RSS
		}
		briefs = append(briefs, dto.ProcessBrief{
			PID:        p.Pid,
			Name:       name,
			CPUPercent: cpuPct,
			MemPercent: memPct,
			MemRSS:     rss,
		})
	}

	sort.Slice(briefs, func(i, j int) bool {
		return briefs[i].CPUPercent > briefs[j].CPUPercent
	})

	if len(briefs) > n {
		briefs = briefs[:n]
	}
	return briefs
}

func shouldSkipPartition(p disk.PartitionStat) bool {
	// 跳过虚拟文件系统
	skipFS := map[string]bool{
		"tmpfs": true, "devtmpfs": true, "devfs": true,
		"squashfs": true, "overlay": true, "autofs": true,
		"sysfs": true, "proc": true, "cgroup": true, "cgroup2": true,
	}
	if skipFS[p.Fstype] {
		return true
	}
	if runtime.GOOS == "darwin" {
		// macOS 跳过系统快照等
		if p.Mountpoint == "/System/Volumes/Data" {
			return false
		}
		if len(p.Mountpoint) > 1 && p.Mountpoint != "/" &&
			(len(p.Device) == 0 || p.Device[0] != '/') {
			return true
		}
	}
	return false
}
