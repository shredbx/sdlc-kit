package property_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/shredbx/sbx-core/pkg/property"
)

// Documents store (S2c, task 2606-006) — PD-layer unit tests on the pure helpers:
// the DocumentVisibility named type (parse + default-to-private), the boundary
// validation (MIME allowlist + size cap — scenario DOC-SC2), the shared object-key
// builder, the per-document Validate gate, and the public-only projection filter.
// These run with no DB (the DocumentStore CRUD against a real pool is integration-
// tested) — they exercise exactly the logic the API boundary depends on.

// ---- DocumentVisibility: named type + parse + default -----------------------

// A blank visibility defaults to PRIVATE — the safe default (public is an explicit
// opt-in per the entity governance). Visibility default MUST be unit-tested.
func TestParseDocumentVisibility_BlankDefaultsPrivate(t *testing.T) {
	for _, raw := range []string{"", "   "} {
		v, err := property.ParseDocumentVisibility(raw)
		if err != nil {
			t.Fatalf("ParseDocumentVisibility(%q): unexpected error %v", raw, err)
		}
		if v != property.VisibilityPrivate {
			t.Fatalf("ParseDocumentVisibility(%q): want private, got %q", raw, v)
		}
	}
}

// public / private (any case, trimmed) parse to the seeded codes.
func TestParseDocumentVisibility_SeededCodes(t *testing.T) {
	cases := map[string]property.DocumentVisibility{
		"public":    property.VisibilityPublic,
		"PUBLIC":    property.VisibilityPublic,
		" private ": property.VisibilityPrivate,
	}
	for raw, want := range cases {
		got, err := property.ParseDocumentVisibility(raw)
		if err != nil {
			t.Fatalf("ParseDocumentVisibility(%q): %v", raw, err)
		}
		if got != want {
			t.Fatalf("ParseDocumentVisibility(%q): want %q, got %q", raw, want, got)
		}
	}
}

// A non-blank value outside {public, private} is rejected (ErrInvalidVisibility) —
// the caller maps it to a 4xx.
func TestParseDocumentVisibility_Invalid(t *testing.T) {
	_, err := property.ParseDocumentVisibility("internal")
	if !errors.Is(err, property.ErrInvalidVisibility) {
		t.Fatalf("want ErrInvalidVisibility, got %v", err)
	}
}

func TestDocumentVisibility_Valid(t *testing.T) {
	if !property.VisibilityPublic.Valid() || !property.VisibilityPrivate.Valid() {
		t.Fatalf("public/private must be valid")
	}
	if property.DocumentVisibility("shared").Valid() {
		t.Fatalf("'shared' must NOT be a valid visibility")
	}
}

// ---- Boundary validation: MIME allowlist + size cap (DOC-SC2) ---------------

// Every allowed MIME within the cap passes the boundary gate.
func TestValidateDocumentUpload_AllowedMIME(t *testing.T) {
	for _, mime := range []string{
		"application/pdf",
		"text/plain",
		"text/plain; charset=utf-8", // parameterized type still matches
		"image/jpeg",
		"image/png",
		"image/webp",
	} {
		if err := property.ValidateDocumentUpload(mime, 1024); err != nil {
			t.Fatalf("ValidateDocumentUpload(%q): unexpected reject %v", mime, err)
		}
	}
}

// DOC-SC2: a disallowed MIME is rejected at the boundary with ErrDocumentMIMENotAllowed
// (so the handler returns a 4xx with NO row and NO object).
func TestValidateDocumentUpload_DisallowedMIME_Rejected(t *testing.T) {
	for _, mime := range []string{
		"application/zip",
		"application/x-msdownload",
		"image/gif",
		"video/mp4",
		"application/octet-stream",
	} {
		err := property.ValidateDocumentUpload(mime, 1024)
		if !errors.Is(err, property.ErrDocumentMIMENotAllowed) {
			t.Fatalf("ValidateDocumentUpload(%q): want ErrDocumentMIMENotAllowed, got %v", mime, err)
		}
	}
}

