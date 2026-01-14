package devbrowser

import (
	"strings"
	"testing"

	"github.com/playwright-community/playwright-go"
)

func TestDeviceWindowSize(t *testing.T) {
	if got := deviceWindowSize(nil); got != nil {
		t.Fatalf("expected nil, got %#v", got)
	}
	if got := deviceWindowSize(&playwright.DeviceDescriptor{}); got != nil {
		t.Fatalf("expected nil for missing viewport, got %#v", got)
	}
	desc := &playwright.DeviceDescriptor{
		Viewport: &playwright.Size{Width: 320, Height: 640},
	}
	got := deviceWindowSize(desc)
	if got == nil || got.Width != 320 || got.Height != 640 {
		t.Fatalf("unexpected window size: %#v", got)
	}
}

func TestApplyDeviceDescriptorDefaultsScreen(t *testing.T) {
	opts := playwright.BrowserTypeLaunchPersistentContextOptions{}
	desc := &playwright.DeviceDescriptor{
		UserAgent:         "ua-test",
		Viewport:          &playwright.Size{Width: 360, Height: 740},
		DeviceScaleFactor: 2,
		IsMobile:          true,
		HasTouch:          true,
	}
	applyDeviceDescriptor(&opts, desc)

	if opts.UserAgent == nil || *opts.UserAgent != "ua-test" {
		t.Fatalf("expected user agent to be set")
	}
	if opts.Viewport == nil || opts.Viewport.Width != 360 || opts.Viewport.Height != 740 {
		t.Fatalf("expected viewport to be set from device")
	}
	if opts.Screen == nil || opts.Screen.Width != 360 || opts.Screen.Height != 740 {
		t.Fatalf("expected screen to default to viewport")
	}
	if opts.DeviceScaleFactor == nil || *opts.DeviceScaleFactor != 2 {
		t.Fatalf("expected device scale factor to be set")
	}
	if opts.IsMobile == nil || !*opts.IsMobile {
		t.Fatalf("expected isMobile to be set")
	}
	if opts.HasTouch == nil || !*opts.HasTouch {
		t.Fatalf("expected hasTouch to be set")
	}
}

func TestApplyDeviceDescriptorUsesScreen(t *testing.T) {
	opts := playwright.BrowserTypeLaunchPersistentContextOptions{}
	desc := &playwright.DeviceDescriptor{
		Viewport: &playwright.Size{Width: 360, Height: 740},
		Screen:   &playwright.Size{Width: 720, Height: 1480},
	}
	applyDeviceDescriptor(&opts, desc)

	if opts.Screen == nil || opts.Screen.Width != 720 || opts.Screen.Height != 1480 {
		t.Fatalf("expected screen to use device screen")
	}
	if opts.Viewport == nil || opts.Viewport.Width != 360 || opts.Viewport.Height != 740 {
		t.Fatalf("expected viewport to use device viewport")
	}
}

func TestResolveDeviceDescriptorCaseInsensitive(t *testing.T) {
	pw, err := playwright.Run()
	if err != nil {
		t.Skip("playwright not available:", err)
	}
	defer pw.Stop()

	// Test exact match
	desc, err := resolveDeviceDescriptor(pw, "iPhone 13")
	if err != nil {
		t.Fatalf("exact match failed: %v", err)
	}
	if desc == nil {
		t.Fatal("expected device descriptor, got nil")
	}

	// Test case-insensitive match - lowercase
	desc, err = resolveDeviceDescriptor(pw, "iphone 13")
	if err != nil {
		t.Fatalf("lowercase match failed: %v", err)
	}
	if desc == nil {
		t.Fatal("expected device descriptor for lowercase, got nil")
	}

	// Test case-insensitive match - uppercase
	desc, err = resolveDeviceDescriptor(pw, "IPHONE 13")
	if err != nil {
		t.Fatalf("uppercase match failed: %v", err)
	}
	if desc == nil {
		t.Fatal("expected device descriptor for uppercase, got nil")
	}

	// Test mixed case
	desc, err = resolveDeviceDescriptor(pw, "IpHoNe 13")
	if err != nil {
		t.Fatalf("mixed case match failed: %v", err)
	}
	if desc == nil {
		t.Fatal("expected device descriptor for mixed case, got nil")
	}
}

func TestResolveDeviceDescriptorErrorMessage(t *testing.T) {
	pw, err := playwright.Run()
	if err != nil {
		t.Skip("playwright not available:", err)
	}
	defer pw.Stop()

	// Test that error message includes available devices
	_, err = resolveDeviceDescriptor(pw, "NonExistentDevice")
	if err == nil {
		t.Fatal("expected error for non-existent device")
	}
	
	errMsg := err.Error()
	if !strings.Contains(errMsg, "Available devices include:") {
		t.Errorf("error message should list available devices, got: %s", errMsg)
	}
	if !strings.Contains(errMsg, "devices") {
		t.Errorf("error message should mention devices command, got: %s", errMsg)
	}
}

func TestResolveDeviceDescriptorEmpty(t *testing.T) {
	pw, err := playwright.Run()
	if err != nil {
		t.Skip("playwright not available:", err)
	}
	defer pw.Stop()

	// Test empty string returns nil without error
	desc, err := resolveDeviceDescriptor(pw, "")
	if err != nil {
		t.Fatalf("empty string should not error: %v", err)
	}
	if desc != nil {
		t.Fatal("expected nil descriptor for empty string")
	}

	// Test whitespace-only string returns nil without error
	desc, err = resolveDeviceDescriptor(pw, "   ")
	if err != nil {
		t.Fatalf("whitespace-only string should not error: %v", err)
	}
	if desc != nil {
		t.Fatal("expected nil descriptor for whitespace-only string")
	}
}
