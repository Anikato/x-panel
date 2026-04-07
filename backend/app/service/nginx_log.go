package service

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
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
	Analyze(req dto.NginxLogAnalysisReq) (*dto.NginxLogAnalysis, error)
	DetectSites() ([]dto.NginxDetectedSite, error)
	AnalyzeSite(req dto.NginxLogAnalyzeReq) (*dto.NginxLogAnalysis, error)
	TailLog(req dto.NginxLogTailReq) (*dto.NginxLogTailResp, error)
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
	IP        string
	Time      time.Time
	Method    string
	URL       string
	Status    int
	Bytes     int64
	UserAgent string
}

// Analyze handles legacy site-ID based analysis
func (s *NginxLogService) Analyze(req dto.NginxLogAnalysisReq) (*dto.NginxLogAnalysis, error) {
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
	if _, err := os.Stat(logPath); err != nil {
		return &dto.NginxLogAnalysis{StatusCodes: make(map[string]int64)}, nil
	}

	days := req.Days
	if days <= 0 {
		days = 1
	}
	cutoff := time.Now().AddDate(0, 0, -days)
	entries, err := parseAccessLog(logPath, cutoff, 0)
	if err != nil {
		return nil, fmt.Errorf("parse log: %v", err)
	}
	return aggregate(entries, days, false, nil), nil
}

// DetectSites scans Nginx config files and extracts server_name + log paths
func (s *NginxLogService) DetectSites() ([]dto.NginxDetectedSite, error) {
	nc := global.CONF.Nginx
	if !nc.IsInstalled() {
		return nil, buserr.New(constant.ErrNginxNotInstalled)
	}

	var confFiles []string
	confDDir := filepath.Join(nc.GetConfDir(), "conf.d")
	if entries, err := os.ReadDir(confDDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".conf") {
				confFiles = append(confFiles, filepath.Join(confDDir, e.Name()))
			}
		}
	}

	if nc.IsSystemMode() {
		sitesDir := nc.GetSitesDir()
		if entries, err := os.ReadDir(sitesDir); err == nil {
			for _, e := range entries {
				p := filepath.Join(sitesDir, e.Name())
				if e.Type()&os.ModeSymlink != 0 {
					if target, err := filepath.EvalSymlinks(p); err == nil {
						p = target
					}
				}
				if !e.IsDir() {
					confFiles = append(confFiles, p)
				}
			}
		}
	}

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
func (s *NginxLogService) AnalyzeSite(req dto.NginxLogAnalyzeReq) (*dto.NginxLogAnalysis, error) {
	nc := global.CONF.Nginx
	if !nc.IsInstalled() {
		return nil, buserr.New(constant.ErrNginxNotInstalled)
	}

	cutoff := parseCutoff(req.TimeRange)
	maxLines := maxLinesForRange(req.TimeRange)

	sites, err := s.DetectSites()
	if err != nil {
		return nil, err
	}

	var logPaths []string
	if req.Site == "" {
		for _, site := range sites {
			if site.AccessLog != "" && site.AccessLog != "off" {
				logPaths = append(logPaths, site.AccessLog)
			}
		}
	} else {
		for _, site := range sites {
			if site.Name == req.Site {
				if site.AccessLog != "" && site.AccessLog != "off" {
					logPaths = append(logPaths, site.AccessLog)
				}
				break
			}
		}
	}

	if len(logPaths) == 0 {
		return &dto.NginxLogAnalysis{StatusCodes: make(map[string]int64)}, nil
	}

	dedupPaths := dedupStrings(logPaths)
	var allEntries []logEntry
	for _, p := range dedupPaths {
		if _, err := os.Stat(p); err != nil {
			continue
		}
		entries, err := parseAccessLog(p, cutoff, maxLines)
		if err != nil {
			continue
		}
		allEntries = append(allEntries, entries...)
	}

	days := daysFromRange(req.TimeRange)
	bannedSet := loadBannedIPSet()
	return aggregate(allEntries, days, true, bannedSet), nil
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
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var lines []string
	if maxLines > 0 {
		lines, err = readLastLines(f, maxLines)
		if err != nil {
			return nil, err
		}
	} else {
		scanner := bufio.NewScanner(f)
		buf := make([]byte, 0, 256*1024)
		scanner.Buffer(buf, 1024*1024)
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			return nil, err
		}
	}

	var entries []logEntry
	for _, line := range lines {
		m := combinedLogRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}

		t, err := time.Parse("02/Jan/2006:15:04:05 -0700", m[2])
		if err != nil {
			continue
		}
		if t.Before(cutoff) {
			continue
		}

		status, _ := strconv.Atoi(m[4])
		bytes, _ := strconv.ParseInt(m[5], 10, 64)

		parts := strings.SplitN(m[3], " ", 3)
		method, url := "", ""
		if len(parts) >= 2 {
			method = parts[0]
			url = parts[1]
		}

		entries = append(entries, logEntry{
			IP:        m[1],
			Time:      t,
			Method:    method,
			URL:       url,
			Status:    status,
			Bytes:     bytes,
			UserAgent: m[6],
		})
	}
	return entries, nil
}

