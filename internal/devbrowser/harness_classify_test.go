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

func TestFirstNonEmptyLine(t *testing.T) {
	if got := firstNonEmptyLine("\n\n hello \nworld"); got != "hello" {
		t.Fatalf("expected hello, got %q", got)
	}
}
