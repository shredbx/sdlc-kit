package property

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// =============================================================================
// DOCUMENT STORE — companion for the property_documents child table
// =============================================================================
//
// A property's documents are a 1:many cascade-delete child collection
// (property_documents), NOT a column on the properties row, so the generated
// mapper cannot map it (companion-loaded — mirrors property_note /
// image_collection). Each document is a single uploaded file (PDF / plain text /
// image-as-document) stored in R2 under the SHARED server-side object-key scheme
// (properties/{propertyId}/documents/{documentId}.{ext}) — the same scheme the
// image store uses, NOT a new URL scheme; only the resulting URL is kept on the
// row.
//
// Every document belongs to one visibility group (DocumentVisibility): PUBLIC
// documents render on the property's public listing; PRIVATE documents are
// admin-only and never enter the public payload. The two groups are kept strictly
// separate in the read paths (the public projection filters visibility = public).
//
// This store owns ONLY the row (Load / Save / Delete). The R2 upload + the
// delete-the-object-on-row-delete are orchestrated by the handler, which owns the
// image.ObjectStore — the row store has no storage dependency (mirrors how the
// note store has no R2 dependency). DocumentObjectKey is the shared key builder
// used by the handler so both the upload key and any future re-derivation match.

// =============================================================================
// DOCUMENT VISIBILITY — named dictionary type (document-visibility entity)
// =============================================================================

// DocumentVisibility places a property document into one of two strictly
// separated groups: PUBLIC (rendered on the property's public listing) or PRIVATE
// (admin-only, never in the public payload). Modeled as a named type — not a raw
// string and not a boolean — so the Go projection carries no raw strings
// (modeling-standard) and the group set can extend without a schema change. FK to
// the document-visibility dictionary; stored as property_documents.visibility
// VARCHAR(20). Default PRIVATE (safe default — public is an explicit opt-in).
type DocumentVisibility string

const (
	// VisibilityPublic — rendered in the Documents section of the public listing.
	VisibilityPublic DocumentVisibility = "public"
	// VisibilityPrivate — admin-only; never reaches the public payload. The default.
	VisibilityPrivate DocumentVisibility = "private"
)

// Valid reports whether v is one of the seeded visibility codes — a parse-time
// guard on the API boundary; the column default + the dictionary enforce the same
// rule at the DB layer.
func (v DocumentVisibility) Valid() bool {
	switch v {
	case VisibilityPublic, VisibilityPrivate:
		return true
	}
	return false
}

// ParseDocumentVisibility normalizes a raw visibility string to a DocumentVisibility,
// defaulting a BLANK value to PRIVATE (the safe default — public is an explicit
// opt-in). A non-blank value that is not a seeded code is rejected with
// ErrInvalidVisibility, so the caller maps it to a 4xx. The input is trimmed and
// lower-cased before matching.
func ParseDocumentVisibility(raw string) (DocumentVisibility, error) {
	clean := strings.ToLower(strings.TrimSpace(raw))
	if clean == "" {
		return VisibilityPrivate, nil
	}
	v := DocumentVisibility(clean)
	if !v.Valid() {
		return "", ErrInvalidVisibility
	}
	return v, nil
}

// =============================================================================
// DOCUMENT — 1:many containment child (Decision #0019)
// =============================================================================

// MaxDocumentBytes is the governed size cap for a single uploaded document
// (project_image_size_policy → document entry: 25 MB). A larger upload is rejected
// at the API boundary (NFR-001) before any object/row is written.
const MaxDocumentBytes int64 = 25 << 20 // 25 MiB

// MaxDocumentTitleLen is the governed cap on a document title (entity.yml
// constraints.maxLength = 120). Titles are trimmed before length is measured.
const MaxDocumentTitleLen = 120

// MaxDocumentFileNameLen is the governed cap on the stored file name (entity.yml
// constraints.maxLength = 255). Trimmed before length is measured.
const MaxDocumentFileNameLen = 255

// MaxDocumentMimeTypeLen is the governed cap on the stored mime type (entity.yml
// constraints.maxLength = 120). In practice mime_type is always the server-derived
// normalized type (a bounded set), but the guard keeps the Go check in lockstep
// with the YAML constraint (YAML→code alignment).
const MaxDocumentMimeTypeLen = 120

