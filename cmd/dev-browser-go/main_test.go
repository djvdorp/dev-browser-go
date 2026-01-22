package main

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func newTestCmd() *cobra.Command {
	globalOpts = &globalOptions{}
	cmd := &cobra.Command{Use: "test"}
	bindGlobalFlags(cmd)
	return cmd
}

func TestApplyGlobalOptionsDeviceOverridesEnvWindow(t *testing.T) {
	t.Setenv("DEV_BROWSER_WINDOW_SIZE", "320x640")
	cmd := newTestCmd()
	if err := cmd.PersistentFlags().Set("device", "Pixel 5"); err != nil {
		t.Fatalf("set device: %v", err)
	}
	if err := applyGlobalOptions(cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if globalOpts.device != "Pixel 5" {
		t.Fatalf("expected device to be set, got %q", globalOpts.device)
	}
	if globalOpts.window != nil {
		t.Fatalf("expected window to be nil with device, got %#v", globalOpts.window)
	}
}

func TestApplyGlobalOptionsDeviceConflictsWithWindow(t *testing.T) {
	cmd := newTestCmd()
	if err := cmd.PersistentFlags().Set("device", "Pixel 5"); err != nil {
		t.Fatalf("set device: %v", err)
	}
	if err := cmd.PersistentFlags().Set("window-size", "100x200"); err != nil {
		t.Fatalf("set window-size: %v", err)
	}
	if err := applyGlobalOptions(cmd); err == nil {
		t.Fatalf("expected error for device + window-size")
	} else if !strings.Contains(err.Error(), "--device") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestApplyGlobalOptionsUsesEnvWindowSize(t *testing.T) {
	t.Setenv("DEV_BROWSER_WINDOW_SIZE", "320x640")
	cmd := newTestCmd()
	if err := applyGlobalOptions(cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if globalOpts.window == nil || globalOpts.window.Width != 320 || globalOpts.window.Height != 640 {
		t.Fatalf("expected window from env, got %#v", globalOpts.window)
	}
}

func TestApplyGlobalOptionsDeviceRequiresValue(t *testing.T) {
	cmd := newTestCmd()
	if err := cmd.PersistentFlags().Set("device", "  "); err != nil {
		t.Fatalf("set device: %v", err)
	}
	if err := applyGlobalOptions(cmd); err == nil {
		t.Fatalf("expected error for empty device")
	}
}

func TestApplyGlobalOptionsDeviceConflictsWithWindowScale(t *testing.T) {
	cmd := newTestCmd()
	if err := cmd.PersistentFlags().Set("device", "Pixel 5"); err != nil {
		t.Fatalf("set device: %v", err)
	}
	if err := cmd.PersistentFlags().Set("window-scale", "0.5"); err != nil {
		t.Fatalf("set window-scale: %v", err)
	}
	if err := applyGlobalOptions(cmd); err == nil {
		t.Fatalf("expected error for device + window-scale")
	}
}

func TestApplyGlobalOptionsAllowsHTMLOutput(t *testing.T) {
	cmd := newTestCmd()
	if err := cmd.PersistentFlags().Set("output", "html"); err != nil {
		t.Fatalf("set output: %v", err)
	}
	if err := applyGlobalOptions(cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if globalOpts.output != "html" {
		t.Fatalf("expected output to be html, got %q", globalOpts.output)
	}
}