// DOC-SC2: an upload over the 10MB cap is rejected with ErrDocumentTooLarge.
func TestValidateDocumentUpload_Oversize_Rejected(t *testing.T) {
	err := property.ValidateDocumentUpload("application/pdf", property.MaxDocumentBytes+1)
	if !errors.Is(err, property.ErrDocumentTooLarge) {
		t.Fatalf("oversize: want ErrDocumentTooLarge, got %v", err)
	}
}

// Exactly at the cap is accepted; a zero/empty upload is rejected.
func TestValidateDocumentUpload_CapBoundary(t *testing.T) {
	if err := property.ValidateDocumentUpload("application/pdf", property.MaxDocumentBytes); err != nil {
		t.Fatalf("at-cap upload should pass, got %v", err)
	}
	if err := property.ValidateDocumentUpload("application/pdf", 0); !errors.Is(err, property.ErrDocumentTooLarge) {
		t.Fatalf("empty upload: want ErrDocumentTooLarge, got %v", err)
	}
}

// ---- Object-key builder: shared scheme, server-derived extension ------------

// The key matches the SHARED image scheme exactly: properties/{pid}/documents/{did}.{ext},
// with the extension derived from the validated MIME (never a client file name).
func TestDocumentObjectKey_SharedScheme(t *testing.T) {
	pid := "11111111-1111-1111-1111-111111111111"
	did := "22222222-2222-2222-2222-222222222222"
	cases := map[string]string{
		"application/pdf":           "properties/" + pid + "/documents/" + did + ".pdf",
		"text/plain; charset=utf-8": "properties/" + pid + "/documents/" + did + ".txt",
		"image/jpeg":                "properties/" + pid + "/documents/" + did + ".jpg",
		"image/png":                 "properties/" + pid + "/documents/" + did + ".png",
		"image/webp":                "properties/" + pid + "/documents/" + did + ".webp",
	}
	for mime, want := range cases {
		if got := property.DocumentObjectKey(pid, did, mime); got != want {
			t.Fatalf("DocumentObjectKey(%q): want %q, got %q", mime, want, got)
		}
	}
}

// ---- Per-document Validate gate ---------------------------------------------

// A document with a file name and a valid size validates; a blank visibility is
// defaulted to private in-place.
func TestPropertyDocument_Validate_DefaultsVisibility(t *testing.T) {
	d := property.PropertyDocument{
		PropertyID: "11111111-1111-1111-1111-111111111111",
		FileName:   "deed.pdf",
		MimeType:   "application/pdf",
		SizeBytes:  2048,
	}
	if err := d.Validate(); err != nil {
		t.Fatalf("Validate: unexpected error %v", err)
	}
	if d.Visibility != property.VisibilityPrivate {
		t.Fatalf("blank visibility must default to private, got %q", d.Visibility)
	}
}

// A blank file name is rejected (the original upload name is required).
func TestPropertyDocument_Validate_FileNameRequired(t *testing.T) {
	d := property.PropertyDocument{FileName: "   ", MimeType: "application/pdf", SizeBytes: 10}
	if err := d.Validate(); !errors.Is(err, property.ErrDocumentFileNameRequired) {
		t.Fatalf("want ErrDocumentFileNameRequired, got %v", err)
	}
}

// An over-length title is rejected (<=120 chars).
func TestPropertyDocument_Validate_TitleTooLong(t *testing.T) {
	long := make([]byte, property.MaxDocumentTitleLen+1)
	for i := range long {
		long[i] = 'x'
	}
	d := property.PropertyDocument{FileName: "f.pdf", Title: string(long), MimeType: "application/pdf", SizeBytes: 10}
	if err := d.Validate(); !errors.Is(err, property.ErrDocumentTitleTooLong) {
		t.Fatalf("want ErrDocumentTitleTooLong, got %v", err)
	}
}

// A file name beyond the 255-char contract cap is rejected.
func TestPropertyDocument_Validate_FileNameTooLong(t *testing.T) {
	long := make([]byte, property.MaxDocumentFileNameLen+1)
	for i := range long {
		long[i] = 'x'
	}
	d := property.PropertyDocument{FileName: string(long), MimeType: "application/pdf", SizeBytes: 10}
	if err := d.Validate(); !errors.Is(err, property.ErrDocumentFileNameTooLong) {
		t.Fatalf("want ErrDocumentFileNameTooLong, got %v", err)
	}
}

