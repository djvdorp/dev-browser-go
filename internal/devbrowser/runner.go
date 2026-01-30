package devbrowser

import (
	"errors"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/playwright-community/playwright-go"
)

type RunResult map[string]interface{}

type ActionsResult struct {
	Results  []map[string]interface{}
	Snapshot string
}

func RunCall(page playwright.Page, name string, args map[string]interface{}, artifactDir string) (RunResult, error) {
	switch name {
	case "goto":
		url, err := requireString(args, "url")
		if err != nil {
			return nil, err
		}
		waitUntil, err := optionalString(args, "wait_until", "domcontentloaded")
		if err != nil {
			return nil, err
		}
		timeoutMs, err := optionalInt(args, "timeout_ms", 45_000)
		if err != nil {
			return nil, err
		}
		_, err = page.Goto(url, playwright.PageGotoOptions{
			WaitUntil: getWaitUntil(waitUntil),
			Timeout:   playwright.Float(float64(timeoutMs)),
		})
		if err != nil {
			return nil, err
		}
		return RunResult{"url": page.URL(), "title": safeTitle(page)}, nil

	case "snapshot":
		engine, err := optionalString(args, "engine", "simple")
		if err != nil {
			return nil, err
		}
		format, err := optionalString(args, "format", "list")
		if err != nil {
			return nil, err
		}
		interactiveOnly, err := optionalBool(args, "interactive_only", true)
		if err != nil {
			return nil, err
		}
		includeHeadings, err := optionalBool(args, "include_headings", true)
		if err != nil {
			return nil, err
		}
		maxItems, err := optionalInt(args, "max_items", 80)
		if err != nil {
			return nil, err
		}
		maxChars, err := optionalInt(args, "max_chars", 8000)
		if err != nil {
			return nil, err
		}

		snap, err := GetSnapshot(page, SnapshotOptions{
			Engine:          engine,
			Format:          format,
			InteractiveOnly: interactiveOnly,
			IncludeHeadings: includeHeadings,
			MaxItems:        maxItems,
			MaxChars:        maxChars,
		})
		if err != nil {
			return nil, err
		}
		return RunResult{
			"url":      page.URL(),
			"title":    safeTitle(page),
			"engine":   engine,
			"format":   format,
			"snapshot": snap.Yaml,
			"items":    snap.Items,
		}, nil

	case "click_ref":
		ref, err := requireString(args, "ref")
		if err != nil {
			return nil, err
		}
		timeoutMs, err := optionalInt(args, "timeout_ms", 15_000)
		if err != nil {
			return nil, err
		}
		el, err := SelectRef(page, ref, "simple")
		if err != nil {
			return nil, err
		}
		err = el.Click(playwright.ElementHandleClickOptions{Timeout: playwright.Float(float64(timeoutMs))})
		_ = el.Dispose()
		if err != nil {
			return nil, err
		}
		return RunResult{"ref": ref, "clicked": true}, nil

	case "fill_ref":
		ref, err := requireString(args, "ref")
		if err != nil {
			return nil, err
		}
		text, err := requireString(args, "text")
		if err != nil {
			return nil, err
		}
		timeoutMs, err := optionalInt(args, "timeout_ms", 15_000)
		if err != nil {
			return nil, err
		}
		el, err := SelectRef(page, ref, "simple")
		if err != nil {
			return nil, err
		}
		err = el.Fill(text, playwright.ElementHandleFillOptions{Timeout: playwright.Float(float64(timeoutMs))})
		_ = el.Dispose()
		if err != nil {
			return nil, err
		}
		return RunResult{"ref": ref, "filled": true}, nil

	case "press":
		key, err := requireString(args, "key")
		if err != nil {
			return nil, err
		}
		if err := page.Keyboard().Press(key); err != nil {
			return nil, err
		}
		return RunResult{"key": key, "pressed": true}, nil

	case "wait":
		strategy, err := optionalString(args, "strategy", "playwright")
		if err != nil {
			return nil, err
		}
		state, err := optionalString(args, "state", "load")
		if err != nil {
			return nil, err
		}
		allowedStates := map[string]bool{"load": true, "domcontentloaded": true, "networkidle": true, "commit": true}
		if !allowedStates[strings.ToLower(state)] {
			return nil, fmt.Errorf("invalid state '%s' (expected one of: load, domcontentloaded, networkidle, commit)", state)
		}
		timeoutMs, err := optionalInt(args, "timeout_ms", 10_000)
		if err != nil {
			return nil, err
		}
		minWaitMs, err := optionalInt(args, "min_wait_ms", 0)
		if err != nil {
			return nil, err
		}

		switch strategy {
		case "playwright":
			return waitPlaywright(page, state, timeoutMs, minWaitMs)
		case "perf":
			return waitPerf(page, state, timeoutMs, minWaitMs)
		default:
			return nil, fmt.Errorf("invalid strategy (expected 'playwright' or 'perf')")
		}

	case "screenshot":
		pathArg, err := optionalString(args, "path", "")
		if err != nil {
			return nil, err
		}
		fullPage, err := optionalBool(args, "full_page", true)
		if err != nil {
			return nil, err
		}
		annotate, err := optionalBool(args, "annotate_refs", false)
		if err != nil {
			return nil, err
		}
		crop, err := optionalCrop(args)
		if err != nil {
			return nil, err
		}

		selector, err := optionalString(args, "selector", "")
		if err != nil {
			return nil, err
		}
		ariaRole, err := optionalString(args, "aria_role", "")
		if err != nil {
			return nil, err
		}
		ariaName, err := optionalString(args, "aria_name", "")
		if err != nil {
			return nil, err
		}
		nth, err := optionalInt(args, "nth", 1)
		if err != nil {
			return nil, err
		}
		padding, err := optionalInt(args, "padding_px", 10)
		if err != nil {
			return nil, err
		}
		targetTimeout, err := optionalInt(args, "timeout_ms", 5_000)
		if err != nil {
			return nil, err
		}

		hasTarget := strings.TrimSpace(selector) != "" || strings.TrimSpace(ariaRole) != "" || strings.TrimSpace(ariaName) != ""
		if crop != nil && hasTarget {
			return nil, errors.New("--crop cannot be combined with selector/aria targeting")
		}

		path, err := SafeArtifactPath(artifactDir, pathArg, fmt.Sprintf("screenshot-%d.png", NowMS()))
		if err != nil {
			return nil, err
		}

		opts := playwright.PageScreenshotOptions{Path: playwright.String(path), FullPage: playwright.Bool(fullPage)}
		var clip *playwright.Rect
		var spec TargetSpec

		if hasTarget {
			spec = TargetSpec{Selector: selector, AriaRole: ariaRole, AriaName: ariaName, Nth: nth, Timeout: targetTimeout}
			box, err := resolveBounds(page, spec)
			if err != nil {
				return nil, err
			}
			vp := viewportSize(page)
			clip, err = clipWithPadding(box, padding, vp)
			if err != nil {
				return nil, err
			}
			opts.Clip = clip
			opts.FullPage = playwright.Bool(false)
		}

		if crop != nil {
			opts.Clip = crop
			opts.FullPage = playwright.Bool(false)
		}

		if annotate {
			_ = DrawRefOverlay(page, 80, "simple")
			page.WaitForTimeout(50)
		}
		_, shotErr := page.Screenshot(opts)
		if annotate {
			_ = ClearRefOverlay(page, "simple")
		}
		if shotErr != nil {
			return nil, shotErr
		}

		res := RunResult{"path": path}
		if clip != nil {
			res["selector"] = selector
			res["aria_role"] = ariaRole
			res["aria_name"] = ariaName
			res["nth"] = spec.effectiveNth()
			res["clip"] = map[string]float64{"x": clip.X, "y": clip.Y, "width": clip.Width, "height": clip.Height}
		}
		return res, nil

	case "style_capture":
		pathArg, err := optionalStringAllowEmpty(args, "path", "")
		if err != nil {
			return nil, err
		}
		cssPathArg, err := optionalStringAllowEmpty(args, "css_path", "")
		if err != nil {
			return nil, err
		}
		mode, err := optionalString(args, "mode", "inline")
		if err != nil {
			return nil, err
		}
		mode = strings.ToLower(strings.TrimSpace(mode))
		if mode == "" {
			mode = "inline"
		}
		if mode != "inline" && mode != "bundle" {
			return nil, fmt.Errorf("invalid mode '%s' (expected inline or bundle)", mode)
		}
		selector, err := optionalString(args, "selector", "")
		if err != nil {
			return nil, err
		}
		maxNodes, err := optionalInt(args, "max_nodes", 1500)
		if err != nil {
			return nil, err
		}
		if maxNodes == 0 {
			maxNodes = 1500
		}
		includeAll, err := optionalBool(args, "include_all", false)
		if err != nil {
			return nil, err
		}
		var stripPtr *bool
		if _, hasStrip := args["strip"]; hasStrip {
			strip, err := optionalBool(args, "strip", true)
			if err != nil {
				return nil, err
			}
			stripPtr = &strip
		}
		properties, err := optionalStringSlice(args, "properties")
		if err != nil {
			return nil, err
		}

		if strings.TrimSpace(cssPathArg) != "" && mode != "bundle" {
			return nil, errors.New("--css-path requires --mode bundle")
		}

		path, err := SafeArtifactPath(artifactDir, pathArg, fmt.Sprintf("style-capture-%d.html", NowMS()))
		if err != nil {
			return nil, err
		}

		result, err := StyleCapture(page, StyleCaptureOptions{
			Mode:       mode,
			Selector:   selector,
			MaxNodes:   maxNodes,
			IncludeAll: includeAll,
			Properties: properties,
			Strip:      stripPtr,
		})
		if err != nil {
			return nil, err
		}
		if err := osWriteFile(path, []byte(result.HTML)); err != nil {
			return nil, err
		}

		res := RunResult{
			"path":        path,
			"html":        result.HTML,
			"css":         result.CSS,
			"mode":        result.Mode,
			"selector":    selector,
			"node_count":  result.NodeCount,
			"truncated":   result.Truncated,
			"include_all": includeAll,
		}
		if stripPtr != nil {
			res["strip"] = *stripPtr
		}
		if len(result.Properties) > 0 {
			res["properties"] = result.Properties
		}
		if strings.TrimSpace(cssPathArg) != "" {
			cssPath, err := SafeArtifactPath(artifactDir, cssPathArg, fmt.Sprintf("style-capture-%d.css", NowMS()))
			if err != nil {
				return nil, err
			}
			if err := osWriteFile(cssPath, []byte(result.CSS)); err != nil {
				return nil, err
			}
			res["css_path"] = cssPath
		}
		return res, nil
	case "save_html":
		includeHTML, err := optionalBool(args, "include_html", true)
		if err != nil {
			return nil, err
		}
		pathArg, err := optionalString(args, "path", "")
		if err != nil {
			return nil, err
		}
		path, err := SafeArtifactPath(artifactDir, pathArg, fmt.Sprintf("page-%d.html", NowMS()))
		if err != nil {
			return nil, err
		}
		html, err := page.Content()
		if err != nil {
			return nil, err
		}
		if err := osWriteFile(path, []byte(html)); err != nil {
			return nil, err
		}
		res := RunResult{"path": path}
		if includeHTML {
			res["html"] = html
		}
		return res, nil

	case "bounds":
		selector, err := optionalString(args, "selector", "")
		if err != nil {
			return nil, err
		}
		ariaRole, err := optionalString(args, "aria_role", "")
		if err != nil {
			return nil, err
		}
		ariaName, err := optionalString(args, "aria_name", "")
		if err != nil {
			return nil, err
		}
		nth, err := optionalInt(args, "nth", 1)
		if err != nil {
			return nil, err
		}
		timeoutMs, err := optionalInt(args, "timeout_ms", 5_000)
		if err != nil {
			return nil, err
		}

		spec := TargetSpec{Selector: selector, AriaRole: ariaRole, AriaName: ariaName, Nth: nth, Timeout: timeoutMs}
		box, err := resolveBounds(page, spec)
		if err != nil {
			return nil, err
		}
		return RunResult{
			"selector":  selector,
			"aria_role": ariaRole,
			"aria_name": ariaName,
			"nth":       spec.effectiveNth(),
			"x":         box.X,
			"y":         box.Y,
			"width":     box.Width,
			"height":    box.Height,
		}, nil

	case "js_eval":
		expression, err := requireString(args, "expression")
		if err != nil {
			return nil, err
		}
		format, err := optionalString(args, "format", "auto")
		if err != nil {
			return nil, err
		}
		selector, err := optionalString(args, "selector", "")
		if err != nil {
			return nil, err
		}
		ariaRole, err := optionalString(args, "aria_role", "")
		if err != nil {
			return nil, err
		}
		ariaName, err := optionalString(args, "aria_name", "")
		if err != nil {
			return nil, err
		}
		nth, err := optionalInt(args, "nth", 1)
		if err != nil {
			return nil, err
		}

		result, err := evaluateJS(page, expression, selector, ariaRole, ariaName, nth)
		if err != nil {
			return nil, err
		}

		res := RunResult{"result": result}
		if format != "auto" {
			res["format"] = format
		}
		return res, nil

	case "inject":
		script, err := optionalString(args, "script", "")
		if err != nil {
			return nil, err
		}
		style, err := optionalString(args, "style", "")
		if err != nil {
			return nil, err
		}
		file, err := optionalString(args, "file", "")
		if err != nil {
			return nil, err
		}
		waitMs, err := optionalInt(args, "wait_ms", 100)
		if err != nil {
			return nil, err
		}

		injected := map[string]bool{}
		if strings.TrimSpace(script) != "" {
			_, err := page.Evaluate(script)
			if err != nil {
				return nil, fmt.Errorf("script injection failed: %w", err)
			}
			injected["script"] = true
		}
		if strings.TrimSpace(style) != "" {
			styleJS := fmt.Sprintf(`(() => {
				const style = document.createElement('style');
				style.textContent = %q;
				document.head.appendChild(style);
				return true;
			})()`, style)
			_, err := page.Evaluate(styleJS)
			if err != nil {
				return nil, fmt.Errorf("style injection failed: %w", err)
			}
			injected["style"] = true
		}
		if strings.TrimSpace(file) != "" {
			content, err := os.ReadFile(file)
			if err != nil {
				return nil, fmt.Errorf("failed to read file %s: %w", file, err)
			}
			ext := strings.ToLower(filepath.Ext(file))
			if ext == ".css" {
				styleJS := fmt.Sprintf(`(() => {
					const style = document.createElement('style');
					style.textContent = %q;
					document.head.appendChild(style);
					return true;
				})()`, string(content))
				_, err = page.Evaluate(styleJS)
				if err != nil {
					return nil, fmt.Errorf("CSS file injection failed: %w", err)
				}
				injected["css_file"] = true
			} else {
				_, err := page.Evaluate(string(content))
				if err != nil {
					return nil, fmt.Errorf("JS file injection failed: %w", err)
				}
				injected["js_file"] = true
			}
		}

		if waitMs > 0 {
			page.WaitForTimeout(float64(waitMs))
		}

		return RunResult{"injected": injected}, nil

	case "asset_snapshot":
		pathArg, err := optionalString(args, "path", "")
		if err != nil {
			return nil, err
		}
		includeAssets, err := optionalBool(args, "include_assets", true)
		if err != nil {
			return nil, err
		}
		assetTypes, err := optionalStringSlice(args, "asset_types")
		if err != nil {
			return nil, err
		}
		maxDepth, err := optionalInt(args, "max_depth", 2)
		if err != nil {
			return nil, err
		}
		stripScripts, err := optionalBool(args, "strip_scripts", false)
		if err != nil {
			return nil, err
		}
		inlineThreshold, err := optionalInt(args, "inline_threshold", 10240)
		if err != nil {
			return nil, err
		}

		path, err := SafeArtifactPath(artifactDir, pathArg, fmt.Sprintf("asset-snapshot-%d.html", NowMS()))
		if err != nil {
			return nil, err
		}

		snapshot, err := createAssetSnapshot(page, includeAssets, assetTypes, maxDepth, stripScripts, inlineThreshold)
		if err != nil {
			return nil, err
		}

		if err := osWriteFile(path, []byte(snapshot.HTML)); err != nil {
			return nil, err
		}

		res := RunResult{
			"path":          path,
			"html":          snapshot.HTML,
			"assets_count":  snapshot.AssetCount,
			"assets_inline": snapshot.InlineCount,
			"assets_linked": snapshot.LinkedCount,
			"stripped":      stripScripts,
		}
		return res, nil

	case "visual_diff":
		baselineArg, err := requireString(args, "baseline_path")
		if err != nil {
			return nil, err
		}
		outputArg, err := optionalString(args, "output_path", "")
		if err != nil {
			return nil, err
		}
		tolerance, err := optionalFloat(args, "tolerance", 0.1)
		if err != nil {
			return nil, err
		}
		if tolerance < 0 || tolerance > 1 {
			return nil, fmt.Errorf("tolerance must be between 0 and 1")
		}
		pixelThreshold, err := optionalInt(args, "pixel_threshold", 10)
		if err != nil {
			return nil, err
		}
		highlight, err := optionalBool(args, "highlight", true)
		if err != nil {
			return nil, err
		}
		ignoreRegions, err := optionalIgnoreRegions(args, "ignore_regions")
		if err != nil {
			return nil, err
		}

		baselinePath, err := resolveInputPath(artifactDir, baselineArg)
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(baselinePath) == "" {
			return nil, errors.New("baseline path is required")
		}

		outputPath := ""
		if strings.TrimSpace(outputArg) != "" {
			outputPath, err = SafeArtifactPath(artifactDir, outputArg, outputArg)
			if err != nil {
				return nil, err
			}
		} else if highlight {
			outputPath, err = SafeArtifactPath(artifactDir, "", fmt.Sprintf("diff-%d.png", NowMS()))
			if err != nil {
				return nil, err
			}
		}

		diffResult, err := compareScreenshots(page, baselinePath, outputPath, tolerance, pixelThreshold, highlight, ignoreRegions)
		if err != nil {
			return nil, err
		}

		res := RunResult{
			"passed":           diffResult.Passed,
			"different_pixels": diffResult.DifferentPixels,
			"diff_percentage":  diffResult.DiffPercentage,
			"baseline_path":    baselinePath,
			"tolerance":        tolerance,
			"pixel_threshold":  pixelThreshold,
			"highlight":        highlight,
		}
		if diffResult.OutputPath != "" {
			res["output_path"] = diffResult.OutputPath
		}
		if len(ignoreRegions) > 0 {
			res["ignored_regions"] = len(ignoreRegions)
		}
		return res, nil

	case "diff_images":
		return runDiffImages(page, args, artifactDir)

	case "save_dom_baseline":
		pathArg, err := requireString(args, "path")
		if err != nil {
			return nil, err
		}
		engine, err := optionalString(args, "engine", "simple")
		if err != nil {
			return nil, err
		}
		maxItems, err := optionalInt(args, "max_items", 200)
		if err != nil {
			return nil, err
		}

		path, err := SafeArtifactPath(artifactDir, pathArg, pathArg)
		if err != nil {
			return nil, err
		}
		snap, err := CaptureDomSnapshot(page, engine, maxItems)
		if err != nil {
			return nil, err
		}
		if err := WriteDomSnapshot(path, snap); err != nil {
			return nil, err
		}
		return RunResult{"path": path, "dom_baseline_saved": true, "engine": snap.Engine, "items": len(snap.Items)}, nil

	case "dom_diff":
		baselineArg, err := requireString(args, "baseline_path")
		if err != nil {
			return nil, err
		}
		engine, err := optionalString(args, "engine", "simple")
		if err != nil {
			return nil, err
		}
		maxItems, err := optionalInt(args, "max_items", 200)
		if err != nil {
			return nil, err
		}

		baselinePath, err := resolveInputPath(artifactDir, baselineArg)
		if err != nil {
			return nil, err
		}
		before, err := ReadDomSnapshot(baselinePath)
		if err != nil {
			return nil, err
		}
		after, err := CaptureDomSnapshot(page, engine, maxItems)
		if err != nil {
			return nil, err
		}
		diff := DiffDomSnapshots(before, after)
		return RunResult{
			"baseline_path": baselinePath,
			"engine":        engine,
			"max_items":     maxItems,
			"added":         diff.Added,
			"removed":       diff.Removed,
			"changed":       diff.Changed,
			"added_count":   diff.AddedCount,
			"removed_count": diff.RemovedCount,
			"changed_count": diff.ChangedCount,
		}, nil

	case "save_baseline":
		pathArg, err := requireString(args, "path")
		if err != nil {
			return nil, err
		}
		fullPage, err := optionalBool(args, "full_page", true)
		if err != nil {
			return nil, err
		}

		path, err := SafeArtifactPath(artifactDir, pathArg, pathArg)
		if err != nil {
			return nil, err
		}

		opts := playwright.PageScreenshotOptions{Path: playwright.String(path), FullPage: playwright.Bool(fullPage)}

		selector, _ := args["selector"].(string)
		ariaRole, _ := args["aria_role"].(string)
		ariaName, _ := args["aria_name"].(string)
		nth, _ := args["nth"].(int)
		padding, _ := args["padding_px"].(int)
		targetTimeout, _ := args["timeout_ms"].(int)

		hasTarget := strings.TrimSpace(selector) != "" || strings.TrimSpace(ariaRole) != "" || strings.TrimSpace(ariaName) != ""

		if hasTarget {
			spec := TargetSpec{Selector: selector, AriaRole: ariaRole, AriaName: ariaName, Nth: nth, Timeout: targetTimeout}
			box, err := resolveBounds(page, spec)
			if err != nil {
				return nil, err
			}
			vp := viewportSize(page)
			clip, err := clipWithPadding(box, padding, vp)
			if err != nil {
				return nil, err
			}
			opts.Clip = clip
			opts.FullPage = playwright.Bool(false)
		}

		_, err = page.Screenshot(opts)
		if err != nil {
			return nil, err
		}

		return RunResult{"path": path, "baseline_saved": true}, nil
	}

	return nil, fmt.Errorf("unknown call '%s'", name)
}

