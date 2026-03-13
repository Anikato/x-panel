package service

import (
	"bufio"
	"fmt"
	"os"
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
)

type INginxLogService interface {
	Analyze(req dto.NginxLogAnalysisReq) (*dto.NginxLogAnalysis, error)
}

type NginxLogService struct {
	websiteRepo repo.IWebsiteRepo
}

func NewINginxLogService() INginxLogService {
	return &NginxLogService{websiteRepo: repo.NewIWebsiteRepo()}
}

// combined log format regex:
// $remote_addr - $remote_user [$time_local] "$request" $status $body_bytes_sent "$http_referer" "$http_user_agent"
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

func (s *NginxLogService) Analyze(req dto.NginxLogAnalysisReq) (*dto.NginxLogAnalysis, error) {
	site, err := s.websiteRepo.Get(repo.WithByID(req.SiteID))
	if err != nil {
		return nil, buserr.New(constant.ErrRecordNotFound)
	}

	nc := global.CONF.Nginx
	if !nc.IsInstalled() {
		return nil, buserr.New(constant.ErrNginxNotInstalled)
	}

	logPath := fmt.Sprintf("%s/logs/%s.access.log", nc.InstallDir, site.Alias)
	if _, err := os.Stat(logPath); err != nil {
		return &dto.NginxLogAnalysis{
			StatusCodes: make(map[string]int64),
		}, nil
	}

	days := req.Days
	if days <= 0 {
		days = 1
	}
	cutoff := time.Now().AddDate(0, 0, -days)

	entries, err := parseAccessLog(logPath, cutoff)
	if err != nil {
		return nil, fmt.Errorf("parse log: %v", err)
	}

	return aggregate(entries, days), nil
}

func parseAccessLog(path string, cutoff time.Time) ([]logEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var entries []logEntry
	scanner := bufio.NewScanner(f)
	buf := make([]byte, 0, 256*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
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
	return entries, scanner.Err()
}

func aggregate(entries []logEntry, days int) *dto.NginxLogAnalysis {
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

	result.TopURLs = topN(urlCount, 10)
	result.TopIPs = topN(ipCount, 10)
	result.TopUserAgents = topN(uaCount, 10)

	result.HourlyStats = timeSeries(hourlyReqs, hourlyBytes)
	result.DailyStats = timeSeries(dailyReqs, dailyBytes)

	return result
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