// An oversize document is rejected by the size invariant.
func TestPropertyDocument_Validate_Oversize(t *testing.T) {
	d := property.PropertyDocument{FileName: "f.pdf", MimeType: "application/pdf", SizeBytes: property.MaxDocumentBytes + 1}
	if err := d.Validate(); !errors.Is(err, property.ErrDocumentTooLarge) {
		t.Fatalf("want ErrDocumentTooLarge, got %v", err)
	}
}

// ---- Public projection filter -----------------------------------------------

// FilterPublicDocuments keeps ONLY the public group — private documents must never
// enter the public payload — and preserves input order, always non-nil.
func TestFilterPublicDocuments(t *testing.T) {
	docs := []property.PropertyDocument{
		{ID: "a", Visibility: property.VisibilityPrivate, SortOrder: 0},
		{ID: "b", Visibility: property.VisibilityPublic, SortOrder: 1},
		{ID: "c", Visibility: property.VisibilityPrivate, SortOrder: 2},
		{ID: "d", Visibility: property.VisibilityPublic, SortOrder: 3},
	}
	got := property.FilterPublicDocuments(docs)
	if len(got) != 2 || got[0].ID != "b" || got[1].ID != "d" {
		t.Fatalf("want public [b d], got %+v", got)
	}
	for _, d := range got {
		if d.Visibility != property.VisibilityPublic {
			t.Fatalf("private document leaked into public projection: %+v", d)
		}
	}
	// Non-nil empty on empty input (stable JSON []).
	if empty := property.FilterPublicDocuments(nil); empty == nil {
		t.Fatalf("FilterPublicDocuments(nil) must be non-nil")
	}
}

// FilterPublicDocuments must STRIP the share token from a PUBLIC document (scope-10
// F8): a share token is a private bearer secret and must never ride the public
// payload, even on a public row that happens to have link-sharing on. The strip is
// on the projection copy only — the caller's original slice is left untouched.
func TestFilterPublicDocuments_StripsShareToken(t *testing.T) {
	tok := "a-live-bearer-token"
	docs := []property.PropertyDocument{
		{ID: "pub-shared", Visibility: property.VisibilityPublic, ShareToken: &tok},
	}
	got := property.FilterPublicDocuments(docs)
	if len(got) != 1 {
		t.Fatalf("want 1 public document, got %d", len(got))
	}
	if got[0].ShareToken != nil {
		t.Fatalf("share token leaked into public projection: %q", *got[0].ShareToken)
	}
	// The caller's original slice must be untouched (defensive by-value projection).
	if docs[0].ShareToken == nil || *docs[0].ShareToken != tok {
		t.Fatalf("projection mutated the caller's slice: %+v", docs[0].ShareToken)
	}
}

// NormalizeDocumentOrder rewrites SortOrder to a dense 0..n-1 in slice order.
func TestNormalizeDocumentOrder(t *testing.T) {
	docs := []property.PropertyDocument{{SortOrder: 9}, {SortOrder: 4}, {SortOrder: 7}}
	got := property.NormalizeDocumentOrder(docs)
	for i := range got {
		if got[i].SortOrder != i {
			t.Fatalf("index %d: want sort_order %d, got %d", i, i, got[i].SortOrder)
		}
	}
}

// ---- Share token: link sharing (scope-07) -----------------------------------

