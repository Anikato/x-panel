package service

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"xpanel/app/dto"

	"github.com/shirou/gopsutil/v4/disk"
)

type IDiskService interface {
	GetDiskInfo() ([]dto.PartitionInfo, error)
	MountRemote(req dto.RemoteMountRequest) error
	UnmountRemote(req dto.RemoteUnmountRequest) error
	ListRemoteMounts() ([]dto.RemoteMountInfo, error)
}

type DiskService struct{}

func NewIDiskService() IDiskService { return &DiskService{} }

func (s *DiskService) GetDiskInfo() ([]dto.PartitionInfo, error) {
	partitions, err := disk.Partitions(false)
	if err != nil {
		return nil, err
	}

	var result []dto.PartitionInfo
	seen := make(map[string]bool)

	for _, p := range partitions {
		if seen[p.Mountpoint] {
			continue
		}

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

func (s *DiskService) MountRemote(req dto.RemoteMountRequest) error {
	if err := os.MkdirAll(req.MountPoint, 0755); err != nil {
		return fmt.Errorf("failed to create mount point: %v", err)
	}

	var cmd *exec.Cmd
	switch req.Protocol {
	case "nfs":
		source := fmt.Sprintf("%s:%s", req.Server, req.SharePath)
		opts := "rw,soft,timeo=30"
		if req.Options != "" {
			opts = req.Options
		}
		cmd = exec.Command("mount", "-t", "nfs", "-o", opts, source, req.MountPoint)

	case "smb", "cifs":
		source := fmt.Sprintf("//%s/%s", req.Server, strings.TrimPrefix(req.SharePath, "/"))
		opts := "rw"
		if req.Username != "" {
			opts += fmt.Sprintf(",username=%s", req.Username)
			if req.Password != "" {
				opts += fmt.Sprintf(",password=%s", req.Password)
			}
		} else {
			opts += ",guest"
		}
		if req.Options != "" {
			opts += "," + req.Options
		}
		cmd = exec.Command("mount", "-t", "cifs", "-o", opts, source, req.MountPoint)

	default:
		return fmt.Errorf("unsupported protocol: %s", req.Protocol)
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("mount failed: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

func (s *DiskService) UnmountRemote(req dto.RemoteUnmountRequest) error {
	out, err := exec.Command("umount", req.MountPoint).CombinedOutput()
	if err != nil {
		return fmt.Errorf("umount failed: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

func (s *DiskService) ListRemoteMounts() ([]dto.RemoteMountInfo, error) {
	partitions, err := disk.Partitions(true)
	if err != nil {
		return nil, err
	}

	remoteFS := map[string]bool{
		"nfs": true, "nfs4": true, "cifs": true, "smb": true, "smbfs": true, "fuse.sshfs": true,
	}

	var result []dto.RemoteMountInfo
	for _, p := range partitions {
		if !remoteFS[p.Fstype] {
			continue
		}
		info := dto.RemoteMountInfo{
			Device:     p.Device,
			MountPoint: p.Mountpoint,
			FSType:     p.Fstype,
			Options:    strings.Join(p.Opts, ","),
		}
		if usage, err := disk.Usage(p.Mountpoint); err == nil {
			info.Total = usage.Total
			info.Used = usage.Used
			info.Free = usage.Free
			info.Percent = usage.UsedPercent
		}
		result = append(result, info)
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
