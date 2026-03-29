package main

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestCLIPerfMetricsWorkflow(t *testing.T) {
	profile := "e2e-perf"
	env := newE2EEnv(t)
	bin := buildCLIForE2E(t)
	pageURL := startAssetSnapshotFixtureServer(t)

	t.Cleanup(func() {
		_, _ = runCLICommand(t, env, 15*time.Second, bin, "--profile", profile, "stop")
	})

	runCLIJSON(t, env, 45*time.Second, bin,
		"--profile", profile,
		"--output", "json",
		"goto", pageURL,
	)

	perfRes := runCLIJSON(t, env, 20*time.Second, bin,
		"--profile", profile,
		"--output", "json",
		"perf-metrics", "--sample-ms", "300", "--top-n", "5",
	)
	if got := strings.TrimSpace(asString(perfRes["url"])); got != pageURL+"/" {
		t.Fatalf("perf-metrics url = %q, want %q", got, pageURL+"/")
	}
	timing, ok := perfRes["timing"].(map[string]any)
	if !ok || timing["navigation"] == nil {
		t.Fatalf("perf-metrics timing missing navigation: %#v", perfRes)
	}
	fps, ok := perfRes["fps"].(map[string]any)
	if !ok {
		t.Fatalf("perf-metrics fps missing: %#v", perfRes)
	}
	if sampleMs := asInt(fps["sampleMs"]); sampleMs != 300 {
		t.Fatalf("perf-metrics fps.sampleMs = %d, want 300: %#v", sampleMs, perfRes)
	}
	if frames := asInt(fps["frames"]); frames <= 0 {
		t.Fatalf("perf-metrics fps.frames = %d, want > 0: %#v", frames, perfRes)
	}
	resources, ok := perfRes["resources"].(map[string]any)
	if !ok {
		t.Fatalf("perf-metrics resources missing: %#v", perfRes)
	}
	if total := asInt(resources["total"]); total < 2 {
		t.Fatalf("perf-metrics resources.total = %d, want >= 2: %#v", total, perfRes)
	}
	top, ok := resources["top"].([]any)
	if !ok || len(top) == 0 {
		t.Fatalf("perf-metrics resources.top missing: %#v", perfRes)
	}
}

func TestCLIDomBaselineAndDiffWorkflow(t *testing.T) {
	profile := "e2e-dom-diff"
	env := newE2EEnv(t)
	bin := buildCLIForE2E(t)
	pageURL := startInteractiveRefsFixtureServer(t)

	t.Cleanup(func() {
		_, _ = runCLICommand(t, env, 15*time.Second, bin, "--profile", profile, "stop")
	})

	runCLIJSON(t, env, 45*time.Second, bin,
		"--profile", profile,
		"--output", "json",
		"goto", pageURL,
	)

	baselineRes := runCLIJSON(t, env, 20*time.Second, bin,
		"--profile", profile,
		"--output", "json",
		"save-dom-baseline", "--path", "dom-baseline.json",
	)
	baselinePath := strings.TrimSpace(asString(baselineRes["path"]))
	if baselinePath == "" {
		t.Fatalf("save-dom-baseline missing path: %#v", baselineRes)
	}
	if _, err := os.Stat(baselinePath); err != nil {
		t.Fatalf("dom baseline file missing: %v", err)
	}

	runCLIJSON(t, env, 15*time.Second, bin,
		"--profile", profile,
		"--output", "json",
		"inject", "--script", `(() => { document.getElementById("run-search").disabled = true; return true; })()`,
	)

	diffRes := runCLIJSON(t, env, 20*time.Second, bin,
		"--profile", profile,
		"--output", "json",
		"dom-diff", "--baseline", "dom-baseline.json",
	)
	if got := strings.TrimSpace(asString(diffRes["baseline_path"])); got == "" {
		t.Fatalf("dom-diff missing baseline_path: %#v", diffRes)
	}
	if changed := asInt(diffRes["changed_count"]); changed < 1 {
		t.Fatalf("dom-diff changed_count = %d, want >= 1: %#v", changed, diffRes)
	}
	changedItems, ok := diffRes["changed"].([]any)
	if !ok || len(changedItems) == 0 {
		t.Fatalf("dom-diff changed entries missing: %#v", diffRes)
	}
	firstChanged, ok := changedItems[0].(map[string]any)
	if !ok {
		t.Fatalf("dom-diff changed entry malformed: %#v", diffRes)
	}
	if !stringSliceContains(firstChanged["fields"], "disabled") {
		t.Fatalf("dom-diff changed fields missing disabled: %#v", diffRes)
	}
}
