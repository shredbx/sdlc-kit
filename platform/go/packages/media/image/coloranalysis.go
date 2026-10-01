package image

import (
	"fmt"
	"image"
	"io"
	"sort"

	_ "image/jpeg"
	_ "image/png"
)

// AnalyzePrimaryColor decodes the image from r and extracts the dominant hex
// color using grid-sampling + frequency-bucket quantization. Returns empty
// string on any error — callers treat failure as non-fatal.
func AnalyzePrimaryColor(r io.Reader, _ ImageFormat) (string, error) {
	img, _, err := image.Decode(r)
	if err != nil {
		return "", fmt.Errorf("decode: %w", err)
	}

	bounds := img.Bounds()
	w := bounds.Max.X - bounds.Min.X
	h := bounds.Max.Y - bounds.Min.Y
	if w == 0 || h == 0 {
		return "", fmt.Errorf("zero-size image")
	}

	stepX := max(1, w/20)
	stepY := max(1, h/20)

	type rgb [3]uint8
	buckets := make(map[rgb]int, 64)

	for y := bounds.Min.Y; y < bounds.Max.Y; y += stepY {
		for x := bounds.Min.X; x < bounds.Max.X; x += stepX {
			r32, g32, b32, _ := img.At(x, y).RGBA()
			r8, g8, b8 := uint8(r32>>8), uint8(g32>>8), uint8(b32>>8)
			buckets[rgb{r8 & 0xe0, g8 & 0xe0, b8 & 0xe0}]++
		}
	}

	if len(buckets) == 0 {
		return "", fmt.Errorf("no pixels sampled")
	}

	type entry struct {
		color rgb
		count int
	}
	ranked := make([]entry, 0, len(buckets))
	for c, n := range buckets {
		ranked = append(ranked, entry{c, n})
	}
	sort.Slice(ranked, func(i, j int) bool { return ranked[i].count > ranked[j].count })

	top := ranked[0].color
	return fmt.Sprintf("#%02x%02x%02x", top[0]+16, top[1]+16, top[2]+16), nil
}
