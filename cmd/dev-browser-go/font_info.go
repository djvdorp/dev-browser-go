package main

import "github.com/spf13/cobra"

func newFontInfoCmd() *cobra.Command {
	var pageName string
	var ref string
	var engine string

	cmd := &cobra.Command{
		Use:   "font-info",
		Short: "Extract font details for an element (computed styles) via snapshot ref",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			payload := map[string]interface{}{
				"ref":    ref,
				"engine": engine,
			}
			return runWithPage(pageName, "font_info", payload)
		},
	}

	cmd.Flags().StringVar(&pageName, "page", "main", "Page name")
	cmd.Flags().StringVar(&ref, "ref", "", "Snapshot ref (required)")
	cmd.Flags().StringVar(&engine, "engine", "simple", "Snapshot engine (simple|aria)")
	_ = cmd.MarkFlagRequired("ref")
	return cmd
}
