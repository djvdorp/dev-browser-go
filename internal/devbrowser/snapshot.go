package devbrowser

import (
	"fmt"

	"github.com/playwright-community/playwright-go"
)

type SnapshotOptions struct {
	Engine          string
	Format          string
	InteractiveOnly bool
	IncludeHeadings bool
	MaxItems        int
	MaxChars        int
}

type SnapshotResult struct {
	Yaml  string
	Items []map[string]interface{}
}

func ensureInjected(page playwright.Page, engine string) error {
	present := false
	if val, err := page.Evaluate("() => Boolean(globalThis.__devBrowser_getAISnapshot)"); err == nil {
		if b, ok := val.(bool); ok {
			present = b
		}
	}
	if !present {
		if _, err := page.Evaluate(baseScript()); err != nil {
			return err
		}
	}

	if engine == "aria" {
		ariaPresent := false
		if val, err := page.Evaluate("() => Boolean(globalThis.__devBrowser_getAISnapshotAria)"); err == nil {
			if b, ok := val.(bool); ok {
				ariaPresent = b
			}
		}
		if !ariaPresent {
			if _, err := page.Evaluate(ariaScript()); err != nil {
				return err
			}
		}
	}
	return nil
}

func GetSnapshot(page playwright.Page, opts SnapshotOptions) (*SnapshotResult, error) {
	if err := ensureInjected(page, opts.Engine); err != nil {
		return nil, err
	}

	payload := map[string]interface{}{
		"engine":          opts.Engine,
		"format":          opts.Format,
		"interactiveOnly": opts.InteractiveOnly,
		"includeHeadings": opts.IncludeHeadings,
		"maxItems":        opts.MaxItems,
		"maxChars":        opts.MaxChars,
	}

	raw, err := page.Evaluate("(opts) => globalThis.__devBrowser_getAISnapshot(opts)", payload)
	if err != nil {
		return nil, err
	}

	m, ok := raw.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected snapshot result")
	}

	yaml, _ := m["yaml"].(string)
	items := []map[string]interface{}{}
	if arr, ok := m["items"].([]interface{}); ok {
		for _, item := range arr {
			if mm, ok := item.(map[string]interface{}); ok {
				items = append(items, mm)
			}
		}
	}

	return &SnapshotResult{Yaml: yaml, Items: items}, nil
}

func SelectRef(page playwright.Page, ref string, engine string) (playwright.ElementHandle, error) {
	if err := ensureInjected(page, engine); err != nil {
		return nil, err
	}
	handle, err := page.EvaluateHandle("(ref) => globalThis.__devBrowser_selectSnapshotRef(ref)", ref)
	if err != nil {
		return nil, err
	}
	if handle == nil {
		return nil, fmt.Errorf("ref '%s' not found", ref)
	}
	element := handle.AsElement()
	if element == nil {
		_ = handle.Dispose()
		return nil, fmt.Errorf("ref '%s' did not resolve to element", ref)
	}
	return element, nil
}

func DrawRefOverlay(page playwright.Page, maxRefs int, engine string) error {
	if err := ensureInjected(page, engine); err != nil {
		return err
	}
	_, err := page.Evaluate("(opts) => globalThis.__devBrowser_drawRefOverlay(opts)", map[string]interface{}{"maxRefs": maxRefs})
	return err
}

func ClearRefOverlay(page playwright.Page, engine string) error {
	if err := ensureInjected(page, engine); err != nil {
		return err
	}
	_, err := page.Evaluate("() => globalThis.__devBrowser_clearRefOverlay()")
	return err
}
