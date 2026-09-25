package property_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/shredbx/sbx-core/pkg/property"
)

// =============================================================================
// IMAGE COLLECTION (Media, task 2606-001) — PD + companion-contract tests.
// ImageCollection is a 1:many cascade-delete child of Property with its OWN UUID
// identity (Decision #0019), mirroring PropertyUnit. A named, ordered subset of
// the property's All-Images pool (members are pkg/image references via the
// image_collection_items join). Exactly one is_default ("Gallery") per property.
// No DB persistence here (that is MD) — companion methods are exercised in
// fixture mode (nil pool) for their validation/guard contract only.
// =============================================================================

// ---- ImageCollection.Validate() ---------------------------------------------

func TestImageCollection_Validate(t *testing.T) {
	cases := []struct {
		name    string
		c       property.ImageCollection
		wantErr bool
	}{
		{"valid default", property.ImageCollection{Name: "Gallery", IsDefault: true}, false},
		{"valid named", property.ImageCollection{Name: "Interior"}, false},
		{"empty name", property.ImageCollection{Name: ""}, true},
		{"whitespace name", property.ImageCollection{Name: "   "}, true},
		{"name at cap", property.ImageCollection{Name: strings.Repeat("x", property.MaxCollectionNameLen)}, false},
		{"name over cap", property.ImageCollection{Name: strings.Repeat("x", property.MaxCollectionNameLen+1)}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.c.Validate(); (err != nil) != tc.wantErr {
				t.Errorf("ImageCollection.Validate() err=%v, wantErr=%v", err, tc.wantErr)
			}
		})
	}
}

// ---- ValidateCollectionSet() — set-level invariants -------------------------

func TestValidateCollectionSet(t *testing.T) {
	cases := []struct {
		name        string
		collections []property.ImageCollection
		wantErr     bool
	}{
		{
			"valid one default + named",
			[]property.ImageCollection{
				{Name: "Gallery", IsDefault: true},
				{Name: "Interior"},
				{Name: "Exterior"},
			},
			false,
		},
		{
			"empty set rejected",
			[]property.ImageCollection{},
			true,
		},
		{
			"no default rejected",
			[]property.ImageCollection{
				{Name: "Interior"},
				{Name: "Exterior"},
			},
			true,
		},
		{
			"two defaults rejected",
			[]property.ImageCollection{
				{Name: "Gallery", IsDefault: true},
				{Name: "Primary", IsDefault: true},
			},
			true,
		},
		{
			"duplicate name rejected (case-insensitive)",
			[]property.ImageCollection{
				{Name: "Gallery", IsDefault: true},
				{Name: "interior"},
				{Name: "Interior"},
			},
			true,
		},
		{
			"member invalid name rejected",
			[]property.ImageCollection{
				{Name: "Gallery", IsDefault: true},
				{Name: ""},
			},
			true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := property.ValidateCollectionSet(tc.collections); (err != nil) != tc.wantErr {
				t.Errorf("ValidateCollectionSet() err=%v, wantErr=%v", err, tc.wantErr)
			}
		})
	}
}

// ---- Unit<->album LINKS (S7 · Decision #0317) — many-to-many, not ownership --
//
// Albums are PROPERTY-owned; a unit SURFACES one or more albums by a unit_collection_links
// row. The link is a reference (independent lifecycle): unlink removes only the link, one
// album may link to many units, and a linked album still lives on the property Media tab.

// UnlinkedCollections returns the albums with NO unit link — the property's own named
// groups + the default "Gallery". An album linked to a unit renders as a unit card, never
// ALSO as a property named group (the #0317 public split). Preserves input order; returns a
// non-nil empty slice on nil input.
func TestUnlinkedCollections(t *testing.T) {
	cols := []property.ImageCollection{
		{ID: "g", Name: "Gallery", IsDefault: true},
		{ID: "i", Name: "Interior"},
		{ID: "c", Name: "Common Areas"}, // linked to a unit below
	}
	links := []property.UnitCollectionLink{
		{UnitID: "unit-a", CollectionID: "c"},
	}
	got := property.UnlinkedCollections(cols, links)
	if len(got) != 2 || got[0].ID != "g" || got[1].ID != "i" {
		t.Errorf("UnlinkedCollections = %v, want [g, i] (linked 'c' excluded, in order)", got)
	}
	// No links → every album is unlinked.
	if all := property.UnlinkedCollections(cols, nil); len(all) != 3 {
		t.Errorf("UnlinkedCollections(no links) = %v, want all 3", all)
	}
	if property.UnlinkedCollections(nil, links) == nil {
		t.Error("UnlinkedCollections(nil) should be a non-nil empty slice")
	}
}

