package main

import (
	"context"
	"encoding/json"
	"image"
	"image/color"
	_ "image/png"
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

func runCLIJSON(t *testing.T, env []string, timeout time.Duration, bin string, args ...string) map[string]any {
	t.Helper()
	stdout, stderr := runCLICommand(t, env, timeout, bin, args...)
	var payload map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &payload); err != nil {
		t.Fatalf("decode json output: %v\nstdout=%q\nstderr=%q", err, stdout, stderr)
	}
	return payload
}

func runCLIJSONWithInput(t *testing.T, env []string, timeout time.Duration, bin string, input string, args ...string) map[string]any {
	t.Helper()
	stdout, stderr := runCLICommandWithInput(t, env, timeout, bin, input, args...)
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

func runCLICommandWithInput(t *testing.T, env []string, timeout time.Duration, bin string, input string, args ...string) (string, string) {
	t.Helper()
	stdout, stderr, code := runCLICommandAllowExitWithInput(t, env, timeout, bin, input, args...)
	if code != 0 {
		t.Fatalf("command failed: %s %s\nstdout=%s\nstderr=%s\nexit=%d", bin, strings.Join(args, " "), stdout, stderr, code)
	}
	return stdout, stderr
}

func runCLICommandAllowExit(t *testing.T, env []string, timeout time.Duration, bin string, args ...string) (string, string, int) {
	return runCLICommandAllowExitWithInput(t, env, timeout, bin, "", args...)
}

func runCLICommandAllowExitWithInput(t *testing.T, env []string, timeout time.Duration, bin string, input string, args ...string) (string, string, int) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Env = env
	cmd.Dir = repoRoot(t)
	if input != "" {
		cmd.Stdin = strings.NewReader(input)
	}
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

func asFloat64(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int:
		return float64(n)
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

func snapshotItems(v any) []map[string]any {
	items, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		m, ok := item.(map[string]any)
		if ok {
			out = append(out, m)
		}
	}
	return out
}

func findSnapshotRef(items []map[string]any, role, name string) string {
	role = strings.ToLower(strings.TrimSpace(role))
	name = strings.ToLower(strings.TrimSpace(name))
	for _, item := range items {
		itemRole := strings.ToLower(strings.TrimSpace(asString(item["role"])))
		itemName := strings.ToLower(strings.TrimSpace(asString(item["name"])))
		if itemRole == role && strings.Contains(itemName, name) {
			return strings.TrimSpace(asString(item["ref"]))
		}
	}
	return ""
}

func networkEntriesContainURL(v any, needle string) bool {
	entries, ok := v.([]any)
	if !ok {
		return false
	}
	for _, entry := range entries {
		m, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		url, _ := m["url"].(string)
		if strings.Contains(url, needle) {
			return true
		}
	}
	return false
}

func previewContainsText(v any, needle string) bool {
	needle = strings.ToLower(strings.TrimSpace(needle))
	entries, ok := v.([]any)
	if !ok {
		return false
	}
	for _, entry := range entries {
		m, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		text := strings.ToLower(strings.TrimSpace(asString(m["text"])))
		if strings.Contains(text, needle) {
			return true
		}
	}
	return false
}

func minimalPNG() []byte {
	return []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
		0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4,
		0x89, 0x00, 0x00, 0x00, 0x0d, 0x49, 0x44, 0x41,
		0x54, 0x78, 0x9c, 0x63, 0xf8, 0xcf, 0xc0, 0xf0,
		0x1f, 0x00, 0x05, 0x00, 0x01, 0xff, 0x89, 0x99,
		0x3d, 0x1d, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45,
		0x4e, 0x44, 0xae, 0x42, 0x60, 0x82,
	}
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
