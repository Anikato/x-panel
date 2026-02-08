package dto

// FirewallBaseInfo 防火墙基本信息
type FirewallBaseInfo struct {
	IsExist  bool   `json:"isExist"`
	IsActive bool   `json:"isActive"`
	Name     string `json:"name"`    // ufw / firewalld
	Version  string `json:"version"`
}

// FirewallOperateReq 防火墙操作请求
type FirewallOperateReq struct {
	Operation string `json:"operation" binding:"required,oneof=enable disable reload"`
}

// PortRuleInfo 端口规则信息
type PortRuleInfo struct {
	Port     string `json:"port"`     // e.g. "80" or "8000:8100"
	Protocol string `json:"protocol"` // tcp, udp, tcp/udp
	Strategy string `json:"strategy"` // allow, deny
	From     string `json:"from"`     // 来源 IP（Anywhere 表示任意）
}

// PortRuleSearch 端口规则搜索
type PortRuleSearch struct {
	PageInfo
	Info     string `json:"info"`
	Strategy string `json:"strategy"` // allow, deny, all
}

// PortRuleCreate 创建端口规则
type PortRuleCreate struct {
	Port     string `json:"port" binding:"required"`
	Protocol string `json:"protocol" binding:"required,oneof=tcp udp tcp/udp"`
	Strategy string `json:"strategy" binding:"required,oneof=allow deny"`
	From     string `json:"from"` // 可选，限制来源 IP
}

// PortRuleDelete 删除端口规则
type PortRuleDelete struct {
	Port     string `json:"port" binding:"required"`
	Protocol string `json:"protocol" binding:"required"`
	Strategy string `json:"strategy" binding:"required"`
	From     string `json:"from"`
}

// IPRuleInfo IP 规则信息
type IPRuleInfo struct {
	Address  string `json:"address"`
	Strategy string `json:"strategy"` // allow, deny
}

// IPRuleCreate 创建 IP 规则
type IPRuleCreate struct {
	Address  string `json:"address" binding:"required"`
	Strategy string `json:"strategy" binding:"required,oneof=allow deny"`
}

// IPRuleDelete 删除 IP 规则
type IPRuleDelete struct {
	Address  string `json:"address" binding:"required"`
	Strategy string `json:"strategy" binding:"required"`
}

// ForwardRuleInfo 转发规则信息
type ForwardRuleInfo struct {
	Port     string `json:"port"`
	Protocol string `json:"protocol"`
	TargetIP string `json:"targetIP"`
	TargetPort string `json:"targetPort"`
}

// ForwardRuleCreate 创建转发规则
type ForwardRuleCreate struct {
	Port       string `json:"port" binding:"required"`
	Protocol   string `json:"protocol" binding:"required,oneof=tcp udp"`
	TargetIP   string `json:"targetIP" binding:"required"`
	TargetPort string `json:"targetPort" binding:"required"`
}