// LinkedCollectionIDs returns the collection IDs linked to a unit (the admin units page:
// which albums are on THIS unit), preserving link order. A shared album appears for each unit
// it is linked to.
func TestLinkedCollectionIDs(t *testing.T) {
	links := []property.UnitCollectionLink{
		{UnitID: "a", CollectionID: "c1", SortOrder: 0},
		{UnitID: "b", CollectionID: "c1", SortOrder: 0}, // c1 shared by a and b
		{UnitID: "a", CollectionID: "c2", SortOrder: 1},
	}
	a := property.LinkedCollectionIDs(links, "a")
	if len(a) != 2 || a[0] != "c1" || a[1] != "c2" {
		t.Errorf("LinkedCollectionIDs(a) = %v, want [c1, c2] in link order", a)
	}
	b := property.LinkedCollectionIDs(links, "b")
	if len(b) != 1 || b[0] != "c1" {
		t.Errorf("LinkedCollectionIDs(b) = %v, want [c1] (shared album)", b)
	}
	if len(property.LinkedCollectionIDs(links, "z")) != 0 {
		t.Error("LinkedCollectionIDs(unknown unit) should be empty")
	}
	if property.LinkedCollectionIDs(nil, "a") == nil {
		t.Error("LinkedCollectionIDs(nil) should be a non-nil empty slice")
	}
}

// LinksForUnits filters the full link set to only those whose unit is in the provided unit
// slice — the public-unit gate for the album-split logic. An album linked ONLY to hidden /
// inactive units returns no links, so it falls back to NamedGalleries instead of disappearing.
// A shared album linked to both an active and a hidden unit keeps its active-unit link.
func TestLinksForUnits(t *testing.T) {
	uA, uB, uHidden := "unit-a", "unit-b", "unit-hidden"
	links := []property.UnitCollectionLink{
		{UnitID: uA, CollectionID: "common", SortOrder: 0},
		{UnitID: uB, CollectionID: "common", SortOrder: 0},   // shared album
		{UnitID: uHidden, CollectionID: "common", SortOrder: 0}, // hidden unit — must be excluded
		{UnitID: uHidden, CollectionID: "solo", SortOrder: 0},   // hidden-only album — must be excluded
	}
	publicUnits := []property.PropertyUnit{
		{ID: uA, Title: "A", IsActive: true},
		{ID: uB, Title: "B", IsActive: true},
	}
	got := property.LinksForUnits(links, publicUnits)
	// 2 links survive: (A,common) and (B,common) in input order. Hidden rows excluded.
	if len(got) != 2 {
		t.Fatalf("LinksForUnits len = %d, want 2; got %v", len(got), got)
	}
	if got[0].UnitID != uA || got[0].CollectionID != "common" {
		t.Errorf("LinksForUnits[0] = %+v, want {UnitID:%q CollectionID:common}", got[0], uA)
	}
	if got[1].UnitID != uB || got[1].CollectionID != "common" {
		t.Errorf("LinksForUnits[1] = %+v, want {UnitID:%q CollectionID:common}", got[1], uB)
	}
	for _, l := range got {
		if l.UnitID == uHidden {
			t.Errorf("LinksForUnits leaked a link for hidden unit %q", uHidden)
		}
	}
	// nil / empty unit set → no links survive.
	if g := property.LinksForUnits(links, nil); len(g) != 0 {
		t.Errorf("LinksForUnits(nil units) = %v, want empty", g)
	}
	// nil links → non-nil empty slice.
	if property.LinksForUnits(nil, publicUnits) == nil {
		t.Error("LinksForUnits(nil links) should be a non-nil empty slice")
	}
}

