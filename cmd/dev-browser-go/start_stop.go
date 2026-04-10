package main

import (
	"fmt"

	devbrowser "github.com/joshp123/dev-browser-go/internal/devbrowser"
	"github.com/spf13/cobra"
)

func newStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Start daemon",
		RunE: func(_ *cobra.Command, _ []string) error {
			headless, window, device, err := desiredDaemonSettings()
			if err != nil {
				return err
			}
			result, err := devbrowser.EnsureDaemon(globalOpts.profile, headless, window, device)
			if err != nil {
				return err
			}
			switch result.Action {
			case devbrowser.DaemonActionStarted:
				fmt.Printf("started profile=%s url=%s %s\n", globalOpts.profile, result.BaseURL, contextSummary(result.Context))
			case devbrowser.DaemonActionReconfigured:
				fmt.Printf("restarted profile=%s url=%s %s\n", globalOpts.profile, result.BaseURL, contextSummary(result.Context))
			default:
				fmt.Printf("reused profile=%s url=%s %s\n", globalOpts.profile, result.BaseURL, contextSummary(result.Context))
			}
			return nil
		},
	}
}

func newStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop daemon",
		RunE: func(_ *cobra.Command, _ []string) error {
			stopped, err := devbrowser.StopDaemon(globalOpts.profile)
			if err != nil {
				return err
			}
			if stopped {
				fmt.Printf("stopped profile=%s\n", globalOpts.profile)
				return nil
			}
			fmt.Printf("not running profile=%s\n", globalOpts.profile)
			return nil
		},
	}
}