func RunActions(page playwright.Page, calls []map[string]interface{}, artifactDir string) (ActionsResult, error) {
	results := []map[string]interface{}{}
	snapshotText := ""

	for _, call := range calls {
		nameVal, ok := call["name"]
		if !ok {
			return ActionsResult{}, errors.New("each call must include name")
		}
		name, ok := nameVal.(string)
		if !ok || strings.TrimSpace(name) == "" {
			return ActionsResult{}, errors.New("each call must include non-empty string 'name'")
		}
		argsVal, ok := call["arguments"]
		if !ok || argsVal == nil {
			argsVal = map[string]interface{}{}
		}
		args, ok := argsVal.(map[string]interface{})
		if !ok {
			return ActionsResult{}, errors.New("call 'arguments' must be an object")
		}

		res, err := RunCall(page, name, args, artifactDir)
		if err != nil {
			return ActionsResult{}, err
		}
		entry := map[string]interface{}{"name": name, "result": res}
		results = append(results, entry)
		if name == "snapshot" {
			if snap, ok := res["snapshot"].(string); ok {
				snapshotText = snap
			}
		}
	}
	return ActionsResult{Results: results, Snapshot: snapshotText}, nil
}

func getWaitUntil(value string) *playwright.WaitUntilState {
	v := strings.ToLower(strings.TrimSpace(value))
	switch v {
	case "commit":
		return playwright.WaitUntilStateCommit
	case "networkidle":
		return playwright.WaitUntilStateNetworkidle
	case "domcontentloaded":
		return playwright.WaitUntilStateDomcontentloaded
	default:
		return playwright.WaitUntilStateLoad
	}
}