// allowedDocumentMIME is the boundary MIME allowlist (entity governance): PDF,
// plain text, and the three common web image formats. An upload whose detected
// content type is not in this set is rejected at the boundary — no object, no row.
var allowedDocumentMIME = map[string]struct{}{
	"application/pdf": {},
	"text/plain":      {},
	"image/jpeg":      {},
	"image/png":       {},
	"image/webp":      {},
}

// ErrInvalidVisibility indicates a supplied visibility value is neither public nor
// private. Test with errors.Is(err, property.ErrInvalidVisibility).
var ErrInvalidVisibility = errors.New("document visibility must be public or private")

// ErrDocumentMIMENotAllowed indicates an upload's MIME type is outside the
// boundary allowlist (PDF / plain text / jpeg / png / webp).
var ErrDocumentMIMENotAllowed = errors.New("document type not allowed (pdf, plain text, jpeg, png or webp only)")

// ErrDocumentTooLarge indicates an upload exceeded MaxDocumentBytes.
var ErrDocumentTooLarge = errors.New("document exceeds the 25MB size cap")

// ErrDocumentFileNameRequired indicates a document was saved without a file name
// (the original upload name is required — it is the download name and the display
// fallback).
var ErrDocumentFileNameRequired = errors.New("document file name is required")

// ErrDocumentTitleTooLong indicates a document title exceeded MaxDocumentTitleLen.
var ErrDocumentTitleTooLong = errors.New("document title must be at most 120 characters")

// ErrDocumentFileNameTooLong indicates a file name exceeded MaxDocumentFileNameLen.
var ErrDocumentFileNameTooLong = errors.New("document file name must be at most 255 characters")

// ErrDocumentMimeTypeTooLong indicates a mime type exceeded MaxDocumentMimeTypeLen.
var ErrDocumentMimeTypeTooLong = errors.New("document mime type must be at most 120 characters")

// PropertyDocument is a single typed file attached to a property — a PDF, plain
// text, or image-as-document — placed in one visibility group. It is a containment
// child (parent = property): own UUID identity, 1:many, cascade-deletes with the
// parent (FK ON DELETE CASCADE) — mirroring ImageCollection / Note, NOT an
// attribute-group. The binary lives in R2 under the shared object-key scheme; only
// URL is kept on the row. Companion-loaded (kept OUT of the generated mapper).
type PropertyDocument struct {
	ID         string `json:"id" yaml:"id"`
	PropertyID string `json:"property_id" yaml:"property_id"`

	// Title is the display label (optional, <=120); falls back to FileName when blank.
	Title string `json:"title" yaml:"title"`
	// FileName is the original uploaded file name (required) — the download name and
	// the display fallback.
	FileName string `json:"file_name" yaml:"file_name"`
	// MimeType is the boundary-validated MIME (pdf / plain text / jpeg / png / webp);
	// drives preview-vs-download selection on the client.
	MimeType string `json:"mime_type" yaml:"mime_type"`
	// SizeBytes is the stored file size, boundary-validated against MaxDocumentBytes.
	SizeBytes int64 `json:"size_bytes" yaml:"size_bytes"`
	// URL is the stored R2 object URL (built server-side via the shared key scheme).
	URL string `json:"url" yaml:"url"`
	// Visibility is PUBLIC (public listing) or PRIVATE (admin-only). Default private.
	Visibility DocumentVisibility `json:"visibility" yaml:"visibility"`
	// SortOrder is the ascending display order within this document's visibility group.
	SortOrder int `json:"sort_order" yaml:"sort_order"`
	// ShareToken is the raw bearer token for anyone-with-link sharing (scope-07):
	// non-nil = link sharing ON (the token IS the public share secret); nil = off.
	// Stored raw (redistributable, re-displayable — Drive UX), never hashed. Omitted
	// from the wire when nil so an unshared document's payload never carries the key.
	ShareToken *string `json:"share_token,omitempty" yaml:"share_token,omitempty"`

	CreatedAt time.Time `json:"created_at" yaml:"created_at"`
}

