package dto

type NginxLogAnalysisReq struct {
	SiteID uint `json:"siteId" binding:"required"`
	Days   int  `json:"days"`
}

type NginxLogAnalysis struct {
	TotalRequests   int64             `json:"totalRequests"`
	UniqueIPs       int               `json:"uniqueIPs"`
	TotalBytes      int64             `json:"totalBytes"`
	StatusCodes     map[string]int64  `json:"statusCodes"`
	TopURLs         []RankItem        `json:"topUrls"`
	TopIPs          []RankItem        `json:"topIps"`
	TopUserAgents   []RankItem        `json:"topUserAgents"`
	HourlyStats     []TimeSeriesPoint `json:"hourlyStats"`
	DailyStats      []TimeSeriesPoint `json:"dailyStats"`
	ErrorRate       float64           `json:"errorRate"`
	ThreatRequests  int64             `json:"threatRequests"`
	ThreatIPs       []RankItem        `json:"threatIPs"`
	TopThreats      []RankItem        `json:"topThreats"`
	CrawlerRequests int64             `json:"crawlerRequests"`
	TopCrawlers     []RankItem        `json:"topCrawlers"`
}

type RankItem struct {
	Name    string `json:"name"`
	Count   int64  `json:"count"`
	Country string `json:"country,omitempty"`
	City    string `json:"city,omitempty"`
	Banned  bool   `json:"banned,omitempty"`
}

type TimeSeriesPoint struct {
	Time     string `json:"time"`
	Requests int64  `json:"requests"`
	Bytes    int64  `json:"bytes"`
}

// ---- Nginx 日志分析（全局/按配置文件） ----

type NginxDetectedSite struct {
	Name      string `json:"name"`
	AccessLog string `json:"accessLog"`
	ErrorLog  string `json:"errorLog"`
	ConfFile  string `json:"confFile"`
}

type NginxLogAnalyzeReq struct {
	Site      string `json:"site"`      // 站点名（server_name），空=全部
	TimeRange string `json:"timeRange"` // 1h, 6h, 24h, 7d, 30d
}

type NginxLogTailReq struct {
	Site  string `json:"site"`  // 站点名，空=全部
	Type  string `json:"type"`  // access / error
	Lines int    `json:"lines"` // 行数
}

type NginxLogTailResp struct {
	Content string `json:"content"`
	Path    string `json:"path"`
}

type NginxLogDrilldownReq struct {
	Site        string `json:"site"`
	TimeRange   string `json:"timeRange"`
	FilterType  string `json:"filterType" validate:"required,oneof=url threat"`
	FilterValue string `json:"filterValue" validate:"required"`
}

type NginxLogDrilldownResp struct {
	IPs  []RankItem `json:"ips"`
	URLs []RankItem `json:"urls"`
}
