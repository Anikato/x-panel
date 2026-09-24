package service

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"xpanel/app/dto"
	"xpanel/app/repo"
	"xpanel/buserr"
	"xpanel/constant"
	"xpanel/global"
	"xpanel/utils/iplocation"
)

type INginxLogService interface {
	Analyze(ctx context.Context, req dto.NginxLogAnalysisReq) (*dto.NginxLogAnalysis, error)
	DetectSites() ([]dto.NginxDetectedSite, error)
	AnalyzeSite(ctx context.Context, req dto.NginxLogAnalyzeReq) (*dto.NginxLogAnalysis, error)
	TailLog(req dto.NginxLogTailReq) (*dto.NginxLogTailResp, error)
	Drilldown(ctx context.Context, req dto.NginxLogDrilldownReq) (*dto.NginxLogDrilldownResp, error)
}

type NginxLogService struct {
	websiteRepo repo.IWebsiteRepo
}

func NewINginxLogService() INginxLogService {
	return &NginxLogService{websiteRepo: repo.NewIWebsiteRepo()}
}

// combined log format regex
var combinedLogRe = regexp.MustCompile(
	`^(\S+)\s+\S+\s+\S+\s+\[([^\]]+)\]\s+"([^"]*?)"\s+(\d{3})\s+(\d+)\s+"[^"]*"\s+"([^"]*)"`,
)

type logEntry struct {
	IP              string
	Time            time.Time
	Method          string
	URL             string
	Status          int
	Bytes           int64
	UserAgent       string
	RequestSeconds  float64
	HasRequestTime  bool
	UpstreamSeconds float64
	HasUpstreamTime bool
}

// Analyze handles legacy site-ID based analysis
func (s *NginxLogService) Analyze(ctx context.Context, req dto.NginxLogAnalysisReq) (*dto.NginxLogAnalysis, error) {
	site, err := s.websiteRepo.Get(repo.WithByID(req.SiteID))
	if err != nil {
		return nil, buserr.New(constant.ErrRecordNotFound)
	}

	nc := global.CONF.Nginx
	if !nc.IsInstalled() {
		return nil, buserr.New(constant.ErrNginxNotInstalled)
	}

	logDir := fmt.Sprintf("%s/sites", nc.GetLogDir())
	logPath := fmt.Sprintf("%s/%s.access.log", logDir, site.PrimaryDomain)
	if strings.TrimSpace(site.AccessLogPath) != "" {
		logPath = strings.TrimSpace(site.AccessLogPath)
	}
	days := req.Days
	if days <= 0 {
		days = 1
	}
	until := time.Now()
	cutoff := until.AddDate(0, 0, -days)
	key := fmt.Sprintf("%s|legacy|%d|%d", logAnalysisStrategyVersion, req.SiteID, days)
	errorPath := fmt.Sprintf("%s/%s.error.log", logDir, site.PrimaryDomain)
	if strings.TrimSpace(site.ErrorLogPath) != "" {
		errorPath = strings.TrimSpace(site.ErrorLogPath)
	}
	return runSiteAnalysis(ctx, key, req.Refresh, []string{logPath}, []string{errorPath}, cutoff, until, true, false)
}

// DetectSites scans Nginx config files and extracts server_name + log paths
func (s *NginxLogService) DetectSites() ([]dto.NginxDetectedSite, error) {
	nc := global.CONF.Nginx
	if !nc.IsInstalled() {
		return nil, buserr.New(constant.ErrNginxNotInstalled)
	}

	active, err := collectActiveNginxConfigPaths(nc.GetMainConf())
	if err != nil {
		return nil, err
	}
	confFiles := make([]string, 0, len(active))
	for path := range active {
		confFiles = append(confFiles, path)
	}
	sort.Strings(confFiles)

	seen := make(map[string]bool)
	var sites []dto.NginxDetectedSite
	defaultLogDir := nc.GetLogDir()

	for _, cf := range confFiles {
		parsed := parseNginxConfForSites(cf, defaultLogDir)
		for _, site := range parsed {
			if seen[site.Name] {
				continue
			}
			seen[site.Name] = true
			sites = append(sites, site)
		}
	}

	sort.Slice(sites, func(i, j int) bool { return sites[i].Name < sites[j].Name })
	return sites, nil
}

