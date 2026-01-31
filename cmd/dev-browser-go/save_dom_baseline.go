package main

import (
	"errors"
	"strings"

	"github.com/spf13/cobra"
)

func newSaveDomBaselineCmd() *cobra.Command {
	var pageName string
	var path string
	var engine string
	var maxItems int

	cmd := &cobra.Command{
		Use:   "save-dom-baseline",
		Short: "Save a DOM snapshot baseline (for structural diffs)",
		Args:  cobra.NoArgs,
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			if strings.TrimSpace(path) == "" {
				return errors.New("--path is required")
			}
			return cmd.ValidateRequiredFlags()
		},
		RunE: func(_ *cobra.Command, _ []string) error {
			payload := map[string]interface{}{
				"path":      path,
				"engine":    engine,
				"max_items": maxItems,
			}
			return runWithPage(pageName, "save_dom_baseline", payload)
		},
	}

	cmd.Flags().StringVar(&pageName, "page", "main", "Page name")
	cmd.Flags().StringVar(&path, "path", "", "Output baseline path (required)")
	cmd.Flags().StringVar(&engine, "engine", "simple", "Snapshot engine (simple|aria)")
	cmd.Flags().IntVar(&maxItems, "max-items", 200, "Max items to capture")
	_ = cmd.MarkFlagRequired("path")

	return cmd
}
