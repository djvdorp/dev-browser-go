package devbrowser

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

type WindowSize struct {
	Width  int
	Height int
}

var (
	defaultWindowSize = WindowSize{Width: 2500, Height: 1920}
	windowSizeRe      = regexp.MustCompile(`^\s*(\d+)\s*[xX*,]\s*(\d+)\s*$`)
)

func WindowSizeFromEnv() (*WindowSize, error) {
	raw := strings.TrimSpace(os.Getenv("DEV_BROWSER_WINDOW_SIZE"))
	if raw == "" {
		return &defaultWindowSize, nil
	}
	normalized := strings.ToLower(strings.ReplaceAll(raw, "px", ""))
	match := windowSizeRe.FindStringSubmatch(normalized)
	if len(match) != 3 {
		return nil, fmt.Errorf("DEV_BROWSER_WINDOW_SIZE must be WIDTHxHEIGHT (e.g. 2500x1920)")
	}
	w, _ := strconv.Atoi(match[1])
	h, _ := strconv.Atoi(match[2])
	if w < 1 || h < 1 {
		return nil, fmt.Errorf("DEV_BROWSER_WINDOW_SIZE must be positive (e.g. 2500x1920)")
	}
	return &WindowSize{Width: w, Height: h}, nil
}

func ChromiumLaunchArgs(cdpPort int, window *WindowSize) []string {
	args := []string{}
	if cdpPort > 0 {
		args = append(args, fmt.Sprintf("--remote-debugging-port=%d", cdpPort))
	}
	if !envTruthy("DEV_BROWSER_USE_KEYCHAIN") {
		args = append(args, "--use-mock-keychain")
	}
	if window != nil {
		args = append(args, fmt.Sprintf("--window-size=%d,%d", window.Width, window.Height))
	}
	return args
}
