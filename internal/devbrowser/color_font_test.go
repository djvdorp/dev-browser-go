package devbrowser

import (
	"testing"
)

func TestColorInfoOptsIncludeTransparent(t *testing.T) {
	opts := ColorInfoOptions{
		IncludeTransparent: true,
	}
	if !opts.IncludeTransparent {
		t.Fatalf("expected IncludeTransparent=true")
	}
}

func TestColorInfoOptsExcludeTransparent(t *testing.T) {
	opts := ColorInfoOptions{
		IncludeTransparent: false,
	}
	if opts.IncludeTransparent {
		t.Fatalf("expected IncludeTransparent=false")
	}
}

func TestColorInfoOptsDefault(t *testing.T) {
	opts := ColorInfoOptions{}
	if opts.IncludeTransparent {
		t.Fatalf("expected default IncludeTransparent=false")
	}
}
