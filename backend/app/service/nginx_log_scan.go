package service

import (
	"bytes"
	"context"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"xpanel/app/dto"
)

const logAnalysisStrategyVersion = "active-v1"

var (
	logScanMaxBytes     int64 = 64 << 20
	logScanMaxLines     int64 = 200000
	logScanMaxLineBytes       = 64 * 1024
	logScanMaxKeys            = 50000
	logScanTimeout            = 10 * time.Second
	logScanDelay        time.Duration
	logScanTestHook     func()
	logScanChunkSize    = 64 * 1024
	logScanTestReadHook func()
)

type scanLimits struct {
	MaxBytes     int64
	MaxLines     int64
	MaxLineBytes int
	MaxKeys      int
	Timeout      time.Duration
}

func defaultScanLimits() scanLimits {
	return scanLimits{
		MaxBytes:     logScanMaxBytes,
		MaxLines:     logScanMaxLines,
		MaxLineBytes: logScanMaxLineBytes,
		MaxKeys:      logScanMaxKeys,
		Timeout:      logScanTimeout,
	}
}

type scanBudget struct {
	limits   scanLimits
	bytes    int64
	lines    int64
	reason   string
	deadline time.Time
}

func newScanBudget(limits scanLimits, now time.Time) *scanBudget {
	b := &scanBudget{limits: limits}
	if limits.Timeout > 0 {
		b.deadline = now.Add(limits.Timeout)
	}
	return b
}

func (b *scanBudget) exhausted() bool {
	if b.reason != "" {
		return true
	}
	if b.limits.MaxLines > 0 && b.lines >= b.limits.MaxLines {
		b.reason = "line_limit"
		return true
	}
	if b.limits.MaxBytes > 0 && b.bytes >= b.limits.MaxBytes {
		b.reason = "byte_limit"
		return true
	}
	if !b.deadline.IsZero() && time.Now().After(b.deadline) {
		b.reason = "time_limit"
		return true
	}
	return false
}

type logSink interface {
	add(logEntry) bool
}

type scanMetaAcc struct {
	scannedBytes   int64
	scannedLines   int64
	matchedLines   int64
	invalidLines   int64
	filesTotal     int
	filesSucceeded int
	filesFailed    int
	failed         []dto.NginxLogFileIssue
	skipped        []dto.NginxLogFileIssue
	reasons        []string
	observedFrom   *time.Time
	observedTo     *time.Time
	partial        bool
}

func (m *scanMetaAcc) hasReason(reason string) bool {
	for _, existing := range m.reasons {
		if existing == reason {
			return true
		}
	}
	return false
}

func (m *scanMetaAcc) addReason(reason string) {
	if reason == "" || m.hasReason(reason) {
		return
	}
	m.reasons = append(m.reasons, reason)
	if reason != "active_log_only" {
		m.partial = true
	}
}

func (m *scanMetaAcc) noteTime(ts time.Time) {
	if m.observedFrom == nil || ts.Before(*m.observedFrom) {
		t := ts
		m.observedFrom = &t
	}
	if m.observedTo == nil || ts.After(*m.observedTo) {
		t := ts
		m.observedTo = &t
	}
}

func (m *scanMetaAcc) toMeta(from, to, generated time.Time) dto.NginxLogAnalysisMeta {
	return dto.NginxLogAnalysisMeta{
		RequestedFrom:  from,
		RequestedTo:    to,
		ObservedFrom:   m.observedFrom,
		ObservedTo:     m.observedTo,
		GeneratedAt:    generated,
		ScannedBytes:   m.scannedBytes,
		ScannedLines:   m.scannedLines,
		MatchedLines:   m.matchedLines,
		InvalidLines:   m.invalidLines,
		Partial:        m.partial,
		Reasons:        append([]string{}, m.reasons...),
		FilesTotal:     m.filesTotal,
		FilesSucceeded: m.filesSucceeded,
		FilesFailed:    m.filesFailed,
		FailedFiles:    m.failed,
		SkippedFiles:   m.skipped,
		SourceScope:    "active",
	}
}

