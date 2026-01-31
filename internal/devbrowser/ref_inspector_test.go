package devbrowser

import (
	"testing"
)

func TestInspectRefOptsStructure(t *testing.T) {
	opts := RefInspectOptions{
		StyleProps: []string{"color", "font-size"},
	}
	if len(opts.StyleProps) != 2 {
		t.Fatalf("expected 2 style props, got %d", len(opts.StyleProps))
	}
	if opts.StyleProps[0] != "color" || opts.StyleProps[1] != "font-size" {
		t.Fatalf("unexpected style props: %v", opts.StyleProps)
	}
}

func TestInspectRefOptsEmpty(t *testing.T) {
	opts := RefInspectOptions{}
	if opts.StyleProps != nil {
		t.Fatalf("expected nil StyleProps, got %v", opts.StyleProps)
	}
}
