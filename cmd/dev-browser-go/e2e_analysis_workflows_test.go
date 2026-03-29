package main

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestCLIInjectAndConsoleWorkflow(t *testing.T) {
	profile := "e2e-inject-console"
	env := newE2EEnv(t)
	bin := buildCLIForE2E(t)
	pageURL := startE2ETestServer(t)

	t.Cleanup(func() {
		_, _ = runCLICommand(t, env, 15*time.Second, bin, "--profile", profile, "stop")
	})

	runCLIJSON(t, env, 45*time.Second, bin,
		"--profile", profile,
		"--output", "json",
		"goto", pageURL,
	)

	styleRes := runCLIJSON(t, env, 15*time.Second, bin,
		"--profile", profile,
		"--output", "json",
		"inject", "--style", "body { background-color: rgb(1, 2, 3); }",
	)
	if !nestedBool(styleRes, "injected", "style") {
		t.Fatalf("style inject result = %#v, want injected.style=true", styleRes)
	}

	bgRes := runCLIJSON(t, env, 15*time.Second, bin,
		"--profile", profile,
		"--output", "json",
		"js-eval", "getComputedStyle(document.body).backgroundColor",
	)
	if got := strings.TrimSpace(asString(bgRes["result"])); got != "rgb(1, 2, 3)" {
		t.Fatalf("backgroundColor = %q, want %q", got, "rgb(1, 2, 3)")
	}

	scriptRes := runCLIJSON(t, env, 15*time.Second, bin,
		"--profile", profile,
		"--output", "json",
		"inject", "--script", `(() => { console.log("e2e-console-marker"); document.body.setAttribute("data-e2e", "ok"); return true; })()`,
	)
	if !nestedBool(scriptRes, "injected", "script") {
		t.Fatalf("script inject result = %#v, want injected.script=true", scriptRes)
	}

	attrRes := runCLIJSON(t, env, 15*time.Second, bin,
		"--profile", profile,
		"--output", "json",
		"js-eval", `document.body.getAttribute("data-e2e")`,
	)
	if got := strings.TrimSpace(asString(attrRes["result"])); got != "ok" {
		t.Fatalf("data-e2e = %q, want %q", got, "ok")
	}

	consoleRes := runCLIJSON(t, env, 15*time.Second, bin,
		"--profile", profile,
		"--output", "json",
		"console", "--level", "all",
	)
	if !consoleEntriesContain(consoleRes["entries"], "e2e-console-marker") {
		t.Fatalf("console output missing marker: %#v", consoleRes)
	}
}

