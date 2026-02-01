package devbrowser

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/playwright-community/playwright-go"
)

// ArtifactMode controls what files diagnose/assert/html-validate write.
//
// none: do not write anything
// minimal: screenshot + report.json
// full: screenshot + report.json + component JSON/YAML files
//
// Note: these modes only affect file writes; JSON output always includes paths when written.
//
// This is intentionally stringly-typed for stable CLI flag parsing.
type ArtifactMode string

const (
	ArtifactModeNone    ArtifactMode = "none"
	ArtifactModeMinimal ArtifactMode = "minimal"
	ArtifactModeFull    ArtifactMode = "full"
)

func (m ArtifactMode) Valid() bool {
	switch m {
	case ArtifactModeNone, ArtifactModeMinimal, ArtifactModeFull:
		return true
	default:
		return false
	}
}

// DiagnoseOptions configures a single diagnose run.
type DiagnoseOptions struct {
	URL string

	WaitState   string
	TimeoutMs   int
	MinWaitMs   int
	PageName    string
	Profile     string
	RunID       string
	Timestamp   time.Time
	ArtifactDir string
	Artifacts   ArtifactMode

	SnapshotEngine string

	NetBodies       bool
	NetMaxBodyBytes int

	PerfSampleMs int
	PerfTopN     int
}

type DiagnoseMeta struct {
	URL         string `json:"url"`
	Page        string `json:"page"`
	Profile     string `json:"profile"`
	TS          string `json:"ts"`
	RunID       string `json:"runId"`
	ArtifactDir string `json:"artifactDir,omitempty"`
}

type DiagnoseConsoleCounts struct {
	Error   int `json:"error"`
	Warning int `json:"warning"`
	Info    int `json:"info"`
}

type DiagnoseConsoleSection struct {
	Entries []ConsoleEntry        `json:"entries"`
	Counts  DiagnoseConsoleCounts `json:"counts"`
}

type DiagnoseNetworkSection struct {
	Total   int            `json:"total"`
	Matched int            `json:"matched"`
	Entries []NetworkEntry `json:"entries"`
}

type DiagnoseSnapshotSection struct {
	Engine string                   `json:"engine"`
	YAML   string                   `json:"yaml"`
	Items  []map[string]interface{} `json:"items"`
}

type DiagnoseArtifacts struct {
	Screenshot string `json:"screenshot,omitempty"`
	Snapshot   string `json:"snapshot,omitempty"`
	Network    string `json:"network,omitempty"`
	Console    string `json:"console,omitempty"`
	Report     string `json:"report,omitempty"`
}

type DiagnoseSummary struct {
	HasConsoleErrors  bool   `json:"hasConsoleErrors"`
	HasHttp4xx5xx     bool   `json:"hasHttp4xx5xx"`
	HasFailedRequests bool   `json:"hasFailedRequests"`
	HasHarnessErrors  bool   `json:"hasHarnessErrors"`
	HarnessErrorCount int    `json:"harnessErrorCount"`
	HasViteOverlay    bool   `json:"hasViteOverlay"`
	ViteOverlayText   string `json:"viteOverlayText,omitempty"`
}

type DiagnoseEvent struct {
	Kind   string         `json:"kind"` // console|network|errorhook|overlay
	TimeMS int64          `json:"time_ms"`
	Data   map[string]any `json:"data"`
}

type DiagnoseHarnessSection struct {
	State map[string]any `json:"state"`
}

type DiagnoseReport struct {
	Meta      DiagnoseMeta            `json:"meta"`
	Console   DiagnoseConsoleSection  `json:"console"`
	Network   DiagnoseNetworkSection  `json:"network"`
	Perf      map[string]any          `json:"perf"`
	Snapshot  DiagnoseSnapshotSection `json:"snapshot"`
	Harness   DiagnoseHarnessSection  `json:"harness"`
	Events    []DiagnoseEvent         `json:"events"`
	Artifacts DiagnoseArtifacts       `json:"artifacts"`
	Summary   DiagnoseSummary         `json:"summary"`
}

