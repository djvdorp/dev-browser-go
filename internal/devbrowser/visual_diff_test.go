package devbrowser

import (
	"image"
	"image/color"
	"testing"
)

func TestDiffImagesNoChanges(t *testing.T) {
	before := image.NewRGBA(image.Rect(0, 0, 2, 2))
	after := image.NewRGBA(image.Rect(0, 0, 2, 2))
	fillRGBA(before, color.RGBA{R: 10, G: 20, B: 30, A: 255})
	fillRGBA(after, color.RGBA{R: 10, G: 20, B: 30, A: 255})

	_, stats, err := diffImages(before, after, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.ChangedPixels != 0 {
		t.Fatalf("expected 0 changed pixels, got %d", stats.ChangedPixels)
	}
	if stats.TotalPixels != 4 {
		t.Fatalf("expected 4 total pixels, got %d", stats.TotalPixels)
	}
}

func TestDiffImagesThreshold(t *testing.T) {
	before := image.NewRGBA(image.Rect(0, 0, 1, 1))
	after := image.NewRGBA(image.Rect(0, 0, 1, 1))
	before.SetRGBA(0, 0, color.RGBA{R: 10, G: 10, B: 10, A: 255})
	after.SetRGBA(0, 0, color.RGBA{R: 12, G: 10, B: 10, A: 255})

	_, stats, err := diffImages(before, after, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.ChangedPixels != 0 {
		t.Fatalf("expected 0 changed pixels, got %d", stats.ChangedPixels)
	}

	_, stats, err = diffImages(before, after, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.ChangedPixels != 1 {
		t.Fatalf("expected 1 changed pixel, got %d", stats.ChangedPixels)
	}
}

func TestDiffImagesSizeMismatch(t *testing.T) {
	before := image.NewRGBA(image.Rect(0, 0, 1, 1))
	after := image.NewRGBA(image.Rect(0, 0, 2, 1))
	if _, _, err := diffImages(before, after, 0); err == nil {
		t.Fatalf("expected error for size mismatch")
	}
}

func fillRGBA(img *image.RGBA, c color.RGBA) {
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			img.SetRGBA(x, y, c)
		}
	}
}