// UnitGalleryCardsFor flattens albums into public unit cards via the LINK: for each link whose
// unit is in the PUBLIC unit set (the caller passes ActiveUnits — and, once S8 lands, the
// units_public-gated set), one card carrying that link's unit_id (a shared album → one card per
// linked public unit). A link to a hidden/absent unit, or to a missing album, never leaves the
// server (the leak guardrail). Order follows the links slice (loader-ordered by unit then sort).
func TestUnitGalleryCardsFor(t *testing.T) {
	uA, uB, uHidden := "unit-a", "unit-b", "unit-hidden"
	cols := []property.ImageCollection{
		{ID: "g", Name: "Gallery", IsDefault: true},
		{ID: "common", Name: "Common Areas"},
		{ID: "a1", Name: "Unit A photos"},
	}
	links := []property.UnitCollectionLink{
		{UnitID: uA, CollectionID: "common", SortOrder: 0},      // public
		{UnitID: uB, CollectionID: "common", SortOrder: 0},      // public (shared album)
		{UnitID: uA, CollectionID: "a1", SortOrder: 1},          // public
		{UnitID: uHidden, CollectionID: "common", SortOrder: 0}, // hidden unit → excluded
		{UnitID: uA, CollectionID: "ghost", SortOrder: 2},       // missing album → skipped
	}
	publicUnits := []property.PropertyUnit{
		{ID: uA, Title: "A", IsActive: true},
		{ID: uB, Title: "B", IsActive: true},
	}
	got := property.UnitGalleryCardsFor(publicUnits, cols, links)
	// 3 cards: (A,common), (B,common), (A,a1) — hidden + ghost excluded, in link order.
	if len(got) != 3 {
		t.Fatalf("UnitGalleryCardsFor len = %d, want 3 (hidden+ghost excluded); got %v", len(got), got)
	}
	if got[0].UnitID != uA || got[0].ID != "common" ||
		got[1].UnitID != uB || got[1].ID != "common" ||
		got[2].UnitID != uA || got[2].ID != "a1" {
		t.Errorf("UnitGalleryCardsFor = %v, want [(A,common),(B,common),(A,a1)] in link order", got)
	}
	for _, c := range got {
		if c.UnitID == uHidden {
			t.Error("UnitGalleryCardsFor leaked a hidden unit's card")
		}
	}
	// Empty public set → no cards (S8 units_public=false → count only).
	if g := property.UnitGalleryCardsFor(nil, cols, links); len(g) != 0 {
		t.Errorf("UnitGalleryCardsFor(nil units) = %v, want empty", g)
	}
	if property.UnitGalleryCardsFor(publicUnits, nil, links) == nil {
		t.Error("UnitGalleryCardsFor(nil collections) should be a non-nil empty slice")
	}
}

// A UnitGalleryCard embeds the album + carries the linked unit_id so the FE joins the unit's
// name/price; the album fields (id/name/items) ride along for the gallery render.
func TestUnitGalleryCard_JSON(t *testing.T) {
	card := property.UnitGalleryCard{
		ImageCollection: property.ImageCollection{ID: "c1", PropertyID: "p1", Name: "Common Areas"},
		UnitID:          "unit-a",
	}
	data, _ := json.Marshal(card)
	for _, want := range []string{"unit_id", "id", "name"} {
		if !strings.Contains(string(data), want) {
			t.Errorf("UnitGalleryCard JSON %s missing %q", data, want)
		}
	}
}

// ---- NormalizeItemOrder() — 0..n dense ordering, stable on reorder ----------

func TestNormalizeItemOrder(t *testing.T) {
	items := []property.ImageCollectionItem{
		{ImageID: "a", SortOrder: 7},
		{ImageID: "b", SortOrder: 3},
		{ImageID: "c", SortOrder: 99},
	}
	got := property.NormalizeItemOrder(items)
	for i, it := range got {
		if it.SortOrder != i {
			t.Errorf("item %d (%s) SortOrder = %d, want %d", i, it.ImageID, it.SortOrder, i)
		}
	}
	// Slice order preserved (display order IS input order).
	if got[0].ImageID != "a" || got[1].ImageID != "b" || got[2].ImageID != "c" {
		t.Errorf("NormalizeItemOrder reordered slice: %v", got)
	}
	// Empty is safe.
	if property.NormalizeItemOrder(nil) != nil {
		t.Error("NormalizeItemOrder(nil) should be nil")
	}
}

