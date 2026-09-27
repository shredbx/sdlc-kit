package image

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// pngBytes encodes img to PNG bytes for the decode-path tests.
func pngBytes(t *testing.T, img image.Image) []byte {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	return buf.Bytes()
}

// transparentLogoPNG builds a 2×2 NRGBA where one pixel is an opaque white "shape"
// and the other three are fully transparent — a monochrome-on-transparent logo.
func transparentLogoPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	img.SetNRGBA(0, 0, color.NRGBA{R: 255, G: 255, B: 255, A: 255}) // the shape
	// (0,1),(1,0),(1,1) left zero-value → fully transparent
	return pngBytes(t, img)
}

func TestHasAlpha(t *testing.T) {
	// Opaque PNG — every pixel A=255.
	opaquePNG := func() []byte {
		img := image.NewNRGBA(image.Rect(0, 0, 2, 2))
		for y := 0; y < 2; y++ {
			for x := 0; x < 2; x++ {
				img.SetNRGBA(x, y, color.NRGBA{R: 10, G: 20, B: 30, A: 255})
			}
		}
		return pngBytes(t, img)
	}()

	// Opaque JPEG — the format carries no alpha channel at all.
	opaqueJPEG := func() []byte {
		img := image.NewRGBA(image.Rect(0, 0, 4, 4))
		for y := 0; y < 4; y++ {
			for x := 0; x < 4; x++ {
				img.Set(x, y, color.RGBA{R: 200, G: 168, B: 81, A: 255})
			}
		}
		var buf bytes.Buffer
		require.NoError(t, jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90}))
		return buf.Bytes()
	}()

	tests := []struct {
		name    string
		data    []byte
		want    bool
		wantErr bool
	}{
		{"transparent png → tintable", transparentLogoPNG(t), true, false},
		{"fully opaque png → not tintable", opaquePNG, false, false},
		{"opaque jpeg → not tintable", opaqueJPEG, false, false},
		{"garbage bytes → error", []byte("not an image"), false, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := HasAlpha(tc.data)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestParseCSSColor(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    color.NRGBA
		wantErr bool
	}{
		// hex
		{"hex 6-digit brand gold", "#C8A851", color.NRGBA{R: 0xC8, G: 0xA8, B: 0x51, A: 0xff}, false},
		{"hex uppercase == lowercase", "#c8a851", color.NRGBA{R: 0xC8, G: 0xA8, B: 0x51, A: 0xff}, false},
		{"hex shorthand #fff", "#fff", color.NRGBA{R: 255, G: 255, B: 255, A: 255}, false},
		{"hex 8-digit with alpha", "#00000080", color.NRGBA{R: 0, G: 0, B: 0, A: 0x80}, false},
		// rgb / rgba
		{"rgb teal", "rgb(13,79,79)", color.NRGBA{R: 13, G: 79, B: 79, A: 255}, false},
		{"rgb spaced", "rgb(13, 79, 79)", color.NRGBA{R: 13, G: 79, B: 79, A: 255}, false},
		{"rgba half alpha", "rgba(200,168,81,0.5)", color.NRGBA{R: 200, G: 168, B: 81, A: 128}, false},
		{"rgb percent red", "rgb(100%,0%,0%)", color.NRGBA{R: 255, G: 0, B: 0, A: 255}, false},
		{"RGB uppercase name", "RGB(255,255,255)", color.NRGBA{R: 255, G: 255, B: 255, A: 255}, false},
		// hsl / hsla — hsl(180,72%,18%) is exactly the brand teal (13,79,79)
		{"hsl red", "hsl(0,100%,50%)", color.NRGBA{R: 255, G: 0, B: 0, A: 255}, false},
		{"hsl teal matches #0d4f4f", "hsl(180,72%,18%)", color.NRGBA{R: 13, G: 79, B: 79, A: 255}, false},
		{"hsla with alpha", "hsla(0,100%,50%,0)", color.NRGBA{R: 255, G: 0, B: 0, A: 0}, false},
		// rejects
		{"named colour rejected", "red", color.NRGBA{}, true},
		{"garbage rejected", "notacolor", color.NRGBA{}, true},
		{"empty rejected", "", color.NRGBA{}, true},
		{"url() injection rejected", "url(x)", color.NRGBA{}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseCSSColor(tc.in)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestTintLogo(t *testing.T) {
	src := transparentLogoPNG(t)

	out, err := TintLogo(src, "#C8A851") // brand gold
	require.NoError(t, err)

	img, err := png.Decode(bytes.NewReader(out))
	require.NoError(t, err)
	require.Equal(t, image.Rect(0, 0, 2, 2), img.Bounds())

	// The opaque shape pixel is recoloured to the tint at full alpha.
	r, g, b, a := img.At(0, 0).RGBA()
	assert.Equal(t, uint32(0xC8C8), r, "shape R = tint R")
	assert.Equal(t, uint32(0xA8A8), g, "shape G = tint G")
	assert.Equal(t, uint32(0x5151), b, "shape B = tint B")
	assert.Equal(t, uint32(0xffff), a, "shape stays fully opaque")

	// A transparent source pixel stays fully transparent (mask preserved).
	_, _, _, a2 := img.At(1, 1).RGBA()
	assert.Equal(t, uint32(0), a2, "transparent stays transparent")
}

func TestTintLogo_TranslucentTintFoldsIntoAlpha(t *testing.T) {
	src := transparentLogoPNG(t)

	// A 50%-alpha tint halves the mask's opacity (source-alpha × tint-alpha).
	out, err := TintLogo(src, "rgba(200,168,81,0.5)")
	require.NoError(t, err)
	img, err := png.Decode(bytes.NewReader(out))
	require.NoError(t, err)

	nrgba, ok := img.(*image.NRGBA)
	require.True(t, ok)
	// shape pixel: source A=255, tint A=128 → 255*128/255 = 128
	assert.Equal(t, uint8(128), nrgba.NRGBAAt(0, 0).A)
	assert.Equal(t, uint8(200), nrgba.NRGBAAt(0, 0).R)
}

func TestTintLogo_RejectsBadColour(t *testing.T) {
	_, err := TintLogo(transparentLogoPNG(t), "notacolor")
	require.Error(t, err)
}

func TestTintLogo_RejectsUndecodableSource(t *testing.T) {
	_, err := TintLogo([]byte("not an image"), "#C8A851")
	require.Error(t, err)
}
