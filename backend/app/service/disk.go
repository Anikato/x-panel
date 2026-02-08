package service

import (
	"xpanel/app/dto"

	"github.com/shirou/gopsutil/v4/disk"
)

type IDiskService interface {
	GetDiskInfo() ([]dto.PartitionInfo, error)
}

type DiskService struct{}

func NewIDiskService() IDiskService { return &DiskService{} }

func (s *DiskService) GetDiskInfo() ([]dto.PartitionInfo, error) {
	partitions, err := disk.Partitions(false)
	if err != nil {
		return nil, err
	}

	var result []dto.PartitionInfo
	seen := make(map[string]bool) // 避免同一挂载点重复

	for _, p := range partitions {
		if seen[p.Mountpoint] {
			continue
		}

		// 跳过虚拟文件系统
		if isVirtualFS(p.Fstype) {
			continue
		}

		usage, err := disk.Usage(p.Mountpoint)
		if err != nil || usage.Total == 0 {
			continue
		}

		info := dto.PartitionInfo{
			Device:      p.Device,
			MountPoint:  p.Mountpoint,
			FSType:      p.Fstype,
			Total:       usage.Total,
			Used:        usage.Used,
			Free:        usage.Free,
			UsedPercent: usage.UsedPercent,
			InodesTotal: usage.InodesTotal,
			InodesUsed:  usage.InodesUsed,
			InodesFree:  usage.InodesFree,
		}
		result = append(result, info)
		seen[p.Mountpoint] = true
	}
	return result, nil
}

func isVirtualFS(fstype string) bool {
	virtual := map[string]bool{
		"tmpfs": true, "devtmpfs": true, "devfs": true,
		"squashfs": true, "overlay": true, "autofs": true,
		"sysfs": true, "proc": true, "cgroup": true, "cgroup2": true,
		"efivarfs": true, "debugfs": true, "tracefs": true,
		"securityfs": true, "pstore": true, "bpf": true,
		"hugetlbfs": true, "mqueue": true, "configfs": true,
		"fusectl": true, "fuse.portal": true,
	}
	return virtual[fstype]
}
