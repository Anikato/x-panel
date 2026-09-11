package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"xpanel/app/dto"
	"xpanel/global"
)

func writeSyntheticLog(tb testing.TB, path string, size int, unique int) {
	tb.Helper()
	f, err := os.Create(path)
	if err != nil {
		tb.Fatal(err)
	}
	defer f.Close()
	now := time.Now()
	var written int
	i := 0
	for written < size {
		ip := fmt.Sprintf("10.0.%d.%d", (i/250)%8, i%250)
		url := fmt.Sprintf("/item/%d", i%unique)
		line := combinedLine(ip, now.Add(-time.Duration(i%3600)*time.Second), "GET", url, 200, 64, "bench-agent") + "\n"
		n, err := f.WriteString(line)
		if err != nil {
			tb.Fatal(err)
		}
		written += n
		i++
	}
}

func setupBenchNginx(tb testing.TB, logPath string) *NginxLogService {
	tb.Helper()
	resetLogAnalysisStateForTest()
	root := tTempDir(tb)
	conf := fmt.Sprintf("server {\n    server_name bench.example.com;\n    access_log %s combined;\n}\n", logPath)
	external := writeNginxFixture(tb, filepath.Join(root, "sites", "site.conf"), conf)
	writeNginxFixture(tb, filepath.Join(root, "conf", "nginx.conf"), "include "+external+";\n")
	writeNginxFixture(tb, filepath.Join(root, "sbin", "nginx"), "")
	previous := global.CONF
	global.CONF.Nginx = global.NginxConfig{InstallDir: root, Mode: "prefix"}
	global.CONF.Nginx.DetectNginx()
	tb.Cleanup(func() {
		global.CONF = previous
		resetLogAnalysisStateForTest()
	})
	return &NginxLogService{}
}

func tTempDir(tb testing.TB) string {
	tb.Helper()
	return tb.TempDir()
}

func sampleHeapPeak(stop <-chan struct{}) *atomic.Uint64 {
	peak := &atomic.Uint64{}
	go func() {
		var ms runtime.MemStats
		ticker := time.NewTicker(time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				runtime.ReadMemStats(&ms)
				if ms.HeapAlloc > peak.Load() {
					peak.Store(ms.HeapAlloc)
				}
				return
			case <-ticker.C:
				runtime.ReadMemStats(&ms)
				if ms.HeapAlloc > peak.Load() {
					peak.Store(ms.HeapAlloc)
				}
			}
		}
	}()
	return peak
}

func TestNginxLogScanPerformanceSamples(t *testing.T) {
	sizes := []int{10 << 20, 100 << 20}
	if os.Getenv("XP_LOG_BENCH_1G") == "1" {
		sizes = append(sizes, 1<<30)
	}
	for _, size := range sizes {
		t.Run(fmt.Sprintf("%dMiB", size>>20), func(t *testing.T) {
			logPath := filepath.Join(t.TempDir(), "access.log")
			writeSyntheticLog(t, logPath, size, 500)
			svc := setupBenchNginx(t, logPath)
			runtime.GC()
			var baseline runtime.MemStats
			runtime.ReadMemStats(&baseline)

			stop := make(chan struct{})
			peak := sampleHeapPeak(stop)
			start := time.Now()
			res, err := svc.AnalyzeSite(context.Background(), dto.NginxLogAnalyzeReq{Site: "bench.example.com", TimeRange: "30d"})
			elapsed := time.Since(start)
			close(stop)
			time.Sleep(2 * time.Millisecond)
			if err != nil {
				t.Fatal(err)
			}
			heapPeak := peak.Load()
			var heapDelta uint64
			if heapPeak > baseline.HeapAlloc {
				heapDelta = heapPeak - baseline.HeapAlloc
			}
			t.Logf("machine=%s/%s size=%dMiB elapsed=%s heapBaseline=%dKiB heapPeak=%dKiB heapDelta=%dKiB scannedBytes=%d scannedLines=%d matched=%d partial=%v reasons=%v cacheHit=%v",
				runtime.GOOS, runtime.GOARCH, size>>20, elapsed, baseline.HeapAlloc/1024, heapPeak/1024, heapDelta/1024,
				res.Meta.ScannedBytes, res.Meta.ScannedLines, res.Meta.MatchedLines, res.Meta.Partial, res.Meta.Reasons, res.Meta.CacheHit)

			repeatStart := time.Now()
			cached, err := svc.AnalyzeSite(context.Background(), dto.NginxLogAnalyzeReq{Site: "bench.example.com", TimeRange: "30d"})
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("repeat cacheHit=%v elapsed=%s", cached.Meta.CacheHit, time.Since(repeatStart))

			started := make(chan struct{})
			logScanTestHook = func() {
				select {
				case <-started:
				default:
					close(started)
				}
			}
			t.Cleanup(func() { logScanTestHook = nil })
			ctx, cancel := context.WithCancel(context.Background())
			errCh := make(chan error, 1)
			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, scanErr := svc.AnalyzeSite(ctx, dto.NginxLogAnalyzeReq{Site: "bench.example.com", TimeRange: "30d", Refresh: true})
				errCh <- scanErr
			}()
			select {
			case <-started:
			case <-time.After(3 * time.Second):
				cancel()
				t.Fatal("in-flight scan never started")
			}
			time.Sleep(15 * time.Millisecond)
			cancelStart := time.Now()
			cancel()
			scanErr := <-errCh
			wg.Wait()
			callerElapsed := time.Since(cancelStart)
			for logAnalysisInFlight() > 0 && time.Since(cancelStart) < 2*time.Second {
				time.Sleep(time.Millisecond)
			}
			t.Logf("in-flight cancel caller returned in %s err=%v; worker stopped in %s inFlight=%d", callerElapsed, scanErr, time.Since(cancelStart), logAnalysisInFlight())
		})
	}
}

func BenchmarkNginxLogAnalyze(b *testing.B) {
	logPath := filepath.Join(b.TempDir(), "access.log")
	writeSyntheticLog(b, logPath, 10<<20, 5000)
	svc := setupBenchNginx(b, logPath)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := svc.AnalyzeSite(context.Background(), dto.NginxLogAnalyzeReq{Site: "bench.example.com", TimeRange: "24h", Refresh: true}); err != nil {
			b.Fatal(err)
		}
	}
}