// AnalyzeSite analyzes logs for a specific site or all sites
func (s *NginxLogService) AnalyzeSite(ctx context.Context, req dto.NginxLogAnalyzeReq) (*dto.NginxLogAnalysis, error) {
	nc := global.CONF.Nginx
	if !nc.IsInstalled() {
		return nil, buserr.New(constant.ErrNginxNotInstalled)
	}
	until := time.Now()
	cutoff := parseCutoffAt(req.TimeRange, until)
	paths, shared, err := s.resolveAccessLogs(req.Site)
	if err != nil {
		return nil, err
	}
	key := fmt.Sprintf("%s|analyze|%s|%s", logAnalysisStrategyVersion, req.Site, req.TimeRange)
	return runSiteAnalysis(ctx, key, req.Refresh, paths, s.resolveErrorLogs(req.Site), cutoff, until, true, shared)
}

// TailLog returns the last N lines of access or error log
func (s *NginxLogService) TailLog(req dto.NginxLogTailReq) (*dto.NginxLogTailResp, error) {
	nc := global.CONF.Nginx
	if !nc.IsInstalled() {
		return nil, buserr.New(constant.ErrNginxNotInstalled)
	}

	sites, err := s.DetectSites()
	if err != nil {
		return nil, err
	}

	lines := req.Lines
	if lines <= 0 {
		lines = 200
	}
	if lines > 5000 {
		lines = 5000
	}

	var logPath string
	if req.Site == "" {
		defaultLogDir := nc.GetLogDir()
		if req.Type == "error" {
			logPath = filepath.Join(defaultLogDir, "error.log")
		} else {
			logPath = filepath.Join(defaultLogDir, "access.log")
		}
	} else {
		for _, site := range sites {
			if site.Name == req.Site {
				if req.Type == "error" {
					logPath = site.ErrorLog
				} else {
					logPath = site.AccessLog
				}
				break
			}
		}
	}

	if logPath == "" || logPath == "off" {
		return &dto.NginxLogTailResp{}, nil
	}

	content, err := tailFile(logPath, lines)
	if err != nil {
		return &dto.NginxLogTailResp{}, nil
	}

	return &dto.NginxLogTailResp{Content: content, Path: logPath}, nil
}

