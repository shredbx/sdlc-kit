package image

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// STORE TESTS (integration: FileBackend + FileObjectStore)
// =============================================================================

func TestStoreUpload(t *testing.T) {
	dir := t.TempDir()
	store := newTestStore(dir)
	ctx := context.Background()

	const ownerID = "11111111-1111-1111-1111-111111111111"
	img, err := store.Upload(ctx, UploadInput{
		File:        bytes.NewReader([]byte("fake-jpeg-data")),
		ContentType: "image/jpeg",
		AltText:     "A luxury villa",
		Owner:       OwnerProperties,
		OwnerID:     ownerID,
		Facet:       FacetGallery,
		CreatedBy:   "test-user",
	})

	require.NoError(t, err)
	assert.NotEmpty(t, img.ID)
	assert.Equal(t, ImageFormatJPG, img.Format)
	assert.Equal(t, FacetGallery, img.Purpose)
	assert.Equal(t, "A luxury villa", img.AltText)
	assert.Equal(t, "test-user", img.CreatedBy)
	// Owner-scoped key: {owner}/{ownerId}/{facet}/{objectId}.{ext} (Decision #0271).
	want := "properties/" + ownerID + "/gallery/" + img.ID + ".jpg"
	assert.Equal(t, want, img.Key)
	assert.Contains(t, img.URL, want)
}

func TestStoreUploadMissingOwner(t *testing.T) {
	dir := t.TempDir()
	store := newTestStore(dir)
	ctx := context.Background()

	// No owner at all → ErrMissingOwner.
	_, err := store.Upload(ctx, UploadInput{
		File:        bytes.NewReader([]byte("data")),
		ContentType: "image/jpeg",
		Facet:       FacetGallery,
	})
	assert.ErrorIs(t, err, ErrMissingOwner)

	// Owner present but no owner ID (non-system) → ErrMissingOwner.
	_, err = store.Upload(ctx, UploadInput{
		File:        bytes.NewReader([]byte("data")),
		ContentType: "image/jpeg",
		Owner:       OwnerProperties,
		Facet:       FacetGallery,
	})
	assert.ErrorIs(t, err, ErrMissingOwner)
}

func TestStoreUploadMissingFacet(t *testing.T) {
	dir := t.TempDir()
	store := newTestStore(dir)
	ctx := context.Background()

	_, err := store.Upload(ctx, UploadInput{
		File:        bytes.NewReader([]byte("data")),
		ContentType: "image/jpeg",
		Owner:       OwnerProperties,
		OwnerID:     "11111111-1111-1111-1111-111111111111",
	})
	assert.ErrorIs(t, err, ErrMissingFacet)
}

func TestStoreUploadSystemAsset(t *testing.T) {
	dir := t.TempDir()
	store := newTestStore(dir)
	ctx := context.Background()

	// System/brand asset: no owner ID; key is system/{facet}/{objectId}.{ext}.
	img, err := store.Upload(ctx, UploadInput{
		File:        bytes.NewReader([]byte("watermark-png")),
		ContentType: "image/png",
		Owner:       OwnerSystem,
		Facet:       FacetWatermark,
	})
	require.NoError(t, err)
	want := "system/watermark/" + img.ID + ".png"
	assert.Equal(t, want, img.Key)
	assert.Contains(t, img.URL, want)
}

func TestStoreUploadInvalidFormat(t *testing.T) {
	dir := t.TempDir()
	store := newTestStore(dir)
	ctx := context.Background()

	_, err := store.Upload(ctx, UploadInput{
		File:        bytes.NewReader([]byte("not-an-image")),
		ContentType: "application/pdf",
	})

	assert.ErrorIs(t, err, ErrInvalidFormat)
}

