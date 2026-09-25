package image

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func minimalDarkJPEG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	for y := range 10 {
		for x := range 10 {
			img.Set(x, y, color.RGBA{R: 26, G: 26, B: 46, A: 255})
		}
	}
	var buf bytes.Buffer
	require.NoError(t, jpeg.Encode(&buf, img, &jpeg.Options{Quality: 80}))
	return buf.Bytes()
}

// =============================================================================
// S-IMG-1.x: Named type constants
// =============================================================================

func TestImageFormatConstants(t *testing.T) {
	assert.Equal(t, ImageFormat("jpg"), ImageFormatJPG)
	assert.Equal(t, ImageFormat("png"), ImageFormatPNG)
	assert.Equal(t, ImageFormat("webp"), ImageFormatWebP)
	assert.Equal(t, ImageFormat("avif"), ImageFormatAVIF)
}

func TestImageFacetConstants(t *testing.T) {
	assert.Equal(t, ImageFacet("cover"), FacetCover)
	assert.Equal(t, ImageFacet("gallery"), FacetGallery)
	assert.Equal(t, ImageFacet("avatar"), FacetAvatar)
	assert.Equal(t, ImageFacet("photo"), FacetPhoto)
	assert.Equal(t, ImageFacet("qr"), FacetQR)
	assert.Equal(t, ImageFacet("hero"), FacetHero)
	assert.Equal(t, ImageFacet("watermark"), FacetWatermark)
	// PurposeGeneral was REMOVED (Decision #0271, OSK-002 — no "general" catch-all).
}

// =============================================================================
// S-IMG-2.1: Upload populates primary_color from real JPEG
// =============================================================================

func TestStoreUploadPopulatesColorAnalysis(t *testing.T) {
	backend := NewMemoryBackend()
	objects := NewMemoryObjectStore("https://media.example.com")
	store := NewStore(backend, objects)
	ctx := context.Background()

	img, err := store.Upload(ctx, UploadInput{
		File:        bytes.NewReader(minimalDarkJPEG(t)),
		ContentType: "image/jpeg",
		Owner:       OwnerProperties,
		OwnerID:     "88888888-8888-8888-8888-888888888888",
		Facet:       FacetCover,
		AltText:     "dark test image",
	})
	require.NoError(t, err)

	assert.Regexp(t, `^#[0-9A-Fa-f]{6}$`, img.PrimaryColor, "PrimaryColor should be hex")
}

// =============================================================================
// S-IMG-2.2: Color analysis failure is non-fatal
// =============================================================================

func TestStoreUploadColorAnalysisFailureNonFatal(t *testing.T) {
	backend := NewMemoryBackend()
	objects := NewMemoryObjectStore("https://media.example.com")
	store := NewStore(backend, objects)
	ctx := context.Background()

	img, err := store.Upload(ctx, UploadInput{
		File:        bytes.NewReader([]byte{}),
		ContentType: "image/jpeg",
		Owner:       OwnerProperties,
		OwnerID:     "99999999-9999-9999-9999-999999999999",
		Facet:       FacetCover,
	})
	require.NoError(t, err, "upload must succeed even when color analysis fails")
	assert.Empty(t, img.PrimaryColor)
}

// =============================================================================
// S-IMG-4.1: MemoryBackend Create/Get round-trips primary_color
// =============================================================================

func TestMemoryBackendColorFieldRoundTrip(t *testing.T) {
	backend := NewMemoryBackend()
	ctx := context.Background()

	img := testDarkImageFromFixture(t)
	require.NoError(t, backend.Create(ctx, img))

	got, err := backend.Get(ctx, img.ID)
	require.NoError(t, err)
	assert.Equal(t, "#1a1a2e", got.PrimaryColor)
}

// =============================================================================
// S-IMG-4.2: FileBackend persists and reloads primary_color
// =============================================================================

func TestFileBackendColorFieldPersistence(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()

	img := testDarkImageFromFixture(t)
	b1 := NewFileBackend(dir)
	require.NoError(t, b1.Create(ctx, img))

	b2 := NewFileBackend(dir)
	got, err := b2.Get(ctx, img.ID)
	require.NoError(t, err)
	assert.Equal(t, "#1a1a2e", got.PrimaryColor)
}

// =============================================================================
// S-IMG-4.5: MemoryBackend List filters by ImagePurpose
// =============================================================================

func TestMemoryBackendListFiltersByFacet(t *testing.T) {
	backend := NewMemoryBackend()
	ctx := context.Background()

	for i, f := range []ImageFacet{FacetCover, FacetCover, FacetGallery} {
		require.NoError(t, backend.Create(ctx, &Image{
			ID:      fmt.Sprintf("img-%d", i),
			Purpose: f,
		}))
	}

	covers, err := backend.List(ctx, ListOpts{Facet: FacetCover})
	require.NoError(t, err)
	assert.Len(t, covers, 2)

	gallery, err := backend.List(ctx, ListOpts{Facet: FacetGallery})
	require.NoError(t, err)
	assert.Len(t, gallery, 1)
}

// =============================================================================
// S-IMG-4.6: Soft delete — deleted image not returned by Get or List
// =============================================================================

func TestMemoryBackendSoftDeleteColorImage(t *testing.T) {
	backend := NewMemoryBackend()
	ctx := context.Background()

	img := testDarkImageFromFixture(t)
	require.NoError(t, backend.Create(ctx, img))
	require.NoError(t, backend.Delete(ctx, img.ID))

	_, err := backend.Get(ctx, img.ID)
	assert.ErrorIs(t, err, ErrImageNotFound)

	list, err := backend.List(ctx, ListOpts{})
	require.NoError(t, err)
	assert.Len(t, list, 0)
}

// =============================================================================
// HELPERS
// =============================================================================

func testDarkImageFromFixture(t *testing.T) *Image {
	t.Helper()
	data, err := os.ReadFile("testdata/fixtures/dark_image.yml")
	require.NoError(t, err)

	var img Image
	require.NoError(t, yaml.Unmarshal(data, &img))
	return &img
}
