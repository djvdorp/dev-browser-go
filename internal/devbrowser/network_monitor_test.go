package devbrowser

import "testing"

func TestMatchNetworkFilters(t *testing.T) {
	entry := NetworkEntry{URL: "https://example.com/api", Method: "GET", Type: "xhr", Status: 200, OK: true}
	if !matchNetwork(entry, NetworkMonitorOptions{URLContains: "example.com"}) {
		t.Fatal("expected url_contains match")
	}
	if matchNetwork(entry, NetworkMonitorOptions{URLContains: "nope"}) {
		t.Fatal("expected url_contains filter")
	}
	if matchNetwork(entry, NetworkMonitorOptions{MethodEquals: "POST"}) {
		t.Fatal("expected method filter")
	}
	if !matchNetwork(entry, NetworkMonitorOptions{StatusMin: 200, StatusMax: 299}) {
		t.Fatal("expected status range match")
	}
	if matchNetwork(entry, NetworkMonitorOptions{StatusEquals: 404}) {
		t.Fatal("expected status equals filter")
	}
	if matchNetwork(entry, NetworkMonitorOptions{OnlyFailed: true}) {
		t.Fatal("expected only_failed to drop OK requests")
	}
}

func TestLooksBinary(t *testing.T) {
	if !looksBinary([]byte{0, 1, 2}) {
		t.Fatal("expected NUL to be binary")
	}
	if looksBinary([]byte("hello world")) {
		t.Fatal("expected text not binary")
	}
}
