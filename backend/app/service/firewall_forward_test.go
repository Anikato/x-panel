package service

import (
	"strings"
	"testing"

	"xpanel/app/dto"
	"xpanel/app/model"
)

func TestRenderPortForwardKeepsUDPRangeAndMasquerade(t *testing.T) {
	rules := []model.FirewallForward{
		{Protocol: "tcp", Port: "5061", TargetIP: "100.100.100.222", TargetPort: "5061"},
		{Protocol: "udp", Port: "10000-10100", TargetIP: "100.100.100.222"},
	}
	script := renderPortForwardNFT(rules)
	for _, want := range []string{
		"tcp dport 5061 dnat to 100.100.100.222:5061",
		"udp dport 10000-10100 dnat to 100.100.100.222",
		"ip daddr 100.100.100.222 tcp dport 5061 masquerade",
		"ip daddr 100.100.100.222 udp dport 10000-10100 masquerade",
		"hook forward priority -10",
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("script missing %q\n%s", want, script)
		}
	}
}

func TestNormalizeForwardRuleRejectsMappedPortRange(t *testing.T) {
	_, err := normalizeForwardRule(dto.ForwardRuleCreate{
		Protocol: "udp", Port: "10000:10100", TargetIP: "100.100.100.222", TargetPort: "20000",
	})
	if err == nil {
		t.Fatal("a port range must keep the original destination ports")
	}
	rule, err := normalizeForwardRule(dto.ForwardRuleCreate{
		Protocol: "udp", Port: "10000:10100", TargetIP: "100.100.100.222",
	})
	if err != nil || rule.Port != "10000-10100" || rule.TargetPort != "" {
		t.Fatalf("rule = %#v err=%v", rule, err)
	}
}

func TestRenderPortForwardRemovesTableWhenEmpty(t *testing.T) {
	script := renderPortForwardNFT(nil)
	if !strings.Contains(script, "delete table ip xpanel_nat") || strings.Contains(script, "dnat") {
		t.Fatalf("empty script = %s", script)
	}
}
