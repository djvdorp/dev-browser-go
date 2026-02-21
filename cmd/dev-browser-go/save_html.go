package main

import (
	"strings"

	"github.com/spf13/cobra"
)

// buildSaveHTMLPayload constructs the daemon payload for save_html.
// When path is empty or whitespace-only the "path" key is omitted,
// letting the daemon choose the default artifact filename.
func buildSaveHTMLPayload(path string) map[string]interface{} {
	payload := map[string]interface{}{}
	if trimmed := strings.TrimSpace(path); trimmed != "" {
		payload["path"] = trimmed
	}
	return payload
}

func newSaveHTMLCmd() *cobra.Command {
	var pageName string
	var pathArg string

	cmd := &cobra.Command{
		Use:   "save-html",
		Short: "Save page HTML",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runWithPage(pageName, "save_html", buildSaveHTMLPayload(pathArg))
		},
	}

	cmd.Flags().StringVar(&pageName, "page", "main", "Page name")
	cmd.Flags().StringVar(&pathArg, "path", "", "Output path")

	return cmd
}
