package main

import (
	"fmt"
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

func TestAttachSelectorFailureContext_PreviewFnError(t *testing.T) {
	res := devbrowser.AssertResult{
		Passed: false,
		FailedChecks: []devbrowser.AssertFailedCheck{{
			ID:      "selectors.min",
			Message: "selector missing",
			Context: map[string]any{"selector": "#err"},
		}},
	}
	evalErr := map[string]string{}
	called := 0
	attachSelectorFailureContext(&res, devbrowser.ArtifactModeFull, evalErr, func(sel string) (any, error) {
		called++
		return nil, fmt.Errorf("preview error")
	})
	if called != 1 {
		t.Fatalf("expected previewFn called once")
	}
	ctx := res.FailedChecks[0].Context
	// Error should be silently ignored; preview not added.
	if _, ok := ctx["preview"]; ok {
		t.Fatalf("expected preview to be omitted on error")
	}
}

func TestAttachSelectorFailureContext_NilContext(t *testing.T) {
	res := devbrowser.AssertResult{
		Passed: false,
		FailedChecks: []devbrowser.AssertFailedCheck{{
			ID:      "selectors.min",
			Message: "selector missing",
			Context: nil, // nil context - no selector available
		}},
	}
	evalErr := map[string]string{"#root": "not found"}
	// Should not panic with nil context, even though there's no selector to process.
	attachSelectorFailureContext(&res, devbrowser.ArtifactModeMinimal, evalErr, nil)
	// Since there's no selector in the nil context, the function skips processing
	// and Context remains nil (no data to attach).
	ctx := res.FailedChecks[0].Context
	if ctx != nil {
		t.Fatalf("expected context to remain nil when no selector present")
	}
}

func TestAttachSelectorFailureContext_FullWithBothEvalErrorAndPreview(t *testing.T) {
	res := devbrowser.AssertResult{
		Passed: false,
		FailedChecks: []devbrowser.AssertFailedCheck{{
			ID:      "selectors.min",
			Message: "selector missing",
			Context: map[string]any{"selector": "#missing"},
		}},
	}
	evalErr := map[string]string{"#missing": "eval failed"}
	called := 0
	attachSelectorFailureContext(&res, devbrowser.ArtifactModeFull, evalErr, func(sel string) (any, error) {
		called++
		return map[string]any{"matched": 0}, nil
	})
	if called != 1 {
		t.Fatalf("expected previewFn called once")
	}
	ctx := res.FailedChecks[0].Context
	if ctx["evalError"] != "eval failed" {
		t.Fatalf("expected evalError to be set")
	}
	if _, ok := ctx["preview"]; !ok {
		t.Fatalf("expected preview to be included in full mode")
	}
}

func TestAttachSelectorFailureContext_NoneStillIncludesEvalError(t *testing.T) {
	res := devbrowser.AssertResult{
		Passed: false,
		FailedChecks: []devbrowser.AssertFailedCheck{{
			ID:      "selectors.min",
			Message: "selector missing",
			Context: map[string]any{"selector": "#test"},
		}},
	}
	evalErr := map[string]string{"#test": "error"}
	called := 0
	attachSelectorFailureContext(&res, devbrowser.ArtifactModeNone, evalErr, func(sel string) (any, error) {
		called++
		return map[string]any{"x": 1}, nil
	})
	if called != 0 {
		t.Fatalf("expected previewFn not called in none mode")
	}
	ctx := res.FailedChecks[0].Context
	if ctx["evalError"] != "error" {
		t.Fatalf("expected evalError to be set even in none mode")
	}
	if _, ok := ctx["preview"]; ok {
		t.Fatalf("expected preview to be omitted in none mode")
	}
}