func TestNormalizeCollectionOrder(t *testing.T) {
	cols := []property.ImageCollection{
		{Name: "Gallery", IsDefault: true, SortOrder: 5},
		{Name: "Interior", SortOrder: 1},
		{Name: "Exterior", SortOrder: 9},
	}
	got := property.NormalizeCollectionOrder(cols)
	for i, c := range got {
		if c.SortOrder != i {
			t.Errorf("collection %d (%s) SortOrder = %d, want %d", i, c.Name, c.SortOrder, i)
		}
	}
}

// ---- DefaultCollection / NamedCollections — public split --------------------

func TestDefaultAndNamedCollections(t *testing.T) {
	cols := []property.ImageCollection{
		{ID: "g", Name: "Gallery", IsDefault: true},
		{ID: "i", Name: "Interior"},
		{ID: "e", Name: "Exterior"},
	}
	def := property.DefaultCollection(cols)
	if def == nil || def.ID != "g" {
		t.Fatalf("DefaultCollection = %v, want the is_default 'Gallery'", def)
	}
	named := property.NamedCollections(cols)
	if len(named) != 2 || named[0].ID != "i" || named[1].ID != "e" {
		t.Errorf("NamedCollections = %v, want [Interior, Exterior] in order", named)
	}
	// No default present → nil, and named is a non-nil empty slice on empty input.
	if property.DefaultCollection([]property.ImageCollection{{Name: "x"}}) != nil {
		t.Error("DefaultCollection with no is_default should be nil")
	}
	if property.NamedCollections(nil) == nil {
		t.Error("NamedCollections(nil) should be a non-nil empty slice")
	}
}

// ---- JSON shape — items always present; image projection omitempty ----------

func TestImageCollection_JSON(t *testing.T) {
	c := property.ImageCollection{
		ID: "c1", PropertyID: "p1", Name: "Interior", SortOrder: 2,
		Items: []property.ImageCollectionItem{{ImageID: "img1", SortOrder: 0, URL: "https://r2/x.jpg"}},
	}
	data, _ := json.Marshal(c)
	for _, want := range []string{"id", "property_id", "name", "is_default", "items", "image_id", "url"} {
		if !strings.Contains(string(data), want) {
			t.Errorf("ImageCollection JSON %s missing %q", data, want)
		}
	}
	// A write-side item (no joined projection) omits the url projection.
	empty, _ := json.Marshal(property.ImageCollectionItem{ImageID: "img2", SortOrder: 1})
	for _, absent := range []string{"url"} {
		if strings.Contains(string(empty), absent) {
			t.Errorf("write-side item JSON = %s, should omit %q", empty, absent)
		}
	}
}

// ---- Service companion contract (fixture mode, nil pool) --------------------

// SaveImageCollection validates the set BEFORE any DB work, so the gate is
// exercised with a nil pool (mirrors SaveUnit).
func TestService_SaveImageCollection_RejectsInvalid(t *testing.T) {
	svc := property.NewPropertyService(nil, nil, "bestierealestate")
	if _, err := svc.SaveImageCollection(context.Background(), property.ImageCollection{Name: ""}); err == nil {
		t.Error("SaveImageCollection with empty name = nil error, want validation error")
	}
}

// LoadImageCollections with a nil pool yields no rows (fixture mode) and never panics.
func TestService_LoadImageCollections_NilPool(t *testing.T) {
	svc := property.NewPropertyService(nil, nil, "bestierealestate")
	cols, err := svc.LoadImageCollections(context.Background(), "some-id")
	if err != nil {
		t.Errorf("LoadImageCollections nil pool err = %v, want nil", err)
	}
	if cols != nil {
		t.Errorf("LoadImageCollections nil pool = %v, want nil", cols)
	}
}

// DeleteImageCollection with a nil pool is a no-op (fixture mode) and never panics.
func TestService_DeleteImageCollection_NilPool(t *testing.T) {
	svc := property.NewPropertyService(nil, nil, "bestierealestate")
	if err := svc.DeleteImageCollection(context.Background(), "some-collection-id"); err != nil {
		t.Errorf("DeleteImageCollection nil pool err = %v, want nil", err)
	}
}

