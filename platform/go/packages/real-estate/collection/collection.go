// Package collection is the domain for the PropertyCollection entity — a curated,
// named, publishable GROUP of properties (the homepage "collection sections",
// e.g. "Beachfront Villas").
//
// PropertyCollection is a TOP-LEVEL aggregate root with its own UUID identity,
// slug, and SeoMeta — mirroring cms.CmsPage (pkg/cms), NOT property.ImageCollection
// (one property's photo galleries). It is the curated "group of properties" shown
// as homepage sections. Membership has a Source discriminator (Decision #0318):
// manual = an admin hand-picks + orders the member properties (the items join);
// smart = a SAVED FILTER EXPRESSION resolved live by the consumer.
//
// DOMAIN-NEUTRALITY (FI-1 / FI-4): the smart filter is held here as an OPAQUE
// json.RawMessage — this package never imports any concrete property-filter type.
// The consumer (BR's api-chi) owns the PropertyFilters shape and decodes/interprets
// the blob. The shared type holds the expression; the consumer interprets it, so a
// new offering can carry its own filter language without touching pkg/collection.
//
// Members are properties REFERENCED (not contained) through the
// property_collection_items join (collection_id, property_id, sort_order) with
// per-collection ordering. The membership join + the JSONB SeoMeta payload are
// companion-loaded (LoadCollectionItems / SaveCollectionItems / LoadSeoMeta /
// SaveSeoMeta in the repository) and kept OUT of any generated mapper — exactly
// what `sbx generate mapper` cannot emit (mirrors property.ImageCollection + the
// property seo_meta column).
//
// The struct is a projection of
// entities/property-collection/entity.yml — keep field names aligned with it and
// with the migration (property_collections / property_collection_items).
package collection

import (
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/shredbx/sbx-core/pkg/seo"
)

// ErrValidation wraps every validation failure. Callers test with
// errors.Is(err, collection.ErrValidation). Mirrors cms.ErrValidation.
var ErrValidation = errors.New("validation")

// Field length caps from entity.yml. Name is the public section heading; Slug is
// the URL key + homepage section key.
const (
	// MaxNameLen is the entity.yml `name` constraint maxLength = 120.
	MaxNameLen = 120
	// MaxSlugLen is the entity.yml `slug` constraint maxLength = 140.
	MaxSlugLen = 140
)

// =============================================================================
// SOURCE — the membership discriminator (a named type, never a raw string)
// =============================================================================

// Source discriminates how a collection's membership is defined (Decision #0318,
// entity.yml `source`). It is a named type — never a raw string in signatures — so
// the discriminator's invariant travels with the value (standing rule: normalize
// discriminators).
//
//   - SourceManual: an admin hand-picks + orders the member properties (the
//     property_collection_items join). Filter is null.
//   - SourceSmart: membership is a SAVED FILTER EXPRESSION (Filter) + Sort, resolved
//     live by the consumer. Items are ignored (FI-4 XOR: filter XOR item references).
type Source string

const (
	// SourceManual is the curated, hand-picked membership (the default for a new
	// collection and the zero-value default applied by OrDefault).
	SourceManual Source = "manual"
	// SourceSmart is the saved-filter membership (an opaque Filter expression + Sort).
	SourceSmart Source = "smart"
)

// Valid reports whether s is one of the two known sources. The empty value is NOT
// valid here — callers default it via OrDefault before validating, so an explicit
// unknown value is still rejected.
func (s Source) Valid() bool {
	return s == SourceManual || s == SourceSmart
}

// OrDefault returns s, or SourceManual when s is the empty value — so an existing
// (pre-#0318) manual collection built without an explicit source defaults to manual.
func (s Source) OrDefault() Source {
	if s == "" {
		return SourceManual
	}
	return s
}

// String renders the source for SQL params.
func (s Source) String() string { return string(s) }

// =============================================================================
// SLUG — the collection URL key (a named type, never a raw string)
// =============================================================================

// slugSeparators collapses runs of ASCII whitespace + URL-hostile punctuation to
// a single hyphen when DERIVING a slug from a name (DeriveSlug). It is the
// derivation helper only — Validate intentionally accepts any non-empty, bounded
// value (entity.yml constrains length, not charset) so a hand-authored slug that
// already reads cleanly is not rejected.
var slugSeparators = regexp.MustCompile(`[^a-z0-9]+`)

