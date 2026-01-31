package main

import (
	"github.com/spf13/cobra"
)

func newTestXPathCmd() *cobra.Command {
	var pageName string
	var xpath string
	var engine string

	cmd := &cobra.Command{
		Use:   "test-xpath",
		Short: "Test an XPath expression (count + preview)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			payload := map[string]interface{}{
				"xpath":  xpath,
				"engine": engine,
			}
			return runWithPage(pageName, "test_xpath", payload)
		},
	}

	cmd.Flags().StringVar(&pageName, "page", "main", "Page name")
	cmd.Flags().StringVar(&xpath, "xpath", "", "XPath expression (required)")
	cmd.Flags().StringVar(&engine, "engine", "simple", "Snapshot engine (simple|aria)")
	_ = cmd.MarkFlagRequired("xpath")

	return cmd
}
