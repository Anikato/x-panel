package v1

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"xpanel/global"

	"github.com/sirupsen/logrus"
)

func installFileAPILog(t *testing.T) {
	t.Helper()
	previous := global.LOG
	global.LOG = logrus.New()
	t.Cleanup(func() { global.LOG = previous })
}

func postMoveFile(t *testing.T, body map[string]any) *httpResponse {
	t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	recorder := performFileHandler(
		http.MethodPost,
		"/files/move",
		"application/json",
		bytes.NewReader(payload),
		(&FileAPI{}).MoveFile,
	)
	return &httpResponse{code: recorder.Code, body: recorder.Body.Bytes(), decoded: decodeFileResponse(t, recorder)}
}

type httpResponse struct {
	code    int
	body    []byte
	decoded map[string]any
}

func TestMoveFileSkipDoesNotOverwriteSameName(t *testing.T) {
	installFileAPILog(t)
	root := t.TempDir()
	srcDir := filepath.Join(root, "src")
	dstDir := filepath.Join(root, "dst")
	src := filepath.Join(srcDir, "important.txt")
	dst := filepath.Join(dstDir, "important.txt")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(src, []byte("source-content"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, []byte("destination-content"), 0o644); err != nil {
		t.Fatal(err)
	}

	res := postMoveFile(t, map[string]any{
		"srcPaths":       []string{src},
		"dstPath":        dstDir,
		"isCopy":         false,
		"conflictPolicy": "skip",
	})
	if res.code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.code, res.body)
	}
	if got := int(res.decoded["code"].(float64)); got != 0 {
		t.Fatalf("code=%d body=%s", got, res.body)
	}

	srcData, err := os.ReadFile(src)
	if err != nil || string(srcData) != "source-content" {
		t.Fatalf("source=%q err=%v", srcData, err)
	}
	dstData, err := os.ReadFile(dst)
	if err != nil || string(dstData) != "destination-content" {
		t.Fatalf("destination=%q err=%v", dstData, err)
	}
}

func TestMoveFileOverwriteReplacesSameName(t *testing.T) {
	installFileAPILog(t)
	root := t.TempDir()
	srcDir := filepath.Join(root, "src")
	dstDir := filepath.Join(root, "dst")
	src := filepath.Join(srcDir, "important.txt")
	dst := filepath.Join(dstDir, "important.txt")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(src, []byte("source-content"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, []byte("destination-content"), 0o644); err != nil {
		t.Fatal(err)
	}

	res := postMoveFile(t, map[string]any{
		"srcPaths":       []string{src},
		"dstPath":        dstDir,
		"isCopy":         false,
		"conflictPolicy": "overwrite",
	})
	if res.code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.code, res.body)
	}
	if got := int(res.decoded["code"].(float64)); got != 0 {
		t.Fatalf("code=%d body=%s", got, res.body)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Fatalf("source still exists: %v", err)
	}
	dstData, err := os.ReadFile(dst)
	if err != nil || string(dstData) != "source-content" {
		t.Fatalf("destination=%q err=%v", dstData, err)
	}
}
