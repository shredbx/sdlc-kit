package image

// logotint.go — the raster twin of the in-DOM CSS logo-tint technique. A
// monochrome-on-transparent brand mark is tinted in the browser with
// `background-color: <tint>` painted through `mask: <logo>` (the alpha channel is
// the mask). CSS never runs on a fetched file, so the favicon / OG image needs a
// real tinted PNG baked server-side — that is TintLogo. HasAlpha is the upload-time
// gate: tinting only makes sense for a logo that carries transparency.
//
// The pixel op mirrors watermark.recolorMark (the recolorable-silhouette contract,
// #0300): flood every pixel's RGB with the tint, keep the source's own alpha as the
// mask. It lives here in pkg/image (not the watermark subpackage) so any consumer —
// BR today, the Fabric configuration later — reuses one implementation.

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png" // PNG encoder for the tinted raster + registers the PNG decoder
	"math"
	"strconv"
	"strings"

	_ "image/jpeg" // register the JPEG decoder for image.Decode

	_ "golang.org/x/image/webp" // register the WebP decoder for image.Decode
)

// HasAlpha reports whether the raster image in data carries real transparency —
// any pixel whose alpha is not fully opaque. It is the upload-time gate for the
// logo tint control: a transparent PNG/WebP → true (tintable as a monochrome mask);
// a JPEG or a fully-opaque PNG → false (tinting it would just repaint a filled
// rectangle, so the control is disabled with a tooltip and the regenerate endpoint
// refuses — no wasted processing). A decode failure returns the error so the caller
// can decide (an undecodable/unsupported format is conservatively not-tintable).
func HasAlpha(data []byte) (bool, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return false, fmt.Errorf("decode: %w", err)
	}
	// Every stdlib image type advertises opacity via Opaque() (a full, efficient
	// Pix scan for the alpha-bearing types; a constant true for YCbCr/JPEG). Use it
	// when present; fall back to a manual per-pixel scan for any exotic type.
	if oi, ok := img.(interface{ Opaque() bool }); ok {
		return !oi.Opaque(), nil
	}
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if _, _, _, a := img.At(x, y).RGBA(); a < 0xffff {
				return true, nil
			}
		}
	}
	return false, nil
}

// TintLogo recolours a monochrome-on-transparent logo to a solid CSS colour,
// preserving the source's alpha channel as a mask — the raster twin of the in-DOM
// `background-color: <tint>` painted through `mask: <logo>`. Every pixel's RGB is
// replaced by the tint; its alpha becomes source-alpha × tint-alpha, so anti-aliased
// edges (and a translucent tint, should one be chosen) survive. The result is a
// straight-alpha PNG for the favicon / OG raster.
//
// cssColor is any colour the site-config write-gate accepts — hex (#rgb / #rrggbb /
// #rrggbbaa), rgb()/rgba(), or hsl()/hsla() — parsed by parseCSSColor to match the
// in-DOM CSS colour byte-for-byte. An unparseable colour or an undecodable source is
// an error; nothing is guessed.
func TintLogo(src []byte, cssColor string) ([]byte, error) {
	tint, err := parseCSSColor(cssColor)
	if err != nil {
		return nil, err
	}
	simg, _, err := image.Decode(bytes.NewReader(src))
	if err != nil {
		return nil, fmt.Errorf("decode logo: %w", err)
	}
	b := simg.Bounds()
	dst := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			// .RGBA() is premultiplied 16-bit; the alpha channel itself is not
			// premultiplied by anything, so a>>8 is the true source alpha.
			_, _, _, a := simg.At(x, y).RGBA()
			a8 := uint32(a >> 8)
			o := dst.PixOffset(x-b.Min.X, y-b.Min.Y)
			dst.Pix[o] = tint.R
			dst.Pix[o+1] = tint.G
			dst.Pix[o+2] = tint.B
			dst.Pix[o+3] = uint8(a8 * uint32(tint.A) / 255)
		}
	}
	var out bytes.Buffer
	if err := png.Encode(&out, dst); err != nil {
		return nil, fmt.Errorf("encode png: %w", err)
	}
	return out.Bytes(), nil
}