// Drilldown returns detailed IPs/URLs for a given URL or threat category
func (s *NginxLogService) Drilldown(ctx context.Context, req dto.NginxLogDrilldownReq) (*dto.NginxLogDrilldownResp, error) {
	nc := global.CONF.Nginx
	if !nc.IsInstalled() {
		return nil, buserr.New(constant.ErrNginxNotInstalled)
	}
	until := time.Now()
	cutoff := parseCutoffAt(req.TimeRange, until)
	if req.Days > 0 {
		cutoff = until.AddDate(0, 0, -req.Days)
	}
	var paths []string
	var shared bool
	var err error
	if req.SiteID > 0 {
		paths, err = s.websiteAccessLogs(req.SiteID)
	} else {
		paths, shared, err = s.resolveAccessLogs(req.Site)
	}
	if err != nil {
		return nil, err
	}
	key := fmt.Sprintf("%s|drill|%d|%s|%s|%d|%s|%s", logAnalysisStrategyVersion, req.SiteID, req.Site, req.TimeRange, req.Days, req.FilterType, req.FilterValue)
	if item, ok := logAnalysisCache.get(key); ok && item.drill != nil {
		out := *item.drill
		out.Meta.CacheHit = true
		return &out, nil
	}
	return logAnalysisFlights.doDrill(ctx, key, func(runCtx context.Context) (*dto.NginxLogDrilldownResp, error) {
		if item, ok := logAnalysisCache.get(key); ok && item.drill != nil {
			out := *item.drill
			out.Meta.CacheHit = true
			return &out, nil
		}
		limits := defaultScanLimits()
		sink := newDrillSink(req.FilterType, req.FilterValue, limits.MaxKeys)
		meta := scanLogFiles(runCtx, paths, cutoff, until, limits, sink)
		if runCtx.Err() != nil {
			return nil, runCtx.Err()
		}
		if meta.filesTotal > 0 && meta.filesSucceeded == 0 && meta.filesFailed == meta.filesTotal {
			return nil, buserr.WithDetail(constant.ErrNginxLogAllFailed, "all access logs failed", nil)
		}
		if sink.capped {
			meta.addReason("key_limit")
		}
		if shared {
			meta.addReason("shared_log_unattributed")
		}
		resp := &dto.NginxLogDrilldownResp{
			IPs:  topNWithGeo(sink.ipCount, 50, true),
			URLs: topN(sink.urlCount, 50),
			Meta: meta.toMeta(cutoff, until, time.Now()),
		}
		resp.Meta.Recalculated = true
		resp.Meta.SharedLog = shared
		markBanned(resp.IPs, loadBannedIPSet())
		logAnalysisCache.put(&analysisCacheItem{
			key: key, drill: resp, size: estimateDrillSize(resp), expires: time.Now().Add(logCacheTTL),
		})
		return resp, nil
	})
}

func (s *NginxLogService) websiteAccessLogs(id uint) ([]string, error) {
	site, err := s.websiteRepo.Get(repo.WithByID(id))
	if err != nil {
		return nil, buserr.New(constant.ErrRecordNotFound)
	}
	logDir := fmt.Sprintf("%s/sites", global.CONF.Nginx.GetLogDir())
	logPath := fmt.Sprintf("%s/%s.access.log", logDir, site.PrimaryDomain)
	if strings.TrimSpace(site.AccessLogPath) != "" {
		logPath = strings.TrimSpace(site.AccessLogPath)
	}
	return []string{logPath}, nil
}

func (s *NginxLogService) resolveErrorLogs(site string) []string {
	sites, err := s.DetectSites()
	if err != nil {
		return nil
	}
	if site == "" {
		var paths []string
		for _, item := range sites {
			if item.ErrorLog != "" && item.ErrorLog != "off" {
				paths = append(paths, item.ErrorLog)
			}
		}
		return dedupStrings(paths)
	}
	for _, item := range sites {
		if item.Name == site && item.ErrorLog != "" && item.ErrorLog != "off" {
			return []string{item.ErrorLog}
		}
	}
	return nil
}

func (s *NginxLogService) resolveAccessLogs(site string) ([]string, bool, error) {
	sites, err := s.DetectSites()
	if err != nil {
		return nil, false, err
	}
	if site == "" {
		var paths []string
		for _, item := range sites {
			if item.AccessLog != "" && item.AccessLog != "off" {
				paths = append(paths, item.AccessLog)
			}
		}
		return dedupStrings(paths), false, nil
	}
	var chosen string
	for _, item := range sites {
		if item.Name == site {
			chosen = item.AccessLog
			break
		}
	}
	if chosen == "" || chosen == "off" {
		return nil, false, nil
	}
	owners := 0
	for _, item := range sites {
		if item.AccessLog == chosen {
			owners++
		}
	}
	return []string{chosen}, owners > 1, nil
}

