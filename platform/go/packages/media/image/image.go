// Package image provides document-type image management with pluggable storage.
//
// Images are first-class document entities with independent identity (UUID),
// own CRUD lifecycle, and dedicated binary storage. Metadata lives in a Backend
// (filesystem or database), binary data lives in an ObjectStore (R2/S3 or filesystem).
//
// The package follows the algebra-interpreter pattern (Decision #0032):
//   - Backend interface = metadata storage algebra
//   - ObjectStore interface = binary storage algebra
//   - FileBackend, MemoryBackend, PostgresBackend = interpreters
//
// Object keys are owner-scoped and hierarchical (Decision #0271,
// object-storage-key-governance OSK-001/002):
//
//	{owner}/{ownerId}/{facet}/{objectId}.{ext}   // entity-owned assets
//	system/{facet}/{objectId}.{ext}              // ownerless brand assets
//
// Because the public URL is literally "https://{cdn}/{key}", a correct key IS a
// correct URL. There is NO flat "{purpose}/{uuid}" scheme and NO "general/"
// catch-all — an unowned object is a domain-modeling gap, not a storage category.
//
// Example (filesystem-only for dev):
//
//	store := image.NewStore(
//	    image.NewFileBackend("/data/images"),
//	    image.NewFileObjectStore("/data/images/objects", "http://localhost:5000"),
//	)
//	img, _ := store.Upload(ctx, image.UploadInput{
//	    File:        file,
//	    ContentType: "image/jpeg",
//	    Owner:       image.OwnerProperties,
//	    OwnerID:     propertyID,        // entity UUID
//	    Facet:       image.FacetCover,
//	})
package image

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// NAMED TYPES — every discriminator is a named type, never a raw string
// =============================================================================

// ImageFormat is the canonical post-decode binary format of the stored image.
// Values match the image.format dictionary (namespace: image.format).
type ImageFormat string

const (
	ImageFormatJPG  ImageFormat = "jpg"
	ImageFormatPNG  ImageFormat = "png"
	ImageFormatWebP ImageFormat = "webp"
	ImageFormatAVIF ImageFormat = "avif"
	// ImageFormatSVG is a VECTOR passthrough format — it is deliberately NOT in
	// validFormats, so it can never enter the raster Upload path (avatars/property
	// images stay raster-only). It exists only as the key extension for a scoped,
	// already-sanitized system brand asset stored DIRECTLY via Store.PutObject (the
	// binary primitive that bypasses the raster validFormats gate). SVG carries no
	// post-decode raster format and is never color-analyzed or cdn-cgi resized.
	ImageFormatSVG ImageFormat = "svg"
)

// ImageOwner is the owning entity's collection name (plural, matching the
// table/route — properties, agents, contacts). It is the first segment of the
// owner-scoped object key (Decision #0271, OSK-001). OwnerSystem is the reserved
// namespace for ownerless brand assets (OSK-002).
//
// An ImageOwner is a lowercase, hyphenated slug (e.g. "properties", "agents",
// "districts") — validated by isValidOwner so it is always a safe key segment.
type ImageOwner string

const (
	// OwnerProperties is the collection name for property-owned assets.
	OwnerProperties ImageOwner = "properties"
	// OwnerAgents is the collection name for agent-owned assets.
	OwnerAgents ImageOwner = "agents"
	// OwnerContacts is the collection name for contact-owned assets (the contact
	// photo, contacts.photo_id). Historically "landlords" — the owner namespace
	// (2026-05-25, #0271) predated the contacts table (2026-05-27); the rename was
	// decided under #0294 (contact roles are derived autotags, never an owner) but
	// never executed until 2607-012. A contact photo is a single-storage slot.
	OwnerContacts ImageOwner = "contacts"
	// OwnerPages is the collection name for CMS-page-owned assets (the per-page hero,
	// cms_pages cover). One hero per page, isolated by ownerId = cms_pages.id — added
	// 2607-012 so per-page heroes don't share the system/hero singleton prefix.
	OwnerPages ImageOwner = "pages"
	// OwnerSystem is the reserved namespace for ownerless brand/system assets
	// (e.g. the watermark logo). System assets have NO ownerId segment:
	// system/{facet}/{objectId}.{ext}.
	OwnerSystem ImageOwner = "system"
)

