package property_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/shredbx/sbx-core/pkg/property"
)

// =============================================================================
// L1 SUPERSET WIRING (M1, task 2606-001) — PD tests.
// Covers the pure-domain additions: named-type Valid() guards, attribute-group
// Validate() range rules, Property.Validate() descent, and JSON omitempty for
// the new structs + extended Rooms/Sizes fields. No persistence here (that is MD).
// =============================================================================

// ---- Named-type Valid() guards (SC-M1-05 parse-guard) -----------------------

func TestOwnership_Valid(t *testing.T) {
	for _, o := range []property.Ownership{property.OwnershipFreehold, property.OwnershipLeasehold, property.OwnershipCompany} {
		if !o.Valid() {
			t.Errorf("Ownership(%q).Valid() = false, want true", o)
		}
	}
	for _, o := range []property.Ownership{"", "rent-to-own", "bogus"} {
		if o.Valid() {
			t.Errorf("Ownership(%q).Valid() = true, want false", o)
		}
	}
}

func TestCondition_Valid(t *testing.T) {
	for _, c := range []property.Condition{property.ConditionNew, property.ConditionExcellent, property.ConditionGood, property.ConditionFair, property.ConditionNeedsRenovation, property.ConditionMoveInReady} {
		if !c.Valid() {
			t.Errorf("Condition(%q).Valid() = false, want true", c)
		}
	}
	for _, c := range []property.Condition{"", "brand-new", "mint"} {
		if c.Valid() {
			t.Errorf("Condition(%q).Valid() = true, want false", c)
		}
	}
}

func TestDirection_Valid(t *testing.T) {
	valid := []property.Direction{
		property.DirectionNorth, property.DirectionSouth, property.DirectionEast, property.DirectionWest,
		property.DirectionNortheast, property.DirectionNorthwest, property.DirectionSoutheast, property.DirectionSouthwest,
	}
	if len(valid) != 8 {
		t.Fatalf("expected 8 compass directions, got %d", len(valid))
	}
	for _, d := range valid {
		if !d.Valid() {
			t.Errorf("Direction(%q).Valid() = false, want true", d)
		}
	}
	for _, d := range []property.Direction{"", "up", "nne"} {
		if d.Valid() {
			t.Errorf("Direction(%q).Valid() = true, want false", d)
		}
	}
}

func TestRoadAccess_Valid(t *testing.T) {
	for _, r := range []property.RoadAccess{property.RoadAccessDirect, property.RoadAccessWalking, property.RoadAccessVehicle} {
		if !r.Valid() {
			t.Errorf("RoadAccess(%q).Valid() = false, want true", r)
		}
	}
	for _, r := range []property.RoadAccess{"", "helicopter", "boat"} {
		if r.Valid() {
			t.Errorf("RoadAccess(%q).Valid() = true, want false", r)
		}
	}
}

// ---- BuildingSpecs.Validate() ranges (SC-M1-05 failure cases) ---------------

func TestBuildingSpecs_Validate(t *testing.T) {
	cases := []struct {
		name    string
		b       property.BuildingSpecs
		wantErr bool
	}{
		{"empty ok", property.BuildingSpecs{}, false},
		{"valid full", property.BuildingSpecs{Floors: ptr(2), FloorLevel: ptr(3), ParkingSpaces: ptr(2), YearBuilt: ptr(2020), LastRenovated: ptr(2023)}, false},
		{"floors zero invalid", property.BuildingSpecs{Floors: ptr(0)}, true},
		{"negative floor level", property.BuildingSpecs{FloorLevel: ptr(-1)}, true},
		{"negative parking", property.BuildingSpecs{ParkingSpaces: ptr(-2)}, true},
		{"year too old", property.BuildingSpecs{YearBuilt: ptr(1700)}, true},
		{"year too future", property.BuildingSpecs{YearBuilt: ptr(3000)}, true},
		{"renovated too old", property.BuildingSpecs{LastRenovated: ptr(1500)}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.b.Validate(); (err != nil) != tc.wantErr {
				t.Errorf("BuildingSpecs.Validate() err=%v, wantErr=%v", err, tc.wantErr)
			}
		})
	}
}

