package devbrowser

import "testing"

func TestOptionalIgnoreRegionsMapSlice(t *testing.T) {
	args := map[string]interface{}{
		"ignore_regions": []map[string]int{
			{"x": 1, "y": 2, "w": 3, "h": 4},
		},
	}
	regions, err := optionalIgnoreRegions(args, "ignore_regions")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(regions) != 1 {
		t.Fatalf("expected 1 region, got %d", len(regions))
	}
	got := regions[0]
	if got.Min.X != 1 || got.Min.Y != 2 || got.Max.X != 4 || got.Max.Y != 6 {
		t.Fatalf("unexpected region: %+v", got)
	}
}

func TestOptionalIgnoreRegionsInterfaceSlice(t *testing.T) {
	args := map[string]interface{}{
		"ignore_regions": []interface{}{
			map[string]interface{}{"x": 5, "y": 6, "w": 7, "h": 8},
		},
	}
	regions, err := optionalIgnoreRegions(args, "ignore_regions")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(regions) != 1 {
		t.Fatalf("expected 1 region, got %d", len(regions))
	}
	got := regions[0]
	if got.Min.X != 5 || got.Min.Y != 6 || got.Max.X != 12 || got.Max.Y != 14 {
		t.Fatalf("unexpected region: %+v", got)
	}
}
