package service

import (
	"fmt"
	"regexp"
	"strings"

	"xpanel/app/dto"
	"xpanel/global"
	"xpanel/utils/cmd"
)

type IFirewallService interface {
	GetBaseInfo() (*dto.FirewallBaseInfo, error)
	Operate(operation string) error
	ListPortRules(req dto.PortRuleSearch) (int64, []dto.PortRuleInfo, error)
	CreatePortRule(req dto.PortRuleCreate) error
	DeletePortRule(req dto.PortRuleDelete) error
	ListIPRules() ([]dto.IPRuleInfo, error)
	CreateIPRule(req dto.IPRuleCreate) error
	DeleteIPRule(req dto.IPRuleDelete) error
}

type FirewallService struct{}

func NewIFirewallService() IFirewallService { return &FirewallService{} }

func (s *FirewallService) GetBaseInfo() (*dto.FirewallBaseInfo, error) {
	info := &dto.FirewallBaseInfo{}

	// 检查 ufw 是否安装
	version, err := cmd.ExecWithOutput("ufw", "version")
	if err != nil {
		info.IsExist = false
		info.Name = "-"
		info.Version = "-"
		return info, nil
	}
	info.IsExist = true
	info.Name = "ufw"

	// 解析版本
	lines := strings.Split(version, "\n")
	if len(lines) > 0 {
		info.Version = strings.TrimPrefix(strings.TrimSpace(lines[0]), "ufw ")
	}

	// 检查状态
	status, _ := cmd.ExecWithOutput("ufw", "status")
	info.IsActive = strings.Contains(status, "Status: active")

	return info, nil
}

func (s *FirewallService) Operate(operation string) error {
	if !isUFWInstalled() {
		return fmt.Errorf("ufw is not installed")
	}
	var args []string
	switch operation {
	case "enable":
		args = []string{"ufw", "--force", "enable"}
	case "disable":
		args = []string{"ufw", "disable"}
	case "reload":
		args = []string{"ufw", "reload"}
	default:
		return fmt.Errorf("unsupported operation: %s", operation)
	}

	output, err := cmd.ExecWithOutput(args[0], args[1:]...)
	if err != nil {
		return fmt.Errorf("%s: %s", err.Error(), output)
	}
	global.LOG.Infof("Firewall %s: %s", operation, strings.TrimSpace(output))
	return nil
}

func (s *FirewallService) ListPortRules(req dto.PortRuleSearch) (int64, []dto.PortRuleInfo, error) {
	if !isUFWInstalled() {
		return 0, nil, nil
	}
	output, err := cmd.ExecWithOutput("ufw", "status", "numbered")
	if err != nil {
		return 0, nil, err
	}

	var rules []dto.PortRuleInfo
	re := regexp.MustCompile(`\[\s*\d+\]\s+(.+?)\s+(ALLOW|DENY)\s+(?:IN\s+)?(.+)`)

	for _, line := range strings.Split(output, "\n") {
		m := re.FindStringSubmatch(strings.TrimSpace(line))
		if m == nil {
			continue
		}
		target := strings.TrimSpace(m[1])
		strategy := strings.ToLower(strings.TrimSpace(m[2]))
		from := strings.TrimSpace(m[3])

		port, proto := parseUFWTarget(target)
		if port == "" {
			continue
		}

		rule := dto.PortRuleInfo{
			Port:     port,
			Protocol: proto,
			Strategy: strategy,
			From:     from,
		}

		// 过滤
		if req.Strategy != "" && req.Strategy != "all" && rule.Strategy != req.Strategy {
			continue
		}
		if req.Info != "" && !strings.Contains(rule.Port, req.Info) && !strings.Contains(rule.From, req.Info) {
			continue
		}

		rules = append(rules, rule)
	}

	// 分页
	total := int64(len(rules))
	start := (req.Page - 1) * req.PageSize
	end := start + req.PageSize
	if start > int(total) {
		return total, nil, nil
	}
	if end > int(total) {
		end = int(total)
	}
	return total, rules[start:end], nil
}

