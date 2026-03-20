package devbrowser

import (
	"errors"
	"fmt"
	"strings"

	"github.com/playwright-community/playwright-go"
)

type PageIdentity struct {
	TargetID string
	URL      string
	Title    string
}

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
	page, identity, err := findPageByTargetID(browser, targetID)
	if err != nil {
		browser.Close()
		pw.Stop()
		return nil, nil, nil, err
	}
	if strings.TrimSpace(identity.TargetID) == "" {
		browser.Close()
		pw.Stop()
		return nil, nil, nil, fmt.Errorf("page lookup for targetId=%s returned empty target id", targetID)
	}
	if identity.TargetID != targetID {
		browser.Close()
		pw.Stop()
		return nil, nil, nil, fmt.Errorf("page lookup mismatch: requested targetId=%s got targetId=%s url=%q title=%q", targetID, identity.TargetID, identity.URL, identity.Title)
	}
	return pw, browser, page, nil
}

func DescribePage(page playwright.Page) (PageIdentity, error) {
	if page == nil {
		return PageIdentity{}, errors.New("page is nil")
	}
	if page.IsClosed() {
		return PageIdentity{URL: page.URL(), Title: safeTitle(page)}, errors.New("page is closed")
	}
	return describePageInContext(page.Context(), page)
}

func findPageByTargetID(browser playwright.Browser, targetID string) (playwright.Page, PageIdentity, error) {
	var available []PageIdentity
	for _, ctx := range browser.Contexts() {
		page, infos := findInContext(ctx, targetID)
		available = append(available, infos...)
		if page != nil {
			info, err := describePageInContext(ctx, page)
			if err != nil {
				return nil, PageIdentity{}, fmt.Errorf("inspect matched page targetId=%s: %w", targetID, err)
			}
			return page, info, nil
		}
	}
	return nil, PageIdentity{}, fmt.Errorf("page with targetId=%s not found; available pages: %s", targetID, formatPageIdentities(available))
}

func findInContext(ctx playwright.BrowserContext, targetID string) (playwright.Page, []PageIdentity) {
	infos := make([]PageIdentity, 0, len(ctx.Pages()))
	for _, page := range ctx.Pages() {
		info, err := describePageInContext(ctx, page)
		if err != nil {
			infos = append(infos, PageIdentity{URL: page.URL(), Title: safeTitle(page)})
			continue
		}
		infos = append(infos, info)
		if info.TargetID == targetID {
			return page, infos
		}
	}
	return nil, infos
}

func describePageInContext(ctx playwright.BrowserContext, page playwright.Page) (PageIdentity, error) {
	identity := PageIdentity{
		URL:   page.URL(),
		Title: safeTitle(page),
	}
	tid, err := resolveTargetID(ctx, page)
	if err != nil {
		return identity, err
	}
	identity.TargetID = tid
	return identity, nil
}

func formatPageIdentities(infos []PageIdentity) string {
	if len(infos) == 0 {
		return "none"
	}
	parts := make([]string, 0, len(infos))
	for _, info := range infos {
		targetID := strings.TrimSpace(info.TargetID)
		if targetID == "" {
			targetID = "unknown"
		}
		url := strings.TrimSpace(info.URL)
		if url == "" {
			url = "about:blank"
		}
		title := strings.TrimSpace(info.Title)
		if title == "" {
			title = "<empty>"
		}
		parts = append(parts, fmt.Sprintf("%s url=%q title=%q", targetID, url, title))
	}
	return strings.Join(parts, "; ")
}
