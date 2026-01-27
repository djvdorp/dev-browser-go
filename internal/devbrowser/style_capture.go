package devbrowser

import (
	"fmt"

	"github.com/playwright-community/playwright-go"
)

type StyleCaptureOptions struct {
	Mode       string
	Selector   string
	MaxNodes   int
	IncludeAll bool
	Properties []string
	Strip      bool
}

type StyleCaptureResult struct {
	HTML       string
	CSS        string
	NodeCount  int
	Truncated  bool
	Mode       string
	Selector   string
	Properties []string
}

func ensureStyleCapture(page playwright.Page) error {
	present := false
	if val, err := page.Evaluate("() => Boolean(globalThis.__devBrowser_styleCapture)"); err == nil {
		if b, ok := val.(bool); ok {
			present = b
		}
	}
	if present {
		return nil
	}
	_, err := page.Evaluate(styleCaptureScript())
	return err
}

func StyleCapture(page playwright.Page, opts StyleCaptureOptions) (*StyleCaptureResult, error) {
	if err := ensureStyleCapture(page); err != nil {
		return nil, err
	}

	payload := map[string]interface{}{
		"mode":       opts.Mode,
		"selector":   opts.Selector,
		"maxNodes":   opts.MaxNodes,
		"includeAll": opts.IncludeAll,
		"strip":      opts.Strip,
	}
	if len(opts.Properties) > 0 {
		payload["properties"] = opts.Properties
	}

	raw, err := page.Evaluate("(opts) => globalThis.__devBrowser_styleCapture(opts)", payload)
	if err != nil {
		return nil, err
	}

	data, ok := raw.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected style capture result")
	}

	html, _ := data["html"].(string)
	css, _ := data["css"].(string)
	nodeCount, _ := asInt(data["nodeCount"])
	truncated, _ := data["truncated"].(bool)
	mode, _ := data["mode"].(string)
	selector, _ := data["selector"].(string)
	properties := []string{}
	if rawProps, ok := data["properties"].([]interface{}); ok {
		for _, item := range rawProps {
			if s, ok := item.(string); ok {
				properties = append(properties, s)
			}
		}
	}

	return &StyleCaptureResult{
		HTML:       html,
		CSS:        css,
		NodeCount:  nodeCount,
		Truncated:  truncated,
		Mode:       mode,
		Selector:   selector,
		Properties: properties,
	}, nil
}