// SetCollectionItems with a nil pool is a no-op (fixture mode) and never panics.
func TestService_SetCollectionItems_NilPool(t *testing.T) {
	svc := property.NewPropertyService(nil, nil, "bestierealestate")
	col, err := svc.SetCollectionItems(context.Background(), "some-collection-id", []string{"img1", "img2"})
	if err != nil {
		t.Errorf("SetCollectionItems nil pool err = %v, want nil", err)
	}
	if col != nil {
		t.Errorf("SetCollectionItems nil pool = %v, want nil", col)
	}
}

// LoadUnitCollectionLinks with a nil pool yields no rows (fixture mode) and never panics
// (Decision #0317 — the unit<->album link load).
func TestService_LoadUnitCollectionLinks_NilPool(t *testing.T) {
	svc := property.NewPropertyService(nil, nil, "bestierealestate")
	links, err := svc.LoadUnitCollectionLinks(context.Background(), "some-property-id")
	if err != nil {
		t.Errorf("LoadUnitCollectionLinks nil pool err = %v, want nil", err)
	}
	if links != nil {
		t.Errorf("LoadUnitCollectionLinks nil pool = %v, want nil", links)
	}
}

// LinkUnitCollection / UnlinkUnitCollection with a nil pool are no-ops (fixture mode) and
// never panic (Decision #0317).
func TestService_LinkUnlinkUnitCollection_NilPool(t *testing.T) {
	svc := property.NewPropertyService(nil, nil, "bestierealestate")
	if err := svc.LinkUnitCollection(context.Background(), "unit-id", "collection-id"); err != nil {
		t.Errorf("LinkUnitCollection nil pool err = %v, want nil", err)
	}
	if err := svc.UnlinkUnitCollection(context.Background(), "unit-id", "collection-id"); err != nil {
		t.Errorf("UnlinkUnitCollection nil pool err = %v, want nil", err)
	}
}

// EnsureDefaultCollection with a nil pool is a no-op (fixture mode): it reports no
// insert and never panics. The idempotent INSERT … ON CONFLICT DO NOTHING is
// integration-tested against a real DB (the partial-unique-index conflict path
// cannot run on a nil pool), mirroring the other companion fixture-mode tests.
func TestService_EnsureDefaultCollection_NilPool(t *testing.T) {
	svc := property.NewPropertyService(nil, nil, "bestierealestate")
	inserted, err := svc.EnsureDefaultCollection(context.Background(), "some-property-id")
	if err != nil {
		t.Errorf("EnsureDefaultCollection nil pool err = %v, want nil", err)
	}
	if inserted {
		t.Errorf("EnsureDefaultCollection nil pool inserted = true, want false")
	}
}

// UnitBelongsToProperty with a nil pool cannot verify ownership and reports false
// (fixture mode) without panicking — the safe default (deny) when there is no DB to
// check against. The real EXISTS query is integration-tested against a live DB. The
// link-create path calls this before linking a unit to an album so a unit can never
// link an album of another property (→ 422).
func TestService_UnitBelongsToProperty_NilPool(t *testing.T) {
	svc := property.NewPropertyService(nil, nil, "bestierealestate")
	ok, err := svc.UnitBelongsToProperty(context.Background(), "unit-id", "property-id")
	if err != nil {
		t.Errorf("UnitBelongsToProperty nil pool err = %v, want nil", err)
	}
	if ok {
		t.Error("UnitBelongsToProperty nil pool = true, want false (no DB to verify ownership)")
	}
}

// DefaultCollectionName is the single source of truth for the built-in gallery
// label and MUST match the migration backfill seed ('Gallery'); the
// ensure-default-on-load self-heal seeds with this exact name, so a drift here
// would create a second "default" under a different label and break the
// one-default-per-property invariant the partial unique index guards.
func TestDefaultCollectionName_MirrorsMigrationSeed(t *testing.T) {
	if property.DefaultCollectionName != "Gallery" {
		t.Errorf("DefaultCollectionName = %q, want %q (migration seed)", property.DefaultCollectionName, "Gallery")
	}
}