// ---- Policies.Validate() ranges (SC-M1-05 failure cases) ---------------------

func TestPolicies_Validate(t *testing.T) {
	cases := []struct {
		name    string
		p       property.Policies
		wantErr bool
	}{
		{"empty ok", property.Policies{}, false},
		{"valid", property.Policies{Inclusions: []string{"wifi"}, MinimumLeaseMonths: ptr(12), SecurityDepositMonths: ptr(2), AdvancePaymentMonths: ptr(1)}, false},
		{"negative min lease", property.Policies{MinimumLeaseMonths: ptr(-1)}, true},
		{"negative deposit", property.Policies{SecurityDepositMonths: ptr(-1)}, true},
		{"negative advance", property.Policies{AdvancePaymentMonths: ptr(-3)}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.p.Validate(); (err != nil) != tc.wantErr {
				t.Errorf("Policies.Validate() err=%v, wantErr=%v", err, tc.wantErr)
			}
		})
	}
}

// ---- Property.Validate() descends into the new attribute-groups -------------

func TestProperty_Validate_DescendsIntoGroups(t *testing.T) {
	bad := &property.Property{ForSale: true, BuildingSpecs: &property.BuildingSpecs{YearBuilt: ptr(3000)}}
	if err := bad.Validate(); err == nil {
		t.Error("Property.Validate() = nil, want error from invalid BuildingSpecs")
	}
	badPol := &property.Property{ForSale: true, Policies: &property.Policies{MinimumLeaseMonths: ptr(-1)}}
	if err := badPol.Validate(); err == nil {
		t.Error("Property.Validate() = nil, want error from invalid Policies")
	}
	ok := &property.Property{
		ForSale:       true,
		BuildingSpecs: &property.BuildingSpecs{Floors: ptr(2), YearBuilt: ptr(2020)},
		Policies:      &property.Policies{MinimumLeaseMonths: ptr(12)},
	}
	if err := ok.Validate(); err != nil {
		t.Errorf("Property.Validate() with valid groups = %v, want nil", err)
	}
}

// ---- JSON omitempty / round-trip --------------------------------------------

func TestBuildingSpecs_JSONOmitempty(t *testing.T) {
	empty, _ := json.Marshal(property.BuildingSpecs{})
	if string(empty) != "{}" {
		t.Errorf("empty BuildingSpecs JSON = %s, want {}", empty)
	}
	full, _ := json.Marshal(property.BuildingSpecs{Floors: ptr(2), YearBuilt: ptr(2020)})
	for _, want := range []string{"floors", "year_built"} {
		if !strings.Contains(string(full), want) {
			t.Errorf("BuildingSpecs JSON %s missing %q", full, want)
		}
	}
}

func TestPolicies_JSONOmitempty(t *testing.T) {
	empty, _ := json.Marshal(property.Policies{})
	if string(empty) != "{}" {
		t.Errorf("empty Policies JSON = %s, want {}", empty)
	}
	full, _ := json.Marshal(property.Policies{Inclusions: []string{"wifi"}, MinimumLeaseMonths: ptr(12)})
	for _, want := range []string{"inclusions", "minimum_lease_months"} {
		if !strings.Contains(string(full), want) {
			t.Errorf("Policies JSON %s missing %q", full, want)
		}
	}
}

func TestRooms_ExtendedFields_RoundTrip(t *testing.T) {
	in := property.Rooms{DiningRooms: ptr(1), Offices: ptr(1), StorageRooms: ptr(2), MaidRooms: ptr(1), GuestRooms: ptr(1)}
	data, _ := json.Marshal(in)
	var out property.Rooms
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal Rooms: %v", err)
	}
	if out.DiningRooms == nil || out.Offices == nil || out.StorageRooms == nil || out.MaidRooms == nil || out.GuestRooms == nil {
		t.Fatalf("extended Rooms fields lost in round-trip: %s", data)
	}
	if *out.MaidRooms != 1 || *out.StorageRooms != 2 {
		t.Errorf("Rooms round-trip mismatch: %+v", out)
	}
}

