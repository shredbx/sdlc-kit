package image

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"

	"github.com/gen2brain/webp"
)

// isWebP reports whether data is a WebP bitstream (RIFF….WEBP magic). The lossy
// encoder's byte output is not stable, so tests assert format + dims, not bytes.
func isWebP(data []byte) bool {
	return len(data) > 11 && bytes.Equal(data[0:4], []byte("RIFF")) && bytes.Equal(data[8:12], []byte("WEBP"))
}

// jpgBytes encodes a solid-color w×h JPEG (the canonical "photo" input).
func jpgBytes(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x % 256), G: uint8(y % 256), B: 128, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatalf("encode jpg %dx%d: %v", w, h, err)
	}
	return buf.Bytes()
}

// encodePNG encodes a w×h PNG; when withAlpha is true the pixels carry transparency
// (exercises the lossless path's alpha handling). Named encodePNG (not pngBytes)
// to avoid clashing with the pre-existing pngBytes helper in logotint_test.go.
func encodePNG(t *testing.T, w, h int, withAlpha bool) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			a := uint8(255)
			if withAlpha {
				a = uint8((x + y) % 256)
			}
			img.Set(x, y, color.NRGBA{R: uint8(x % 256), G: uint8(y % 256), B: 64, A: a})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png %dx%d: %v", w, h, err)
	}
	return buf.Bytes()
}

