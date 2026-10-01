package image

// Tests for ResizeSquarePNG — the square, guaranteed-PNG favicon bake (task 2607-057,
// BR favicon Phase 2, story P2-S1). A browser tab slot and an iOS home-screen tile are
// both square, so the helper must fit + center the logo on a size×size transparent canvas
// (aspect preserved, never upscaled) and emit a real PNG. Reuses the process_test.go image
// helpers (encodePNG, jpgBytes, fakePNGHeader) — same package. Traces SC-P2-S1a (bake) and
// SC-P2-S1b (an undecodable source is rejected — the endpoint guards SVG upstream).

import (
	"bytes"
	"errors"
	"image"
	"image/png"
	"testing"
)

// isPNG reports whether data begins with the 8-byte PNG signature. The favicon bake must
// emit a PNG regardless of the source format (a fetched icon never runs CSS; PNG is the
// broadly-supported raster icon format).
func isPNG(data []byte) bool {
	return len(data) >= 8 && bytes.Equal(data[:8], []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a})
}

// decodeImg decodes bytes to an image.Image or fails the test.
func decodeImg(t *testing.T, data []byte) image.Image {
	t.Helper()
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("output does not decode: %v", err)
	}
	return img
}

// TC-P2-01 / SC-P2-S1a — a 512×512 source → exactly 64×64 and 180×180 PNGs.
func TestResizeSquarePNG_FitsSquareExactDims(t *testing.T) {
	src := encodePNG(t, 512, 512, true)
	for _, size := range []int{64, 180} {
		out, err := ResizeSquarePNG(src, size)
		if err != nil {
			t.Fatalf("ResizeSquarePNG(512, %d): %v", size, err)
		}
		if !isPNG(out) {
			t.Fatalf("size %d: output is not a PNG", size)
		}
		cfg, _, err := image.DecodeConfig(bytes.NewReader(out))
		if err != nil {
			t.Fatalf("size %d: output header does not decode: %v", size, err)
		}
		if cfg.Width != size || cfg.Height != size {
			t.Fatalf("size %d: output dims %dx%d, want %dx%d", size, cfg.Width, cfg.Height, size, size)
		}
	}
}

// A WIDE (2:1) source is letterboxed onto the square canvas (aspect preserved, never
// stretched): output is still exactly size×size, and the top/bottom padding rows outside
// the fitted 64×32 sub-rect are fully transparent.
func TestResizeSquarePNG_NonSquareLetterboxed(t *testing.T) {
	src := encodePNG(t, 400, 200, true) // 2:1 landscape → fits 64×32 centered (offY=16)
	out, err := ResizeSquarePNG(src, 64)
	if err != nil {
		t.Fatalf("ResizeSquarePNG: %v", err)
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(out))
	if err != nil || cfg.Width != 64 || cfg.Height != 64 {
		t.Fatalf("output dims %dx%d (err %v), want 64×64", cfg.Width, cfg.Height, err)
	}
	img := decodeImg(t, out)
	// Row 2 is in the top padding band (rows 0..15) — must be fully transparent.
	if _, _, _, a := img.At(32, 2).RGBA(); a != 0 {
		t.Errorf("top-padding pixel (32,2) alpha = %d, want 0 (transparent letterbox)", a>>8)
	}
}

// A fully-transparent source stays fully transparent through the downscale — the alpha
// channel is not flattened to opaque (favicon edges must stay clean).
func TestResizeSquarePNG_PreservesAlpha(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 128, 128)) // every pixel {0,0,0,0} — transparent
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode transparent source: %v", err)
	}
	out, err := ResizeSquarePNG(buf.Bytes(), 64) // 128 → 64 exercises the scale path
	if err != nil {
		t.Fatalf("ResizeSquarePNG: %v", err)
	}
	decoded := decodeImg(t, out)
	for _, p := range []image.Point{{X: 0, Y: 0}, {X: 32, Y: 32}, {X: 63, Y: 63}} {
		if _, _, _, a := decoded.At(p.X, p.Y).RGBA(); a != 0 {
			t.Errorf("pixel %v alpha = %d, want 0 (transparent preserved)", p, a>>8)
		}
	}
}

// A source SMALLER than the target is NOT upscaled: it keeps its native size, centered on
// the size×size canvas. Proven by an opaque 40×40 source → 180 canvas: the center is opaque
// (the logo), a far-corner padding pixel is transparent (the logo did NOT fill the canvas).
func TestResizeSquarePNG_NeverUpscales(t *testing.T) {
	src := encodePNG(t, 40, 40, false) // opaque, smaller than 180
	out, err := ResizeSquarePNG(src, 180)
	if err != nil {
		t.Fatalf("ResizeSquarePNG: %v", err)
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(out))
	if err != nil || cfg.Width != 180 || cfg.Height != 180 {
		t.Fatalf("output dims %dx%d (err %v), want 180×180 canvas", cfg.Width, cfg.Height, err)
	}
	decoded := decodeImg(t, out)
	if _, _, _, a := decoded.At(5, 5).RGBA(); a != 0 {
		t.Errorf("corner (5,5) alpha = %d, want 0 — an upscaled logo would have filled it", a>>8)
	}
	if _, _, _, a := decoded.At(90, 90).RGBA(); a>>8 != 255 {
		t.Errorf("center (90,90) alpha = %d, want 255 (the opaque logo sits centered)", a>>8)
	}
}

// TC-P2-03 / SC-P2-S1b — an undecodable source (SVG / garbage / empty) → ErrUnsupportedImage.
// The endpoint guards SVG upstream; this is the defense-in-depth twin.
func TestResizeSquarePNG_RejectsUndecodable(t *testing.T) {
	cases := map[string][]byte{
		"svg":     []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16"><rect width="16" height="16"/></svg>`),
		"garbage": []byte("not an image at all"),
		"empty":   nil,
	}
	for name, src := range cases {
		if _, err := ResizeSquarePNG(src, 64); !errors.Is(err, ErrUnsupportedImage) {
			t.Errorf("%s: want ErrUnsupportedImage, got %v", name, err)
		}
	}
}

// A decode-bomb source (huge header, no pixels) → ErrImageTooLarge (the same megapixel
// guard as Process), rejected BEFORE the impossible full decode.
func TestResizeSquarePNG_BombRejected(t *testing.T) {
	if _, err := ResizeSquarePNG(fakePNGHeader(t, 50000, 50000), 64); !errors.Is(err, ErrImageTooLarge) {
		t.Fatalf("want ErrImageTooLarge for a 50000×50000 header, got %v", err)
	}
}

// A non-positive size is a programming error → a plain error, never a zero-size PNG.
func TestResizeSquarePNG_InvalidSize(t *testing.T) {
	if _, err := ResizeSquarePNG(encodePNG(t, 64, 64, false), 0); err == nil {
		t.Fatal("size 0: expected an error, got nil")
	}
}
