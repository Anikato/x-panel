package service

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"xpanel/app/dto"
	"xpanel/app/model"
	"xpanel/app/repo"
	"xpanel/global"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func combinedLine(ip string, ts time.Time, method, url string, status int, nBytes int64, ua string) string {
	return fmt.Sprintf(`%s - - [%s] "%s %s HTTP/1.1" %d %d "-" "%s"`,
		ip, ts.Format("02/Jan/2006:15:04:05 -0700"), method, url, status, nBytes, ua)
}

func writeLines(t *testing.T, path string, lines []string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func setupLogAnalysisLimits(t *testing.T) {
	t.Helper()
	resetLogAnalysisStateForTest()
	t.Cleanup(resetLogAnalysisStateForTest)
}

func setupNginxWithAccessLog(t *testing.T, siteName, logPath, body string) *NginxLogService {
	t.Helper()
	setupLogAnalysisLimits(t)
	root := t.TempDir()
	if body != "" {
		if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil && !filepath.IsAbs(logPath) {
			t.Fatal(err)
		}
	}
	if !filepath.IsAbs(logPath) {
		logPath = filepath.Join(root, logPath)
	}
	if body != "" {
		writeLines(t, logPath, strings.Split(strings.TrimSuffix(body, "\n"), "\n"))
	}
	conf := fmt.Sprintf("server {\n    server_name %s;\n    access_log %s combined;\n}\n", siteName, logPath)
	external := writeNginxFixture(t, filepath.Join(root, "sites", "site.conf"), conf)
	writeNginxFixture(t, filepath.Join(root, "conf", "nginx.conf"), "include "+external+";\n")
	writeNginxFixture(t, filepath.Join(root, "sbin", "nginx"), "")
	previous := global.CONF
	global.CONF.Nginx = global.NginxConfig{InstallDir: root, Mode: "prefix"}
	global.CONF.Nginx.DetectNginx()
	t.Cleanup(func() { global.CONF = previous })
	return &NginxLogService{}
}

func setupLegacyAnalyze(t *testing.T, logPath, body string) (*NginxLogService, uint) {
	t.Helper()
	setupLogAnalysisLimits(t)
	installFakeNginx(t, false)
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "nginx-log.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.Website{}); err != nil {
		t.Fatal(err)
	}
	previous := global.DB
	global.DB = db
	t.Cleanup(func() { global.DB = previous })
	if body != "" {
		writeLines(t, logPath, strings.Split(strings.TrimSuffix(body, "\n"), "\n"))
	}
	site := model.Website{PrimaryDomain: "legacy.example.com", Alias: "legacy", AccessLogPath: logPath}
	if err := repo.NewIWebsiteRepo().Create(&site); err != nil {
		t.Fatal(err)
	}
	return &NginxLogService{websiteRepo: repo.NewIWebsiteRepo()}, site.ID
}

func TestAnalyzeMissingLogReturnsErrorNotZeros(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.access.log")
	svc, id := setupLegacyAnalyze(t, missing, "")
	res, err := svc.Analyze(context.Background(), dto.NginxLogAnalysisReq{SiteID: id, Days: 1})
	if err == nil {
		t.Fatalf("missing log returned %#v, want error", res)
	}
}

func TestAnalyzeSiteUnsupportedFormatReturnsError(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "access.log")
	writeLines(t, logPath, []string{"this is not combined format", "still not a log line", "nope"})
	svc := setupNginxWithAccessLog(t, "fmt.example.com", logPath, "")
	// setupNginxWithAccessLog overwrote empty body; rewrite
	writeLines(t, logPath, []string{"this is not combined format", "still not a log line", "nope"})
	_, err := svc.AnalyzeSite(context.Background(), dto.NginxLogAnalyzeReq{Site: "fmt.example.com", TimeRange: "24h"})
	if err == nil {
		t.Fatal("unsupported format should not return zero stats")
	}
}

