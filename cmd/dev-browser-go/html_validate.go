package main

import (
	"fmt"
	"time"

	devbrowser "github.com/joshp123/dev-browser-go/internal/devbrowser"
	"github.com/spf13/cobra"
)

func newHTMLValidateCmd() *cobra.Command {
	var pageName string
	var targetURL string
	var waitState string
	var timeoutMs int
	var minWaitMs int

	cmd := &cobra.Command{
		Use:   "html-validate",
		Short: "Lite HTML validation (report-only)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
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

			if targetURL != "" {
				if _, err := devbrowser.RunCall(page, "goto", map[string]interface{}{"url": targetURL, "timeout_ms": timeoutMs}, devbrowser.ArtifactDir(globalOpts.profile)); err != nil {
					return err
				}
			}
			_, _ = devbrowser.RunCall(page, "wait", map[string]interface{}{"state": waitState, "timeout_ms": timeoutMs, "min_wait_ms": minWaitMs}, devbrowser.ArtifactDir(globalOpts.profile))

			htmlStr, err := page.Content()
			if err != nil {
				return err
			}
			findings, err := devbrowser.ValidateHTML(htmlStr)
			if err != nil {
				return err
			}

			report := devbrowser.NewHTMLValidateReport(page.URL(), pageName, globalOpts.profile, time.Now(), findings)
			out, err := devbrowser.WriteOutput(globalOpts.profile, globalOpts.output, report, globalOpts.outPath)
			if err != nil {
				return err
			}
			fmt.Println(out)
			return nil // report-only; always exit 0
		},
	}

	cmd.Flags().StringVar(&targetURL, "url", "", "Optional URL to navigate to")
	cmd.Flags().StringVar(&pageName, "page", "main", "Page name")
	cmd.Flags().StringVar(&waitState, "wait", "networkidle", "Wait state (load|domcontentloaded|networkidle|commit)")
	cmd.Flags().IntVar(&timeoutMs, "timeout-ms", 45_000, "Timeout in ms")
	cmd.Flags().IntVar(&minWaitMs, "min-wait-ms", 250, "Minimum wait time in ms")

	return cmd
}
