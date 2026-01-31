package main

import (
	"github.com/spf13/cobra"
)

func newNetworkMonitorCmd() *cobra.Command {
	var pageName string
	var waitState string
	var timeoutMs int
	var minWaitMs int
	var maxEntries int
	var includeBodies bool
	var includeHeaders bool
	var maxBodyBytes int
	var urlContains string
	var method string
	var typ string
	var status int
	var statusMin int
	var statusMax int
	var onlyFailed bool

	cmd := &cobra.Command{
		Use:   "network-monitor",
		Short: "Capture network requests/responses (headers, bodies, filtering)",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			payload := map[string]interface{}{
				"wait_state":      waitState,
				"timeout_ms":      timeoutMs,
				"min_wait_ms":     minWaitMs,
				"max_entries":     maxEntries,
				"include_bodies":  includeBodies,
				"include_headers": includeHeaders,
				"max_body_bytes":  maxBodyBytes,
				"url_contains":    urlContains,
				"method":          method,
				"type":            typ,
				"status":          status,
				"status_min":      statusMin,
				"status_max":      statusMax,
				"only_failed":     onlyFailed,
			}
			return runWithPage(pageName, "network_monitor", payload)
		},
	}

	cmd.Flags().StringVar(&pageName, "page", "main", "Page name")
	cmd.Flags().StringVar(&waitState, "wait", "networkidle", "Wait state (load|domcontentloaded|networkidle|commit)")
	cmd.Flags().IntVar(&timeoutMs, "timeout-ms", 45_000, "Timeout for waiting")
	cmd.Flags().IntVar(&minWaitMs, "min-wait-ms", 0, "Minimum wait before evaluation")

	cmd.Flags().IntVar(&maxEntries, "max-entries", 200, "Max network entries to retain")
	cmd.Flags().BoolVar(&includeHeaders, "headers", true, "Include request/response headers")
	cmd.Flags().BoolVar(&includeBodies, "bodies", false, "Include request/response bodies (can be large)")
	cmd.Flags().IntVar(&maxBodyBytes, "max-body-bytes", 64*1024, "Max bytes per body before truncation")

	cmd.Flags().StringVar(&urlContains, "url-contains", "", "Filter: URL contains substring")
	cmd.Flags().StringVar(&method, "method", "", "Filter: HTTP method equals")
	cmd.Flags().StringVar(&typ, "type", "", "Filter: resource type equals")
	cmd.Flags().IntVar(&status, "status", 0, "Filter: status equals")
	cmd.Flags().IntVar(&statusMin, "status-min", 0, "Filter: status >=")
	cmd.Flags().IntVar(&statusMax, "status-max", 0, "Filter: status <=")
	cmd.Flags().BoolVar(&onlyFailed, "failed", false, "Only include failed (non-2xx/3xx or request failed)")

	return cmd
}
