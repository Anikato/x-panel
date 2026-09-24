package service

import (
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseCombinedLineReadsTimingWhenPresent(t *testing.T) {
	combined := []byte(`1.2.3.4 - - [22/Sep/2026:10:00:00 +0000] "GET /old HTTP/1.1" 200 12 "-" "ua"`)
	entry, ok := parseCombinedLine(combined)
	if !ok || entry.HasRequestTime || entry.Status != 200 || entry.URL != "/old" {
		t.Fatalf("combined = %#v ok=%v", entry, ok)
	}
	extended := []byte(`1.2.3.4 - - [22/Sep/2026:10:00:00 +0000] "GET /slow HTTP/1.1" 504 12 "-" "ua" 1.250 "2.500, 0.010"`)
	entry, ok = parseCombinedLine(extended)
	if !ok || !entry.HasRequestTime || entry.RequestSeconds != 1.25 || !entry.HasUpstreamTime || entry.UpstreamSeconds != 2.5 {
		t.Fatalf("extended = %#v ok=%v", entry, ok)
	}
}

func TestScanIncludesRotatedAndGzipLogs(t *testing.T) {
	dir := t.TempDir()
	current := filepath.Join(dir, "access.log")
	rotated := current + ".1.gz"
	stamp := time.Now().UTC().Format("02/Jan/2006:15:04:05 -0700")
	line := "9.9.9.9 - - [" + stamp + "] \"GET /rotated HTTP/1.1\" 404 8 \"-\" \"ua\" 0.020 \"-\"\n"
	f, err := os.Create(rotated)
	if err != nil {
		t.Fatal(err)
	}
	zw := gzip.NewWriter(f)
	if _, err := zw.Write([]byte(line)); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil || f.Close() != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(current, []byte("8.8.8.8 - - ["+stamp+"] \"GET /now HTTP/1.1\" 200 8 \"-\" \"ua\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	sink := newStatsSink(1000)
	now := time.Now()
	meta := scanLogFiles(t.Context(), []string{current}, now.Add(-time.Hour), now.Add(time.Minute), defaultScanLimits(), sink)
	if meta.matchedLines != 2 {
		t.Fatalf("matched = %d, want 2 (current + rotated)", meta.matchedLines)
	}
	if sink.statusCodes["404"] != 1 || sink.statusCodes["200"] != 1 {
		t.Fatalf("status = %#v", sink.statusCodes)
	}
	if len(sink.result(false, nil).TopSlowURLs) == 0 || sink.result(false, nil).TopSlowURLs[0].Name != "/rotated" {
		t.Fatalf("slow = %#v", sink.result(false, nil).TopSlowURLs)
	}
}

func TestErrorLogSummaryGroupsCauses(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "error.log")
	now := time.Now()
	recent := now.Format("2006/01/02 15:04:05")
	old := now.Add(-48 * time.Hour).Format("2006/01/02 15:04:05")
	body := "" +
		recent + " [error] 1#1: *1 upstream timed out (110: Connection timed out) while reading\n" +
		recent + " [error] 1#1: *2 connect() failed (111: Connection refused) while connecting to upstream\n" +
		recent + " [error] 1#1: *3 no live upstreams while connecting to upstream\n" +
		old + " [error] 1#1: *4 upstream timed out old\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	items := summarizeErrorLogs([]string{path}, now.Add(-time.Hour), now.Add(time.Minute))
	got := map[string]int64{}
	for _, item := range items {
		got[item.Name] = item.Count
	}
	if got["upstream timed out"] != 1 || got["connect() failed"] != 1 || got["no live upstreams"] != 1 {
		t.Fatalf("summary = %#v", items)
	}
}
