package dto

type NginxLogAnalysisReq struct {
	SiteID uint   `json:"siteId" binding:"required"`
	Days   int    `json:"days"` // 0=today, 7, 30 etc.
}

type NginxLogAnalysis struct {
	TotalRequests int64              `json:"totalRequests"`
	UniqueIPs     int                `json:"uniqueIPs"`
	TotalBytes    int64              `json:"totalBytes"`
	StatusCodes   map[string]int64   `json:"statusCodes"`
	TopURLs       []RankItem         `json:"topUrls"`
	TopIPs        []RankItem         `json:"topIps"`
	TopUserAgents []RankItem         `json:"topUserAgents"`
	HourlyStats   []TimeSeriesPoint  `json:"hourlyStats"`
	DailyStats    []TimeSeriesPoint  `json:"dailyStats"`
	ErrorRate     float64            `json:"errorRate"`
}

type RankItem struct {
	Name  string `json:"name"`
	Count int64  `json:"count"`
}

type TimeSeriesPoint struct {
	Time     string `json:"time"`
	Requests int64  `json:"requests"`
	Bytes    int64  `json:"bytes"`
}