func TestCLIVisualBaselineAndDiffWorkflow(t *testing.T) {
	profile := "e2e-visual"
	env := newE2EEnv(t)
	bin := buildCLIForE2E(t)
	pageURL := startE2ETestServer(t)

	t.Cleanup(func() {
		_, _ = runCLICommand(t, env, 15*time.Second, bin, "--profile", profile, "stop")
	})

	runCLIJSON(t, env, 45*time.Second, bin,
		"--profile", profile,
		"--window-size", "1280x800",
		"--output", "json",
		"goto", pageURL,
	)

	baselineRes := runCLIJSON(t, env, 20*time.Second, bin,
		"--profile", profile,
		"--output", "json",
		"save-baseline", "--path", "baseline.png", "--no-full-page",
	)
	baselinePath := strings.TrimSpace(asString(baselineRes["path"]))
	if baselinePath == "" {
		t.Fatalf("save-baseline result missing path: %#v", baselineRes)
	}
	if _, err := os.Stat(baselinePath); err != nil {
		t.Fatalf("baseline file missing: %v", err)
	}

	runCLIJSON(t, env, 15*time.Second, bin,
		"--profile", profile,
		"--output", "json",
		"inject", "--style", "main { background: rgb(200, 20, 20) !important; } h1 { color: rgb(255, 255, 0) !important; }",
	)

	visualRes := runCLIJSON(t, env, 20*time.Second, bin,
		"--profile", profile,
		"--output", "json",
		"visual-diff", "--baseline", "baseline.png", "--output", "visual-diff.png", "--pixel-threshold", "0", "--tolerance", "0",
	)
	if passed, ok := visualRes["passed"].(bool); !ok || passed {
		t.Fatalf("visual-diff expected passed=false after mutation: %#v", visualRes)
	}
	if diffPixels := asInt(visualRes["different_pixels"]); diffPixels <= 0 {
		t.Fatalf("visual-diff different_pixels = %d, want > 0", diffPixels)
	}
	if out := strings.TrimSpace(asString(visualRes["output_path"])); out == "" {
		t.Fatalf("visual-diff missing output_path: %#v", visualRes)
	}

	diffImagesRes := runCLIJSON(t, env, 20*time.Second, bin,
		"--profile", profile,
		"--output", "json",
		"diff-images", "--before", "baseline.png", "--after", "after.png", "--diff-path", "diff-images.png", "--no-full-page",
	)
	if changed := asInt(diffImagesRes["changed_pixels"]); changed <= 0 {
		t.Fatalf("diff-images changed_pixels = %d, want > 0", changed)
	}
	if match, ok := diffImagesRes["match"].(bool); !ok || match {
		t.Fatalf("diff-images expected match=false after mutation: %#v", diffImagesRes)
	}
	for _, key := range []string{"before_path", "after_path", "diff_path"} {
		path := strings.TrimSpace(asString(diffImagesRes[key]))
		if path == "" {
			t.Fatalf("diff-images missing %s: %#v", key, diffImagesRes)
		}
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("diff-images file %s missing: %v", key, err)
		}
	}
}

func TestCLIDiagnoseAssertAndHTMLValidate(t *testing.T) {
	profile := "e2e-harness"
	env := newE2EEnv(t)
	bin := buildCLIForE2E(t)
	pageURL := startTitledServer(t, "diagnose fixture", `<main id="app-root">fixture</main>`)
	validateURL := startHTMLValidateFixtureServer(t)

	t.Cleanup(func() {
		_, _ = runCLICommand(t, env, 15*time.Second, bin, "--profile", profile, "stop")
	})

	runCLIJSON(t, env, 45*time.Second, bin,
		"--profile", profile,
		"--output", "json",
		"goto", pageURL,
	)
	runCLIJSON(t, env, 15*time.Second, bin,
		"--profile", profile,
		"--output", "json",
		"inject", "--script", `(() => { console.error("diagnose-error-marker"); return true; })()`,
	)

	diagnoseRes := runCLIJSON(t, env, 30*time.Second, bin,
		"--profile", profile,
		"--output", "json",
		"diagnose", "--artifact-mode", "none",
	)
	if !nestedBool(diagnoseRes, "summary", "hasConsoleErrors") {
		t.Fatalf("diagnose summary missing console error signal: %#v", diagnoseRes)
	}
	if !consoleEntriesContain(diagnoseRes["console"].(map[string]any)["entries"], "diagnose-error-marker") {
		t.Fatalf("diagnose console missing marker: %#v", diagnoseRes)
	}

	passRules := `{"selectors":[{"selector":"#app-root","min":1}],"maxConsole":{"error":1}}`
	passRes := runCLIJSON(t, env, 30*time.Second, bin,
		"--profile", profile,
		"--output", "json",
		"assert", "--artifact-mode", "none", "--rules", passRules,
	)
	if passed, ok := passRes["passed"].(bool); !ok || !passed {
		t.Fatalf("assert pass case failed unexpectedly: %#v", passRes)
	}

	failRules := `{"selectors":[{"selector":".missing-selector","min":1}]}`
	failRes, exitCode := runCLIJSONWithExit(t, env, 30*time.Second, bin,
		"--profile", profile,
		"--output", "json",
		"assert", "--artifact-mode", "none", "--rules", failRules,
	)
	if exitCode != 2 {
		t.Fatalf("assert fail exit code = %d, want 2; result=%#v", exitCode, failRes)
	}
	if passed, ok := failRes["passed"].(bool); !ok || passed {
		t.Fatalf("assert fail case passed unexpectedly: %#v", failRes)
	}

	validateRes := runCLIJSON(t, env, 30*time.Second, bin,
		"--profile", profile,
		"--output", "json",
		"html-validate", "--url", validateURL,
	)
	if !findingsContainRule(validateRes["findings"], "duplicate-id") ||
		!findingsContainRule(validateRes["findings"], "img-alt") ||
		!findingsContainRule(validateRes["findings"], "control-name") {
		t.Fatalf("html-validate missing expected findings: %#v", validateRes)
	}
}

