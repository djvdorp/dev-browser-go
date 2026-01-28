package devbrowser

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/playwright-community/playwright-go"
)

func runDiffImages(page playwright.Page, args map[string]interface{}, artifactDir string) (RunResult, error) {
	beforeArg, err := optionalString(args, "before_path", "")
	if err != nil {
		return nil, err
	}
	afterArg, err := optionalString(args, "after_path", "")
	if err != nil {
		return nil, err
	}
	diffArg, err := optionalString(args, "diff_path", "")
	if err != nil {
		return nil, err
	}
	afterWaitMs, err := optionalInt(args, "after_wait_ms", 0)
	if err != nil {
		return nil, err
	}
	threshold, err := optionalInt(args, "threshold", 0)
	if err != nil {
		return nil, err
	}
	if threshold > 255 {
		return nil, fmt.Errorf("threshold must be between 0 and 255")
	}

	ts := NowMS()
	beforeDefault := fmt.Sprintf("before-%d.png", ts)
	afterDefault := fmt.Sprintf("after-%d.png", ts+1)

	beforePath, beforeCapture, err := resolveDiffPath(artifactDir, beforeArg, "before")
	if err != nil {
		return nil, err
	}
	if beforeCapture {
		beforePath, err = captureScreenshotPath(page, args, artifactDir, beforeArg, beforeDefault, RunCall)
		if err != nil {
			return nil, err
		}
	}

	afterPath, afterCapture, err := resolveDiffPath(artifactDir, afterArg, "after")
	if err != nil {
		return nil, err
	}
	if afterCapture && afterWaitMs > 0 {
		page.WaitForTimeout(float64(afterWaitMs))
	}
	if afterCapture {
		afterPath, err = captureScreenshotPath(page, args, artifactDir, afterArg, afterDefault, RunCall)
		if err != nil {
			return nil, err
		}
	}

	diffPath, err := SafeArtifactPath(artifactDir, diffArg, fmt.Sprintf("diff-%d.png", ts+2))
	if err != nil {
		return nil, err
	}

	beforeImg, err := loadImage(beforePath)
	if err != nil {
		return nil, err
	}
	afterImg, err := loadImage(afterPath)
	if err != nil {
		return nil, err
	}
	diffImg, stats, err := diffImages(beforeImg, afterImg, uint8(threshold))
	if err != nil {
		return nil, err
	}
	if err := writePNG(diffPath, diffImg); err != nil {
		return nil, err
	}

	diffRatio := 0.0
	if stats.TotalPixels > 0 {
		diffRatio = float64(stats.ChangedPixels) / float64(stats.TotalPixels)
	}

	return RunResult{
		"before_path":     beforePath,
		"after_path":      afterPath,
		"diff_path":       diffPath,
		"path":            diffPath,
		"before_captured": beforeCapture,
		"after_captured":  afterCapture,
		"changed_pixels":  stats.ChangedPixels,
		"total_pixels":    stats.TotalPixels,
		"diff_ratio":      diffRatio,
		"width":           stats.Width,
		"height":          stats.Height,
		"threshold":       threshold,
		"match":           stats.ChangedPixels == 0,
	}, nil
}

func resolveDiffPath(artifactDir, pathArg, label string) (string, bool, error) {
	resolved, err := resolveInputPath(artifactDir, pathArg)
	if err != nil {
		return "", false, err
	}
	if strings.TrimSpace(resolved) == "" {
		return "", true, nil
	}
	info, err := os.Stat(resolved)
	if err == nil {
		if info.IsDir() {
			return "", false, fmt.Errorf("%s path is a directory: %s", label, resolved)
		}
		return resolved, false, nil
	}
	if os.IsNotExist(err) {
		return resolved, true, nil
	}
	return "", false, err
}

func captureScreenshotPath(
	page playwright.Page,
	args map[string]interface{},
	artifactDir, pathArg, defaultName string,
	callFn func(playwright.Page, string, map[string]interface{}, string) (RunResult, error),
) (string, error) {
	shotArgs := cloneArgs(args)
	if strings.TrimSpace(pathArg) != "" {
		shotArgs["path"] = pathArg
	} else {
		shotArgs["path"] = defaultName
	}
	if callFn == nil {
		return "", errors.New("screenshot call is not configured")
	}
	res, err := callFn(page, "screenshot", shotArgs, artifactDir)
	if err != nil {
		return "", err
	}
	path, ok := res["path"].(string)
	if !ok || strings.TrimSpace(path) == "" {
		return "", errors.New("screenshot did not return path")
	}
	return path, nil
}

func cloneArgs(args map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(args))
	for key, value := range args {
		out[key] = value
	}
	return out
}
