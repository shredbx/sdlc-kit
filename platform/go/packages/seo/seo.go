// Package seo provides the SeoMeta value type — the per-entity search-engine and
// open-graph metadata payload persisted as a single JSONB column.
//
// SeoMeta satisfies the Bindable protocol STRUCTURALLY (FI-1: protocols as
// currency, no framework wrapper). Any entity that wants SEO/OG control embeds a
// *SeoMeta field and rounds it through one nullable JSONB column via the
// driver.Valuer / sql.Scanner methods below — exactly as pkg/address.MapView and
// pkg/property.Translations do for their JSONB columns. There is no registry and
// no base class: conformance is the method set, the protocol is the contract.
//
// All fields are optional. An entirely-empty SeoMeta persists as SQL NULL (Value
// returns nil) so an entity that has not set any SEO override never stores a
// literal "{}" — the column reads back as the zero value (rule LOC-001 parity:
// nullable/omitempty JSONB writes NULL, never an empty object).
package seo

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"unicode/utf8"
)

// =============================================================================
// VALIDATION — sentinel errors (match pkg/property's sentinel-error idiom;
// no raw inline error strings at call sites, no any).
// =============================================================================

// ErrMetaTitleTooLong indicates MetaTitle exceeds MaxMetaTitleLen runes. Search
// engines truncate titles past ~60 characters in the SERP, so a longer value is
// rejected at the write boundary rather than silently clipped.
var ErrMetaTitleTooLong = errors.New("meta_title must be <= 60 characters")

// ErrMetaDescriptionTooLong indicates MetaDescription exceeds
// MaxMetaDescriptionLen runes. Search engines truncate descriptions past ~160
// characters; a longer value is rejected rather than clipped.
var ErrMetaDescriptionTooLong = errors.New("meta_description must be <= 160 characters")

// Rune-length caps for the two SERP-rendered fields. Counted in runes (not
// bytes) so multibyte content (Thai, CJK) is measured by visible characters.
const (
	MaxMetaTitleLen       = 60
	MaxMetaDescriptionLen = 160
)

// =============================================================================
// SEO META — JSONB value type
// =============================================================================

// SeoMeta is the per-entity SEO + Open Graph metadata payload. Every field is
// optional (omitempty) — an unset field falls back to the consumer's derived
// default (e.g. the property title) at render time. Stored as one JSONB column;
// see Value / Scan. The struct is the single source of the wire shape consumed by
// the frontend SeoHead renderer:
//
//	{ meta_title?, meta_description?, meta_keywords?: string[],
//	  og_title?, og_description?, og_image?, canonical_url?, noindex? } | null
type SeoMeta struct {
	MetaTitle       string   `json:"meta_title,omitempty"`
	MetaDescription string   `json:"meta_description,omitempty"`
	MetaKeywords    []string `json:"meta_keywords,omitempty"`
	OgTitle         string   `json:"og_title,omitempty"`
	OgDescription   string   `json:"og_description,omitempty"`
	OgImage         string   `json:"og_image,omitempty"`
	CanonicalURL    string   `json:"canonical_url,omitempty"`
	// Noindex, when true, instructs the SeoHead renderer to emit
	// <meta name="robots" content="noindex"> so the entity is excluded from search
	// indexes. Optional; omitempty so an unset value (the common case) is absent
	// from the JSONB payload and an old row that predates this key reads back false.
	// A bool needs no length gate, so Validate is a passthrough for it.
	Noindex bool `json:"noindex,omitempty"`
}

// IsZero reports whether no SEO metadata has been set — i.e. every field is its
// zero value. Value writes SQL NULL in that case so an entity without overrides
// never persists a literal "{}".
func (s SeoMeta) IsZero() bool {
	return s.MetaTitle == "" &&
		s.MetaDescription == "" &&
		len(s.MetaKeywords) == 0 &&
		s.OgTitle == "" &&
		s.OgDescription == "" &&
		s.OgImage == "" &&
		s.CanonicalURL == "" &&
		!s.Noindex
}

// Value implements driver.Valuer for the JSONB column. Mirrors the
// pkg/address.MapView / pkg/property.Translations pattern: marshal to JSON, or
// return nil for the zero value so the row's column reads as SQL NULL (never a
// literal "null" string or "{}").
func (s SeoMeta) Value() (driver.Value, error) {
	if s.IsZero() {
		return nil, nil
	}
	return json.Marshal(s)
}

// Scan implements sql.Scanner. Accepts []byte (the pgx default) or string. A NULL
// column (nil src) or an empty payload scans to the zero value — not an error — so
// a row without SEO metadata round-trips cleanly.
func (s *SeoMeta) Scan(src any) error {
	if src == nil {
		*s = SeoMeta{}
		return nil
	}
	var data []byte
	switch v := src.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return fmt.Errorf("SeoMeta.Scan: unsupported source type %T", src)
	}
	if len(data) == 0 || string(data) == "null" {
		*s = SeoMeta{}
		return nil
	}
	return json.Unmarshal(data, s)
}

// Validate enforces the SERP length invariants: MetaTitle <= 60 runes and
// MetaDescription <= 160 runes. Returns the first failing invariant as a sentinel
// error (ErrMetaTitleTooLong / ErrMetaDescriptionTooLong) so callers can branch on
// errors.Is. All other fields are passthrough — OG/canonical values degrade
// gracefully at the rendering layer and are not length-checked here.
func (s SeoMeta) Validate() error {
	if utf8.RuneCountInString(s.MetaTitle) > MaxMetaTitleLen {
		return ErrMetaTitleTooLong
	}
	if utf8.RuneCountInString(s.MetaDescription) > MaxMetaDescriptionLen {
		return ErrMetaDescriptionTooLong
	}
	return nil
}
