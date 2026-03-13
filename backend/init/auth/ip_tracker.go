package auth

import (
	"sync"
	"time"
)

const (
	CaptchaThreshold = 3
	ExpireDuration   = 30 * time.Minute
)

type ipRecord struct {
	FailCount int
	LastFail  time.Time
}

type IPTracker struct {
	mu      sync.Mutex
	records map[string]*ipRecord
}

func NewIPTracker() *IPTracker {
	t := &IPTracker{records: make(map[string]*ipRecord)}
	go t.cleanupLoop()
	return t
}

func (t *IPTracker) IncrementFail(ip string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	rec, ok := t.records[ip]
	if !ok {
		rec = &ipRecord{}
		t.records[ip] = rec
	}
	rec.FailCount++
	rec.LastFail = time.Now()
}

func (t *IPTracker) NeedCaptcha(ip string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	rec, ok := t.records[ip]
	if !ok {
		return false
	}
	if time.Since(rec.LastFail) > ExpireDuration {
		delete(t.records, ip)
		return false
	}
	return rec.FailCount >= CaptchaThreshold
}

func (t *IPTracker) Clear(ip string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.records, ip)
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
		t.mu.Unlock()
	}
}