// ImageFacet is the role of an asset within its owner (cover, gallery, photo,
// qr, hero, watermark). It is the {facet} segment of the owner-scoped object key
// and the value stored in the images.purpose column. Values match the
// image.purpose dictionary (namespace: image.purpose).
//
// Naming (Decision #0271): the column/dictionary is historically called
// "purpose"; conceptually each value is a FACET within an owner prefix. The
// ImagePurpose alias preserves the column-name vocabulary for the storage layer.
type ImageFacet string

// ImagePurpose is a backward-compatible alias for ImageFacet — the value stored
// in the images.purpose column. New code SHOULD use ImageFacet; the alias keeps
// the persistence-layer "purpose" vocabulary and existing callers compiling.
type ImagePurpose = ImageFacet

const (
	FacetCover     ImageFacet = "cover"
	FacetGallery   ImageFacet = "gallery"
	FacetAvatar    ImageFacet = "avatar"
	FacetPhoto     ImageFacet = "photo"
	FacetQR        ImageFacet = "qr"
	FacetHero      ImageFacet = "hero"
	FacetWatermark ImageFacet = "watermark"
	// FacetLogo is the displayed square brand mark stored under the reserved
	// system/logo/ prefix (OSK-002, ownerless brand asset). DISTINCT from
	// FacetWatermark (an overlay consumed only by the watermark pipeline, never
	// displayed): FacetLogo is the logo shown in the public header/footer, the admin
	// sidebar badge, and SEO/OG. ≤512px master; PNG/SVG passthrough else WebP.
	FacetLogo ImageFacet = "logo"
	// FacetFavicon is the purpose-built, guaranteed-PNG favicon SET baked from the
	// current logo (FacetLogo) on every logo/tint change (task 2607-057) — a 64px tab
	// icon + a 180px apple-touch icon — stored under the reserved system/favicon/
	// prefix (OSK-002, ownerless brand asset). DISTINCT from FacetLogo: the logo is the
	// displayed brand mark; the favicon is the square, letterboxed, cross-device tab/tile
	// raster derived from it (a fetched icon can never run CSS, so it must be pre-baked).
	FacetFavicon ImageFacet = "favicon"
)

// Deprecated aliases — kept so existing callers compile during migration.
// Prefer the Facet* constants. PurposeGeneral is REMOVED (Decision #0271,
// OSK-002): there is no "general" catch-all facet.
const (
	PurposeCover     = FacetCover
	PurposeGallery   = FacetGallery
	PurposeAvatar    = FacetAvatar
	PurposeHero      = FacetHero
	PurposeWatermark = FacetWatermark
)

// validFormats is the set of accepted upload formats.
var validFormats = map[ImageFormat]bool{
	ImageFormatJPG:  true,
	ImageFormatPNG:  true,
	ImageFormatWebP: true,
	ImageFormatAVIF: true,
}

// =============================================================================
// ERRORS
// =============================================================================

var (
	ErrImageNotFound = errors.New("image not found")
	ErrInvalidID     = errors.New("invalid image ID")
	ErrInvalidFormat = errors.New("unsupported image format")
	ErrFileTooLarge  = errors.New("file exceeds maximum size")
	ErrUploadFailed  = errors.New("upload failed")
	ErrBackendError  = errors.New("storage backend error")

	// ErrMissingOwner is returned when an upload omits the owner (or owner id for
	// a non-system owner). Owner-scoped keys are mandatory (Decision #0271,
	// OSK-001/002) — an unowned object is a modeling gap, not a storage category.
	ErrMissingOwner = errors.New("owner and owner ID are required (system assets use Owner=system with no owner ID)")
	// ErrInvalidOwner is returned when the owner is not a valid key segment
	// (lowercase letters, digits, hyphens).
	ErrInvalidOwner = errors.New("invalid owner: must be a lowercase hyphenated collection name")
	// ErrMissingFacet is returned when an upload omits the facet (cover, gallery,
	// photo, qr, …). The facet is a required key segment.
	ErrMissingFacet = errors.New("facet is required")
	// ErrInvalidFacet is returned when the facet is not a safe key segment
	// (lowercase letters, digits, hyphens — no slashes, dots, or path traversal).
	ErrInvalidFacet = errors.New("invalid facet: must be a lowercase hyphenated segment")
	// ErrInvalidObjectID is returned when the object ID is not a valid UUID, so it
	// can never escape its key segment or forge a path.
	ErrInvalidObjectID = errors.New("invalid object ID: must be a UUID")
)

