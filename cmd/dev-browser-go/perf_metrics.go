package main

import (
	"github.com/spf13/cobra"
)

func newPerfMetricsCmd() *cobra.Command {
	var pageName string
	var sampleMs int
	var topN int

	cmd := &cobra.Command{
		Use:   "perf-metrics",
		Short: "Collect performance metrics (timing, resources, CWV best-effort, FPS sample)",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			payload := map[string]interface{}{
				"sample_ms": sampleMs,
				"top_n":     topN,
			}
			return runWithPage(pageName, "perf_metrics", payload)
		},
	}

	cmd.Flags().StringVar(&pageName, "page", "main", "Page name")
	cmd.Flags().IntVar(&sampleMs, "sample-ms", 1200, "FPS sample window in ms")
	cmd.Flags().IntVar(&topN, "top-n", 20, "Top N slowest resources to include")

	return cmd
}