func Diagnose(page playwright.Page, opts DiagnoseOptions) (*DiagnoseReport, error) {
	if strings.TrimSpace(opts.PageName) == "" {
		opts.PageName = "main"
	}
	if strings.TrimSpace(opts.Profile) == "" {
		opts.Profile = "default"
	}
	if opts.Timestamp.IsZero() {
		opts.Timestamp = time.Now()
	}
	if strings.TrimSpace(opts.RunID) == "" {
		opts.RunID = NewDiagnoseRunID()
	}
	if strings.TrimSpace(opts.WaitState) == "" {
		opts.WaitState = "networkidle"
	}
	if opts.TimeoutMs <= 0 {
		opts.TimeoutMs = 45_000
	}
	if opts.MinWaitMs < 0 {
		opts.MinWaitMs = 0
	}
	if strings.TrimSpace(opts.SnapshotEngine) == "" {
		opts.SnapshotEngine = "simple"
	}
	if opts.NetMaxBodyBytes <= 0 {
		opts.NetMaxBodyBytes = 32 * 1024
	}

	// Navigate (optional).
	if strings.TrimSpace(opts.URL) != "" {
		_, err := RunCall(page, "goto", map[string]interface{}{
			"url":        opts.URL,
			"wait_until": "domcontentloaded",
			"timeout_ms": opts.TimeoutMs,
		}, opts.ArtifactDir)
		if err != nil {
			// Diagnose should still return a report if possible; caller decides exit code.
			// Here we return error because subsequent calls likely fail without a page.
			return nil, err
		}
	}

	// Wait for state.
	_, _ = RunCall(page, "wait", map[string]interface{}{
		"state":       opts.WaitState,
		"timeout_ms":  opts.TimeoutMs,
		"min_wait_ms": opts.MinWaitMs,
	}, opts.ArtifactDir)

	// Console: read via daemon endpoint in CLI layer; Diagnose() only handles Playwright-based primitives.
	// Callers can populate Console section via SetConsole.

	// Network.
	netRes, _ := RunCall(page, "network_monitor", map[string]interface{}{
		"wait_state":      opts.WaitState,
		"timeout_ms":      opts.TimeoutMs,
		"min_wait_ms":     opts.MinWaitMs,
		"include_bodies":  opts.NetBodies,
		"max_body_bytes":  opts.NetMaxBodyBytes,
		"include_headers": true,
	}, opts.ArtifactDir)

	netEntries := []NetworkEntry{}
	total := 0
	matched := 0
	if v, ok := netRes["entries"].([]NetworkEntry); ok {
		netEntries = v
	} else if raw, ok := netRes["entries"].([]interface{}); ok {
		// defensive: should not happen in-process, but keep it robust.
		b, _ := json.Marshal(raw)
		_ = json.Unmarshal(b, &netEntries)
	}
	if v, ok := netRes["total"].(int); ok {
		total = v
	} else if v, ok := netRes["total"].(float64); ok {
		total = int(v)
	}
	if v, ok := netRes["matched"].(int); ok {
		matched = v
	} else if v, ok := netRes["matched"].(float64); ok {
		matched = int(v)
	}

	// Perf.
	perf, _ := GetPerfMetrics(page, PerfMetricsOptions{SampleMs: opts.PerfSampleMs, TopN: opts.PerfTopN})

	// Snapshot.
	snap, _ := GetSnapshot(page, SnapshotOptions{Engine: opts.SnapshotEngine, Format: "list", InteractiveOnly: false, IncludeHeadings: true, MaxItems: 200, MaxChars: 120_000})

	// Screenshot.
	shotPath := ""
	if opts.Artifacts != ArtifactModeNone {
		// Use the runner screenshot to share crop/annotate logic & safe path logic.
		res, err := RunCall(page, "screenshot", map[string]interface{}{
			"full_page": true,
			"path":      "screenshot.png",
		}, opts.ArtifactDir)
		if err == nil {
			if p, _ := res["path"].(string); strings.TrimSpace(p) != "" {
				shotPath = p
			}
		}
	}

	report := &DiagnoseReport{
		Meta: DiagnoseMeta{
			URL:         safeString(page.URL()),
			Page:        opts.PageName,
			Profile:     opts.Profile,
			TS:          opts.Timestamp.UTC().Format(time.RFC3339Nano),
			RunID:       opts.RunID,
			ArtifactDir: opts.ArtifactDir,
		},
		Console: DiagnoseConsoleSection{
			Entries: nil,
			Counts:  DiagnoseConsoleCounts{},
		},
		Network:  DiagnoseNetworkSection{Total: total, Matched: matched, Entries: netEntries},
		Perf:     perf,
		Snapshot: DiagnoseSnapshotSection{Engine: opts.SnapshotEngine, YAML: snap.Yaml, Items: snap.Items},
		Harness:  DiagnoseHarnessSection{State: nil},
		Events:   []DiagnoseEvent{},
		Artifacts: DiagnoseArtifacts{
			Screenshot: shotPath,
		},
	}

	// Harness state (JS hooks + Vite overlay best-effort).
	if hs, err := ReadHarnessState(page); err == nil && hs != nil {
		report.Harness.State = hs
	}

	// Deterministic ordering.
	SortNetworkEntries(report.Network.Entries)

	// Build combined timeline events (console is populated later via SetConsole).
	report.Events = BuildDiagnoseEvents(nil, report.Network.Entries, report.Harness.State)

	report.computeSummary()
	return report, nil
}