func TestStoreGet(t *testing.T) {
	dir := t.TempDir()
	store := newTestStore(dir)
	ctx := context.Background()

	uploaded, err := store.Upload(ctx, UploadInput{
		File:        bytes.NewReader([]byte("png-data")),
		ContentType: "image/png",
		Owner:       OwnerAgents,
		OwnerID:     "22222222-2222-2222-2222-222222222222",
		Facet:       FacetAvatar,
	})
	require.NoError(t, err)

	got, err := store.Get(ctx, uploaded.ID)
	require.NoError(t, err)
	assert.Equal(t, uploaded.ID, got.ID)
	assert.Equal(t, ImageFormatPNG, got.Format)
}

func TestStoreGetNotFound(t *testing.T) {
	dir := t.TempDir()
	store := newTestStore(dir)
	ctx := context.Background()

	_, err := store.Get(ctx, "nonexistent-id")
	assert.ErrorIs(t, err, ErrImageNotFound)
}

func TestStoreGetEmptyID(t *testing.T) {
	dir := t.TempDir()
	store := newTestStore(dir)
	ctx := context.Background()

	_, err := store.Get(ctx, "")
	assert.ErrorIs(t, err, ErrInvalidID)
}

func TestStoreDelete(t *testing.T) {
	dir := t.TempDir()
	store := newTestStore(dir)
	ctx := context.Background()

	img, err := store.Upload(ctx, UploadInput{
		File:        bytes.NewReader([]byte("data")),
		ContentType: "image/webp",
		Owner:       OwnerProperties,
		OwnerID:     "33333333-3333-3333-3333-333333333333",
		Facet:       FacetCover,
	})
	require.NoError(t, err)

	err = store.Delete(ctx, img.ID)
	require.NoError(t, err)

	_, err = store.Get(ctx, img.ID)
	assert.ErrorIs(t, err, ErrImageNotFound)
}

func TestStoreList(t *testing.T) {
	dir := t.TempDir()
	store := newTestStore(dir)
	ctx := context.Background()

	// Upload 3 images: 2 gallery, 1 avatar
	for _, facet := range []ImageFacet{FacetGallery, FacetGallery, FacetAvatar} {
		_, err := store.Upload(ctx, UploadInput{
			File:        bytes.NewReader([]byte("data")),
			ContentType: "image/jpeg",
			Owner:       OwnerProperties,
			OwnerID:     "44444444-4444-4444-4444-444444444444",
			Facet:       facet,
		})
		require.NoError(t, err)
	}

	// List all
	all, err := store.List(ctx, ListOpts{})
	require.NoError(t, err)
	assert.Len(t, all, 3)

	// List by facet
	gallery, err := store.List(ctx, ListOpts{Facet: FacetGallery})
	require.NoError(t, err)
	assert.Len(t, gallery, 2)

	avatars, err := store.List(ctx, ListOpts{Facet: FacetAvatar})
	require.NoError(t, err)
	assert.Len(t, avatars, 1)
}

func TestStoreListPagination(t *testing.T) {
	dir := t.TempDir()
	store := newTestStore(dir)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		_, err := store.Upload(ctx, UploadInput{
			File:        bytes.NewReader([]byte("data")),
			ContentType: "image/jpeg",
			Owner:       OwnerProperties,
			OwnerID:     "55555555-5555-5555-5555-555555555555",
			Facet:       FacetGallery,
		})
		require.NoError(t, err)
	}

	page, err := store.List(ctx, ListOpts{Limit: 2})
	require.NoError(t, err)
	assert.Len(t, page, 2)
}

func TestStoreUpdateAltText(t *testing.T) {
	dir := t.TempDir()
	store := newTestStore(dir)
	ctx := context.Background()

	img, err := store.Upload(ctx, UploadInput{
		File:        bytes.NewReader([]byte("data")),
		ContentType: "image/jpeg",
		AltText:     "original",
		Owner:       OwnerProperties,
		OwnerID:     "66666666-6666-6666-6666-666666666666",
		Facet:       FacetGallery,
	})
	require.NoError(t, err)

	err = store.UpdateAltText(ctx, img.ID, "updated alt text")
	require.NoError(t, err)

	got, err := store.Get(ctx, img.ID)
	require.NoError(t, err)
	assert.Equal(t, "updated alt text", got.AltText)
}