func waitPlaywright(page playwright.Page, state string, timeoutMs int, minWaitMs int) (RunResult, error) {
	start := time.Now()
	if minWaitMs > 0 {
		page.WaitForTimeout(float64(minWaitMs))
	}
	var loadState *playwright.LoadState
	switch strings.ToLower(state) {
	case "domcontentloaded", "commit":
		loadState = playwright.LoadStateDomcontentloaded
	case "networkidle":
		loadState = playwright.LoadStateNetworkidle
	default:
		loadState = playwright.LoadStateLoad
	}
	err := page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{State: loadState, Timeout: playwright.Float(float64(timeoutMs))})
	timedOut := isTimeout(err)
	if err != nil && !timedOut {
		return nil, err
	}
	readyState := ""
	if rs, err := page.Evaluate("() => document.readyState"); err == nil {
		if str, ok := rs.(string); ok {
			readyState = str
		}
	}
	waited := int(time.Since(start).Milliseconds())
	if waited < minWaitMs {
		waited = minWaitMs
	}
	return RunResult{
		"ok":          !timedOut,
		"strategy":    "playwright",
		"state":       state,
		"timed_out":   timedOut,
		"waited_ms":   waited,
		"ready_state": readyState,
	}, nil
}

func waitPerf(page playwright.Page, state string, timeoutMs int, minWaitMs int) (RunResult, error) {
	pollInterval := 50 * time.Millisecond
	start := time.Now()
	if minWaitMs > 0 {
		page.WaitForTimeout(float64(minWaitMs))
	}
	deadline := start.Add(time.Duration(timeoutMs) * time.Millisecond)
	lastReady := ""
	lastPending := 0
	success := false

	for time.Now().Before(deadline) {
		data, err := page.Evaluate(perfLoadStateJS)
		if err == nil {
			if m, ok := data.(map[string]interface{}); ok {
				if rs, ok := m["readyState"].(string); ok {
					lastReady = rs
				}
				if pending, ok := asInt(m["pendingRequests"]); ok {
					lastPending = pending
				}
			}
		}

		if lastReady != "" && readyStateSatisfies(lastReady, state) && lastPending == 0 {
			success = true
			break
		}
		page.WaitForTimeout(float64(pollInterval.Milliseconds()))
	}

	waited := int(time.Since(start).Milliseconds())
	return RunResult{
		"ok":               success,
		"strategy":         "perf",
		"state":            state,
		"timed_out":        !success,
		"waited_ms":        waited,
		"ready_state":      lastReady,
		"pending_requests": lastPending,
	}, nil
}

