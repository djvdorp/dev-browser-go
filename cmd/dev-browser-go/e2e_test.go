package main

import (
	"context"
	"encoding/json"
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

func runCLIJSON(t *testing.T, env []string, timeout time.Duration, bin string, args ...string) map[string]any {
	t.Helper()
	stdout, stderr := runCLICommand(t, env, timeout, bin, args...)
	var payload map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &payload); err != nil {
		t.Fatalf("decode json output: %v\nstdout=%q\nstderr=%q", err, stdout, stderr)
	}
	return payload
}

func runCLICommand(t *testing.T, env []string, timeout time.Duration, bin string, args ...string) (string, string) {
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
			t.Fatalf("command failed: %s %s\nstdout=%s\nstderr=%s", bin, strings.Join(args, " "), stdout, exitErr.Stderr)
		}
		t.Fatalf("run command: %v", err)
	}
	return string(stdout), stderr
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