func TestCLINetworkMonitorWorkflow(t *testing.T) {
	profile := "e2e-network"
	env := newE2EEnv(t)
	bin := buildCLIForE2E(t)
	pageURL := startNetworkFixtureServer(t)

	t.Cleanup(func() {
		_, _ = runCLICommand(t, env, 15*time.Second, bin, "--profile", profile, "stop")
	})

	allRes := runCLIJSON(t, env, 20*time.Second, bin,
		"--profile", profile,
		"--output", "json",
		"network-monitor", "--url", pageURL+"/burst", "--wait", "load", "--min-wait-ms", "1200", "--url-contains", "/api/", "--status-min", "200", "--status-max", "599",
	)
	if matched := asInt(allRes["matched"]); matched < 2 {
		t.Fatalf("network-monitor matched = %d, want >= 2: %#v", matched, allRes)
	}
	if !networkEntriesContainURL(allRes["entries"], "/api/ok") || !networkEntriesContainURL(allRes["entries"], "/api/fail") {
		t.Fatalf("network-monitor missing expected URLs: %#v", allRes)
	}

	failedRes := runCLIJSON(t, env, 20*time.Second, bin,
		"--profile", profile,
		"--output", "json",
		"network-monitor", "--url", pageURL+"/burst", "--wait", "load", "--min-wait-ms", "1200", "--url-contains", "/api/", "--failed",
	)
	if matched := asInt(failedRes["matched"]); matched < 1 {
		t.Fatalf("network-monitor failed-only matched = %d, want >= 1: %#v", matched, failedRes)
	}
	if !networkEntriesContainURL(failedRes["entries"], "/api/fail") {
		t.Fatalf("failed-only network-monitor missing /api/fail: %#v", failedRes)
	}
	if networkEntriesContainURL(failedRes["entries"], "/api/ok") {
		t.Fatalf("failed-only network-monitor unexpectedly included /api/ok: %#v", failedRes)
	}
}

func TestCLIAssetSnapshotWorkflow(t *testing.T) {
	profile := "e2e-asset-snapshot"
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

	snapshotRes := runCLIJSON(t, env, 20*time.Second, bin,
		"--profile", profile,
		"--output", "json",
		"asset-snapshot", "--path", "offline.html", "--strip-scripts",
	)
	path := strings.TrimSpace(asString(snapshotRes["path"]))
	if path == "" {
		t.Fatalf("asset-snapshot missing path: %#v", snapshotRes)
	}
	if count := asInt(snapshotRes["assets_count"]); count < 2 {
		t.Fatalf("asset-snapshot assets_count = %d, want >= 2: %#v", count, snapshotRes)
	}
	if stripped, ok := snapshotRes["stripped"].(bool); !ok || !stripped {
		t.Fatalf("asset-snapshot stripped flag missing: %#v", snapshotRes)
	}
	html := asString(snapshotRes["html"])
	if !strings.Contains(html, "asset-snapshot-marker") {
		t.Fatalf("asset-snapshot html missing marker: %q", html)
	}
	if !strings.Contains(html, "<!-- <script") {
		t.Fatalf("asset-snapshot html did not strip script tags: %q", html)
	}
	saved, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read asset-snapshot file: %v", err)
	}
	savedHTML := string(saved)
	if !strings.Contains(savedHTML, "asset-snapshot-marker") || !strings.Contains(savedHTML, "<!-- <script") {
		t.Fatalf("saved asset snapshot missing expected content: %q", savedHTML)
	}
}
