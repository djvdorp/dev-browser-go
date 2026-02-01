package devbrowser

import "testing"

func TestClassifyViteOverlay(t *testing.T) {
	if got := ClassifyViteOverlay("Failed to resolve import \"react\""); got != "missing-module" {
		t.Fatalf("expected missing-module, got %q", got)
	}
	if got := ClassifyViteOverlay("SyntaxError: Unexpected token 'export'"); got != "syntax-error" {
		t.Fatalf("expected syntax-error, got %q", got)
	}
}

func TestClassifyHarnessError(t *testing.T) {
	tests := []struct {
		typ      string
		message  string
		expected string
	}{
		{"unhandledrejection", "fetch failed", "unhandledrejection-fetch"},
		{"unhandledrejection", "FETCH error", "unhandledrejection-fetch"},
		{"error", "TypeError: cannot read property", "type-error"},
		{"error", "ReferenceError: x is not defined", "reference-error"},
		{"error", "SyntaxError: unexpected token", "syntax-error"},
		{"error", "some generic error", "unknown"},
		{"unhandledrejection", "something else", "unknown"},
	}
	for _, tt := range tests {
		got := ClassifyHarnessError(tt.typ, tt.message)
		if got != tt.expected {
			t.Errorf("ClassifyHarnessError(%q, %q) = %q, want %q", tt.typ, tt.message, got, tt.expected)
		}
	}
}

func TestFirstNonEmptyLine(t *testing.T) {
	if got := firstNonEmptyLine("\n\n hello \nworld"); got != "hello" {
		t.Fatalf("expected hello, got %q", got)
	}
}