const perfLoadStateJS = `() => {
  const doc = globalThis.document;
  const perf = globalThis.performance;
  const readyState = doc && typeof doc.readyState === "string" ? doc.readyState : "unknown";
  if (!perf || typeof perf.getEntriesByType !== "function" || typeof perf.now !== "function") {
    return { readyState, pendingRequests: 0 };
  }

  const now = perf.now();
  const resources = perf.getEntriesByType("resource") || [];

  const adPatterns = [
    "doubleclick.net",
    "googlesyndication.com",
    "googletagmanager.com",
    "google-analytics.com",
    "facebook.net",
    "connect.facebook.net",
    "analytics",
    "ads",
    "tracking",
    "pixel",
    "hotjar.com",
    "clarity.ms",
    "mixpanel.com",
    "segment.com",
    "newrelic.com",
    "nr-data.net",
    "/tracker/",
    "/collector/",
    "/beacon/",
    "/telemetry/",
    "/log/",
    "/events/",
    "/track.",
    "/metrics/",
  ];

  const nonCriticalTypes = ["img", "image", "icon", "font"];

  let pending = 0;
  for (const entry of resources) {
    if (!entry || entry.responseEnd !== 0) continue;
    const url = String(entry.name || "");

    if (!url || url.startsWith("data:") || url.length > 500) continue;
    if (adPatterns.some((p) => url.includes(p))) continue;

    const loadingDuration = now - (entry.startTime || 0);
    if (loadingDuration > 10000) continue;

    const resourceType = String(entry.initiatorType || "unknown");
    if (nonCriticalTypes.includes(resourceType) && loadingDuration > 3000) continue;

    const isImageUrl = /\.(jpg|jpeg|png|gif|webp|svg|ico)(\?|$)/i.test(url);
    if (isImageUrl && loadingDuration > 3000) continue;

    pending++;
  }
  return { readyState, pendingRequests: pending };
}`

