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
	fleetDefaultInterval = 5 * time.Minute
	fleetMinInterval     = 30 * time.Second
	fleetMaxInterval     = 24 * time.Hour
	fleetStartupDelay    = 15 * time.Second
	fleetTaskTimeout     = 10 * time.Second
	fleetTaskOutputLimit = 64 * 1024

	fleetSettingEnabled       = "FleetEnabled"
	fleetSettingEndpoint      = "FleetEndpoint"
	fleetSettingInstanceID    = "FleetInstanceID"
	fleetSettingInstanceToken = "FleetInstanceToken"
	fleetSettingInterval      = "FleetHeartbeatIntervalSeconds"
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
	InstanceID               string    `json:"instanceId"`
	InstanceToken            string    `json:"instanceToken"`
	ServerTime               time.Time `json:"serverTime"`
	HeartbeatIntervalSeconds int       `json:"heartbeatIntervalSeconds"`
}

type fleetHeartbeatData struct {
	ServerTime               time.Time `json:"serverTime"`
	HeartbeatIntervalSeconds int       `json:"heartbeatIntervalSeconds"`
}

type fleetTaskPollRequest struct {
	InstanceID string `json:"instanceId"`
}

type fleetTaskPollData struct {
	Tasks []fleetTask `json:"tasks"`
}

type fleetTask struct {
	ID      uint            `json:"id"`
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type fleetTailPanelLogPayload struct {
	Lines   int    `json:"lines"`
	Level   string `json:"level"`
	Keyword string `json:"keyword"`
}

type fleetTaskReportRequest struct {
	InstanceID string `json:"instanceId"`
	TaskID     uint   `json:"taskId"`
	Status     string `json:"status"`
	Output     string `json:"output"`
	Error      string `json:"error"`
	ExitCode   int    `json:"exitCode"`
	Truncated  bool   `json:"truncated"`
}

func (s *FleetReporterService) Start() {
	go func() {
		time.Sleep(fleetStartupDelay)
		for {
			interval := s.reportOnce()
			timer := time.NewTimer(interval)
			<-timer.C
		}
	}()
}

func (s *FleetReporterService) reportOnce() time.Duration {
	enabled := s.settingValue(fleetSettingEnabled, "enable")
	if strings.EqualFold(enabled, "disable") {
		return s.reportInterval()
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
			return s.reportInterval()
		}
	}

	payload, err := buildFleetPayload(instanceID)
	if err != nil {
		global.LOG.Debugf("fleet reporter build payload failed: %v", err)
		return s.reportInterval()
	}

	token := s.settingValue(fleetSettingInstanceToken, "")
	if token == "" {
		if err := s.register(endpoint, payload); err != nil {
			global.LOG.Debugf("fleet reporter register failed: %v", err)
		} else if token = s.settingValue(fleetSettingInstanceToken, ""); token != "" {
			s.pollTasks(endpoint, payload.InstanceID, token)
		}
		return s.reportInterval()
	}

	if err := s.heartbeat(endpoint, payload, token); err != nil {
		global.LOG.Debugf("fleet reporter heartbeat failed: %v", err)
		if strings.Contains(err.Error(), "401") || strings.Contains(err.Error(), "404") {
			_ = settingRepo.CreateOrUpdate(fleetSettingInstanceToken, "")
			if err := s.register(endpoint, payload); err != nil {
				global.LOG.Debugf("fleet reporter re-register failed: %v", err)
			}
		}
	} else {
		s.pollTasks(endpoint, payload.InstanceID, token)
	}
	return s.reportInterval()
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
	if err := settingRepo.CreateOrUpdate(fleetSettingInstanceToken, data.InstanceToken); err != nil {
		return err
	}
	s.applyServerInterval(data.HeartbeatIntervalSeconds)
	return nil
}

func (s *FleetReporterService) heartbeat(endpoint string, payload fleetPayload, token string) error {
	var data fleetHeartbeatData
	if err := s.postJSON(endpoint+"/api/v1/fleet/heartbeat", token, payload, &data); err != nil {
		return err
	}
	s.applyServerInterval(data.HeartbeatIntervalSeconds)
	return nil
}

