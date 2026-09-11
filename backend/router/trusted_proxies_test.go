package router

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"xpanel/app/api/v1/helper"
	"xpanel/global"
	"xpanel/init/auth"

	"github.com/gin-gonic/gin"
)

func probeEngine(t *testing.T) *gin.Engine {
	t.Helper()
	engine := Setup(gin.TestMode)
	engine.GET("/api/v1/__client-ip", func(c *gin.Context) {
		c.String(http.StatusOK, helper.GetClientIP(c))
	})
	return engine
}

func clientIPFrom(t *testing.T, engine *gin.Engine, remoteAddr, forwarded, realIP string) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/__client-ip", nil)
	req.RemoteAddr = remoteAddr
	if forwarded != "" {
		req.Header.Set("X-Forwarded-For", forwarded)
	}
	if realIP != "" {
		req.Header.Set("X-Real-IP", realIP)
	}
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	return rec.Body.String()
}

func TestSetupIgnoresSpoofedForwardedIP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	previous := global.CONF
	global.CONF.System.TrustedProxies = nil
	t.Cleanup(func() { global.CONF = previous })

	got := clientIPFrom(t, probeEngine(t), "192.0.2.10:54321", "198.51.100.1", "203.0.113.1")
	if got != "192.0.2.10" {
		t.Fatalf("client IP = %q, want remote address", got)
	}
}

func TestSetupTrustedProxyUsesForwardedClientIP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	previous := global.CONF
	global.CONF.System.TrustedProxies = []string{"192.0.2.10"}
	t.Cleanup(func() { global.CONF = previous })

	got := clientIPFrom(t, probeEngine(t), "192.0.2.10:54321", "198.51.100.7", "")
	if got != "198.51.100.7" {
		t.Fatalf("client IP = %q, want forwarded client", got)
	}
}

func TestSpoofedForwardedIPDoesNotSplitCaptchaCounter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	previous := global.CONF
	global.CONF.System.TrustedProxies = nil
	t.Cleanup(func() { global.CONF = previous })

	engine := probeEngine(t)
	tracker := auth.NewIPTracker()
	var lastIP string
	for i := 0; i < auth.CaptchaThreshold; i++ {
		lastIP = clientIPFrom(t, engine, "192.0.2.10:54321", fmt.Sprintf("198.51.100.%d", i+1), "")
		if lastIP != "192.0.2.10" {
			t.Fatalf("client IP = %q", lastIP)
		}
		tracker.IncrementFail(lastIP)
	}
	if !tracker.NeedCaptcha(lastIP) {
		t.Fatal("same direct source should still trigger captcha")
	}
}

func TestSetupUntrustedSourceCannotSpoofTrustedProxy(t *testing.T) {
	gin.SetMode(gin.TestMode)
	previous := global.CONF
	global.CONF.System.TrustedProxies = []string{"10.0.0.1"}
	t.Cleanup(func() { global.CONF = previous })

	got := clientIPFrom(t, probeEngine(t), "192.0.2.10:54321", "198.51.100.7", "198.51.100.7")
	if got != "192.0.2.10" {
		t.Fatalf("client IP = %q, want remote address", got)
	}
}
