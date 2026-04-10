package main

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestCLIWorkflowPersistsDaemonAndCapturesDesktopScreenshot(t *testing.T) {
	profile := "e2e-desktop"
	env := newE2EEnv(t)
	bin := buildCLIForE2E(t)
	pageURL := startE2ETestServer(t)

	t.Cleanup(func() {
		_, _ = runCLICommand(t, env, 15*time.Second, bin, "--profile", profile, "stop")
	})

	gotoRes := runCLIJSON(t, env, 60*time.Second, bin,
		"--profile", profile,
		"--window-size", "1280x800",
		"--output", "json",
		"goto", pageURL,
	)
	if got := strings.TrimSpace(asString(gotoRes["title"])); got != "dev-browser-go e2e" {
		t.Fatalf("goto title = %q, want %q", got, "dev-browser-go e2e")
	}

	statusOut, _ := runCLICommand(t, env, 15*time.Second, bin,
		"--profile", profile,
		"status",
	)
	if !strings.Contains(statusOut, "ok profile="+profile) {
		t.Fatalf("status output = %q, want healthy daemon", statusOut)
	}

	shotRes := runCLIJSON(t, env, 30*time.Second, bin,
		"--profile", profile,
		"--output", "json",
		"screenshot", "--path", "desktop-shot.png", "--no-full-page",
	)
	shotPath := strings.TrimSpace(asString(shotRes["path"]))
	if shotPath == "" {
		t.Fatalf("screenshot result missing path: %#v", shotRes)
	}
	assertScreenshotLooksReal(t, shotPath, 1280)
}

func TestCLIWorkflowPersistsDaemonAndCapturesMobileScreenshot(t *testing.T) {
	profile := "e2e-mobile"
	env := newE2EEnv(t)
	bin := buildCLIForE2E(t)
	pageURL := startE2ETestServer(t)

	t.Cleanup(func() {
		_, _ = runCLICommand(t, env, 15*time.Second, bin, "--profile", profile, "stop")
	})

	gotoRes := runCLIJSON(t, env, 60*time.Second, bin,
		"--profile", profile,
		"--device", "Galaxy S9+",
		"--output", "json",
		"goto", pageURL,
	)
	if got := strings.TrimSpace(asString(gotoRes["title"])); got != "dev-browser-go e2e" {
		t.Fatalf("goto title = %q, want %q", got, "dev-browser-go e2e")
	}

	statusOut, _ := runCLICommand(t, env, 15*time.Second, bin,
		"--profile", profile,
		"status",
	)
	if !strings.Contains(statusOut, "ok profile="+profile) {
		t.Fatalf("status output = %q, want healthy daemon", statusOut)
	}

	shotRes := runCLIJSON(t, env, 30*time.Second, bin,
		"--profile", profile,
		"--output", "json",
		"screenshot", "--path", "mobile-shot.png", "--no-full-page",
	)
	shotPath := strings.TrimSpace(asString(shotRes["path"]))
	if shotPath == "" {
		t.Fatalf("screenshot result missing path: %#v", shotRes)
	}
	width, _ := assertScreenshotLooksReal(t, shotPath, 0)
	if width > 1500 {
		t.Fatalf("mobile screenshot width = %d, want <= 1500", width)
	}
}

