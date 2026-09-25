package property

import (
	"errors"
	"strings"
	"time"
)

// =============================================================================
// IMAGE COLLECTION — 1:many containment child (Decision #0019)
// =============================================================================
//
// ImageCollection is a named, ordered gallery of a property's images — the curated
// groups shown on the public listing (a default "Gallery" plus named ones like
// Interior, Exterior, Floor Plan). It is a containment child (parent = property):
// own UUID identity, 1:many, cascade-deletes with the parent (FK ON DELETE
// CASCADE) — mirroring PropertyUnit / transaction_party, NOT an attribute-group
// (those are 1:1 LEFT-JOIN with a shared PK).
//
// Each collection is an ordered, named subset of the property's All-Images pool
// (the pool = the existing property_images membership of pkg/image assets).
// Members are pkg/image assets REFERENCED (not contained) through the
// image_collection_items join (collection_id, image_id, sort_order) with
// per-collection ordering; deleting a collection removes its memberships only,
// never the underlying image — the image stays in the pool (FK RESTRICT on the
// item's image_id).
//
// Every property has exactly one DEFAULT collection (IsDefault = true, the
// "Gallery") rendered as the listing's primary gallery; named collections render
// as labelled groups under the details on the public page.
//
// Companion-loaded (LoadImageCollections / SaveImageCollection /
// DeleteImageCollection) — kept OUT of the generated mapper: the join +
// per-item ordering are exactly what `sbx generate mapper` cannot emit
// (mirrors PropertyUnit + property.images).

// MaxCollectionNameLen is the governed cap on a collection name (entity.yml
// constraints.maxLength = 80). Names are trimmed before length is measured.
const MaxCollectionNameLen = 80

// DefaultCollectionName is the name of the built-in default "Gallery" collection
// — the listing's primary gallery, exactly one per property. It is the single
// source of truth for that label, mirrored by the migration backfill seed
// ('Gallery') and used by the ensure-default-on-load self-heal so an existing or
// freshly-created property always has a curatable Gallery.
const DefaultCollectionName = "Gallery"

// ErrCollectionNameRequired indicates an ImageCollection was saved without a name.
// Name is the gallery label and is UNIQUE within its parent property.
var ErrCollectionNameRequired = errors.New("image collection name is required")

// ErrCollectionNameTooLong indicates a collection name exceeded MaxCollectionNameLen.
var ErrCollectionNameTooLong = errors.New("image collection name must be at most 80 characters")

// ErrDuplicateCollectionName indicates two collections in the same property share a
// name (case-insensitive). Name is UNIQUE within a property (entity governance).
var ErrDuplicateCollectionName = errors.New("a collection with this name already exists for this property")

// ErrNotExactlyOneDefault indicates a property's collection set does not have
// exactly one is_default collection. Exactly one default per property is the
// invariant enforced both here (the set-level gate) and by a partial unique index.
var ErrNotExactlyOneDefault = errors.New("a property must have exactly one default image collection")

// ImageCollectionItem is one ordered membership row — a REFERENCE to a pkg/image
// asset in the property's pool, with this collection's per-item ordering. It is
// never a copy of the image row; ImageID is the FK to images(id) (RESTRICT, so a
// pool image cannot be deleted while a collection references it). URL is the joined
// image projection, populated at read time for one-round-trip render (omitted on
// the write path where only ImageID + SortOrder are meaningful). Image dimensions
// are NOT projected — the images table does not store width/height.
type ImageCollectionItem struct {
	ImageID   string `json:"image_id" yaml:"image_id"`
	SortOrder int    `json:"sort_order" yaml:"sort_order"`

	// Joined image projection (read-side only). Zero-value on the write path.
	URL string `json:"url,omitempty" yaml:"url,omitempty"`

	// AltText is the image's curated alt text, joined read-side alongside URL so
	// gallery renders keep per-image alt (a11y/SEO) instead of a title-derived
	// fallback. Empty when the image has none.
	AltText string `json:"alt_text,omitempty" yaml:"alt_text,omitempty"`
}

