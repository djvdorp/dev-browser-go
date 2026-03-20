package devbrowser

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestSaveHTMLReconnectWaitsForNonEmptyBody(t *testing.T) {
	profile := fmt.Sprintf("test-save-html-%d", time.Now().UnixNano())
	port, err := chooseFreePort()
	if err != nil {
		t.Fatalf("choose daemon port: %v", err)
	}
	cdpPort, err := chooseFreePort()
	if err != nil {
		t.Fatalf("choose cdp port: %v", err)
	}
	daemonErr := make(chan error, 1)
	go func() {
		daemonErr <- ServeDaemon(DaemonOptions{
			Profile:   profile,
			Host:      "127.0.0.1",
			Port:      port,
			CDPPort:   cdpPort,
			Headless:  true,
			StateFile: StateFile(profile),
		})
	}()
	if err := waitForTestDaemon(profile, daemonErr); err != nil {
		skipIfBrowserUnavailable(t, err)
		t.Fatalf("start daemon: %v", err)
	}
	t.Cleanup(func() {
		_, _ = StopDaemon(profile)
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, `<!doctype html>
<html>
<head>
  <meta charset="utf-8">
  <title>Save HTML Regression</title>
  <script>
    document.addEventListener('DOMContentLoaded', () => {
      setTimeout(() => {
        const main = document.createElement('main');
        main.id = 'marker';
        main.textContent = 'save-html regression marker';
        document.body.appendChild(main);
      }, 150);
    });
  </script>
</head>
<body></body>
</html>`)
	}))
	defer server.Close()

	pageName := "save-html-regression"
	first, err := EnsurePageInfo(profile, true, pageName, nil, "")
	if err != nil {
		skipIfBrowserUnavailable(t, err)
		t.Fatalf("ensure page for goto: %v", err)
	}
	pw, browser, page, err := OpenPage(first.WSEndpoint, first.TargetID)
	if err != nil {
		skipIfBrowserUnavailable(t, err)
		t.Fatalf("open page for goto: %v", err)
	}
	if _, err := RunCall(page, "goto", map[string]interface{}{
		"url":        server.URL,
		"wait_until": "domcontentloaded",
		"timeout_ms": 10_000,
	}, ArtifactDir(profile)); err != nil {
		_ = browser.Close()
		_ = pw.Stop()
		t.Fatalf("goto: %v", err)
	}
	_ = browser.Close()
	_ = pw.Stop()

	second, err := EnsurePageInfo(profile, true, pageName, nil, "")
	if err != nil {
		t.Fatalf("ensure page for save_html: %v", err)
	}
	pw, browser, page, err = OpenPage(second.WSEndpoint, second.TargetID)
	if err != nil {
		t.Fatalf("open page for save_html: %v", err)
	}
	defer browser.Close()
	defer pw.Stop()

	result, err := RunCall(page, "save_html", map[string]interface{}{
		"path":       "save-html-regression.html",
		"timeout_ms": 10_000,
	}, ArtifactDir(profile))
	if err != nil {
		t.Fatalf("save_html: %v", err)
	}

	html, _ := result["html"].(string)
	path, _ := result["path"].(string)
	if strings.TrimSpace(path) == "" {
		t.Fatal("save_html did not return artifact path")
	}
	if result["url"] != server.URL+"/" && result["url"] != server.URL {
		t.Fatalf("unexpected result url: %#v", result["url"])
	}
	if result["title"] != "Save HTML Regression" {
		t.Fatalf("unexpected result title: %#v", result["title"])
	}
	if htmlLen, ok := result["html_length"].(int); !ok || htmlLen <= len("<html><head></head><body></body></html>") {
		t.Fatalf("unexpected html_length: %#v", result["html_length"])
	}
	if strings.TrimSpace(html) == "<html><head></head><body></body></html>" {
		t.Fatalf("save_html returned empty shell: %q", html)
	}
	if !strings.Contains(html, "save-html regression marker") {
		t.Fatalf("save_html html missing expected marker: %q", html)
	}

	saved, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read saved html: %v", err)
	}
	savedHTML := string(saved)
	if strings.TrimSpace(savedHTML) == "<html><head></head><body></body></html>" {
		t.Fatalf("saved file contains empty shell: %q", savedHTML)
	}
	if !strings.Contains(savedHTML, "save-html regression marker") {
		t.Fatalf("saved file missing expected marker: %q", savedHTML)
	}
}

func skipIfBrowserUnavailable(t *testing.T, err error) {
	t.Helper()
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "playwright"):
		t.Skipf("browser environment unavailable: %v", err)
	case strings.Contains(msg, "chromium"):
		t.Skipf("browser environment unavailable: %v", err)
	case strings.Contains(msg, "executable doesn't exist"):
		t.Skipf("browser environment unavailable: %v", err)
	}
}

func waitForTestDaemon(profile string, daemonErr <-chan error) error {
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		if IsDaemonHealthy(profile) {
			return nil
		}
		select {
		case err := <-daemonErr:
			if err == nil {
				return errors.New("daemon exited before becoming healthy")
			}
			return err
		default:
		}
		time.Sleep(200 * time.Millisecond)
	}
	select {
	case err := <-daemonErr:
		if err != nil {
			return err
		}
	default:
	}
	return fmt.Errorf("timed out waiting for daemon health (profile=%s)", profile)
}