// GenerateShareToken mints a URL-safe bearer token: 32 random bytes base64url
// (RawURLEncoding → 43 chars, no padding), so it drops straight into a path
// segment. Two calls differ (fresh entropy each time). This is the redistributable
// share secret (stored raw), so URL-safety + unguessable length are the contract.
func TestGenerateShareToken_URLSafeAndUnique(t *testing.T) {
	a, err := property.GenerateShareToken()
	if err != nil {
		t.Fatalf("GenerateShareToken: %v", err)
	}
	b, err := property.GenerateShareToken()
	if err != nil {
		t.Fatalf("GenerateShareToken: %v", err)
	}
	if a == "" || b == "" {
		t.Fatalf("token must be non-empty (got %q, %q)", a, b)
	}
	if a == b {
		t.Fatalf("two tokens must differ (both %q)", a)
	}
	// 32 bytes RawURLEncoding = ceil(32*8/6) = 43 chars, no '=' padding.
	if len(a) != 43 {
		t.Fatalf("token length: want 43 (32B base64url), got %d (%q)", len(a), a)
	}
	for _, r := range a {
		if !((r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_') {
			t.Fatalf("token %q carries a non-URL-safe rune %q", a, r)
		}
	}
}

// The share_token field is omitted from the wire when nil (sharing off) and present
// as a string when set — so an unshared document's payload never carries the key,
// and a shared one exposes the token the manager copies (omitempty on *string).
func TestPropertyDocument_ShareTokenWireShape(t *testing.T) {
	// nil → key absent.
	off := property.PropertyDocument{ID: "d1", FileName: "f.pdf", Visibility: property.VisibilityPrivate}
	raw, err := json.Marshal(off)
	if err != nil {
		t.Fatalf("marshal off: %v", err)
	}
	var offMap map[string]any
	_ = json.Unmarshal(raw, &offMap)
	if _, present := offMap["share_token"]; present {
		t.Fatalf("share_token must be omitted when nil, got %s", raw)
	}

	// set → key present with the token value.
	tok := "abc123"
	on := property.PropertyDocument{ID: "d1", FileName: "f.pdf", Visibility: property.VisibilityPrivate, ShareToken: &tok}
	raw, err = json.Marshal(on)
	if err != nil {
		t.Fatalf("marshal on: %v", err)
	}
	var onMap map[string]any
	_ = json.Unmarshal(raw, &onMap)
	if onMap["share_token"] != "abc123" {
		t.Fatalf("share_token wire value: want abc123, got %v (%s)", onMap["share_token"], raw)
	}
}

// Nil-pool (fixture) contract of the share companions — mirrors Save/Get/Delete:
// EnableShare echoes a freshly-minted token without persisting; DisableShare is a
// no-op nil; GetByShareToken yields nil,nil. The real DB round-trip (enable →
// resolve 302 → revoke → 404) is exercised by the handler e2e, exactly as the other
// document CRUD paths are (a nil pool cannot run the UPDATE … RETURNING).
func TestDocumentShareCompanions_NilPool(t *testing.T) {
	store := property.NewDocumentStore(nil, "bestierealestate")

	doc, err := store.EnableShare(context.Background(), "p1", "d1")
	if err != nil {
		t.Fatalf("EnableShare(nil pool): %v", err)
	}
	if doc == nil || doc.ShareToken == nil || *doc.ShareToken == "" {
		t.Fatalf("EnableShare(nil pool): want fixture echo with a token, got %+v", doc)
	}

	if err := store.DisableShare(context.Background(), "p1", "d1"); err != nil {
		t.Fatalf("DisableShare(nil pool): want nil, got %v", err)
	}

	got, err := store.GetByShareToken(context.Background(), "anything")
	if err != nil {
		t.Fatalf("GetByShareToken(nil pool): %v", err)
	}
	if got != nil {
		t.Fatalf("GetByShareToken(nil pool): want nil, got %+v", got)
	}
}

// ---- Wire shape -------------------------------------------------------------

// The PropertyDocument wire shape carries the exact fields the API serializes.
func TestPropertyDocument_JSONShape(t *testing.T) {
	d := property.PropertyDocument{
		ID:         "11111111-1111-1111-1111-111111111111",
		PropertyID: "22222222-2222-2222-2222-222222222222",
		Title:      "Title Deed",
		FileName:   "deed.pdf",
		MimeType:   "application/pdf",
		SizeBytes:  123456,
		URL:        "https://cdn.example.com/properties/x/documents/y.pdf",
		Visibility: property.VisibilityPublic,
		SortOrder:  0,
		CreatedAt:  time.Date(2026, 6, 16, 9, 0, 0, 0, time.UTC),
	}
	raw, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var back map[string]any
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, k := range []string{"id", "property_id", "title", "file_name", "mime_type",
		"size_bytes", "url", "visibility", "sort_order", "created_at"} {
		if _, ok := back[k]; !ok {
			t.Fatalf("missing JSON field %q in %s", k, raw)
		}
	}
	if back["visibility"] != "public" {
		t.Fatalf("visibility wire value: want public, got %v", back["visibility"])
	}
}
