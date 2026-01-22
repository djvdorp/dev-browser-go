package devbrowser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/playwright-community/playwright-go"
)

func TestResolveDiffPathEmpty(t *testing.T) {
	dir := t.TempDir()
	path, capture, err := resolveDiffPath(dir, "", "before")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != "" {
		t.Fatalf("expected empty path, got %q", path)
	}
	if !capture {
		t.Fatalf("expected capture=true")
	}
}

func TestResolveDiffPathExistingFile(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "baseline.png")
	if err := os.WriteFile(filePath, []byte("x"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	path, capture, err := resolveDiffPath(dir, "baseline.png", "before")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		t.Fatalf("abs path: %v", err)
	}
	if path != absPath {
		t.Fatalf("expected %q, got %q", absPath, path)
	}
	if capture {
		t.Fatalf("expected capture=false")
	}
}

func TestResolveDiffPathMissingFile(t *testing.T) {
	dir := t.TempDir()
	path, capture, err := resolveDiffPath(dir, "missing.png", "before")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(dir, "missing.png")
	want, err = filepath.Abs(want)
	if err != nil {
		t.Fatalf("abs path: %v", err)
	}
	if path != want {
		t.Fatalf("expected %q, got %q", want, path)
	}
	if !capture {
		t.Fatalf("expected capture=true")
	}
}

func TestResolveDiffPathDir(t *testing.T) {
	dir := t.TempDir()
	childDir := filepath.Join(dir, "folder")
	if err := os.MkdirAll(childDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if _, _, err := resolveDiffPath(dir, "folder", "before"); err == nil {
		t.Fatalf("expected error for directory path")
	}
}

func TestCaptureScreenshotPathUsesExplicitPath(t *testing.T) {
	var gotName string
	var gotArgs map[string]interface{}
	callFn := func(_ playwright.Page, name string, args map[string]interface{}, _ string) (RunResult, error) {
		gotName = name
		gotArgs = args
		path, _ := args["path"].(string)
		return RunResult{"path": path}, nil
	}

	path, err := captureScreenshotPath(nil, map[string]interface{}{}, t.TempDir(), "custom.png", "default.png", callFn)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotName != "screenshot" {
		t.Fatalf("expected screenshot call, got %q", gotName)
	}
	if gotArgs["path"] != "custom.png" {
		t.Fatalf("expected path arg custom.png, got %v", gotArgs["path"])
	}
	if path != "custom.png" {
		t.Fatalf("expected path custom.png, got %q", path)
	}
}

func TestCaptureScreenshotPathUsesDefault(t *testing.T) {
	callFn := func(_ playwright.Page, _ string, args map[string]interface{}, _ string) (RunResult, error) {
		path, _ := args["path"].(string)
		return RunResult{"path": path}, nil
	}

	path, err := captureScreenshotPath(nil, map[string]interface{}{}, t.TempDir(), "", "default.png", callFn)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != "default.png" {
		t.Fatalf("expected default.png, got %q", path)
	}
}

func TestCaptureScreenshotPathMissingReturn(t *testing.T) {
	callFn := func(_ playwright.Page, _ string, _ map[string]interface{}, _ string) (RunResult, error) {
		return RunResult{}, nil
	}

	if _, err := captureScreenshotPath(nil, map[string]interface{}{}, t.TempDir(), "custom.png", "default.png", callFn); err == nil {
		t.Fatalf("expected error when screenshot returns no path")
	}
}
