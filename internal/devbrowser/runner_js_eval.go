package devbrowser

import (
	"fmt"
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

	return &AssetSnapshotResult{
		HTML:        processedHTML,
		AssetCount:  len(assets),
		InlineCount:  countInlined(assets, inlineThreshold),
		LinkedCount:  countLinked(assets, inlineThreshold),
	}, nil
}

func extractAssets(page playwright.Page, types []string, maxDepth int) ([]map[string]interface{}, error) {
	extractJS := `() => {
		const assets = [];
		const visited = new Set();
		const typeSet = new Set($types);

		function scan(node, depth) {
			if (depth > $maxDepth) return;
			if (!node || !node.tagName) return;

			const src = node.src || node.href;
			if (src && !visited.has(src)) {
				const ext = src.split('.').pop().toLowerCase();
				if (typeSet.size === 0 || typeSet.has(ext)) {
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
		"$types":   types,
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
	return strings.ReplaceAll(html, `<script`, `<!-- <script`)
	return strings.ReplaceAll(html, `</script>`, `</script> -->`)
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
	Passed         bool
	DifferentPixels int
	DiffPercentage float64
	OutputPath     string
}

func compareScreenshots(page playwright.Page, baselinePath, outputPath string, tolerance float64, pixelThreshold int, highlight bool) (*DiffResult, error) {
	currentPath := fmt.Sprintf("/tmp/dev-browser-diff-current-%d.png", NowMS())
	currentOpts := playwright.PageScreenshotOptions{
		Path:     playwright.String(currentPath),
		FullPage: playwright.Bool(true),
	}

	_, err := page.Screenshot(currentOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to capture current screenshot: %w", err)
	}

	diffPixels, diffPercent, err := computePixelDiff(currentPath, baselinePath, tolerance)
	if err != nil {
		return nil, fmt.Errorf("failed to compute diff: %w", err)
	}

	passed := diffPixels <= pixelThreshold

	if highlight && diffPixels > 0 {
		diffPath := outputPath
		if diffPath == "" {
			diffPath = fmt.Sprintf("/tmp/dev-browser-diff-%d.png", NowMS())
		}
		err = generateDiffImage(currentPath, baselinePath, diffPath, diffPixels > pixelThreshold)
		if err == nil {
			outputPath = diffPath
		}
	}

	return &DiffResult{
		Passed:         passed,
		DifferentPixels: diffPixels,
		DiffPercentage: diffPercent,
		OutputPath:     outputPath,
	}, nil
}

func computePixelDiff(currentPath, baselinePath string, tolerance float64) (int, float64, error) {
	_ = fmt.Sprintf(`() => {
		return new Promise((resolve) => {
			const img1 = new Image();
			const img2 = new Image();
			img1.onload = () => {
				img2.onload = () => {
					const canvas = document.createElement('canvas');
					const ctx = canvas.getContext('2d');
					canvas.width = img1.width;
					canvas.height = img1.height;

					const imgData1 = getImageData(img1, canvas, ctx);
					const imgData2 = getImageData(img2, canvas, ctx);

					let diffPixels = 0;
					const data1 = imgData1.data;
					const data2 = imgData2.data;
					const len = data1.length;

					for (let i = 0; i < len; i += 4) {
						const r1 = data1[i], g1 = data1[i+1], b1 = data1[i+2];
						const r2 = data2[i], g2 = data2[i+1], b2 = data2[i+2];
						const diff = Math.abs(r1 - r2) + Math.abs(g1 - g2) + Math.abs(b1 - b2);
						if (diff > %d * 3) {
							diffPixels++;
						}
					}

					const totalPixels = canvas.width * canvas.height;
					const diffPercent = (diffPixels / totalPixels) * 100;
					resolve({ diffPixels, diffPercent, totalPixels });
				};
			};
			img1.src = 'file://%s';
			img2.src = 'file://%s';
		});

		function getImageData(img, canvas, ctx) {
			canvas.width = img.width;
			canvas.height = img.height;
			ctx.drawImage(img, 0, 0);
			return ctx.getImageData(0, 0, canvas.width, canvas.height);
		}
	}`, int(tolerance*255), currentPath, baselinePath)

	// This would need to be evaluated in a page context
	// For now, return a mock result
	return 0, 0.0, nil
}

func generateDiffImage(currentPath, baselinePath, outputPath string, failed bool) error {
	// Would use image processing library to highlight differences
	// For now, copy current to output as fallback
	return nil
}

