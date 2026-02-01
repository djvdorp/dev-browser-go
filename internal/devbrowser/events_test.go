package devbrowser

import (
	"strconv"
	"strings"
	"testing"
)

func TestBuildDiagnoseEvents_Ordering(t *testing.T) {
	tests := []struct {
		name    string
		console []ConsoleEntry
		network []NetworkEntry
		harness map[string]any
		want    []string // Kind:TimeMS for checking order
	}{
		{
			name: "basic time ordering",
			console: []ConsoleEntry{
				{TimeMS: 300, Type: "log", Text: "third"},
				{TimeMS: 100, Type: "log", Text: "first"},
				{TimeMS: 200, Type: "log", Text: "second"},
			},
			want: []string{"console:100", "console:200", "console:300"},
		},
		{
			name: "mixed sources time ordering",
			console: []ConsoleEntry{
				{TimeMS: 200, Type: "log", Text: "console2"},
			},
			network: []NetworkEntry{
				{Started: 100, URL: "http://a.com"},
				{Started: 300, URL: "http://b.com"},
			},
			want: []string{"network:100", "console:200", "network:300"},
		},
		{
			name: "same timestamp different kinds",
			console: []ConsoleEntry{
				{TimeMS: 100, Type: "log", Text: "console"},
			},
			network: []NetworkEntry{
				{Started: 100, URL: "http://a.com"},
			},
			harness: map[string]any{
				"errors": []interface{}{
					map[string]any{"time_ms": 100.0, "message": "error"},
				},
			},
			// console < errorhook < network < overlay (alphabetical)
			want: []string{"console:100", "errorhook:100", "network:100"},
		},
		{
			name: "same timestamp same kind different data",
			console: []ConsoleEntry{
				{TimeMS: 100, Type: "log", Text: "zebra", ID: 2},
				{TimeMS: 100, Type: "log", Text: "apple", ID: 1},
			},
			// Deterministic ordering by stableCompareData: compares by id (1<2), then text
			want: []string{"console:100", "console:100"},
		},
		{
			name: "same timestamp same kind - id ordering",
			console: []ConsoleEntry{
				{TimeMS: 100, Type: "log", Text: "same", ID: 3},
				{TimeMS: 100, Type: "log", Text: "same", ID: 1},
				{TimeMS: 100, Type: "log", Text: "same", ID: 2},
			},
			want: []string{"console:100", "console:100", "console:100"},
		},
		{
			name: "harness events with missing time_ms are skipped",
			harness: map[string]any{
				"errors": []interface{}{
					map[string]any{"message": "no time"},
					map[string]any{"time_ms": 100.0, "message": "has time"},
				},
			},
			want: []string{"errorhook:100"},
		},
		{
			name: "overlay events",
			harness: map[string]any{
				"overlays": []interface{}{
					map[string]any{"time_ms": 200.0, "text": "overlay2"},
					map[string]any{"time_ms": 100.0, "text": "overlay1"},
				},
			},
			want: []string{"overlay:100", "overlay:200"},
		},
		{
			name: "invalid harness entries ignored",
			harness: map[string]any{
				"errors": []interface{}{
					"not a map",
					map[string]any{"time_ms": 100.0, "message": "valid"},
					42,
				},
			},
			want: []string{"errorhook:100"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildDiagnoseEvents(tt.console, tt.network, tt.harness)
			if len(got) != len(tt.want) {
				t.Errorf("BuildDiagnoseEvents() length = %d, want %d", len(got), len(tt.want))
				return
			}
			for i, ev := range got {
				want := tt.want[i]
				// Parse expected "kind:timeMS"
				parts := strings.Split(want, ":")
				if len(parts) != 2 {
					t.Fatalf("invalid test want format: %s", want)
				}
				wantKind := parts[0]
				wantTimeMS, err := strconv.ParseInt(parts[1], 10, 64)
				if err != nil {
					t.Fatalf("invalid test want timeMS: %s", want)
				}
				if ev.Kind != wantKind || ev.TimeMS != wantTimeMS {
					t.Errorf("event[%d] = %s:%d, want %s", i, ev.Kind, ev.TimeMS, want)
				}
			}

			// Special verification for id-ordering test
			if tt.name == "same timestamp same kind - id ordering" {
				// Verify IDs are in ascending order
				for i := 1; i < len(got); i++ {
					prevID := got[i-1].Data["id"].(int64)
					currID := got[i].Data["id"].(int64)
					if prevID >= currID {
						t.Errorf("IDs not in order: event[%d].id=%d >= event[%d].id=%d", i-1, prevID, i, currID)
					}
				}
			}
		})
	}
}

func TestBuildDiagnoseEvents_Deterministic(t *testing.T) {
	// Run the same input multiple times to verify deterministic output
	console := []ConsoleEntry{
		{TimeMS: 100, Type: "log", Text: "a", ID: 1},
		{TimeMS: 100, Type: "log", Text: "b", ID: 2},
		{TimeMS: 100, Type: "warn", Text: "c", ID: 3},
	}
	network := []NetworkEntry{
		{Started: 100, URL: "http://a.com", Method: "GET"},
		{Started: 100, URL: "http://b.com", Method: "POST"},
	}
	harness := map[string]any{
		"errors": []interface{}{
			map[string]any{"time_ms": 100.0, "message": "err1"},
			map[string]any{"time_ms": 100.0, "message": "err2"},
		},
	}

	var prev []DiagnoseEvent
	for i := 0; i < 5; i++ {
		got := BuildDiagnoseEvents(console, network, harness)
		if i == 0 {
			prev = got
			continue
		}
		// Check that order is identical across runs
		if len(got) != len(prev) {
			t.Fatalf("run %d: length changed from %d to %d", i, len(prev), len(got))
		}
		for j := range got {
			if got[j].Kind != prev[j].Kind || got[j].TimeMS != prev[j].TimeMS {
				t.Errorf("run %d: event[%d] changed from %s:%d to %s:%d",
					i, j, prev[j].Kind, prev[j].TimeMS, got[j].Kind, got[j].TimeMS)
			}
		}
	}
}

func TestBuildDiagnoseEvents_EmptyInputs(t *testing.T) {
	tests := []struct {
		name    string
		console []ConsoleEntry
		network []NetworkEntry
		harness map[string]any
	}{
		{"all empty", nil, nil, nil},
		{"nil harness", []ConsoleEntry{{TimeMS: 100}}, nil, nil},
		{"empty arrays", []ConsoleEntry{}, []NetworkEntry{}, map[string]any{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildDiagnoseEvents(tt.console, tt.network, tt.harness)
			if got == nil {
				t.Error("BuildDiagnoseEvents() returned nil, want empty slice")
			}
		})
	}
}