func readyStateSatisfies(ready string, state string) bool {
	rs := strings.ToLower(ready)
	if state == "domcontentloaded" || state == "commit" {
		return rs == "interactive" || rs == "complete"
	}
	return rs == "complete"
}

func requireString(args map[string]interface{}, key string) (string, error) {
	raw, ok := args[key]
	if !ok {
		return "", fmt.Errorf("expected non-empty string '%s'", key)
	}
	str, ok := raw.(string)
	if !ok || strings.TrimSpace(str) == "" {
		return "", fmt.Errorf("expected non-empty string '%s'", key)
	}
	return str, nil
}

func optionalString(args map[string]interface{}, key string, def string) (string, error) {
	raw, ok := args[key]
	if !ok {
		return def, nil
	}
	str, ok := raw.(string)
	if !ok || strings.TrimSpace(str) == "" {
		return "", fmt.Errorf("expected string '%s'", key)
	}
	return str, nil
}

func optionalBool(args map[string]interface{}, key string, def bool) (bool, error) {
	raw, ok := args[key]
	if !ok {
		return def, nil
	}
	b, ok := raw.(bool)
	if !ok {
		return false, fmt.Errorf("expected boolean '%s'", key)
	}
	return b, nil
}

func optionalInt(args map[string]interface{}, key string, def int) (int, error) {
	raw, ok := args[key]
	if !ok {
		return def, nil
	}
	if i, ok := asInt(raw); ok {
		if i < 0 {
			return 0, fmt.Errorf("expected non-negative integer '%s'", key)
		}
		return i, nil
	}
	return 0, fmt.Errorf("expected non-negative integer '%s'", key)
}

