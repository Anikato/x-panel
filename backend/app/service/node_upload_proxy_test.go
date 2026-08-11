package service

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"xpanel/app/model"
)

func TestDoProxyRequestForwardsMultipartContentType(t *testing.T) {
	const contentType = "multipart/form-data; boundary=xpanel-test"
	client := &http.Client{Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		if got := r.Header.Get("Content-Type"); got != contentType {
			t.Errorf("content type = %q", got)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		if string(body) != "multipart-body" {
			t.Errorf("body = %q", body)
		}
		return &http.Response{
			StatusCode: http.StatusCreated,
			Body:       io.NopCloser(strings.NewReader(`{"code":0}`)),
			Header:     make(http.Header),
		}, nil
	})}

	node := &model.Node{Address: "http://node.test", Token: "agent-token"}
	data, status, err := (&NodeService{}).doNodeRequest(
		context.Background(),
		node,
		http.MethodPost,
		"/api/v1/files/upload",
		contentType,
		strings.NewReader("multipart-body"),
		client,
	)
	if err != nil {
		t.Fatal(err)
	}
	if status != http.StatusCreated || string(data) != `{"code":0}` {
		t.Fatalf("status=%d data=%q", status, data)
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}
