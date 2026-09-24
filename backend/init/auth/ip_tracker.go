package auth

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	CaptchaThreshold = 3
	ExpireDuration   = 30 * time.Minute
)

type ipRecord struct {
	FailCount int       `json:"failCount"`
	LastFail  time.Time `json:"lastFail"`
}

type failureState struct {
	GlobalCount int                 `json:"globalCount"`
	GlobalLast  time.Time           `json:"globalLast"`
	Records     map[string]ipRecord `json:"records"`
}

type IPTracker struct {
	mu          sync.Mutex
	path        string
	globalCount int
	globalLast  time.Time
	records     map[string]*ipRecord
}

func NewIPTracker() *IPTracker {
	t := &IPTracker{records: make(map[string]*ipRecord)}
	go t.cleanupLoop()
	return t
}

func (t *IPTracker) UseFile(path string) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.path = path
	return t.loadLocked()
}

func (t *IPTracker) IncrementFail(ip string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	now := time.Now()
	rec, ok := t.records[ip]
	if !ok {
		rec = &ipRecord{}
		t.records[ip] = rec
	}
	rec.FailCount++
	rec.LastFail = now
	if t.globalLast.IsZero() || now.Sub(t.globalLast) > ExpireDuration {
		t.globalCount = 0
	}
	t.globalCount++
	t.globalLast = now
	_ = t.saveLocked()
}

func (t *IPTracker) NeedCaptcha(ip string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.captchaRequiredLocked(ip, time.Now()) {
		return true
	}
	return false
}

func (t *IPTracker) captchaRequiredLocked(ip string, now time.Time) bool {
	if !t.globalLast.IsZero() && now.Sub(t.globalLast) <= ExpireDuration && t.globalCount >= CaptchaThreshold {
		return true
	}
	rec, ok := t.records[ip]
	if !ok {
		return false
	}
	if now.Sub(rec.LastFail) > ExpireDuration {
		delete(t.records, ip)
		return false
	}
	return rec.FailCount >= CaptchaThreshold
}

func (t *IPTracker) Clear(ip string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.records, ip)
	t.globalCount = 0
	t.globalLast = time.Time{}
	_ = t.saveLocked()
}

func (t *IPTracker) cleanupLoop() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		t.mu.Lock()
		now := time.Now()
		for ip, rec := range t.records {
			if now.Sub(rec.LastFail) > ExpireDuration {
				delete(t.records, ip)
			}
		}
		if !t.globalLast.IsZero() && now.Sub(t.globalLast) > ExpireDuration {
			t.globalCount = 0
			t.globalLast = time.Time{}
		}
		_ = t.saveLocked()
		t.mu.Unlock()
	}
}

func (t *IPTracker) loadLocked() error {
	if t.path == "" {
		return nil
	}
	data, err := os.ReadFile(t.path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	var state failureState
	if err := json.Unmarshal(data, &state); err != nil {
		return err
	}
	t.globalCount = state.GlobalCount
	t.globalLast = state.GlobalLast
	t.records = make(map[string]*ipRecord, len(state.Records))
	now := time.Now()
	for ip, rec := range state.Records {
		if now.Sub(rec.LastFail) > ExpireDuration {
			continue
		}
		copied := rec
		t.records[ip] = &copied
	}
	if !t.globalLast.IsZero() && now.Sub(t.globalLast) > ExpireDuration {
		t.globalCount = 0
		t.globalLast = time.Time{}
	}
	return nil
}

func (t *IPTracker) saveLocked() error {
	if t.path == "" {
		return nil
	}
	state := failureState{
		GlobalCount: t.globalCount,
		GlobalLast:  t.globalLast,
		Records:     make(map[string]ipRecord, len(t.records)),
	}
	for ip, rec := range t.records {
		state.Records[ip] = *rec
	}
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(t.path), 0700); err != nil {
		return err
	}
	tmp := t.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, t.path)
}
