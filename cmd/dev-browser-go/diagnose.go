package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"time"

	devbrowser "github.com/joshp123/dev-browser-go/internal/devbrowser"
	"github.com/spf13/cobra"
)

func newDiagnoseCmd() *cobra.Command {
	var pageName string
	var targetURL string
	var waitState string
	var timeoutMs int
	var minWaitMs int
	var snapshotEngine string
	var netBodies bool
	var netMaxBodyBytes int
	var perfSampleMs int
	var perfTopN int
	var artifactMode string
	var artifactDir string

	cmd := &cobra.Command{
		Use:   "diagnose",
		Short: "One-call diagnostic report for agent loops (report-only)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			mode := devbrowser.ArtifactMode(artifactMode)
			if !mode.Valid() {
				return fmt.Errorf("--artifact-mode must be none|minimal|full")
			}
			base, err := startDaemonIfNeeded()
			if err != nil {
				return err
			}

			ws, tid, err := devbrowser.EnsurePage(globalOpts.profile, globalOpts.headless, pageName, globalOpts.window, globalOpts.device)
			if err != nil {
				return err
			}
			pw, browser, page, err := devbrowser.OpenPage(ws, tid)
			if err != nil {
				return err
			}
			defer browser.Close()
			defer pw.Stop()

			ts := time.Now()
			runID := devbrowser.NewDiagnoseRunID()
			root := devbrowser.ArtifactDir(globalOpts.profile)
			runDir := ""
			if mode != devbrowser.ArtifactModeNone {
				if artifactDir != "" {
					// When user passes a relative path, treat it as relative to artifact root.
					runDir, err = devbrowser.SafeArtifactPath(root, artifactDir, "")
					if err != nil {
						return err
					}
				} else {
					runDir = devbrowser.DefaultRunArtifactDir(root, runID, ts)
				}
			}

			report, err := devbrowser.Diagnose(page, devbrowser.DiagnoseOptions{
				URL:             targetURL,
				WaitState:       waitState,
				TimeoutMs:       timeoutMs,
				MinWaitMs:       minWaitMs,
				PageName:        pageName,
				Profile:         globalOpts.profile,
				RunID:           runID,
				Timestamp:       ts,
				ArtifactDir:     runDir,
				Artifacts:       mode,
				SnapshotEngine:  snapshotEngine,
				NetBodies:       netBodies,
				NetMaxBodyBytes: netMaxBodyBytes,
				PerfSampleMs:    perfSampleMs,
				PerfTopN:        perfTopN,
			})
			if err != nil {
				return err
			}

			// Populate console entries from daemon (stable, cross-tool behavior).
			consoleEntries, err := readConsoleEntries(base, pageName, 200)
			if err == nil {
				report.SetConsole(consoleEntries)
			}

			// Write artifacts (best-effort).
			_ = devbrowser.WriteDiagnoseArtifacts(report, mode)

			out, err := devbrowser.WriteOutput(globalOpts.profile, globalOpts.output, report, globalOpts.outPath)
			if err != nil {
				return err
			}
			fmt.Println(out)
			return nil // diagnose is report-only; always exit 0.
		},
	}

	cmd.Flags().StringVar(&targetURL, "url", "", "Optional URL to navigate to")
	cmd.Flags().StringVar(&pageName, "page", "main", "Page name")
	cmd.Flags().StringVar(&waitState, "wait", "networkidle", "Wait state (load|domcontentloaded|networkidle|commit)")
	cmd.Flags().IntVar(&timeoutMs, "timeout-ms", 45_000, "Timeout in ms")
	cmd.Flags().IntVar(&minWaitMs, "min-wait-ms", 250, "Minimum wait time in ms")
	cmd.Flags().StringVar(&snapshotEngine, "snapshot-engine", "simple", "Snapshot engine (simple|aria)")
	cmd.Flags().BoolVar(&netBodies, "net-bodies", false, "Include network bodies")
	cmd.Flags().IntVar(&netMaxBodyBytes, "net-max-body-bytes", 32*1024, "Max network body bytes (when --net-bodies)")
	cmd.Flags().IntVar(&perfSampleMs, "perf-sample-ms", 1200, "Perf metrics sample ms")
	cmd.Flags().IntVar(&perfTopN, "perf-top-n", 20, "Perf metrics top-N resources")
	cmd.Flags().StringVar(&artifactMode, "artifact-mode", string(devbrowser.ArtifactModeMinimal), "Artifacts: none|minimal|full")
	cmd.Flags().StringVar(&artifactDir, "artifact-dir", "", "Artifact directory (relative to artifact root unless absolute). Default: per-run dir")

	return cmd
}

func readConsoleEntries(base, pageName string, limit int) ([]devbrowser.ConsoleEntry, error) {
	endpoint := fmt.Sprintf("%s/pages/%s/console", base, url.PathEscape(pageName))
	q := url.Values{}
	q.Set("limit", strconv.Itoa(limit))
	q.Set("levels", "all")
	endpoint += "?" + q.Encode()
	data, err := devbrowser.HTTPJSON("GET", endpoint, nil, 5*time.Second)
	if err != nil {
		return nil, err
	}
	if ok, _ := data["ok"].(bool); !ok {
		return nil, fmt.Errorf("console failed: %v", data["error"])
	}
	entriesAny, _ := data["entries"].([]interface{})
	b, _ := json.Marshal(entriesAny)
	var entries []devbrowser.ConsoleEntry
	_ = json.Unmarshal(b, &entries)
	return entries, nil
}
