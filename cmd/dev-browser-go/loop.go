package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	devbrowser "github.com/joshp123/dev-browser-go/internal/devbrowser"
	"github.com/spf13/cobra"
)

type LoopOutput struct {
	RunID       string                     `json:"runId"`
	ArtifactDir string                     `json:"artifactDir,omitempty"`
	Summary     devbrowser.DiagnoseSummary `json:"summary"`
	Assert      devbrowser.AssertResult    `json:"assert"`
}

func newLoopCmd() *cobra.Command {
	var pageName string
	var targetURL string
	var rulesRaw string
	var waitState string
	var timeoutMs int
	var minWaitMs int
	var artifactMode string

	var watch bool
	var watchIntervalMs int
	var watchPathsRaw string

	cmd := &cobra.Command{
		Use:   "loop",
		Short: "Run diagnose+assert once (or watch) for agent/dev loops",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			rules, err := devbrowser.ParseAssertRules(rulesRaw)
			if err != nil {
				return err
			}
			mode := devbrowser.ArtifactMode(artifactMode)
			if !mode.Valid() {
				return fmt.Errorf("--artifact-mode must be none|minimal|full")
			}

			watchPaths := []string{}
			for _, p := range strings.Split(watchPathsRaw, ",") {
				p = strings.TrimSpace(p)
				if p != "" {
					watchPaths = append(watchPaths, p)
				}
			}
			if watch && len(watchPaths) == 0 {
				watchPaths = []string{"."}
			}
			if watchIntervalMs <= 0 {
				watchIntervalMs = 750
			}

			runOnce := func() (devbrowser.AssertResult, devbrowser.DiagnoseSummary, string, string, error) {
				base, err := startDaemonIfNeeded()
				if err != nil {
					return devbrowser.AssertResult{}, devbrowser.DiagnoseSummary{}, "", "", err
				}

				ws, tid, err := devbrowser.EnsurePage(globalOpts.profile, globalOpts.headless, pageName, globalOpts.window, globalOpts.device)
				if err != nil {
					return devbrowser.AssertResult{}, devbrowser.DiagnoseSummary{}, "", "", err
				}
				pw, browser, page, err := devbrowser.OpenPage(ws, tid)
				if err != nil {
					return devbrowser.AssertResult{}, devbrowser.DiagnoseSummary{}, "", "", err
				}
				defer browser.Close()
				defer pw.Stop()

				ts := time.Now()
				ctx := devbrowser.NewRunContext(devbrowser.RunOptions{Profile: globalOpts.profile, Timestamp: ts})
				runID := ctx.RunID
				runDir := ""
				if mode != devbrowser.ArtifactModeNone {
					runDir = ctx.DefaultRunDir()
					_ = ctx.EnsureDir(runDir)
				}

				report, err := devbrowser.Diagnose(page, devbrowser.DiagnoseOptions{
					URL:         targetURL,
					WaitState:   waitState,
					TimeoutMs:   timeoutMs,
					MinWaitMs:   minWaitMs,
					PageName:    pageName,
					Profile:     globalOpts.profile,
					RunID:       runID,
					Timestamp:   ts,
					ArtifactDir: runDir,
					Artifacts:   mode,
				})
				if err != nil {
					return devbrowser.AssertResult{}, devbrowser.DiagnoseSummary{}, runID, runDir, err
				}

				consoleEntries, err := readConsoleEntries(base, pageName, 200)
				if err == nil {
					report.SetConsole(consoleEntries)
				}

				selectorCounts := map[string]int{}
				selectorEvalErr := map[string]string{}
				for _, sel := range rules.Selectors {
					count, err := devbrowser.CountSelector(page, sel.Selector)
					if err != nil {
						selectorEvalErr[sel.Selector] = err.Error()
						count = 0
					}
					selectorCounts[sel.Selector] = count
				}

				result := devbrowser.EvaluateAssert(report, rules, selectorCounts, nil)

				// Attach deterministic previews for failed selector checks (best-effort).
				for i := range result.FailedChecks {
					id := result.FailedChecks[i].ID
					if id != "selectors.min" && id != "selectors.max" {
						continue
					}
					ctx := result.FailedChecks[i].Context
					if ctx == nil {
						ctx = map[string]any{}
					}
					selRaw, _ := ctx["selector"].(string)
					selStr := strings.TrimSpace(selRaw)
					if selStr == "" {
						continue
					}
					if errMsg, ok := selectorEvalErr[selStr]; ok {
						ctx["evalError"] = errMsg
					}
					preview, err := devbrowser.SelectorPreview(page, selStr, devbrowser.SelectorPreviewOptions{Limit: 5})
					if err == nil {
						ctx["preview"] = preview
					}
					result.FailedChecks[i].Context = ctx
				}

				_ = devbrowser.WriteDiagnoseArtifacts(report, mode)
				_, _ = devbrowser.WriteAssertArtifacts(runDir, result, mode)

				return result, report.Summary, runID, runDir, nil
			}

			lastStamp := int64(0)
			if watch {
				lastStamp = watchStamp(watchPaths)
			}

			for {
				result, summary, runID, runDir, err := runOnce()
				if err != nil {
					return err
				}

				outObj := LoopOutput{RunID: runID, ArtifactDir: runDir, Summary: summary, Assert: result}
				if err := writeLoopOutput(outObj); err != nil {
					return err
				}

				if !watch {
					if result.Passed {
						return nil
					}
					return devbrowser.ExitCodeError{Code: 2}
				}

				// Watch mode: wait for changes.
				deadline := time.Now().Add(time.Duration(watchIntervalMs) * time.Millisecond)
				for time.Now().Before(deadline) {
					time.Sleep(50 * time.Millisecond)
				}
				cur := watchStamp(watchPaths)
				if cur == lastStamp {
					continue
				}
				lastStamp = cur
			}
		},
	}

	cmd.Flags().StringVar(&targetURL, "url", "", "Optional URL to navigate to")
	cmd.Flags().StringVar(&pageName, "page", "main", "Page name")
	cmd.Flags().StringVar(&waitState, "wait", "networkidle", "Wait state (load|domcontentloaded|networkidle|commit)")
	cmd.Flags().IntVar(&timeoutMs, "timeout-ms", 45_000, "Timeout in ms")
	cmd.Flags().IntVar(&minWaitMs, "min-wait-ms", 250, "Minimum wait time in ms")
	cmd.Flags().StringVar(&rulesRaw, "rules", "", "Rules JSON string, or @path/to/rules.json")
	cmd.Flags().StringVar(&artifactMode, "artifact-mode", string(devbrowser.ArtifactModeMinimal), "Artifacts: none|minimal|full")

	cmd.Flags().BoolVar(&watch, "watch", false, "Watch for changes and re-run")
	cmd.Flags().IntVar(&watchIntervalMs, "watch-interval-ms", 750, "Watch poll interval in ms")
	cmd.Flags().StringVar(&watchPathsRaw, "watch-paths", ".", "Comma-separated paths to watch (files or dirs)")

	_ = cmd.MarkFlagRequired("rules")
	return cmd
}

