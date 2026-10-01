package collection

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/shredbx/sbx-core/pkg/seo"
)

func strptr(s string) *string { return &s }

// =============================================================================
// Slug.Validate — required + length-bounded (entity.yml slug: 1..140)
// =============================================================================

func TestSlug_Validate(t *testing.T) {
	cases := []struct {
		name    string
		slug    Slug
		wantErr bool
	}{
		{"simple", "beachfront-villas", false},
		{"single char", "a", false},
		{"max length", Slug(strings.Repeat("a", MaxSlugLen)), false},
		{"empty", "", true},
		{"whitespace only", "   ", true},
		{"over max length", Slug(strings.Repeat("a", MaxSlugLen+1)), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.slug.Validate()
			if tc.wantErr && err == nil {
				t.Fatalf("expected error for slug %q, got nil", tc.slug)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error for slug %q: %v", tc.slug, err)
			}
		})
	}
}

// =============================================================================
// DeriveSlug — name → clean kebab slug (auto-derivation when slug absent)
// =============================================================================

func TestDeriveSlug(t *testing.T) {
	cases := []struct {
		in   string
		want Slug
	}{
		{"Beachfront Villas", "beachfront-villas"},
		{"New Developments!", "new-developments"},
		{"  Trim  Me  ", "trim-me"},
		{"Sea-View & Pool", "sea-view-pool"},
		{"UPPER lower", "upper-lower"},
		{"---edges---", "edges"},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			got := DeriveSlug(tc.in)
			if got != tc.want {
				t.Fatalf("DeriveSlug(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestDeriveSlug_CapsAtMaxLen(t *testing.T) {
	long := strings.Repeat("a", MaxSlugLen+50)
	got := DeriveSlug(long)
	if len([]rune(got.String())) > MaxSlugLen {
		t.Fatalf("DeriveSlug exceeded MaxSlugLen: got %d runes", len([]rune(got.String())))
	}
	// The derived slug must itself pass Slug.Validate.
	if err := got.Validate(); err != nil {
		t.Fatalf("derived slug failed Validate: %v", err)
	}
}

// =============================================================================
// PropertyCollection.Validate — name 1..120, slug required, seo_meta gated
// =============================================================================

func TestPropertyCollection_Validate(t *testing.T) {
	valid := PropertyCollection{Name: "Beachfront Villas", Slug: "beachfront-villas"}
	if err := valid.Validate(); err != nil {
		t.Fatalf("expected valid collection to pass, got: %v", err)
	}

	t.Run("empty name rejected", func(t *testing.T) {
		c := PropertyCollection{Name: "  ", Slug: "x"}
		if err := c.Validate(); err == nil {
			t.Fatal("expected error for empty name")
		}
	})

	t.Run("over-long name rejected", func(t *testing.T) {
		c := PropertyCollection{Name: strings.Repeat("x", MaxNameLen+1), Slug: "x"}
		if err := c.Validate(); err == nil {
			t.Fatal("expected error for over-long name")
		}
	})

	t.Run("max-length name accepted", func(t *testing.T) {
		c := PropertyCollection{Name: strings.Repeat("x", MaxNameLen), Slug: "x"}
		if err := c.Validate(); err != nil {
			t.Fatalf("expected max-length name to pass, got: %v", err)
		}
	})

	t.Run("empty slug rejected", func(t *testing.T) {
		c := PropertyCollection{Name: "OK", Slug: ""}
		if err := c.Validate(); err == nil {
			t.Fatal("expected error for empty slug")
		}
	})

	t.Run("over-long seo meta_title rejected", func(t *testing.T) {
		c := PropertyCollection{
			Name:    "OK",
			Slug:    "ok",
			SeoMeta: &seo.SeoMeta{MetaTitle: strings.Repeat("t", seo.MaxMetaTitleLen+1)},
		}
		if err := c.Validate(); err == nil {
			t.Fatal("expected error for over-long meta_title")
		} else if !errors.Is(err, seo.ErrMetaTitleTooLong) {
			t.Fatalf("expected ErrMetaTitleTooLong, got: %v", err)
		}
	})

	t.Run("description does not affect validity", func(t *testing.T) {
		c := PropertyCollection{Name: "OK", Slug: "ok", Description: strptr("any copy")}
		if err := c.Validate(); err != nil {
			t.Fatalf("description should not fail validation, got: %v", err)
		}
	})
}

// =============================================================================
// NormalizeItemOrder / NormalizeCollectionOrder — dense 0..n in slice order
// =============================================================================

func TestNormalizeItemOrder(t *testing.T) {
	items := []PropertyCollectionItem{
		{PropertyID: "a", SortOrder: 9},
		{PropertyID: "b", SortOrder: 4},
		{PropertyID: "c", SortOrder: 7},
	}
	got := NormalizeItemOrder(items)
	for i, it := range got {
		if it.SortOrder != i {
			t.Fatalf("item %d: SortOrder = %d, want %d", i, it.SortOrder, i)
		}
	}
	// Order is preserved (slice order is the desired display order).
	if got[0].PropertyID != "a" || got[1].PropertyID != "b" || got[2].PropertyID != "c" {
		t.Fatalf("NormalizeItemOrder reordered the slice: %+v", got)
	}
}

func TestNormalizeItemOrder_Empty(t *testing.T) {
	if got := NormalizeItemOrder(nil); got != nil {
		t.Fatalf("nil input should return nil, got: %+v", got)
	}
}

func TestNormalizeCollectionOrder(t *testing.T) {
	cols := []PropertyCollection{
		{Name: "A", Slug: "a", SortOrder: 5},
		{Name: "B", Slug: "b", SortOrder: 2},
	}
	got := NormalizeCollectionOrder(cols)
	for i, c := range got {
		if c.SortOrder != i {
			t.Fatalf("collection %d: SortOrder = %d, want %d", i, c.SortOrder, i)
		}
	}
	if got[0].Slug != "a" || got[1].Slug != "b" {
		t.Fatalf("NormalizeCollectionOrder reordered the slice: %+v", got)
	}
}

// =============================================================================
// Source — membership discriminator (Decision #0318, FI-4)
// =============================================================================

func TestSource_ValidAndOrDefault(t *testing.T) {
	if !SourceManual.Valid() || !SourceSmart.Valid() {
		t.Fatal("SourceManual and SourceSmart must be Valid")
	}
	if Source("").Valid() {
		t.Error("the empty source is NOT Valid (callers default it via OrDefault first)")
	}
	if Source("bogus").Valid() {
		t.Error("an unknown source must not be Valid")
	}
	if Source("").OrDefault() != SourceManual {
		t.Error("the empty source must default to manual (backward-compat with pre-#0318 collections)")
	}
	if SourceSmart.OrDefault() != SourceSmart {
		t.Error("a set source must be returned unchanged by OrDefault")
	}
}

// A manual collection (the default) needs no filter. The empty source defaults to
// manual and validates.
func TestPropertyCollection_Validate_ManualSourceNoFilter(t *testing.T) {
	c := PropertyCollection{Name: "Curated", Slug: "curated"} // Source "" → manual
	if err := c.Validate(); err != nil {
		t.Fatalf("a manual collection without a filter must validate, got: %v", err)
	}
	cManual := PropertyCollection{Name: "Curated", Slug: "curated", Source: SourceManual}
	if err := cManual.Validate(); err != nil {
		t.Fatalf("an explicit manual collection must validate, got: %v", err)
	}
}

// A smart collection MUST carry a filter expression (FI-4 XOR). Absent filter →
// rejected; present opaque filter → accepted.
func TestPropertyCollection_Validate_SmartRequiresFilter(t *testing.T) {
	noFilter := PropertyCollection{Name: "For Lease", Slug: "for-lease", Source: SourceSmart}
	if err := noFilter.Validate(); err == nil {
		t.Fatal("a smart collection without a filter must be rejected")
	}

	withFilter := PropertyCollection{
		Name:   "For Lease",
		Slug:   "for-lease",
		Source: SourceSmart,
		Filter: json.RawMessage(`{"for_lease":true}`),
	}
	if err := withFilter.Validate(); err != nil {
		t.Fatalf("a smart collection with a filter must validate, got: %v", err)
	}
}

// An explicit unknown source is rejected even though the empty value defaults to
// manual.
func TestPropertyCollection_Validate_UnknownSourceRejected(t *testing.T) {
	c := PropertyCollection{Name: "X", Slug: "x", Source: Source("bogus")}
	if err := c.Validate(); err == nil {
		t.Fatal("an explicit unknown source must be rejected")
	}
}
