package service

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"xpanel/app/dto"
)

func proxyTestPaths(t *testing.T) proxyConfigPaths {
	t.Helper()
	root := t.TempDir()
	return proxyConfigPaths{
		environment:  filepath.Join(root, "environment"),
		profile:      filepath.Join(root, "xpanel-proxy.sh"),
		apt:          filepath.Join(root, "99xpanel-proxy"),
		docker:       filepath.Join(root, "daemon.json"),
		dockerDropIn: filepath.Join(root, "http-proxy.conf"),
	}
}

func writeProxyFixture(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func expectProxyFile(t *testing.T, path string, expected ...string) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range expected {
		if !strings.Contains(string(content), value) {
			t.Fatalf("%s is missing %q", path, value)
		}
	}
}

func expectProxyFileRemoved(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		t.Fatalf("file %s should have been removed: %v", path, err)
	}
}

func TestProxyScopeSystemAndNarrowerScopes(t *testing.T) {
	openSettingServiceDatabase(t)
	paths := proxyTestPaths(t)
	previousPaths := hostProxyPaths
	hostProxyPaths = paths
	t.Cleanup(func() { hostProxyPaths = previousPaths })
	writeProxyFixture(t, paths.environment, "LANG=C.UTF-8\n")
	writeProxyFixture(t, paths.docker, "{\"registry-mirrors\":[\"https://mirror.example\"]}\n")

	service := NewISettingService()
	request := dto.ProxyConfigUpdate{Scope: "system", Type: "http", Address: "http://proxy.example:3128"}
	result, err := service.SaveProxy(request)
	if err != nil || !result.DockerChanged || !result.DockerRestartPending {
		t.Fatalf("save system scope: %+v, %v", result, err)
	}
	expectProxyFile(t, paths.environment, "LANG=C.UTF-8\n", "http_proxy=\"http://proxy.example:3128\"")
	expectProxyFile(t, paths.profile, "export HTTPS_PROXY='http://proxy.example:3128'")
	expectProxyFile(t, paths.apt, "Acquire::http::Proxy \"http://proxy.example:3128\";", "Acquire::https::Proxy \"http://proxy.example:3128\";")
	expectProxyFile(t, paths.docker, "http://proxy.example:3128", "https://mirror.example")

	setting, err := service.GetSettingInfo()
	if err != nil || setting.ProxyScope != "system" || !setting.ProxyDocker || !setting.DockerRestartPending {
		t.Fatalf("reloaded settings: %+v, %v", setting, err)
	}

	request.Address = ""
	request.Scope = "panel-docker"
	result, err = service.SaveProxy(request)
	if err != nil || result.DockerChanged || !result.DockerRestartPending {
		t.Fatalf("narrow to panel and Docker: %+v, %v", result, err)
	}
	expectProxyFileRemoved(t, paths.profile)
	expectProxyFileRemoved(t, paths.apt)
	content, err := os.ReadFile(paths.environment)
	if err != nil || string(content) != "LANG=C.UTF-8\n" {
		t.Fatalf("unrelated environment settings changed: %q, %v", content, err)
	}

	request.Scope = "panel"
	result, err = service.SaveProxy(request)
	if err != nil || !result.DockerChanged || !result.DockerRestartPending {
		t.Fatalf("narrow to panel: %+v, %v", result, err)
	}
	content, err = os.ReadFile(paths.docker)
	if err != nil || !strings.Contains(string(content), "https://mirror.example") || strings.Contains(string(content), "proxy.example") {
		t.Fatalf("Docker proxy removed without changing mirrors: %s, %v", content, err)
	}

	request.Scope = "off"
	if _, err := service.SaveProxy(request); err != nil {
		t.Fatal(err)
	}
	setting, err = service.GetSettingInfo()
	if err != nil || setting.ProxyScope != "off" || !setting.ProxyAddressSet || !setting.DockerRestartPending {
		t.Fatalf("off retains address and pending status: %+v, %v", setting, err)
	}
}

func TestProxyScopeMigratesOnlyOwnedLegacyFiles(t *testing.T) {
	openSettingServiceDatabase(t)
	paths := proxyTestPaths(t)
	const oldEnv = "LANG=C.UTF-8\nhttp_proxy=\"http://old.example:3128\"\nHTTPS_PROXY=\"http://independent.example:3128\"\n"
	writeProxyFixture(t, paths.environment, oldEnv)
	writeProxyFixture(t, paths.profile, legacyProxyProfile+"export http_proxy=\"http://old.example:3128\"\n")
	writeProxyFixture(t, paths.apt, legacyProxyAPT+"Acquire::http::Proxy \"http://old.example:3128\";\n")
	writeProxyFixture(t, paths.dockerDropIn, legacyProxyDocker+"Environment=\"HTTP_PROXY=http://old.example:3128\"\n")

	newSettings := map[string]string{"ProxyType": "http", "ProxyAddress": "proxy.example:3128", "ProxyEnable": "enable", "ProxyDocker": "enable", "ProxySystem": "enable"}
	if _, err := saveProxySettings(nil, newSettings, paths, true); err == nil || !strings.Contains(err.Error(), "HTTPS_PROXY") {
		t.Fatalf("system proxy should protect independent proxy: %v", err)
	}
	expectProxyFile(t, paths.environment, oldEnv)
	expectProxyFile(t, paths.profile, legacyProxyProfile, "http://old.example:3128")

	result, err := saveProxySettings(nil, map[string]string{"ProxyEnable": "disable"}, paths, true)
	if err != nil || !result.DockerRestartPending || result.LegacySystemProxy {
		t.Fatalf("remove old configuration: %+v, %v", result, err)
	}
	expectProxyFileRemoved(t, paths.profile)
	expectProxyFileRemoved(t, paths.apt)
	expectProxyFileRemoved(t, paths.dockerDropIn)
	content, err := os.ReadFile(paths.environment)
	if err != nil || string(content) != "LANG=C.UTF-8\nHTTPS_PROXY=\"http://independent.example:3128\"\n" {
		t.Fatalf("legacy migration rewrote independent variables: %q, %v", content, err)
	}
}

