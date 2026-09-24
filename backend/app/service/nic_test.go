package service

import (
	"net"
	"os"
	"testing"

	"xpanel/app/dto"
)

func TestMain(m *testing.M) {
	lookupWifiSpeed = func(string) int { return 0 }
	lookupEthtoolLink = func(string) (int, string) { return 0, "" }
	os.Exit(m.Run())
}

func TestParseEthtoolLink(t *testing.T) {
	raw := "Settings for eth0:\n\tSpeed: 10000Mb/s\n\tDuplex: Full\n"
	mbps, duplex := parseEthtoolLink(raw)
	if mbps != 10000 || duplex != "full" {
		t.Fatalf("ethtool = %d %q", mbps, duplex)
	}
	mbps, duplex = parseEthtoolLink("Speed: Unknown!\nDuplex: Unknown! (255)\n")
	if mbps != 0 || duplex != "" {
		t.Fatalf("unknown ethtool = %d %q", mbps, duplex)
	}
}

func TestListNics_UsesEthtoolWhenVirtualSysfsSpeedMissing(t *testing.T) {
	prev := lookupEthtoolLink
	lookupEthtoolLink = func(name string) (int, string) {
		if name == "eth0" {
			return 10000, "full"
		}
		return 0, ""
	}
	defer func() { lookupEthtoolLink = prev }()

	listed := listNicsFrom(func() ([]rawIface, error) {
		return []rawIface{{Name: "eth0", Flags: net.FlagUp}}, nil
	}, func(string) nicSysfs {
		return nicSysfs{OperState: "up", Carrier: "1", Speed: "-1", Duplex: "unknown", HasDevice: true}
	}, true)
	if len(listed) != 1 || listed[0].SpeedMbps != 10000 || listed[0].Duplex != "full" || listed[0].SpeedState != "negotiated" {
		t.Fatalf("virtio speed = %#v", listed)
	}
}

func TestListNics_ConnectedWithoutSpeedIsUnreported(t *testing.T) {
	listed := listNicsFrom(func() ([]rawIface, error) {
		return []rawIface{{Name: "eth0", Flags: net.FlagUp}}, nil
	}, func(string) nicSysfs {
		return nicSysfs{OperState: "up", Carrier: "1", Speed: "-1", HasDevice: true}
	}, true)
	if len(listed) != 1 || listed[0].SpeedMbps != 0 || listed[0].SpeedState != "unreported" {
		t.Fatalf("missing virtual speed = %#v", listed)
	}
}

func TestListNics_UsesWifiBitrateWhenSysfsSpeedMissing(t *testing.T) {
	prev := lookupWifiSpeed
	lookupWifiSpeed = func(name string) int {
		if name == "wlan0" {
			return 2402
		}
		return 0
	}
	defer func() { lookupWifiSpeed = prev }()

	listed := listNicsFrom(func() ([]rawIface, error) {
		return []rawIface{{Name: "wlan0", Flags: net.FlagUp}}, nil
	}, func(string) nicSysfs {
		return nicSysfs{HasWireless: true, HasDevice: true, Carrier: "1", OperState: "up"}
	}, true)
	if len(listed) != 1 || listed[0].SpeedMbps != 2402 {
		t.Fatalf("wifi speed = %#v", listed)
	}
}

func TestClassifyKind_WifiFromWirelessDir(t *testing.T) {
	got := classifyKind(nicSysfs{HasWireless: true})
	if got != "wifi" {
		t.Fatalf("kind = %q, want wifi", got)
	}
}

func TestClassifyKind_WifiFromPhy80211(t *testing.T) {
	got := classifyKind(nicSysfs{HasPhy80211: true})
	if got != "wifi" {
		t.Fatalf("kind = %q, want wifi", got)
	}
}

