package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"xpanel/app/model"
	"xpanel/global"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func runOperationLogRequest(
	t *testing.T,
	path string,
	contentType string,
	body []byte,
	handlerErr error,
) ([]byte, model.OperationLog) {
	t.Helper()

	db, err := gorm.Open(
		sqlite.Open(filepath.Join(t.TempDir(), "operation-log.db")),
		&gorm.Config{Logger: logger.Default.LogMode(logger.Silent)},
	)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := db.AutoMigrate(&model.OperationLog{}); err != nil {
		t.Fatalf("migrate operation log: %v", err)
	}
	global.DB = db
	global.LOG = logrus.New()

	gin.SetMode(gin.TestMode)
	var received []byte
	router := gin.New()
	router.Use(OperationLog())
	router.POST(path, func(c *gin.Context) {
		received, _ = io.ReadAll(c.Request.Body)
		if handlerErr != nil {
			_ = c.Error(handlerErr)
		}
		c.Status(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	router.ServeHTTP(httptest.NewRecorder(), request)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		var item model.OperationLog
		if err := db.Order("id DESC").First(&item).Error; err == nil {
			return received, item
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("operation log was not persisted")
	return nil, model.OperationLog{}
}

func TestOperationLogOmitsSensitiveRouteBody(t *testing.T) {
	secret := "SENTINEL_PRIVATE_KEY_7e4a"
	body := []byte(`{"name":"node","privateKey":"` + secret + `","sshPassword":"pw"}`)

	_, item := runOperationLogRequest(t, "/api/v1/hosts", "application/json", body, nil)

	if item.Body != "[sensitive body omitted]" || strings.Contains(item.Body, secret) {
		t.Fatalf("sensitive body leaked: %q", item.Body)
	}
}

func TestOperationLogRecursivelyRedactsOrdinaryJSON(t *testing.T) {
	agentSecret := "SENTINEL_AGENT_TOKEN_53f1"
	statsSecret := "SENTINEL_STATS_PASSWORD_c84e"
	body := []byte(
		`{"enabled":true,"nested":[{"Agent_Token":"` + agentSecret +
			`","stats-pass":"` + statsSecret + `"}]}`,
	)

	received, item := runOperationLogRequest(
		t,
		"/api/v1/notifications/preference",
		"application/json",
		body,
		nil,
	)

	if strings.Contains(item.Body, agentSecret) ||
		strings.Contains(item.Body, statsSecret) ||
		!strings.Contains(item.Body, `"enabled":true`) {
		t.Fatalf("unexpected sanitized body: %q", item.Body)
	}
	if !bytes.Equal(received, body) {
		t.Fatalf("downstream body = %q, want %q", received, body)
	}
	var decoded map[string]interface{}
	if err := json.Unmarshal([]byte(item.Body), &decoded); err != nil {
		t.Fatalf("sanitized body is not valid JSON: %v", err)
	}
}

func TestSensitiveOperationPathsCoverCredentialRoutes(t *testing.T) {
	sensitivePaths := []string{
		"/api/v1/gost/services",
		"/api/v1/gost/chains/update",
		"/api/v1/containers",
		"/api/v1/containers/compose",
		"/api/v1/toolbox/services/create",
		"/api/v1/toolbox/services/unit",
		"/api/v1/nginx/conf-file/save",
		"/api/v1/haproxy/stats/settings",
	}
	for _, path := range sensitivePaths {
		if !isSensitiveOperationPath(path) {
			t.Errorf("credential-bearing path is not sensitive: %s", path)
		}
	}
	if isSensitiveOperationPath("/api/v1/notifications/preference") {
		t.Fatal("ordinary notification preference path must remain JSON-redacted")
	}
}

func TestOperationLogPreservesOversizedRequestBody(t *testing.T) {
	body := []byte(`{"content":"` + strings.Repeat("x", maxLogBodySize+4096) + `"}`)

	received, item := runOperationLogRequest(
		t,
		"/api/v1/files/save",
		"application/json",
		body,
		nil,
	)

	if !bytes.Equal(received, body) {
		t.Fatalf("downstream body length = %d, want %d", len(received), len(body))
	}
	if item.Body != "[sensitive body omitted]" {
		t.Fatalf("log body was not omitted; length=%d", len(item.Body))
	}
}

func TestOperationLogOmitsUnsafeBodyFormats(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        []byte
		want        string
	}{
		{
			name:        "plain text",
			contentType: "text/plain",
			body:        []byte("password=SENTINEL_TEXT_SECRET"),
			want:        "[request body omitted: text/plain]",
		},
		{
			name:        "invalid json",
			contentType: "application/json",
			body:        []byte(`{"password":`),
			want:        invalidJSONOmitted,
		},
		{
			name:        "multipart",
			contentType: "multipart/form-data; boundary=test",
			body:        []byte("SENTINEL_MULTIPART_SECRET"),
			want:        "[request body omitted: multipart/form-data]",
		},
		{
			name:        "url encoded",
			contentType: "application/x-www-form-urlencoded",
			body:        []byte("password=SENTINEL_FORM_SECRET"),
			want:        "[request body omitted: application/x-www-form-urlencoded]",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			received, item := runOperationLogRequest(
				t,
				"/api/v1/notifications/preference",
				test.contentType,
				test.body,
				nil,
			)
			if item.Body != test.want {
				t.Fatalf("body = %q, want %q", item.Body, test.want)
			}
			if !bytes.Equal(received, test.body) {
				t.Fatalf("downstream body = %q, want %q", received, test.body)
			}
		})
	}
}

func TestOperationLogRedactsErrorMessage(t *testing.T) {
	tests := []struct {
		name    string
		secret  string
		message string
	}{
		{
			name:    "bearer token",
			secret:  "SENTINEL_BEARER_3cc2",
			message: "upstream authorization=Bearer SENTINEL_BEARER_3cc2",
		},
		{
			name:    "password assignment",
			secret:  "SENTINEL_MESSAGE_PASSWORD_a913",
			message: "database password=SENTINEL_MESSAGE_PASSWORD_a913 rejected",
		},
		{
			name:    "quoted json field",
			secret:  "SENTINEL_MESSAGE_AGENT_TOKEN_d721",
			message: `request failed: {"AgentToken":"SENTINEL_MESSAGE_AGENT_TOKEN_d721"}`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, item := runOperationLogRequest(
				t,
				"/api/v1/notifications/preference",
				"application/json",
				[]byte(`{}`),
				fmt.Errorf("%s", test.message),
			)
			if strings.Contains(item.Message, test.secret) {
				t.Fatalf("message leaked: %q", item.Message)
			}
		})
	}
}
