package main

import (
	"errors"
	"strings"

	"github.com/spf13/cobra"
)

func newDomDiffCmd() *cobra.Command {
	var pageName string
	var baseline string
	var engine string
	var maxItems int

	cmd := &cobra.Command{
		Use:   "dom-diff",
		Short: "Compare current DOM snapshot against a baseline",
		Args:  cobra.NoArgs,
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			if strings.TrimSpace(baseline) == "" {
				return errors.New("--baseline is required")
			}
			return nil
		},
		RunE: func(_ *cobra.Command, _ []string) error {
			payload := map[string]interface{}{
				"baseline_path": baseline,
				"engine":        engine,
				"max_items":     maxItems,
			}
			return runWithPage(pageName, "dom_diff", payload)
		},
	}

	cmd.Flags().StringVar(&pageName, "page", "main", "Page name")
	cmd.Flags().StringVar(&baseline, "baseline", "", "Baseline DOM snapshot path (required)")
	cmd.Flags().StringVar(&engine, "engine", "simple", "Snapshot engine (simple|aria)")
	cmd.Flags().IntVar(&maxItems, "max-items", 200, "Max items to capture")
	_ = cmd.MarkFlagRequired("baseline")

	return cmd
}