// ImageCollection is a named, ordered gallery within a property. See the file
// header for the full model. The Items slice is the ordered membership (by
// ImageCollectionItem.SortOrder); it is loaded/saved alongside the collection by
// the companion methods.
type ImageCollection struct {
	ID         string `json:"id" yaml:"id"`
	PropertyID string `json:"property_id" yaml:"property_id"`

	// Name is the gallery label (required, <=80); unique within the parent property.
	Name string `json:"name" yaml:"name"`

	// IsDefault marks the built-in "Gallery" — the listing's primary gallery.
	// Exactly one per property (partial unique index + the set-level gate).
	IsDefault bool `json:"is_default" yaml:"is_default"`

	// SortOrder is the ascending display order of this collection within the
	// property's media column.
	SortOrder int `json:"sort_order" yaml:"sort_order"`

	// Items is the ordered membership of pkg/image assets (references, not copies),
	// ordered by ImageCollectionItem.SortOrder. nil/empty = an empty gallery.
	Items []ImageCollectionItem `json:"items" yaml:"items"`

	CreatedAt time.Time `json:"created_at" yaml:"created_at"`
	UpdatedAt time.Time `json:"updated_at" yaml:"updated_at"`
}

// Validate enforces the per-collection invariants: a non-empty, length-bounded
// name. A nil receiver is valid (no collection). Set-level invariants
// (unique-name-within-property + exactly-one-default) are enforced by
// ValidateCollectionSet, since they need the whole property's set.
func (c *ImageCollection) Validate() error {
	if c == nil {
		return nil
	}
	name := strings.TrimSpace(c.Name)
	if name == "" {
		return ErrCollectionNameRequired
	}
	if len(name) > MaxCollectionNameLen {
		return ErrCollectionNameTooLong
	}
	return nil
}

// ValidateCollectionSet enforces the property-level invariants over a property's
// full collection set: every collection validates, names are unique within the
// property (case-insensitive, after trim), and exactly one collection is the
// default. An empty set is rejected (a property always has at least its default
// "Gallery"). This is the gate the save path runs before persisting a set change.
func ValidateCollectionSet(collections []ImageCollection) error {
	if len(collections) == 0 {
		return ErrNotExactlyOneDefault
	}
	seen := make(map[string]struct{}, len(collections))
	defaults := 0
	for i := range collections {
		if err := collections[i].Validate(); err != nil {
			return err
		}
		key := strings.ToLower(strings.TrimSpace(collections[i].Name))
		if _, dup := seen[key]; dup {
			return ErrDuplicateCollectionName
		}
		seen[key] = struct{}{}
		if collections[i].IsDefault {
			defaults++
		}
	}
	if defaults != 1 {
		return ErrNotExactlyOneDefault
	}
	return nil
}

// NormalizeItemOrder rewrites the items' SortOrder to a dense 0..n-1 sequence in
// the slice's current order — the canonical ordering used when persisting a
// membership/reorder write. It preserves slice order (the desired display order is
// the input order) and returns the same slice with SortOrder normalized. A
// nil/empty input is returned unchanged.
func NormalizeItemOrder(items []ImageCollectionItem) []ImageCollectionItem {
	for i := range items {
		items[i].SortOrder = i
	}
	return items
}

// NormalizeCollectionOrder rewrites the collections' SortOrder to a dense 0..n-1
// sequence in the slice's current order — the canonical ordering used when
// persisting a collection-reorder write. Mirrors NormalizeItemOrder for the
// collection level.
func NormalizeCollectionOrder(collections []ImageCollection) []ImageCollection {
	for i := range collections {
		collections[i].SortOrder = i
	}
	return collections
}

// DefaultCollection returns the single is_default collection from a set, or nil
// when none is present. The PUBLIC primary gallery is this collection; named
// collections are the rest (see NamedCollections). It does not enforce the
// exactly-one invariant (use ValidateCollectionSet for that) — it returns the
// FIRST default it finds so a public read degrades gracefully.
func DefaultCollection(collections []ImageCollection) *ImageCollection {
	for i := range collections {
		if collections[i].IsDefault {
			return &collections[i]
		}
	}
	return nil
}

// NamedCollections returns the non-default collections, preserving input order
// (which the loader sorts by SortOrder). These render as labelled groups under
// the details on the public page. A nil/empty input yields a non-nil empty slice.
//
// NOTE (#0317): albums are PROPERTY-owned; a unit surfaces albums by LINK, not
// ownership. The public split keys on the link — NamedCollections(UnlinkedCollections(all,
// links)) are the property's own named groups; UnitGalleryCardsFor projects the linked
// albums into their unit cards.
func NamedCollections(collections []ImageCollection) []ImageCollection {
	named := make([]ImageCollection, 0, len(collections))
	for i := range collections {
		if !collections[i].IsDefault {
			named = append(named, collections[i])
		}
	}
	return named
}

