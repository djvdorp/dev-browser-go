package main

import (
	"strings"

	"github.com/spf13/cobra"
)

func newDiffImagesCmd() *cobra.Command {
	var pageName string
	var beforePath string
	var afterPath string
	var diffPath string
	var fullPage bool
	var annotate bool
	var crop string
	var selector string
	var ariaRole string
	var ariaName string
	var nth int
	var padding int
	var timeout int
	var afterWait int
	var threshold int

	cmd := &cobra.Command{
		Use:   "diff-images",
		Short: "Capture before/after screenshots and diff",
		Args:  cobra.NoArgs,
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			return applyNoFlag(cmd, "full-page")
		},
		RunE: func(_ *cobra.Command, _ []string) error {
			payload := map[string]interface{}{
				"full_page":     fullPage,
				"annotate_refs": annotate,
				"nth":           nth,
				"padding_px":    padding,
				"timeout_ms":    timeout,
				"after_wait_ms": afterWait,
				"threshold":     threshold,
			}
			if strings.TrimSpace(beforePath) != "" {
				payload["before_path"] = beforePath
			}
			if strings.TrimSpace(afterPath) != "" {
				payload["after_path"] = afterPath
			}
			if strings.TrimSpace(diffPath) != "" {
				payload["diff_path"] = diffPath
			}
			if strings.TrimSpace(crop) != "" {
				payload["crop"] = crop
			}
			if strings.TrimSpace(selector) != "" {
				payload["selector"] = selector
			}
			if strings.TrimSpace(ariaRole) != "" {
				payload["aria_role"] = ariaRole
			}
			if strings.TrimSpace(ariaName) != "" {
				payload["aria_name"] = ariaName
			}
			return runWithPage(pageName, "diff_images", payload)
		},
	}

	cmd.Flags().StringVar(&pageName, "page", "main", "Page name")
	cmd.Flags().StringVar(&beforePath, "before", "", "Before image path (existing or capture target)")
	cmd.Flags().StringVar(&afterPath, "after", "", "After image path (existing or capture target)")
	cmd.Flags().StringVar(&diffPath, "diff-path", "", "Diff image output path")
	cmd.Flags().BoolVar(&fullPage, "full-page", true, "Full page")
	cmd.Flags().BoolVar(&annotate, "annotate-refs", false, "Annotate refs")
	cmd.Flags().StringVar(&crop, "crop", "", "Crop x,y,w,h")
	cmd.Flags().StringVar(&selector, "selector", "", "CSS selector for element crop")
	cmd.Flags().StringVar(&ariaRole, "aria-role", "", "ARIA role for element crop")
	cmd.Flags().StringVar(&ariaName, "aria-name", "", "ARIA name for element crop")
	cmd.Flags().IntVar(&nth, "nth", 1, "Nth match (1-based)")
	cmd.Flags().IntVar(&padding, "padding-px", 10, "Padding around element in px")
	cmd.Flags().IntVar(&timeout, "timeout-ms", 5_000, "Timeout ms for element wait")
	cmd.Flags().IntVar(&afterWait, "after-wait-ms", 0, "Wait between before/after capture (ms)")
	cmd.Flags().IntVar(&threshold, "threshold", 0, "Per-channel diff threshold (0-255)")
	cmd.Flags().Bool("no-full-page", false, "Disable full page")

	return cmd
}
