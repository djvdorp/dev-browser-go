package main

import "github.com/spf13/cobra"

func newColorInfoCmd() *cobra.Command {
	var pageName string
	var ref string
	var engine string
	var includeTransparent bool

	cmd := &cobra.Command{
		Use:   "color-info",
		Short: "Extract colors for an element (computed styles -> rgb/hex) via snapshot ref",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			payload := map[string]interface{}{
				"ref":                 ref,
				"engine":              engine,
				"include_transparent": includeTransparent,
			}
			return runWithPage(pageName, "color_info", payload)
		},
	}

	cmd.Flags().StringVar(&pageName, "page", "main", "Page name")
	cmd.Flags().StringVar(&ref, "ref", "", "Snapshot ref (required)")
	cmd.Flags().StringVar(&engine, "engine", "simple", "Snapshot engine (simple|aria)")
	cmd.Flags().BoolVar(&includeTransparent, "include-transparent", false, "Include fully transparent colors (alpha=0)")
	_ = cmd.MarkFlagRequired("ref")
	return cmd
}