func (r *DiagnoseReport) SetConsole(entries []ConsoleEntry) {
	// Sort deterministically.
	SortConsoleEntries(entries)

	// Truncate any huge console payloads to keep JSON stable and bounded.
	for i := range entries {
		entries[i].Text, _, _ = clampBody(entries[i].Text, 4096)
		entries[i].URL, _, _ = clampBody(entries[i].URL, 1024)
	}

	counts := DiagnoseConsoleCounts{}
	for _, e := range entries {
		switch consoleLevelForType(e.Type) {
		case "error":
			counts.Error++
		case "warning":
			counts.Warning++
		default:
			counts.Info++
		}
	}
	r.Console = DiagnoseConsoleSection{Entries: entries, Counts: counts}
	// Rebuild events when console is populated.
	r.Events = BuildDiagnoseEvents(r.Console.Entries, r.Network.Entries, r.Harness.State)
	r.computeSummary()
}

func (r *DiagnoseReport) computeSummary() {
	// Console errors.
	hasConsoleErrors := r.Console.Counts.Error > 0

	// Network.
	has4xx5xx := false
	hasFailed := false
	for _, e := range r.Network.Entries {
		if e.Status >= 400 {
			has4xx5xx = true
		}
		if !e.OK || strings.TrimSpace(e.Error) != "" {
			hasFailed = true
		}
	}

	// Harness.
	hasHarnessErrors := false
	harnessErrorCount := 0
	hasViteOverlay := false
	viteOverlayText := ""
	if r.Harness.State != nil {
		if arr, ok := r.Harness.State["errors"].([]interface{}); ok {
			harnessErrorCount = len(arr)
			hasHarnessErrors = harnessErrorCount > 0
		}
		if arr, ok := r.Harness.State["overlays"].([]interface{}); ok && len(arr) > 0 {
			hasViteOverlay = true
			// Try to pull the most recent overlay text.
			last := arr[len(arr)-1]
			if m, ok := last.(map[string]any); ok {
				if t, ok := m["text"].(string); ok {
					viteOverlayText = strings.TrimSpace(t)
				}
			}
		}
	}
	if viteOverlayText != "" {
		viteOverlayText, _, _ = clampBody(viteOverlayText, 800)
	}

	r.Summary = DiagnoseSummary{
		HasConsoleErrors:  hasConsoleErrors,
		HasHttp4xx5xx:     has4xx5xx,
		HasFailedRequests: hasFailed,
		HasHarnessErrors:  hasHarnessErrors,
		HarnessErrorCount: harnessErrorCount,
		HasViteOverlay:    hasViteOverlay,
		ViteOverlayText:   viteOverlayText,
	}
}

func WriteDiagnoseArtifacts(report *DiagnoseReport, mode ArtifactMode) error {
	if report == nil {
		return nil
	}
	if mode == ArtifactModeNone {
		return nil
	}
	if strings.TrimSpace(report.Meta.ArtifactDir) == "" {
		return nil
	}
	if err := os.MkdirAll(report.Meta.ArtifactDir, 0o755); err != nil {
		return err
	}

	writeJSON := func(name string, v any) (string, error) {
		path := filepath.Join(report.Meta.ArtifactDir, name)
		b, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			return "", err
		}
		b = append(b, '\n')
		if err := os.WriteFile(path, b, 0o644); err != nil {
			return "", err
		}
		return path, nil
	}

	// Always write report.json when artifacts enabled.
	if p, err := writeJSON("report.json", report); err == nil {
		report.Artifacts.Report = p
	}
	if mode == ArtifactModeMinimal {
		return nil
	}

	if p, err := writeJSON("console.json", report.Console); err == nil {
		report.Artifacts.Console = p
	}
	if p, err := writeJSON("network.json", report.Network); err == nil {
		report.Artifacts.Network = p
	}
	// Snapshot YAML in its own file for human inspection.
	if strings.TrimSpace(report.Snapshot.YAML) != "" {
		path := filepath.Join(report.Meta.ArtifactDir, "snapshot.yaml")
		_ = os.WriteFile(path, []byte(report.Snapshot.YAML), 0o644)
		report.Artifacts.Snapshot = path
	}

	return nil
}
