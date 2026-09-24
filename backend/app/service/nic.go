package service

import (
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"xpanel/app/dto"
)

type nicSysfs struct {
	OperState   string
	Carrier     string
	Speed       string
	Duplex      string
	HasWireless bool
	HasPhy80211 bool
	HasBridge   bool
	HasBonding  bool
	HasVLAN     bool
	HasDevice   bool
	DevType     string
}

type rawIface struct {
	Name         string
	HardwareAddr string
	Flags        net.Flags
	Addrs        []string
}

var (
	nicNameSafe       = regexp.MustCompile(`^[A-Za-z0-9:._-]+$`)
	inventoryDrop     = regexp.MustCompile(`^(lo|docker\d*|docker_gwbridge|br-[0-9a-f]{6,}|veth|fwbr|fwln|fwpr|tap|tunl|tun\d*|cni|flannel|cali|virbr|vnet|dummy|tailscale)`)
	wifiBitrateLine   = regexp.MustCompile(`(?i)(?:rx |tx )?bitrate:\s*([\d.]+)\s*MBit`)
	ethtoolSpeedLine  = regexp.MustCompile(`(?i)^Speed:\s*(\d+)\s*Mb/s`)
	ethtoolDuplexLine = regexp.MustCompile(`(?i)^Duplex:\s*(Full|Half)\b`)
	lookupWifiSpeed   = wifiLinkSpeedMbps
	lookupEthtoolLink = ethtoolLink
)

func classifyKind(fs nicSysfs) string {
	if fs.HasWireless || fs.HasPhy80211 || fs.DevType == "wlan" {
		return "wifi"
	}
	if fs.HasBridge || fs.DevType == "bridge" {
		return "bridge"
	}
	if fs.HasBonding || fs.DevType == "bond" {
		return "bond"
	}
	if fs.HasVLAN || fs.DevType == "vlan" {
		return "vlan"
	}
	if fs.HasDevice {
		return "ethernet"
	}
	return "virtual"
}

func parseSpeedMbps(raw string) int {
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || n <= 0 {
		return 0
	}
	return n
}

func parseWifiBitrateMbps(raw string) int {
	max := 0.0
	for _, match := range wifiBitrateLine.FindAllStringSubmatch(raw, -1) {
		v, err := strconv.ParseFloat(match[1], 64)
		if err != nil || v <= 0 {
			continue
		}
		if v > max {
			max = v
		}
	}
	if max <= 0 {
		return 0
	}
	return int(max + 0.5)
}

func wifiLinkSpeedMbps(name string) int {
	if !nicNameSafe.MatchString(name) {
		return 0
	}
	out, err := exec.Command("iw", "dev", name, "link").Output()
	if err != nil {
		return 0
	}
	return parseWifiBitrateMbps(string(out))
}

func carrierConnected(raw string) bool {
	return strings.TrimSpace(raw) == "1"
}

func normalizeDuplex(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "full":
		return "full"
	case "half":
		return "half"
	default:
		return ""
	}
}

func parseEthtoolLink(raw string) (int, string) {
	speed := 0
	duplex := ""
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if match := ethtoolSpeedLine.FindStringSubmatch(line); match != nil {
			speed = parseSpeedMbps(match[1])
		}
		if match := ethtoolDuplexLine.FindStringSubmatch(line); match != nil {
			duplex = normalizeDuplex(match[1])
		}
	}
	return speed, duplex
}

func ethtoolLink(name string) (int, string) {
	if !nicNameSafe.MatchString(name) {
		return 0, ""
	}
	out, err := exec.Command("ethtool", name).Output()
	if err != nil {
		return 0, ""
	}
	return parseEthtoolLink(string(out))
}

func speedState(mbps int, connected bool) string {
	if mbps > 0 {
		return "negotiated"
	}
	if connected {
		return "unreported"
	}
	return "unnegotiated"
}

func isInventoryNIC(name string) bool {
	n := strings.ToLower(strings.TrimSpace(name))
	if n == "" {
		return false
	}
	return !inventoryDrop.MatchString(n)
}