func asInt(v interface{}) (int, bool) {
	switch t := v.(type) {
	case int:
		return t, true
	case int64:
		return int(t), true
	case float64:
		return int(t), true
	case float32:
		return int(t), true
	default:
		return 0, false
	}
}

func optionalIgnoreRegions(args map[string]interface{}, key string) ([]image.Rectangle, error) {
	raw, ok := args[key]
	if !ok || raw == nil {
		return nil, nil
	}
	switch list := raw.(type) {
	case []map[string]int:
		regions := make([]image.Rectangle, 0, len(list))
		for _, entry := range list {
			w := entry["w"]
			h := entry["h"]
			if w <= 0 || h <= 0 {
				continue
			}
			regions = append(regions, image.Rect(entry["x"], entry["y"], entry["x"]+w, entry["y"]+h))
		}
		return regions, nil
	case []interface{}:
		regions := make([]image.Rectangle, 0, len(list))
		for _, item := range list {
			entry, ok := item.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("expected object in '%s'", key)
			}
			x, ok := asInt(entry["x"])
			if !ok {
				return nil, fmt.Errorf("expected integer x in '%s'", key)
			}
			y, ok := asInt(entry["y"])
			if !ok {
				return nil, fmt.Errorf("expected integer y in '%s'", key)
			}
			w, ok := asInt(entry["w"])
			if !ok {
				return nil, fmt.Errorf("expected integer w in '%s'", key)
			}
			h, ok := asInt(entry["h"])
			if !ok {
				return nil, fmt.Errorf("expected integer h in '%s'", key)
			}
			if w <= 0 || h <= 0 {
				continue
			}
			regions = append(regions, image.Rect(x, y, x+w, y+h))
		}
		return regions, nil
	default:
		return nil, fmt.Errorf("expected array '%s'", key)
	}
}