// =============================================================================
// DOCUMENT TYPE
// =============================================================================

// Image represents image metadata with a reference to binary data in object storage.
type Image struct {
	ID  string `json:"id"  yaml:"id"`
	Key string `json:"key" yaml:"key"` // R2/S3 object key
	URL string `json:"url" yaml:"url"` // CDN URL — permanent

	Format  ImageFormat `json:"format"   yaml:"format"`
	Size    int64       `json:"size"     yaml:"size"`    // bytes
	AltText string      `json:"alt_text" yaml:"alt_text"`
	// Purpose is the asset facet (cover, gallery, photo, qr, …) — the {facet}
	// segment of the key and the value stored in images.purpose. See ImageFacet.
	Purpose ImageFacet `json:"purpose" yaml:"purpose"`

	// Dominant hex color from grid-sampling analysis (e.g. "#1a1a2e").
	// Empty until analysis completes. Used for placeholder/blur-up backgrounds.
	PrimaryColor string `json:"primary_color" yaml:"primary_color"`

	CreatedBy string     `json:"created_by,omitempty" yaml:"created_by,omitempty"`
	CreatedAt time.Time  `json:"created_at"           yaml:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"           yaml:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" yaml:"deleted_at,omitempty"`
}

// =============================================================================
// INPUT / OPTION TYPES
// =============================================================================

// UploadInput is the input for uploading a new image.
//
// Owner-scoped keys are mandatory (Decision #0271). The caller MUST supply:
//   - Owner   — owning entity collection (OwnerProperties, OwnerAgents, …) or
//     OwnerSystem for ownerless brand assets.
//   - OwnerID — the owning entity UUID. REQUIRED for every owner except
//     OwnerSystem (system assets have no owner-id segment).
//   - Facet   — the asset role within the owner (FacetCover, FacetGallery,
//     FacetPhoto, FacetQR, FacetWatermark, …).
//
// The resulting key is {owner}/{ownerId}/{facet}/{objectId}.{ext}, or
// system/{facet}/{objectId}.{ext} for OwnerSystem.
type UploadInput struct {
	File        io.Reader
	ContentType string // MIME type: image/jpeg, image/png, etc.
	AltText     string

	Owner   ImageOwner // owning entity collection (required)
	OwnerID string     // owning entity UUID (required unless Owner == OwnerSystem)
	Facet   ImageFacet // asset role within the owner (required)

	// Process, when non-nil, makes the SERVER the authoritative normalizer
	// (task 2607-008): the input is decoded → resized to the facet cap →
	// re-encoded as WebP BEFORE storage, so a misbehaving client (iPad/Safari)
	// can never store a non-WebP or oversized original. Build it via
	// FacetProcessOptions(facet). Nil = today's passthrough (the caller's bytes
	// stored verbatim — used by SVG/system assets and alpha-sensitive facets
	// until they opt in). The stored object always ends up WebP when set.
	Process *ProcessOptions

	CreatedBy string
}

// ListOpts filters and paginates image listings.
type ListOpts struct {
	Facet  ImageFacet // filter by facet (images.purpose); empty = all
	Limit  int
	Offset int
}

// =============================================================================
// BACKEND INTERFACE — metadata storage algebra
// =============================================================================

// Backend is the interface for image metadata storage.
type Backend interface {
	Create(ctx context.Context, img *Image) error
	Get(ctx context.Context, id string) (*Image, error)
	Update(ctx context.Context, img *Image) error
	Delete(ctx context.Context, id string) error
	// DeleteByKey soft-deletes the metadata row matching the given object key,
	// if any. Idempotent + best-effort: a missing row is a no-op. The metadata
	// counterpart of Store.ReclaimByKey (consumer-driven replace-reclaim).
	DeleteByKey(ctx context.Context, key string) error
	// ListBySlot returns the non-deleted images stored under the owner-scoped slot
	// prefix {owner}/{ownerId}/{facet}/ (or system/{facet}/) — i.e. every image
	// Upload placed in that one slot. Used by Store to reclaim the prior image(s)
	// when a replacement lands in a single-storage slot. The prefix is the package's
	// own key scheme (slotPrefix, derived from buildKey), so this needs no
	// owner/owner_id columns — it matches on the key.
	ListBySlot(ctx context.Context, owner ImageOwner, ownerID string, facet ImageFacet) ([]Image, error)
	List(ctx context.Context, opts ListOpts) ([]Image, error)
}

