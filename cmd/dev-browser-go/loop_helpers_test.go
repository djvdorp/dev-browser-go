package main

import (
	"testing"

	devbrowser "github.com/joshp123/dev-browser-go/internal/devbrowser"
)

func TestAttachSelectorFailureContext_MinimalOmitsPreviewKeepsEvalError(t *testing.T) {
	res := devbrowser.AssertResult{
		Passed: false,
		FailedChecks: []devbrowser.AssertFailedCheck{{
			ID:      "selectors.min",
			Message: "selector missing",
			Context: map[string]any{"selector": "#root"},
		}},
	}
	evalErr := map[string]string{"#root": "boom"}
	called := 0
	attachSelectorFailureContext(&res, devbrowser.ArtifactModeMinimal, evalErr, func(sel string) (any, error) {
		called++
		return map[string]any{"x": 1}, nil
	})
	if called != 0 {
		t.Fatalf("expected previewFn not called")
	}
	ctx := res.FailedChecks[0].Context
	if ctx["evalError"] != "boom" {
		t.Fatalf("expected evalError to be set")
	}
	if _, ok := ctx["preview"]; ok {
		t.Fatalf("expected preview to be omitted")
	}
}

func TestAttachSelectorFailureContext_FullIncludesPreview(t *testing.T) {
	res := devbrowser.AssertResult{
		Passed: false,
		FailedChecks: []devbrowser.AssertFailedCheck{{
			ID:      "selectors.max",
			Message: "selector too many",
			Context: map[string]any{"selector": ".err"},
		}},
	}
	evalErr := map[string]string{":missing": "ignored"}
	called := 0
	attachSelectorFailureContext(&res, devbrowser.ArtifactModeFull, evalErr, func(sel string) (any, error) {
		called++
		return []string{"a", "b"}, nil
	})
	if called != 1 {
		t.Fatalf("expected previewFn called once")
	}
	ctx := res.FailedChecks[0].Context
	if _, ok := ctx["preview"]; !ok {
		t.Fatalf("expected preview to be included")
	}
	if _, ok := ctx["evalError"]; ok {
		t.Fatalf("did not expect evalError")
	}
}
