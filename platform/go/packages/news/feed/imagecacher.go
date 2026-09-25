// R2ImageCacher is the BR-local rss.ImageCacher (Decision #0271): it downloads
// an upstream news image (SSRF-guarded via rss.SafeGet), downscales it to a
// small news thumbnail (stdlib box-average resize → JPEG), and uploads it to R2
// under the system/news/{itemId}.jpg key (owner=system, facet=news). The public
// listing then serves the cached R2 URL rather than hotlinking the upstream.
//
// It runs ONLY in the background refresh job (cmd/refresh-feeds + the admin
// "Refresh now" trigger) — never the request path. It keeps pkg/feed
// storage-agnostic: the engine calls the rss.ImageCacher seam; this adapter is
// the concrete R2 implementation wired in main.go.
package feed

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"net/http"

	// Register JPEG + PNG decoders for image.Decode (the upstream may be either).
	_ "image/jpeg"
	_ "image/png"

	"github.com/google/uuid"

	"github.com/shredbx/sbx-core/pkg/rss"
)

// Uploader is the narrow object-store surface the cacher needs — exactly the
// subset of image.ObjectStore used here, so main.go passes the R2ObjectStore
// directly. Keeping it narrow avoids coupling to the full pkg/image.Store.
type Uploader interface {
	Upload(ctx context.Context, key string, data io.Reader, contentType string) error
	URL(key string) string
}

// newsImageNamespace is the deterministic UUIDv5 namespace for deriving a stable
// per-item object id from (sourceID, guid). A stable id means a re-cache
// overwrites the same R2 object rather than orphaning the old one.
var newsImageNamespace = uuid.MustParse("6f9b1f3e-1c2a-4d5b-8e7f-0a1b2c3d4e5f")

// DefaultNewsThumbMaxEdge caps the thumbnail's longest edge. A news card thumb
// needs no more (project_image_size_policy: well under the 2048 gallery cap).
const DefaultNewsThumbMaxEdge = 640

// imageFetchMaxBytes caps the upstream image download (guards a hostile origin).
const imageFetchMaxBytes int64 = 8 << 20 // 8 MB

// maxImagePixels caps decoded pixel area (decode-bomb guard): a small compressed
// file can decode to gigapixels and OOM the refresh worker. 40MP is generous for
// any real article image; checked via DecodeConfig before the full decode.
const maxImagePixels int64 = 40 * 1_000_000

// R2ImageCacher implements rss.ImageCacher over an Uploader + a guarded client.
type R2ImageCacher struct {
	uploader Uploader
	client   *http.Client
	maxEdge  int
}

// NewR2ImageCacher wires the cacher. client SHOULD be rss.DefaultHTTPClient()
// (it carries the SSRF redirect guard); maxEdge <= 0 falls back to
// DefaultNewsThumbMaxEdge.
func NewR2ImageCacher(uploader Uploader, client *http.Client, maxEdge int) *R2ImageCacher {
	if maxEdge <= 0 {
		maxEdge = DefaultNewsThumbMaxEdge
	}
	if client == nil {
		client = rss.DefaultHTTPClient()
	}
	return &R2ImageCacher{uploader: uploader, client: client, maxEdge: maxEdge}
}

// Compile-time assertion that R2ImageCacher satisfies rss.ImageCacher.
var _ rss.ImageCacher = (*R2ImageCacher)(nil)

// CacheImage downloads srcURL (SSRF-guarded, size-capped), thumbnails it, and
// uploads the JPEG to system/news/{itemId}.jpg. It returns the public CDN URL.
// Implements rss.ImageCacher.
func (c *R2ImageCacher) CacheImage(ctx context.Context, key rss.ItemKey, srcURL string) (string, error) {
	raw, err := rss.SafeGet(ctx, c.client, srcURL, imageFetchMaxBytes)
	if err != nil {
		return "", fmt.Errorf("news image fetch %q: %w", srcURL, err)
	}

	thumb, err := thumbnailJPEG(raw, c.maxEdge)
	if err != nil {
		return "", fmt.Errorf("news image thumbnail %q: %w", srcURL, err)
	}

	objectID := uuid.NewSHA1(newsImageNamespace, []byte(key.SourceID+"\x00"+key.GUID)).String()
	objKey := "system/news/" + objectID + ".jpg"

	if err := c.uploader.Upload(ctx, objKey, bytes.NewReader(thumb), "image/jpeg"); err != nil {
		return "", fmt.Errorf("news image upload %q: %w", objKey, err)
	}
	return c.uploader.URL(objKey), nil
}