func (s *FleetReporterService) postJSON(url, token string, payload interface{}, out interface{}) error {
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

func (s *FleetReporterService) pollTasks(endpoint, instanceID, token string) {
	var data fleetTaskPollData
	req := fleetTaskPollRequest{InstanceID: instanceID}
	if err := s.postJSON(endpoint+"/api/v1/fleet/tasks/poll", token, req, &data); err != nil {
		global.LOG.Debugf("fleet reporter poll tasks failed: %v", err)
		return
	}
	for _, task := range data.Tasks {
		report := s.executeTask(instanceID, task)
		if err := s.postJSON(endpoint+"/api/v1/fleet/tasks/report", token, report, nil); err != nil {
			global.LOG.Debugf("fleet reporter report task failed: task=%d err=%v", task.ID, err)
		}
	}
}

func (s *FleetReporterService) executeTask(instanceID string, task fleetTask) fleetTaskReportRequest {
	report := fleetTaskReportRequest{
		InstanceID: instanceID,
		TaskID:     task.ID,
		Status:     "failed",
		ExitCode:   1,
	}
	if task.Type != "tail_panel_log" {
		report.Error = "unsupported task type"
		return report
	}

	done := make(chan fleetTaskReportRequest, 1)
	go func() {
		output, err := executeTailPanelLog(task.Payload)
		result := fleetTaskReportRequest{
			InstanceID: instanceID,
			TaskID:     task.ID,
			Status:     "success",
			Output:     output,
			ExitCode:   0,
		}
		if err != nil {
			result.Status = "failed"
			result.Error = err.Error()
			result.ExitCode = 1
		}
		result.Output, result.Truncated = truncateFleetTaskOutput(result.Output)
		done <- result
	}()

	select {
	case result := <-done:
		return result
	case <-time.After(fleetTaskTimeout):
		report.Error = "task execution timeout"
		return report
	}
}

func executeTailPanelLog(raw json.RawMessage) (string, error) {
	payload := fleetTailPanelLogPayload{Lines: 200, Level: "ERROR"}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &payload); err != nil {
			return "", err
		}
	}
	if payload.Lines <= 0 {
		payload.Lines = 200
	}
	if payload.Lines > 500 {
		payload.Lines = 500
	}
	payload.Level = strings.ToUpper(strings.TrimSpace(payload.Level))
	if payload.Level == "" {
		payload.Level = "ERROR"
	}
	if payload.Level != "INFO" && payload.Level != "WARN" && payload.Level != "ERROR" {
		return "", fmt.Errorf("invalid log level")
	}
	if len(payload.Keyword) > 128 {
		payload.Keyword = payload.Keyword[:128]
	}
	return NewILogService().GetSystemLog(payload.Lines, payload.Level, strings.TrimSpace(payload.Keyword))
}

func truncateFleetTaskOutput(value string) (string, bool) {
	if len(value) <= fleetTaskOutputLimit {
		return value, false
	}
	return value[:fleetTaskOutputLimit], true
}

func (s *FleetReporterService) settingValue(key, fallback string) string {
	value, err := settingRepo.GetValueByKey(key)
	if err != nil || strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}

func (s *FleetReporterService) reportInterval() time.Duration {
	raw := s.settingValue(fleetSettingInterval, "300")
	seconds, err := time.ParseDuration(raw + "s")
	if err != nil {
		return fleetDefaultInterval
	}
	if seconds < fleetMinInterval {
		return fleetMinInterval
	}
	if seconds > fleetMaxInterval {
		return fleetMaxInterval
	}
	return seconds
}

func (s *FleetReporterService) applyServerInterval(seconds int) {
	if seconds <= 0 {
		return
	}
	if seconds < int(fleetMinInterval.Seconds()) {
		seconds = int(fleetMinInterval.Seconds())
	}
	if seconds > int(fleetMaxInterval.Seconds()) {
		seconds = int(fleetMaxInterval.Seconds())
	}
	if err := settingRepo.CreateOrUpdate(fleetSettingInterval, fmt.Sprintf("%d", seconds)); err != nil {
		global.LOG.Debugf("fleet reporter save heartbeat interval failed: %v", err)
	}
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