// parseCSSColor parses the three machine colour forms the site-config gate permits
// (see cmspage.cssColorPattern / the client color-format.ts) into a straight-alpha
// NRGBA:
//
//	hex : #rgb | #rrggbb | #rrggbbaa
//	rgb : rgb(r,g,b) | rgba(r,g,b,a)   — channels 0-255 or 0-100%, alpha 0-1
//	hsl : hsl(h,s%,l%) | hsla(h,s%,l%,a)
//
// It mirrors that grammar so the baked raster matches the in-DOM CSS colour exactly.
// The function name is matched case-insensitively (CSS colour functions are), as are
// hex digits. Anything outside the grammar is an error — never guessed, never a
// silent fallback (an opaque black surprise would be worse than a clear failure).
func parseCSSColor(s string) (color.NRGBA, error) {
	v := strings.TrimSpace(s)
	if v == "" {
		return color.NRGBA{}, fmt.Errorf("empty colour")
	}
	lower := strings.ToLower(v)
	switch {
	case strings.HasPrefix(lower, "#"):
		return parseHexColor(v)
	case strings.HasPrefix(lower, "rgb"):
		return parseFuncColor(lower, "rgb")
	case strings.HasPrefix(lower, "hsl"):
		return parseFuncColor(lower, "hsl")
	default:
		return color.NRGBA{}, fmt.Errorf("unrecognised colour %q", s)
	}
}

// parseHexColor parses #rgb / #rrggbb / #rrggbbaa (shorthand expanded to full form).
func parseHexColor(s string) (color.NRGBA, error) {
	h := strings.TrimPrefix(s, "#")
	if len(h) == 3 { // #rgb → #rrggbb
		h = string([]byte{h[0], h[0], h[1], h[1], h[2], h[2]})
	}
	if len(h) != 6 && len(h) != 8 {
		return color.NRGBA{}, fmt.Errorf("invalid hex colour %q", s)
	}
	n, err := strconv.ParseUint(h, 16, 64)
	if err != nil {
		return color.NRGBA{}, fmt.Errorf("invalid hex colour %q: %w", s, err)
	}
	if len(h) == 6 {
		return color.NRGBA{R: uint8(n >> 16), G: uint8(n >> 8), B: uint8(n), A: 0xff}, nil
	}
	return color.NRGBA{R: uint8(n >> 24), G: uint8(n >> 16), B: uint8(n >> 8), A: uint8(n)}, nil
}

// parseFuncColor parses the functional forms rgb()/rgba() and hsl()/hsla() (name is
// the lowercase base "rgb" or "hsl"; the input is already lowercased + trimmed). A
// 4th component is the alpha (0-1). Whitespace inside the parens is tolerated.
func parseFuncColor(s, name string) (color.NRGBA, error) {
	inner, ok := cutFunc(s, name)
	if !ok {
		return color.NRGBA{}, fmt.Errorf("invalid %s colour %q", name, s)
	}
	parts := strings.Split(inner, ",")
	if len(parts) != 3 && len(parts) != 4 {
		return color.NRGBA{}, fmt.Errorf("%s() needs 3 or 4 components: %q", name, s)
	}

	c := color.NRGBA{A: 0xff}
	var err error
	if name == "rgb" {
		if c.R, err = rgbChannel(parts[0]); err != nil {
			return color.NRGBA{}, fmt.Errorf("invalid rgb colour %q: %w", s, err)
		}
		if c.G, err = rgbChannel(parts[1]); err != nil {
			return color.NRGBA{}, fmt.Errorf("invalid rgb colour %q: %w", s, err)
		}
		if c.B, err = rgbChannel(parts[2]); err != nil {
			return color.NRGBA{}, fmt.Errorf("invalid rgb colour %q: %w", s, err)
		}
	} else { // hsl — hue degrees, saturation %, lightness %
		hue, herr := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
		sat, serr := parsePercent(parts[1])
		light, lerr := parsePercent(parts[2])
		if herr != nil || serr != nil || lerr != nil {
			return color.NRGBA{}, fmt.Errorf("invalid hsl colour %q", s)
		}
		c.R, c.G, c.B = hslToRGB(hue, clampUnit(sat), clampUnit(light))
	}

	if len(parts) == 4 {
		if c.A, err = parseAlpha(parts[3]); err != nil {
			return color.NRGBA{}, fmt.Errorf("invalid %s alpha %q: %w", name, s, err)
		}
	}
	return c, nil
}

