package main

import (
	"errors"
	"strings"

	"github.com/spf13/cobra"
)

func newSaveBaselineCmd() *cobra.Command {
	var pageName string
	var pathArg string
	var fullPage bool
	var selector string
	var ariaRole string
	var ariaName string
	var nth int
	var padding int
	var timeout int

	cmd := &cobra.Command{
		Use:   "save-baseline",
		Short: "Save current page state as visual baseline",
		Args:  cobra.NoArgs,
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			if err := applyNoFlag(cmd, "full-page"); err != nil {
				return err
			}
			if strings.TrimSpace(pathArg) == "" {
				return errors.New("--path is required")
			}
			if padding < 0 {
				return errors.New("--padding-px must be >= 0")
			}
			if timeout < 0 {
				return errors.New("--timeout-ms must be >= 0")
			}
			return nil
		},
		RunE: func(_ *cobra.Command, _ []string) error {
			payload := map[string]interface{}{
				"full_page":   fullPage,
				"nth":         nth,
				"padding_px":  padding,
				"timeout_ms":  timeout,
				"is_baseline": true,
				"path":        pathArg,
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
			return runWithPage(pageName, "save_baseline", payload)
		},
	}

	cmd.Flags().StringVar(&pageName, "page", "main", "Page name")
	cmd.Flags().StringVar(&pathArg, "path", "", "Output path (required)")
	cmd.Flags().BoolVar(&fullPage, "full-page", true, "Full page screenshot")
	cmd.Flags().Bool("no-full-page", false, "Disable full page screenshot")
	cmd.Flags().StringVar(&selector, "selector", "", "CSS selector for element capture")
	cmd.Flags().StringVar(&ariaRole, "aria-role", "", "ARIA role for element capture")
	cmd.Flags().StringVar(&ariaName, "aria-name", "", "ARIA name for element capture")
	cmd.Flags().IntVar(&nth, "nth", 1, "Nth match (1-based)")
	cmd.Flags().IntVar(&padding, "padding-px", 10, "Padding around element")
	cmd.Flags().IntVar(&timeout, "timeout-ms", 5_000, "Timeout for element wait")

	return cmd
}
