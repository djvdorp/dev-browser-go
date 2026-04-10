package main

import (
	"fmt"
	"strings"

	devbrowser "github.com/joshp123/dev-browser-go/internal/devbrowser"
	"github.com/spf13/cobra"
)

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show daemon status",
		RunE: func(_ *cobra.Command, _ []string) error {
			health, err := devbrowser.ReadDaemonHealth(globalOpts.profile)
			if err != nil {
				return err
			}
			if health != nil && health.OK {
				pageURL := strings.TrimSpace(health.PageURL)
				if pageURL == "" {
					pageURL = "about:blank"
				}
				fmt.Printf("ok profile=%s url=%s %s page=%s\n", globalOpts.profile, fmt.Sprintf("http://%s:%d", health.Host, health.Port), contextSummary(health.Context), pageURL)
				return nil
			}
			fmt.Printf("not running profile=%s\n", globalOpts.profile)
			return nil
		},
	}
}
