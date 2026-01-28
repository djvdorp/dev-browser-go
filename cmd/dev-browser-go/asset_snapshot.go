package main

import (
	"strings"

	"github.com/spf13/cobra"
)

func newAssetSnapshotCmd() *cobra.Command {
	var pageName string
	var pathArg string
	var includeAssets bool
	var assetTypes string
	var maxDepth int
	var stripScripts bool
	var inlineThreshold int

	cmd := &cobra.Command{
		Use:   "asset-snapshot",
		Short: "Save HTML with linked assets for offline review",
		Args:  cobra.NoArgs,
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			return applyNoFlag(cmd, "include-assets")
		},
		RunE: func(_ *cobra.Command, _ []string) error {
			payload := map[string]interface{}{
				"include_assets":   includeAssets,
				"max_depth":        maxDepth,
				"strip_scripts":    stripScripts,
				"inline_threshold": inlineThreshold,
			}
			if strings.TrimSpace(pathArg) != "" {
				payload["path"] = pathArg
			}
			if strings.TrimSpace(assetTypes) != "" {
				payload["asset_types"] = splitCommaList(assetTypes)
			}
			return runWithPage(pageName, "asset_snapshot", payload)
		},
	}

	cmd.Flags().StringVar(&pageName, "page", "main", "Page name")
	cmd.Flags().StringVar(&pathArg, "path", "", "Output HTML path")
	cmd.Flags().BoolVar(&includeAssets, "include-assets", true, "Discover and keep asset links")
	cmd.Flags().Bool("no-include-assets", false, "Skip asset discovery")
	cmd.Flags().StringVar(&assetTypes, "asset-types", "css,js,font,image", "Asset types to download")
	cmd.Flags().IntVar(&maxDepth, "max-depth", 2, "Max depth for asset discovery")
	cmd.Flags().BoolVar(&stripScripts, "strip-scripts", false, "Remove script tags")
	cmd.Flags().IntVar(&inlineThreshold, "inline-threshold", 10240, "Inline assets under this size (bytes)")

	return cmd
}
