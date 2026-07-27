package v1

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSearchCertificateRenewalPlanRejectsInvalidRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(
		http.MethodPost,
		"/api/v1/certificates/renewal-plan/search",
		strings.NewReader(`{"page":0,"pageSize":0}`),
	)
	context.Request.Header.Set("Content-Type", "application/json")

	(&SSLAPI{}).SearchCertificateRenewalPlan(context)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if !strings.Contains(recorder.Body.String(), "500") {
		t.Fatalf("unexpected response: %s", recorder.Body.String())
	}
}

func TestSearchCertificateRenewalPlanRejectsInvalidManagementType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(
		http.MethodPost,
		"/api/v1/certificates/renewal-plan/search",
		strings.NewReader(`{"page":1,"pageSize":20,"managementType":"unexpected"}`),
	)
	context.Request.Header.Set("Content-Type", "application/json")

	(&SSLAPI{}).SearchCertificateRenewalPlan(context)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if !strings.Contains(recorder.Body.String(), "500") {
		t.Fatalf("unexpected response: %s", recorder.Body.String())
	}
}
