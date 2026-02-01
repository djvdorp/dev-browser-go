package devbrowser

import "testing"

func TestDiagnoseSummary_HarnessOverlay(t *testing.T) {
	r := &DiagnoseReport{}
	r.Console.Counts = DiagnoseConsoleCounts{Error: 0}
	r.Network.Entries = nil
	r.Harness.State = map[string]any{
		"errors": []interface{}{
			map[string]any{"time_ms": 1.0, "message": "boom"},
		},
		"overlays": []interface{}{
			map[string]any{"time_ms": 2.0, "type": "vite", "text": "Vite error overlay\n  at App"},
		},
	}

	r.computeSummary()

	if !r.Summary.HasHarnessErrors {
		t.Fatalf("expected HasHarnessErrors")
	}
	if r.Summary.HarnessErrorCount != 1 {
		t.Fatalf("expected HarnessErrorCount=1, got %d", r.Summary.HarnessErrorCount)
	}
	if !r.Summary.HasViteOverlay {
		t.Fatalf("expected HasViteOverlay")
	}
	if r.Summary.ViteOverlayText == "" {
		t.Fatalf("expected ViteOverlayText to be populated")
	}
}

func TestDiagnoseSummary_NoHarness(t *testing.T) {
	r := &DiagnoseReport{}
	r.Console.Counts = DiagnoseConsoleCounts{Error: 0}
	r.Network.Entries = nil
	// Harness.State left nil

	r.computeSummary()

	if r.Summary.HasHarnessErrors {
		t.Fatalf("expected HasHarnessErrors=false")
	}
	if r.Summary.HarnessErrorCount != 0 {
		t.Fatalf("expected HarnessErrorCount=0")
	}
	if r.Summary.HasViteOverlay {
		t.Fatalf("expected HasViteOverlay=false")
	}
	if r.Summary.ViteOverlayText != "" {
		t.Fatalf("expected ViteOverlayText empty")
	}
}

func TestDiagnoseSummary_ViteOverlayTextClamped(t *testing.T) {
	long := make([]byte, 2000)
	for i := range long {
		long[i] = 'a'
	}

	r := &DiagnoseReport{}
	r.Console.Counts = DiagnoseConsoleCounts{Error: 0}
	r.Network.Entries = nil
	r.Harness.State = map[string]any{
		"errors": []interface{}{},
		"overlays": []interface{}{
			map[string]any{"time_ms": 2.0, "type": "vite", "text": string(long)},
		},
	}

	r.computeSummary()

	if !r.Summary.HasViteOverlay {
		t.Fatalf("expected HasViteOverlay")
	}
	if len(r.Summary.ViteOverlayText) == 0 {
		t.Fatalf("expected ViteOverlayText")
	}
	if len(r.Summary.ViteOverlayText) > 800 {
		t.Fatalf("expected clamped ViteOverlayText <= 800 chars, got %d", len(r.Summary.ViteOverlayText))
	}
}
