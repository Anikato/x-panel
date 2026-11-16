package iplocation

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

type databaseDownloadTransport func(*http.Request) (*http.Response, error)

func (f databaseDownloadTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestInvalidDownloadDoesNotReplaceIPDatabase(t *testing.T) {
	dataDir := t.TempDir()
	service := &Service{}
	service.Init(dataDir)
	path := filepath.Join(dataDir, "GeoLite2-City.mmdb")
	const oldDatabase = "previous database contents"
	if err := os.WriteFile(path, []byte(oldDatabase), 0600); err != nil {
		t.Fatal(err)
	}
	var payload bytes.Buffer
	gz := gzip.NewWriter(&payload)
	if _, err := gz.Write([]byte("a valid gzip that does not contain an MMDB")); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}

	client := &http.Client{Transport: databaseDownloadTransport(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(&payload)}, nil
	})}
	if err := service.DownloadDB(context.Background(), client); err == nil {
		t.Fatal("invalid MMDB was accepted")
	}
	content, err := os.ReadFile(path)
	if err != nil || string(content) != oldDatabase {
		t.Fatalf("previous file was changed: %v", err)
	}
	files, err := os.ReadDir(dataDir)
	if err != nil || len(files) != 1 {
		t.Fatalf("temporary download was not cleaned up: %v %v", files, err)
	}
}
