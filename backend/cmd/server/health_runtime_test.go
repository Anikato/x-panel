package main

import (
	"context"
	"errors"
	"os"
	"reflect"
	"syscall"
	"testing"

	"xpanel/app/service"
)

func TestCollectHealthRuntimeUsesFixedProbesAndOmitsPaths(t *testing.T) {
	var calls [][]string
	runner := func(_ context.Context, name string, args ...string) ([]byte, error) {
		call := append([]string{name}, args...)
		calls = append(calls, call)
		if args[1] == "docker.service" {
			return nil, errors.New("not installed")
		}
		return []byte("ActiveState=active\nUnitFileState=enabled\nActiveEnterTimestampUSec=1793260800000000\n"), nil
	}
	var paths []string
	statfs := func(path string, stat *syscall.Statfs_t) error {
		paths = append(paths, path)
		stat.Blocks = 100
		stat.Bavail = 25
		stat.Bsize = 4096
		stat.Files = 200
		stat.Ffree = 50
		return nil
	}

	data := &service.HealthSnapshotData{}
	collectHealthRuntimeWith(data, "/secret/xpanel-data", "/secret/xpanel.db", runner, statfs, os.Stat)

	if len(calls) != 4 {
		t.Fatalf("calls=%v", calls)
	}
	for index, target := range healthServiceUnits {
		want := []string{"systemctl", "show", target.unit, "--property=ActiveState", "--property=UnitFileState", "--property=ActiveEnterTimestampUSec", "--no-pager"}
		if !reflect.DeepEqual(calls[index], want) {
			t.Fatalf("call[%d]=%v want=%v", index, calls[index], want)
		}
	}
	if data.Services[3].ActiveState != "unknown" || data.Services[3].UnitFileState != "unknown" {
		t.Fatalf("docker=%+v", data.Services[3])
	}
	if !reflect.DeepEqual(paths, []string{"/", "/secret/xpanel-data"}) {
		t.Fatalf("paths=%v", paths)
	}
	if len(data.Storage) != 2 || data.Storage[0].Name != "root" || data.Storage[1].Name != "xpanel-data" {
		t.Fatalf("storage=%+v", data.Storage)
	}
	if data.Storage[0].UsedPercent != 75 || data.Storage[0].InodesUsedPct != 75 {
		t.Fatalf("storage=%+v", data.Storage[0])
	}
	if data.Panel.StartedAt != "2026-10-29T08:00:00Z" {
		t.Fatalf("started_at=%q", data.Panel.StartedAt)
	}
}

func TestCollectHealthSecurityReportsOnlyStableIssueCodes(t *testing.T) {
	dir := t.TempDir()
	databasePath := dir + "/xpanel.db"
	if err := os.WriteFile(databasePath, []byte("test"), 0o666); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(databasePath, 0o666); err != nil {
		t.Fatal(err)
	}
	data := &service.HealthSnapshotData{}
	collectHealthSecurity(data, dir, databasePath, os.Stat)
	if data.Security.DatabasePermissionsSecure || !data.Security.DataDirPermissionsSecure {
		t.Fatalf("security=%+v", data.Security)
	}
	if !reflect.DeepEqual(data.Security.Issues, []string{"database_permissions"}) {
		t.Fatalf("issues=%v", data.Security.Issues)
	}
}

func TestParseHealthPropertiesIgnoresMalformedLines(t *testing.T) {
	values := parseHealthProperties([]byte("ActiveState=failed\nmalformed\nUnitFileState=disabled\n"))
	if values["ActiveState"] != "failed" || values["UnitFileState"] != "disabled" {
		t.Fatalf("values=%v", values)
	}
}
