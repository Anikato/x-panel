package service

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"xpanel/app/dto"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	hostUtil "github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"
	gnet "github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/process"
)

type IMonitorService interface {
	GetCurrentStats() (*dto.SystemStats, error)
}

type MonitorService struct{}

func NewIMonitorService() IMonitorService { return &MonitorService{} }

var (
	lastNetIO   []gnet.IOCountersStat
	lastNetTime time.Time
	netMu       sync.Mutex

	cachedPublicIPv4 string
	cachedPublicIPv6 string
	publicIPMu       sync.Mutex
	publicIPLastTime time.Time
)

func (s *MonitorService) GetCurrentStats() (*dto.SystemStats, error) {
	stats := &dto.SystemStats{}

	hostInfo, err := hostUtil.Info()
	if err == nil {
		stats.Host = dto.SystemHostInfo{
			Hostname:        hostInfo.Hostname,
			OS:              hostInfo.OS,
			Platform:        hostInfo.Platform,
			PlatformVersion: hostInfo.PlatformVersion,
			KernelVersion:   hostInfo.KernelVersion,
			KernelArch:      hostInfo.KernelArch,
			Virtualization:  detectVirtualization(hostInfo.VirtualizationSystem),
		}
	}

	stats.Host.Timezone = getTimezone()
	stats.Host.DNSServers = getDNSServers()
	stats.Host.Interfaces = getNetworkInterfaces()

	ipv4, ipv6 := getCachedPublicIP()
	stats.Host.PublicIPv4 = ipv4
	stats.Host.PublicIPv6 = ipv6

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

	// 负载
	loadStat, err := load.Avg()
	if err == nil {
		stats.Load = dto.LoadStats{
			Load1:  loadStat.Load1,
			Load5:  loadStat.Load5,
			Load15: loadStat.Load15,
		}
	}

	// 磁盘
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

	// 网络
	netIO, err := gnet.IOCounters(true)
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

	stats.TopProcess = getTopProcesses(5)

	uptime, _ := hostUtil.Uptime()
	stats.Uptime = uptime

	return stats, nil
}

// getNetworkInterfaces 获取所有网卡信息（IP/MAC/状态）
func getNetworkInterfaces() []dto.InterfaceInfo {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
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
	return result
}

// getCachedPublicIP 获取公网 IP（缓存 5 分钟）
func getCachedPublicIP() (string, string) {
	publicIPMu.Lock()
	defer publicIPMu.Unlock()

	if time.Since(publicIPLastTime) < 5*time.Minute && (cachedPublicIPv4 != "" || cachedPublicIPv6 != "") {
		return cachedPublicIPv4, cachedPublicIPv6
	}

	var wg sync.WaitGroup
	var ipv4, ipv6 string

	wg.Add(1)
	go func() {
		defer wg.Done()
		ipv4 = fetchPublicIP("https://api.ipify.org", 3*time.Second)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		ipv6 = fetchPublicIP("https://api6.ipify.org", 3*time.Second)
	}()
	wg.Wait()

	cachedPublicIPv4 = ipv4
	cachedPublicIPv6 = ipv6
	publicIPLastTime = time.Now()
	return ipv4, ipv6
}

func fetchPublicIP(url string, timeout time.Duration) string {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return ""
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 128))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(body))
}

// getTimezone 获取系统时区
func getTimezone() string {
	zone, _ := time.Now().Zone()
	loc := time.Now().Location()
	if loc != nil && loc.String() != "Local" {
		return fmt.Sprintf("%s (%s)", loc.String(), zone)
	}
	// Linux: 从 /etc/timezone 或 timedatectl 读取
	if data, err := os.ReadFile("/etc/timezone"); err == nil {
		tz := strings.TrimSpace(string(data))
		if tz != "" {
			return fmt.Sprintf("%s (%s)", tz, zone)
		}
	}
	return zone
}

// getDNSServers 读取 /etc/resolv.conf 中的 DNS 服务器
func getDNSServers() []string {
	f, err := os.Open("/etc/resolv.conf")
	if err != nil {
		return nil
	}
	defer f.Close()

	var servers []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "nameserver") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				servers = append(servers, fields[1])
			}
		}
	}
	return servers
}

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

// detectVirtualization 增强虚拟化类型检测，gopsutil 结果为空时使用回退方法
func detectVirtualization(gopsutilResult string) string {
	if gopsutilResult != "" {
		return gopsutilResult
	}
	// 回退 1: systemd-detect-virt
	if out, err := exec.Command("systemd-detect-virt").Output(); err == nil {
		virt := strings.TrimSpace(string(out))
		if virt != "" && virt != "none" {
			return virt
		}
	}
	// 回退 2: 检查 DMI 产品名
	if data, err := os.ReadFile("/sys/class/dmi/id/product_name"); err == nil {
		product := strings.TrimSpace(string(data))
		lower := strings.ToLower(product)
		switch {
		case strings.Contains(lower, "virtualbox"):
			return "virtualbox"
		case strings.Contains(lower, "vmware"):
			return "vmware"
		case strings.Contains(lower, "kvm"), strings.Contains(lower, "qemu"):
			return "kvm"
		case strings.Contains(lower, "hyper-v"):
			return "hyperv"
		case strings.Contains(lower, "xen"):
			return "xen"
		case strings.Contains(lower, "parallels"):
			return "parallels"
		case product != "":
			return product
		}
	}
	// 回退 3: 检查 /proc/cpuinfo 中的虚拟化标记
	if data, err := os.ReadFile("/proc/cpuinfo"); err == nil {
		content := strings.ToLower(string(data))
		if strings.Contains(content, "hypervisor") {
			return "virtual"
		}
	}
	return ""
}

func shouldSkipPartition(p disk.PartitionStat) bool {
	skipFS := map[string]bool{
		"tmpfs": true, "devtmpfs": true, "devfs": true,
		"squashfs": true, "overlay": true, "autofs": true,
		"sysfs": true, "proc": true, "cgroup": true, "cgroup2": true,
	}
	if skipFS[p.Fstype] {
		return true
	}
	if runtime.GOOS == "darwin" {
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