// =============================================================================
// FILE OBJECT STORE TESTS
// =============================================================================

func TestFileObjectStoreUploadDownload(t *testing.T) {
	dir := t.TempDir()
	objStore := NewFileObjectStore(filepath.Join(dir, "objects"), "http://localhost:5000/images")

	ctx := context.Background()
	content := []byte("binary-image-content")

	err := objStore.Upload(ctx, "gallery/test-id.jpg", bytes.NewReader(content), "image/jpeg")
	require.NoError(t, err)

	// Verify file exists on disk
	path := filepath.Join(dir, "objects", "gallery", "test-id.jpg")
	_, err = os.Stat(path)
	require.NoError(t, err)

	// Download and verify content
	reader, err := objStore.Download(ctx, "gallery/test-id.jpg")
	require.NoError(t, err)
	defer reader.Close()

	downloaded, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, content, downloaded)
}

func TestFileObjectStoreURL(t *testing.T) {
	objStore := NewFileObjectStore("/tmp", "http://localhost:5000/images")
	url := objStore.URL("gallery/abc-123.jpg")
	assert.Equal(t, "http://localhost:5000/images/gallery/abc-123.jpg", url)
}

func TestFileObjectStoreDeleteIdempotent(t *testing.T) {
	dir := t.TempDir()
	objStore := NewFileObjectStore(dir, "http://localhost:5000/images")
	ctx := context.Background()

	// Delete nonexistent key — should not error (idempotent)
	err := objStore.Delete(ctx, "nonexistent/key.jpg")
	assert.NoError(t, err)
}

// =============================================================================
// HELPER TESTS
// =============================================================================

func TestFormatFromContentType(t *testing.T) {
	tests := []struct {
		ct       string
		expected ImageFormat
	}{
		{"image/jpeg", ImageFormatJPG},
		{"image/png", ImageFormatPNG},
		{"image/webp", ImageFormatWebP},
		{"image/avif", ImageFormatAVIF},
		{"application/pdf", ""},
		{"text/plain", ""},
	}

	for _, tt := range tests {
		t.Run(tt.ct, func(t *testing.T) {
			assert.Equal(t, tt.expected, formatFromContentType(tt.ct))
		})
	}
}

func TestBuildKey(t *testing.T) {
	// objectID is always a server-minted UUID (Decision #0271 — buildKey validates it).
	oid := uuid.NewString()

	// Owner-scoped: {owner}/{ownerId}/{facet}/{objectId}.{ext} (OSK-001).
	k, err := buildKey(OwnerProperties, "p-1", FacetGallery, oid, ImageFormatJPG)
	require.NoError(t, err)
	assert.Equal(t, "properties/p-1/gallery/"+oid+".jpg", k)

	k, err = buildKey(OwnerAgents, "a-9", FacetQR, oid, ImageFormatPNG)
	require.NoError(t, err)
	assert.Equal(t, "agents/a-9/qr/"+oid+".png", k)

	// System/brand asset: system/{facet}/{objectId}.{ext} — no owner-id (OSK-002).
	k, err = buildKey(OwnerSystem, "", FacetWatermark, oid, ImageFormatPNG)
	require.NoError(t, err)
	assert.Equal(t, "system/watermark/"+oid+".png", k)
}

