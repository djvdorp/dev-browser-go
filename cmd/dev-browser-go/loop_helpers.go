package main

import (
	"strings"

	devbrowser "github.com/joshp123/dev-browser-go/internal/devbrowser"
)

func attachSelectorFailureContext(result *devbrowser.AssertResult, mode devbrowser.ArtifactMode, selectorEvalErr map[string]string, previewFn func(sel string) (any, error)) {
	if result == nil {
		return
	}
	for i := range result.FailedChecks {
		id := result.FailedChecks[i].ID
		if id != "selectors.min" && id != "selectors.max" {
			continue
		}
		ctx := result.FailedChecks[i].Context
		if ctx == nil {
			ctx = map[string]any{}
		}
		selRaw, _ := ctx["selector"].(string)
		selStr := strings.TrimSpace(selRaw)
		if selStr == "" {
			continue
		}
		// evalError is deterministic; keep it regardless of artifact mode.
		if errMsg, ok := selectorEvalErr[selStr]; ok {
			ctx["evalError"] = errMsg
		}

		// preview reflects live page state and can be non-deterministic; only include when artifacts are full.
		if mode == devbrowser.ArtifactModeFull && previewFn != nil {
			if preview, err := previewFn(selStr); err == nil {
				ctx["preview"] = preview
			}
		}

		result.FailedChecks[i].Context = ctx
	}
}
