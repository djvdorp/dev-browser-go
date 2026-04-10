package devbrowser

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

type BrowserContextSettings struct {
	Headless          bool        `json:"headless"`
	Device            string      `json:"device,omitempty"`
	Window            *WindowSize `json:"window,omitempty"`
	Viewport          *WindowSize `json:"viewport,omitempty"`
	Screen            *WindowSize `json:"screen,omitempty"`
	DeviceScaleFactor float64     `json:"deviceScaleFactor,omitempty"`
	IsMobile          bool        `json:"isMobile,omitempty"`
	HasTouch          bool        `json:"hasTouch,omitempty"`
	UserAgent         string      `json:"userAgent,omitempty"`
}

type DaemonAction string

const (
	DaemonActionStarted      DaemonAction = "started"
	DaemonActionReused       DaemonAction = "reused"
	DaemonActionReconfigured DaemonAction = "reconfigured"
)

type DaemonStartResult struct {
	Action     DaemonAction           `json:"action"`
	Reason     string                 `json:"reason,omitempty"`
	Context    BrowserContextSettings `json:"context"`
	BaseURL    string                 `json:"baseURL,omitempty"`
	Profile    string                 `json:"profile,omitempty"`
	PageURL    string                 `json:"pageURL,omitempty"`
	WSEndpoint string                 `json:"wsEndpoint,omitempty"`
}

type DaemonHealth struct {
	OK         bool                   `json:"ok"`
	PID        int                    `json:"pid"`
	Host       string                 `json:"host,omitempty"`
	Port       int                    `json:"port,omitempty"`
	Profile    string                 `json:"profile,omitempty"`
	CDPPort    int                    `json:"cdpPort,omitempty"`
	WSEndpoint string                 `json:"wsEndpoint,omitempty"`
	Version    string                 `json:"version,omitempty"`
	Context    BrowserContextSettings `json:"context"`
	PageURL    string                 `json:"pageURL,omitempty"`
}

func cloneWindowSize(src *WindowSize) *WindowSize {
	if src == nil {
		return nil
	}
	copy := *src
	return &copy
}

func normalizeContextRequest(headless bool, window *WindowSize, device string) BrowserContextSettings {
	settings := BrowserContextSettings{
		Headless: headless,
		Device:   strings.TrimSpace(device),
		Window:   cloneWindowSize(window),
	}
	if settings.Window == nil && settings.Device == "" {
		defaultSize := DefaultWindowSize()
		settings.Window = &defaultSize
	}
	return settings
}

func effectiveContextMatches(current BrowserContextSettings, requested BrowserContextSettings) bool {
	if current.Headless != requested.Headless {
		return false
	}
	currentDevice := strings.TrimSpace(current.Device)
	requestedDevice := strings.TrimSpace(requested.Device)
	if !strings.EqualFold(currentDevice, requestedDevice) {
		return false
	}
	if requestedDevice != "" {
		return true
	}
	return windowSizesEqual(current.Window, requested.Window)
}

func windowSizesEqual(a, b *WindowSize) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Width == b.Width && a.Height == b.Height
}

func formatContextSummary(settings BrowserContextSettings) string {
	parts := []string{fmt.Sprintf("headless=%t", settings.Headless)}
	if device := strings.TrimSpace(settings.Device); device != "" {
		parts = append(parts, fmt.Sprintf("device=%s", device))
	}
	if settings.Window != nil {
		parts = append(parts, fmt.Sprintf("window=%dx%d", settings.Window.Width, settings.Window.Height))
	}
	if settings.Viewport != nil {
		parts = append(parts, fmt.Sprintf("viewport=%dx%d", settings.Viewport.Width, settings.Viewport.Height))
	}
	if settings.Screen != nil {
		parts = append(parts, fmt.Sprintf("screen=%dx%d", settings.Screen.Width, settings.Screen.Height))
	}
	if settings.DeviceScaleFactor > 0 {
		parts = append(parts, fmt.Sprintf("scale=%.2f", settings.DeviceScaleFactor))
	}
	if settings.IsMobile {
		parts = append(parts, "mobile=true")
	}
	if settings.HasTouch {
		parts = append(parts, "touch=true")
	}
	return strings.Join(parts, " ")
}

func describeContextDiff(current BrowserContextSettings, requested BrowserContextSettings) string {
	diffs := make([]string, 0, 3)
	if current.Headless != requested.Headless {
		diffs = append(diffs, fmt.Sprintf("headless=%t", requested.Headless))
	}
	if !strings.EqualFold(strings.TrimSpace(current.Device), strings.TrimSpace(requested.Device)) {
		if requested.Device != "" {
			diffs = append(diffs, fmt.Sprintf("device=%s", requested.Device))
		} else {
			diffs = append(diffs, "device=none")
		}
	}
	if strings.TrimSpace(requested.Device) == "" && !windowSizesEqual(current.Window, requested.Window) {
		if requested.Window != nil {
			diffs = append(diffs, fmt.Sprintf("window=%dx%d", requested.Window.Width, requested.Window.Height))
		} else {
			diffs = append(diffs, "window=auto")
		}
	}
	sort.Strings(diffs)
	return strings.Join(diffs, " ")
}

func floatNearlyEqual(a, b float64) bool {
	return math.Abs(a-b) < 0.0001
}