func writeLoopOutput(obj LoopOutput) error {
	switch globalOpts.output {
	case "json":
		b, err := json.MarshalIndent(obj, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(b))
		return nil
	case "summary":
		if obj.Assert.Passed {
			fmt.Printf("PASS runId=%s\n", obj.RunID)
			return nil
		}
		reason := "failed"
		if obj.Summary.HasViteOverlay {
			reason = "vite-overlay"
		} else if obj.Summary.HasHarnessErrors {
			reason = "harness-error"
		} else if obj.Summary.HasConsoleErrors {
			reason = "console-error"
		} else if obj.Summary.HasFailedRequests {
			reason = "network-failed"
		} else if obj.Summary.HasHttp4xx5xx {
			reason = "network-4xx5xx"
		}
		fmt.Printf("FAIL(%s) runId=%s checks=%d\n", reason, obj.RunID, len(obj.Assert.FailedChecks))
		if obj.Summary.ViteOverlayTopLine != "" {
			fmt.Printf("vite: %s\n", obj.Summary.ViteOverlayTopLine)
		}
		if obj.Summary.HarnessErrorTopLine != "" {
			fmt.Printf("error: %s\n", obj.Summary.HarnessErrorTopLine)
		}
		return nil
	case "path":
		path := globalOpts.outPath
		if strings.TrimSpace(path) == "" {
			path = fmt.Sprintf("loop-%d.json", devbrowser.NowMS())
		}
		p, err := devbrowser.SafeArtifactPath(devbrowser.ArtifactDir(globalOpts.profile), path, fmt.Sprintf("loop-%d.json", devbrowser.NowMS()))
		if err != nil {
			return err
		}
		b, err := json.MarshalIndent(obj, "", "  ")
		if err != nil {
			return err
		}
		b = append(b, '\n')
		if err := os.WriteFile(p, b, 0o644); err != nil {
			return err
		}
		fmt.Println(p)
		return nil
	default:
		// fallback: summary
		b, _ := json.Marshal(obj)
		fmt.Println(string(b))
		return nil
	}
}

func watchStamp(paths []string) int64 {
	var max int64
	for _, p := range paths {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		info, err := os.Stat(p)
		if err != nil {
			continue
		}
		if !info.IsDir() {
			ts := info.ModTime().UnixNano()
			if ts > max {
				max = ts
			}
			continue
		}
		_ = filepath.WalkDir(p, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			name := d.Name()
			if d.IsDir() {
				if name == ".git" || name == "node_modules" || name == "dist" || name == "build" {
					return filepath.SkipDir
				}
				return nil
			}
			if strings.HasPrefix(name, ".") {
				return nil
			}
			info, err := d.Info()
			if err != nil {
				return nil
			}
			ts := info.ModTime().UnixNano()
			if ts > max {
				max = ts
			}
			return nil
		})
	}
	return max
}