// Slug is the unique, URL-safe identifier for a PropertyCollection (e.g.
// "beachfront-villas") — the future /collections/{slug} public route + the
// homepage section key, unique among LIVE (non-deleted) collections. It is a
// named type — never a raw string in signatures — so the value's invariant
// travels with it (standing rule: normalize discriminators). Mirrors cms.Slug.
type Slug string

// String renders the slug for SQL params and URLs.
func (s Slug) String() string { return string(s) }

// Validate enforces the entity.yml `slug` constraints: required (non-empty after
// trim) and bounded at MaxSlugLen runes. Unlike cms.Slug it does NOT impose a
// kebab-case charset — the entity.yml `slug` attribute constrains only length, and
// DeriveSlug already produces a clean kebab form when the admin omits the slug.
func (s Slug) Validate() error {
	v := strings.TrimSpace(string(s))
	if v == "" {
		return errors.New("slug is required")
	}
	if len([]rune(v)) > MaxSlugLen {
		return errors.New("slug must be at most 140 characters")
	}
	return nil
}

// DeriveSlug builds a clean, URL-safe slug from a collection name: lowercase,
// collapse every run of non-[a-z0-9] (whitespace + punctuation) to a single
// hyphen, strip leading/trailing hyphens, cap at MaxSlugLen runes. Used by the
// admin Create path when the caller omits an explicit slug (entity.yml: slug is
// auto-derived from name when absent). Mirrors property.TagSlug's formula
// (lower → replace → cap → trim) adapted to the collection charset + cap.
func DeriveSlug(name string) Slug {
	s := slugSeparators.ReplaceAllString(strings.ToLower(name), "-")
	s = strings.Trim(s, "-")
	if r := []rune(s); len(r) > MaxSlugLen {
		s = strings.Trim(string(r[:MaxSlugLen]), "-")
	}
	return Slug(s)
}

// =============================================================================
// MEMBERSHIP — ordered property references (companion-loaded)
// =============================================================================

// PropertyCollectionItem is one ordered membership row — a REFERENCE to a property
// in this collection, with the collection's per-item ordering. It is never a copy
// of the property row; PropertyID is the FK to properties(id) (CASCADE, so a
// hard-deleted property drops its membership). It carries ONLY the reference +
// ordering — no presentation payload.
//
// This is a PD-layer struct, so it stays free of any presentation/`any` field: the
// member CARD projection (public or admin) is composed at READ time by the consuming
// handler (which owns the card shape), keyed by PropertyID — it is never attached to
// the domain item. Keeping the item presentation-free preserves layer purity and
// removes `any` from core (no raw `any` in Go).
type PropertyCollectionItem struct {
	PropertyID string `json:"property_id" yaml:"property_id"`
	SortOrder  int    `json:"sort_order" yaml:"sort_order"`
}

// =============================================================================
// PROPERTY COLLECTION — the entity
// =============================================================================

