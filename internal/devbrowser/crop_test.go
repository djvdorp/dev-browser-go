package devbrowser

import (
	"testing"

	"github.com/playwright-community/playwright-go"
)

func TestOptionalCropString(t *testing.T) {
	crop, err := optionalCrop(map[string]interface{}{"crop": "10,20,4000,4000"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expect := playwright.Rect{X: 10, Y: 20, Width: 2000, Height: 2000}
	if *crop != expect {
		t.Fatalf("expected %+v got %+v", expect, *crop)
	}
}

func TestOptionalCropObject(t *testing.T) {
	crop, err := optionalCrop(map[string]interface{}{"crop": map[string]interface{}{"x": 5, "y": 6, "width": 7, "height": 8}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expect := playwright.Rect{X: 5, Y: 6, Width: 7, Height: 8}
	if *crop != expect {
		t.Fatalf("expected %+v got %+v", expect, *crop)
	}
}

func TestOptionalCropInvalid(t *testing.T) {
	if _, err := optionalCrop(map[string]interface{}{"crop": "bad"}); err == nil {
		t.Fatalf("expected error")
	}
	if _, err := optionalCrop(map[string]interface{}{"crop": "1,2,3,-1"}); err == nil {
		t.Fatalf("expected error")
	}
}
