package devbrowser

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	_ "image/jpeg"
)

type DiffStats struct {
	ChangedPixels int
	TotalPixels   int
	Width         int
	Height        int
}

func resolveInputPath(artifactDir, pathArg string) (string, error) {
	raw := strings.TrimSpace(pathArg)
	if raw == "" {
		return "", nil
	}
	expanded := os.ExpandEnv(raw)
	if strings.HasPrefix(expanded, "~") {
		if home, err := os.UserHomeDir(); err == nil {
			expanded = filepath.Join(home, strings.TrimPrefix(expanded, "~"))
		}
	}
	if !filepath.IsAbs(expanded) {
		expanded = filepath.Join(artifactDir, expanded)
	}
	resolved, err := filepath.Abs(expanded)
	if err != nil {
		return "", err
	}
	return resolved, nil
}

func loadImage(path string) (image.Image, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("image path is required")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	img, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}
	return img, nil
}

func writePNG(path string, img image.Image) (err error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := file.Close(); err == nil && cerr != nil {
			err = cerr
		}
	}()
	err = png.Encode(file, img)
	return err
}

func diffImages(before, after image.Image, threshold uint8) (*image.RGBA, DiffStats, error) {
	if before == nil || after == nil {
		return nil, DiffStats{}, errors.New("images are required")
	}
	beforeBounds := before.Bounds()
	afterBounds := after.Bounds()
	if beforeBounds.Dx() != afterBounds.Dx() || beforeBounds.Dy() != afterBounds.Dy() {
		return nil, DiffStats{}, fmt.Errorf(
			"image sizes differ (before %dx%d, after %dx%d)",
			beforeBounds.Dx(),
			beforeBounds.Dy(),
			afterBounds.Dx(),
			afterBounds.Dy(),
		)
	}

	width := beforeBounds.Dx()
	height := beforeBounds.Dy()
	diff := image.NewRGBA(image.Rect(0, 0, width, height))
	changed := 0

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			bc := rgba8(before.At(beforeBounds.Min.X+x, beforeBounds.Min.Y+y))
			ac := rgba8(after.At(afterBounds.Min.X+x, afterBounds.Min.Y+y))
			if pixelDelta(bc, ac) > threshold {
				changed++
				diff.SetRGBA(x, y, highlightPixel(ac))
			} else {
				diff.SetRGBA(x, y, ac)
			}
		}
	}

	return diff, DiffStats{
		ChangedPixels: changed,
		TotalPixels:   width * height,
		Width:         width,
		Height:        height,
	}, nil
}

func diffImagesWithIgnore(before, after image.Image, threshold uint8, ignoreRegions []image.Rectangle) (*image.RGBA, DiffStats, error) {
	if before == nil || after == nil {
		return nil, DiffStats{}, errors.New("images are required")
	}
	beforeBounds := before.Bounds()
	afterBounds := after.Bounds()
	if beforeBounds.Dx() != afterBounds.Dx() || beforeBounds.Dy() != afterBounds.Dy() {
		return nil, DiffStats{}, fmt.Errorf(
			"image sizes differ (before %dx%d, after %dx%d)",
			beforeBounds.Dx(),
			beforeBounds.Dy(),
			afterBounds.Dx(),
			afterBounds.Dy(),
		)
	}

	width := beforeBounds.Dx()
	height := beforeBounds.Dy()
	diff := image.NewRGBA(image.Rect(0, 0, width, height))
	changed := 0

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if pointIgnored(x, y, ignoreRegions) {
				ac := rgba8(after.At(afterBounds.Min.X+x, afterBounds.Min.Y+y))
				diff.SetRGBA(x, y, ac)
				continue
			}
			bc := rgba8(before.At(beforeBounds.Min.X+x, beforeBounds.Min.Y+y))
			ac := rgba8(after.At(afterBounds.Min.X+x, afterBounds.Min.Y+y))
			if pixelDelta(bc, ac) > threshold {
				changed++
				diff.SetRGBA(x, y, highlightPixel(ac))
			} else {
				diff.SetRGBA(x, y, ac)
			}
		}
	}

	return diff, DiffStats{
		ChangedPixels: changed,
		TotalPixels:   width * height,
		Width:         width,
		Height:        height,
	}, nil
}

func pointIgnored(x, y int, regions []image.Rectangle) bool {
	if len(regions) == 0 {
		return false
	}
	pt := image.Pt(x, y)
	for _, region := range regions {
		if pt.In(region) {
			return true
		}
	}
	return false
}

func rgba8(c color.Color) color.RGBA {
	r, g, b, a := c.RGBA()
	return color.RGBA{
		R: uint8(r >> 8),
		G: uint8(g >> 8),
		B: uint8(b >> 8),
		A: uint8(a >> 8),
	}
}

func pixelDelta(a, b color.RGBA) uint8 {
	max := 0
	for _, d := range []int{
		absInt(int(a.R) - int(b.R)),
		absInt(int(a.G) - int(b.G)),
		absInt(int(a.B) - int(b.B)),
		absInt(int(a.A) - int(b.A)),
	} {
		if d > max {
			max = d
		}
	}
	return uint8(max)
}

func highlightPixel(base color.RGBA) color.RGBA {
	return color.RGBA{
		R: blendChannel(base.R, 255),
		G: blendChannel(base.G, 0),
		B: blendChannel(base.B, 0),
		A: base.A,
	}
}

func blendChannel(base, overlay uint8) uint8 {
	return uint8((int(base) + int(overlay)) / 2)
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