func readLastLines(f *os.File, n int) ([]string, error) {
	stat, err := f.Stat()
	if err != nil {
		return nil, err
	}
	size := stat.Size()
	if size == 0 {
		return nil, nil
	}

	chunkSize := int64(64 * 1024)
	var lines []string
	offset := size

	for offset > 0 && len(lines) < n+1 {
		readSize := chunkSize
		if offset < readSize {
			readSize = offset
		}
		offset -= readSize

		buf := make([]byte, readSize)
		if _, err := f.ReadAt(buf, offset); err != nil && err != io.EOF {
			return nil, err
		}

		chunk := string(buf)
		chunkLines := strings.Split(chunk, "\n")

		if len(lines) > 0 {
			chunkLines[len(chunkLines)-1] += lines[0]
			lines = lines[1:]
		}
		lines = append(chunkLines, lines...)
	}

	if len(lines) > 0 && lines[0] == "" {
		lines = lines[1:]
	}
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return lines, nil
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

// --- Aggregation ---

func aggregate(entries []logEntry, days int, withGeo bool, bannedIPs map[string]bool) *dto.NginxLogAnalysis {
	result := &dto.NginxLogAnalysis{
		TotalRequests: int64(len(entries)),
		StatusCodes:   make(map[string]int64),
	}

	if len(entries) == 0 {
		return result
	}

	ipSet := make(map[string]struct{})
	urlCount := make(map[string]int64)
	ipCount := make(map[string]int64)
	uaCount := make(map[string]int64)
	hourlyReqs := make(map[string]int64)
	hourlyBytes := make(map[string]int64)
	dailyReqs := make(map[string]int64)
	dailyBytes := make(map[string]int64)
	threatCatCount := make(map[string]int64)
	threatIPCount := make(map[string]int64)

	var errors int64

	for _, e := range entries {
		ipSet[e.IP] = struct{}{}
		result.TotalBytes += e.Bytes

		cat := fmt.Sprintf("%dxx", e.Status/100)
		result.StatusCodes[cat]++
		if e.Status >= 400 {
			errors++
		}

		urlCount[e.URL]++
		ipCount[e.IP]++
		uaCount[e.UserAgent]++

		if tc := classifyThreat(e.URL); tc != "" {
			result.ThreatRequests++
			threatCatCount[tc]++
			threatIPCount[e.IP]++
		}

		h := e.Time.Format("2006-01-02 15:00")
		hourlyReqs[h]++
		hourlyBytes[h] += e.Bytes

		d := e.Time.Format("2006-01-02")
		dailyReqs[d]++
		dailyBytes[d] += e.Bytes
	}

	result.UniqueIPs = len(ipSet)
	if result.TotalRequests > 0 {
		result.ErrorRate = float64(errors) / float64(result.TotalRequests) * 100
	}

	result.TopURLs = topN(urlCount, 20)
	result.TopIPs = topNWithGeo(ipCount, 20, withGeo)
	result.TopUserAgents = topN(uaCount, 10)
	result.TopThreats = topN(threatCatCount, 20)
	result.ThreatIPs = topNWithGeo(threatIPCount, 10, withGeo)

	if bannedIPs != nil {
		markBanned(result.TopIPs, bannedIPs)
		markBanned(result.ThreatIPs, bannedIPs)
	}

	result.HourlyStats = timeSeries(hourlyReqs, hourlyBytes)
	result.DailyStats = timeSeries(dailyReqs, dailyBytes)

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
	now := time.Now()
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

func daysFromRange(timeRange string) int {
	switch timeRange {
	case "1h", "6h", "24h":
		return 1
	case "7d":
		return 7
	case "30d":
		return 30
	default:
		return 1
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