func TestCLIWorkflowSaveHTMLReconnectCapturesDelayedBody(t *testing.T) {
	profile := "e2e-save-html"
	env := newE2EEnv(t)
	bin := buildCLIForE2E(t)
	pageURL := startDelayedBodyServer(t, 800*time.Millisecond)

	t.Cleanup(func() {
		_, _ = runCLICommand(t, env, 15*time.Second, bin, "--profile", profile, "stop")
	})

	gotoRes := runCLIJSON(t, env, 60*time.Second, bin,
		"--profile", profile,
		"--output", "json",
		"goto", pageURL,
	)
	if got := strings.TrimSpace(asString(gotoRes["title"])); got != "dev-browser-go delayed body" {
		t.Fatalf("goto title = %q, want %q", got, "dev-browser-go delayed body")
	}

	saveRes := runCLIJSON(t, env, 30*time.Second, bin,
		"--profile", profile,
		"--output", "json",
		"save-html", "--path", "delayed-body.html",
	)
	if got := asString(saveRes["title"]); got != "dev-browser-go delayed body" {
		t.Fatalf("save-html title = %q, want %q", got, "dev-browser-go delayed body")
	}
	html := asString(saveRes["html"])
	if !strings.Contains(html, "delayed body marker") {
		t.Fatalf("save-html html missing delayed marker: %q", html)
	}
	path := strings.TrimSpace(asString(saveRes["path"]))
	if path == "" {
		t.Fatalf("save-html result missing path: %#v", saveRes)
	}
	saved, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read save-html artifact: %v", err)
	}
	if !strings.Contains(string(saved), "delayed body marker") {
		t.Fatalf("saved html missing delayed marker: %q", string(saved))
	}
}

func TestCLIWorkflowJSEvalPositionalExpression(t *testing.T) {
	profile := "e2e-js-eval"
	env := newE2EEnv(t)
	bin := buildCLIForE2E(t)
	pageURL := startE2ETestServer(t)

	t.Cleanup(func() {
		_, _ = runCLICommand(t, env, 15*time.Second, bin, "--profile", profile, "stop")
	})

	runCLIJSON(t, env, 60*time.Second, bin,
		"--profile", profile,
		"--output", "json",
		"goto", pageURL,
	)

	evalRes := runCLIJSON(t, env, 15*time.Second, bin,
		"--profile", profile,
		"--output", "json",
		"js-eval", "document.title",
	)
	if got := strings.TrimSpace(asString(evalRes["result"])); got != "dev-browser-go e2e" {
		t.Fatalf("js-eval result = %q, want %q", got, "dev-browser-go e2e")
	}
}

func TestCLIWorkflowReconfiguresProfileWhenContextFlagsChange(t *testing.T) {
	profile := "e2e-reconfigure-profile"
	env := newE2EEnv(t)
	bin := buildCLIForE2E(t)
	pageURL := startE2ETestServer(t)

	t.Cleanup(func() {
		_, _ = runCLICommand(t, env, 15*time.Second, bin, "--profile", profile, "stop")
	})

	runCLIJSON(t, env, 60*time.Second, bin,
		"--profile", profile,
		"--window-size", "1280x800",
		"--output", "json",
		"goto", pageURL,
	)

	desktopEval := runCLIJSON(t, env, 15*time.Second, bin,
		"--profile", profile,
		"--output", "json",
		"js-eval", "({innerWidth: window.innerWidth, userAgent: navigator.userAgent, href: location.href})",
	)
	desktop, ok := desktopEval["result"].(map[string]any)
	if !ok {
		t.Fatalf("desktop js-eval result = %#v, want object", desktopEval["result"])
	}
	if got := asInt(desktop["innerWidth"]); got != 1280 {
		t.Fatalf("desktop innerWidth = %d, want 1280", got)
	}
	if got := asString(desktop["href"]); !strings.Contains(got, pageURL) {
		t.Fatalf("desktop href = %q, want %q", got, pageURL)
	}

	runCLIJSON(t, env, 60*time.Second, bin,
		"--profile", profile,
		"--device", "iPhone 13",
		"--output", "json",
		"goto", pageURL,
	)

	statusOut, _ := runCLICommand(t, env, 15*time.Second, bin,
		"--profile", profile,
		"status",
	)
	if !strings.Contains(statusOut, "device=iPhone 13") {
		t.Fatalf("status output = %q, want mobile device", statusOut)
	}
	if !strings.Contains(statusOut, "viewport=390x") {
		t.Fatalf("status output = %q, want iPhone viewport", statusOut)
	}

	mobileEval := runCLIJSON(t, env, 15*time.Second, bin,
		"--profile", profile,
		"--output", "json",
		"js-eval", "({innerWidth: window.innerWidth, userAgent: navigator.userAgent, href: location.href})",
	)
	mobile, ok := mobileEval["result"].(map[string]any)
	if !ok {
		t.Fatalf("mobile js-eval result = %#v, want object", mobileEval["result"])
	}
	if got := asInt(mobile["innerWidth"]); got != 390 {
		t.Fatalf("mobile innerWidth = %d, want 390", got)
	}
	if got := asString(mobile["userAgent"]); !strings.Contains(strings.ToLower(got), "iphone") {
		t.Fatalf("mobile userAgent = %q, want iphone", got)
	}
	if got := asString(mobile["href"]); !strings.Contains(got, pageURL) {
		t.Fatalf("mobile href = %q, want %q", got, pageURL)
	}
}