func TestSizes_ExtendedFields_RoundTrip(t *testing.T) {
	in := property.Sizes{TotalArea: ptr(420.0), UsableArea: ptr(380.5), BalconyArea: ptr(35.0)}
	data, _ := json.Marshal(in)
	var out property.Sizes
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal Sizes: %v", err)
	}
	if out.TotalArea == nil || out.UsableArea == nil || out.BalconyArea == nil {
		t.Fatalf("extended Sizes fields lost in round-trip: %s", data)
	}
	if *out.TotalArea != 420.0 || *out.BalconyArea != 35.0 {
		t.Errorf("Sizes round-trip mismatch: %+v", out)
	}
}

// ---- Property carries the new scalars ---------------------------------------

func TestProperty_NewScalars_JSON(t *testing.T) {
	p := property.Property{
		ForSale:    true,
		Ownership:  ptr(property.OwnershipLeasehold),
		Condition:  ptr(property.ConditionExcellent),
		Direction:  ptr(property.DirectionSoutheast),
		RoadAccess: ptr(property.RoadAccessVehicle),
		Rank:       property.RankHighest,
	}
	data, _ := json.Marshal(p)
	for _, want := range []string{"ownership_type", "condition", "direction", "road_access", "rank"} {
		if !strings.Contains(string(data), want) {
			t.Errorf("Property JSON missing %q: %s", want, data)
		}
	}
}

// SC-M1-05 (SI 422 path, unit): the service rejects an unknown dictionary code at
// the wire boundary (validateCreate -> validateListingAttrs) BEFORE any DB write.
// Uses a nil repo/pool — validation runs first, so no database is needed.
func TestService_Create_RejectsInvalidDictCode(t *testing.T) {
	svc := property.NewPropertyService(nil, nil, "bestierealestate")

	badCond := property.Condition("brand-new")
	if _, err := svc.Create(context.Background(), property.Property{Title: ptr("X"), ForSale: true, Condition: &badCond}); err == nil {
		t.Error("Create with invalid condition code = nil error, want validation error")
	}

	badDir := property.Direction("up")
	if _, err := svc.Create(context.Background(), property.Property{Title: ptr("X"), ForSale: true, Direction: &badDir}); err == nil {
		t.Error("Create with invalid direction code = nil error, want validation error")
	}

	// topography joins the discriminators (task 2607-001) — an unseeded code must be
	// rejected at the wire boundary (clean 422), not left to the DB FK (500).
	badTopo := property.Topography("steep")
	if _, err := svc.Create(context.Background(), property.Property{Title: ptr("X"), ForSale: true, Topography: &badTopo}); err == nil {
		t.Error("Create with invalid topography code = nil error, want validation error")
	}

	// A valid listing with good codes + specs passes validation (nil repo -> returns the property).
	okCond := property.ConditionExcellent
	if _, err := svc.Create(context.Background(), property.Property{
		Title: ptr("X"), ForSale: true, Condition: &okCond,
		BuildingSpecs: &property.BuildingSpecs{Floors: ptr(2), YearBuilt: ptr(2020)},
	}); err != nil {
		t.Errorf("Create with valid codes/specs = %v, want nil", err)
	}
}

// SC-M1-05 (range 422, unit): out-of-range building/policy values are rejected.
func TestService_Create_RejectsBadRanges(t *testing.T) {
	svc := property.NewPropertyService(nil, nil, "bestierealestate")

	if _, err := svc.Create(context.Background(), property.Property{
		Title: ptr("X"), ForSale: true,
		BuildingSpecs: &property.BuildingSpecs{YearBuilt: ptr(3000)},
	}); err == nil {
		t.Error("Create with year_built=3000 = nil error, want validation error")
	}
	if _, err := svc.Create(context.Background(), property.Property{
		Title: ptr("X"), ForSale: true,
		Policies: &property.Policies{MinimumLeaseMonths: ptr(-1)},
	}); err == nil {
		t.Error("Create with minimum_lease_months=-1 = nil error, want validation error")
	}
}
