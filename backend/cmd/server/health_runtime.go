package main

import (
	"context"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	"xpanel/app/service"
	"xpanel/app/version"
	"xpanel/global"
)

const healthProbeTimeout = 2 * time.Second

var healthServiceUnits = []struct {
	name string
	unit string
}{
	{name: "xpanel", unit: "xpanel.service"},
	{name: "nezha-agent", unit: "xpanel-nezha-agent.service"},
	{name: "nginx", unit: "nginx.service"},
	{name: "docker", unit: "docker.service"},
}

type healthCommandRunner func(context.Context, string, ...string) ([]byte, error)
type healthStatfsFunc func(string, *syscall.Statfs_t) error
type healthStatFunc func(string) (os.FileInfo, error)

func collectHealthRuntime(data *service.HealthSnapshotData) {
	collectHealthRuntimeWith(data, global.CONF.System.DataDir, global.CONF.System.DbPath, runHealthCommand, syscall.Statfs, os.Stat)
}

func collectHealthRuntimeWith(data *service.HealthSnapshotData, dataDir, databasePath string, runner healthCommandRunner, statfs healthStatfsFunc, stat healthStatFunc) {
	info := version.Get()
	data.Panel.Version = info.Version
	data.Panel.CommitHash = info.CommitHash
	data.Panel.BuildTime = info.BuildTime

	data.Services = make([]service.HealthServiceSnapshot, 0, len(healthServiceUnits))
	for _, target := range healthServiceUnits {
		ctx, cancel := context.WithTimeout(context.Background(), healthProbeTimeout)
		output, _ := runner(ctx, "systemctl", "show", target.unit, "--property=ActiveState", "--property=UnitFileState", "--property=ActiveEnterTimestampUSec", "--no-pager")
		cancel()
		values := parseHealthProperties(output)
		activeState := values["ActiveState"]
		if activeState == "" {
			activeState = "unknown"
		}
		unitFileState := values["UnitFileState"]
		if unitFileState == "" {
			unitFileState = "unknown"
		}
		startedAt := healthSystemdTimestamp(values["ActiveEnterTimestampUSec"])
		data.Services = append(data.Services, service.HealthServiceSnapshot{
			Name:          target.name,
			Unit:          target.unit,
			ActiveState:   activeState,
			UnitFileState: unitFileState,
			StartedAt:     startedAt,
		})
		if target.name == "xpanel" {
			data.Panel.StartedAt = startedAt
		}
	}

	data.Storage = make([]service.HealthStorageSnapshot, 0, 2)
	if snapshot, ok := collectHealthStorage("root", "/", statfs); ok {
		data.Storage = append(data.Storage, snapshot)
	}
	if dataDir != "" {
		if snapshot, ok := collectHealthStorage("xpanel-data", dataDir, statfs); ok {
			data.Storage = append(data.Storage, snapshot)
		}
	}
	collectHealthSecurity(data, dataDir, databasePath, stat)
}

func healthSystemdTimestamp(value string) string {
	microseconds, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil || microseconds <= 0 {
		return ""
	}
	return time.UnixMicro(microseconds).UTC().Format(time.RFC3339)
}

func collectHealthSecurity(data *service.HealthSnapshotData, dataDir, databasePath string, stat healthStatFunc) {
	data.Security = service.HealthSecuritySummary{DatabasePermissionsSecure: true, DataDirPermissionsSecure: true, Issues: make([]string, 0)}
	if databasePath != "" {
		if info, err := stat(databasePath); err != nil || info.Mode().Perm()&0022 != 0 {
			data.Security.DatabasePermissionsSecure = false
			data.Security.Issues = append(data.Security.Issues, "database_permissions")
		}
	}
	if dataDir != "" {
		if info, err := stat(dataDir); err != nil || info.Mode().Perm()&0002 != 0 {
			data.Security.DataDirPermissionsSecure = false
			data.Security.Issues = append(data.Security.Issues, "data_dir_permissions")
		}
	}
}

func runHealthCommand(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).CombinedOutput()
}

func parseHealthProperties(output []byte) map[string]string {
	values := make(map[string]string)
	for _, line := range strings.Split(string(output), "\n") {
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		values[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	return values
}

func collectHealthStorage(name, path string, statfs healthStatfsFunc) (service.HealthStorageSnapshot, bool) {
	var stat syscall.Statfs_t
	if err := statfs(path, &stat); err != nil {
		return service.HealthStorageSnapshot{}, false
	}
	totalBytes := stat.Blocks * uint64(stat.Bsize)
	availableBytes := stat.Bavail * uint64(stat.Bsize)
	inodesTotal := stat.Files
	inodesAvailable := stat.Ffree
	return service.HealthStorageSnapshot{
		Name:            name,
		TotalBytes:      totalBytes,
		AvailableBytes:  availableBytes,
		UsedPercent:     healthUsedPercent(totalBytes, availableBytes),
		InodesTotal:     inodesTotal,
		InodesAvailable: inodesAvailable,
		InodesUsedPct:   healthUsedPercent(inodesTotal, inodesAvailable),
	}, true
}

func healthUsedPercent(total, available uint64) float64 {
	if total == 0 || available >= total {
		return 0
	}
	return float64(total-available) * 100 / float64(total)
}