func TestAnalyzeCountsAndTimeBounds(t *testing.T) {
	now := time.Now()
	inside := now.Add(-2 * time.Hour)
	outside := now.Add(-48 * time.Hour)
	logPath := filepath.Join(t.TempDir(), "access.log")
	lines := []string{
		combinedLine("203.0.113.1", outside, "GET", "/old", 200, 10, "ua"),
		combinedLine("203.0.113.1", inside, "GET", "/a", 200, 11, "ua"),
		combinedLine("203.0.113.2", inside, "GET", "/b", 404, 12, "ua"),
	}
	svc := setupNginxWithAccessLog(t, "bound.example.com", logPath, strings.Join(lines, "\n"))
	res, err := svc.AnalyzeSite(context.Background(), dto.NginxLogAnalyzeReq{Site: "bound.example.com", TimeRange: "24h"})
	if err != nil {
		t.Fatal(err)
	}
	if res.TotalRequests != 2 {
		t.Fatalf("total=%d, want 2 (outside window excluded)", res.TotalRequests)
	}
	if res.UniqueIPs != 2 {
		t.Fatalf("uv=%d", res.UniqueIPs)
	}
	if res.Meta.MatchedLines != 2 || res.Meta.Partial {
		t.Fatalf("meta=%#v", res.Meta)
	}
	if res.Meta.ObservedFrom == nil || res.Meta.ObservedTo == nil {
		t.Fatal("observed range missing")
	}
	if res.Meta.SourceScope != "active" {
		t.Fatalf("sourceScope=%q", res.Meta.SourceScope)
	}
	if !res.Meta.RequestedFrom.Before(res.Meta.RequestedTo) {
		t.Fatalf("requested window %v .. %v", res.Meta.RequestedFrom, res.Meta.RequestedTo)
	}
}

func TestLineBudgetMarksPartialAndLegacyAnalyzeIsBounded(t *testing.T) {
	now := time.Now()
	var lines []string
	for i := 0; i < 80; i++ {
		lines = append(lines, combinedLine("203.0.113.10", now.Add(-time.Duration(i)*time.Second), "GET", fmt.Sprintf("/p/%d", i), 200, 8, "ua"))
	}
	logPath := filepath.Join(t.TempDir(), "access.log")
	svc, id := setupLegacyAnalyze(t, logPath, strings.Join(lines, "\n"))
	logScanMaxLines = 20
	res, err := svc.Analyze(context.Background(), dto.NginxLogAnalysisReq{SiteID: id, Days: 1})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Meta.Partial || !containsReason(res.Meta.Reasons, "line_limit") {
		t.Fatalf("legacy analyze meta=%#v", res.Meta)
	}
	if res.Meta.ScannedLines > 20 || res.TotalRequests > 20 {
		t.Fatalf("legacy unlimited read still happened: scanned=%d total=%d", res.Meta.ScannedLines, res.TotalRequests)
	}
}