func assembleInterface(name, mac string, flags net.Flags, addrs []string, fs nicSysfs) dto.InterfaceInfo {
	info := dto.InterfaceInfo{
		Name:      name,
		MAC:       mac,
		OperState: fs.OperState,
		Kind:      classifyKind(fs),
		SpeedMbps: parseSpeedMbps(fs.Speed),
		Duplex:    normalizeDuplex(fs.Duplex),
		Connected: carrierConnected(fs.Carrier),
	}
	info.SpeedState = speedState(info.SpeedMbps, info.Connected)
	if flags&net.FlagUp != 0 {
		info.Status = "up"
	} else {
		info.Status = "down"
	}
	for _, addr := range addrs {
		if strings.Contains(addr, ":") {
			info.IPv6 = append(info.IPv6, addr)
		} else {
			info.IPv4 = append(info.IPv4, addr)
		}
	}
	return info
}

func listNicsFrom(list func() ([]rawIface, error), sysfs func(string) nicSysfs, inventoryOnly bool) []dto.InterfaceInfo {
	ifaces, err := list()
	if err != nil {
		return nil
	}
	var result []dto.InterfaceInfo
	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		if inventoryOnly && !isInventoryNIC(iface.Name) {
			continue
		}
		info := assembleInterface(iface.Name, iface.HardwareAddr, iface.Flags, iface.Addrs, sysfs(iface.Name))
		if info.Kind == "wifi" && info.SpeedMbps == 0 && lookupWifiSpeed != nil {
			info.SpeedMbps = lookupWifiSpeed(iface.Name)
		}
		if info.SpeedMbps == 0 && info.Connected && lookupEthtoolLink != nil {
			mbps, duplex := lookupEthtoolLink(iface.Name)
			if mbps > 0 {
				info.SpeedMbps = mbps
			}
			if info.Duplex == "" && duplex != "" {
				info.Duplex = duplex
			}
		}
		info.SpeedState = speedState(info.SpeedMbps, info.Connected)
		result = append(result, info)
	}
	return result
}

func ListInventoryNics() []dto.InterfaceInfo {
	return listNicsFrom(readHostIfaces, readNicSysfs, true)
}

func readHostIfaces() ([]rawIface, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	out := make([]rawIface, 0, len(ifaces))
	for _, iface := range ifaces {
		raw := rawIface{
			Name:         iface.Name,
			HardwareAddr: iface.HardwareAddr.String(),
			Flags:        iface.Flags,
		}
		if addrs, err := iface.Addrs(); err == nil {
			for _, addr := range addrs {
				raw.Addrs = append(raw.Addrs, addr.String())
			}
		}
		out = append(out, raw)
	}
	return out, nil
}

func readNicSysfs(name string) nicSysfs {
	if !nicNameSafe.MatchString(name) {
		return nicSysfs{}
	}
	base := filepath.Join("/sys/class/net", name)
	fs := nicSysfs{
		OperState:   readSysfsTrim(filepath.Join(base, "operstate")),
		Carrier:     readSysfsTrim(filepath.Join(base, "carrier")),
		Speed:       readSysfsTrim(filepath.Join(base, "speed")),
		Duplex:      readSysfsTrim(filepath.Join(base, "duplex")),
		HasWireless: exists(filepath.Join(base, "wireless")),
		HasPhy80211: exists(filepath.Join(base, "phy80211")),
		HasBridge:   exists(filepath.Join(base, "bridge")),
		HasBonding:  exists(filepath.Join(base, "bonding")),
		HasDevice:   exists(filepath.Join(base, "device")),
		DevType:     ueventDevType(filepath.Join(base, "uevent")),
	}
	if exists(filepath.Join("/proc/net/vlan", name)) {
		fs.HasVLAN = true
	}
	if fs.DevType == "vlan" {
		fs.HasVLAN = true
	}
	return fs
}

func readSysfsTrim(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func ueventDevType(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "DEVTYPE=") {
			return strings.TrimSpace(strings.TrimPrefix(line, "DEVTYPE="))
		}
	}
	return ""
}