// cutFunc strips the "<name>(" / "<name>a(" wrapper and the trailing ")", returning
// the inner argument list. name is "rgb" or "hsl" (the optional trailing "a" is the
// rgba/hsla alpha form).
func cutFunc(s, name string) (string, bool) {
	rest := strings.TrimPrefix(s, name)
	if rest == s {
		return "", false
	}
	rest = strings.TrimPrefix(rest, "a") // optional alpha-form letter
	if !strings.HasPrefix(rest, "(") || !strings.HasSuffix(rest, ")") {
		return "", false
	}
	return rest[1 : len(rest)-1], true
}

// rgbChannel parses one rgb() channel: 0-255 integer or a 0-100% percentage.
func rgbChannel(p string) (uint8, error) {
	p = strings.TrimSpace(p)
	if strings.HasSuffix(p, "%") {
		f, err := strconv.ParseFloat(strings.TrimSuffix(p, "%"), 64)
		if err != nil {
			return 0, err
		}
		return clampByte(f / 100 * 255), nil
	}
	f, err := strconv.ParseFloat(p, 64)
	if err != nil {
		return 0, err
	}
	return clampByte(f), nil
}

// parsePercent parses a "NN%" (or bare "NN") value into a 0..1 fraction.
func parsePercent(p string) (float64, error) {
	f, err := strconv.ParseFloat(strings.TrimSuffix(strings.TrimSpace(p), "%"), 64)
	if err != nil {
		return 0, err
	}
	return f / 100, nil
}

// parseAlpha parses a 0-1 alpha component into a 0-255 byte.
func parseAlpha(p string) (uint8, error) {
	f, err := strconv.ParseFloat(strings.TrimSpace(p), 64)
	if err != nil {
		return 0, err
	}
	return clampByte(clampUnit(f) * 255), nil
}

// hslToRGB converts HSL (h in degrees, s/l in 0..1) to an 8-bit RGB triple using
// the standard chroma/hue-sector formula.
func hslToRGB(h, s, l float64) (uint8, uint8, uint8) {
	h = math.Mod(h, 360)
	if h < 0 {
		h += 360
	}
	c := (1 - math.Abs(2*l-1)) * s
	hp := h / 60
	x := c * (1 - math.Abs(math.Mod(hp, 2)-1))
	var r1, g1, b1 float64
	switch {
	case hp < 1:
		r1, g1, b1 = c, x, 0
	case hp < 2:
		r1, g1, b1 = x, c, 0
	case hp < 3:
		r1, g1, b1 = 0, c, x
	case hp < 4:
		r1, g1, b1 = 0, x, c
	case hp < 5:
		r1, g1, b1 = x, 0, c
	default:
		r1, g1, b1 = c, 0, x
	}
	m := l - c/2
	return clampByte((r1 + m) * 255), clampByte((g1 + m) * 255), clampByte((b1 + m) * 255)
}

// clampUnit clamps v to [0,1].
func clampUnit(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// clampByte rounds and clamps a float to a uint8 (0..255).
func clampByte(v float64) uint8 {
	if v <= 0 {
		return 0
	}
	if v >= 255 {
		return 255
	}
	return uint8(v + 0.5)
}
