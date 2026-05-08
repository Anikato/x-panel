package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestGinRoutePriority(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/apps/tags", func(c *gin.Context) { c.String(200, "tags") })
	r.GET("/apps/:key", func(c *gin.Context) { c.String(200, "key="+c.Param("key")) })
	r.GET("/apps/detail", func(c *gin.Context) { c.String(200, "detail") })

	req := httptest.NewRequest(http.MethodGet, "/apps/detail?appId=1&version=1.0", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if body := w.Body.String(); body != "detail" {
		t.Fatalf("expected static /apps/detail route to match before /apps/:key, got %q", body)
	}
}