func TestBuildKeyErrors(t *testing.T) {
	oid := uuid.NewString()

	// Missing owner.
	_, err := buildKey("", "p-1", FacetGallery, oid, ImageFormatJPG)
	assert.ErrorIs(t, err, ErrMissingOwner)

	// Non-system owner without owner ID.
	_, err = buildKey(OwnerProperties, "", FacetGallery, oid, ImageFormatJPG)
	assert.ErrorIs(t, err, ErrMissingOwner)

	// Missing facet.
	_, err = buildKey(OwnerProperties, "p-1", "", oid, ImageFormatJPG)
	assert.ErrorIs(t, err, ErrMissingFacet)

	// Missing object id.
	_, err = buildKey(OwnerProperties, "p-1", FacetGallery, "", ImageFormatJPG)
	assert.ErrorIs(t, err, ErrInvalidID)

	// Invalid owner (uppercase / slash / path traversal must be rejected).
	for _, bad := range []ImageOwner{"Properties", "a/b", "../etc", "agents.", "-bad"} {
		_, err = buildKey(bad, "p-1", FacetGallery, oid, ImageFormatJPG)
		assert.ErrorIsf(t, err, ErrInvalidOwner, "owner %q should be invalid", bad)
	}

	// Invalid facet — a slash / path-traversal / dot facet must never reach the key
	// (defense-in-depth: buildKey is documented "safe to call directly").
	for _, bad := range []ImageFacet{"../x", "a/b", "Cover", "cover.", "-x", "qr/../etc"} {
		_, err = buildKey(OwnerProperties, "p-1", bad, oid, ImageFormatJPG)
		assert.ErrorIsf(t, err, ErrInvalidFacet, "facet %q should be invalid", bad)
	}

	// Invalid object ID — anything that is not a UUID is rejected so it can never
	// forge a path segment or extension.
	for _, bad := range []string{"abc", "../../etc/passwd", "a/b", "not-a-uuid", "12345"} {
		_, err = buildKey(OwnerProperties, "p-1", FacetGallery, bad, ImageFormatJPG)
		assert.ErrorIsf(t, err, ErrInvalidObjectID, "objectID %q should be invalid", bad)
	}
}

// =============================================================================
// MEMORY BACKEND TESTS
// =============================================================================

func TestMemoryBackendCRUD(t *testing.T) {
	backend := NewMemoryBackend()
	ctx := context.Background()

	img := &Image{
		ID:        "test-id-1",
		Key:       "gallery/test-id-1.jpg",
		URL:       "http://test/gallery/test-id-1.jpg",
		Format:    ImageFormatJPG,
		Purpose:   PurposeGallery,
		AltText:   "test image",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	// Create
	err := backend.Create(ctx, img)
	require.NoError(t, err)

	// Get
	got, err := backend.Get(ctx, "test-id-1")
	require.NoError(t, err)
	assert.Equal(t, "test-id-1", got.ID)
	assert.Equal(t, PurposeGallery, got.Purpose)

	// Update
	got.AltText = "updated"
	err = backend.Update(ctx, got)
	require.NoError(t, err)

	got2, err := backend.Get(ctx, "test-id-1")
	require.NoError(t, err)
	assert.Equal(t, "updated", got2.AltText)

	// Delete (soft)
	err = backend.Delete(ctx, "test-id-1")
	require.NoError(t, err)

	_, err = backend.Get(ctx, "test-id-1")
	assert.ErrorIs(t, err, ErrImageNotFound)
}

func TestMemoryBackendList(t *testing.T) {
	backend := NewMemoryBackend()
	ctx := context.Background()
	now := time.Now().UTC()

	for i, facet := range []ImageFacet{FacetGallery, FacetGallery, FacetAvatar, FacetCover} {
		err := backend.Create(ctx, &Image{
			ID:        fmt.Sprintf("img-%d", i),
			Purpose:   facet,
			CreatedAt: now,
			UpdatedAt: now,
		})
		require.NoError(t, err)
	}

	// All
	all, err := backend.List(ctx, ListOpts{})
	require.NoError(t, err)
	assert.Len(t, all, 4)

	// Filter by facet
	gallery, err := backend.List(ctx, ListOpts{Facet: FacetGallery})
	require.NoError(t, err)
	assert.Len(t, gallery, 2)

	// Pagination
	page, err := backend.List(ctx, ListOpts{Limit: 2})
	require.NoError(t, err)
	assert.Len(t, page, 2)
}

// =============================================================================
// MEMORY OBJECT STORE TESTS
// =============================================================================

func TestMemoryObjectStoreCRUD(t *testing.T) {
	objStore := NewMemoryObjectStore("https://test.example.com")
	ctx := context.Background()

	content := []byte("test-binary-data")

	// Upload
	err := objStore.Upload(ctx, "gallery/test.jpg", bytes.NewReader(content), "image/jpeg")
	require.NoError(t, err)
	assert.True(t, objStore.Has("gallery/test.jpg"))
	assert.Equal(t, 1, objStore.Len())

	// Download
	reader, err := objStore.Download(ctx, "gallery/test.jpg")
	require.NoError(t, err)
	defer reader.Close()

	downloaded, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, content, downloaded)

	// URL
	url := objStore.URL("gallery/test.jpg")
	assert.Equal(t, "https://test.example.com/gallery/test.jpg", url)

	// Delete
	err = objStore.Delete(ctx, "gallery/test.jpg")
	require.NoError(t, err)
	assert.False(t, objStore.Has("gallery/test.jpg"))
	assert.Equal(t, 0, objStore.Len())
}

