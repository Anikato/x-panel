package router

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCertificateBatchDeletePrivateRoutesRegistered(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := Setup(gin.TestMode)
	want := map[string]bool{
		http.MethodPost + " /api/v1/certificates/batch-del":       true,
		http.MethodPost + " /api/v1/certificates/cleanup-expired": true,
	}
	found := map[string]bool{}
	for _, route := range r.Routes() {
		key := route.Method + " " + route.Path
		if want[key] {
			found[key] = true
		}
	}
	for key := range want {
		if !found[key] {
			t.Fatalf("missing private route %s", key)
		}
	}
}
