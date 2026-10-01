package property_test

import (
	"testing"

	"github.com/shredbx/sbx-core/pkg/property"
)

// =============================================================================
// Sizes.IsZero / Rooms.IsZero (task 2607-001) — the collapse predicate the unit
// companion loader (LoadUnits) relies on. Highest-risk contract: a row of all-NULL
// size/room columns collapses to a nil *Sizes / *Rooms (absence), but a CONFIRMED
// pointer-to-0 is REAL DATA and MUST survive (never be treated as absence). No DB
// needed — pure PD, fixture style (mirrors unit_test.go).
// =============================================================================

// ---- Sizes.IsZero() ---------------------------------------------------------

func TestSizes_IsZero(t *testing.T) {
	if !(*property.Sizes)(nil).IsZero() {
		t.Error("(*Sizes)(nil).IsZero() = false, want true (a nil receiver is zero)")
	}
	if !(&property.Sizes{}).IsZero() {
		t.Error("(&Sizes{}).IsZero() = false, want true (all fields nil)")
	}
	// Any single set field — INCLUDING a confirmed 0.0 — flips it to non-zero.
	cases := []struct {
		name string
		s    property.Sizes
	}{
		{"land_size (ptr 0.0)", property.Sizes{LandSize: ptr(0.0)}}, // 0-preservation: real data
		{"house_size", property.Sizes{HouseSize: ptr(120.5)}},
		{"living_size", property.Sizes{LivingSize: ptr(80.0)}},
		{"total_area", property.Sizes{TotalArea: ptr(200.0)}},
		{"usable_area", property.Sizes{UsableArea: ptr(150.0)}},
		{"balcony_area", property.Sizes{BalconyArea: ptr(10.0)}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.s.IsZero() {
				t.Errorf("Sizes{%s}.IsZero() = true, want false (a set field is data)", tc.name)
			}
		})
	}
}

// ---- Rooms.IsZero() ---------------------------------------------------------

func TestRooms_IsZero(t *testing.T) {
	if !(*property.Rooms)(nil).IsZero() {
		t.Error("(*Rooms)(nil).IsZero() = false, want true (a nil receiver is zero)")
	}
	if !(&property.Rooms{}).IsZero() {
		t.Error("(&Rooms{}).IsZero() = false, want true (all fields nil)")
	}
	cases := []struct {
		name string
		r    property.Rooms
	}{
		{"bedrooms (ptr 0)", property.Rooms{Bedrooms: ptr(0)}}, // 0-preservation: real data
		{"bathrooms", property.Rooms{Bathrooms: ptr(2)}},
		{"kitchens", property.Rooms{Kitchens: ptr(1)}},
		{"living_rooms", property.Rooms{LivingRooms: ptr(1)}},
		{"dining_rooms", property.Rooms{DiningRooms: ptr(1)}},
		{"offices", property.Rooms{Offices: ptr(1)}},
		{"storage_rooms", property.Rooms{StorageRooms: ptr(1)}},
		{"maid_rooms", property.Rooms{MaidRooms: ptr(1)}},
		{"guest_rooms", property.Rooms{GuestRooms: ptr(1)}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.r.IsZero() {
				t.Errorf("Rooms{%s}.IsZero() = true, want false (a set field is data)", tc.name)
			}
		})
	}
}

// ---- The 0-preservation branch, explicit --------------------------------------

// A pointer-to-zero is a CONFIRMED value (0 bedrooms / 0.0 sqm is a fact, not a
// draft-blank), distinct from an unset nil field. IsZero MUST NOT collapse it —
// otherwise the loader would silently drop a confirmed zero back to "no data".
func TestSizesRooms_ZeroPointerIsRealData(t *testing.T) {
	if (&property.Sizes{LandSize: ptr(0.0)}).IsZero() {
		t.Error("Sizes{LandSize: ptr(0.0)}.IsZero() = true, want false — a confirmed 0.0 is data, not absence")
	}
	if (&property.Rooms{Bedrooms: ptr(0)}).IsZero() {
		t.Error("Rooms{Bedrooms: ptr(0)}.IsZero() = true, want false — a confirmed 0 bedrooms is data, not absence")
	}
}

// ---- LoadUnits collapse contract (struct level, no DB) ------------------------

// LoadUnits scans the flat size/room columns into a value-typed group, then keeps
// it only when it carries data: `var g Sizes; scan...; if !g.IsZero() { u.Sizes = &g }`.
// This asserts that exact rule without a database: an all-NULL group collapses to a
// nil pointer (absence), while a group with any set field — including a confirmed
// ptr(0) — survives as a non-nil pointer WITH its value intact (not nil-collapsed).
func TestSizesRooms_LoadCollapseContract(t *testing.T) {
	// The precise expression LoadUnits applies after Scan.
	collapseSizes := func(g property.Sizes) *property.Sizes {
		if g.IsZero() {
			return nil
		}
		return &g
	}
	collapseRooms := func(g property.Rooms) *property.Rooms {
		if g.IsZero() {
			return nil
		}
		return &g
	}

	// All-NULL row → collapsed to nil (mirrors Property's LEFT JOIN yielding nil).
	if collapseSizes(property.Sizes{}) != nil {
		t.Error("all-nil Sizes must collapse to a nil *Sizes")
	}
	if collapseRooms(property.Rooms{}) != nil {
		t.Error("all-nil Rooms must collapse to a nil *Rooms")
	}

	// A confirmed ptr(0) field survives the round-trip with its value intact.
	gotS := collapseSizes(property.Sizes{LandSize: ptr(0.0)})
	if gotS == nil || gotS.LandSize == nil || *gotS.LandSize != 0.0 {
		t.Errorf("Sizes{LandSize: ptr(0.0)} must survive as *Sizes{0.0}, got %+v", gotS)
	}
	gotR := collapseRooms(property.Rooms{Bedrooms: ptr(0)})
	if gotR == nil || gotR.Bedrooms == nil || *gotR.Bedrooms != 0 {
		t.Errorf("Rooms{Bedrooms: ptr(0)} must survive as *Rooms{0}, got %+v", gotR)
	}
}