// Validate enforces the per-document invariants (PD function validate-document):
// a non-empty file name, a length-bounded title, a valid (defaulted) visibility,
// and a size within the cap. MIME is validated separately at the boundary against
// the detected content type (ValidateDocumentUpload) since the detected type — not
// a client claim — is the authority. A nil receiver is valid (no document).
func (d *PropertyDocument) Validate() error {
	if d == nil {
		return nil
	}
	if strings.TrimSpace(d.FileName) == "" {
		return ErrDocumentFileNameRequired
	}
	// Caps are CHARACTER counts (entity.yml maxLength), not bytes — measure runes so a
	// multibyte Thai title/file name is not over-rejected at the byte boundary.
	if utf8.RuneCountInString(strings.TrimSpace(d.FileName)) > MaxDocumentFileNameLen {
		return ErrDocumentFileNameTooLong
	}
	if utf8.RuneCountInString(strings.TrimSpace(d.Title)) > MaxDocumentTitleLen {
		return ErrDocumentTitleTooLong
	}
	if utf8.RuneCountInString(strings.TrimSpace(d.MimeType)) > MaxDocumentMimeTypeLen {
		return ErrDocumentMimeTypeTooLong
	}
	// Default a blank visibility to private (safe default) and reject any other
	// non-seeded value.
	if d.Visibility == "" {
		d.Visibility = VisibilityPrivate
	}
	if !d.Visibility.Valid() {
		return ErrInvalidVisibility
	}
	if d.SizeBytes < 0 || d.SizeBytes > MaxDocumentBytes {
		return ErrDocumentTooLarge
	}
	return nil
}

// ValidateDocumentUpload is the BOUNDARY gate run on a fresh multipart upload
// BEFORE any object or row is written (NFR-001). It validates the DETECTED MIME
// against the allowlist and the byte size against the cap — the two checks that
// must fail-closed before storage. The caller passes the server-detected content
// type (never a client claim) and the read byte length. On success the upload is
// safe to store; on failure NO object and NO row are created.
func ValidateDocumentUpload(detectedMIME string, sizeBytes int64) error {
	mime := normalizeMIME(detectedMIME)
	if _, ok := allowedDocumentMIME[mime]; !ok {
		return ErrDocumentMIMENotAllowed
	}
	if sizeBytes <= 0 || sizeBytes > MaxDocumentBytes {
		return ErrDocumentTooLarge
	}
	return nil
}

// normalizeMIME strips any parameter (e.g. "text/plain; charset=utf-8" →
// "text/plain") and lower-cases the media type so the allowlist check matches the
// canonical type regardless of charset/boundary parameters http.DetectContentType
// appends.
func normalizeMIME(ct string) string {
	if i := strings.IndexByte(ct, ';'); i >= 0 {
		ct = ct[:i]
	}
	return strings.ToLower(strings.TrimSpace(ct))
}

// DocumentExt maps a validated MIME type to the object-key file extension used in
// the shared key scheme. Mirrors the image store's format-from-content-type: the
// extension is derived from the SERVER-detected type, never a client-supplied file
// name, so the stored key can never carry a forged extension.
func DocumentExt(mime string) string {
	switch normalizeMIME(mime) {
	case "application/pdf":
		return "pdf"
	case "text/plain":
		return "txt"
	case "image/jpeg":
		return "jpg"
	case "image/png":
		return "png"
	case "image/webp":
		return "webp"
	}
	return "bin"
}

// DocumentObjectKey assembles the shared server-side R2 object key for a property
// document: properties/{propertyId}/documents/{documentId}.{ext} — the SAME
// owner/owner-id/facet/object-id scheme the image store uses (owner=properties,
// facet=documents), NOT a new URL scheme. The extension is derived from the
// validated MIME via DocumentExt. Both ids are server-minted UUIDs.
func DocumentObjectKey(propertyID, documentID, mime string) string {
	return fmt.Sprintf("properties/%s/documents/%s.%s", propertyID, documentID, DocumentExt(mime))
}

// NewDocumentID mints a server-side document UUID — used to build the object key
// BEFORE the upload so the same id keys both the R2 object and the row.
func NewDocumentID() string {
	return uuid.NewString()
}

