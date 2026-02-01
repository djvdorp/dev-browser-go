package devbrowser

import (
	"strings"
	"testing"
)

func TestDaemonVersion(t *testing.T) {
	v := DaemonVersion()

	// Should not be empty
	if v == "" {
		t.Fatal("DaemonVersion() returned empty string")
	}

	// Should have the expected prefix
	expectedPrefix := "go-dev-browser-daemon/"
	if !strings.HasPrefix(v, expectedPrefix) {
		t.Fatalf("DaemonVersion() = %q, want prefix %q", v, expectedPrefix)
	}

	// Should have a hash portion after the prefix (at least some hex chars)
	hashPart := strings.TrimPrefix(v, expectedPrefix)
	if len(hashPart) == 0 {
		t.Fatal("DaemonVersion() missing hash portion after prefix")
	}

	// Should be deterministic - calling it multiple times should return the same value
	v2 := DaemonVersion()
	if v != v2 {
		t.Fatalf("DaemonVersion() not deterministic: first call = %q, second call = %q", v, v2)
	}
}
