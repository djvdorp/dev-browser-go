package main

import (
	"fmt"

	devbrowser "github.com/joshp123/dev-browser-go/internal/devbrowser"
	"github.com/spf13/cobra"
)

func newDevicesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "devices",
		Short: "List device profile names",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			devices, err := devbrowser.ListDeviceNames()
			if err != nil {
				return err
			}
			out, err := devbrowser.WriteOutput(globalOpts.profile, globalOpts.output, map[string]any{"devices": devices}, globalOpts.outPath)
			if err != nil {
				return err
			}
			fmt.Println(out)
			return nil
		},
	}
}
