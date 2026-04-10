package main

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	devbrowser "github.com/joshp123/dev-browser-go/internal/devbrowser"
	"github.com/playwright-community/playwright-go"
)

func runWithPage(pageName, tool string, args map[string]interface{}) error {
	pw, browser, page, err := openNamedPage(pageName)
	if err != nil {
		return err
	}
	defer browser.Close()
	defer pw.Stop()

	res, err := devbrowser.RunCall(page, tool, args, devbrowser.ArtifactDir(globalOpts.profile))
	if err != nil {
		return err
	}
	out, err := devbrowser.WriteOutput(globalOpts.profile, globalOpts.output, res, globalOpts.outPath)
	if err != nil {
		return err
	}
	fmt.Println(out)
	return nil
}

func openNamedPage(pageName string) (*playwright.Playwright, playwright.Browser, playwright.Page, error) {
	sessionInfo, err := ensurePageInfoForCommand(pageName)
	if err != nil {
		return nil, nil, nil, err
	}
	pw, browser, page, err := devbrowser.OpenPage(sessionInfo.WSEndpoint, sessionInfo.TargetID)
	if err != nil {
		return nil, nil, nil, err
	}
	if err := verifyReopenedPage(pageName, sessionInfo, page); err != nil {
		_ = browser.Close()
		_ = pw.Stop()
		return nil, nil, nil, err
	}
	return pw, browser, page, nil
}

func verifyReopenedPage(pageName string, expected devbrowser.PageSessionInfo, page playwright.Page) error {
	actual, err := devbrowser.DescribePage(page)
	if err != nil {
		return fmt.Errorf("reopen page %q targetId=%s: inspect reopened page: %w", pageName, expected.TargetID, err)
	}
	if strings.TrimSpace(expected.TargetID) != "" && actual.TargetID != expected.TargetID {
		return fmt.Errorf("reopen page %q: targetId mismatch: expected=%s actual=%s expected_url=%q expected_title=%q actual_url=%q actual_title=%q", pageName, expected.TargetID, actual.TargetID, expected.URL, expected.Title, actual.URL, actual.Title)
	}
	if expectedPageWasLoaded(expected) && actualPageLooksBlank(actual) {
		return fmt.Errorf("reopen page %q: reopened blank page targetId=%s expected_url=%q expected_title=%q actual_url=%q actual_title=%q", pageName, expected.TargetID, expected.URL, expected.Title, actual.URL, actual.Title)
	}
	return nil
}

func ensurePageInfoForCommand(pageName string) (devbrowser.PageSessionInfo, error) {
	result, err := ensureDaemonForCommand()
	if err != nil {
		return devbrowser.PageSessionInfo{}, err
	}
	base := result.BaseURL
	if strings.TrimSpace(base) == "" {
		base = devbrowser.DaemonBaseURL(globalOpts.profile)
	}
	if base == "" {
		return devbrowser.PageSessionInfo{}, errors.New("daemon state missing after start")
	}
	data, err := devbrowser.HTTPJSON("POST", base+"/pages", map[string]any{"name": pageName}, 10*time.Second)
	if err != nil {
		return devbrowser.PageSessionInfo{}, err
	}
	ws, _ := data["wsEndpoint"].(string)
	tid, _ := data["targetId"].(string)
	if strings.TrimSpace(ws) == "" {
		return devbrowser.PageSessionInfo{}, errors.New("daemon did not return wsEndpoint")
	}
	if strings.TrimSpace(tid) == "" {
		return devbrowser.PageSessionInfo{}, errors.New("daemon did not return targetId")
	}
	info := devbrowser.PageSessionInfo{
		WSEndpoint: ws,
		PageIdentity: devbrowser.PageIdentity{
			TargetID: tid,
		},
	}
	if pageURL, _ := data["url"].(string); strings.TrimSpace(pageURL) != "" {
		info.URL = pageURL
	}
	if title, _ := data["title"].(string); strings.TrimSpace(title) != "" {
		info.Title = title
	}
	return info, nil
}

func expectedPageWasLoaded(info devbrowser.PageSessionInfo) bool {
	return !isBlankPageURL(info.URL) || strings.TrimSpace(info.Title) != ""
}

