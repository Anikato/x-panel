package service

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"xpanel/app/dto"
	"xpanel/app/model"
	"xpanel/global"
)

const xpanelNatTable = "ip xpanel_nat"

func (s *FirewallService) ListForwards() ([]dto.ForwardRuleInfo, error) {
	var rules []model.FirewallForward
	if err := global.DB.Order("id asc").Find(&rules).Error; err != nil {
		return nil, err
	}
	items := make([]dto.ForwardRuleInfo, 0, len(rules))
	for _, rule := range rules {
		items = append(items, dto.ForwardRuleInfo{
			ID: rule.ID, Protocol: rule.Protocol, Port: rule.Port,
			TargetIP: rule.TargetIP, TargetPort: rule.TargetPort,
		})
	}
	return items, nil
}

func (s *FirewallService) CreateForward(req dto.ForwardRuleCreate) error {
	rule, err := normalizeForwardRule(req)
	if err != nil {
		return err
	}
	if err := global.DB.Create(&rule).Error; err != nil {
		return err
	}
	if err := s.applyStoredForwards(); err != nil {
		_ = global.DB.Delete(&rule).Error
		_ = s.applyStoredForwards()
		return err
	}
	if err := allowUFWRoute(rule); err != nil {
		return err
	}
	return nil
}

func (s *FirewallService) DeleteForward(id uint) error {
	var rule model.FirewallForward
	if err := global.DB.First(&rule, id).Error; err != nil {
		return fmt.Errorf("转发规则不存在")
	}
	if err := global.DB.Delete(&rule).Error; err != nil {
		return err
	}
	if err := s.applyStoredForwards(); err != nil {
		return err
	}
	_ = deleteUFWRoute(rule)
	return nil
}

func ApplyStoredFirewallForwards() {
	if global.DB == nil {
		return
	}
	if err := (&FirewallService{}).applyStoredForwards(); err != nil && global.LOG != nil {
		global.LOG.Warnf("apply firewall port forwards: %v", err)
	}
}

func (s *FirewallService) applyStoredForwards() error {
	var rules []model.FirewallForward
	if global.DB != nil {
		if err := global.DB.Order("id asc").Find(&rules).Error; err != nil {
			return err
		}
	}
	if len(rules) > 0 {
		if err := enableIPv4Forward(); err != nil {
			return err
		}
	}
	return applyNFT(renderPortForwardNFT(rules))
}

func normalizeForwardRule(req dto.ForwardRuleCreate) (model.FirewallForward, error) {
	protocol := strings.ToLower(strings.TrimSpace(req.Protocol))
	if protocol != "tcp" && protocol != "udp" && protocol != "both" {
		return model.FirewallForward{}, fmt.Errorf("协议只支持 tcp、udp 或 both")
	}
	port, err := normalizePortRange(req.Port)
	if err != nil {
		return model.FirewallForward{}, fmt.Errorf("外部端口: %w", err)
	}
	ip := net.ParseIP(strings.TrimSpace(req.TargetIP))
	if ip == nil || ip.To4() == nil {
		return model.FirewallForward{}, fmt.Errorf("目标 IP 必须是 IPv4 地址")
	}
	targetPort := strings.TrimSpace(req.TargetPort)
	if targetPort != "" {
		if strings.Contains(port, "-") {
			return model.FirewallForward{}, fmt.Errorf("端口段转发请把目标端口留空，以保持原端口")
		}
		normalized, err := normalizePortRange(targetPort)
		if err != nil || strings.Contains(normalized, "-") {
			return model.FirewallForward{}, fmt.Errorf("目标端口必须是单个端口")
		}
		targetPort = normalized
	}
	return model.FirewallForward{
		Protocol: protocol, Port: port, TargetIP: ip.To4().String(), TargetPort: targetPort,
	}, nil
}

func normalizePortRange(value string) (string, error) {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, ":", "-")
	parts := strings.Split(value, "-")
	if len(parts) == 1 {
		port, err := parsePortNumber(parts[0])
		if err != nil {
			return "", err
		}
		return strconv.Itoa(port), nil
	}
	if len(parts) != 2 {
		return "", fmt.Errorf("端口格式应为 80 或 8000:8100")
	}
	start, err := parsePortNumber(parts[0])
	if err != nil {
		return "", err
	}
	end, err := parsePortNumber(parts[1])
	if err != nil {
		return "", err
	}
	if start > end {
		return "", fmt.Errorf("起始端口不能大于结束端口")
	}
	return strconv.Itoa(start) + "-" + strconv.Itoa(end), nil
}

func parsePortNumber(value string) (int, error) {
	port, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || port < 1 || port > 65535 {
		return 0, fmt.Errorf("端口必须在 1-65535 之间")
	}
	return port, nil
}

