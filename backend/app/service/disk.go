package service

import (
	"bufio"
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

var nfsPresets = map[string]string{
	"default":  "rw,soft,timeo=30,retrans=3",
	"unstable": "rw,soft,timeo=10,retrans=2,actimeo=60,noatime",
	"lan":      "rw,hard,timeo=600,retrans=5,rsize=1048576,wsize=1048576",
}

var cifsPresets = map[string]string{
	"default":  "rw,soft,echo_interval=10,actimeo=30",
	"unstable": "rw,soft,echo_interval=5,actimeo=30,cache=loose,nobrl,noserverino",
	"lan":      "rw,hard,cache=strict,rsize=4194304,wsize=4194304",
}

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
	var fstabFS, fstabSource, fstabOpts string

	switch req.Protocol {
	case "nfs":
		source := fmt.Sprintf("%s:%s", req.Server, req.SharePath)
		opts := resolveOptions(req.Options, req.Preset, nfsPresets)
		cmd = exec.Command("mount", "-t", "nfs", "-o", opts, source, req.MountPoint)
		fstabFS = "nfs"
		fstabSource = source
		fstabOpts = opts

	case "smb", "cifs":
		source := fmt.Sprintf("//%s/%s", req.Server, strings.TrimPrefix(req.SharePath, "/"))
		opts := resolveOptions(req.Options, req.Preset, cifsPresets)
		if req.Username != "" {
			opts += fmt.Sprintf(",username=%s", req.Username)
			if req.Password != "" {
				opts += fmt.Sprintf(",password=%s", req.Password)
			}
		} else {
			opts += ",guest"
		}
		cmd = exec.Command("mount", "-t", "cifs", "-o", opts, source, req.MountPoint)
		fstabFS = "cifs"
		fstabSource = source
		fstabOpts = opts

	default:
		return fmt.Errorf("unsupported protocol: %s", req.Protocol)
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("mount failed: %s", strings.TrimSpace(string(out)))
	}

	if req.Persist {
		if err := addFstabEntry(fstabSource, req.MountPoint, fstabFS, fstabOpts); err != nil {
			return fmt.Errorf("mount succeeded but fstab write failed: %v", err)
		}
	}

	return nil
}

func (s *DiskService) UnmountRemote(req dto.RemoteUnmountRequest) error {
	out, err := exec.Command("umount", req.MountPoint).CombinedOutput()
	if err != nil {
		out2, err2 := exec.Command("umount", "-l", req.MountPoint).CombinedOutput()
		if err2 != nil {
			return fmt.Errorf("umount failed: %s", strings.TrimSpace(string(out)))
		}
		_ = out2
	}
	if req.RemoveFstab {
		_ = removeFstabEntry(req.MountPoint)
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

	fstabEntries := loadFstabMountPoints()

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
			InFstab:    fstabEntries[p.Mountpoint],
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

func resolveOptions(custom, preset string, presets map[string]string) string {
	if custom != "" {
		return custom
	}
	if p, ok := presets[preset]; ok {
		return p
	}
	return presets["default"]
}

// --- fstab helpers ---

const fstabPath = "/etc/fstab"
const fstabTag = "# xpanel-managed"

func addFstabEntry(source, mountPoint, fsType, opts string) error {
	if hasFstabEntry(mountPoint) {
		return nil
	}
	f, err := os.OpenFile(fstabPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	line := fmt.Sprintf("%s\t%s\t%s\t%s\t0\t0\t%s\n", source, mountPoint, fsType, opts, fstabTag)
	_, err = f.WriteString(line)
	return err
}

func removeFstabEntry(mountPoint string) error {
	lines, err := readFstabLines()
	if err != nil {
		return err
	}
	var kept []string
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[1] == mountPoint {
			continue
		}
		kept = append(kept, line)
	}
	return os.WriteFile(fstabPath, []byte(strings.Join(kept, "\n")+"\n"), 0644)
}

func hasFstabEntry(mountPoint string) bool {
	lines, err := readFstabLines()
	if err != nil {
		return false
	}
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[1] == mountPoint {
			return true
		}
	}
	return false
}

func loadFstabMountPoints() map[string]bool {
	result := make(map[string]bool)
	lines, err := readFstabLines()
	if err != nil {
		return result
	}
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			result[fields[1]] = true
		}
	}
	return result
}

func readFstabLines() ([]string, error) {
	f, err := os.Open(fstabPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			lines = append(lines, line)
			continue
		}
		lines = append(lines, line)
	}
	return lines, scanner.Err()
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
