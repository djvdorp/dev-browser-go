package devbrowser

import (
	"fmt"
	"image"
	"math"
	"strings"

	"github.com/playwright-community/playwright-go"
)

func evaluateJS(page playwright.Page, expression string, selector, ariaRole, ariaName string, nth int) (interface{}, error) {
	if strings.TrimSpace(selector) != "" || strings.TrimSpace(ariaRole) != "" || strings.TrimSpace(ariaName) != "" {
		spec := TargetSpec{Selector: selector, AriaRole: ariaRole, AriaName: ariaName, Nth: nth, Timeout: 5000}
		el, err := SelectBySpec(page, spec)
		if err != nil {
			return nil, err
		}
		defer el.Dispose()
		return el.Evaluate(expression)
	}
	return page.Evaluate(expression)
}

func SelectBySpec(page playwright.Page, spec TargetSpec) (playwright.ElementHandle, error) {
	selector := strings.TrimSpace(spec.Selector)
	ariaRole := strings.TrimSpace(spec.AriaRole)
	ariaName := strings.TrimSpace(spec.AriaName)

	if selector == "" && ariaRole == "" {
		return nil, fmt.Errorf("selector or aria_role is required")
	}

	var locator playwright.Locator
	if selector != "" {
		locator = page.Locator(selector)
	} else {
		opts := playwright.PageGetByRoleOptions{}
		if ariaName != "" {
			opts.Name = playwright.String(ariaName)
		}
		locator = page.GetByRole(playwright.AriaRole(ariaRole), opts)
	}

	target := locator
	nth := spec.effectiveNth()
	if nth > 1 {
		target = locator.Nth(nth - 1)
	}

	if err := target.WaitFor(playwright.LocatorWaitForOptions{
		State:   playwright.WaitForSelectorStateVisible,
		Timeout: playwright.Float(float64(spec.timeoutMs())),
	}); err != nil {
		return nil, fmt.Errorf("target not found or not visible (%s): %w", spec.describe(), err)
	}

	el, err := target.ElementHandle()
	if err != nil {
		return nil, fmt.Errorf("failed to get element handle (%s): %w", spec.describe(), err)
	}
	return el, nil
}

func optionalFloat(args map[string]interface{}, key string, def float64) (float64, error) {
	raw, ok := args[key]
	if !ok {
		return def, nil
	}
	switch v := raw.(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int:
		return float64(v), nil
	default:
		return 0, fmt.Errorf("expected number '%s'", key)
	}
}

type AssetSnapshotResult struct {
	HTML        string
	AssetCount  int
	InlineCount int
	LinkedCount int
}

func createAssetSnapshot(page playwright.Page, includeAssets bool, assetTypes []string, maxDepth int, stripScripts bool, inlineThreshold int) (*AssetSnapshotResult, error) {
	html, err := page.Content()
	if err != nil {
		return nil, err
	}

	if !includeAssets {
		return &AssetSnapshotResult{HTML: html, AssetCount: 0}, nil
	}

	assets, err := extractAssets(page, assetTypes, maxDepth)
	if err != nil {
		return nil, err
	}

	processedHTML := processAssets(html, assets, inlineThreshold, stripScripts)

	inlineCount := countInlined(assets, inlineThreshold)
	linkedCount := len(assets) - inlineCount
	if linkedCount < 0 {
		linkedCount = 0
	}

	return &AssetSnapshotResult{
		HTML:        processedHTML,
		AssetCount:  len(assets),
		InlineCount: inlineCount,
		LinkedCount: linkedCount,
	}, nil
}

