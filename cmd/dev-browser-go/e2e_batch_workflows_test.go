package main

import (
	"strings"
	"testing"
	"time"
)

func TestCLICallWorkflow(t *testing.T) {
	profile := "e2e-call"
	env := newE2EEnv(t)
	bin := buildCLIForE2E(t)
	pageURL := startTitledServer(t, "call fixture", `<main id="call-root">call fixture body</main>`)

	t.Cleanup(func() {
		_, _ = runCLICommand(t, env, 15*time.Second, bin, "--profile", profile, "stop")
	})

	gotoRes := runCLIJSON(t, env, 45*time.Second, bin,
		"--profile", profile,
		"--output", "json",
		"call", "goto", "--args", `{"url":"`+pageURL+`","timeout_ms":45000}`,
	)
	if got := strings.TrimSpace(asString(gotoRes["title"])); got != "call fixture" {
		t.Fatalf("call goto title = %q, want %q", got, "call fixture")
	}

	snapshotRes := runCLIJSON(t, env, 20*time.Second, bin,
		"--profile", profile,
		"--output", "json",
		"call", "snapshot", "--args", `{"format":"list","interactive_only":false,"max_chars":4000}`,
	)
	snapshotText := asString(snapshotRes["snapshot"])
	if !strings.Contains(snapshotText, "call fixture body") {
		t.Fatalf("call snapshot missing expected body text: %#v", snapshotRes)
	}

	evalRes := runCLIJSON(t, env, 15*time.Second, bin,
		"--profile", profile,
		"--output", "json",
		"call", "js_eval", "--args", `{"expression":"document.getElementById('call-root').textContent.trim()"}`,
	)
	if got := strings.TrimSpace(asString(evalRes["result"])); got != "call fixture body" {
		t.Fatalf("call js_eval result = %q, want %q", got, "call fixture body")
	}
}

func TestCLIActionsWorkflowFromStdin(t *testing.T) {
	profile := "e2e-actions"
	env := newE2EEnv(t)
	bin := buildCLIForE2E(t)
	pageURL := startTitledServer(t, "actions fixture", `<main id="actions-root">actions fixture body</main>`)

	t.Cleanup(func() {
		_, _ = runCLICommand(t, env, 15*time.Second, bin, "--profile", profile, "stop")
	})

	callsJSON := `[
		{"name":"goto","arguments":{"url":"` + pageURL + `","timeout_ms":45000}},
		{"name":"inject","arguments":{"script":"(() => { document.body.setAttribute('data-actions','ok'); return true; })()","wait_ms":50}},
		{"name":"js_eval","arguments":{"expression":"document.body.getAttribute('data-actions')"}},
		{"name":"snapshot","arguments":{"format":"list","interactive_only":false,"max_chars":4000}}
	]`

	actionsRes := runCLIJSONWithInput(t, env, 45*time.Second, bin, callsJSON,
		"--profile", profile,
		"--output", "json",
		"actions",
	)

	results, ok := actionsRes["results"].([]any)
	if !ok || len(results) != 4 {
		t.Fatalf("actions results length = %d, want 4: %#v", len(results), actionsRes)
	}
	third, ok := results[2].(map[string]any)
	if !ok {
		t.Fatalf("actions third result malformed: %#v", actionsRes)
	}
	if got := strings.TrimSpace(asString(third["name"])); got != "js_eval" {
		t.Fatalf("actions third result name = %q, want %q", got, "js_eval")
	}
	thirdResult, ok := third["result"].(map[string]any)
	if !ok {
		t.Fatalf("actions third nested result malformed: %#v", third)
	}
	if got := strings.TrimSpace(asString(thirdResult["result"])); got != "ok" {
		t.Fatalf("actions js_eval result = %q, want %q", got, "ok")
	}

	snapshotText := asString(actionsRes["snapshot"])
	if !strings.Contains(snapshotText, "actions fixture body") {
		t.Fatalf("actions snapshot missing fixture text: %#v", actionsRes)
	}
}
