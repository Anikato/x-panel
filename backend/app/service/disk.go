package service

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"xpanel/app/dto"

	"github.com/shirou/gopsutil/v4/disk"
)

type IDiskService interface {
	GetDiskInfo() ([]dto.PartitionInfo, error)
	BrowseShares(req dto.BrowseSharesRequest) ([]string, error)
	MountRemote(req dto.RemoteMountRequest) error
	UnmountRemote(req dto.RemoteUnmountRequest) error
	ListRemoteMounts() ([]dto.RemoteMountInfo, error)
	ListBlockDevices() ([]dto.BlockDevice, error)
	MountLocal(req dto.LocalMountRequest) error
	UnmountLocal(req dto.LocalUnmountRequest) error
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

// BrowseShares 列出远程服务器上可用的共享/导出路径
func (s *DiskService) BrowseShares(req dto.BrowseSharesRequest) ([]string, error) {
	switch req.Protocol {
	case "nfs":
		return browseNFSExports(req.Server)
	case "smb", "cifs":
		return browseSMBShares(req.Server, req.Username, req.Password)
	default:
		return nil, fmt.Errorf("unsupported protocol: %s", req.Protocol)
	}
}

// browseNFSExports 使用 showmount -e 列出 NFS 导出路径
func browseNFSExports(server string) ([]string, error) {
	if _, err := exec.LookPath("showmount"); err != nil {
		return nil, fmt.Errorf("showmount not found, please install nfs-common: %v", err)
	}
	out, err := exec.Command("showmount", "-e", "--no-headers", server).Output()
	if err != nil {
		// 某些发行版不支持 --no-headers，退而求其次
		out, err = exec.Command("showmount", "-e", server).Output()
		if err != nil {
			return nil, fmt.Errorf("showmount failed: %s", strings.TrimSpace(string(out)))
		}
	}
	var exports []string
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(strings.ToLower(line), "export list") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 1 && strings.HasPrefix(fields[0], "/") {
			exports = append(exports, fields[0])
		}
	}
	if len(exports) == 0 {
		return nil, fmt.Errorf("no NFS exports found on %s", server)
	}
	return exports, nil
}

// browseSMBShares 使用 smbclient -gL 列出 SMB/CIFS 共享名
func browseSMBShares(server, username, password string) ([]string, error) {
	if _, err := exec.LookPath("smbclient"); err != nil {
		return nil, fmt.Errorf("smbclient not found, please install samba-client: %v", err)
	}
	args := []string{"-gL", "//" + server}
	if username != "" {
		creds := username
		if password != "" {
			creds += "%" + password
		}
		args = append(args, "-U", creds)
	} else {
		args = append(args, "-N")
	}
	// 设置超时，避免长时间等待
	args = append(args, "--timeout=10")
	out, _ := exec.Command("smbclient", args...).CombinedOutput()

	var shares []string
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// -g 格式：Type|Name|Comment，取 Disk 类型
		parts := strings.SplitN(line, "|", 3)
		if len(parts) >= 2 && strings.EqualFold(parts[0], "Disk") {
			name := strings.TrimSpace(parts[1])
			if name != "" {
				shares = append(shares, name)
			}
		}
	}
	if len(shares) == 0 {
		return nil, fmt.Errorf("no SMB shares found on %s (check credentials or server availability)", server)
	}
	return shares, nil
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

// --- Block device helpers ---

type lsblkOutput struct {
	Blockdevices []lsblkDevice `json:"blockdevices"`
}

type lsblkDevice struct {
	Name       string        `json:"name"`
	Size       json.Number   `json:"size"`
	Fstype     *string       `json:"fstype"`
	Mountpoint *string       `json:"mountpoint"`
	Type       string        `json:"type"`
	Model      *string       `json:"model"`
	Children   []lsblkDevice `json:"children,omitempty"`
}

func toLsblkDTO(d lsblkDevice) dto.BlockDevice {
	bd := dto.BlockDevice{
		Name: d.Name,
		Type: d.Type,
	}
	if s, err := d.Size.Int64(); err == nil {
		bd.Size = uint64(s)
	}
	if d.Fstype != nil {
		bd.FSType = *d.Fstype
	}
	if d.Mountpoint != nil {
		bd.MountPoint = *d.Mountpoint
	}
	if d.Model != nil {
		bd.Model = strings.TrimSpace(*d.Model)
	}
	for _, ch := range d.Children {
		bd.Children = append(bd.Children, toLsblkDTO(ch))
	}
	return bd
}

func (s *DiskService) ListBlockDevices() ([]dto.BlockDevice, error) {
	out, err := exec.Command("lsblk", "-Jb", "-o", "NAME,SIZE,FSTYPE,MOUNTPOINT,TYPE,MODEL").Output()
	if err != nil {
		return nil, fmt.Errorf("lsblk failed: %v", err)
	}
	var parsed lsblkOutput
	if err := json.Unmarshal(out, &parsed); err != nil {
		return nil, fmt.Errorf("parse lsblk output: %v", err)
	}
	var result []dto.BlockDevice
	for _, d := range parsed.Blockdevices {
		if d.Type == "rom" || d.Type == "loop" {
			continue
		}
		result = append(result, toLsblkDTO(d))
	}
	return result, nil
}

func (s *DiskService) MountLocal(req dto.LocalMountRequest) error {
	device := req.Device
	if !strings.HasPrefix(device, "/dev/") {
		device = "/dev/" + device
	}
	mountPoint := filepath.Clean(req.MountPoint)

	if err := os.MkdirAll(mountPoint, 0755); err != nil {
		return fmt.Errorf("failed to create mount point: %v", err)
	}

	args := []string{}
	if req.FSType != "" {
		args = append(args, "-t", req.FSType)
	}
	args = append(args, device, mountPoint)

	out, err := exec.Command("mount", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("mount failed: %s", strings.TrimSpace(string(out)))
	}

	if req.Persist {
		fsType := req.FSType
		if fsType == "" {
			fsType = "auto"
		}
		if err := addFstabEntry(device, mountPoint, fsType, "defaults"); err != nil {
			return fmt.Errorf("mount succeeded but fstab write failed: %v", err)
		}
	}
	return nil
}

func (s *DiskService) UnmountLocal(req dto.LocalUnmountRequest) error {
	mountPoint := filepath.Clean(req.MountPoint)
	out, err := exec.Command("umount", mountPoint).CombinedOutput()
	if err != nil {
		out2, err2 := exec.Command("umount", "-l", mountPoint).CombinedOutput()
		if err2 != nil {
			return fmt.Errorf("umount failed: %s", strings.TrimSpace(string(out)))
		}
		_ = out2
	}
	if req.RemoveFstab {
		_ = removeFstabEntry(mountPoint)
	}
	return nil
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