func extractAssets(page playwright.Page, types []string, maxDepth int) ([]map[string]interface{}, error) {
	extractJS := `() => {
		const assets = [];
		const visited = new Set();
		const typeSet = new Set($types.map((t) => String(t || "").toLowerCase()));
		const typeGroups = {
			image: new Set(["png", "jpg", "jpeg", "gif", "webp", "svg", "avif", "bmp", "ico"]),
			font: new Set(["woff", "woff2", "ttf", "otf", "eot"]),
		};

		function scan(node, depth) {
			if (depth > $maxDepth) return;
			if (!node || !node.tagName) return;

			const src = node.src || node.href;
			if (src && !visited.has(src)) {
				const clean = src.split('?')[0].split('#')[0];
				const ext = clean.split('.').pop().toLowerCase();
				const matchesType =
					typeSet.size === 0 ||
					typeSet.has(ext) ||
					Array.from(typeSet).some((t) => typeGroups[t] && typeGroups[t].has(ext));
				if (matchesType) {
					assets.push({
						url: src,
						tag: node.tagName.toLowerCase(),
						type: node.type || '',
						rel: node.rel || ''
					});
					visited.add(src);
				}
			}

			for (const child of node.children) {
				scan(child, depth + 1);
			}
		}

		scan(document.body, 0);
		return assets;
	}`

	result, err := page.Evaluate(extractJS, map[string]interface{}{
		"$types":    types,
		"$maxDepth": maxDepth,
	})

	if err != nil {
		return nil, err
	}

	arr, ok := result.([]interface{})
	if !ok {
		return []map[string]interface{}{}, nil
	}

	assets := make([]map[string]interface{}, 0, len(arr))
	for _, v := range arr {
		if m, ok := v.(map[string]interface{}); ok {
			assets = append(assets, m)
		}
	}
	return assets, nil
}

func processAssets(html string, assets []map[string]interface{}, inlineThreshold int, stripScripts bool) string {
	processed := html
	if stripScripts {
		processed = removeScripts(processed)
	}
	return processed
}

func removeScripts(html string) string {
	processed := strings.ReplaceAll(html, `<script`, `<!-- <script`)
	processed = strings.ReplaceAll(processed, `</script>`, `</script> -->`)
	return processed
}

func countInlined(assets []map[string]interface{}, threshold int) int {
	count := 0
	for _, a := range assets {
		if size, ok := a["size"].(int); ok && size > 0 && size <= threshold {
			count++
		}
	}
	return count
}

func countLinked(assets []map[string]interface{}, threshold int) int {
	count := 0
	for _, a := range assets {
		if size, ok := a["size"].(int); ok && size > threshold {
			count++
		}
	}
	return count
}

type DiffResult struct {
	Passed          bool
	DifferentPixels int
	DiffPercentage  float64
	OutputPath      string
}

func compareScreenshots(page playwright.Page, baselinePath, outputPath string, tolerance float64, pixelThreshold int, highlight bool, ignoreRegions []image.Rectangle) (*DiffResult, error) {
	currentPath := fmt.Sprintf("/tmp/dev-browser-diff-current-%d.png", NowMS())
	currentOpts := playwright.PageScreenshotOptions{
		Path:     playwright.String(currentPath),
		FullPage: playwright.Bool(true),
	}

	_, err := page.Screenshot(currentOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to capture current screenshot: %w", err)
	}

	beforeImg, err := loadImage(baselinePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load baseline image: %w", err)
	}

	afterImg, err := loadImage(currentPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load current image: %w", err)
	}

	threshold := uint8(math.Round(tolerance * 255))
	diffImg, stats, err := diffImagesWithIgnore(beforeImg, afterImg, threshold, ignoreRegions)
	if err != nil {
		return nil, fmt.Errorf("failed to compute diff: %w", err)
	}

	diffPixels := stats.ChangedPixels
	diffPercent := 0.0
	if stats.TotalPixels > 0 {
		diffPercent = float64(stats.ChangedPixels) / float64(stats.TotalPixels) * 100
	}
	passed := diffPixels <= pixelThreshold

	if highlight && outputPath != "" {
		if err := writePNG(outputPath, diffImg); err != nil {
			return nil, fmt.Errorf("failed to write diff image: %w", err)
		}
	}

	return &DiffResult{
		Passed:          passed,
		DifferentPixels: diffPixels,
		DiffPercentage:  diffPercent,
		OutputPath:      outputPath,
	}, nil
}