// shareTokenBytes is the entropy of a document share token (32 bytes → 256 bits →
// unguessable; 43 base64url chars).
const shareTokenBytes = 32

// GenerateShareToken mints a URL-safe bearer token for anyone-with-link document
// sharing (scope-07): 32 crypto/rand bytes base64url-encoded (RawURLEncoding — no
// padding, so it drops straight into a path segment). The RAW token IS the share
// secret (stored raw, redistributable — Drive UX); revoking = NULLing the column,
// re-enabling = a fresh token (old links die). Lives in pkg/property (auth-free) and
// does NOT import pkg/auth — this is a bearer link, not a session credential.
func GenerateShareToken() (string, error) {
	b := make([]byte, shareTokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate share token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// =============================================================================
// PD: ordering + public projection
// =============================================================================

// NormalizeDocumentOrder rewrites the documents' SortOrder to a dense 0..n-1
// sequence in the slice's current order — the canonical per-group ordering used
// when persisting a reorder write. Mirrors NormalizeItemOrder. A nil/empty input
// is returned unchanged.
func NormalizeDocumentOrder(docs []PropertyDocument) []PropertyDocument {
	for i := range docs {
		docs[i].SortOrder = i
	}
	return docs
}

// FilterPublicDocuments returns only the PUBLIC documents from a set, preserving
// input order (which the loader sorts by sort_order). This is the public-projection
// filter — PRIVATE documents must NEVER enter the public payload. A nil/empty input
// yields a non-nil empty slice (stable JSON [], never null).
//
// The share token is stripped from every projected row (scope-10 F8): it is a private
// bearer secret and must never ride the public payload, even on a public row that has
// link-sharing on. The strip is on the projection COPY only — the caller's slice is
// left untouched (by-value projection, matching the belt-and-suspenders call sites).
func FilterPublicDocuments(docs []PropertyDocument) []PropertyDocument {
	public := make([]PropertyDocument, 0, len(docs))
	for i := range docs {
		if docs[i].Visibility == VisibilityPublic {
			d := docs[i]
			d.ShareToken = nil
			public = append(public, d)
		}
	}
	return public
}

// =============================================================================
// DOCUMENT STORE — row CRUD (companion-loaded, no storage dependency)
// =============================================================================

// DocumentStore reads and writes a property's document rows against the
// {schema}.property_documents table. It owns ONLY the row — the R2 object lifecycle
// is the handler's (which holds the image.ObjectStore). Construct with
// NewDocumentStore. Pass a nil pool only in tests that exercise pure helpers.
type DocumentStore struct {
	pool   pgxQuerier
	schema string
}

// NewDocumentStore wires the store to a pool + schema.
func NewDocumentStore(pool pgxQuerier, schema string) DocumentStore {
	return DocumentStore{pool: pool, schema: schema}
}

// Load fetches a property's documents ordered by (visibility, sort_order) — the
// per-group ordered list the admin renders. When publicOnly is true the read is
// filtered to visibility = public (the public-listing projection); PRIVATE
// documents never enter that result. A nil pool (fixture mode) yields a non-nil
// empty slice. Read-only.
func (s DocumentStore) Load(ctx context.Context, propertyID string, publicOnly bool) ([]PropertyDocument, error) {
	docs := []PropertyDocument{}
	if s.pool == nil {
		return docs, nil
	}

	sql := `SELECT id, property_id, COALESCE(title, ''), file_name, mime_type,
	               size_bytes, url, visibility, sort_order, share_token, created_at
	          FROM ` + s.schema + `.property_documents
	         WHERE property_id = $1 AND deleted_at IS NULL`
	if publicOnly {
		sql += ` AND visibility = 'public'`
	}
	sql += ` ORDER BY visibility, sort_order, created_at`

	rows, err := s.pool.Query(ctx, sql, propertyID)
	if err != nil {
		return nil, fmt.Errorf("load property documents: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var d PropertyDocument
		if err := rows.Scan(&d.ID, &d.PropertyID, &d.Title, &d.FileName, &d.MimeType,
			&d.SizeBytes, &d.URL, &d.Visibility, &d.SortOrder, &d.ShareToken, &d.CreatedAt); err != nil {
			return nil, fmt.Errorf("load property documents: scan: %w", err)
		}
		docs = append(docs, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("load property documents: rows: %w", err)
	}
	return docs, nil
}

// Get returns a single document scoped to its property, or nil when absent (a
// missing / wrong-property / soft-deleted row is nil, not an error). Used by the
// handler to 404 cleanly on a wrong property/document pair before a PATCH/DELETE.
// A nil pool yields nil, nil.
func (s DocumentStore) Get(ctx context.Context, propertyID, docID string) (*PropertyDocument, error) {
	if s.pool == nil {
		return nil, nil
	}
	var d PropertyDocument
	err := s.pool.QueryRow(ctx,
		`SELECT id, property_id, COALESCE(title, ''), file_name, mime_type,
		        size_bytes, url, visibility, sort_order, share_token, created_at
		   FROM `+s.schema+`.property_documents
		  WHERE id = $1 AND property_id = $2 AND deleted_at IS NULL`,
		docID, propertyID).Scan(&d.ID, &d.PropertyID, &d.Title, &d.FileName, &d.MimeType,
		&d.SizeBytes, &d.URL, &d.Visibility, &d.SortOrder, &d.ShareToken, &d.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get property document %s: %w", docID, err)
	}
	return &d, nil
}

// Save validates then upserts a single document row — INSERT … ON CONFLICT (id) DO
// UPDATE (title / visibility / sort_order are mutable; file_name / mime_type /
// size_bytes / url are write-once and not updated on conflict). Validation runs
// FIRST (so the gate is exercised in fixture mode); an id is assigned for a new
// document. A nil pool returns the document with its assigned id/timestamps without
// persisting (fixture echo). The caller sets PropertyID + URL (after the R2 upload).
func (s DocumentStore) Save(ctx context.Context, q rowQuerier, d PropertyDocument) (PropertyDocument, error) {
	if err := d.Validate(); err != nil {
		return PropertyDocument{}, fmt.Errorf("%w: %v", ErrValidation, err)
	}
	if d.ID == "" {
		d.ID = NewDocumentID()
	}
	if d.CreatedAt.IsZero() {
		d.CreatedAt = time.Now()
	}
	if s.pool == nil {
		return d, nil
	}
	err := q.QueryRow(ctx,
		`INSERT INTO `+s.schema+`.property_documents
			(id, property_id, title, file_name, mime_type, size_bytes, url, visibility, sort_order, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		 ON CONFLICT (id) DO UPDATE SET
			title      = EXCLUDED.title,
			visibility = EXCLUDED.visibility,
			sort_order = EXCLUDED.sort_order,
			updated_at = now()
		 RETURNING id, property_id, COALESCE(title, ''), file_name, mime_type,
		           size_bytes, url, visibility, sort_order, share_token, created_at`,
		d.ID, d.PropertyID, d.Title, d.FileName, d.MimeType, d.SizeBytes, d.URL,
		d.Visibility, d.SortOrder, d.CreatedAt).
		Scan(&d.ID, &d.PropertyID, &d.Title, &d.FileName, &d.MimeType, &d.SizeBytes,
			&d.URL, &d.Visibility, &d.SortOrder, &d.ShareToken, &d.CreatedAt)
	if err != nil {
		return PropertyDocument{}, fmt.Errorf("save property document: %w", err)
	}
	return d, nil
}

// Delete soft-deletes one document row scoped to its property (sets deleted_at).
// The delete is idempotent — a missing / wrong-property row is a no-op (not an
// error), so a repeated delete returns nil and the handler returns 204 regardless.
// The R2 object delete is the handler's responsibility (best-effort) — this store
// owns only the row. A nil pool is a no-op.
func (s DocumentStore) Delete(ctx context.Context, q execQuerier, propertyID, docID string) error {
	if s.pool == nil {
		return nil
	}
	_, err := q.Exec(ctx,
		`UPDATE `+s.schema+`.property_documents
		    SET deleted_at = now()
		  WHERE id = $1 AND property_id = $2 AND deleted_at IS NULL`,
		docID, propertyID)
	if err != nil {
		return fmt.Errorf("delete property document %s: %w", docID, err)
	}
	return nil
}

// =============================================================================
// DOCUMENT SHARE — anyone-with-link bearer token (scope-07, no expiry)
// =============================================================================
//
// A property document may be shared by a raw bearer token (share_token): present =
// link sharing ON, NULL = off. The token resolves publicly (GetByShareToken → 302 to
// the stored R2 URL) regardless of the document's visibility group — the binary is
// already public-by-URL, so the token adds a REVOCABLE, presentable wrapper, not
// storage secrecy. Uses UPDATE … RETURNING via QueryRow (not Exec) so the store
// leans only on the pgxQuerier surface it already holds (mirrors Get), no extra q.

// EnableShare turns on link sharing for one document scoped to its property: mint a
// FRESH token, store it (replacing any prior token — re-enable resets the link, so
// old links die, matching Drive's "reset link"), and return the updated row. Returns
// nil when the document is absent / wrong-property / soft-deleted (the handler 404s).
// A nil pool echoes a minted token without persisting (fixture parity with Save).
func (s DocumentStore) EnableShare(ctx context.Context, propertyID, docID string) (*PropertyDocument, error) {
	token, err := GenerateShareToken()
	if err != nil {
		return nil, err
	}
	if s.pool == nil {
		return &PropertyDocument{ID: docID, PropertyID: propertyID, ShareToken: &token}, nil
	}
	var d PropertyDocument
	err = s.pool.QueryRow(ctx,
		`UPDATE `+s.schema+`.property_documents
		    SET share_token = $1, updated_at = now()
		  WHERE id = $2 AND property_id = $3 AND deleted_at IS NULL
		 RETURNING id, property_id, COALESCE(title, ''), file_name, mime_type,
		           size_bytes, url, visibility, sort_order, share_token, created_at`,
		token, docID, propertyID).Scan(&d.ID, &d.PropertyID, &d.Title, &d.FileName, &d.MimeType,
		&d.SizeBytes, &d.URL, &d.Visibility, &d.SortOrder, &d.ShareToken, &d.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("enable document share %s: %w", docID, err)
	}
	return &d, nil
}

// DisableShare revokes link sharing (NULLs share_token) for one document scoped to
// its property. Idempotent — a missing / wrong-property / already-off row is a no-op
// (RETURNING yields no row → nil), so a repeated revoke still returns nil. A nil pool
// is a no-op.
func (s DocumentStore) DisableShare(ctx context.Context, propertyID, docID string) error {
	if s.pool == nil {
		return nil
	}
	var id string
	err := s.pool.QueryRow(ctx,
		`UPDATE `+s.schema+`.property_documents
		    SET share_token = NULL, updated_at = now()
		  WHERE id = $1 AND property_id = $2 AND deleted_at IS NULL
		 RETURNING id`,
		docID, propertyID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil // absent / already off — idempotent
	}
	if err != nil {
		return fmt.Errorf("disable document share %s: %w", docID, err)
	}
	return nil
}

// GetByShareToken resolves a share token to its document across ANY property (the
// public link carries no property id), excluding soft-deleted rows. The partial
// UNIQUE index guarantees at most one match. Returns nil when the token is unknown /
// revoked / soft-deleted (the handler 404s). A nil pool yields nil, nil.
func (s DocumentStore) GetByShareToken(ctx context.Context, token string) (*PropertyDocument, error) {
	if s.pool == nil {
		return nil, nil
	}
	var d PropertyDocument
	err := s.pool.QueryRow(ctx,
		`SELECT id, property_id, COALESCE(title, ''), file_name, mime_type,
		        size_bytes, url, visibility, sort_order, share_token, created_at
		   FROM `+s.schema+`.property_documents
		  WHERE share_token = $1 AND deleted_at IS NULL`,
		token).Scan(&d.ID, &d.PropertyID, &d.Title, &d.FileName, &d.MimeType,
		&d.SizeBytes, &d.URL, &d.Visibility, &d.SortOrder, &d.ShareToken, &d.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get document by share token: %w", err)
	}
	return &d, nil
}
