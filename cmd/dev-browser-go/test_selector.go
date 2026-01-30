package main

import (
	"github.com/spf13/cobra"
)

func newTestSelectorCmd() *cobra.Command {
	var pageName string
	var selector string
	var engine string

	cmd := &cobra.Command{
		Use:   "test-selector",
		Short: "Test a CSS selector (count + preview)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			_ = cmd.MarkFlagRequired("selector")
			payload := map[string]interface{}{
				"selector": selector,
				"engine":   engine,
			}
			return runWithPage(pageName, "test_selector", payload)
		},
	}

	cmd.Flags().StringVar(&pageName, "page", "main", "Page name")
	cmd.Flags().StringVar(&selector, "selector", "", "CSS selector (required)")
	cmd.Flags().StringVar(&engine, "engine", "simple", "Snapshot engine (simple|aria)")
	_ = cmd.MarkFlagRequired("selector")

	return cmd
}
