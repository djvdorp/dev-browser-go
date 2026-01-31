package main

import (
	"github.com/spf13/cobra"
)

func newInspectRefCmd() *cobra.Command {
	var pageName string
	var ref string
	var engine string
	var styleProps []string

	cmd := &cobra.Command{
		Use:   "inspect-ref",
		Short: "Inspect a snapshot ref (attrs, selector/xpath, states, bbox)",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			payload := map[string]interface{}{
				"ref":         ref,
				"engine":      engine,
				"style_props": styleProps,
			}
			return runWithPage(pageName, "inspect_ref", payload)
		},
	}

	cmd.Flags().StringVar(&pageName, "page", "main", "Page name")
	cmd.Flags().StringVar(&ref, "ref", "", "Snapshot ref (required)")
	cmd.Flags().StringVar(&engine, "engine", "simple", "Snapshot engine (simple|aria)")
	cmd.Flags().StringSliceVar(&styleProps, "style-prop", nil, "Computed style property to include (repeatable)")
	_ = cmd.MarkFlagRequired("ref")

	return cmd
}