func TestCLILifecycleNamedPagesAndClosePage(t *testing.T) {
	profile := "e2e-lifecycle"
	env := newE2EEnv(t)
	bin := buildCLIForE2E(t)
	mainURL := startTitledServer(t, "main page title", "main page body")
	secondaryURL := startTitledServer(t, "secondary page title", "secondary page body")

	t.Cleanup(func() {
		_, _ = runCLICommand(t, env, 15*time.Second, bin, "--profile", profile, "stop")
	})

	startOut, _ := runCLICommand(t, env, 30*time.Second, bin,
		"--profile", profile,
		"--window-size", "1280x800",
		"start",
	)
	if !strings.Contains(startOut, "started profile="+profile) {
		t.Fatalf("start output = %q, want started", startOut)
	}

	statusOut, _ := runCLICommand(t, env, 15*time.Second, bin,
		"--profile", profile,
		"status",
	)
	if !strings.Contains(statusOut, "ok profile="+profile) {
		t.Fatalf("status output = %q, want healthy daemon", statusOut)
	}

	runCLIJSON(t, env, 45*time.Second, bin,
		"--profile", profile,
		"--output", "json",
		"goto", mainURL,
	)
	runCLIJSON(t, env, 45*time.Second, bin,
		"--profile", profile,
		"--output", "json",
		"goto", secondaryURL, "--page", "secondary",
	)

	pagesRes := runCLIJSON(t, env, 15*time.Second, bin,
		"--profile", profile,
		"--output", "json",
		"list-pages",
	)
	pages := toStringSet(pagesRes["pages"])
	if !pages["main"] || !pages["secondary"] {
		t.Fatalf("list-pages missing expected pages: %#v", pagesRes)
	}

	mainTitle := runCLIJSON(t, env, 15*time.Second, bin,
		"--profile", profile,
		"--output", "json",
		"js-eval", "document.title",
	)
	if got := strings.TrimSpace(asString(mainTitle["result"])); got != "main page title" {
		t.Fatalf("main page title = %q, want %q", got, "main page title")
	}

	secondaryTitle := runCLIJSON(t, env, 15*time.Second, bin,
		"--profile", profile,
		"--output", "json",
		"--page", "secondary",
		"js-eval", "document.title",
	)
	if got := strings.TrimSpace(asString(secondaryTitle["result"])); got != "secondary page title" {
		t.Fatalf("secondary page title = %q, want %q", got, "secondary page title")
	}

	closeRes := runCLIJSON(t, env, 15*time.Second, bin,
		"--profile", profile,
		"--output", "json",
		"close-page", "secondary",
	)
	if closed, ok := closeRes["closed"].(bool); !ok || !closed {
		t.Fatalf("close-page result = %#v, want closed=true", closeRes)
	}

	pagesAfterClose := runCLIJSON(t, env, 15*time.Second, bin,
		"--profile", profile,
		"--output", "json",
		"list-pages",
	)
	remaining := toStringSet(pagesAfterClose["pages"])
	if !remaining["main"] || remaining["secondary"] {
		t.Fatalf("list-pages after close unexpected: %#v", pagesAfterClose)
	}

	stopOut, _ := runCLICommand(t, env, 15*time.Second, bin,
		"--profile", profile,
		"stop",
	)
	if !strings.Contains(stopOut, "stopped profile="+profile) {
		t.Fatalf("stop output = %q, want stopped", stopOut)
	}
}
