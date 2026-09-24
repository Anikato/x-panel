package dto

import "time"

type NginxLogAnalysisReq struct {
	SiteID  uint `json:"siteId" binding:"required"`
	Days    int  `json:"days"`
	Refresh bool `json:"refresh"`
}

type NginxLogAnalysis struct {
	TotalRequests   int64                `json:"totalRequests"`
	UniqueIPs       int                  `json:"uniqueIPs"`
	TotalBytes      int64                `json:"totalBytes"`
	StatusCodes     map[string]int64     `json:"statusCodes"`
	TopURLs         []RankItem           `json:"topUrls"`
	TopIPs          []RankItem           `json:"topIps"`
	TopUserAgents   []RankItem           `json:"topUserAgents"`
	HourlyStats     []TimeSeriesPoint    `json:"hourlyStats"`
	DailyStats      []TimeSeriesPoint    `json:"dailyStats"`
	ErrorRate       float64              `json:"errorRate"`
	ThreatRequests  int64                `json:"threatRequests"`
	ThreatIPs       []RankItem           `json:"threatIPs"`
	TopThreats      []RankItem           `json:"topThreats"`
	CrawlerRequests int64                `json:"crawlerRequests"`
	TopCrawlers     []RankItem           `json:"topCrawlers"`
	AvgRequestMs    float64              `json:"avgRequestMs"`
	SlowRequests    int64                `json:"slowRequests"`
	TopSlowURLs     []SlowURL            `json:"topSlowUrls"`
	ErrorSummary    []RankItem           `json:"errorSummary"`
	Meta            NginxLogAnalysisMeta `json:"meta"`
}

type SlowURL struct {
	Name  string  `json:"name"`
	Count int64   `json:"count"`
	AvgMs float64 `json:"avgMs"`
	MaxMs float64 `json:"maxMs"`
}

type NginxLogAnalysisMeta struct {
	RequestedFrom  time.Time           `json:"requestedFrom"`
	RequestedTo    time.Time           `json:"requestedTo"`
	ObservedFrom   *time.Time          `json:"observedFrom,omitempty"`
	ObservedTo     *time.Time          `json:"observedTo,omitempty"`
	GeneratedAt    time.Time           `json:"generatedAt"`
	ScannedBytes   int64               `json:"scannedBytes"`
	ScannedLines   int64               `json:"scannedLines"`
	MatchedLines   int64               `json:"matchedLines"`
	InvalidLines   int64               `json:"invalidLines"`
	Partial        bool                `json:"partial"`
	Reasons        []string            `json:"reasons,omitempty"`
	FilesTotal     int                 `json:"filesTotal"`
	FilesSucceeded int                 `json:"filesSucceeded"`
	FilesFailed    int                 `json:"filesFailed"`
	FailedFiles    []NginxLogFileIssue `json:"failedFiles,omitempty"`
	SkippedFiles   []NginxLogFileIssue `json:"skippedFiles,omitempty"`
	SourceScope    string              `json:"sourceScope"`
	CacheHit       bool                `json:"cacheHit"`
	Recalculated   bool                `json:"recalculated,omitempty"`
	SharedLog      bool                `json:"sharedLog,omitempty"`
}

type NginxLogFileIssue struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
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
	Refresh   bool   `json:"refresh"`
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
	SiteID      uint   `json:"siteId"`
	Days        int    `json:"days"`
	TimeRange   string `json:"timeRange"`
	FilterType  string `json:"filterType" validate:"required,oneof=url ip threat"`
	FilterValue string `json:"filterValue" validate:"required"`
}

type NginxLogDrilldownResp struct {
	IPs  []RankItem           `json:"ips"`
	URLs []RankItem           `json:"urls"`
	Meta NginxLogAnalysisMeta `json:"meta"`
}