func scanLogFiles(ctx context.Context, paths []string, cutoff, until time.Time, limits scanLimits, sink logSink) scanMetaAcc {
	meta := scanMetaAcc{filesTotal: len(paths)}
	budget := newScanBudget(limits, until)
	if logScanTestHook != nil {
		logScanTestHook()
	}
	if logScanDelay > 0 {
		select {
		case <-ctx.Done():
			meta.addReason("canceled")
			return meta
		case <-time.After(logScanDelay):
		}
	}
	for _, path := range paths {
		if ctx.Err() != nil {
			meta.addReason("canceled")
			break
		}
		if budget.exhausted() {
			meta.addReason(budget.reason)
			meta.skipped = append(meta.skipped, dto.NginxLogFileIssue{Path: path, Reason: budget.reason})
			continue
		}
		info, err := os.Stat(path)
		if err != nil {
			meta.filesFailed++
			meta.failed = append(meta.failed, dto.NginxLogFileIssue{Path: path, Reason: "read_error"})
			meta.addReason("read_error")
			continue
		}
		snapshot := info.Size()
		if err := scanLogFile(ctx, path, snapshot, cutoff, budget, sink, &meta); err != nil {
			if ctx.Err() != nil {
				meta.addReason("canceled")
				break
			}
			meta.filesFailed++
			meta.failed = append(meta.failed, dto.NginxLogFileIssue{Path: path, Reason: "read_error"})
			meta.addReason("read_error")
			continue
		}
		meta.filesSucceeded++
		if budget.exhausted() {
			meta.addReason(budget.reason)
		}
	}
	return meta
}

func scanLogFile(ctx context.Context, path string, snapshot int64, cutoff time.Time, budget *scanBudget, sink logSink, meta *scanMetaAcc) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	if live, err := f.Stat(); err == nil && live.Size() < snapshot {
		snapshot = live.Size()
		meta.addReason("truncated_file")
	}
	end := snapshot
	if end == 0 {
		return nil
	}

	var last [1]byte
	if _, err := f.ReadAt(last[:], end-1); err != nil && err != io.EOF {
		return err
	}
	if last[0] != '\n' {
		nl, found, err := findLastNewline(ctx, f, end, budget, meta)
		if err != nil {
			return err
		}
		if !found {
			if budget.reason != "" {
				meta.addReason(budget.reason)
			} else if !meta.hasReason("truncated_file") && budget.limits.MaxLineBytes > 0 && end > int64(budget.limits.MaxLineBytes) {
				meta.invalidLines++
				meta.addReason("oversized_line")
			}
			return nil
		}
		trailing := end - (nl + 1)
		if budget.limits.MaxLineBytes > 0 && trailing > int64(budget.limits.MaxLineBytes) {
			meta.invalidLines++
			meta.addReason("oversized_line")
		}
		end = nl + 1
	}

	const chunkSize = 64 * 1024
	leftover := []byte{}
	pos := end
	for pos > 0 {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if budget.exhausted() {
			return nil
		}
		readSize := int64(chunkSize)
		remainBudget := budget.limits.MaxBytes - budget.bytes
		if budget.limits.MaxBytes > 0 && remainBudget <= 0 {
			budget.reason = "byte_limit"
			return nil
		}
		if budget.limits.MaxBytes > 0 && readSize > remainBudget {
			readSize = remainBudget
		}
		if pos < readSize {
			readSize = pos
		}
		buf := make([]byte, readSize)
		n, err := f.ReadAt(buf, pos-readSize)
		if err != nil && err != io.EOF {
			return err
		}
		buf = buf[:n]
		budget.bytes += int64(n)
		meta.scannedBytes += int64(n)
		pos -= int64(n)

		data := make([]byte, 0, len(buf)+len(leftover))
		data = append(data, buf...)
		data = append(data, leftover...)

		parts := bytes.Split(data, []byte{'\n'})
		var complete [][]byte
		if pos > 0 {
			leftover = parts[0]
			if budget.limits.MaxLineBytes > 0 && len(leftover) > budget.limits.MaxLineBytes {
				meta.invalidLines++
				meta.addReason("oversized_line")
				leftover = nil
			}
			complete = parts[1:]
		} else {
			leftover = nil
			complete = parts
		}
		if len(complete) > 0 && len(complete[len(complete)-1]) == 0 {
			complete = complete[:len(complete)-1]
		}
		for i := len(complete) - 1; i >= 0; i-- {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if budget.exhausted() {
				return nil
			}
			line := complete[i]
			if budget.limits.MaxLineBytes > 0 && len(line) > budget.limits.MaxLineBytes {
				meta.invalidLines++
				meta.addReason("oversized_line")
				continue
			}
			budget.lines++
			meta.scannedLines++
			entry, ok := parseCombinedLine(line)
			if !ok {
				meta.invalidLines++
				continue
			}
			if entry.Time.Before(cutoff) {
				continue
			}
			meta.matchedLines++
			meta.noteTime(entry.Time)
			if !sink.add(entry) {
				meta.addReason("key_limit")
				budget.reason = "key_limit"
				return nil
			}
		}
	}
	return nil
}