func TestProxyScopeRollsBackWhenAHostFileCannotBeWritten(t *testing.T) {
	openSettingServiceDatabase(t)
	paths := proxyTestPaths(t)
	paths.profile = filepath.Join(filepath.Dir(paths.profile), "missing", "profile.sh")
	writeProxyFixture(t, paths.environment, "LANG=C.UTF-8\n")
	const dockerOriginal = "{\"registry-mirrors\":[\"https://mirror.example\"]}\n"
	writeProxyFixture(t, paths.docker, dockerOriginal)
	newSettings := map[string]string{"ProxyType": "http", "ProxyAddress": "proxy.example:3128", "ProxyEnable": "enable", "ProxyDocker": "enable", "ProxySystem": "enable"}
	if _, err := saveProxySettings(nil, newSettings, paths, false); err == nil {
		t.Fatal("host write failure should fail saving the selected scope")
	}
	content, err := os.ReadFile(paths.environment)
	if err != nil || string(content) != "LANG=C.UTF-8\n" {
		t.Fatalf("previous environment not restored: %q, %v", content, err)
	}
	expectProxyFile(t, paths.docker, dockerOriginal)
	setting, err := NewISettingService().GetSettingInfo()
	if err != nil || setting.ProxyScope != "off" || setting.DockerRestartPending {
		t.Fatalf("database changed after failed save: %+v, %v", setting, err)
	}
}

func TestSystemProxyProfileShellQuotesBypass(t *testing.T) {
	paths := proxyTestPaths(t)
	const bypass = "localhost,'$(exit 99)'"
	changes, err := planSystemProxy(paths, "http://proxy.example:3128", bypass)
	if err != nil {
		t.Fatal(err)
	}
	for _, change := range changes {
		if _, err := change.apply(); err != nil {
			t.Fatal(err)
		}
	}
	result, err := exec.Command("sh", "-c", ". \"$1\"\nprintf '%s' \"$NO_PROXY\"", "sh", paths.profile).Output()
	if err != nil || string(result) != bypass {
		t.Fatalf("shell proxy bypass = %q, %v", result, err)
	}
}

func TestDockerProxySaveWaitsForExplicitRestart(t *testing.T) {
	openSettingServiceDatabase(t)
	paths := proxyTestPaths(t)
	bin := t.TempDir()
	log := filepath.Join(bin, "systemctl.log")
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("XPANEL_TEST_SYSTEMCTL_LOG", log)
	systemctl := "#!/bin/sh\nprintf '%s\\n' \"$*\" >> \"$XPANEL_TEST_SYSTEMCTL_LOG\"\n" +
		"if [ \"$1\" = restart ] && [ \"$XPANEL_TEST_RESTART_FAIL\" = 1 ]; then exit 1; fi\n"
	if err := os.WriteFile(filepath.Join(bin, "systemctl"), []byte(systemctl), 0700); err != nil {
		t.Fatal(err)
	}
	values := map[string]string{"ProxyType": "http", "ProxyAddress": "proxy.example:3128", "ProxyEnable": "enable", "ProxyDocker": "enable"}
	result, err := saveProxySettings(nil, values, paths, false)
	if err != nil || !result.DockerRestartPending {
		t.Fatalf("save Docker proxy: %+v, %v", result, err)
	}
	expectProxyFileRemoved(t, log)

	container := NewIContainerService()
	t.Setenv("XPANEL_TEST_RESTART_FAIL", "1")
	if err := container.ControlDockerService("restart"); err == nil {
		t.Fatal("restart error was ignored")
	}
	pending, err := dockerProxyRestartPending()
	if err != nil || !pending {
		t.Fatalf("failed restart removed pending status: %v, %v", pending, err)
	}

	t.Setenv("XPANEL_TEST_RESTART_FAIL", "")
	if err := container.ControlDockerService("restart"); err != nil {
		t.Fatal(err)
	}
	pending, err = dockerProxyRestartPending()
	if err != nil || pending {
		t.Fatalf("successful restart should clear pending status: %v, %v", pending, err)
	}
	expectProxyFile(t, log, "daemon-reload\nrestart docker\n")
}