func TestMemoryObjectStoreDownloadNotFound(t *testing.T) {
	objStore := NewMemoryObjectStore("https://test.example.com")
	ctx := context.Background()

	_, err := objStore.Download(ctx, "nonexistent.jpg")
	assert.ErrorIs(t, err, ErrImageNotFound)
}

// =============================================================================
// STORE WITH MEMORY BACKENDS
// =============================================================================

func TestStoreWithMemoryBackends(t *testing.T) {
	backend := NewMemoryBackend()
	objects := NewMemoryObjectStore("https://media.example.com")
	store := NewStore(backend, objects)
	ctx := context.Background()

	// Upload
	img, err := store.Upload(ctx, UploadInput{
		File:        bytes.NewReader([]byte("jpeg-bytes")),
		ContentType: "image/jpeg",
		AltText:     "test villa",
		Owner:       OwnerProperties,
		OwnerID:     "77777777-7777-7777-7777-777777777777",
		Facet:       FacetHero,
		CreatedBy:   "test-user",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, img.ID)
	assert.Equal(t, ImageFormatJPG, img.Format)
	assert.Equal(t, FacetHero, img.Purpose)
	assert.Contains(t, img.URL, "https://media.example.com/properties/77777777-7777-7777-7777-777777777777/hero/")
	assert.True(t, objects.Has(img.Key))

	// Get
	got, err := store.Get(ctx, img.ID)
	require.NoError(t, err)
	assert.Equal(t, img.ID, got.ID)

	// Delete (soft delete on metadata, binary stays)
	err = store.Delete(ctx, img.ID)
	require.NoError(t, err)

	_, err = store.Get(ctx, img.ID)
	assert.ErrorIs(t, err, ErrImageNotFound)

	// Binary still exists in object store (soft delete)
	assert.True(t, objects.Has(img.Key))
}

// TestStoreDeleteObject covers DeleteObject — the key-addressed delete primitive that
// is the counterpart of PutObject/GetObject. Unlike Store.Delete (which soft-deletes an
// images ROW and leaves the binary), DeleteObject removes the binary DIRECTLY by key,
// with no metadata row involved — the shape a key-only system asset (e.g. a superseded
// logo-tint raster) needs to be reclaimed by.
func TestStoreDeleteObject(t *testing.T) {
	objects := NewMemoryObjectStore("https://media.example.com")
	store := NewStore(NewMemoryBackend(), objects)
	ctx := context.Background()

	// A key-addressed system object with NO images row — exactly what the logo-tint
	// baker writes via PutObject.
	key := "system/logo/abc.png"
	url, err := store.PutObject(ctx, key, bytes.NewReader([]byte("png-bytes")), "image/png")
	require.NoError(t, err)
	assert.Contains(t, url, key)
	require.True(t, objects.Has(key))

	// DeleteObject removes the binary directly by key.
	require.NoError(t, store.DeleteObject(ctx, key))
	assert.False(t, objects.Has(key))

	// An empty key is rejected; deleting an already-absent key is idempotent (no error),
	// so a best-effort cleanup of a possibly-missing object never fails the caller.
	assert.ErrorIs(t, store.DeleteObject(ctx, ""), ErrInvalidID)
	assert.NoError(t, store.DeleteObject(ctx, key))
}

// =============================================================================
// REPLACE-RECLAIM TESTS (single-storage slots — the centralized mechanism)
// =============================================================================
//
// These exercise Store's upload-time replace-reclaim: when a ReplacePolicy marks an
// (owner, facet) as a single-storage slot, a new upload reclaims the prior image
// (binary + row) automatically, with no consumer wiring. Pool slots append. Nil policy
// is append-only (today's behavior). All use the in-memory fakes — no DB, no R2.

// fakeReplacePolicy is a test double for ReplacePolicy — a static set of single
// (owner, facet) slots. Mirrors how a real consumer adapter (BR's) would declare its
// cardinality: a map lookup over (owner, facet).
type fakeReplacePolicy struct{ single map[string]bool }

func (p fakeReplacePolicy) IsReplaceSlot(owner ImageOwner, facet ImageFacet) bool {
	return p.single[string(owner)+"|"+string(facet)]
}

// listErrorBackend wraps MemoryBackend but fails ListBySlot, to prove reclaim is
// best-effort (an upload succeeds even when the prior-slot enumeration errors).
type listErrorBackend struct{ *MemoryBackend }

func (listErrorBackend) ListBySlot(context.Context, ImageOwner, string, ImageFacet) ([]Image, error) {
	return nil, errors.New("simulated list failure")
}

func TestUploadReplaceSlotReclaimsPrior(t *testing.T) {
	backend := NewMemoryBackend()
	objects := NewMemoryObjectStore("https://media.example.com")
	// One avatar per agent — a single-storage slot.
	policy := fakeReplacePolicy{single: map[string]bool{"agents|avatar": true}}
	store := NewStore(backend, objects).WithReplacePolicy(policy)
	ctx := context.Background()
	const agentID = "88888888-8888-8888-8888-888888888888"

	first, err := store.Upload(ctx, UploadInput{
		File: bytes.NewReader([]byte("avatar-A")), ContentType: "image/jpeg",
		Owner: OwnerAgents, OwnerID: agentID, Facet: FacetAvatar,
	})
	require.NoError(t, err)
	require.True(t, objects.Has(first.Key))

	// Replace: a second avatar to the SAME slot.
	second, err := store.Upload(ctx, UploadInput{
		File: bytes.NewReader([]byte("avatar-B")), ContentType: "image/jpeg",
		Owner: OwnerAgents, OwnerID: agentID, Facet: FacetAvatar,
	})
	require.NoError(t, err)

	// The prior is reclaimed (binary + row gone); only the new one remains.
	assert.False(t, objects.Has(first.Key), "prior avatar binary must be reclaimed on replace")
	assert.True(t, objects.Has(second.Key), "new avatar binary must remain")
	_, err = backend.Get(ctx, first.ID)
	assert.ErrorIs(t, err, ErrImageNotFound, "prior avatar row must be reclaimed on replace")
	got, err := backend.Get(ctx, second.ID)
	require.NoError(t, err)
	assert.Equal(t, FacetAvatar, got.Purpose)
}

func TestUploadReplaceSlotSystemAsset(t *testing.T) {
	// The ownerless system/logo slot is single too (one system logo). Its prefix is
	// system/logo/ (no owner-id segment) — this exercises the OwnerSystem branch of
	// slotPrefix, proving reclaim keys off the right prefix for brand assets.
	backend := NewMemoryBackend()
	objects := NewMemoryObjectStore("https://media.example.com")
	policy := fakeReplacePolicy{single: map[string]bool{"system|logo": true}}
	store := NewStore(backend, objects).WithReplacePolicy(policy)
	ctx := context.Background()

	first, err := store.Upload(ctx, UploadInput{
		File: bytes.NewReader([]byte("logo-A")), ContentType: "image/png",
		Owner: OwnerSystem, Facet: FacetLogo,
	})
	require.NoError(t, err)
	second, err := store.Upload(ctx, UploadInput{
		File: bytes.NewReader([]byte("logo-B")), ContentType: "image/png",
		Owner: OwnerSystem, Facet: FacetLogo,
	})
	require.NoError(t, err)

	assert.False(t, objects.Has(first.Key), "prior system logo binary must be reclaimed")
	assert.True(t, objects.Has(second.Key))
	_, err = backend.Get(ctx, first.ID)
	assert.ErrorIs(t, err, ErrImageNotFound, "prior system logo row must be reclaimed")
}

func TestUploadPoolSlotKeepsAll(t *testing.T) {
	// (agents, avatar) is single in the policy; (properties, gallery) is NOT — it is a
	// pool. Two gallery uploads to the same owner must BOTH survive (append; no reclaim).
	backend := NewMemoryBackend()
	objects := NewMemoryObjectStore("https://media.example.com")
	policy := fakeReplacePolicy{single: map[string]bool{"agents|avatar": true}}
	store := NewStore(backend, objects).WithReplacePolicy(policy)
	ctx := context.Background()
	const propID = "99999999-9999-9999-9999-999999999999"

	first, err := store.Upload(ctx, UploadInput{
		File: bytes.NewReader([]byte("gallery-A")), ContentType: "image/jpeg",
		Owner: OwnerProperties, OwnerID: propID, Facet: FacetGallery,
	})
	require.NoError(t, err)
	second, err := store.Upload(ctx, UploadInput{
		File: bytes.NewReader([]byte("gallery-B")), ContentType: "image/jpeg",
		Owner: OwnerProperties, OwnerID: propID, Facet: FacetGallery,
	})
	require.NoError(t, err)

	assert.True(t, objects.Has(first.Key), "pool slot must keep the first image")
	assert.True(t, objects.Has(second.Key), "pool slot must keep the second image")
	all, err := backend.List(ctx, ListOpts{Facet: FacetGallery})
	require.NoError(t, err)
	assert.Len(t, all, 2)
}

func TestUploadNilPolicyAppendOnly(t *testing.T) {
	// No WithReplacePolicy → append-only even for a naturally-single slot. This is the
	// regression guard: existing callers that never plug a policy see zero behavior
	// change (today's append-only behavior, the same as before the mechanism existed).
	backend := NewMemoryBackend()
	objects := NewMemoryObjectStore("https://media.example.com")
	store := NewStore(backend, objects)
	ctx := context.Background()
	const agentID = "77777777-7777-7777-7777-777777777777"

	first, err := store.Upload(ctx, UploadInput{
		File: bytes.NewReader([]byte("avatar-A")), ContentType: "image/jpeg",
		Owner: OwnerAgents, OwnerID: agentID, Facet: FacetAvatar,
	})
	require.NoError(t, err)
	_, err = store.Upload(ctx, UploadInput{
		File: bytes.NewReader([]byte("avatar-B")), ContentType: "image/jpeg",
		Owner: OwnerAgents, OwnerID: agentID, Facet: FacetAvatar,
	})
	require.NoError(t, err)

	assert.True(t, objects.Has(first.Key), "nil policy must not reclaim (append-only — today's behavior)")
}

func TestUploadReplaceSlotFirstUploadNoPrior(t *testing.T) {
	// First upload to a single slot — nothing to reclaim. Must succeed cleanly with no
	// error and exactly one image in the slot.
	backend := NewMemoryBackend()
	objects := NewMemoryObjectStore("https://media.example.com")
	policy := fakeReplacePolicy{single: map[string]bool{"agents|avatar": true}}
	store := NewStore(backend, objects).WithReplacePolicy(policy)
	ctx := context.Background()

	img, err := store.Upload(ctx, UploadInput{
		File: bytes.NewReader([]byte("avatar")), ContentType: "image/jpeg",
		Owner: OwnerAgents, OwnerID: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", Facet: FacetAvatar,
	})
	require.NoError(t, err)
	assert.True(t, objects.Has(img.Key))
	all, err := backend.List(ctx, ListOpts{Facet: FacetAvatar})
	require.NoError(t, err)
	assert.Len(t, all, 1)
}

func TestUploadReplaceSlotReclaimsPriorRealObjectStore(t *testing.T) {
	// REAL-I/O proof (not the in-memory fake): FileBackend + FileObjectStore. A
	// replacement upload to a single-storage slot must physically REMOVE the prior
	// binary from the object store (the file is gone from disk) AND its metadata row.
	// This is the exact Store.Upload → reclaimSlotPriors → ReclaimByKey → objects.Delete
	// path R2ObjectStore runs in prod — FileObjectStore.Delete is the same ObjectStore
	// contract (S3 DeleteObject is R2's impl), so a file removed here = an R2 object
	// removed there. Asserts PRESENCE of removal (os.IsNotExist), not absence of error.
	dir := t.TempDir()
	store := newTestStore(dir).WithReplacePolicy(fakeReplacePolicy{
		single: map[string]bool{"agents|avatar": true},
	})
	ctx := context.Background()
	const agentID = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"

	first, err := store.Upload(ctx, UploadInput{
		File: bytes.NewReader([]byte("avatar-A")), ContentType: "image/jpeg",
		Owner: OwnerAgents, OwnerID: agentID, Facet: FacetAvatar,
	})
	require.NoError(t, err)
	firstPath := filepath.Join(dir, "objects", first.Key)
	_, err = os.Stat(firstPath)
	require.NoError(t, err, "prior avatar binary must exist on disk right after the first upload")

	// Replace: a second avatar to the SAME slot.
	second, err := store.Upload(ctx, UploadInput{
		File: bytes.NewReader([]byte("avatar-B")), ContentType: "image/jpeg",
		Owner: OwnerAgents, OwnerID: agentID, Facet: FacetAvatar,
	})
	require.NoError(t, err)

	// The prior binary is REMOVED from the object store (file gone) — the R2-equivalent
	// DeleteObject ran via ReclaimByKey. Only the replacement remains on disk.
	_, err = os.Stat(firstPath)
	assert.True(t, os.IsNotExist(err), "prior avatar binary must be REMOVED from the object store on replace (R2 object gone)")
	_, err = os.Stat(filepath.Join(dir, "objects", second.Key))
	require.NoError(t, err, "replacement avatar binary must remain on disk")

	// And the prior metadata row is reclaimed too.
	_, err = store.Get(ctx, first.ID)
	assert.ErrorIs(t, err, ErrImageNotFound, "prior avatar metadata row must be reclaimed on replace")
}

func TestUploadReplaceSlotReclaimFailureDoesNotFailUpload(t *testing.T) {
	// The reclaim is best-effort: even when the prior-slot enumeration outright fails,
	// the upload must still return the new image (the new image is stored BEFORE the
	// reclaim runs; a reclaim hiccup is swallowed). listErrorBackend simulates the
	// failure by erroring on ListBySlot.
	objects := NewMemoryObjectStore("https://media.example.com")
	be := listErrorBackend{NewMemoryBackend()}
	policy := fakeReplacePolicy{single: map[string]bool{"agents|avatar": true}}
	store := NewStore(be, objects).WithReplacePolicy(policy)
	ctx := context.Background()

	img, err := store.Upload(ctx, UploadInput{
		File: bytes.NewReader([]byte("avatar")), ContentType: "image/jpeg",
		Owner: OwnerAgents, OwnerID: "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", Facet: FacetAvatar,
	})
	require.NoError(t, err, "upload must succeed even when reclaim cannot enumerate priors")
	assert.True(t, objects.Has(img.Key), "new image must be stored regardless of reclaim outcome")
}

// =============================================================================
// TEST HELPERS
// =============================================================================

func newTestStore(dir string) *Store {
	metaDir := filepath.Join(dir, "meta")
	objDir := filepath.Join(dir, "objects")
	backend := NewFileBackend(metaDir)
	objects := NewFileObjectStore(objDir, "http://localhost:5000/images")
	return NewStore(backend, objects)
}
