package main

import (
	"errors"
	"fmt"
	"net/url"
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
	sessionInfo, err := devbrowser.EnsurePageInfo(globalOpts.profile, globalOpts.headless, pageName, globalOpts.window, globalOpts.device)
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
	if err := devbrowser.StartDaemon(globalOpts.profile, globalOpts.headless, globalOpts.window, globalOpts.device); err != nil {
		return "", err
	}
	base := devbrowser.DaemonBaseURL(globalOpts.profile)
	if base == "" {
		return "", errors.New("daemon state missing after start")
	}
	return base, nil
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
