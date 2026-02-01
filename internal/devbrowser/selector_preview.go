package devbrowser

import (
	"fmt"
	"strings"

	"github.com/playwright-community/playwright-go"
)

type SelectorPreviewOptions struct {
	Limit int
	// TextMaxChars truncates textContent/innerText preview.
	TextMaxChars int
}

// SelectorPreview returns a deterministic preview of elements matching a selector.
// This is intended for assert failure context.
func SelectorPreview(page playwright.Page, selector string, opts SelectorPreviewOptions) ([]map[string]any, error) {
	selector = strings.TrimSpace(selector)
	if selector == "" {
		return nil, fmt.Errorf("selector is required")
	}
	if opts.Limit <= 0 {
		opts.Limit = 5
	}
	if opts.TextMaxChars <= 0 {
		opts.TextMaxChars = 120
	}

	// Reuse the injected JS helper (same logic as `test-selector` command) for stability.
	// Then truncate to the requested limit.
	m, err := TestSelector(page, selector, "simple")
	if err != nil {
		return nil, err
	}
	raw, _ := m["preview"].([]interface{})
	out := make([]map[string]any, 0, len(raw))
	for _, v := range raw {
		mm, ok := v.(map[string]any)
		if !ok {
			continue
		}
		out = append(out, mm)
		if len(out) >= opts.Limit {
			break
		}
	}
	return out, nil
}
