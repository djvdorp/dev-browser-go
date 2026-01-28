package main

import (
	"errors"
	"strings"

	"github.com/spf13/cobra"
)

func newInjectCmd() *cobra.Command {
	var pageName string
	var script string
	var style string
	var file string
	var wait int

	cmd := &cobra.Command{
		Use:   "inject",
		Short: "Inject JavaScript or CSS into page",
		Args:  cobra.NoArgs,
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			scriptSet := cmd.Flags().Changed("script") || strings.TrimSpace(script) != ""
			styleSet := cmd.Flags().Changed("style") || strings.TrimSpace(style) != ""
			fileSet := cmd.Flags().Changed("file") || strings.TrimSpace(file) != ""

			if !scriptSet && !styleSet && !fileSet {
				return errors.New("one of --script, --style, or --file is required")
			}
			if scriptSet && styleSet {
				return errors.New("--script and --style cannot be used together")
			}
			if fileSet && (scriptSet || styleSet) {
				return errors.New("--file cannot be used with --script or --style")
			}
			if wait < 0 {
				return errors.New("--wait-ms must be >= 0")
			}
			return nil
		},
		RunE: func(_ *cobra.Command, _ []string) error {
			payload := map[string]interface{}{
				"wait_ms": wait,
			}
			if strings.TrimSpace(script) != "" {
				payload["script"] = script
			}
			if strings.TrimSpace(style) != "" {
				payload["style"] = style
			}
			if strings.TrimSpace(file) != "" {
				payload["file"] = file
			}
			return runWithPage(pageName, "inject", payload)
		},
	}

	cmd.Flags().StringVar(&pageName, "page", "main", "Page name")
	cmd.Flags().StringVar(&script, "script", "", "JavaScript to inject")
	cmd.Flags().StringVar(&style, "style", "", "CSS to inject")
	cmd.Flags().StringVar(&file, "file", "", "Path to JS or CSS file to inject")
	cmd.Flags().IntVar(&wait, "wait-ms", 100, "Wait time in ms after injection")

	return cmd
}