func runSiteAnalysis(ctx context.Context, key string, refresh bool, paths, errorPaths []string, cutoff, until time.Time, withGeo, shared bool) (*dto.NginxLogAnalysis, error) {
	if !refresh {
		if item, ok := logAnalysisCache.get(key); ok && item.result != nil {
			out := *item.result
			out.Meta.CacheHit = true
			return &out, nil
		}
	}
	return logAnalysisFlights.doAnalysis(ctx, key, func(runCtx context.Context) (*dto.NginxLogAnalysis, error) {
		if !refresh {
			if item, ok := logAnalysisCache.get(key); ok && item.result != nil {
				out := *item.result
				out.Meta.CacheHit = true
				return &out, nil
			}
		}
		limits := defaultScanLimits()
		if len(paths) == 0 {
			res := &dto.NginxLogAnalysis{StatusCodes: map[string]int64{}}
			empty := scanMetaAcc{}
			res.Meta = empty.toMeta(cutoff, until, time.Now())
			return res, nil
		}
		sink := newStatsSink(limits.MaxKeys)
		meta := scanLogFiles(runCtx, paths, cutoff, until, limits, sink)
		if runCtx.Err() != nil {
			return nil, runCtx.Err()
		}
		if meta.filesTotal > 0 && meta.filesSucceeded == 0 && meta.filesFailed == meta.filesTotal {
			return nil, buserr.WithDetail(constant.ErrNginxLogAllFailed, "all access logs failed", nil)
		}
		if meta.matchedLines == 0 && meta.invalidLines > 0 {
			return nil, buserr.New(constant.ErrNginxLogUnsupportedFormat)
		}
		if sink.capped {
			meta.addReason("key_limit")
		}
		if shared {
			meta.addReason("shared_log_unattributed")
		}
		result := sink.result(withGeo, loadBannedIPSet())
		result.ErrorSummary = summarizeErrorLogs(errorPaths, cutoff, until)
		result.Meta = meta.toMeta(cutoff, until, time.Now())
		result.Meta.SharedLog = shared
		logAnalysisCache.put(&analysisCacheItem{
			key: key, result: result, size: estimateAnalysisSize(result), expires: time.Now().Add(logCacheTTL),
		})
		return result, nil
	})
}

// --- Nginx config parsing ---

var (
	serverNameRe = regexp.MustCompile(`(?i)^\s*server_name\s+(.+?)\s*;`)
	accessLogRe  = regexp.MustCompile(`(?i)^\s*access_log\s+([^\s;]+)`)
	errorLogRe   = regexp.MustCompile(`(?i)^\s*error_log\s+([^\s;]+)`)
)

func parseNginxConfForSites(confPath, defaultLogDir string) []dto.NginxDetectedSite {
	data, err := os.ReadFile(confPath)
	if err != nil {
		return nil
	}

	lines := strings.Split(string(data), "\n")
	var sites []dto.NginxDetectedSite

	depth := 0
	inServer := false
	serverDepth := 0

	var curNames []string
	var curAccess, curError string

	flushServer := func() {
		if len(curNames) == 0 {
			curNames = []string{"_"}
		}
		name := strings.Join(curNames, " ")
		if name == "_" || name == "localhost" || name == "" {
			name = filepath.Base(confPath)
		}
		sites = append(sites, dto.NginxDetectedSite{
			Name:      name,
			AccessLog: curAccess,
			ErrorLog:  curError,
			ConfFile:  confPath,
		})
		curNames = nil
		curAccess = ""
		curError = ""
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		opens := strings.Count(trimmed, "{")
		closes := strings.Count(trimmed, "}")

		if !inServer && strings.HasPrefix(trimmed, "server") && strings.Contains(trimmed, "{") {
			inServer = true
			serverDepth = depth
			depth += opens - closes
			continue
		}

		if inServer {
			if m := serverNameRe.FindStringSubmatch(trimmed); m != nil {
				names := strings.Fields(m[1])
				curNames = append(curNames, names...)
			}
			if m := accessLogRe.FindStringSubmatch(trimmed); m != nil {
				curAccess = m[1]
			}
			if m := errorLogRe.FindStringSubmatch(trimmed); m != nil {
				curError = m[1]
			}
		}

		depth += opens - closes

		if inServer && depth <= serverDepth {
			flushServer()
			inServer = false
		}
	}

	for i := range sites {
		if sites[i].AccessLog == "" {
			sites[i].AccessLog = filepath.Join(defaultLogDir, "access.log")
		}
		if sites[i].ErrorLog == "" {
			sites[i].ErrorLog = filepath.Join(defaultLogDir, "error.log")
		}
	}

	return sites
}

