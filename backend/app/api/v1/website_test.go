package v1

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"xpanel/app/model"
	"xpanel/global"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func performWebsiteHandler(method, path, body string, handler gin.HandlerFunc) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(method, path, strings.NewReader(body))
	context.Request.Header.Set("Content-Type", "application/json")
	handler(context)
	return recorder
}

func requireWebsiteResponseCode(t *testing.T, recorder *httptest.ResponseRecorder, want int) map[string]any {
	t.Helper()
	var response map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response %q: %v", recorder.Body.String(), err)
	}
	if got, _ := response["code"].(float64); int(got) != want {
		t.Fatalf("response code = %v, want %d: %s", response["code"], want, recorder.Body.String())
	}
	return response
}

func installWebsiteHandlerDatabase(t *testing.T) {
	t.Helper()
	database, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := database.AutoMigrate(&model.Website{}); err != nil {
		t.Fatalf("migrate website: %v", err)
	}
	previous := global.DB
	global.DB = database
	t.Cleanup(func() { global.DB = previous })
}

func TestWebsiteExternalInspectRejectsMissingPath(t *testing.T) {
	recorder := performWebsiteHandler(http.MethodPost, "/websites/external/inspect", `{}`, (&WebsiteAPI{}).InspectExternalNginxSite)
	requireWebsiteResponseCode(t, recorder, http.StatusInternalServerError)
}

func TestWebsiteExternalRefreshRejectsMissingID(t *testing.T) {
	recorder := performWebsiteHandler(http.MethodPost, "/websites/external/refresh", `{}`, (&WebsiteAPI{}).RefreshExternalNginxSite)
	requireWebsiteResponseCode(t, recorder, http.StatusInternalServerError)
}

func TestWebsiteCertificateHealthRejectsMissingID(t *testing.T) {
	recorder := performWebsiteHandler(http.MethodPost, "/websites/certificate-health", `{}`, (&WebsiteAPI{}).CheckWebsiteCertificateHealth)
	requireWebsiteResponseCode(t, recorder, http.StatusInternalServerError)
}

func TestWebsiteCertificateHealthBatchBindsAllRequest(t *testing.T) {
	installWebsiteHandlerDatabase(t)
	recorder := performWebsiteHandler(http.MethodPost, "/websites/certificate-health/batch", `{"all":true}`, (&WebsiteAPI{}).CheckWebsiteCertificateHealthBatch)
	response := requireWebsiteResponseCode(t, recorder, 0)
	data, ok := response["data"].([]any)
	if !ok || len(data) != 0 {
		t.Fatalf("batch data = %#v, want empty array", response["data"])
	}
}

func TestWebsiteSourceContentErrorDoesNotLeakContent(t *testing.T) {
	installWebsiteHandlerDatabase(t)
	const secret = "server { ssl_certificate_key /do/not/leak.pem; }"
	recorder := performWebsiteHandler(
		http.MethodPost,
		"/websites/conf-content",
		`{"id":999,"content":"`+secret+`"}`,
		(&WebsiteAPI{}).GetSiteConfContent,
	)
	requireWebsiteResponseCode(t, recorder, http.StatusInternalServerError)
	if strings.Contains(recorder.Body.String(), secret) {
		t.Fatalf("error response leaked source content: %s", recorder.Body.String())
	}
}
