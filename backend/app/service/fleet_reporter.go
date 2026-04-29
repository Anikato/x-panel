package service

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"xpanel/app/version"
	"xpanel/global"

	"github.com/shirou/gopsutil/v4/cpu"
	hostUtil "github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"
)

const (
	fleetDefaultEndpoint = "https://fcapi.qm.mk"
	fleetReportInterval  = 30 * time.Minute
	fleetStartupDelay    = 15 * time.Second

	fleetSettingEnabled       = "FleetEnabled"
	fleetSettingEndpoint      = "FleetEndpoint"
	fleetSettingInstanceID    = "FleetInstanceID"
	fleetSettingInstanceToken = "FleetInstanceToken"
)

type IFleetReporterService interface {
	Start()
}

type FleetReporterService struct {
	client *http.Client
}

func NewIFleetReporterService() IFleetReporterService {
	return &FleetReporterService{
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

type fleetPayload struct {
	InstanceID string             `json:"instanceId"`
	Panel      fleetPanelPayload  `json:"panel"`
	Host       fleetHostPayload   `json:"host"`
	CPU        fleetCPUPayload    `json:"cpu"`
	Memory     fleetMemoryPayload `json:"memory"`
}

type fleetPanelPayload struct {
	Version    string `json:"version"`
	CommitHash string `json:"commitHash"`
	BuildTime  string `json:"buildTime"`
	GoVersion  string `json:"goVersion"`
}

type fleetHostPayload struct {
	Hostname        string `json:"hostname"`
	OS              string `json:"os"`
	Platform        string `json:"platform"`
	PlatformVersion string `json:"platformVersion"`
	KernelVersion   string `json:"kernelVersion"`
	KernelArch      string `json:"kernelArch"`
	Uptime          uint64 `json:"uptime"`
	Timezone        string `json:"timezone"`
	Virtualization  string `json:"virtualization"`
	TCPCongestion   string `json:"tcpCongestion"`
}

type fleetCPUPayload struct {
	ModelName    string `json:"modelName"`
	Cores        int    `json:"cores"`
	LogicalCores int    `json:"logicalCores"`
}

type fleetMemoryPayload struct {
	Total uint64 `json:"total"`
}

type fleetResponse struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

type fleetRegisterData struct {
	InstanceID    string    `json:"instanceId"`
	InstanceToken string    `json:"instanceToken"`
	ServerTime    time.Time `json:"serverTime"`
}

func (s *FleetReporterService) Start() {
	go func() {
		time.Sleep(fleetStartupDelay)
		s.reportOnce()

		ticker := time.NewTicker(fleetReportInterval)
		defer ticker.Stop()
		for range ticker.C {
			s.reportOnce()
		}
	}()
}

func (s *FleetReporterService) reportOnce() {
	enabled := s.settingValue(fleetSettingEnabled, "enable")
	if strings.EqualFold(enabled, "disable") {
		return
	}

	endpoint := strings.TrimRight(s.settingValue(fleetSettingEndpoint, fleetDefaultEndpoint), "/")
	if endpoint == "" {
		endpoint = fleetDefaultEndpoint
	}

	instanceID := s.settingValue(fleetSettingInstanceID, "")
	if instanceID == "" {
		instanceID = newFleetInstanceID()
		if err := settingRepo.CreateOrUpdate(fleetSettingInstanceID, instanceID); err != nil {
			global.LOG.Debugf("fleet reporter save instance id failed: %v", err)
			return
		}
	}

	payload, err := buildFleetPayload(instanceID)
	if err != nil {
		global.LOG.Debugf("fleet reporter build payload failed: %v", err)
		return
	}

	token := s.settingValue(fleetSettingInstanceToken, "")
	if token == "" {
		if err := s.register(endpoint, payload); err != nil {
			global.LOG.Debugf("fleet reporter register failed: %v", err)
		}
		return
	}

	if err := s.heartbeat(endpoint, payload, token); err != nil {
		global.LOG.Debugf("fleet reporter heartbeat failed: %v", err)
		if strings.Contains(err.Error(), "401") || strings.Contains(err.Error(), "404") {
			_ = settingRepo.CreateOrUpdate(fleetSettingInstanceToken, "")
			if err := s.register(endpoint, payload); err != nil {
				global.LOG.Debugf("fleet reporter re-register failed: %v", err)
			}
		}
	}
}

func (s *FleetReporterService) register(endpoint string, payload fleetPayload) error {
	var data fleetRegisterData
	if err := s.postJSON(endpoint+"/api/v1/fleet/register", "", payload, &data); err != nil {
		return err
	}
	if data.InstanceToken == "" {
		return fmt.Errorf("empty instance token")
	}
	if data.InstanceID != "" && data.InstanceID != payload.InstanceID {
		return fmt.Errorf("unexpected instance id: %s", data.InstanceID)
	}
	return settingRepo.CreateOrUpdate(fleetSettingInstanceToken, data.InstanceToken)
}

func (s *FleetReporterService) heartbeat(endpoint string, payload fleetPayload, token string) error {
	return s.postJSON(endpoint+"/api/v1/fleet/heartbeat", token, payload, nil)
}

func (s *FleetReporterService) postJSON(url, token string, payload fleetPayload, out interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "X-Panel Fleet Reporter")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var envelope fleetResponse
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 || envelope.Code < 200 || envelope.Code >= 300 {
		return fmt.Errorf("fleet server returned http=%d code=%d message=%s", resp.StatusCode, envelope.Code, envelope.Message)
	}
	if out != nil && len(envelope.Data) > 0 {
		if err := json.Unmarshal(envelope.Data, out); err != nil {
			return err
		}
	}
	return nil
}

func (s *FleetReporterService) settingValue(key, fallback string) string {
	value, err := settingRepo.GetValueByKey(key)
	if err != nil || strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}

func buildFleetPayload(instanceID string) (fleetPayload, error) {
	info := version.Get()
	payload := fleetPayload{
		InstanceID: instanceID,
		Panel: fleetPanelPayload{
			Version:    info.Version,
			CommitHash: info.CommitHash,
			BuildTime:  info.BuildTime,
			GoVersion:  info.GoVersion,
		},
	}

	if hostInfo, err := hostUtil.Info(); err == nil {
		payload.Host = fleetHostPayload{
			Hostname:        hostInfo.Hostname,
			OS:              hostInfo.OS,
			Platform:        hostInfo.Platform,
			PlatformVersion: hostInfo.PlatformVersion,
			KernelVersion:   hostInfo.KernelVersion,
			KernelArch:      hostInfo.KernelArch,
			Virtualization:  detectVirtualization(hostInfo.VirtualizationSystem),
		}
	}
	payload.Host.Timezone = getTimezone()
	payload.Host.TCPCongestion = getTCPCongestion()
	payload.Host.Uptime, _ = hostUtil.Uptime()

	cpuInfo, _ := cpu.Info()
	if len(cpuInfo) > 0 {
		payload.CPU.ModelName = cpuInfo[0].ModelName
	}
	payload.CPU.Cores, _ = cpu.Counts(false)
	payload.CPU.LogicalCores, _ = cpu.Counts(true)

	if memStat, err := mem.VirtualMemory(); err == nil {
		payload.Memory.Total = memStat.Total
	}

	return payload, nil
}

func newFleetInstanceID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("xpanel-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}
