package service

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"xpanel/app/dto"
)

func TestPanelDownloadProxyChangesImmediatelyAndHonorsBypass(t *testing.T) {
	openSettingServiceDatabase(t)
	t.Setenv("HTTPS_PROXY", "http://system.example:3128")
	service := NewISettingService()
	settings := dto.ProxyConfigUpdate{
		Type: "socks5", Address: "proxy.example:1080", Enabled: true, NoProxy: ".internal.example",
	}
	if _, err := service.SaveProxy(settings); err != nil {
		t.Fatal(err)
	}
	client := NewPanelHTTPClient(time.Second)
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatal("panel proxy transport was not installed")
	}
	for target, want := range map[string]string{
		"https://downloads.example":        "socks5://proxy.example:1080",
		"https://service.internal.example": "",
		"http://127.0.0.1/":                "",
	} {
		req, err := http.NewRequest(http.MethodGet, target, nil)
		if err != nil {
			t.Fatal(err)
		}
		proxy, err := transport.Proxy(req)
		if err != nil || (proxy == nil && want != "") || (proxy != nil && proxy.String() != want) {
			t.Fatalf("proxy for %s: %v, %v", target, proxy, err)
		}
	}
	if got := os.Getenv("HTTPS_PROXY"); got != "http://system.example:3128" {
		t.Fatalf("system proxy environment changed: %s", got)
	}
	settings.Enabled = false
	settings.Address = ""
	if _, err := service.SaveProxy(settings); err != nil {
		t.Fatal(err)
	}
	if NewPanelHTTPClient(time.Second).Transport != nil {
		t.Fatal("disabled panel proxy still installed")
	}
}

func TestDockerProxyConfigPreservesMirrorAndCanRollBack(t *testing.T) {
	path := filepath.Join(t.TempDir(), "daemon.json")
	original := []byte("{\"registry-mirrors\":[\"https://mirror.example\"]}\n")
	if err := os.WriteFile(path, original, 0644); err != nil {
		t.Fatal(err)
	}
	desired := map[string]string{"http-proxy": "http://proxy.example:8000"}
	rollback, err := saveDockerProxyConfig(path, nil, desired)
	if err != nil || rollback == nil {
		t.Fatalf("save Docker proxy: %v", err)
	}
	content, err := os.ReadFile(path)
	if err != nil || !strings.Contains(string(content), "https://mirror.example") ||
		!strings.Contains(string(content), "http://proxy.example:8000") {
		t.Fatalf("updated Docker configuration is incomplete: %v", err)
	}
	if _, err := saveDockerProxyConfig(path, nil, map[string]string{"http-proxy": "http://other.example"}); err == nil {
		t.Fatal("unmanaged proxy settings were not protected")
	}
	if err := rollback(); err != nil {
		t.Fatal(err)
	}
	content, err = os.ReadFile(path)
	if err != nil || string(content) != string(original) {
		t.Fatalf("Docker configuration was not restored: %v", err)
	}
}
