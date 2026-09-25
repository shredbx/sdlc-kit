package feed

import (
	"bytes"
	"context"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/shredbx/sbx-core/pkg/rss"
)

// pngHeaderWithDims emits just a PNG signature + IHDR chunk declaring w×h. It is
// NOT a decodable image, but image.DecodeConfig reads dimensions from IHDR alone
// — so this lets the decode-bomb guard be tested without allocating a giant
// bitmap (the whole point of checking dimensions BEFORE the full decode).
func pngHeaderWithDims(w, h uint32) []byte {
	var ihdr bytes.Buffer
	binary.Write(&ihdr, binary.BigEndian, w)
	binary.Write(&ihdr, binary.BigEndian, h)
	ihdr.Write([]byte{8, 2, 0, 0, 0}) // bitdepth=8, colortype=2(RGB), comp/filter/interlace=0
	chunk := append([]byte("IHDR"), ihdr.Bytes()...)
	var buf bytes.Buffer
	buf.Write([]byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a}) // signature
	binary.Write(&buf, binary.BigEndian, uint32(len(ihdr.Bytes())))
	buf.Write(chunk)
	binary.Write(&buf, binary.BigEndian, crc32.ChecksumIEEE(chunk))
	return buf.Bytes()
}

// TestThumbnailJPEG_RejectsDecodeBomb proves the megapixel cap rejects an image
// whose declared dimensions exceed maxImagePixels BEFORE the full decode.
func TestThumbnailJPEG_RejectsDecodeBomb(t *testing.T) {
	// 50000×50000 = 2.5 gigapixels, well over the 40MP cap.
	if _, err := thumbnailJPEG(pngHeaderWithDims(50000, 50000), 480); err == nil {
		t.Fatal("thumbnailJPEG accepted a gigapixel image; decode-bomb guard missing")
	} else if !strings.Contains(err.Error(), "too large") {
		t.Fatalf("expected 'too large' error, got %v", err)
	}
}

// makePNG builds a w×h solid-color PNG for resize tests.
func makePNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	c := color.RGBA{R: 10, G: 120, B: 200, A: 255}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}

func TestThumbnailJPEG_DownscalesKeepingAspect(t *testing.T) {
	out, err := thumbnailJPEG(makePNG(t, 1600, 800), 480)
	if err != nil {
		t.Fatalf("thumbnailJPEG: %v", err)
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(out))
	if err != nil {
		t.Fatalf("decode config: %v", err)
	}
	if cfg.Width != 480 || cfg.Height != 240 {
		t.Errorf("got %dx%d, want 480x240", cfg.Width, cfg.Height)
	}
	if _, err := jpeg.Decode(bytes.NewReader(out)); err != nil {
		t.Errorf("output not JPEG: %v", err)
	}
}

func TestThumbnailJPEG_NoUpscale(t *testing.T) {
	out, err := thumbnailJPEG(makePNG(t, 300, 200), 480)
	if err != nil {
		t.Fatalf("thumbnailJPEG: %v", err)
	}
	cfg, _, _ := image.DecodeConfig(bytes.NewReader(out))
	if cfg.Width != 300 || cfg.Height != 200 {
		t.Errorf("got %dx%d, want 300x200 (no upscale)", cfg.Width, cfg.Height)
	}
}

func TestThumbnailJPEG_GarbageInput(t *testing.T) {
	if _, err := thumbnailJPEG([]byte("not-an-image"), 480); err == nil {
		t.Error("expected error on garbage input")
	}
}

// fakeUploader records uploads and returns a deterministic URL.
type fakeUploader struct {
	lastKey         string
	lastContentType string
	lastBytes       int
	uploaded        []byte
}

func (u *fakeUploader) Upload(_ context.Context, key string, data io.Reader, contentType string) error {
	b, _ := io.ReadAll(data)
	u.lastKey = key
	u.lastContentType = contentType
	u.lastBytes = len(b)
	u.uploaded = b
	return nil
}

func (u *fakeUploader) URL(key string) string { return "https://cdn.test/" + key }

// imgTransport serves a PNG for the image URL.
type imgTransport struct{ png []byte }

func (t imgTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if strings.Contains(req.URL.String(), "img.test") {
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewReader(t.png)),
			Header:     http.Header{},
		}, nil
	}
	return &http.Response{StatusCode: 404, Body: http.NoBody, Header: http.Header{}}, nil
}

func TestR2ImageCacher_CacheImage_KeyAndUpload(t *testing.T) {
	pngBytes := makePNG(t, 1200, 900)
	up := &fakeUploader{}
	c := NewR2ImageCacher(up, &http.Client{Transport: imgTransport{png: pngBytes}}, 480)

	key := rss.ItemKey{SourceID: "bangkokpost", GUID: "guid-abc-123"}
	url, err := c.CacheImage(context.Background(), key, "https://img.test/photo.png")
	if err != nil {
		t.Fatalf("CacheImage: %v", err)
	}

	// Object key: Decision #0271 system/news/{itemId}.jpg (owner=system, facet=news).
	if !strings.HasPrefix(up.lastKey, "system/news/") {
		t.Errorf("key = %q, want system/news/ prefix", up.lastKey)
	}
	if !strings.HasSuffix(up.lastKey, ".jpg") {
		t.Errorf("key = %q, want .jpg suffix", up.lastKey)
	}
	if up.lastContentType != "image/jpeg" {
		t.Errorf("contentType = %q, want image/jpeg", up.lastContentType)
	}
	if up.lastBytes == 0 {
		t.Error("uploaded zero bytes")
	}
	if url != "https://cdn.test/"+up.lastKey {
		t.Errorf("returned url = %q, want cdn url for key", url)
	}
	// The uploaded payload is a downscaled JPEG (480 wide for a 4:3 source).
	cfg, _, derr := image.DecodeConfig(bytes.NewReader(up.uploaded))
	if derr != nil {
		t.Fatalf("uploaded not decodable: %v", derr)
	}
	if cfg.Width != 480 {
		t.Errorf("uploaded width = %d, want 480 (thumbnail)", cfg.Width)
	}
}

func TestR2ImageCacher_StableKeyAcrossCalls(t *testing.T) {
	up := &fakeUploader{}
	c := NewR2ImageCacher(up, &http.Client{Transport: imgTransport{png: makePNG(t, 600, 400)}}, 480)
	key := rss.ItemKey{SourceID: "s", GUID: "g"}

	if _, err := c.CacheImage(context.Background(), key, "https://img.test/a.png"); err != nil {
		t.Fatalf("CacheImage #1: %v", err)
	}
	first := up.lastKey
	if _, err := c.CacheImage(context.Background(), key, "https://img.test/a.png"); err != nil {
		t.Fatalf("CacheImage #2: %v", err)
	}
	if up.lastKey != first {
		t.Errorf("key not stable: %q != %q (same item must map to same object key)", up.lastKey, first)
	}
}