func renderPortForwardNFT(rules []model.FirewallForward) string {
	var b strings.Builder
	b.WriteString("add table " + xpanelNatTable + "\n")
	b.WriteString("delete table " + xpanelNatTable + "\n")
	if len(rules) == 0 {
		return b.String()
	}
	b.WriteString("table " + xpanelNatTable + " {\n")
	b.WriteString("  chain prerouting {\n")
	b.WriteString("    type nat hook prerouting priority dstnat; policy accept;\n")
	for _, rule := range rules {
		for _, protocol := range forwardProtocols(rule.Protocol) {
			fmt.Fprintf(&b, "    %s dport %s dnat to %s\n", protocol, rule.Port, natDestination(rule))
		}
	}
	b.WriteString("  }\n")
	b.WriteString("  chain postrouting {\n")
	b.WriteString("    type nat hook postrouting priority srcnat; policy accept;\n")
	for _, rule := range rules {
		for _, protocol := range forwardProtocols(rule.Protocol) {
			fmt.Fprintf(&b, "    ip daddr %s %s dport %s masquerade\n", rule.TargetIP, protocol, masqueradePort(rule))
		}
	}
	b.WriteString("  }\n")
	b.WriteString("  chain forward {\n")
	b.WriteString("    type filter hook forward priority -10; policy accept;\n")
	for _, rule := range rules {
		for _, protocol := range forwardProtocols(rule.Protocol) {
			fmt.Fprintf(&b, "    ip daddr %s %s dport %s accept\n", rule.TargetIP, protocol, masqueradePort(rule))
		}
	}
	b.WriteString("  }\n")
	b.WriteString("}\n")
	return b.String()
}

func forwardProtocols(protocol string) []string {
	if protocol == "both" {
		return []string{"tcp", "udp"}
	}
	return []string{protocol}
}

func natDestination(rule model.FirewallForward) string {
	if rule.TargetPort == "" {
		return rule.TargetIP
	}
	return rule.TargetIP + ":" + rule.TargetPort
}

func masqueradePort(rule model.FirewallForward) string {
	if rule.TargetPort != "" {
		return rule.TargetPort
	}
	return rule.Port
}

var applyNFT = func(script string) error {
	if _, err := exec.LookPath("nft"); err != nil {
		return fmt.Errorf("未找到 nft 命令。端口转发使用 nftables，不需要安装 ufw")
	}
	cmd := exec.Command("nft", "-f", "-")
	cmd.Stdin = strings.NewReader(script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		text := strings.TrimSpace(string(output))
		if text == "" {
			return err
		}
		return fmt.Errorf("%s", text)
	}
	return nil
}

func enableIPv4Forward() error {
	content := "net.ipv4.ip_forward=1\n"
	if err := os.MkdirAll("/etc/sysctl.d", 0755); err == nil {
		_ = os.WriteFile("/etc/sysctl.d/99-xpanel-ip-forward.conf", []byte(content), 0644)
	}
	cmd := exec.Command("sysctl", "-w", "net.ipv4.ip_forward=1")
	output, err := cmd.CombinedOutput()
	if err != nil {
		text := strings.TrimSpace(string(output))
		if text == "" {
			return fmt.Errorf("打开 IP 转发失败: %v", err)
		}
		return fmt.Errorf("打开 IP 转发失败: %s", text)
	}
	return nil
}

func ufwIsActive() bool {
	output, err := exec.Command("ufw", "status").CombinedOutput()
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(string(output)), "status: active")
}

func allowUFWRoute(rule model.FirewallForward) error {
	if !ufwIsActive() {
		return nil
	}
	for _, protocol := range forwardProtocols(rule.Protocol) {
		args := []string{"route", "allow", "proto", protocol, "to", rule.TargetIP, "port", ufwPort(masqueradePort(rule))}
		output, err := exec.Command("ufw", args...).CombinedOutput()
		if err != nil && !strings.Contains(strings.ToLower(string(output)), "existing") {
			return fmt.Errorf("%s", strings.TrimSpace(string(output)))
		}
	}
	return nil
}

func deleteUFWRoute(rule model.FirewallForward) error {
	if !ufwIsActive() {
		return nil
	}
	for _, protocol := range forwardProtocols(rule.Protocol) {
		args := []string{"route", "delete", "allow", "proto", protocol, "to", rule.TargetIP, "port", ufwPort(masqueradePort(rule))}
		output, err := exec.Command("ufw", args...).CombinedOutput()
		if err != nil && !strings.Contains(strings.ToLower(string(output)), "could not delete") {
			return fmt.Errorf("%s", strings.TrimSpace(string(output)))
		}
	}
	return nil
}

func ufwPort(port string) string {
	return strings.ReplaceAll(port, "-", ":")
}
