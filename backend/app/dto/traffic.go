package dto

import "time"

// TrafficConfigCreate 创建/更新流量监控配置
type TrafficConfigCreate struct {
	InterfaceName string `json:"interfaceName" binding:"required"`
	MonthlyLimit  uint64 `json:"monthlyLimit"`
	ResetDay      int    `json:"resetDay" binding:"required,min=1,max=28"`
	Enabled       bool   `json:"enabled"`
}

// TrafficConfigDTO 流量监控配置响应
type TrafficConfigDTO struct {
	ID            uint   `json:"id"`
	InterfaceName string `json:"interfaceName"`
	MonthlyLimit  uint64 `json:"monthlyLimit"`
	ResetDay      int    `json:"resetDay"`
	Enabled       bool   `json:"enabled"`
}

// TrafficStatsRequest 流量统计查询请求
type TrafficStatsRequest struct {
	InterfaceName string `json:"interfaceName" binding:"required"`
	StartTime     string `json:"startTime" binding:"required"` // RFC3339 or 2006-01-02
	EndTime       string `json:"endTime" binding:"required"`
	GroupBy       string `json:"groupBy"` // "hour" or "day", default "day"
}

// TrafficStatsItem 流量统计数据点
type TrafficStatsItem struct {
	Timestamp string `json:"timestamp"`
	BytesSent uint64 `json:"bytesSent"`
	BytesRecv uint64 `json:"bytesRecv"`
}

// TrafficStatsResponse 流量统计查询响应
type TrafficStatsResponse struct {
	InterfaceName string             `json:"interfaceName"`
	Items         []TrafficStatsItem `json:"items"`
	TotalSent     uint64             `json:"totalSent"`
	TotalRecv     uint64             `json:"totalRecv"`
}

// TrafficSummaryItem 单网卡的当前计费周期汇总
type TrafficSummaryItem struct {
	InterfaceName string    `json:"interfaceName"`
	MonthlyLimit  uint64    `json:"monthlyLimit"`
	ResetDay      int       `json:"resetDay"`
	PeriodStart   time.Time `json:"periodStart"`
	PeriodEnd     time.Time `json:"periodEnd"`
	TotalSent     uint64    `json:"totalSent"`
	TotalRecv     uint64    `json:"totalRecv"`
	TotalUsed     uint64    `json:"totalUsed"`
	UsedPercent   float64   `json:"usedPercent"`
	Enabled       bool      `json:"enabled"`
}

// TrafficDeleteConfig 删除流量监控配置
type TrafficDeleteConfig struct {
	InterfaceName string `json:"interfaceName" binding:"required"`
}