func (s *FirewallService) CreatePortRule(req dto.PortRuleCreate) error {
	if !isUFWInstalled() {
		return fmt.Errorf("ufw is not installed")
	}
	args := []string{"ufw"}
	if req.Strategy == "deny" {
		args = append(args, "deny")
	} else {
		args = append(args, "allow")
	}

	if req.From != "" && req.From != "Anywhere" {
		args = append(args, "from", req.From, "to", "any", "port", req.Port)
	} else {
		args = append(args, req.Port)
	}

	if req.Protocol != "" && req.Protocol != "tcp/udp" {
		args = append(args, "proto", req.Protocol) // 追加到 port 参数之后
		// 重新构建：ufw allow port/proto 格式
		args = []string{"ufw"}
		if req.Strategy == "deny" {
			args = append(args, "deny")
		} else {
			args = append(args, "allow")
		}
		if req.From != "" && req.From != "Anywhere" {
			args = append(args, "from", req.From, "to", "any", "port", req.Port, "proto", req.Protocol)
		} else {
			args = append(args, req.Port+"/"+req.Protocol)
		}
	}

	output, err := cmd.ExecWithOutput(args[0], args[1:]...)
	if err != nil {
		return fmt.Errorf("%s: %s", err.Error(), output)
	}
	global.LOG.Infof("Firewall rule created: %v", args[1:])
	return nil
}

func (s *FirewallService) DeletePortRule(req dto.PortRuleDelete) error {
	if !isUFWInstalled() {
		return fmt.Errorf("ufw is not installed")
	}
	args := []string{"ufw", "delete"}
	if req.Strategy == "deny" {
		args = append(args, "deny")
	} else {
		args = append(args, "allow")
	}

	if req.From != "" && req.From != "Anywhere" {
		args = append(args, "from", req.From, "to", "any", "port", req.Port)
		if req.Protocol != "" && req.Protocol != "tcp/udp" {
			args = append(args, "proto", req.Protocol)
		}
	} else {
		portStr := req.Port
		if req.Protocol != "" && req.Protocol != "tcp/udp" {
			portStr += "/" + req.Protocol
		}
		args = append(args, portStr)
	}

	output, err := cmd.ExecWithOutput(args[0], args[1:]...)
	if err != nil {
		return fmt.Errorf("%s: %s", err.Error(), output)
	}
	global.LOG.Infof("Firewall rule deleted: %v", args[1:])
	return nil
}

func (s *FirewallService) ListIPRules() ([]dto.IPRuleInfo, error) {
	if !isUFWInstalled() {
		return nil, nil
	}
	output, err := cmd.ExecWithOutput("ufw", "status", "numbered")
	if err != nil {
		return nil, err
	}

	var rules []dto.IPRuleInfo
	re := regexp.MustCompile(`\[\s*\d+\]\s+Anywhere\s+(ALLOW|DENY)\s+(?:IN\s+)?(\S+)`)

	for _, line := range strings.Split(output, "\n") {
		m := re.FindStringSubmatch(strings.TrimSpace(line))
		if m == nil {
			continue
		}
		rules = append(rules, dto.IPRuleInfo{
			Strategy: strings.ToLower(m[1]),
			Address:  m[2],
		})
	}
	return rules, nil
}

func (s *FirewallService) CreateIPRule(req dto.IPRuleCreate) error {
	if !isUFWInstalled() {
		return fmt.Errorf("ufw is not installed")
	}
	var args []string
	if req.Strategy == "deny" {
		args = []string{"ufw", "deny", "from", req.Address}
	} else {
		args = []string{"ufw", "allow", "from", req.Address}
	}

	output, err := cmd.ExecWithOutput(args[0], args[1:]...)
	if err != nil {
		return fmt.Errorf("%s: %s", err.Error(), output)
	}
	return nil
}

func (s *FirewallService) DeleteIPRule(req dto.IPRuleDelete) error {
	if !isUFWInstalled() {
		return fmt.Errorf("ufw is not installed")
	}
	var args []string
	if req.Strategy == "deny" {
		args = []string{"ufw", "delete", "deny", "from", req.Address}
	} else {
		args = []string{"ufw", "delete", "allow", "from", req.Address}
	}

	output, err := cmd.ExecWithOutput(args[0], args[1:]...)
	if err != nil {
		return fmt.Errorf("%s: %s", err.Error(), output)
	}
	return nil
}

// isUFWInstalled 检查 ufw 是否安装
func isUFWInstalled() bool {
	_, err := cmd.ExecWithOutput("which", "ufw")
	return err == nil
}

// parseUFWTarget 解析 ufw status 输出中的端口/协议
func parseUFWTarget(target string) (port, protocol string) {
	target = strings.TrimSuffix(target, " (v6)")
	if strings.Contains(target, "/") {
		parts := strings.SplitN(target, "/", 2)
		return parts[0], parts[1]
	}
	// 纯数字或范围
	return target, "tcp/udp"
}
