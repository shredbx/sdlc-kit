package property_test

import (
	"testing"

	"github.com/shredbx/sbx-core/pkg/property"
	"github.com/shredbx/sbx-core/pkg/repository/postgres"
)

// TestPostgresMapper_ImplementsInterface verifies the compile-time constraint.
func TestPostgresMapper_ImplementsInterface(t *testing.T) {
	var _ postgres.Mapper[property.Property] = property.NewPostgresMapper("bestierealestate")
}

func TestPostgresMapper_TableName(t *testing.T) {
	m := property.NewPostgresMapper("bestierealestate")
	if m.TableName() != "bestierealestate.properties" {
		t.Errorf("want 'bestierealestate.properties', got %q", m.TableName())
	}
}

func TestPostgresMapper_TableName_DifferentSchema(t *testing.T) {
	m := property.NewPostgresMapper("bestays")
	if m.TableName() != "bestays.properties" {
		t.Errorf("want 'bestays.properties', got %q", m.TableName())
	}
}

func TestPostgresMapper_Columns_Count(t *testing.T) {
	m := property.NewPostgresMapper("bestierealestate")
	cols := m.Columns()
	// 31 properties + 12 address + 6 sizes + 9 rooms = 58
	// (post-2605-095: Location→Address adds unit + country, drops region; renames area→sub_district, lat/lng→latitude/longitude; post-2605-067: address gains polygon JSONB column; post-2605-133: address gains map_view JSONB column; post-2605-139: properties gains lifecycle_status BR dictionary FK; post-2605-148: map_view → location_map_view + region_map_view per-picker split; post-2605-150: properties gains landlord_contact_id FK; post-2605-156: properties gains listing_agent_id FK; post-2605-241: properties gains translations JSONB column — Decision #0278 P2; post-2606-001 (M1): properties +6 scalars (ownership_type/condition/direction/road_access/is_featured/listing_priority), sizes +3 (total/usable/balcony_area), rooms +5 (dining/offices/storage/maid/guest_rooms); post-2606-054 (Task 6): properties drop the retired location-dictionary FK — Decision #0295; post-2606-120 (P3): properties gains slug URL key column; post-2606-120 (P1 Batch 3): properties +3 visibility flags location_public/plot_public/policies_public; post-2606-001 (Decision #0318): properties -2 +1 — retire is_featured + listing_priority, add the single rank SMALLINT; post-2607-001: properties +1 topography scalar dictionary FK)
	if len(cols) != 58 {
		t.Errorf("want 58 columns, got %d: %v", len(cols), cols)
	}
}

func TestPostgresMapper_ToRow_NilEnums(t *testing.T) {
	m := property.NewPostgresMapper("bestierealestate")
	p := property.Property{ID: "test-id", IsPublished: false, ForSale: true}
	row, err := m.ToRow(p)
	if err != nil {
		t.Fatalf("ToRow: %v", err)
	}
	if row["title_deed"] != (*string)(nil) {
		t.Error("nil TitleDeed must produce nil *string in row")
	}
}

func TestPostgresMapper_ToRow_ListingFlagsAndPrices(t *testing.T) {
	m := property.NewPostgresMapper("bestierealestate")
	salePrice := int64(15000000)
	p := property.Property{
		ID:        "test-id",
		ForSale:   true,
		ForLease:  false,
		SalePrice: &salePrice,
	}
	row, err := m.ToRow(p)
	if err != nil {
		t.Fatalf("ToRow: %v", err)
	}
	if row["for_sale"] != true {
		t.Errorf("want for_sale=true, got %v", row["for_sale"])
	}
	if row["for_lease"] != false {
		t.Errorf("want for_lease=false, got %v", row["for_lease"])
	}
	sp, ok := row["sale_price"].(*int64)
	if !ok || sp == nil || *sp != 15000000 {
		t.Errorf("want sale_price=*int64(15000000), got %v", row["sale_price"])
	}
	if row["lease_price"] != (*int64)(nil) {
		t.Errorf("want lease_price=nil *int64, got %v", row["lease_price"])
	}
}

// TestPostgresMapper_ToRow_Rank proves the formalized merchandising rank (Decision
// #0318) round-trips through ToRow as the single `rank` column, and that the retired
// is_featured / listing_priority columns are gone from the row map.
func TestPostgresMapper_ToRow_Rank(t *testing.T) {
	m := property.NewPostgresMapper("bestierealestate")
	p := property.Property{ID: "test-id", ForSale: true, Rank: property.RankHigh}
	row, err := m.ToRow(p)
	if err != nil {
		t.Fatalf("ToRow: %v", err)
	}
	rank, ok := row["rank"].(property.Rank)
	if !ok || rank != property.RankHigh {
		t.Errorf("want rank=property.Rank(3), got %v (%T)", row["rank"], row["rank"])
	}
	if _, present := row["is_featured"]; present {
		t.Error("is_featured must be removed from the row map (retired by Decision #0318)")
	}
	if _, present := row["listing_priority"]; present {
		t.Error("listing_priority must be removed from the row map (renamed to rank)")
	}
}

func TestPostgresMapper_FieldColumn_Aliases(t *testing.T) {
	m := property.NewPostgresMapper("bestierealestate")
	cases := map[string]string{
		"is_published": "p.is_published",
		"for_sale":     "p.for_sale",
		"for_lease":    "p.for_lease",
		"sale_price":   "p.sale_price",
		"lease_price":  "p.lease_price",
		"sub_district": "loc.sub_district",
		"country":      "loc.country",
		"province":     "loc.province",
		"deleted_at":   "p.deleted_at",
	}
	for field, want := range cases {
		if got := m.FieldColumn(field); got != want {
			t.Errorf("FieldColumn(%q): want %q, got %q", field, want, got)
		}
	}
}