func TestByteBudgetIsSharedAcrossFiles(t *testing.T) {
	logScanMaxBytes = 800
	now := time.Now()
	root := t.TempDir()
	aPath := filepath.Join(root, "a.log")
	bPath := filepath.Join(root, "b.log")
	var lines []string
	for i := 0; i < 40; i++ {
		lines = append(lines, combinedLine("203.0.113.10", now, "GET", fmt.Sprintf("/long/path/item/%d", i), 200, 8, "Mozilla/5.0 test-agent"))
	}
	body := strings.Join(lines, "\n")
	writeLines(t, aPath, lines)
	writeLines(t, bPath, lines)
	setupLogAnalysisLimits(t)
	logScanMaxBytes = 800
	conf := fmt.Sprintf("server {\n    server_name a.example.com;\n    access_log %s combined;\n}\nserver {\n    server_name b.example.com;\n    access_log %s combined;\n}\n", aPath, bPath)
	external := writeNginxFixture(t, filepath.Join(root, "sites", "site.conf"), conf)
	writeNginxFixture(t, filepath.Join(root, "conf", "nginx.conf"), "include "+external+";\n")
	writeNginxFixture(t, filepath.Join(root, "sbin", "nginx"), "")
	previous := global.CONF
	global.CONF.Nginx = global.NginxConfig{InstallDir: root, Mode: "prefix"}
	global.CONF.Nginx.DetectNginx()
	t.Cleanup(func() { global.CONF = previous })

	res, err := (&NginxLogService{}).AnalyzeSite(context.Background(), dto.NginxLogAnalyzeReq{TimeRange: "24h"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Meta.ScannedBytes > 800 {
		t.Fatalf("scannedBytes=%d exceeds shared budget", res.Meta.ScannedBytes)
	}
	if !res.Meta.Partial {
		t.Fatalf("expected partial when two files share 800 byte budget, meta=%#v", res.Meta)
	}
	if len(res.Meta.SkippedFiles) == 0 && !containsReason(res.Meta.Reasons, "byte_limit") {
		t.Fatalf("shared budget should skip or mark byte_limit: %#v", res.Meta)
	}
	_ = body
}

func TestTimeBudgetReturnsPartialNotError(t *testing.T) {
	now := time.Now()
	var lines []string
	for i := 0; i < 30; i++ {
		lines = append(lines, combinedLine("203.0.113.3", now, "GET", fmt.Sprintf("/t/%d", i), 200, 2, "ua"))
	}
	logPath := filepath.Join(t.TempDir(), "access.log")
	svc := setupNginxWithAccessLog(t, "time.example.com", logPath, strings.Join(lines, "\n"))
	logScanTimeout = time.Nanosecond
	logScanDelay = 20 * time.Millisecond
	res, err := svc.AnalyzeSite(context.Background(), dto.NginxLogAnalyzeReq{Site: "time.example.com", TimeRange: "24h"})
	if err != nil {
		t.Fatalf("time budget should return partial result, err=%v", err)
	}
	if !res.Meta.Partial || !containsReason(res.Meta.Reasons, "time_limit") {
		t.Fatalf("meta=%#v", res.Meta)
	}
}

func TestKeyLimitStopsScanAndMarksPartial(t *testing.T) {
	now := time.Now()
	var lines []string
	for i := 0; i < 30; i++ {
		lines = append(lines, combinedLine(fmt.Sprintf("203.0.113.%d", i+1), now, "GET", fmt.Sprintf("/u/%d", i), 200, 8, "ua"))
	}
	logPath := filepath.Join(t.TempDir(), "access.log")
	svc := setupNginxWithAccessLog(t, "keys.example.com", logPath, strings.Join(lines, "\n"))
	logScanMaxKeys = 8
	res, err := svc.AnalyzeSite(context.Background(), dto.NginxLogAnalyzeReq{Site: "keys.example.com", TimeRange: "24h"})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Meta.Partial || !containsReason(res.Meta.Reasons, "key_limit") {
		t.Fatalf("meta=%#v", res.Meta)
	}
	if res.UniqueIPs > 8 {
		t.Fatalf("uv=%d exceeded key cap", res.UniqueIPs)
	}
}

func TestOversizedIncompleteTrailingLineDoesNotSkipFile(t *testing.T) {
	now := time.Now()
	logPath := filepath.Join(t.TempDir(), "access.log")
	good1 := combinedLine("203.0.113.21", now, "GET", "/keep-a", 200, 4, "ua")
	good2 := combinedLine("203.0.113.22", now, "GET", "/keep-b", 200, 5, "ua")
	if len(good1) > 160 || len(good2) > 160 {
		t.Fatalf("fixture lines longer than test limit: %d %d", len(good1), len(good2))
	}
	trailer := strings.Repeat("x", 400)
	content := good1 + "\n" + good2 + "\n" + trailer
	if err := os.WriteFile(logPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	svc := setupNginxWithAccessLog(t, "trail.example.com", logPath, "")
	if err := os.WriteFile(logPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	logScanMaxLineBytes = 160
	res, err := svc.AnalyzeSite(context.Background(), dto.NginxLogAnalyzeReq{Site: "trail.example.com", TimeRange: "24h"})
	if err != nil {
		t.Fatalf("trailing oversized incomplete line skipped the file: %v", err)
	}
	if res.TotalRequests != 2 {
		t.Fatalf("total=%d, want 2 complete lines kept", res.TotalRequests)
	}
	if res.Meta.InvalidLines < 1 || !containsReason(res.Meta.Reasons, "oversized_line") {
		t.Fatalf("expected oversized trailing line to be marked: %#v", res.Meta)
	}
}

func TestFindLastNewlineStopsWhenTrailingExceedsByteBudget(t *testing.T) {
	now := time.Now()
	logPath := filepath.Join(t.TempDir(), "access.log")
	good1 := combinedLine("203.0.113.21", now, "GET", "/keep-a", 200, 4, "ua")
	good2 := combinedLine("203.0.113.22", now, "GET", "/keep-b", 200, 5, "ua")
	trailer := strings.Repeat("x", 300)
	content := good1 + "\n" + good2 + "\n" + trailer
	svc := setupNginxWithAccessLog(t, "budget-trail.example.com", logPath, "")
	if err := os.WriteFile(logPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	logScanChunkSize = 32
	logScanMaxBytes = 64
	res, err := svc.AnalyzeSite(context.Background(), dto.NginxLogAnalyzeReq{Site: "budget-trail.example.com", TimeRange: "24h"})
	if err != nil {
		t.Fatalf("over-budget trailing search should return partial, err=%v", err)
	}
	if !res.Meta.Partial || !containsReason(res.Meta.Reasons, "byte_limit") {
		t.Fatalf("expected byte_limit partial, meta=%#v", res.Meta)
	}
	if res.Meta.ScannedBytes > 64 {
		t.Fatalf("searched past byte budget: scannedBytes=%d", res.Meta.ScannedBytes)
	}
}

func TestFindLastNewlineStopsWhenFileTruncated(t *testing.T) {
	now := time.Now()
	logPath := filepath.Join(t.TempDir(), "access.log")
	good := combinedLine("203.0.113.23", now, "GET", "/keep", 200, 4, "ua")
	content := good + "\n" + strings.Repeat("x", 200)
	svc := setupNginxWithAccessLog(t, "trunc-trail.example.com", logPath, "")
	if err := os.WriteFile(logPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	logScanChunkSize = 32
	reads := 0
	logScanTestReadHook = func() {
		reads++
		if reads >= 2 {
			if err := os.Truncate(logPath, 0); err != nil {
				t.Errorf("truncate: %v", err)
			}
		}
	}
	t.Cleanup(func() { logScanTestReadHook = nil })
	res, err := svc.AnalyzeSite(context.Background(), dto.NginxLogAnalyzeReq{Site: "trunc-trail.example.com", TimeRange: "24h"})
	if err != nil {
		t.Fatalf("truncated search should return partial, err=%v", err)
	}
	if !res.Meta.Partial || !containsReason(res.Meta.Reasons, "truncated_file") {
		t.Fatalf("expected truncated_file, meta=%#v", res.Meta)
	}
}

func TestGlobalQueueRejectsWhenFull(t *testing.T) {
	now := time.Now()
	root := t.TempDir()
	setupLogAnalysisLimits(t)
	logScanMaxQueue = 2
	logScanDelay = 120 * time.Millisecond
	var conf bytes.Buffer
	for i := 0; i < 3; i++ {
		p := filepath.Join(root, fmt.Sprintf("q%d.log", i))
		name := fmt.Sprintf("q%d.example.com", i)
		writeLines(t, p, []string{combinedLine("203.0.113.30", now, "GET", fmt.Sprintf("/q/%d", i), 200, 1, "ua")})
		fmt.Fprintf(&conf, "server {\n    server_name %s;\n    access_log %s combined;\n}\n", name, p)
	}
	external := writeNginxFixture(t, filepath.Join(root, "sites", "site.conf"), conf.String())
	writeNginxFixture(t, filepath.Join(root, "conf", "nginx.conf"), "include "+external+";\n")
	writeNginxFixture(t, filepath.Join(root, "sbin", "nginx"), "")
	previous := global.CONF
	global.CONF.Nginx = global.NginxConfig{InstallDir: root, Mode: "prefix"}
	global.CONF.Nginx.DetectNginx()
	t.Cleanup(func() { global.CONF = previous })

	svc := &NginxLogService{}
	var wg sync.WaitGroup
	errs := make(chan error, 3)
	for i := 0; i < 3; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := svc.AnalyzeSite(context.Background(), dto.NginxLogAnalyzeReq{
				Site: fmt.Sprintf("q%d.example.com", i), TimeRange: "24h",
			})
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	busy := 0
	ok := 0
	for err := range errs {
		if err == nil {
			ok++
			continue
		}
		if strings.Contains(err.Error(), "ErrNginxLogBusy") || strings.Contains(err.Error(), "正在进行") {
			busy++
			continue
		}
		t.Fatalf("unexpected error: %v", err)
	}
	if busy < 1 || ok < 1 {
		t.Fatalf("global queue cap not enforced: ok=%d busy=%d", ok, busy)
	}
}

func TestOversizedAndIncompleteLinesAreNotCounted(t *testing.T) {
	now := time.Now()
	logPath := filepath.Join(t.TempDir(), "access.log")
	good := combinedLine("203.0.113.9", now, "GET", "/ok", 200, 4, "ua")
	oversized := combinedLine("203.0.113.9", now, "GET", "/"+strings.Repeat("x", 200), 200, 4, "ua")
	incomplete := combinedLine("203.0.113.8", now, "GET", "/partial", 200, 4, "ua")
	incomplete = incomplete[:len(incomplete)/2]
	content := good + "\n" + oversized + "\n" + incomplete
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(logPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	svc := setupNginxWithAccessLog(t, "line.example.com", logPath, "")
	if err := os.WriteFile(logPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	logScanMaxLineBytes = 160
	res, err := svc.AnalyzeSite(context.Background(), dto.NginxLogAnalyzeReq{Site: "line.example.com", TimeRange: "24h"})
	if err != nil {
		t.Fatal(err)
	}
	if res.TotalRequests != 1 {
		t.Fatalf("total=%d, want only complete short line", res.TotalRequests)
	}
	if res.Meta.InvalidLines < 1 {
		t.Fatalf("oversized line not tracked: %#v", res.Meta)
	}
}

func TestCrossChunkLineIsParsed(t *testing.T) {
	now := time.Now()
	line := combinedLine("198.51.100.7", now, "GET", "/chunked", 200, 9, "ua")
	padding := strings.Repeat("203.0.113.1 - - ["+now.Format("02/Jan/2006:15:04:05 -0700")+`] "GET /pad HTTP/1.1" 200 1 "-" "ua"`+"\n", 40)
	logPath := filepath.Join(t.TempDir(), "access.log")
	body := padding + line + "\n"
	svc := setupNginxWithAccessLog(t, "chunk.example.com", logPath, strings.TrimSuffix(body, "\n"))
	res, err := svc.AnalyzeSite(context.Background(), dto.NginxLogAnalyzeReq{Site: "chunk.example.com", TimeRange: "24h"})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, item := range res.TopURLs {
		if item.Name == "/chunked" {
			found = true
		}
	}
	if !found {
		t.Fatalf("cross-chunk /chunked missing from %#v", res.TopURLs)
	}
}

func TestPartialFileFailureStillReturnsMarkedResult(t *testing.T) {
	now := time.Now()
	root := t.TempDir()
	okPath := filepath.Join(root, "ok.log")
	writeLines(t, okPath, []string{combinedLine("203.0.113.4", now, "GET", "/ok", 200, 3, "ua")})
	missing := filepath.Join(root, "missing.log")
	setupLogAnalysisLimits(t)
	conf := fmt.Sprintf("server {\n    server_name ok.example.com;\n    access_log %s combined;\n}\nserver {\n    server_name bad.example.com;\n    access_log %s combined;\n}\n", okPath, missing)
	external := writeNginxFixture(t, filepath.Join(root, "sites", "site.conf"), conf)
	writeNginxFixture(t, filepath.Join(root, "conf", "nginx.conf"), "include "+external+";\n")
	writeNginxFixture(t, filepath.Join(root, "sbin", "nginx"), "")
	previous := global.CONF
	global.CONF.Nginx = global.NginxConfig{InstallDir: root, Mode: "prefix"}
	global.CONF.Nginx.DetectNginx()
	t.Cleanup(func() { global.CONF = previous })

	res, err := (&NginxLogService{}).AnalyzeSite(context.Background(), dto.NginxLogAnalyzeReq{TimeRange: "24h"})
	if err != nil {
		t.Fatal(err)
	}
	if res.TotalRequests != 1 || !res.Meta.Partial || res.Meta.FilesFailed < 1 {
		t.Fatalf("partial failure meta=%#v total=%d", res.Meta, res.TotalRequests)
	}
}

func TestSharedLogIsNotDoubleCounted(t *testing.T) {
	now := time.Now()
	root := t.TempDir()
	shared := filepath.Join(root, "shared.log")
	writeLines(t, shared, []string{combinedLine("203.0.113.5", now, "GET", "/shared", 200, 5, "ua")})
	setupLogAnalysisLimits(t)
	conf := fmt.Sprintf("server {\n    server_name one.example.com;\n    access_log %s combined;\n}\nserver {\n    server_name two.example.com;\n    access_log %s combined;\n}\n", shared, shared)
	external := writeNginxFixture(t, filepath.Join(root, "sites", "site.conf"), conf)
	writeNginxFixture(t, filepath.Join(root, "conf", "nginx.conf"), "include "+external+";\n")
	writeNginxFixture(t, filepath.Join(root, "sbin", "nginx"), "")
	previous := global.CONF
	global.CONF.Nginx = global.NginxConfig{InstallDir: root, Mode: "prefix"}
	global.CONF.Nginx.DetectNginx()
	t.Cleanup(func() { global.CONF = previous })

	svc := &NginxLogService{}
	all, err := svc.AnalyzeSite(context.Background(), dto.NginxLogAnalyzeReq{TimeRange: "24h"})
	if err != nil {
		t.Fatal(err)
	}
	if all.TotalRequests != 1 {
		t.Fatalf("shared log double counted: %d", all.TotalRequests)
	}
	one, err := svc.AnalyzeSite(context.Background(), dto.NginxLogAnalyzeReq{Site: "one.example.com", TimeRange: "24h"})
	if err != nil {
		t.Fatal(err)
	}
	if !one.Meta.SharedLog || !one.Meta.Partial || !containsReason(one.Meta.Reasons, "shared_log_unattributed") {
		t.Fatalf("single-site shared log should be marked: %#v", one.Meta)
	}
}

func TestInFlightCancelStopsScan(t *testing.T) {
	now := time.Now()
	logPath := filepath.Join(t.TempDir(), "access.log")
	svc := setupNginxWithAccessLog(t, "live-cancel.example.com", logPath, combinedLine("203.0.113.6", now, "GET", "/c", 200, 2, "ua"))
	logScanDelay = 300 * time.Millisecond
	started := make(chan struct{})
	logScanTestHook = func() { close(started) }
	t.Cleanup(func() { logScanTestHook = nil })
	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		_, err := svc.AnalyzeSite(ctx, dto.NginxLogAnalyzeReq{Site: "live-cancel.example.com", TimeRange: "24h"})
		errCh <- err
	}()
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		cancel()
		t.Fatal("scan did not start")
	}
	start := time.Now()
	cancel()
	err := <-errCh
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("in-flight cancel returned success")
	}
	if elapsed > 150*time.Millisecond {
		t.Fatalf("in-flight cancel took %s, want well under scan delay", elapsed)
	}
	logScanDelay = 0
	logScanTestHook = nil
	followStart := time.Now()
	if _, followErr := svc.AnalyzeSite(context.Background(), dto.NginxLogAnalyzeReq{Site: "live-cancel.example.com", TimeRange: "24h"}); followErr != nil {
		t.Fatalf("follow-up scan after cancel: %v", followErr)
	}
	if time.Since(followStart) > 200*time.Millisecond {
		t.Fatalf("canceled scan still occupied the slot for %s", time.Since(followStart))
	}
}

func TestAnalyzeCancelStopsScan(t *testing.T) {
	now := time.Now()
	var lines []string
	for i := 0; i < 200; i++ {
		lines = append(lines, combinedLine("203.0.113.6", now, "GET", fmt.Sprintf("/c/%d", i), 200, 2, "ua"))
	}
	logPath := filepath.Join(t.TempDir(), "access.log")
	svc := setupNginxWithAccessLog(t, "cancel.example.com", logPath, strings.Join(lines, "\n"))
	logScanDelay = 30 * time.Millisecond
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := svc.AnalyzeSite(ctx, dto.NginxLogAnalyzeReq{Site: "cancel.example.com", TimeRange: "24h"})
	if err == nil {
		t.Fatal("canceled request should not return a completed scan")
	}
}

func TestIdenticalRequestsMergeAndCacheIsolates(t *testing.T) {
	now := time.Now()
	logPath := filepath.Join(t.TempDir(), "access.log")
	svc := setupNginxWithAccessLog(t, "cache.example.com", logPath, combinedLine("203.0.113.7", now, "GET", "/hit", 200, 7, "ua"))
	logScanDelay = 80 * time.Millisecond
	var scans atomic.Int32
	logScanTestHook = func() { scans.Add(1) }
	t.Cleanup(func() { logScanTestHook = nil })

	var wg sync.WaitGroup
	var first, second *dto.NginxLogAnalysis
	wg.Add(2)
	go func() {
		defer wg.Done()
		res, err := svc.AnalyzeSite(context.Background(), dto.NginxLogAnalyzeReq{Site: "cache.example.com", TimeRange: "24h"})
		if err != nil {
			t.Errorf("first: %v", err)
			return
		}
		first = res
	}()
	go func() {
		defer wg.Done()
		res, err := svc.AnalyzeSite(context.Background(), dto.NginxLogAnalyzeReq{Site: "cache.example.com", TimeRange: "24h"})
		if err != nil {
			t.Errorf("second: %v", err)
			return
		}
		second = res
	}()
	wg.Wait()
	if scans.Load() != 1 {
		t.Fatalf("merged requests scanned %d times", scans.Load())
	}
	if first == nil || second == nil || first.TotalRequests != 1 || second.TotalRequests != 1 {
		t.Fatalf("merged results %#v %#v", first, second)
	}

	cached, err := svc.AnalyzeSite(context.Background(), dto.NginxLogAnalyzeReq{Site: "cache.example.com", TimeRange: "24h"})
	if err != nil {
		t.Fatal(err)
	}
	if !cached.Meta.CacheHit {
		t.Fatal("expected cache hit on repeat")
	}

	otherLog := filepath.Join(t.TempDir(), "other.log")
	writeLines(t, otherLog, []string{combinedLine("203.0.113.8", now, "GET", "/other", 200, 1, "ua")})
	other := setupNginxWithAccessLog(t, "other.example.com", otherLog, "")
	if err := os.WriteFile(otherLog, []byte(combinedLine("203.0.113.8", now, "GET", "/other", 200, 1, "ua")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := other.AnalyzeSite(context.Background(), dto.NginxLogAnalyzeReq{Site: "other.example.com", TimeRange: "24h"})
	if err != nil {
		t.Fatal(err)
	}
	if res.TotalRequests != 1 {
		t.Fatalf("other site total=%d", res.TotalRequests)
	}
	for _, item := range res.TopURLs {
		if item.Name == "/hit" {
			t.Fatalf("cache leaked across sites: %#v", res.TopURLs)
		}
	}
}

func TestCacheCapacityIsBounded(t *testing.T) {
	now := time.Now()
	root := t.TempDir()
	setupLogAnalysisLimits(t)
	logCacheMaxEntries = 2
	var conf bytes.Buffer
	svc := &NginxLogService{}
	for i := 0; i < 4; i++ {
		p := filepath.Join(root, fmt.Sprintf("s%d.log", i))
		name := fmt.Sprintf("s%d.example.com", i)
		writeLines(t, p, []string{combinedLine("203.0.113.9", now, "GET", fmt.Sprintf("/s/%d", i), 200, 1, "ua")})
		fmt.Fprintf(&conf, "server {\n    server_name %s;\n    access_log %s combined;\n}\n", name, p)
	}
	external := writeNginxFixture(t, filepath.Join(root, "sites", "site.conf"), conf.String())
	writeNginxFixture(t, filepath.Join(root, "conf", "nginx.conf"), "include "+external+";\n")
	writeNginxFixture(t, filepath.Join(root, "sbin", "nginx"), "")
	previous := global.CONF
	global.CONF.Nginx = global.NginxConfig{InstallDir: root, Mode: "prefix"}
	global.CONF.Nginx.DetectNginx()
	t.Cleanup(func() { global.CONF = previous })

	for i := 0; i < 4; i++ {
		name := fmt.Sprintf("s%d.example.com", i)
		if _, err := svc.AnalyzeSite(context.Background(), dto.NginxLogAnalyzeReq{Site: name, TimeRange: "24h"}); err != nil {
			t.Fatal(err)
		}
	}
	if got := logAnalysisCacheSize(); got > 2 {
		t.Fatalf("cache entries=%d", got)
	}
}

func TestDrilldownUsesSameBudgetAndMeta(t *testing.T) {
	now := time.Now()
	var lines []string
	for i := 0; i < 40; i++ {
		url := "/keep"
		if i%2 == 0 {
			url = "/other"
		}
		lines = append(lines, combinedLine("203.0.113.11", now, "GET", url, 200, 2, "ua"))
	}
	logPath := filepath.Join(t.TempDir(), "access.log")
	svc := setupNginxWithAccessLog(t, "drill.example.com", logPath, strings.Join(lines, "\n"))
	logScanMaxLines = 15
	res, err := svc.Drilldown(context.Background(), dto.NginxLogDrilldownReq{
		Site: "drill.example.com", TimeRange: "24h", FilterType: "url", FilterValue: "/keep",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Meta.SourceScope != "active" || !res.Meta.Recalculated {
		t.Fatalf("drilldown meta=%#v", res.Meta)
	}
	if !res.Meta.Partial || !containsReason(res.Meta.Reasons, "line_limit") {
		t.Fatalf("drilldown ignored budget: %#v", res.Meta)
	}
}

func TestOneWaiterCancelDoesNotDropOthers(t *testing.T) {
	now := time.Now()
	logPath := filepath.Join(t.TempDir(), "access.log")
	svc := setupNginxWithAccessLog(t, "wait.example.com", logPath, combinedLine("203.0.113.12", now, "GET", "/w", 200, 1, "ua"))
	logScanDelay = 120 * time.Millisecond
	ctx, cancel := context.WithCancel(context.Background())
	var got *dto.NginxLogAnalysis
	var gotErr error
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		time.Sleep(10 * time.Millisecond)
		cancel()
		_, _ = svc.AnalyzeSite(ctx, dto.NginxLogAnalyzeReq{Site: "wait.example.com", TimeRange: "24h"})
	}()
	go func() {
		defer wg.Done()
		got, gotErr = svc.AnalyzeSite(context.Background(), dto.NginxLogAnalyzeReq{Site: "wait.example.com", TimeRange: "24h"})
	}()
	wg.Wait()
	if gotErr != nil || got == nil || got.TotalRequests != 1 {
		t.Fatalf("remaining waiter lost result err=%v res=%#v", gotErr, got)
	}
}

func containsReason(reasons []string, want string) bool {
	for _, r := range reasons {
		if r == want {
			return true
		}
	}
	return false
}

func TestReadLastLinesDoesNotGrowByPrepend(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tail.log")
	var b strings.Builder
	for i := 0; i < 1000; i++ {
		fmt.Fprintf(&b, "line-%04d\n", i)
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	lines, err := readLastLines(f, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 5 || lines[0] != "line-0995" || lines[4] != "line-0999" {
		t.Fatalf("lines=%#v", lines)
	}
}