func findLastNewline(ctx context.Context, f *os.File, end int64, budget *scanBudget, meta *scanMetaAcc) (int64, bool, error) {
	chunkSize := int64(logScanChunkSize)
	if chunkSize <= 0 {
		chunkSize = 64 * 1024
	}
	pos := end
	for pos > 0 {
		if ctx.Err() != nil {
			return 0, false, ctx.Err()
		}
		if budget.exhausted() {
			return 0, false, nil
		}
		readSize := chunkSize
		if budget.limits.MaxBytes > 0 {
			remain := budget.limits.MaxBytes - budget.bytes
			if remain <= 0 {
				budget.reason = "byte_limit"
				return 0, false, nil
			}
			if readSize > remain {
				readSize = remain
			}
		}
		if pos < readSize {
			readSize = pos
		}
		if logScanTestReadHook != nil {
			logScanTestReadHook()
		}
		buf := make([]byte, readSize)
		n, err := f.ReadAt(buf, pos-readSize)
		if err != nil && err != io.EOF {
			return 0, false, err
		}
		if n == 0 {
			meta.addReason("truncated_file")
			return 0, false, nil
		}
		budget.bytes += int64(n)
		meta.scannedBytes += int64(n)
		buf = buf[:n]
		if idx := bytes.LastIndexByte(buf, '\n'); idx >= 0 {
			return pos - readSize + int64(idx), true, nil
		}
		pos -= int64(n)
	}
	return 0, false, nil
}

func parseCombinedLine(line []byte) (logEntry, bool) {
	m := combinedLogRe.FindSubmatch(line)
	if m == nil {
		return logEntry{}, false
	}
	t, err := time.Parse("02/Jan/2006:15:04:05 -0700", string(m[2]))
	if err != nil {
		return logEntry{}, false
	}
	status, _ := strconv.Atoi(string(m[4]))
	nBytes, _ := strconv.ParseInt(string(m[5]), 10, 64)
	parts := strings.SplitN(string(m[3]), " ", 3)
	method, url := "", ""
	if len(parts) >= 2 {
		method = parts[0]
		url = parts[1]
	}
	return logEntry{
		IP:        string(m[1]),
		Time:      t,
		Method:    method,
		URL:       url,
		Status:    status,
		Bytes:     nBytes,
		UserAgent: string(m[6]),
	}, true
}

type statsSink struct {
	maxKeys         int
	capped          bool
	totalBytes      int64
	errors          int64
	ipSet           map[string]struct{}
	urlCount        map[string]int64
	ipCount         map[string]int64
	uaCount         map[string]int64
	hourlyReqs      map[string]int64
	hourlyBytes     map[string]int64
	dailyReqs       map[string]int64
	dailyBytes      map[string]int64
	statusCodes     map[string]int64
	threatCatCount  map[string]int64
	threatIPCount   map[string]int64
	crawlerCatCount map[string]int64
	threatRequests  int64
	crawlerRequests int64
}

func newStatsSink(maxKeys int) *statsSink {
	return &statsSink{
		maxKeys:         maxKeys,
		ipSet:           map[string]struct{}{},
		urlCount:        map[string]int64{},
		ipCount:         map[string]int64{},
		uaCount:         map[string]int64{},
		hourlyReqs:      map[string]int64{},
		hourlyBytes:     map[string]int64{},
		dailyReqs:       map[string]int64{},
		dailyBytes:      map[string]int64{},
		statusCodes:     map[string]int64{},
		threatCatCount:  map[string]int64{},
		threatIPCount:   map[string]int64{},
		crawlerCatCount: map[string]int64{},
	}
}

func (s *statsSink) wouldExceed(m map[string]int64, key string) bool {
	if s.maxKeys <= 0 {
		return false
	}
	if _, ok := m[key]; ok {
		return false
	}
	return len(m) >= s.maxKeys
}

