package main

import (
	"errors"
	"os"
	"strconv"
	"strings"

	devbrowser "github.com/joshp123/dev-browser-go/internal/devbrowser"
	"github.com/spf13/cobra"
)

type globalOptions struct {
	profile     string
	headless    bool
	headed      bool
	output      string
	outPath     string
	windowSize  string
	windowScale float64
	window      *devbrowser.WindowSize
	device      string
}

var globalOpts = &globalOptions{}

func bindGlobalFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(&globalOpts.profile, "profile", getenvDefault("DEV_BROWSER_PROFILE", "default"), "Browser profile")
	cmd.PersistentFlags().BoolVar(&globalOpts.headless, "headless", defaultHeadless(), "Force headless")
	cmd.PersistentFlags().BoolVar(&globalOpts.headed, "headed", false, "Disable headless")
	cmd.PersistentFlags().StringVar(&globalOpts.windowSize, "window-size", getenvDefault("DEV_BROWSER_WINDOW_SIZE", ""), "Viewport WxH")
	cmd.PersistentFlags().Float64Var(&globalOpts.windowScale, "window-scale", 1.0, "Viewport scale (1, 0.75, 0.5)")
	cmd.PersistentFlags().StringVar(&globalOpts.device, "device", "", "Device profile name (Playwright)")
	cmd.PersistentFlags().StringVar(&globalOpts.output, "output", "summary", "Output format (summary|json|path)")
	cmd.PersistentFlags().StringVar(&globalOpts.outPath, "out", "", "Output path when --output=path")
}

func applyGlobalOptions(cmd *cobra.Command) error {
	if err := resolveHeadless(cmd); err != nil {
		return err
	}
	if err := resolveDevice(cmd); err != nil {
		return err
	}
	if err := resolveWindow(cmd); err != nil {
		return err
	}
	if globalOpts.output != "summary" && globalOpts.output != "json" && globalOpts.output != "path" {
		return errors.New("--output must be summary|json|path")
	}
	return nil
}

func resolveHeadless(cmd *cobra.Command) error {
	headlessChanged := flagChanged(cmd, "headless")
	headedChanged := flagChanged(cmd, "headed")
	if headedChanged && headlessChanged {
		return errors.New("use either --headless or --headed")
	}
	if headedChanged {
		globalOpts.headless = false
		return nil
	}
	if headlessChanged && !globalOpts.headless {
		return errors.New("use --headed to disable headless")
	}
	return nil
}

func resolveWindow(cmd *cobra.Command) error {
	windowScaleChanged := flagChanged(cmd, "window-scale")
	windowSizeChanged := flagChanged(cmd, "window-size")
	if strings.TrimSpace(globalOpts.device) != "" {
		if windowScaleChanged || windowSizeChanged {
			return errors.New("use either --device or --window-size/--window-scale")
		}
		globalOpts.window = nil
		return nil
	}
	if strings.TrimSpace(globalOpts.windowSize) != "" && windowScaleChanged {
		return errors.New("use either --window-size or --window-scale")
	}
	scaleVal := 1.0
	if windowScaleChanged {
		scaleVal = globalOpts.windowScale
	}
	window, err := devbrowser.ResolveWindowSize(globalOpts.windowSize, scaleVal)
	if err != nil {
		return err
	}
	globalOpts.window = window
	return nil
}

func resolveDevice(cmd *cobra.Command) error {
	globalOpts.device = strings.TrimSpace(globalOpts.device)
	if flagChanged(cmd, "device") && globalOpts.device == "" {
		return errors.New("--device requires a non-empty value")
	}
	return nil
}

func flagChanged(cmd *cobra.Command, name string) bool {
	if flag := cmd.Flags().Lookup(name); flag != nil {
		return flag.Changed
	}
	if flag := cmd.PersistentFlags().Lookup(name); flag != nil {
		return flag.Changed
	}
	if flag := cmd.InheritedFlags().Lookup(name); flag != nil {
		return flag.Changed
	}
	return false
}

func defaultHeadless() bool {
	if strings.TrimSpace(os.Getenv("HEADLESS")) == "" {
		return true
	}
	return envTruthy("HEADLESS")
}

func envTruthy(name string) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(name)))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

func getenvDefault(name, def string) string {
	v := strings.TrimSpace(os.Getenv(name))
	if v == "" {
		return def
	}
	return v
}

func getenvInt(name string, def int) int {
	v := strings.TrimSpace(os.Getenv(name))
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func requireArgs(count int, errMsg string) cobra.PositionalArgs {
	return func(_ *cobra.Command, args []string) error {
		if len(args) != count {
			return errors.New(errMsg)
		}
		return nil
	}
}

func maxArgs(max int, errMsg string) cobra.PositionalArgs {
	return func(_ *cobra.Command, args []string) error {
		if len(args) > max {
			return errors.New(errMsg)
		}
		return nil
	}
}
