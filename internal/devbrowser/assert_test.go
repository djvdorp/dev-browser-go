package devbrowser

import "testing"

func intPtr(v int) *int           { return &v }
func floatPtr(v float64) *float64 { return &v }

func TestParseAssertRules_Basic(t *testing.T) {
	raw := `{
  "maxConsole": {"error": 0},
  "network": {"maxFailed": 0, "maxStatus": {"min": 400, "count": 0}},
  "selectors": [{"selector": ".error", "max": 0}, {"selector": "[data-testid='app-root']", "min": 1}],
  "perf": {"lcpMaxMs": 2500, "clsMax": 0.1},
  "harness": {"maxErrors": 0, "maxOverlays": 0}
}`
	rules, err := ParseAssertRules(raw)
	if err != nil {
		t.Fatalf("ParseAssertRules error: %v", err)
	}
	if rules.MaxConsole["error"] != 0 {
		t.Fatalf("expected maxConsole.error=0")
	}
	if rules.Network == nil || rules.Network.MaxFailed != 0 {
		t.Fatalf("expected network.maxFailed=0")
	}
	if len(rules.Selectors) != 2 {
		t.Fatalf("expected 2 selectors")
	}
	if rules.Perf == nil || rules.Perf.CLSMax == nil {
		t.Fatalf("expected perf fields")
	}
}

func TestEvaluateAssert_Failures(t *testing.T) {
	report := &DiagnoseReport{}
	report.Console.Counts = DiagnoseConsoleCounts{Error: 2, Warning: 0, Info: 0}
	report.Network.Entries = []NetworkEntry{{URL: "https://x", Method: "GET", Status: 500, OK: false}}
	report.Perf = map[string]any{"cwv": map[string]any{"lcp": 3000.0, "cls": 0.25}}
	report.Harness.State = map[string]any{
		"errors":   []interface{}{map[string]any{"time_ms": 1.0, "message": "boom"}},
		"overlays": []interface{}{map[string]any{"time_ms": 2.0, "text": "vite overlay"}},
	}

	rules := &AssertRules{
		MaxConsole: map[string]int{"error": 0},
		Network:    &AssertNetwork{MaxFailed: 0, MaxStatus: &AssertStatusCount{Min: 400, Count: 0}},
		Selectors:  []AssertSelector{{Selector: ".error", Max: intPtr(0)}},
		Perf:       &AssertPerf{LCPMaxMs: floatPtr(2500), CLSMax: floatPtr(0.1)},
		Harness:    &AssertHarness{MaxErrors: intPtr(0), MaxOverlays: intPtr(0)},
	}

	selectorCounts := map[string]int{".error": 1}
	res := EvaluateAssert(report, rules, selectorCounts, nil)
	if res.Passed {
		t.Fatalf("expected failed")
	}
	if len(res.FailedChecks) == 0 {
		t.Fatalf("expected failed checks")
	}
}

func TestEvaluateAssert_Pass(t *testing.T) {
	report := &DiagnoseReport{}
	report.Console.Counts = DiagnoseConsoleCounts{Error: 0, Warning: 0, Info: 0}
	report.Network.Entries = []NetworkEntry{{URL: "https://x", Method: "GET", Status: 200, OK: true}}
	report.Perf = map[string]any{"cwv": map[string]any{"lcp": 1200.0, "cls": 0.01}}
	report.Harness.State = map[string]any{"errors": []interface{}{}, "overlays": []interface{}{}}

	rules := &AssertRules{
		MaxConsole: map[string]int{"error": 0},
		Network:    &AssertNetwork{MaxFailed: 0, MaxStatus: &AssertStatusCount{Min: 400, Count: 0}},
		Selectors:  []AssertSelector{{Selector: "[data-testid='app-root']", Min: intPtr(1)}},
		Perf:       &AssertPerf{LCPMaxMs: floatPtr(2500), CLSMax: floatPtr(0.1)},
		Harness:    &AssertHarness{MaxErrors: intPtr(0), MaxOverlays: intPtr(0)},
	}

	selectorCounts := map[string]int{"[data-testid='app-root']": 1}
	res := EvaluateAssert(report, rules, selectorCounts, nil)
	if !res.Passed {
		t.Fatalf("expected passed; failedChecks=%v", res.FailedChecks)
	}
}
