package devbrowser

import (
	"strings"
	"testing"
)

func TestStartDaemonRejectsDeviceAndWindow(t *testing.T) {
	profile := "test-device-conflict"
	_, _ = StopDaemon(profile)

	window := &WindowSize{Width: 100, Height: 200}
	err := StartDaemon(profile, true, window, "Pixel 5")
	if err == nil {
		t.Fatalf("expected error for device + window")
	}
	if !strings.Contains(err.Error(), "use either") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWriteOutputHTML(t *testing.T) {
	out, err := WriteOutput("test-profile", "html", map[string]any{"html": "<html></html>"}, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "<html></html>" {
		t.Fatalf("expected html output, got %q", out)
	}
}

func TestWriteOutputHTMLMissing(t *testing.T) {
	_, err := WriteOutput("test-profile", "html", map[string]any{}, "")
	if err == nil {
		t.Fatalf("expected error for missing html output")
	}
	if !strings.Contains(err.Error(), "html output not available") {
		t.Fatalf("unexpected error: %v", err)
	}
}
