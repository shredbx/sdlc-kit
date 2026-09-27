package image

// favicon.go — bake a square, guaranteed-PNG favicon from a raster brand logo (task
// 2607-057, BR favicon Phase 2). A browser tab slot and an iOS home-screen tile are both
// SQUARE, so a non-square logo (a wide wordmark) must be letterboxed onto a transparent
// square canvas, never stretched. ResizeSquarePNG is the raster twin of the tint bake
// (logotint.go): decode → fit/center → image/png-encode, reusing the SAME high-quality
// downscale (xdraw.CatmullRom) + never-upscale scaler (scaleLongestEdge) as Process, but
// emitting a PNG (a fetched favicon never runs CSS; PNG is the broadly-supported raster
// icon format). It lives here in pkg/image so any consumer — BR today, a Fabric
// configuration later — bakes favicons through ONE implementation. No new dependency, no
// cdn-cgi (avoids the watermark lane's 9524 risk).

import (
	"bytes"
	"fmt"
	"image"
	"image/png" // PNG encoder for the baked favicon (the raster decoders are registered by the sibling files)

	xdraw "golang.org/x/image/draw"
)

// ResizeSquarePNG decodes a raster logo and renders it into an EXACTLY size×size
// straight-alpha PNG: the logo is scaled to FIT within the square (longest edge → size,
// aspect preserved, NEVER upscaled) and CENTERED on a transparent canvas. The output is
// always exactly size×size — a browser tab / iOS tile is square, so a non-square logo is
// letterboxed with transparent padding rather than distorted.
//
// It reuses scaleLongestEdge (the one never-upscale scaler) + xdraw.CatmullRom (the same
// high-quality downscale as Process) and PNG-encodes like TintLogo. The package-level
// decoders (png/jpeg registered by logotint.go, webp by process.go's gen2brain import)
// cover every raster the logo-upload path accepts. An SVG is undecodable here (an explicit
// non-goal — the caller guards it upstream) → ErrUnsupportedImage; a decode-bomb source →
// ErrImageTooLarge (the SAME megapixel guard as Process). Pure: bytes in → bytes out.
func ResizeSquarePNG(src []byte, size int) ([]byte, error) {
	if size <= 0 {
		return nil, fmt.Errorf("favicon: size must be positive, got %d", size)
	}
	if len(src) == 0 {
		return nil, ErrUnsupportedImage
	}

	// 1. DecodeConfig (header only) — bomb-guard BEFORE the full decode (mirrors Process).
	cfg, _, err := image.DecodeConfig(bytes.NewReader(src))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnsupportedImage, err)
	}
	if int64(cfg.Width)*int64(cfg.Height) > maxImagePixels {
		return nil, ErrImageTooLarge
	}

	// 2. Full decode (jpeg/png/webp — decoders registered by the sibling files).
	simg, _, err := image.Decode(bytes.NewReader(src))
	if err != nil {
		return nil, fmt.Errorf("%w: decode: %v", ErrUnsupportedImage, err)
	}
	b := simg.Bounds()
	sw, sh := b.Dx(), b.Dy()

	// 3. Fit the longest edge to size (aspect preserved, never upscaled) — the one scaler.
	fw, fh := scaleLongestEdge(sw, sh, size)

	// 4. Center the fitted logo on a size×size transparent canvas.
	canvas := image.NewNRGBA(image.Rect(0, 0, size, size))
	offX := (size - fw) / 2
	offY := (size - fh) / 2
	dst := image.Rect(offX, offY, offX+fw, offY+fh)
	if fw == sw && fh == sh {
		// No scaling needed (source already ≤ size) — draw it verbatim over the transparent
		// canvas; CatmullRom on an identity rect would needlessly resample the edges.
		xdraw.Draw(canvas, dst, simg, b.Min, xdraw.Over)
	} else {
		xdraw.CatmullRom.Scale(canvas, dst, simg, b, xdraw.Over, nil)
	}

	// 5. Encode PNG (straight-alpha, so anti-aliased/letterbox edges stay clean).
	var out bytes.Buffer
	if err := png.Encode(&out, canvas); err != nil {
		return nil, fmt.Errorf("favicon: encode png: %w", err)
	}
	return out.Bytes(), nil
}