// webpBytes encodes a w×h WebP via gen2brain — the round-trip input that proves a
// WebP upload is decoded + re-encoded (not rejected or stored verbatim).
func webpBytes(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.NRGBA{R: uint8(x % 256), G: uint8(y % 256), B: 90, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := webp.Encode(&buf, img, webp.Options{Quality: 80}); err != nil {
		t.Fatalf("encode webp %dx%d: %v", w, h, err)
	}
	return buf.Bytes()
}

// fakePNGHeader writes a PNG signature + IHDR with the given dimensions but NO
// pixel data. image.DecodeConfig reads only the header, so this returns cfg with
// those dims WITHOUT allocating the bitmap — the decode-bomb fixture.
func fakePNGHeader(t *testing.T, w, h int) []byte {
	t.Helper()
	var buf bytes.Buffer
	buf.Write([]byte{137, 'P', 'N', 'G', 13, 10, 26, 10}) // PNG signature
	var ihdr bytes.Buffer
	_ = binary.Write(&ihdr, binary.BigEndian, uint32(w))
	_ = binary.Write(&ihdr, binary.BigEndian, uint32(h))
	ihdr.Write([]byte{8, 2, 0, 0, 0}) // bitDepth=8, colorType=RGB, compression/filter/interlace=0
	_ = binary.Write(&buf, binary.BigEndian, uint32(13)) // IHDR length
	buf.WriteString("IHDR")
	buf.Write(ihdr.Bytes())
	var crcBuf bytes.Buffer
	crcBuf.WriteString("IHDR")
	crcBuf.Write(ihdr.Bytes())
	_ = binary.Write(&buf, binary.BigEndian, crc32.ChecksumIEEE(crcBuf.Bytes()))
	return buf.Bytes()
}

func TestProcess_DownscalesLargeJPEG(t *testing.T) {
	out, w, h, err := Process(jpgBytes(t, 4000, 3000), ProcessOptions{MaxDimension: 2048, Quality: 82})
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	if !isWebP(out) {
		t.Fatalf("output is not WebP (got %d bytes)", len(out))
	}
	if w > 2048 || h > 2048 {
		t.Fatalf("dims %dx%d exceed 2048 longest-edge cap", w, h)
	}
	// Aspect preserved: 4000×3000 → 2048×1536.
	if w != 2048 || h != 1536 {
		t.Fatalf("dims %dx%d, want 2048×1536 (aspect preserved)", w, h)
	}
}

func TestProcess_PNGReencodedToWebP(t *testing.T) {
	out, w, h, err := Process(encodePNG(t, 100, 80, false), FacetProcessOptions(FacetGallery))
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	if !isWebP(out) {
		t.Fatalf("PNG was not re-encoded to WebP")
	}
	if w != 100 || h != 80 {
		t.Fatalf("dims %dx%d, want 100×80 (under cap, unchanged)", w, h)
	}
}

func TestProcess_WebPInputDecoded(t *testing.T) {
	out, w, h, err := Process(webpBytes(t, 300, 200), FacetProcessOptions(FacetGallery))
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	if !isWebP(out) {
		t.Fatalf("WebP input was not re-encoded")
	}
	if w != 300 || h != 200 {
		t.Fatalf("dims %dx%d, want 300×200", w, h)
	}
}

func TestProcess_TinyImage(t *testing.T) {
	out, w, h, err := Process(jpgBytes(t, 1, 1), FacetProcessOptions(FacetGallery))
	if err != nil {
		t.Fatalf("Process 1×1: %v", err)
	}
	if !isWebP(out) {
		t.Fatalf("1×1 output not WebP")
	}
	if w != 1 || h != 1 {
		t.Fatalf("dims %dx%d, want 1×1", w, h)
	}
}

func TestProcess_AtCapNoUpscale(t *testing.T) {
	// Exactly at the cap — dimensions must be unchanged (never upscaled).
	out, w, h, err := Process(jpgBytes(t, 2048, 2048), FacetProcessOptions(FacetGallery))
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	if !isWebP(out) {
		t.Fatalf("output not WebP")
	}
	if w != 2048 || h != 2048 {
		t.Fatalf("dims %dx%d, want 2048×2048 (at cap, no upscale)", w, h)
	}
}

func TestProcess_BombRejected(t *testing.T) {
	// A PNG whose header claims 50000×50000 but has no pixel data. DecodeConfig
	// returns those dims → bomb-guard fires BEFORE the (impossible) full decode.
	_, _, _, err := Process(fakePNGHeader(t, 50000, 50000), FacetProcessOptions(FacetGallery))
	if err == nil {
		t.Fatal("expected ErrImageTooLarge for a 50000×50000 header, got nil")
	}
	if err != ErrImageTooLarge {
		t.Fatalf("want ErrImageTooLarge, got %v", err)
	}
}

func TestProcess_NonImageRejected(t *testing.T) {
	_, _, _, err := Process([]byte("not an image at all"), FacetProcessOptions(FacetGallery))
	if err == nil {
		t.Fatal("expected error for non-image input, got nil")
	}
}

func TestProcess_EmptyRejected(t *testing.T) {
	_, _, _, err := Process(nil, FacetProcessOptions(FacetGallery))
	if err == nil {
		t.Fatal("expected error for empty input, got nil")
	}
}

func TestProcess_LosslessAlpha(t *testing.T) {
	// A PNG with alpha through the lossless path must still produce valid WebP
	// with dimensions preserved.
	out, w, h, err := Process(encodePNG(t, 64, 48, true), ProcessOptions{MaxDimension: 512, Lossless: true})
	if err != nil {
		t.Fatalf("Process lossless: %v", err)
	}
	if !isWebP(out) {
		t.Fatalf("lossless output not WebP")
	}
	if w != 64 || h != 48 {
		t.Fatalf("dims %dx%d, want 64×48", w, h)
	}
}

func TestScaleLongestEdge(t *testing.T) {
	cases := []struct{ sw, sh, max, dw, dh int }{
		{4000, 3000, 2048, 2048, 1536}, // landscape, downscale
		{3000, 4000, 2048, 1536, 2048}, // portrait, downscale
		{100, 80, 2048, 100, 80},       // under cap, unchanged
		{2048, 2048, 2048, 2048, 2048}, // at cap, unchanged
		{1024, 1024, 0, 1024, 1024},    // max=0 (no cap)
	}
	for _, c := range cases {
		dw, dh := scaleLongestEdge(c.sw, c.sh, c.max)
		if dw != c.dw || dh != c.dh {
			t.Errorf("scaleLongestEdge(%d,%d,max=%d) = %d×%d, want %d×%d", c.sw, c.sh, c.max, dw, dh, c.dw, c.dh)
		}
	}
}

func TestFacetProcessOptions(t *testing.T) {
	if got := FacetProcessOptions(FacetGallery); got.MaxDimension != 2048 || got.Lossless {
		t.Errorf("gallery = %+v, want MaxDimension 2048 lossy", got)
	}
	if got := FacetProcessOptions(FacetAvatar); got.MaxDimension != 512 {
		t.Errorf("avatar MaxDimension = %d, want 512", got.MaxDimension)
	}
	if got := FacetProcessOptions(FacetQR); !got.Lossless {
		t.Error("qr should be lossless")
	}
}
