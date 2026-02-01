package devbrowser

import (
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultRunArtifactDir_Format(t *testing.T) {
	root := filepath.Join("/tmp", "artifacts")
	ts := time.Date(2026, 2, 1, 12, 34, 56, 789, time.FixedZone("X", 3600))
	runID := "12345678-aaaa-bbbb-cccc-ddddeeeeffff"
	got := DefaultRunArtifactDir(root, runID, ts)
	want := filepath.Join(root, "run-20260201T113456Z-12345678")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestTruncateStringRunes(t *testing.T) {
	body, truncated := TruncateStringRunes("hello", 5)
	if body != "hello" || truncated {
		t.Fatalf("expected no truncation, got %q truncated=%v", body, truncated)
	}

	body, truncated = TruncateStringRunes("hello", 3)
	if body != "hel" || !truncated {
		t.Fatalf("expected truncation, got %q truncated=%v", body, truncated)
	}
}