// --- Log parsing ---

func parseAccessLog(path string, cutoff time.Time, maxLines int) ([]logEntry, error) {
	limits := defaultScanLimits()
	if maxLines > 0 && int64(maxLines) < limits.MaxLines {
		limits.MaxLines = int64(maxLines)
	}
	sink := &collectSink{}
	meta := scanLogFiles(context.Background(), []string{path}, cutoff, time.Now(), limits, sink)
	if meta.filesFailed > 0 && meta.filesSucceeded == 0 {
		return nil, fmt.Errorf("read log: %s", path)
	}
	return sink.entries, nil
}

func readLastLines(f *os.File, n int) ([]string, error) {
	stat, err := f.Stat()
	if err != nil {
		return nil, err
	}
	size := stat.Size()
	if size == 0 || n <= 0 {
		return nil, nil
	}

	const chunkSize = 64 * 1024
	var chunks [][]byte
	offset := size
	newlines := 0
	for offset > 0 && newlines <= n {
		readSize := int64(chunkSize)
		if offset < readSize {
			readSize = offset
		}
		offset -= readSize
		buf := make([]byte, readSize)
		if _, err := f.ReadAt(buf, offset); err != nil && err != io.EOF {
			return nil, err
		}
		chunks = append(chunks, buf)
		for _, b := range buf {
			if b == '\n' {
				newlines++
			}
		}
	}
	total := 0
	for _, c := range chunks {
		total += len(c)
	}
	data := make([]byte, 0, total)
	for i := len(chunks) - 1; i >= 0; i-- {
		data = append(data, chunks[i]...)
	}
	text := string(data)
	parts := strings.Split(text, "\n")
	if offset > 0 && len(parts) > 0 {
		parts = parts[1:]
	}
	if len(parts) > 0 && parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}
	if len(parts) > n {
		parts = parts[len(parts)-n:]
	}
	return parts, nil
}

// --- Threat detection ---

var threatPatterns = []struct {
	Name    string
	Pattern *regexp.Regexp
}{
	{"PHP 探测", regexp.MustCompile(`(?i)\.(php|asp|aspx|jsp|cgi)\b`)},
	{"WordPress 扫描", regexp.MustCompile(`(?i)(wp-admin|wp-login|wp-content|xmlrpc\.php)`)},
	{"路径遍历", regexp.MustCompile(`(\.\./|\.\.%2[fF])`)},
	{"敏感文件", regexp.MustCompile(`(?i)(\.(env|git|bak|sql|tar|gz|zip|rar)|config\.(json|yaml|yml|php)|\.htaccess|\.DS_Store)`)},
	{"SQL 注入", regexp.MustCompile(`(?i)(union\s+select|or\s+1\s*=\s*1|'\s*(or|and)\s+'|--\s*$)`)},
	{"Shell/命令注入", regexp.MustCompile(`(?i)(/etc/passwd|/bin/(sh|bash)|cmd=|exec\(|system\()`)},
	{"扫描器探测", regexp.MustCompile(`(?i)(/actuator|/solr|/api/v1/pods|/manager/html|/console|/phpmyadmin|/admin)`)},
}

func classifyThreat(url string) string {
	for _, tp := range threatPatterns {
		if tp.Pattern.MatchString(url) {
			return tp.Name
		}
	}
	return ""
}

// --- Crawler detection ---

