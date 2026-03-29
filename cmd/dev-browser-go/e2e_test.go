package main

import (
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

var (
	builtBinaryOnce sync.Once
	builtBinaryPath string
	builtBinaryErr  error
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

func newE2EEnv(t *testing.T) []string {
	t.Helper()
	root := t.TempDir()
	cacheHome := filepath.Join(root, "cache")
	stateHome := filepath.Join(root, "state")
	if err := os.MkdirAll(cacheHome, 0o755); err != nil {
		t.Fatalf("mkdir cache: %v", err)
	}
	if err := os.MkdirAll(stateHome, 0o755); err != nil {
		t.Fatalf("mkdir state: %v", err)
	}

	env := append([]string{}, os.Environ()...)
	env = append(env,
		"HEADLESS=1",
		"XDG_CACHE_HOME="+cacheHome,
		"XDG_STATE_HOME="+stateHome,
	)
	return env
}

func buildCLIForE2E(t *testing.T) string {
	t.Helper()
	builtBinaryOnce.Do(func() {
		if _, err := exec.LookPath("go"); err != nil {
			builtBinaryErr = err
			return
		}
		outDir, err := os.MkdirTemp("", "dev-browser-go-e2e-*")
		if err != nil {
			builtBinaryErr = err
			return
		}
		builtBinaryPath = filepath.Join(outDir, "dev-browser-go")
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		cmd := exec.CommandContext(ctx, "go", "build", "-o", builtBinaryPath, "./cmd/dev-browser-go")
		cmd.Dir = repoRoot(t)
		out, err := cmd.CombinedOutput()
		if ctx.Err() == context.DeadlineExceeded {
			builtBinaryErr = ctx.Err()
			return
		}
		if err != nil {
			builtBinaryErr = &execError{err: err, output: string(out)}
			return
		}
	})

	if builtBinaryErr != nil {
		t.Fatalf("build e2e binary: %v", builtBinaryErr)
	}
	return builtBinaryPath
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func startE2ETestServer(t *testing.T) string {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<!doctype html>
<html>
<head>
  <meta charset="utf-8">
  <title>dev-browser-go e2e</title>
  <style>
    html, body {
      margin: 0;
      min-height: 100%;
      background: #123456;
      color: #f7fafc;
      font-family: sans-serif;
    }
    main {
      min-height: 100vh;
      display: grid;
      place-items: center;
      background:
        linear-gradient(135deg, rgba(18, 52, 86, 1) 0%, rgba(0, 160, 160, 1) 100%);
    }
    h1 {
      font-size: 48px;
      margin: 0;
    }
  </style>
</head>
<body>
  <main>
    <h1>daemon persistence check</h1>
  </main>
</body>
</html>`))
	}))
	t.Cleanup(server.Close)
	return server.URL
}

func startDelayedBodyServer(t *testing.T, delay time.Duration) string {
	t.Helper()
	delayMS := int(delay / time.Millisecond)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = fmt.Fprintf(w, `<!doctype html>
<html>
<head>
  <meta charset="utf-8">
  <title>dev-browser-go delayed body</title>
  <script>
    document.addEventListener('DOMContentLoaded', () => {
      setTimeout(() => {
        const main = document.createElement('main');
        main.id = 'marker';
        main.textContent = 'delayed body marker';
        document.body.appendChild(main);
      }, %d);
    });
  </script>
</head>
<body></body>
</html>`, delayMS)
	}))
	t.Cleanup(server.Close)
	return server.URL
}

func startTitledServer(t *testing.T, title, body string) string {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = fmt.Fprintf(w, `<!doctype html>
<html>
<head><meta charset="utf-8"><title>%s</title></head>
<body><main>%s</main></body>
</html>`, title, body)
	}))
	t.Cleanup(server.Close)
	return server.URL
}

func startHTMLValidateFixtureServer(t *testing.T) string {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<!doctype html>
<html>
<head><meta charset="utf-8"><title>html validate fixture</title></head>
<body>
  <div id="dup"></div>
  <span id="dup"></span>
  <img src="missing-alt.png">
  <input type="text">
</body>
</html>`))
	}))
	t.Cleanup(server.Close)
	return server.URL
}

func runCLIJSON(t *testing.T, env []string, timeout time.Duration, bin string, args ...string) map[string]any {
	t.Helper()
	stdout, stderr := runCLICommand(t, env, timeout, bin, args...)
	var payload map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &payload); err != nil {
		t.Fatalf("decode json output: %v\nstdout=%q\nstderr=%q", err, stdout, stderr)
	}
	return payload
}

func runCLIJSONWithExit(t *testing.T, env []string, timeout time.Duration, bin string, args ...string) (map[string]any, int) {
	t.Helper()
	stdout, stderr, code := runCLICommandAllowExit(t, env, timeout, bin, args...)
	var payload map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &payload); err != nil {
		t.Fatalf("decode json output: %v\nstdout=%q\nstderr=%q\nexit=%d", err, stdout, stderr, code)
	}
	return payload, code
}