func actualPageLooksBlank(info devbrowser.PageIdentity) bool {
	return isBlankPageURL(info.URL) && strings.TrimSpace(info.Title) == ""
}

func isBlankPageURL(raw string) bool {
	url := strings.TrimSpace(raw)
	return url == "" || url == "about:blank"
}

func startDaemonIfNeeded() (string, error) {
	result, err := ensureDaemonForCommand()
	if err != nil {
		return "", err
	}
	base := result.BaseURL
	if base == "" {
		base = devbrowser.DaemonBaseURL(globalOpts.profile)
	}
	if base == "" {
		return "", errors.New("daemon state missing after start")
	}
	return base, nil
}

func ensureDaemonForCommand() (devbrowser.DaemonStartResult, error) {
	headless, window, device, err := desiredDaemonSettings()
	if err != nil {
		return devbrowser.DaemonStartResult{}, err
	}
	result, err := devbrowser.EnsureDaemon(globalOpts.profile, headless, window, device)
	if err != nil {
		return devbrowser.DaemonStartResult{}, err
	}
	announceDaemonAction(result)
	return result, nil
}

func desiredDaemonSettings() (bool, *devbrowser.WindowSize, string, error) {
	headless := globalOpts.headless
	window := cloneCLIWindow(globalOpts.window)
	device := strings.TrimSpace(globalOpts.device)

	health, err := devbrowser.ReadDaemonHealth(globalOpts.profile)
	if err != nil {
		return false, nil, "", err
	}
	if health == nil {
		return headless, window, device, nil
	}

	if !globalOpts.headlessSet {
		headless = health.Context.Headless
	}
	if !globalOpts.deviceSet && !globalOpts.windowSet {
		device = strings.TrimSpace(health.Context.Device)
		if device != "" {
			window = nil
		} else {
			window = cloneCLIWindow(health.Context.Window)
		}
	}

	if window != nil && device != "" {
		return false, nil, "", fmt.Errorf("use either --window-size/--window-scale or --device")
	}
	return headless, window, device, nil
}

func announceDaemonAction(result devbrowser.DaemonStartResult) {
	switch result.Action {
	case devbrowser.DaemonActionStarted:
		fmt.Fprintf(os.Stderr, "profile %s started with %s\n", globalOpts.profile, contextSummary(result.Context))
	case devbrowser.DaemonActionReused:
		fmt.Fprintf(os.Stderr, "profile %s reused with %s\n", globalOpts.profile, contextSummary(result.Context))
	case devbrowser.DaemonActionReconfigured:
		reason := strings.TrimSpace(result.Reason)
		if reason == "" {
			reason = contextSummary(result.Context)
		}
		fmt.Fprintf(os.Stderr, "profile %s restarted to apply %s\n", globalOpts.profile, reason)
	}
}

func contextSummary(settings devbrowser.BrowserContextSettings) string {
	parts := []string{fmt.Sprintf("headless=%t", settings.Headless)}
	if device := strings.TrimSpace(settings.Device); device != "" {
		parts = append(parts, fmt.Sprintf("device=%s", device))
	}
	if settings.Window != nil {
		parts = append(parts, fmt.Sprintf("window=%dx%d", settings.Window.Width, settings.Window.Height))
	}
	if settings.Viewport != nil {
		parts = append(parts, fmt.Sprintf("viewport=%dx%d", settings.Viewport.Width, settings.Viewport.Height))
	}
	return strings.Join(parts, " ")
}

func cloneCLIWindow(src *devbrowser.WindowSize) *devbrowser.WindowSize {
	if src == nil {
		return nil
	}
	copy := *src
	return &copy
}

func deletePage(name string) error {
	base, err := startDaemonIfNeeded()
	if err != nil {
		return err
	}
	encoded := url.PathEscape(name)
	data, err := devbrowser.HTTPJSON("DELETE", base+"/pages/"+encoded, nil, 5*time.Second)
	if err != nil {
		return err
	}
	if ok, _ := data["ok"].(bool); !ok {
		return fmt.Errorf("close failed: %v", data["error"])
	}
	out, err := devbrowser.WriteOutput(globalOpts.profile, globalOpts.output, map[string]any{"page": name, "closed": true}, globalOpts.outPath)
	if err != nil {
		return err
	}
	fmt.Println(out)
	return nil
}