// thumbnailJPEG decodes raw (JPEG/PNG), downscales it so its longest edge is at
// most maxEdge (aspect preserved, never upscaled) via a box-average resample,
// and re-encodes as JPEG. Stdlib-only (no external resize dep — matches the
// pkg/feed zero-dependency posture).
func thumbnailJPEG(raw []byte, maxEdge int) ([]byte, error) {
	// Decode-bomb guard: read only the header first and reject oversized pixel
	// dimensions BEFORE the full decode, which would otherwise allocate the whole
	// source bitmap (a small, highly-compressed PNG can decode to gigapixels).
	cfg, _, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("decode config: %w", err)
	}
	if int64(cfg.Width)*int64(cfg.Height) > maxImagePixels {
		return nil, fmt.Errorf("image too large: %dx%d exceeds %d megapixel cap", cfg.Width, cfg.Height, maxImagePixels/1_000_000)
	}

	src, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	b := src.Bounds()
	sw, sh := b.Dx(), b.Dy()
	if sw == 0 || sh == 0 {
		return nil, fmt.Errorf("zero-size image")
	}

	dw, dh := scaledDims(sw, sh, maxEdge)
	dst := boxResize(src, dw, dh)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: 82}); err != nil {
		return nil, fmt.Errorf("encode jpeg: %w", err)
	}
	return buf.Bytes(), nil
}

// scaledDims returns the target dimensions clamping the longest edge to maxEdge
// while preserving aspect ratio. An image already within maxEdge is unchanged
// (no upscaling).
func scaledDims(w, h, maxEdge int) (int, int) {
	if w <= maxEdge && h <= maxEdge {
		return w, h
	}
	if w >= h {
		nh := h * maxEdge / w
		if nh < 1 {
			nh = 1
		}
		return maxEdge, nh
	}
	nw := w * maxEdge / h
	if nw < 1 {
		nw = 1
	}
	return nw, maxEdge
}

// boxResize downscales src to dw×dh by averaging each destination pixel's source
// box (a simple, dependency-free area resample — adequate for a small news
// thumbnail). When dims are unchanged it still copies into an RGBA so the JPEG
// encoder gets an opaque image.
func boxResize(src image.Image, dw, dh int) *image.RGBA {
	b := src.Bounds()
	sw, sh := b.Dx(), b.Dy()
	dst := image.NewRGBA(image.Rect(0, 0, dw, dh))

	for dy := 0; dy < dh; dy++ {
		sy0 := b.Min.Y + dy*sh/dh
		sy1 := b.Min.Y + (dy+1)*sh/dh
		if sy1 <= sy0 {
			sy1 = sy0 + 1
		}
		for dx := 0; dx < dw; dx++ {
			sx0 := b.Min.X + dx*sw/dw
			sx1 := b.Min.X + (dx+1)*sw/dw
			if sx1 <= sx0 {
				sx1 = sx0 + 1
			}
			var rs, gs, bs, n uint64
			for sy := sy0; sy < sy1; sy++ {
				for sx := sx0; sx < sx1; sx++ {
					r, g, bb, _ := src.At(sx, sy).RGBA()
					rs += uint64(r >> 8)
					gs += uint64(g >> 8)
					bs += uint64(bb >> 8)
					n++
				}
			}
			if n == 0 {
				n = 1
			}
			i := dst.PixOffset(dx, dy)
			dst.Pix[i] = uint8(rs / n)
			dst.Pix[i+1] = uint8(gs / n)
			dst.Pix[i+2] = uint8(bs / n)
			dst.Pix[i+3] = 0xff
		}
	}
	return dst
}
