package main

import (
	"strings"

	"github.com/spf13/cobra"
)

func newStyleCaptureCmd() *cobra.Command {
	var pageName string
	var pathArg string
	var cssPath string
	var mode string
	var selector string
	var maxNodes int
	var includeAll bool
	var properties string
	var strip bool

	cmd := &cobra.Command{
		Use:   "style-capture",
		Short: "Capture computed styles and inline/bundle CSS",
		Args:  cobra.NoArgs,
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			return applyNoFlag(cmd, "strip")
		},
		RunE: func(_ *cobra.Command, _ []string) error {
			payload := map[string]interface{}{
				"mode":        mode,
				"max_nodes":   maxNodes,
				"include_all": includeAll,
				"strip":       strip,
			}
			if strings.TrimSpace(pathArg) != "" {
				payload["path"] = pathArg
			}
			if strings.TrimSpace(cssPath) != "" {
				payload["css_path"] = cssPath
			}
			if strings.TrimSpace(selector) != "" {
				payload["selector"] = selector
			}
			if strings.TrimSpace(properties) != "" {
				payload["properties"] = splitCommaList(properties)
			}
			return runWithPage(pageName, "style_capture", payload)
		},
	}

	cmd.Flags().StringVar(&pageName, "page", "main", "Page name")
	cmd.Flags().StringVar(&pathArg, "path", "", "Output HTML path")
	cmd.Flags().StringVar(&cssPath, "css-path", "", "Output CSS path (bundle mode only)")
	cmd.Flags().StringVar(&mode, "mode", "inline", "Mode (inline|bundle)")
	cmd.Flags().StringVar(&selector, "selector", "", "CSS selector to scope capture")
	cmd.Flags().IntVar(&maxNodes, "max-nodes", 1500, "Max elements to inline")
	cmd.Flags().BoolVar(&includeAll, "include-all", false, "Include all computed properties")
	cmd.Flags().StringVar(&properties, "properties", "", "Comma-separated CSS properties")
	cmd.Flags().BoolVar(&strip, "strip", true, "Strip scripts/styles/links")
	cmd.Flags().Bool("no-strip", false, "Keep scripts/styles/links")

	return cmd
}

func splitCommaList(value string) []string {
	raw := strings.Split(value, ",")
	out := make([]string, 0, len(raw))
	for _, entry := range raw {
		if trimmed := strings.TrimSpace(entry); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