// PropertyCollection is a single curated group of properties. Document-type: UUID
// identity, slug, SeoMeta, soft delete. Field order in the struct is cosmetic; the
// storage column order is fixed by the repository's SELECT/scan. Items is the
// ordered membership (by PropertyCollectionItem.SortOrder); it + SeoMeta are
// loaded/saved alongside the collection by the repository companion methods.
type PropertyCollection struct {
	ID string `json:"id" yaml:"id"`

	// Name is the public section heading (required, 1..120).
	Name string `json:"name" yaml:"name"`

	// Slug is the URL key + homepage section key, unique among live collections.
	Slug Slug `json:"slug" yaml:"slug"`

	// Description is optional supporting copy under the heading (full-width).
	Description *string `json:"description,omitempty" yaml:"description,omitempty"`

	// CoverImageURL is a READ-side DERIVED projection — the cover of this collection's
	// FIRST member property (lowest item sort_order, tiebreak property id), resolved
	// to that property's cover image URL. A collection has NO stored cover. It is the
	// zero value (nil) on every write path; the repository derives + populates it on
	// reads via a correlated subquery (public reads from the first PUBLISHED member,
	// admin reads from the first member regardless). nil when the collection has no
	// members, or the first member has no cover. Never persisted.
	CoverImageURL *string `json:"cover_image_url,omitempty" yaml:"cover_image_url,omitempty"`

	// SeoMeta is the optional per-collection SEO + Open Graph override payload — the
	// SHARED seo.SeoMeta value type (the same one property/cms_page carry) as a
	// nullable JSONB column. nil = derive meta/OG from name + cover at render time.
	// Companion-loaded (LoadSeoMeta / SaveSeoMeta) — a pointer so an absent block is
	// omitted from the JSON, mirroring property.SeoMeta.
	SeoMeta *seo.SeoMeta `json:"seo_meta,omitempty" yaml:"seo_meta,omitempty"`

	// Published is the visibility gate — only published collections appear in public
	// reads and on the homepage. Default false (a new collection is a draft).
	Published bool `json:"published" yaml:"published"`

	// SortOrder is the ascending display order among collections (homepage order).
	SortOrder int `json:"sort_order" yaml:"sort_order"`

	// Source is the membership discriminator (manual | smart). The zero value ("")
	// defaults to SourceManual via Source.OrDefault so a pre-#0318 collection built
	// without an explicit source stays manual.
	Source Source `json:"source" yaml:"source"`

	// Filter is the smart-only OPAQUE catalog filter expression (the consumer's
	// PropertyFilters language, stored as JSONB). This package keeps it domain-neutral
	// — a json.RawMessage it never decodes; the consumer (api-chi) owns the concrete
	// shape and interprets it (FI-1 / FI-4). nil/empty on a manual collection.
	Filter json.RawMessage `json:"filter,omitempty" yaml:"filter,omitempty"`

	// Sort is the smart-only catalog sort code applied when resolving the smart
	// membership (default "rank_desc"). nil on a manual collection.
	Sort *string `json:"sort,omitempty" yaml:"sort,omitempty"`

	// Items is the ordered membership of property references (read side resolves
	// each to the public property card). nil/empty = an empty collection. Ignored
	// (never written) on a smart collection (FI-4 XOR).
	Items []PropertyCollectionItem `json:"items" yaml:"items"`

	CreatedAt time.Time `json:"created_at" yaml:"created_at"`
	UpdatedAt time.Time `json:"updated_at" yaml:"updated_at"`
}

// Validate enforces the entity.yml constraints: name is required + bounded
// (1..120 after trim) and slug is required + bounded (1..140 after trim). The
// optional SeoMeta (when present) is length-gated for the two SERP-rendered fields
// (meta_title <=60, meta_description <=160), mirroring how cms.CmsPage.Validate
// gates its embedded seo_meta at the save boundary.
func (c PropertyCollection) Validate() error {
	name := strings.TrimSpace(c.Name)
	if name == "" {
		return errors.New("name is required")
	}
	if len([]rune(name)) > MaxNameLen {
		return errors.New("name must be at most 120 characters")
	}
	if err := c.Slug.Validate(); err != nil {
		return err
	}
	// Source discriminator: the empty value defaults to manual (backward-compat with
	// pre-#0318 collections); an explicit unknown value is rejected. A smart collection
	// MUST carry a filter expression (FI-4 XOR: smart ⇒ filter set). The filter is an
	// OPAQUE blob — only its presence is checked here, never its shape (the consumer
	// owns interpretation).
	source := c.Source.OrDefault()
	if !source.Valid() {
		return errors.New("source must be manual or smart")
	}
	if source == SourceSmart && len(c.Filter) == 0 {
		return errors.New("a smart collection requires a filter")
	}
	if c.SeoMeta != nil {
		if err := c.SeoMeta.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// NormalizeItemOrder rewrites the items' SortOrder to a dense 0..n-1 sequence in
// the slice's current order — the canonical ordering used when persisting a
// membership/reorder write. It preserves slice order (the desired display order is
// the input order) and returns the same slice with SortOrder normalized. A
// nil/empty input is returned unchanged. Mirrors property.NormalizeItemOrder.
func NormalizeItemOrder(items []PropertyCollectionItem) []PropertyCollectionItem {
	for i := range items {
		items[i].SortOrder = i
	}
	return items
}

// NormalizeCollectionOrder rewrites the collections' SortOrder to a dense 0..n-1
// sequence in the slice's current order — the canonical ordering used when
// persisting a collection-reorder write (the homepage section order). Mirrors
// NormalizeItemOrder for the collection level / property.NormalizeCollectionOrder.
func NormalizeCollectionOrder(collections []PropertyCollection) []PropertyCollection {
	for i := range collections {
		collections[i].SortOrder = i
	}
	return collections
}
