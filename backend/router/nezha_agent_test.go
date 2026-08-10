package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestNezhaAgentPrivateRoutesRegistered(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := Setup(gin.TestMode)

	want := map[string]bool{
		http.MethodGet + " /api/v1/nezha-agent/status":   true,
		http.MethodPut + " /api/v1/nezha-agent/config":   true,
		http.MethodPost + " /api/v1/nezha-agent/operate": true,
		http.MethodPost + " /api/v1/nezha-agent/install": true,
	}
	found := map[string]bool{}
	for _, route := range r.Routes() {
		key := route.Method + " " + route.Path
		if want[key] {
			found[key] = true
		}
		if route.Path == "/api/v1/nezha-agent/logs" {
			t.Fatal("nezha-agent logs route must not be registered")
		}
	}
	for key := range want {
		if !found[key] {
			t.Fatalf("missing private route %s", key)
		}
	}
}

func TestNezhaAgentRoutesRequireJWT(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := Setup(gin.TestMode)

	// Only unauthenticated requests: avoid JWT write paths that trigger
	// OperationLog's async database side effects.
	cases := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/nezha-agent/status"},
		{http.MethodPut, "/api/v1/nezha-agent/config"},
		{http.MethodPost, "/api/v1/nezha-agent/operate"},
		{http.MethodPost, "/api/v1/nezha-agent/install"},
	}
	for _, tc := range cases {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s status = %d, want 401", tc.method, tc.path, rec.Code)
		}
	}
}