func TestClassifyKind_BridgeBondVlan(t *testing.T) {
	cases := []struct {
		fs   nicSysfs
		want string
	}{
		{nicSysfs{HasBridge: true}, "bridge"},
		{nicSysfs{HasBonding: true}, "bond"},
		{nicSysfs{HasVLAN: true}, "vlan"},
		{nicSysfs{DevType: "vlan"}, "vlan"},
	}
	for _, tc := range cases {
		if got := classifyKind(tc.fs); got != tc.want {
			t.Fatalf("kind = %q, want %q for %+v", got, tc.want, tc.fs)
		}
	}
}

func TestClassifyKind_PhysicalEthernet(t *testing.T) {
	got := classifyKind(nicSysfs{HasDevice: true})
	if got != "ethernet" {
		t.Fatalf("kind = %q, want ethernet", got)
	}
}

func TestClassifyKind_NoDeviceIsVirtual(t *testing.T) {
	got := classifyKind(nicSysfs{})
	if got != "virtual" {
		t.Fatalf("kind = %q, want virtual", got)
	}
}

func TestParseWifiBitrateMbps(t *testing.T) {
	sample := `Connected to 02:00:00:12:60:b8 (on wlp0s20f3)
SSID: Considerra_X
rx bitrate: 2401.9 MBit/s 160MHz HE-MCS 11 HE-NSS 2
tx bitrate: 2401.9 MBit/s 160MHz HE-MCS 11 HE-NSS 2
`
	if got := parseWifiBitrateMbps(sample); got != 2402 {
		t.Fatalf("bitrate = %d, want 2402", got)
	}
	if got := parseWifiBitrateMbps("Not connected."); got != 0 {
		t.Fatalf("disconnected bitrate = %d, want 0", got)
	}
	if got := parseWifiBitrateMbps("tx bitrate: 6.0 MBit/s\nrx bitrate: 144.4 MBit/s"); got != 144 {
		t.Fatalf("max bitrate = %d, want 144", got)
	}
}

func TestParseSpeedMbps(t *testing.T) {
	cases := map[string]int{
		"1000": 1000,
		"100":  100,
		"2500": 2500,
		"-1":   0,
		"":     0,
		"0":    0,
		"foo":  0,
	}
	for in, want := range cases {
		if got := parseSpeedMbps(in); got != want {
			t.Fatalf("parseSpeedMbps(%q) = %d, want %d", in, got, want)
		}
	}
}

func TestCarrierConnected(t *testing.T) {
	if !carrierConnected("1") {
		t.Fatal("carrier 1 should be connected")
	}
	if carrierConnected("0") {
		t.Fatal("carrier 0 should be disconnected")
	}
	if carrierConnected("") {
		t.Fatal("missing carrier should not claim connected")
	}
}

func TestAssembleInterface_DisconnectedPhysical(t *testing.T) {
	info := assembleInterface("eth0", "aa:bb:cc:dd:ee:ff", net.FlagUp, nil, nicSysfs{
		OperState: "down",
		Carrier:   "0",
		Speed:     "-1",
		Duplex:    "unknown",
		HasDevice: true,
	})
	if info.Name != "eth0" {
		t.Fatalf("name = %q", info.Name)
	}
	if info.Status != "up" {
		t.Fatalf("admin status = %q, want up", info.Status)
	}
	if info.Connected {
		t.Fatal("unplugged nic must not be connected")
	}
	if info.Kind != "ethernet" {
		t.Fatalf("kind = %q, want ethernet", info.Kind)
	}
	if info.SpeedMbps != 0 {
		t.Fatalf("speed = %d, want 0 when unnegotiated", info.SpeedMbps)
	}
	if info.SpeedState != "unnegotiated" {
		t.Fatalf("speed state = %q", info.SpeedState)
	}
}