// =============================================================================
// OBJECT STORE INTERFACE — binary storage algebra
// =============================================================================

// ObjectStore is the interface for binary image data storage (R2/S3/filesystem).
type ObjectStore interface {
	Upload(ctx context.Context, key string, data io.Reader, contentType string) error
	Download(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
	URL(key string) string
}

// =============================================================================
// REPLACE POLICY — the cardinality protocol (consumer-provided)
// =============================================================================

// ReplacePolicy reports whether an upload's (owner, facet) targets a SINGLE-storage
// slot — one image at a time, so a new upload supersedes (and the package reclaims)
// the prior — or a MULTI pool (append; keep all). It is the ONLY carrier of business
// cardinality across the boundary: pkg/image never hardcodes which facets are single,
// because that is consumer knowledge (an agent has one avatar; a property has many
// gallery photos; a system has one logo). The package defines the protocol; the
// consumer implements it. This is the adapter/protocol pattern (governance #9):
// "single image always replaces" becomes a property of the package, declared once in
// config, inherited by every consumer with zero per-site wiring.
//
// A nil policy means append-only — today's behavior — so existing Store callers that
// do not plug a policy in are unchanged (zero behavior change, no opt-in required).
type ReplacePolicy interface {
	// IsReplaceSlot reports whether (owner, facet) holds a single image that a new
	// upload supersedes. ownerID is intentionally NOT part of the decision: every
	// owner of a given kind has the same per-facet cardinality (every agent has one
	// avatar). The ownerID only scopes WHICH slot's prior is reclaimed at upload time.
	IsReplaceSlot(owner ImageOwner, facet ImageFacet) bool
}

// =============================================================================
// STORE — high-level operations combining Backend + ObjectStore
// =============================================================================

// Store manages images using metadata Backend and binary ObjectStore.
//
// When a ReplacePolicy is set (via WithReplacePolicy), Upload additionally reclaims
// the prior image(s) in a single-storage slot the moment a replacement lands — the
// package's centralized, config-driven replace-reclaim. With no policy the store is
// append-only (today's behavior).
type Store struct {
	backend Backend
	objects ObjectStore
	policy  ReplacePolicy // nil = append-only (default; no replace-reclaim)
}

// NewStore creates a new image store. The store is append-only until a ReplacePolicy
// is plugged in via WithReplacePolicy.
func NewStore(backend Backend, objects ObjectStore) *Store {
	return &Store{backend: backend, objects: objects}
}

// WithReplacePolicy returns a Store that auto-reclaims the prior single-storage image
// when a replacement is uploaded to a slot the policy marks as single (one image at a
// time). It is the SINGLE opt-in: a consumer declares its cardinality once (a map of
// single slots), and every upload to those slots reclaims the prior automatically —
// no handler, repo, or frontend touches cleanup. Nil/zero-value policies leave the
// store append-only. Returns the receiver so it chains at construction
// (NewStore(b, o).WithReplacePolicy(p)).
func (s *Store) WithReplacePolicy(p ReplacePolicy) *Store {
	s.policy = p
	return s
}

// Upload uploads binary data to object store, runs color analysis, then creates
// the metadata record. Color analysis failure is non-fatal: upload succeeds with
// PrimaryColor empty.
//
// When input.Process is set (task 2607-008), the input is decoded → resized →
// re-encoded as WebP BEFORE the storage path runs, so the server is the
// authoritative normalizer (the stored object is always WebP ≤ cap regardless of
// what the client sent). This swaps input.File + ContentType for the normalized
// WebP bytes; the rest of the function is unchanged. Nil Process = passthrough.
func (s *Store) Upload(ctx context.Context, input UploadInput) (*Image, error) {
	if input.Process != nil {
		raw, err := io.ReadAll(input.File)
		if err != nil {
			return nil, fmt.Errorf("read upload: %w", err)
		}
		webpData, _, _, err := Process(raw, *input.Process)
		if err != nil {
			return nil, fmt.Errorf("normalize: %w", err)
		}
		input.File = bytes.NewReader(webpData)
		input.ContentType = "image/webp"
	}

	format := formatFromContentType(input.ContentType)
	if !validFormats[format] {
		return nil, ErrInvalidFormat
	}

	if err := input.validate(); err != nil {
		return nil, err
	}
	facet := input.Facet

	id := uuid.New().String()
	key, err := buildKey(input.Owner, input.OwnerID, facet, id, format)
	if err != nil {
		return nil, err
	}

	// TeeReader buffers bytes for color analysis while uploading to object store.
	var colorBuf bytes.Buffer
	tee := io.TeeReader(input.File, &colorBuf)

	if err := s.objects.Upload(ctx, key, tee, input.ContentType); err != nil {
		return nil, errors.Join(ErrUploadFailed, err)
	}

	size := int64(colorBuf.Len())

	primaryColor, _ := AnalyzePrimaryColor(&colorBuf, format)

	now := time.Now().UTC()
	img := &Image{
		ID:           id,
		Key:          key,
		URL:          s.objects.URL(key),
		Format:       format,
		Size:         size,
		AltText:      input.AltText,
		Purpose:      facet,
		PrimaryColor: primaryColor,
		CreatedAt:    now,
		UpdatedAt:    now,
		CreatedBy:    input.CreatedBy,
	}

	if err := s.backend.Create(ctx, img); err != nil {
		_ = s.objects.Delete(ctx, key)
		return nil, err
	}

	// Replace-reclaim (single-storage slots): if a policy marks this (owner, facet)
	// as a one-image-at-a-time slot, every PRIOR image in that slot is now superseded
	// garbage — reclaim it (binary + row). Best-effort: the new image is already live
	// (object + row stored above), so a reclaim hiccup is swallowed and never fails
	// the upload; at worst an orphan sits until the next replace. The just-stored image
	// is skipped (it shares the slot prefix). Nil policy = append-only (no reclaim),
	// i.e. today's behavior.
	if s.policy != nil && s.policy.IsReplaceSlot(input.Owner, facet) {
		s.reclaimSlotPriors(ctx, input.Owner, input.OwnerID, facet, img.ID)
	}

	return img, nil
}

// reclaimSlotPriors reclaims every non-deleted image in the (owner, ownerId, facet)
// slot except the one with keepID (the replacement just stored). Best-effort + silent:
// each reclaim error is swallowed (the new image is already live; a leftover orphan is
// the only consequence). Used by Store.Upload for single-storage replace-reclaim.
func (s *Store) reclaimSlotPriors(ctx context.Context, owner ImageOwner, ownerID string, facet ImageFacet, keepID string) {
	priors, err := s.backend.ListBySlot(ctx, owner, ownerID, facet)
	if err != nil {
		return // best-effort: can't enumerate → leave priors (rare; same as today)
	}
	for i := range priors {
		if priors[i].ID == keepID {
			continue // keep the just-stored replacement
		}
		_ = s.ReclaimByKey(ctx, priors[i].Key) // binary (swallowed) + row (DeleteByKey)
	}
}

// PutObject writes a binary object DIRECTLY to the ObjectStore under the given
// key and returns its permanent CDN URL — WITHOUT creating an image metadata row.
//
// It is the thin "binary only" primitive for system-owned brand assets that are
// NOT entries in the authoring image library (e.g. the published watermark overlay
// PNG, system/watermarks/<uuid>.png — Decision #0271). A normal cover/gallery
// upload goes through Upload, which ADDITIONALLY records an images row (so it shows
// in GET /api/images?facet=…). PutObject deliberately omits that row: a watermark
// overlay must never pollute the facet=watermark authoring library.
//
// The caller owns the key (it must already be a valid object key — see buildKey /
// the system/{facet}/{id}.{ext} scheme). PutObject reuses the SAME ObjectStore put
// + URL derivation Upload uses, so the resulting URL is identical to what an Upload
// of the same key would yield.
func (s *Store) PutObject(ctx context.Context, key string, data io.Reader, contentType string) (string, error) {
	if key == "" {
		return "", ErrUploadFailed
	}
	if err := s.objects.Upload(ctx, key, data, contentType); err != nil {
		return "", errors.Join(ErrUploadFailed, err)
	}
	return s.objects.URL(key), nil
}

// GetObject reads a binary object DIRECTLY from the ObjectStore by its key,
// returning a ReadCloser the caller MUST close — WITHOUT touching the image
// metadata table.
//
// It is the read counterpart of PutObject: the "binary only" primitive a derived-
// variant pipeline uses to fetch a source binary by R2 key (e.g. the watermark apply
// worker downloading an original gallery photo to bake, or the published mark-unit
// PNG to re-tile — Slice C). A normal metadata lookup goes through Get (by id); this
// fetches bytes by key, so it works for keys that have NO images row (system assets).
func (s *Store) GetObject(ctx context.Context, key string) (io.ReadCloser, error) {
	if key == "" {
		return nil, ErrInvalidID
	}
	return s.objects.Download(ctx, key)
}

// DeleteObject removes a binary object DIRECTLY from the ObjectStore by its key,
// WITHOUT touching the image metadata table — the delete counterpart of PutObject /
// GetObject. Unlike Store.Delete (which soft-deletes an images ROW and deliberately
// leaves the binary), this is the primitive for reclaiming a key-addressed system
// asset that has NO images row: e.g. superseding a baked logo-tint raster on re-bake.
// Deleting an already-absent key is idempotent (the ObjectStore adapters treat it as a
// no-op success), so a best-effort cleanup never fails on a missing object.
func (s *Store) DeleteObject(ctx context.Context, key string) error {
	if key == "" {
		return ErrInvalidID
	}
	return s.objects.Delete(ctx, key)
}

// Get retrieves image metadata by ID.
func (s *Store) Get(ctx context.Context, id string) (*Image, error) {
	if id == "" {
		return nil, ErrInvalidID
	}
	return s.backend.Get(ctx, id)
}

// Delete soft-deletes an image (marks deleted_at, does NOT remove binary from object store).
func (s *Store) Delete(ctx context.Context, id string) error {
	if id == "" {
		return ErrInvalidID
	}
	return s.backend.Delete(ctx, id)
}

// ReclaimByKey frees BOTH the binary object AND its metadata row for the given
// key — the reclaim path a consumer uses when it replaces a single-owner image
// (logo / avatar / hero / contact photo / content cover) and the previous asset
// must be gone. Distinct from Delete (row-only soft-delete, binary kept) and
// DeleteObject (binary-only, for key-addressed system assets with no row):
// ReclaimByKey does both, best-effort + idempotent — a missing binary or row is
// a no-op, so re-reclaim never fails. The caller MUST have validated the key is
// a managed asset (the managedObjectKey SSRF guard on the API layer) first.
func (s *Store) ReclaimByKey(ctx context.Context, key string) error {
	if key == "" {
		return ErrInvalidID
	}
	// Binary first (idempotent), then the metadata row (best-effort). A row
	// pointing at a freed binary is useless, so reclaim drops both. A binary
	// delete error is swallowed: the object may already be gone (idempotent
	// re-reclaim) and the row still needs dropping either way.
	_ = s.objects.Delete(ctx, key)
	return s.backend.DeleteByKey(ctx, key)
}

// List returns images matching the given options.
func (s *Store) List(ctx context.Context, opts ListOpts) ([]Image, error) {
	return s.backend.List(ctx, opts)
}

// UpdateAltText updates the alt text for an image.
func (s *Store) UpdateAltText(ctx context.Context, id, altText string) error {
	img, err := s.backend.Get(ctx, id)
	if err != nil {
		return err
	}
	img.AltText = altText
	img.UpdatedAt = time.Now().UTC()
	return s.backend.Update(ctx, img)
}

// =============================================================================
// HELPERS
// =============================================================================

func formatFromContentType(ct string) ImageFormat {
	switch ct {
	case "image/jpeg":
		return ImageFormatJPG
	case "image/png":
		return ImageFormatPNG
	case "image/webp":
		return ImageFormatWebP
	case "image/avif":
		return ImageFormatAVIF
	default:
		return ""
	}
}

// validate enforces the owner-scoped key contract (Decision #0271, OSK-001/002):
// a valid owner, an owner ID for every owner except OwnerSystem, and a facet.
func (in UploadInput) validate() error {
	if in.Owner == "" {
		return ErrMissingOwner
	}
	if !isValidOwner(in.Owner) {
		return ErrInvalidOwner
	}
	if in.Facet == "" {
		return ErrMissingFacet
	}
	if !isValidSegment(string(in.Facet)) {
		return ErrInvalidFacet
	}
	// Every owner except the reserved "system" namespace requires an owner ID.
	if in.Owner != OwnerSystem && in.OwnerID == "" {
		return ErrMissingOwner
	}
	return nil
}

// slotPrefix is the owner-scoped key prefix every image in a single (owner, ownerId,
// facet) slot shares, MINUS the per-image uuid+ext:
//
//	{owner}/{ownerId}/{facet}/      // entity-owned
//	system/{facet}/                 // ownerless brand asset (no owner-id segment)
//
// It is derived from the SAME scheme buildKey uses (Decision #0271), so a slot lookup
// by this prefix always matches exactly the images Upload stored for that slot — and
// ONLY those: the trailing slash means a sibling facet (e.g. "avatar" vs "avatar-extra")
// can never bleed in. Used by Backend.ListBySlot to find the prior image(s) a
// replacement supersedes. No validation: the inputs are already validated upstream at
// upload time (buildKey), and this prefix is only ever derived from a valid upload.
func slotPrefix(owner ImageOwner, ownerID string, facet ImageFacet) string {
	if owner == OwnerSystem {
		return fmt.Sprintf("%s/%s/", owner, facet)
	}
	return fmt.Sprintf("%s/%s/%s/", owner, ownerID, facet)
}

// SystemObjectKey builds the ownerless system-asset object key
// system/{facet}/{objectId}.{ext} (Decision #0271, OSK-002) for a direct
// Store.PutObject caller that bypasses the raster Upload path — e.g. a scoped,
// already-sanitized SVG brand logo whose ImageFormatSVG is intentionally outside
// validFormats. It REUSES buildKey (OwnerSystem, empty owner ID), so the segment
// path-hardening (facet is a safe slug, objectID is a UUID) and the exact
// system/{facet}/{id}.{ext} shape are identical to what Store.Upload would derive —
// the key is never hand-rolled divergently. objectID MUST be a server-minted UUID.
func SystemObjectKey(facet ImageFacet, objectID string, format ImageFormat) (string, error) {
	return buildKey(OwnerSystem, "", facet, objectID, format)
}

// buildKey assembles the owner-scoped object key (Decision #0271, OSK-001/002):
//
//	{owner}/{ownerId}/{facet}/{objectId}.{ext}   // entity-owned
//	system/{facet}/{objectId}.{ext}              // OwnerSystem (no owner-id segment)
//
// It validates the same invariants as UploadInput.validate AND path-hardens every
// interpolated segment — owner + facet are validated as safe slugs and objectID as
// a UUID — so it is safe to call directly (the upcoming agent QR/photo callers will
// use it). No segment can escape its position with a slash, dot, or "..". There is
// no flat "{purpose}/{uuid}" form and no "general/" catch-all.
func buildKey(owner ImageOwner, ownerID string, facet ImageFacet, objectID string, format ImageFormat) (string, error) {
	if owner == "" {
		return "", ErrMissingOwner
	}
	if !isValidOwner(owner) {
		return "", ErrInvalidOwner
	}
	if facet == "" {
		return "", ErrMissingFacet
	}
	if !isValidSegment(string(facet)) {
		return "", ErrInvalidFacet
	}
	if objectID == "" {
		return "", ErrInvalidID
	}
	// objectID is always a server-minted UUID; reject anything else so it can never
	// forge a path segment or extension.
	if _, err := uuid.Parse(objectID); err != nil {
		return "", ErrInvalidObjectID
	}

	if owner == OwnerSystem {
		// Ownerless brand asset: system/{facet}/{objectId}.{ext}
		return fmt.Sprintf("%s/%s/%s.%s", owner, facet, objectID, format), nil
	}

	if ownerID == "" {
		return "", ErrMissingOwner
	}
	return fmt.Sprintf("%s/%s/%s/%s.%s", owner, ownerID, facet, objectID, format), nil
}

// isValidOwner reports whether o is a safe owner key segment (see isValidSegment).
func isValidOwner(o ImageOwner) bool {
	return isValidSegment(string(o))
}

// isValidSegment reports whether s is a safe object-key path segment: a non-empty
// lowercase string of letters, digits, and hyphens (e.g. "properties", "agents",
// "districts", "cover", "gallery"), with no leading/trailing hyphen. This keeps
// any interpolated segment (owner, facet) from escaping its position in the key —
// no slashes, dots, or path traversal ("..").
func isValidSegment(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= '0' && r <= '9':
		case r == '-':
		default:
			return false
		}
	}
	// A leading/trailing hyphen is not a valid segment slug.
	return !strings.HasPrefix(s, "-") && !strings.HasSuffix(s, "-")
}