func (s *statsSink) add(e logEntry) bool {
	if s.wouldExceed(s.urlCount, e.URL) || s.wouldExceed(s.ipCount, e.IP) || s.wouldExceed(s.uaCount, e.UserAgent) {
		s.capped = true
		return false
	}
	s.ipSet[e.IP] = struct{}{}
	s.totalBytes += e.Bytes
	cat := statusClass(e.Status)
	s.statusCodes[cat]++
	if e.Status >= 400 {
		s.errors++
	}
	s.urlCount[e.URL]++
	s.ipCount[e.IP]++
	s.uaCount[e.UserAgent]++
	if tc := classifyThreat(e.URL); tc != "" {
		s.threatRequests++
		s.threatCatCount[tc]++
		s.threatIPCount[e.IP]++
	}
	if cc := classifyCrawler(e.UserAgent); cc != "" {
		s.crawlerRequests++
		s.crawlerCatCount[cc]++
	}
	h := e.Time.Format("2006-01-02 15:00")
	s.hourlyReqs[h]++
	s.hourlyBytes[h] += e.Bytes
	d := e.Time.Format("2006-01-02")
	s.dailyReqs[d]++
	s.dailyBytes[d] += e.Bytes
	return true
}

func statusClass(status int) string {
	return strconv.Itoa(status/100) + "xx"
}

func (s *statsSink) result(withGeo bool, banned map[string]bool) *dto.NginxLogAnalysis {
	out := &dto.NginxLogAnalysis{
		TotalRequests:   0,
		StatusCodes:     s.statusCodes,
		TotalBytes:      s.totalBytes,
		ThreatRequests:  s.threatRequests,
		CrawlerRequests: s.crawlerRequests,
	}
	for _, n := range s.statusCodes {
		out.TotalRequests += n
	}
	out.UniqueIPs = len(s.ipSet)
	if out.TotalRequests > 0 {
		out.ErrorRate = float64(s.errors) / float64(out.TotalRequests) * 100
	}
	out.TopURLs = topN(s.urlCount, 20)
	out.TopIPs = topNWithGeo(s.ipCount, 20, withGeo)
	out.TopUserAgents = topN(s.uaCount, 10)
	out.TopThreats = topN(s.threatCatCount, 20)
	out.ThreatIPs = topNWithGeo(s.threatIPCount, 10, withGeo)
	out.TopCrawlers = topN(s.crawlerCatCount, 20)
	if banned != nil {
		markBanned(out.TopIPs, banned)
		markBanned(out.ThreatIPs, banned)
	}
	out.HourlyStats = timeSeries(s.hourlyReqs, s.hourlyBytes)
	out.DailyStats = timeSeries(s.dailyReqs, s.dailyBytes)
	return out
}

type drillSink struct {
	filterType  string
	filterValue string
	maxKeys     int
	capped      bool
	ipCount     map[string]int64
	urlCount    map[string]int64
}

func newDrillSink(filterType, filterValue string, maxKeys int) *drillSink {
	return &drillSink{
		filterType:  filterType,
		filterValue: filterValue,
		maxKeys:     maxKeys,
		ipCount:     map[string]int64{},
		urlCount:    map[string]int64{},
	}
}

func (s *drillSink) add(e logEntry) bool {
	switch s.filterType {
	case "url":
		if e.URL != s.filterValue {
			return true
		}
		if s.maxKeys > 0 {
			if _, ok := s.ipCount[e.IP]; !ok && len(s.ipCount) >= s.maxKeys {
				s.capped = true
				return false
			}
		}
		s.ipCount[e.IP]++
	case "ip":
		if e.IP != s.filterValue {
			return true
		}
		if s.maxKeys > 0 {
			if _, ok := s.urlCount[e.URL]; !ok && len(s.urlCount) >= s.maxKeys {
				s.capped = true
				return false
			}
		}
		s.urlCount[e.URL]++
	case "threat":
		if classifyThreat(e.URL) != s.filterValue {
			return true
		}
		if s.maxKeys > 0 {
			_, ipKnown := s.ipCount[e.IP]
			_, urlKnown := s.urlCount[e.URL]
			if (!ipKnown && len(s.ipCount) >= s.maxKeys) || (!urlKnown && len(s.urlCount) >= s.maxKeys) {
				s.capped = true
				return false
			}
		}
		s.ipCount[e.IP]++
		s.urlCount[e.URL]++
	}
	return true
}

type collectSink struct {
	entries []logEntry
}

func (s *collectSink) add(e logEntry) bool {
	s.entries = append(s.entries, e)
	return true
}