func TestAssembleInterface_ConnectedGigabit(t *testing.T) {
	info := assembleInterface("enp1s0", "11:22:33:44:55:66", net.FlagUp, []string{"192.168.1.10/24", "fe80::1/64"}, nicSysfs{
		OperState: "up",
		Carrier:   "1",
		Speed:     "1000",
		Duplex:    "full",
		HasDevice: true,
	})
	if !info.Connected {
		t.Fatal("carrier up should be connected")
	}
	if info.SpeedMbps != 1000 || info.SpeedState != "negotiated" {
		t.Fatalf("speed = %d state=%q, want 1000 negotiated", info.SpeedMbps, info.SpeedState)
	}
	if info.Duplex != "full" {
		t.Fatalf("duplex = %q, want full", info.Duplex)
	}
	if len(info.IPv4) != 1 || info.IPv4[0] != "192.168.1.10/24" {
		t.Fatalf("ipv4 = %#v", info.IPv4)
	}
	if len(info.IPv6) != 1 {
		t.Fatalf("ipv6 = %#v", info.IPv6)
	}
}

func TestIsInventoryNIC(t *testing.T) {
	keep := []string{"eth0", "enp1s0", "wlan0", "wlp2s0", "vmbr0", "eno1"}
	drop := []string{"lo", "docker0", "veth1234", "br-ab12cd34ef56", "tailscale0", "tun0"}
	for _, name := range keep {
		if !isInventoryNIC(name) {
			t.Fatalf("should keep %q", name)
		}
	}
	for _, name := range drop {
		if isInventoryNIC(name) {
			t.Fatalf("should drop %q", name)
		}
	}
}

func TestListNics_SkipsLoopbackAndNoise(t *testing.T) {
	listed := listNicsFrom(func() ([]rawIface, error) {
		return []rawIface{
			{Name: "lo", Flags: net.FlagLoopback | net.FlagUp},
			{Name: "eth0", HardwareAddr: "aa:bb:cc:dd:ee:ff", Flags: net.FlagUp, Addrs: []string{"10.0.0.2/24"}},
			{Name: "docker0", Flags: net.FlagUp, Addrs: []string{"172.17.0.1/16"}},
			{Name: "wlan0", HardwareAddr: "de:ad:be:ef:00:01", Flags: 0},
		}, nil
	}, func(name string) nicSysfs {
		switch name {
		case "eth0":
			return nicSysfs{OperState: "up", Carrier: "1", Speed: "1000", Duplex: "full", HasDevice: true}
		case "wlan0":
			return nicSysfs{OperState: "down", Carrier: "0", HasWireless: true, HasDevice: true}
		default:
			return nicSysfs{}
		}
	}, true)
	if len(listed) != 2 {
		t.Fatalf("listed %d nics, want 2: %#v", len(listed), namesOf(listed))
	}
	if listed[0].Name != "eth0" || listed[1].Name != "wlan0" {
		t.Fatalf("order/names = %#v", namesOf(listed))
	}
	if listed[1].Kind != "wifi" {
		t.Fatalf("wlan kind = %q, want wifi", listed[1].Kind)
	}
}

func TestListInventoryNics_ClassifiesSysfsWifi(t *testing.T) {
	entries, err := os.ReadDir("/sys/class/net")
	if err != nil {
		t.Skip(err.Error())
	}
	wifiName := ""
	for _, entry := range entries {
		name := entry.Name()
		if !isInventoryNIC(name) {
			continue
		}
		if _, err := os.Stat("/sys/class/net/" + name + "/phy80211"); err == nil {
			wifiName = name
			break
		}
	}
	if wifiName == "" {
		t.Skip("no inventory wifi nic")
	}
	nics := ListInventoryNics()
	var wifi *dto.InterfaceInfo
	for i := range nics {
		if nics[i].Name == wifiName {
			wifi = &nics[i]
			break
		}
	}
	if wifi == nil {
		t.Fatalf("wifi nic %s missing from inventory: %v", wifiName, namesOf(nics))
	}
	if wifi.Kind != "wifi" {
		t.Fatalf("kind = %q, want wifi", wifi.Kind)
	}
}

func namesOf(list []dto.InterfaceInfo) []string {
	out := make([]string, len(list))
	for i, n := range list {
		out[i] = n.Name
	}
	return out
}
