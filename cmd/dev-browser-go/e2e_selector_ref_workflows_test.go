package main

import (
	"strings"
	"testing"
	"time"
)

func TestCLISelectorAndRefWorkflow(t *testing.T) {
	profile := "e2e-selector-ref"
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

	snapshotRes := runCLIJSON(t, env, 20*time.Second, bin,
		"--profile", profile,
		"--output", "json",
		"snapshot", "--format", "list", "--max-chars", "4000",
	)
	items := snapshotItems(snapshotRes["items"])
	searchRef := findSnapshotRef(items, "textbox", "Search query")
	buttonRef := findSnapshotRef(items, "button", "Run search")
	if searchRef == "" || buttonRef == "" {
		t.Fatalf("snapshot missing expected refs: %#v", snapshotRes)
	}

	inspectRes := runCLIJSON(t, env, 20*time.Second, bin,
		"--profile", profile,
		"--output", "json",
		"inspect-ref", "--ref", buttonRef, "--style-prop", "display",
	)
	if got := strings.TrimSpace(asString(inspectRes["role"])); got != "button" {
		t.Fatalf("inspect-ref role = %q, want %q", got, "button")
	}
	if got := strings.TrimSpace(asString(inspectRes["name"])); got != "Run search" {
		t.Fatalf("inspect-ref name = %q, want %q", got, "Run search")
	}
	if strings.TrimSpace(asString(inspectRes["selector"])) == "" || strings.TrimSpace(asString(inspectRes["xpath"])) == "" {
		t.Fatalf("inspect-ref missing selector/xpath: %#v", inspectRes)
	}
	bbox, ok := inspectRes["bbox"].(map[string]any)
	if !ok || asFloat64(bbox["width"]) <= 0 || asFloat64(bbox["height"]) <= 0 {
		t.Fatalf("inspect-ref bbox invalid: %#v", inspectRes)
	}

	selectorRes := runCLIJSON(t, env, 20*time.Second, bin,
		"--profile", profile,
		"--output", "json",
		"test-selector", "--selector", "button[type='button']",
	)
	if count := asInt(selectorRes["count"]); count != 1 {
		t.Fatalf("test-selector count = %d, want 1: %#v", count, selectorRes)
	}
	if !previewContainsText(selectorRes["preview"], "Run search") {
		t.Fatalf("test-selector preview missing button text: %#v", selectorRes)
	}

	xpathRes := runCLIJSON(t, env, 20*time.Second, bin,
		"--profile", profile,
		"--output", "json",
		"test-xpath", "--xpath", "//button[@type='button']",
	)
	if count := asInt(xpathRes["count"]); count != 1 {
		t.Fatalf("test-xpath count = %d, want 1: %#v", count, xpathRes)
	}
	if !previewContainsText(xpathRes["preview"], "Run search") {
		t.Fatalf("test-xpath preview missing button text: %#v", xpathRes)
	}

	boundsRes := runCLIJSON(t, env, 20*time.Second, bin,
		"--profile", profile,
		"--output", "json",
		"bounds", "--selector", "#search",
	)
	if asFloat64(boundsRes["width"]) <= 0 || asFloat64(boundsRes["height"]) <= 0 {
		t.Fatalf("bounds invalid: %#v", boundsRes)
	}
}

func TestCLIInteractionWorkflowWithRefs(t *testing.T) {
	profile := "e2e-interaction-ref"
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

	snapshotRes := runCLIJSON(t, env, 20*time.Second, bin,
		"--profile", profile,
		"--output", "json",
		"snapshot", "--format", "list", "--max-chars", "4000",
	)
	items := snapshotItems(snapshotRes["items"])
	searchRef := findSnapshotRef(items, "textbox", "Search query")
	buttonRef := findSnapshotRef(items, "button", "Run search")
	if searchRef == "" || buttonRef == "" {
		t.Fatalf("snapshot missing expected refs: %#v", snapshotRes)
	}

	fillRes := runCLIJSON(t, env, 20*time.Second, bin,
		"--profile", profile,
		"--output", "json",
		"fill-ref", searchRef, "Galaxy S25",
	)
	if filled, ok := fillRes["filled"].(bool); !ok || !filled {
		t.Fatalf("fill-ref failed: %#v", fillRes)
	}

	pressRes := runCLIJSON(t, env, 20*time.Second, bin,
		"--profile", profile,
		"--output", "json",
		"press", "Enter",
	)
	if pressed, ok := pressRes["pressed"].(bool); !ok || !pressed {
		t.Fatalf("press failed: %#v", pressRes)
	}

	waitRes := runCLIJSON(t, env, 20*time.Second, bin,
		"--profile", profile,
		"--output", "json",
		"wait", "--strategy", "perf", "--state", "load", "--min-wait-ms", "250",
	)
	if waited := asInt(waitRes["waited_ms"]); waited < 200 {
		t.Fatalf("wait waited_ms = %d, want >= 200: %#v", waited, waitRes)
	}

	enterResult := runCLIJSON(t, env, 15*time.Second, bin,
		"--profile", profile,
		"--output", "json",
		"js-eval", `document.getElementById("result").textContent.trim()`,
	)
	if got := strings.TrimSpace(asString(enterResult["result"])); got != "Galaxy S25 via enter" {
		t.Fatalf("enter interaction result = %q, want %q", got, "Galaxy S25 via enter")
	}

	clickRes := runCLIJSON(t, env, 20*time.Second, bin,
		"--profile", profile,
		"--output", "json",
		"click-ref", buttonRef,
	)
	if clicked, ok := clickRes["clicked"].(bool); !ok || !clicked {
		t.Fatalf("click-ref failed: %#v", clickRes)
	}

	runCLIJSON(t, env, 20*time.Second, bin,
		"--profile", profile,
		"--output", "json",
		"wait", "--strategy", "perf", "--state", "load", "--min-wait-ms", "250",
	)

	clickResult := runCLIJSON(t, env, 15*time.Second, bin,
		"--profile", profile,
		"--output", "json",
		"js-eval", `document.getElementById("result").textContent.trim()`,
	)
	if got := strings.TrimSpace(asString(clickResult["result"])); got != "Galaxy S25 via click" {
		t.Fatalf("click interaction result = %q, want %q", got, "Galaxy S25 via click")
	}
}