func optionalCrop(args map[string]interface{}) (*playwright.Rect, error) {
	raw, ok := args["crop"]
	if !ok || raw == nil {
		return nil, nil
	}

	toVals := func(seq []interface{}) ([]int, error) {
		if len(seq) != 4 {
			return nil, errors.New("crop must have 4 items: x,y,width,height")
		}
		vals := make([]int, 4)
		for i, v := range seq {
			n, ok := asInt(v)
			if !ok || n < 0 {
				return nil, errors.New("crop values must be non-negative integers")
			}
			vals[i] = n
		}
		if vals[2] < 1 || vals[3] < 1 {
			return nil, errors.New("crop width/height must be positive")
		}
		if vals[2] > 2000 {
			vals[2] = 2000
		}
		if vals[3] > 2000 {
			vals[3] = 2000
		}
		return vals, nil
	}

	var vals []int
	switch t := raw.(type) {
	case string:
		parts := strings.Split(t, ",")
		if len(parts) != 4 {
			return nil, errors.New("--crop must be x,y,width,height")
		}
		seq := make([]interface{}, 0, 4)
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p == "" {
				return nil, errors.New("--crop must be x,y,width,height")
			}
			seq = append(seq, p)
		}
		valsSeq := make([]interface{}, 0, 4)
		for _, s := range seq {
			num, err := strconv.Atoi(s.(string))
			if err != nil {
				return nil, errors.New("crop values must be integers")
			}
			valsSeq = append(valsSeq, num)
		}
		var err error
		vals, err = toVals(valsSeq)
		if err != nil {
			return nil, err
		}
	case map[string]interface{}:
		seq := []interface{}{t["x"], t["y"], t["width"], t["height"]}
		v, err := toVals(seq)
		if err != nil {
			return nil, err
		}
		vals = v
	case []interface{}:
		v, err := toVals(t)
		if err != nil {
			return nil, err
		}
		vals = v
	default:
		return nil, errors.New("crop must be string, array, or object")
	}

	return &playwright.Rect{X: float64(vals[0]), Y: float64(vals[1]), Width: float64(vals[2]), Height: float64(vals[3])}, nil
}

func safeTitle(page playwright.Page) string {
	if page == nil {
		return ""
	}
	if title, err := page.Title(); err == nil {
		return title
	}
	return ""
}

func isTimeout(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "timeout")
}

func osWriteFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0o644)
}
