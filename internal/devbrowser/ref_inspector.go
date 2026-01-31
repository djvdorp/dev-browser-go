package devbrowser

import (
	"fmt"

	"github.com/playwright-community/playwright-go"
)

type RefInspectOptions struct {
	StyleProps []string
}

func InspectRef(page playwright.Page, ref string, engine string, opts RefInspectOptions) (map[string]interface{}, error) {
	if err := ensureInjected(page, engine); err != nil {
		return nil, err
	}
	payload := map[string]interface{}{"ref": ref, "opts": map[string]interface{}{"styleProps": opts.StyleProps}}
	res, err := page.Evaluate(`(p) => globalThis.__devBrowser_inspectRef(p.ref, p.opts)`, payload)
	if err != nil {
		return nil, err
	}
	m, ok := res.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected inspect result")
	}
	return m, nil
}

func TestSelector(page playwright.Page, selector string, engine string) (map[string]interface{}, error) {
	if err := ensureInjected(page, engine); err != nil {
		return nil, err
	}
	res, err := page.Evaluate(`(sel) => globalThis.__devBrowser_testSelector(sel)`, selector)
	if err != nil {
		return nil, err
	}
	m, ok := res.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected selector test result")
	}
	return m, nil
}

func TestXPath(page playwright.Page, xpath string, engine string) (map[string]interface{}, error) {
	if err := ensureInjected(page, engine); err != nil {
		return nil, err
	}
	res, err := page.Evaluate(`(xp) => globalThis.__devBrowser_testXPath(xp)`, xpath)
	if err != nil {
		return nil, err
	}
	m, ok := res.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected xpath test result")
	}
	return m, nil
}