// UnitCollectionLink is one (unit, album) link row — a Decision #0317 reference, NOT
// ownership. A property album is surfaced against a unit by a link; the album stays
// property-owned and survives the link's (or the unit's) deletion. SortOrder is the album's
// order within the unit. Mirrors ImageCollectionItem (the collection<->image membership) one
// level up.
type UnitCollectionLink struct {
	UnitID       string `json:"unit_id" yaml:"unit_id"`
	CollectionID string `json:"collection_id" yaml:"collection_id"`
	SortOrder    int    `json:"sort_order" yaml:"sort_order"`
}

// UnlinkedCollections returns the collections with NO unit link — the property's own named
// groups plus the default "Gallery". An album linked to >=1 unit renders as a unit card, never
// ALSO as a property named group (the #0317 public split). Preserves input order; a nil/empty
// input yields a non-nil empty slice. This is the filter the public named-groups projection
// applies: NamedCollections(UnlinkedCollections(all, links)).
func UnlinkedCollections(collections []ImageCollection, links []UnitCollectionLink) []ImageCollection {
	linked := make(map[string]struct{}, len(links))
	for i := range links {
		linked[links[i].CollectionID] = struct{}{}
	}
	out := make([]ImageCollection, 0, len(collections))
	for i := range collections {
		if _, ok := linked[collections[i].ID]; !ok {
			out = append(out, collections[i])
		}
	}
	return out
}

// LinkedCollectionIDs returns the collection IDs linked to unitID, preserving link order — the
// admin units page's "which albums are on THIS unit". A shared album appears once per unit it is
// linked to. A nil/empty input, or a unit with no links, yields a non-nil empty slice.
func LinkedCollectionIDs(links []UnitCollectionLink, unitID string) []string {
	out := make([]string, 0, len(links))
	for i := range links {
		if links[i].UnitID == unitID {
			out = append(out, links[i].CollectionID)
		}
	}
	return out
}

// LinksForUnits filters the full property link set to only those whose unit is in the provided
// public unit slice. Call this BEFORE UnlinkedCollections so an album linked ONLY to a hidden /
// inactive unit falls back to NamedGalleries instead of disappearing from the public page. A
// shared album linked to both an active and a hidden unit keeps its active-unit link.
// nil/empty units → no links survive (non-nil empty slice).
func LinksForUnits(links []UnitCollectionLink, units []PropertyUnit) []UnitCollectionLink {
	pub := make(map[string]struct{}, len(units))
	for _, u := range units {
		pub[u.ID] = struct{}{}
	}
	out := make([]UnitCollectionLink, 0, len(links))
	for _, l := range links {
		if _, ok := pub[l.UnitID]; ok {
			out = append(out, l)
		}
	}
	return out
}

// UnitGalleryCard is a public-detail projection: a property album surfaced against ONE linked
// unit. The same album linked to N public units yields N cards (each with its unit_id) — the
// link is the projection, not ownership (Decision #0317). The FE joins unit_id -> the unit's
// name/price; the embedded album carries id/name/items for the gallery render.
type UnitGalleryCard struct {
	ImageCollection
	UnitID string `json:"unit_id"`
}

// UnitGalleryCardsFor flattens the property's albums into public unit-gallery cards: for each
// link whose unit is in the PUBLIC unit set, one card carrying that link's unit_id. The caller
// MUST pass the PUBLIC unit set — ActiveUnits (and, once S8 lands, the units_public-gated set) —
// so a hidden / inactive / absent unit's linked album never leaves the server (the leak
// guardrail; passing the RAW unit set would leak hidden-unit cards). A link to a missing album
// is skipped. Order follows the links slice (loader-ordered by unit then sort_order). A
// nil/empty input yields a non-nil empty slice (stable public JSON []).
func UnitGalleryCardsFor(units []PropertyUnit, all []ImageCollection, links []UnitCollectionLink) []UnitGalleryCard {
	pub := make(map[string]struct{}, len(units))
	for i := range units {
		pub[units[i].ID] = struct{}{}
	}
	byID := make(map[string]ImageCollection, len(all))
	for i := range all {
		byID[all[i].ID] = all[i]
	}
	out := make([]UnitGalleryCard, 0, len(links))
	for i := range links {
		if _, ok := pub[links[i].UnitID]; !ok {
			continue
		}
		col, ok := byID[links[i].CollectionID]
		if !ok {
			continue
		}
		out = append(out, UnitGalleryCard{ImageCollection: col, UnitID: links[i].UnitID})
	}
	return out
}
