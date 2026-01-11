package devbrowser

import (
	"fmt"

	"github.com/playwright-community/playwright-go"
)

func OpenPage(wsEndpoint string, targetID string) (*playwright.Playwright, playwright.Browser, playwright.Page, error) {
	pw, err := playwright.Run()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("playwright run: %w", err)
	}
	browser, err := pw.Chromium.ConnectOverCDP(wsEndpoint)
	if err != nil {
		pw.Stop()
		return nil, nil, nil, fmt.Errorf("connect over CDP: %w", err)
	}
	page, err := findPageByTargetID(browser, targetID)
	if err != nil {
		browser.Close()
		pw.Stop()
		return nil, nil, nil, err
	}
	return pw, browser, page, nil
}

func findPageByTargetID(browser playwright.Browser, targetID string) (playwright.Page, error) {
	for _, ctx := range browser.Contexts() {
		if page := findInContext(ctx, targetID); page != nil {
			return page, nil
		}
	}
	return nil, fmt.Errorf("page with targetId=%s not found", targetID)
}

func findInContext(ctx playwright.BrowserContext, targetID string) playwright.Page {
	for _, page := range ctx.Pages() {
		session, err := ctx.NewCDPSession(page)
		if err != nil {
			continue
		}
		info, err := session.Send("Target.getTargetInfo", map[string]interface{}{})
		session.Detach()
		if err != nil {
			continue
		}
		if ti, ok := info.(map[string]interface{}); ok {
			if tm, ok := ti["targetInfo"].(map[string]interface{}); ok {
				if tid, ok := tm["targetId"].(string); ok && tid == targetID {
					return page
				}
			}
		}
	}
	return nil
}
