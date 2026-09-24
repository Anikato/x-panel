package v1

import (
	"net/http"
	"testing"
)

func TestTerminalOriginAllowedForSameHostAndNonBrowser(t *testing.T) {
	same := &http.Request{Host: "panel.example:7777", Header: http.Header{"Origin": []string{"https://panel.example:7777"}}}
	if !terminalOriginAllowed(same) {
		t.Fatal("same-host browser origin should be allowed")
	}
	other := &http.Request{Host: "panel.example:7777", Header: http.Header{"Origin": []string{"https://evil.example"}}}
	if terminalOriginAllowed(other) {
		t.Fatal("cross-origin browser request should be rejected")
	}
	plain := &http.Request{Host: "panel.example:7777", Header: http.Header{}}
	if !terminalOriginAllowed(plain) {
		t.Fatal("non-browser client without Origin should be allowed")
	}
}
