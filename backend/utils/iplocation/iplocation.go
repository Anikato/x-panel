package iplocation

import (
	"compress/gzip"
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
	mu     sync.RWMutex
	db     *geoip2.Reader
	dbPath string
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
	db := s.db
	s.mu.RUnlock()

	if db == nil {
		return info
	}

	ip := net.ParseIP(ipStr)
	if ip == nil {
		return info
	}

	record, err := db.City(ip)
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
	info := DBInfo{Path: s.dbPath}

	s.mu.RLock()
	info.Loaded = s.db != nil
	s.mu.RUnlock()

	if fi, err := os.Stat(s.dbPath); err == nil {
		info.Size = fi.Size()
		info.UpdatedAt = fi.ModTime().Format("2006-01-02 15:04:05")
	}
	return info
}

// DownloadDB downloads the free DB-IP City Lite MMDB database.
// URL format: https://download.db-ip.com/free/dbip-city-lite-YYYY-MM.mmdb.gz
func (s *Service) DownloadDB() error {
	now := time.Now()
	url := fmt.Sprintf("https://download.db-ip.com/free/dbip-city-lite-%d-%02d.mmdb.gz", now.Year(), now.Month())

	if err := os.MkdirAll(filepath.Dir(s.dbPath), 0755); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}

	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Get(url)
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

	tmpPath := s.dbPath + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}

	if _, err := io.Copy(f, gz); err != nil {
		f.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("write file: %w", err)
	}
	f.Close()

	if err := os.Rename(tmpPath, s.dbPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("rename file: %w", err)
	}

	s.mu.Lock()
	s.tryLoad()
	s.mu.Unlock()

	return nil
}
