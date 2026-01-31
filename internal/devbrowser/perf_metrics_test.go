package devbrowser

import (
	"testing"
)

func TestPerfMetricsOptsDefaults(t *testing.T) {
	opts := PerfMetricsOptions{}
	if opts.SampleMs != 0 {
		t.Fatalf("expected default SampleMs=0, got %d", opts.SampleMs)
	}
	if opts.TopN != 0 {
		t.Fatalf("expected default TopN=0, got %d", opts.TopN)
	}
}

func TestPerfMetricsOptsCustom(t *testing.T) {
	opts := PerfMetricsOptions{
		SampleMs: 2000,
		TopN:     10,
	}
	if opts.SampleMs != 2000 {
		t.Fatalf("expected SampleMs=2000, got %d", opts.SampleMs)
	}
	if opts.TopN != 10 {
		t.Fatalf("expected TopN=10, got %d", opts.TopN)
	}
}