var crawlerPatterns = []struct {
	Name    string
	Pattern *regexp.Regexp
}{
	{"Googlebot", regexp.MustCompile(`(?i)googlebot`)},
	{"Bingbot", regexp.MustCompile(`(?i)bingbot`)},
	{"Baidu Spider", regexp.MustCompile(`(?i)baiduspider`)},
	{"YandexBot", regexp.MustCompile(`(?i)yandexbot`)},
	{"DuckDuckBot", regexp.MustCompile(`(?i)duckduckbot`)},
	{"Sogou", regexp.MustCompile(`(?i)sogou`)},
	{"Bytespider", regexp.MustCompile(`(?i)bytespider`)},
	{"GPTBot", regexp.MustCompile(`(?i)gptbot`)},
	{"ClaudeBot", regexp.MustCompile(`(?i)claudebot`)},
	{"Applebot", regexp.MustCompile(`(?i)applebot`)},
	{"SemrushBot", regexp.MustCompile(`(?i)semrushbot`)},
	{"AhrefsBot", regexp.MustCompile(`(?i)ahrefsbot`)},
	{"其他爬虫", regexp.MustCompile(`(?i)(bot|crawler|spider|scraper|crawl)`)},
}

func classifyCrawler(ua string) string {
	for _, cp := range crawlerPatterns {
		if cp.Pattern.MatchString(ua) {
			return cp.Name
		}
	}
	return ""
}

func loadBannedIPSet() map[string]bool {
	result := make(map[string]bool)
	svc := NewIFail2banService()
	banned, err := svc.ListBanned()
	if err != nil {
		return result
	}
	for _, b := range banned {
		result[b.IP] = true
	}
	return result
}

func markBanned(items []dto.RankItem, bannedIPs map[string]bool) {
	for i := range items {
		if bannedIPs[items[i].Name] {
			items[i].Banned = true
		}
	}
}

func topN(m map[string]int64, n int) []dto.RankItem {
	items := make([]dto.RankItem, 0, len(m))
	for k, v := range m {
		items = append(items, dto.RankItem{Name: k, Count: v})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Count > items[j].Count })
	if len(items) > n {
		items = items[:n]
	}
	return items
}

func topNWithGeo(m map[string]int64, n int, withGeo bool) []dto.RankItem {
	items := topN(m, n)
	if !withGeo {
		return items
	}
	ipSvc := iplocation.GetService()
	for i := range items {
		info := ipSvc.Lookup(items[i].Name)
		items[i].Country = info.Country
		items[i].City = info.City
	}
	return items
}

func timeSeries(reqs, bytes map[string]int64) []dto.TimeSeriesPoint {
	points := make([]dto.TimeSeriesPoint, 0, len(reqs))
	for k, v := range reqs {
		points = append(points, dto.TimeSeriesPoint{
			Time:     k,
			Requests: v,
			Bytes:    bytes[k],
		})
	}
	sort.Slice(points, func(i, j int) bool { return points[i].Time < points[j].Time })
	return points
}

// --- Helpers ---

func parseCutoff(timeRange string) time.Time {
	return parseCutoffAt(timeRange, time.Now())
}

func parseCutoffAt(timeRange string, now time.Time) time.Time {
	switch timeRange {
	case "1h":
		return now.Add(-1 * time.Hour)
	case "6h":
		return now.Add(-6 * time.Hour)
	case "24h":
		return now.Add(-24 * time.Hour)
	case "7d":
		return now.AddDate(0, 0, -7)
	case "30d":
		return now.AddDate(0, 0, -30)
	default:
		return now.Add(-24 * time.Hour)
	}
}

func maxLinesForRange(timeRange string) int {
	switch timeRange {
	case "1h":
		return 50000
	case "6h":
		return 100000
	case "24h":
		return 200000
	case "7d":
		return 500000
	case "30d":
		return 1000000
	default:
		return 200000
	}
}

func tailFile(path string, lines int) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	result, err := readLastLines(f, lines)
	if err != nil {
		return "", err
	}
	return strings.Join(result, "\n"), nil
}

func dedupStrings(ss []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, s := range ss {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}