func runCLICommand(t *testing.T, env []string, timeout time.Duration, bin string, args ...string) (string, string) {
	t.Helper()
	stdout, stderr, code := runCLICommandAllowExit(t, env, timeout, bin, args...)
	if code != 0 {
		t.Fatalf("command failed: %s %s\nstdout=%s\nstderr=%s\nexit=%d", bin, strings.Join(args, " "), stdout, stderr, code)
	}
	return stdout, stderr
}

func runCLICommandAllowExit(t *testing.T, env []string, timeout time.Duration, bin string, args ...string) (string, string, int) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Env = env
	cmd.Dir = repoRoot(t)
	stdout, err := cmd.Output()
	stderr := ""
	if exitErr := (*exec.ExitError)(nil); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			t.Fatalf("command timed out: %s %s", bin, strings.Join(args, " "))
		}
		if ok := asExitError(err, &exitErr); ok {
			stderr = string(exitErr.Stderr)
			maybeSkipForBrowserUnavailable(t, stdout, exitErr.Stderr)
			return string(stdout), stderr, exitErr.ExitCode()
		}
		t.Fatalf("run command: %v", err)
	}
	return string(stdout), stderr, 0
}

func asExitError(err error, target **exec.ExitError) bool {
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		return false
	}
	*target = exitErr
	return true
}

func maybeSkipForBrowserUnavailable(t *testing.T, stdout, stderr []byte) {
	t.Helper()
	combined := strings.ToLower(string(stdout) + "\n" + string(stderr))
	switch {
	case strings.Contains(combined, "playwright"):
		t.Skipf("browser environment unavailable: %s", strings.TrimSpace(combined))
	case strings.Contains(combined, "chromium"):
		t.Skipf("browser environment unavailable: %s", strings.TrimSpace(combined))
	case strings.Contains(combined, "executable doesn't exist"):
		t.Skipf("browser environment unavailable: %s", strings.TrimSpace(combined))
	}
}

func assertScreenshotLooksReal(t *testing.T, path string, wantWidth int) (int, int) {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open screenshot %q: %v", path, err)
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		t.Fatalf("decode screenshot %q: %v", path, err)
	}
	bounds := img.Bounds()
	if wantWidth > 0 && bounds.Dx() != wantWidth {
		t.Fatalf("screenshot width = %d, want %d", bounds.Dx(), wantWidth)
	}
	if bounds.Dy() <= 0 {
		t.Fatalf("screenshot height = %d, want > 0", bounds.Dy())
	}

	x := bounds.Min.X + maxInt(1, bounds.Dx()/10)
	y := bounds.Min.Y + maxInt(1, bounds.Dy()/10)
	if nearWhite(img.At(x, y)) {
		t.Fatalf("screenshot pixel at (%d,%d) looks blank/white", x, y)
	}
	return bounds.Dx(), bounds.Dy()
}

func nearWhite(c color.Color) bool {
	r, g, b, _ := c.RGBA()
	return r > 0xf000 && g > 0xf000 && b > 0xf000
}

func asString(v any) string {
	s, _ := v.(string)
	return s
}

func asInt(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	default:
		return 0
	}
}

func toStringSet(v any) map[string]bool {
	out := map[string]bool{}
	switch items := v.(type) {
	case []any:
		for _, item := range items {
			if s, ok := item.(string); ok {
				out[s] = true
			}
		}
	case []string:
		for _, s := range items {
			out[s] = true
		}
	}
	return out
}

func nestedBool(v any, outerKey, innerKey string) bool {
	m, ok := v.(map[string]any)
	if !ok {
		return false
	}
	inner, ok := m[outerKey].(map[string]any)
	if ok {
		b, _ := inner[innerKey].(bool)
		return b
	}
	innerRaw, ok := m[outerKey].(map[string]interface{})
	if ok {
		b, _ := innerRaw[innerKey].(bool)
		return b
	}
	return false
}

func consoleEntriesContain(v any, needle string) bool {
	entries, ok := v.([]any)
	if !ok {
		return false
	}
	for _, entry := range entries {
		m, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		text, _ := m["text"].(string)
		if strings.Contains(text, needle) {
			return true
		}
	}
	return false
}

func findingsContainRule(v any, ruleID string) bool {
	findings, ok := v.([]any)
	if !ok {
		return false
	}
	for _, finding := range findings {
		m, ok := finding.(map[string]any)
		if !ok {
			continue
		}
		if got, _ := m["ruleId"].(string); got == ruleID {
			return true
		}
	}
	return false
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

type execError struct {
	err    error
	output string
}

func (e *execError) Error() string {
	if strings.TrimSpace(e.output) == "" {
		return e.err.Error()
	}
	return e.err.Error() + ": " + strings.TrimSpace(e.output)
}
