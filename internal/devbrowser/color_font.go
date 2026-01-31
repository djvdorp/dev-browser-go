package devbrowser

import (
	"fmt"

	"github.com/playwright-community/playwright-go"
)

type ColorInfoOptions struct {
	IncludeTransparent bool
}

func ColorInfo(page playwright.Page, ref string, engine string, opts ColorInfoOptions) (map[string]interface{}, error) {
	if err := ensureInjected(page, engine); err != nil {
		return nil, err
	}
	payload := map[string]interface{}{"ref": ref, "opts": map[string]interface{}{"includeTransparent": opts.IncludeTransparent}}
	res, err := page.Evaluate(`(p) => globalThis.__devBrowser_colorInfo(p.ref, p.opts)`, payload)
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
