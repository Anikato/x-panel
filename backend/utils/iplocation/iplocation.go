package iplocation

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/oschwald/geoip2-golang"
)

type IPInfo struct {
	IP          string `json:"ip"`
	Country     string `json:"country"`
	CountryCode string `json:"countryCode"`
	City        string `json:"city"`
	Region      string `json:"region"`
}

type DBInfo struct {
	Loaded    bool   `json:"loaded"`
	Path      string `json:"path"`
	Size      int64  `json:"size"`
	UpdatedAt string `json:"updatedAt"`
}

var (
	instance *Service
	once     sync.Once
)

type Service struct {
	mu         sync.RWMutex
	downloadMu sync.Mutex
	db         *geoip2.Reader
	dbPath     string
}

func GetService() *Service {
	once.Do(func() {
		instance = &Service{}
	})
	return instance
}

func (s *Service) Init(dataDir string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.dbPath = filepath.Join(dataDir, "GeoLite2-City.mmdb")
	s.tryLoad()
}

func (s *Service) tryLoad() {
	if s.db != nil {
		s.db.Close()
		s.db = nil
	}
	if _, err := os.Stat(s.dbPath); err == nil {
		if db, err := geoip2.Open(s.dbPath); err == nil {
			s.db = db
		}
	}
}

func (s *Service) Lookup(ipStr string) IPInfo {
	info := IPInfo{IP: ipStr}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.db == nil {
		return info
	}

	ip := net.ParseIP(ipStr)
	if ip == nil {
		return info
	}

	record, err := s.db.City(ip)
	if err != nil {
		return info
	}

	info.Country = record.Country.Names["zh-CN"]
	if info.Country == "" {
		info.Country = record.Country.Names["en"]
	}
	info.CountryCode = record.Country.IsoCode
	info.City = record.City.Names["zh-CN"]
	if info.City == "" {
		info.City = record.City.Names["en"]
	}
	if len(record.Subdivisions) > 0 {
		info.Region = record.Subdivisions[0].Names["zh-CN"]
		if info.Region == "" {
			info.Region = record.Subdivisions[0].Names["en"]
		}
	}

	return info
}

func (s *Service) LookupBatch(ips []string) map[string]IPInfo {
	result := make(map[string]IPInfo, len(ips))
	for _, ip := range ips {
		result[ip] = s.Lookup(ip)
	}
	return result
}

func (s *Service) IsLoaded() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.db != nil
}

func (s *Service) GetDBInfo() DBInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	info := DBInfo{Path: s.dbPath, Loaded: s.db != nil}

	if fi, err := os.Stat(s.dbPath); err == nil {
		info.Size = fi.Size()
		info.UpdatedAt = fi.ModTime().Format("2006-01-02 15:04:05")
	}
	return info
}

// DownloadDB downloads the free DB-IP City Lite MMDB database.
// URL format: https://download.db-ip.com/free/dbip-city-lite-YYYY-MM.mmdb.gz
func (s *Service) DownloadDB(ctx context.Context, client *http.Client) error {
	if !s.downloadMu.TryLock() {
		return fmt.Errorf("IP database download is already running")
	}
	defer s.downloadMu.Unlock()

	s.mu.RLock()
	dbPath := s.dbPath
	s.mu.RUnlock()
	now := time.Now()
	address := fmt.Sprintf("https://download.db-ip.com/free/dbip-city-lite-%d-%02d.mmdb.gz", now.Year(), now.Month())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return err
	}

	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create IP database directory: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	gz, err := gzip.NewReader(resp.Body)
	if err != nil {
		return fmt.Errorf("gzip decompress: %w", err)
	}
	defer gz.Close()

	file, err := os.CreateTemp(dir, ".ip-database-*.tmp")
	if err != nil {
		return fmt.Errorf("create IP database file: %w", err)
	}
	tmpPath := file.Name()
	defer os.Remove(tmpPath)
	_, copyErr := io.Copy(file, gz)
	closeErr := file.Close()
	if copyErr != nil {
		return fmt.Errorf("write IP database: %w", copyErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close IP database file: %w", closeErr)
	}

	db, err := geoip2.Open(tmpPath)
	if err != nil {
		return fmt.Errorf("validate IP database: %w", err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.Rename(tmpPath, dbPath); err != nil {
		_ = db.Close()
		return fmt.Errorf("replace IP database: %w", err)
	}
	if s.db != nil {
		_ = s.db.Close()
	}
	s.db = db
	return nil
}
