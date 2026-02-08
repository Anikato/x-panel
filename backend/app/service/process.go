package service

import (
	"fmt"
	"sort"
	"strings"
	"syscall"

	"xpanel/app/dto"

	"github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/process"
)

type IProcessService interface {
	ListProcesses(req dto.ProcessSearchReq) ([]dto.ProcessInfo, error)
	StopProcess(req dto.ProcessStopReq) error
	ListConnections() ([]dto.NetworkConnInfo, error)
}

type ProcessService struct{}

func NewIProcessService() IProcessService { return &ProcessService{} }

func (s *ProcessService) ListProcesses(req dto.ProcessSearchReq) ([]dto.ProcessInfo, error) {
	procs, err := process.Processes()
	if err != nil {
		return nil, err
	}

	var result []dto.ProcessInfo
	for _, p := range procs {
		info := dto.ProcessInfo{PID: p.Pid}

		name, _ := p.Name()
		info.Name = name

		status, _ := p.Status()
		if len(status) > 0 {
			info.Status = mapStatus(status[0])
		}

		username, _ := p.Username()
		info.Username = username

		cpuPct, _ := p.CPUPercent()
		info.CPUPercent = cpuPct

		memPct, _ := p.MemoryPercent()
		info.MemPercent = memPct

		memInfo, _ := p.MemoryInfo()
		if memInfo != nil {
			info.MemRSS = memInfo.RSS
		}

		createTime, _ := p.CreateTime()
		info.StartTime = createTime

		threads, _ := p.NumThreads()
		info.NumThreads = threads

		cmdline, _ := p.Cmdline()
		info.CmdLine = cmdline

		ppid, _ := p.Ppid()
		info.PPID = ppid

		// 过滤
		if req.PID > 0 && info.PID != req.PID {
			continue
		}
		if req.Name != "" && !strings.Contains(strings.ToLower(info.Name), strings.ToLower(req.Name)) {
			continue
		}
		if req.Username != "" && !strings.Contains(strings.ToLower(info.Username), strings.ToLower(req.Username)) {
			continue
		}
		if req.Status != "" && info.Status != req.Status {
			continue
		}

		result = append(result, info)
	}

	// 排序
	sortBy := req.SortBy
	if sortBy == "" {
		sortBy = "cpu"
	}
	sort.Slice(result, func(i, j int) bool {
		var less bool
		switch sortBy {
		case "cpu":
			less = result[i].CPUPercent > result[j].CPUPercent
		case "mem":
			less = result[i].MemPercent > result[j].MemPercent
		case "pid":
			less = result[i].PID < result[j].PID
		case "name":
			less = result[i].Name < result[j].Name
		default:
			less = result[i].CPUPercent > result[j].CPUPercent
		}
		if req.SortDesc {
			return !less
		}
		return less
	})

	return result, nil
}

func (s *ProcessService) StopProcess(req dto.ProcessStopReq) error {
	p, err := process.NewProcess(req.PID)
	if err != nil {
		return fmt.Errorf("process %d not found", req.PID)
	}

	switch req.Signal {
	case "kill":
		return p.SendSignal(syscall.SIGKILL)
	case "stop":
		return p.SendSignal(syscall.SIGSTOP)
	default: // "term" or empty
		return p.SendSignal(syscall.SIGTERM)
	}
}

func (s *ProcessService) ListConnections() ([]dto.NetworkConnInfo, error) {
	conns, err := net.Connections("inet")
	if err != nil {
		return nil, err
	}

	var result []dto.NetworkConnInfo
	nameCache := make(map[int32]string)

	for _, c := range conns {
		if c.Status == "NONE" || c.Status == "" {
			continue
		}
		info := dto.NetworkConnInfo{
			PID:        c.Pid,
			LocalAddr:  c.Laddr.IP,
			LocalPort:  c.Laddr.Port,
			RemoteAddr: c.Raddr.IP,
			RemotePort: c.Raddr.Port,
			Status:     c.Status,
		}
		switch c.Type {
		case syscall.SOCK_STREAM:
			if strings.Contains(c.Laddr.IP, ":") {
				info.Protocol = "tcp6"
			} else {
				info.Protocol = "tcp"
			}
		case syscall.SOCK_DGRAM:
			if strings.Contains(c.Laddr.IP, ":") {
				info.Protocol = "udp6"
			} else {
				info.Protocol = "udp"
			}
		}

		if c.Pid > 0 {
			if name, ok := nameCache[c.Pid]; ok {
				info.Name = name
			} else {
				p, err := process.NewProcess(c.Pid)
				if err == nil {
					n, _ := p.Name()
					info.Name = n
					nameCache[c.Pid] = n
				}
			}
		}
		result = append(result, info)
	}
	return result, nil
}

func mapStatus(s string) string {
	switch s {
	case "R", "running":
		return "running"
	case "S", "sleeping", "idle":
		return "sleeping"
	case "T", "stopped":
		return "stopped"
	case "Z", "zombie":
		return "zombie"
	default:
		return s
	}
}
