package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newVisualDiffCmd() *cobra.Command {
	var pageName string
	var baseline string
	var output string
	var tolerance float64
	var pixelThreshold int
	var highlight bool
	var ignoreRegions string

	cmd := &cobra.Command{
		Use:   "visual-diff",
		Short: "Compare current page screenshot against baseline",
		Args:  cobra.NoArgs,
		PreRunE: func(_ *cobra.Command, _ []string) error {
			if strings.TrimSpace(baseline) == "" {
				return errors.New("baseline path is required")
			}
			return nil
		},
		RunE: func(_ *cobra.Command, _ []string) error {
			payload := map[string]interface{}{
				"baseline_path":      baseline,
				"tolerance":          tolerance,
				"pixel_threshold":    pixelThreshold,
				"highlight":          highlight,
				"diff_output_format": "simple",
			}
			if strings.TrimSpace(output) != "" {
				payload["output_path"] = output
			}
			if strings.TrimSpace(ignoreRegions) != "" {
				payload["ignore_regions"] = parseIgnoreRegions(ignoreRegions)
			}
			return runWithPage(pageName, "visual_diff", payload)
		},
	}

	cmd.Flags().StringVar(&pageName, "page", "main", "Page name")
	cmd.Flags().StringVar(&baseline, "baseline", "", "Baseline screenshot path (required)")
	cmd.Flags().StringVar(&output, "output", "", "Output diff image path")
	cmd.Flags().Float64Var(&tolerance, "tolerance", 0.1, "Color tolerance (0.0-1.0)")
	cmd.Flags().IntVar(&pixelThreshold, "pixel-threshold", 10, "Max different pixels before fail")
	cmd.Flags().BoolVar(&highlight, "highlight", true, "Highlight differences in output")
	cmd.Flags().StringVar(&ignoreRegions, "ignore", "", "Ignore regions (x,y,w,h;x,y,w,h)")

	return cmd
}

func parseIgnoreRegions(value string) []map[string]int {
	raw := strings.Split(value, ";")
	regions := make([]map[string]int, 0, len(raw))
	for _, r := range raw {
		parts := strings.Split(strings.TrimSpace(r), ",")
		if len(parts) != 4 {
			continue
		}
		region := make(map[string]int)
		region["x"] = parseIntOrZero(parts[0])
		region["y"] = parseIntOrZero(parts[1])
		region["w"] = parseIntOrZero(parts[2])
		region["h"] = parseIntOrZero(parts[3])
		regions = append(regions, region)
	}
	return regions
}

func parseIntOrZero(s string) int {
	var n int
	if _, err := fmt.Sscanf(strings.TrimSpace(s), "%d", &n); err != nil {
		return 0
	}
	return n
}
