package main

import "testing"

func TestParseIgnoreRegions(t *testing.T) {
	regions := parseIgnoreRegions("10,20,30,40; 5,6,7,8")
	if len(regions) != 2 {
		t.Fatalf("expected 2 regions, got %d", len(regions))
	}
	if regions[0]["x"] != 10 || regions[0]["y"] != 20 || regions[0]["w"] != 30 || regions[0]["h"] != 40 {
		t.Fatalf("unexpected first region: %+v", regions[0])
	}
	if regions[1]["x"] != 5 || regions[1]["y"] != 6 || regions[1]["w"] != 7 || regions[1]["h"] != 8 {
		t.Fatalf("unexpected second region: %+v", regions[1])
	}
}

func TestParseIgnoreRegionsSkipsInvalid(t *testing.T) {
	regions := parseIgnoreRegions("1,2,3; a,b,c,d")
	if len(regions) != 1 {
		t.Fatalf("expected 1 region, got %d", len(regions))
	}
}
