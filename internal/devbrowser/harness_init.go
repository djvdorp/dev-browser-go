package devbrowser

import (
	_ "embed"
	"fmt"

	"github.com/playwright-community/playwright-go"
)

//go:embed harness_assets/harness_init.js
var harnessInitJS string

func InstallHarnessInit(context playwright.BrowserContext) error {
	if context == nil {
		return nil
	}
	// Inject into every new document before page scripts run.
	return context.AddInitScript(playwright.Script{Content: playwright.String(harnessInitJS)})
}

func EnsureHarnessOnPage(page playwright.Page) {
	if page == nil {
		return
	}
	// Best-effort: install for the current document as well (AddInitScript only affects future navigations/frames).
	res, err := page.Evaluate(`() => Boolean(globalThis.__devBrowser_getHarnessState)`)
	if err == nil {
		if installed, ok := res.(bool); ok && installed {
			// Harness is already installed; avoid re-running the large init script.
			return
		}
	}
	_, _ = page.Evaluate(harnessInitJS)
}

func ReadHarnessState(page playwright.Page) (map[string]any, error) {
	if page == nil {
		return nil, nil
	}
	res, err := page.Evaluate(`() => (globalThis.__devBrowser_getHarnessState ? globalThis.__devBrowser_getHarnessState() : null)`)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, nil
	}
	m, ok := res.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected harness state of type %T: %v", res, res)
	}
	return m, nil
}
