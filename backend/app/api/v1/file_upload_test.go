package v1

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"xpanel/app/model"
	"xpanel/global"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func performFileHandler(method, target, contentType string, body io.Reader, handler gin.HandlerFunc) *httptest.ResponseRecorder {
	testingMode := gin.Mode()
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(method, target, body)
	context.Request.Header.Set("Content-Type", contentType)
	handler(context)
	gin.SetMode(testingMode)
	return recorder
}

func installFileHandlerDatabase(t *testing.T) *gorm.DB {
	t.Helper()
	database, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.AutoMigrate(&model.Setting{}, &model.Notification{}); err != nil {
		t.Fatal(err)
	}
	previous := global.DB
	global.DB = database
	t.Cleanup(func() { global.DB = previous })
	return database
}

func performUpload(t *testing.T, fields map[string]string, filename, content string) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for _, key := range []string{"path", "relativePath", "overwrite", "batch"} {
		if value, ok := fields[key]; ok {
			if err := writer.WriteField(key, value); err != nil {
				t.Fatal(err)
			}
		}
	}
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(part, content); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	return performFileHandler(http.MethodPost, "/files/upload", writer.FormDataContentType(), &body, (&FileAPI{}).UploadFile)
}

func decodeFileResponse(t *testing.T, recorder *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var response map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response %q: %v", recorder.Body.String(), err)
	}
	return response
}

func TestUploadPreflightReportsConflicts(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "index.html"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	body := strings.NewReader(`{"targetPath":` + strconvQuote(root) + `,"relativePaths":["index.html","assets/app.js"]}`)
	recorder := performFileHandler(
		http.MethodPost,
		"/files/upload/preflight",
		"application/json",
		body,
		(&FileAPI{}).PreflightUpload,
	)

	response := decodeFileResponse(t, recorder)
	if got := int(response["code"].(float64)); got != 0 {
		t.Fatalf("code=%d response=%s", got, recorder.Body.String())
	}
	data := response["data"].(map[string]any)
	conflicts := data["conflicts"].([]any)
	if len(conflicts) != 1 || conflicts[0] != "index.html" {
		t.Fatalf("conflicts=%#v", conflicts)
	}
}

func strconvQuote(value string) string {
	data, _ := json.Marshal(value)
	return string(data)
}

func TestFileUploadPreservesRelativePath(t *testing.T) {
	installFileHandlerDatabase(t)
	root := t.TempDir()
	recorder := performUpload(t, map[string]string{
		"path":         root,
		"relativePath": "site/index.html",
		"overwrite":    "false",
		"batch":        "true",
	}, "index.html", "new")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	data, err := os.ReadFile(filepath.Join(root, "site", "index.html"))
	if err != nil || string(data) != "new" {
		t.Fatalf("data=%q err=%v", data, err)
	}
}

func TestFileUploadSkipReturnsConflict(t *testing.T) {
	installFileHandlerDatabase(t)
	root := t.TempDir()
	target := filepath.Join(root, "config.json")
	if err := os.WriteFile(target, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}

	recorder := performUpload(t, map[string]string{
		"path":         root,
		"relativePath": "config.json",
		"overwrite":    "false",
		"batch":        "true",
	}, "config.json", "new")
	if recorder.Code != http.StatusConflict {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	data, err := os.ReadFile(target)
	if err != nil || string(data) != "old" {
		t.Fatalf("data=%q err=%v", data, err)
	}
}

func TestFileUploadBatchSuppressesNotification(t *testing.T) {
	database := installFileHandlerDatabase(t)
	root := t.TempDir()
	recorder := performUpload(t, map[string]string{
		"path":         root,
		"relativePath": "notes.txt",
		"overwrite":    "false",
		"batch":        "true",
	}, "notes.txt", "new")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var count int64
	if err := database.Model(&model.Notification{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("notifications=%d", count)
	}
}
