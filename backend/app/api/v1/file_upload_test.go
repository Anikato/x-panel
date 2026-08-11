package v1

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
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

func performChunkUpload(t *testing.T, fields map[string]string, filename string, content []byte) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for _, key := range []string{"path", "relativePath", "uploadID", "chunkIndex", "chunkCount", "totalSize", "checksum"} {
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
	if _, err := part.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return performFileHandler(http.MethodPost, "/files/upload/chunk", writer.FormDataContentType(), &body, (&FileAPI{}).UploadFileChunk)
}

func chunkFields(root, relativePath, uploadID string, content []byte) map[string]string {
	sum := sha256.Sum256(content)
	return map[string]string{
		"path":         root,
		"relativePath": relativePath,
		"uploadID":     uploadID,
		"chunkIndex":   "0",
		"chunkCount":   "1",
		"totalSize":    fmt.Sprintf("%d", len(content)),
		"checksum":     fmt.Sprintf("%x", sum),
	}
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

func TestFileUploadChunkLifecycle(t *testing.T) {
	installFileHandlerDatabase(t)
	root := t.TempDir()
	content := []byte("chunked upload")
	uploadID := "66666666666666666666666666666666"

	chunkRecorder := performChunkUpload(t, chunkFields(root, "nested/archive.bin", uploadID, content), "archive.bin", content)
	if chunkRecorder.Code != http.StatusOK {
		t.Fatalf("chunk status=%d body=%s", chunkRecorder.Code, chunkRecorder.Body.String())
	}
	completeBody := strings.NewReader(`{"targetPath":` + strconvQuote(root) + `,"relativePath":"nested/archive.bin","uploadID":"` + uploadID + `","totalSize":14,"overwrite":false,"batch":true}`)
	completeRecorder := performFileHandler(
		http.MethodPost, "/files/upload/chunk/complete", "application/json",
		completeBody, (&FileAPI{}).CompleteFileChunks,
	)
	if completeRecorder.Code != http.StatusOK {
		t.Fatalf("complete status=%d body=%s", completeRecorder.Code, completeRecorder.Body.String())
	}
	data, err := os.ReadFile(filepath.Join(root, "nested", "archive.bin"))
	if err != nil || !bytes.Equal(data, content) {
		t.Fatalf("data=%q err=%v", data, err)
	}
}

func TestFileUploadChunkRejectsInvalidChecksum(t *testing.T) {
	root := t.TempDir()
	content := []byte("chunk")
	fields := chunkFields(root, "archive.bin", "77777777777777777777777777777777", content)
	fields["checksum"] = strings.Repeat("0", 64)
	recorder := performChunkUpload(t, fields, "archive.bin", content)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if _, err := os.Stat(filepath.Join(root, "archive.bin")); !os.IsNotExist(err) {
		t.Fatalf("target stat err=%v", err)
	}
}

func TestFileUploadChunkRejectsMalformedChecksumAsBadRequest(t *testing.T) {
	root := t.TempDir()
	content := []byte("chunk")
	fields := chunkFields(root, "archive.bin", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", content)
	fields["checksum"] = "not-a-sha256"
	recorder := performChunkUpload(t, fields, "archive.bin", content)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestFileUploadChunkRejectsMalformedUploadIDAsBadRequest(t *testing.T) {
	root := t.TempDir()
	content := []byte("chunk")
	fields := chunkFields(root, "archive.bin", "invalid-upload-id", content)
	recorder := performChunkUpload(t, fields, "archive.bin", content)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestCompleteFileChunksSizeMismatchPreservesExistingFile(t *testing.T) {
	installFileHandlerDatabase(t)
	root := t.TempDir()
	target := filepath.Join(root, "archive.bin")
	if err := os.WriteFile(target, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	content := []byte("new")
	uploadID := "88888888888888888888888888888888"
	chunkRecorder := performChunkUpload(t, chunkFields(root, "archive.bin", uploadID, content), "archive.bin", content)
	if chunkRecorder.Code != http.StatusOK {
		t.Fatalf("chunk status=%d body=%s", chunkRecorder.Code, chunkRecorder.Body.String())
	}
	completeBody := strings.NewReader(`{"targetPath":` + strconvQuote(root) + `,"relativePath":"archive.bin","uploadID":"` + uploadID + `","totalSize":4,"overwrite":true,"batch":true}`)
	completeRecorder := performFileHandler(
		http.MethodPost, "/files/upload/chunk/complete", "application/json",
		completeBody, (&FileAPI{}).CompleteFileChunks,
	)
	if completeRecorder.Code != http.StatusBadRequest {
		t.Fatalf("complete status=%d body=%s", completeRecorder.Code, completeRecorder.Body.String())
	}
	data, err := os.ReadFile(target)
	if err != nil || string(data) != "old" {
		t.Fatalf("target data=%q err=%v", data, err)
	}
}
