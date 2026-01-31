package devbrowser

import (
	"fmt"

	"github.com/playwright-community/playwright-go"
)

func ColorInfo(page playwright.Page, ref string, engine string) (map[string]interface{}, error) {
	if err := ensureInjected(page, engine); err != nil {
		return nil, err
	}
	res, err := page.Evaluate(`(ref) => globalThis.__devBrowser_colorInfo(ref)`, ref)
	if err != nil {
		return nil, err
	}
	m, ok := res.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected colorInfo result")
	}
	return m, nil
}

func FontInfo(page playwright.Page, ref string, engine string) (map[string]interface{}, error) {
	if err := ensureInjected(page, engine); err != nil {
		return nil, err
	}
	res, err := page.Evaluate(`(ref) => globalThis.__devBrowser_fontInfo(ref)`, ref)
	if err != nil {
		return nil, err
	}
	m, ok := res.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected fontInfo result")
	}
	return m, nil
}
