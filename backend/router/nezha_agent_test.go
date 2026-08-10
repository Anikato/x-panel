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

	want := map[string]string{
		http.MethodGet:  "/api/v1/nezha-agent/status",
		http.MethodPut:  "/api/v1/nezha-agent/config",
		http.MethodPost: "/api/v1/nezha-agent/operate",
	}
	found := map[string]bool{}
	for _, route := range r.Routes() {
		if path, ok := want[route.Method]; ok && route.Path == path {
			found[route.Method+" "+route.Path] = true
		}
		if route.Path == "/api/v1/nezha-agent/logs" {
			t.Fatal("nezha-agent logs route must not be registered")
		}
	}
	for method, path := range want {
		key := method + " " + path
		if !found[key] {
			t.Fatalf("missing private route %s %s", method, path)
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
