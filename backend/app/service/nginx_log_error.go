package service

import (
	"bufio"
	"compress/gzip"
	"io"
	"os"
	"strings"
	"time"

	"xpanel/app/dto"
)

func summarizeErrorLogs(paths []string, cutoff, until time.Time) []dto.RankItem {
	counts := map[string]int64{}
	for _, path := range expandLogPaths(paths) {
		if path == "" || path == "off" {
			continue
		}
		lines, err := readErrorLogLines(path)
		if err != nil {
			continue
		}
		for _, line := range lines {
			ts, ok := parseErrorLogTime(line)
			if ok && (ts.Before(cutoff) || ts.After(until)) {
				continue
			}
			if !ok {
				continue
			}
			if label := classifyErrorLine(line); label != "" {
				counts[label]++
			}
		}
	}
	return topN(counts, 12)
}

func readErrorLogLines(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var reader io.Reader = f
	if strings.HasSuffix(strings.ToLower(path), ".gz") {
		zr, err := gzip.NewReader(f)
		if err != nil {
			return nil, err
		}
		defer zr.Close()
		reader = io.LimitReader(zr, 8<<20)
	} else if info, err := f.Stat(); err == nil && info.Size() > 4<<20 {
		if _, err := f.Seek(info.Size()-(4<<20), io.SeekStart); err == nil {
			reader = f
		}
	}
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

func parseErrorLogTime(line string) (time.Time, bool) {
	if len(line) < 19 {
		return time.Time{}, false
	}
	ts, err := time.ParseInLocation("2006/01/02 15:04:05", line[:19], time.Local)
	return ts, err == nil
}

func classifyErrorLine(line string) string {
	switch {
	case strings.Contains(line, "[info]"), strings.Contains(line, "[notice]"), strings.Contains(line, "[debug]"):
		return ""
	case strings.Contains(line, "upstream timed out"):
		return "upstream timed out"
	case strings.Contains(line, "no live upstreams"):
		return "no live upstreams"
	case strings.Contains(line, "connect() failed"):
		return "connect() failed"
	case strings.Contains(line, "recv() failed"):
		return "recv() failed"
	case strings.Contains(line, "SSL_do_handshake"):
		return "SSL handshake failed"
	case strings.Contains(line, "limiting requests"):
		return "limiting requests"
	case strings.Contains(line, "Permission denied"):
		return "permission denied"
	case strings.Contains(line, "directory index of"):
		return "directory index forbidden"
	case strings.Contains(line, "[error]"), strings.Contains(line, "[crit]"), strings.Contains(line, "[alert]"), strings.Contains(line, "[emerg]"):
		return "other error"
	default:
		return ""
	}
}
