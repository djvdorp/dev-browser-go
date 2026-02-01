package main

import (
	"fmt"
	"strings"
	"time"

	devbrowser "github.com/joshp123/dev-browser-go/internal/devbrowser"
	"github.com/spf13/cobra"
)

func newAssertCmd() *cobra.Command {
	var pageName string
	var targetURL string
	var rulesRaw string
	var waitState string
	var timeoutMs int
	var minWaitMs int
	var artifactMode string

	cmd := &cobra.Command{
		Use:   "assert",
		Short: "Deterministic gating (exit 0 pass, 2 fail)",
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
			ctx := devbrowser.NewRunContext(devbrowser.RunOptions{
				Profile:      globalOpts.profile,
				ArtifactRoot: devbrowser.ArtifactDir(globalOpts.profile),
				Timestamp:    ts,
			})
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
				// keep diagnose defaults for other collection
			})
			if err != nil {
				return err
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

			out, err := devbrowser.WriteOutput(globalOpts.profile, globalOpts.output, result, globalOpts.outPath)
			if err != nil {
				return err
			}
			fmt.Println(out)

			if result.Passed {
				return nil
			}
			return devbrowser.ExitCodeError{Code: 2}
		},
	}

	cmd.Flags().StringVar(&targetURL, "url", "", "Optional URL to navigate to")
	cmd.Flags().StringVar(&pageName, "page", "main", "Page name")
	cmd.Flags().StringVar(&waitState, "wait", "networkidle", "Wait state (load|domcontentloaded|networkidle|commit)")
	cmd.Flags().IntVar(&timeoutMs, "timeout-ms", 45_000, "Timeout in ms")
	cmd.Flags().IntVar(&minWaitMs, "min-wait-ms", 250, "Minimum wait time in ms")
	cmd.Flags().StringVar(&rulesRaw, "rules", "", "Rules JSON string, or @path/to/rules.json")
	cmd.Flags().StringVar(&artifactMode, "artifact-mode", string(devbrowser.ArtifactModeMinimal), "Artifacts: none|minimal|full")

	_ = cmd.MarkFlagRequired("rules")
	return cmd
}
