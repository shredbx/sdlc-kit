// process.go — server-side image normalization: any raster input → WebP ≤ cap.
//
// WHY (task 2607-008): client-side resize (@sbx/ui-image) is unreliable on
// iPad/Safari/iPhone, so the SERVER must be the authoritative normalizer. This
// makes "we never store JPEG, always WebP" an enforced invariant at the store
// chokepoint, not a client convention that can be bypassed by a misbehaving
// device. Pure function (bytes in → bytes out) so it is trivially unit-testable.
//
// PIPELINE (stdlib + golang.org/x/image/draw + github.com/gen2brain/webp):
//  1. DecodeConfig + megapixel bomb-guard (reject BEFORE the full-bitmap alloc)
//  2. image.Decode (jpeg/png/webp — gen2brain/webp's init() registers the webp
//     decoder globally, so importing it here covers decode AND encode)
//  3. downscale longest edge → MaxDimension (aspect preserved, never upscaled)
//     via draw.CatmullRom (higher quality than the box-average in pkg/feed)
//  4. gen2brain/webp.Encode — lossy Quality (photos) or VP8L Lossless (alpha)
//
// REUSE: the decode-bomb-guard + scaledDims shape is promoted here from
// pkg/feed/imagecacher.go (whose helpers are unexported). One shared normalizer
// now, not two. Output bytes are NOT byte-stable across encoders (lossy) —
// tests assert the RIFF….WEBP magic + final dimensions, never exact bytes.

package image

import (
	"bytes"
	"errors"
	"fmt"
	"image"

	"github.com/gen2brain/webp"
	xdraw "golang.org/x/image/draw"
)

// maxImagePixels caps the decoded pixel area (decode-bomb guard). A small,
// highly-compressed input can decode to gigapixels and OOM the server; reject
// it before the full decode allocates the source bitmap. ~40MP ≈ 6400×6400 —
// generous for any real photo, well below the 16383×16383 WebP encode ceiling.
const maxImagePixels int64 = 40_000_000

// ErrImageTooLarge is returned when the decoded dimensions exceed the megapixel
// cap (defense against decode bombs — see maxImagePixels).
var ErrImageTooLarge = errors.New("image dimensions exceed the megapixel cap")

// ErrUnsupportedImage is returned when the input cannot be decoded as a raster
// image (jpeg/png/webp). SVG and AVIF are not handled by this path.
var ErrUnsupportedImage = errors.New("unsupported image format")

// ProcessOptions controls server-side normalization. Build via
// FacetProcessOptions so the cap + quality are facet-appropriate; a zero-value
// MaxDimension means "encode at native dimensions" (still re-encoded to WebP).
type ProcessOptions struct {
	// MaxDimension caps the longest edge in pixels; an image already within the
	// cap keeps its native dimensions (never upscaled). 0 → no cap.
	MaxDimension int
	// Quality is lossy WebP quality [0..100] (100 implies lossless). Ignored
	// when Lossless is true. Defaulted to 82 when 0.
	Quality int
	// Lossless selects VP8L lossless encoding — for alpha-heavy assets (logos,
	// QR). Photos use lossy (dramatically smaller).
	Lossless bool
}

// defaultQuality mirrors the gallery/photo standard (and pkg/feed's JPEG q82).
const defaultQuality = 82

// Process decodes, optionally downscales, and re-encodes data as WebP. Returns
// the WebP bytes and the final (width, height). Pure: no I/O, deterministic in
// format + dimensions (the lossy byte stream itself is not byte-stable).
func Process(data []byte, opts ProcessOptions) ([]byte, int, int, error) {
	if len(data) == 0 {
		return nil, 0, 0, ErrUnsupportedImage
	}

	// 1. DecodeConfig (header only) — bomb-guard BEFORE the full decode.
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, 0, 0, fmt.Errorf("%w: %v", ErrUnsupportedImage, err)
	}
	if int64(cfg.Width)*int64(cfg.Height) > maxImagePixels {
		return nil, 0, 0, ErrImageTooLarge
	}

	// 2. Full decode (jpeg/png/webp — webp decoder registered by gen2brain init).
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, 0, 0, fmt.Errorf("%w: decode: %v", ErrUnsupportedImage, err)
	}
	b := src.Bounds()
	sw, sh := b.Dx(), b.Dy()

	// 3. Downscale if over cap (aspect preserved, never upscaled).
	dw, dh := scaleLongestEdge(sw, sh, opts.MaxDimension)
	var dst image.Image = src
	if dw != sw || dh != sh {
		rect := image.Rect(0, 0, dw, dh)
		scaled := image.NewNRGBA(rect)
		xdraw.CatmullRom.Scale(scaled, rect, src, b, xdraw.Src, nil)
		dst = scaled
	}

	// 4. Encode WebP.
	quality := opts.Quality
	if quality == 0 {
		quality = defaultQuality
	}
	var buf bytes.Buffer
	if err := webp.Encode(&buf, dst, webp.Options{
		Quality:  quality,
		Lossless: opts.Lossless,
		Method:   4, // quality/speed trade-off (0=fast … 6=slower-better); 4 = default
	}); err != nil {
		return nil, 0, 0, fmt.Errorf("webp encode: %w", err)
	}
	return buf.Bytes(), dw, dh, nil
}

// scaleLongestEdge returns dimensions with the longest edge clamped to max,
// aspect ratio preserved, never upscaled. max<=0 or already-within → unchanged.
// Mirrors pkg/feed/imagecacher.scaledDims (promoted here as the one scaler).
func scaleLongestEdge(sw, sh, max int) (int, int) {
	if max <= 0 || (sw <= max && sh <= max) {
		return sw, sh
	}
	longest := sw
	if sh > sw {
		longest = sh
	}
	if longest == 0 {
		return sw, sh
	}
	// Integer math (avoid float rounding drift): scale × 10000, truncate, / 10000.
	scale := float64(max) / float64(longest)
	dw := sw * int(scale*10000) / 10000
	if dw < 1 {
		dw = 1
	}
	dh := sh * int(scale*10000) / 10000
	if dh < 1 {
		dh = 1
	}
	return dw, dh
}

// FacetProcessOptionsPointer is the convenience caller for UploadInput.Process
// (a *ProcessOptions where nil = passthrough). It returns the facet's options
// by pointer so the store can distinguish "normalize with these opts" from the
// nil "store verbatim" path.
func FacetProcessOptionsPointer(facet ImageFacet) *ProcessOptions {
	opts := FacetProcessOptions(facet)
	return &opts
}

// FacetProcessOptions returns the canonical normalize options for a facet.
// gallery/cover/photo/hero → 2048 lossy q82; avatar → 512 q85; qr → 512 lossless.
// logo/watermark are intentionally NOT defaulted (alpha-sensitive — they opt in
// explicitly with Lossless when migrated; until then they keep using PutObject).
func FacetProcessOptions(facet ImageFacet) ProcessOptions {
	switch facet {
	case FacetAvatar:
		return ProcessOptions{MaxDimension: 512, Quality: 85}
	case FacetQR:
		return ProcessOptions{MaxDimension: 512, Lossless: true}
	case FacetHero:
		return ProcessOptions{MaxDimension: 2048, Quality: 82}
	case FacetGallery, FacetCover, FacetPhoto:
		return ProcessOptions{MaxDimension: 2048, Quality: 82}
	default:
		// Conservative default for any unlisted facet (defensive — opt-in callers
		// pass a known facet). Keeps a future facet from storing a giant original.
		return ProcessOptions{MaxDimension: 2048, Quality: 82}
	}
}
