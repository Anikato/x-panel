package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"xpanel/app/dto"
	"xpanel/global"

	"github.com/sirupsen/logrus"
)

func TestWgetStopsWhenBodyExceedsDestinationBudget(t *testing.T) {
	previousBudget, previousLog := downloadBudget, global.LOG
	downloadBudget = func(string) (int64, error) { return 4, nil }
	global.LOG = logrus.New()
	t.Cleanup(func() {
		downloadBudget = previousBudget
		global.LOG = previousLog
	})

	dir := t.TempDir()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("0123456789"))
	}))
	defer server.Close()

	err := (&FileService{}).WgetWithTracker(context.Background(), dto.FileWgetReq{
		URL:  server.URL + "/payload.bin",
		Path: dir,
	}, nil)
	if err == nil {
		t.Fatal("expected download to stop at the destination budget")
	}
	entries, readErr := os.ReadDir(dir)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if len(entries) != 0 {
		t.Fatalf("download left %d files behind", len(entries))
	}
}
