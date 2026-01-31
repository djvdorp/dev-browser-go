package devbrowser

import (
	"fmt"

	"github.com/playwright-community/playwright-go"
)

// CountSelector returns document.querySelectorAll(selector).length.
func CountSelector(page playwright.Page, selector string) (int, error) {
	js := `() => {
  const sel = arguments[0];
  try {
    return document.querySelectorAll(String(sel)).length;
  } catch (e) {
    return { __error: String(e && e.message ? e.message : e) };
  }
}`
	res, err := page.Evaluate(js, selector)
	if err != nil {
		return 0, err
	}
	switch v := res.(type) {
	case float64:
		return int(v), nil
	case int:
		return v, nil
	case map[string]any:
		if msg, ok := v["__error"].(string); ok {
			return 0, fmt.Errorf("selector eval error: %s", msg)
		}
	}
	return 0, fmt.Errorf("unexpected selector count result")
}
