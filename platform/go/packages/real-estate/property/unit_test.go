package property_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/shredbx/sbx-core/pkg/money"
	"github.com/shredbx/sbx-core/pkg/property"
)

// =============================================================================
// PROPERTY UNIT (M2, task 2606-001) — PD + companion-contract tests.
// PropertyUnit is a 1:many cascade-delete child of Property with its OWN UUID
// identity (Decision #0019), mirroring transaction_party. Structurally Priced
// (sale/lease satang + currency) · Publishable (is_active gate) · Bindable
// (canvas Source). No DB persistence here (that is MD) — companion methods are
// exercised in fixture mode (nil pool) for their validation/guard contract only.
// =============================================================================

// ---- PropertyUnit.Validate() ------------------------------------------------

func TestPropertyUnit_Validate(t *testing.T) {
	cases := []struct {
		name    string
		u       property.PropertyUnit
		wantErr bool
	}{
		{"valid minimal", property.PropertyUnit{Title: "Unit A2"}, false},
		{"valid full", property.PropertyUnit{Title: "Unit A2", SalePrice: ptr(int64(500000000)), LeasePrice: ptr(int64(2500000)), PriceCurrency: ptr(money.CurrencyCode("THB"))}, false},
		{"empty title", property.PropertyUnit{Title: ""}, true},
		{"whitespace title", property.PropertyUnit{Title: "   "}, true},
		{"negative sale price", property.PropertyUnit{Title: "U", SalePrice: ptr(int64(-1))}, true},
		{"negative lease price", property.PropertyUnit{Title: "U", LeasePrice: ptr(int64(-1))}, true},
		{"unknown currency", property.PropertyUnit{Title: "U", PriceCurrency: ptr(money.CurrencyCode("THBB"))}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.u.Validate(); (err != nil) != tc.wantErr {
				t.Errorf("PropertyUnit.Validate() err=%v, wantErr=%v", err, tc.wantErr)
			}
		})
	}
}

// ---- Property.Validate() descends into Units --------------------------------

func TestProperty_Validate_DescendsIntoUnits(t *testing.T) {
	bad := &property.Property{ForSale: true, Units: []property.PropertyUnit{{Title: ""}}}
	if err := bad.Validate(); err == nil {
		t.Error("Property.Validate() = nil, want error from invalid Unit")
	}
	ok := &property.Property{ForSale: true, Units: []property.PropertyUnit{{Title: "Unit A2"}}}
	if err := ok.Validate(); err != nil {
		t.Errorf("Property.Validate() with valid units = %v, want nil", err)
	}
}

// ---- JSON omitempty ---------------------------------------------------------

func TestPropertyUnit_JSONOmitempty(t *testing.T) {
	empty, _ := json.Marshal(property.PropertyUnit{})
	for _, absent := range []string{"description", "cover_image_id", "sale_price", "lease_price", "price_currency"} {
		if strings.Contains(string(empty), absent) {
			t.Errorf("empty PropertyUnit JSON = %s, should omit %q", empty, absent)
		}
	}
	full, _ := json.Marshal(property.PropertyUnit{Title: "U", SalePrice: ptr(int64(1)), PriceCurrency: ptr(money.CurrencyCode("THB"))})
	for _, want := range []string{"title", "sale_price", "price_currency", "is_active"} {
		if !strings.Contains(string(full), want) {
			t.Errorf("PropertyUnit JSON %s missing %q", full, want)
		}
	}
}

func TestProperty_Units_JSON(t *testing.T) {
	p := property.Property{ForSale: true, Units: []property.PropertyUnit{{Title: "Unit A2"}}}
	data, _ := json.Marshal(p)
	if !strings.Contains(string(data), "units") {
		t.Errorf("Property JSON missing units: %s", data)
	}
	empty, _ := json.Marshal(property.Property{ForSale: true})
	if strings.Contains(string(empty), "units") {
		t.Errorf("empty Property JSON should omit units: %s", empty)
	}
}

// ---- Service companion contract (fixture mode, nil pool) --------------------

// SaveUnit validates BEFORE any DB work, so the validation gate is exercised
// with a nil pool (mirrors how Create validates before touching the repo).
func TestService_SaveUnit_RejectsInvalid(t *testing.T) {
	svc := property.NewPropertyService(nil, nil, "bestierealestate")
	if _, err := svc.SaveUnit(context.Background(), property.PropertyUnit{Title: ""}); err == nil {
		t.Error("SaveUnit with empty title = nil error, want validation error")
	}
}

// LoadUnits with a nil pool yields no rows (fixture mode) and never panics.
func TestService_LoadUnits_NilPool(t *testing.T) {
	svc := property.NewPropertyService(nil, nil, "bestierealestate")
	units, err := svc.LoadUnits(context.Background(), "some-id")
	if err != nil {
		t.Errorf("LoadUnits nil pool err = %v, want nil", err)
	}
	if units != nil {
		t.Errorf("LoadUnits nil pool = %v, want nil", units)
	}
}

// DeleteUnit with a nil pool is a no-op (fixture mode) and never panics.
func TestService_DeleteUnit_NilPool(t *testing.T) {
	svc := property.NewPropertyService(nil, nil, "bestierealestate")
	if err := svc.DeleteUnit(context.Background(), "some-unit-id"); err != nil {
		t.Errorf("DeleteUnit nil pool err = %v, want nil", err)
	}
}

// ActiveUnits is the Publishable gate the public surfaces apply: only active units,
// in input order; an inactive unit must NEVER survive (no public leak). Pure — this
// locks the gate without a DB (the handler that calls it needs a pool to load rows).
func TestActiveUnits(t *testing.T) {
	in := []property.PropertyUnit{
		{Title: "A", IsActive: true, SortOrder: 0},
		{Title: "B", IsActive: false, SortOrder: 1},
		{Title: "C", IsActive: true, SortOrder: 2},
	}
	got := property.ActiveUnits(in)
	if len(got) != 2 {
		t.Fatalf("ActiveUnits len = %d, want 2 (B is inactive)", len(got))
	}
	if got[0].Title != "A" || got[1].Title != "C" {
		t.Errorf("ActiveUnits = [%s, %s], want [A, C] (order preserved, inactive dropped)", got[0].Title, got[1].Title)
	}
	for _, u := range got {
		if !u.IsActive {
			t.Errorf("ActiveUnits leaked an inactive unit: %s", u.Title)
		}
	}
	if property.ActiveUnits(nil) == nil {
		t.Error("ActiveUnits(nil) should be a non-nil empty slice")
	}
}

// ---- Bindable surface (A0+A6, task 2606-004) -------------------------------

// PropertyUnit is a structural Bindable Source: it reports (BindKind, BindID) so
// Calendar and Canvas can reference ONE unit by the same (kind, id) pair they use
// for a whole property — under a DISTINCT kind "property-unit" (vs a property's
// "property") so a binding resolves to the single unit, not the parent. This locks
// the promise unit.go's own doc comment already makes.
func TestPropertyUnit_BindableSurface(t *testing.T) {
	u := property.PropertyUnit{ID: "11111111-1111-1111-1111-111111111111", Title: "Villa A2"}

	if got := u.BindKind(); got != "property-unit" {
		t.Errorf("BindKind() = %q, want %q", got, "property-unit")
	}
	if got := u.BindKind(); got != property.SourceKindPropertyUnit {
		t.Errorf("BindKind() = %q, want the SourceKindPropertyUnit constant %q", got, property.SourceKindPropertyUnit)
	}
	if got := u.BindID(); got != u.ID {
		t.Errorf("BindID() = %q, want the unit UUID %q", got, u.ID)
	}
	if property.SourceKindPropertyUnit == "property" {
		t.Error(`SourceKindPropertyUnit must differ from the whole-property kind "property"`)
	}
}

// SourceKindPropertyUnit is an UNTYPED string constant: it assigns to a bare string
// field (Reference.RefType / SourceRef.kind) with no conversion — the compile-time
// guard that the Foundation kind carries no Office-module type and so forces no
// upward import. A regression here is a BUILD error, not a test failure.
var _ string = property.SourceKindPropertyUnit

// ---- Disabled state + CalculableUnits (Decision #0316, task 2606-121) -------

// CalculableUnits is the listing-status MATH set (Decision #0316): every unit a
// manager has NOT disabled, regardless of deal state. A disabled unit is excluded
// from availability math entirely (as if it does not exist) — DISTINCT from a
// deal-closed (is_active=false) unit, which still counts toward the math as a
// closed unit. This is the set resolveListingStatus() reasons over.
func TestCalculableUnits(t *testing.T) {
	in := []property.PropertyUnit{
		{Title: "A", IsActive: true, Disabled: false},  // open — counts
		{Title: "B", IsActive: false, Disabled: false}, // deal-closed — counts (as a closed unit)
		{Title: "C", IsActive: true, Disabled: true},   // manager-hidden — excluded entirely
	}
	got := property.CalculableUnits(in)
	if len(got) != 2 {
		t.Fatalf("CalculableUnits len = %d, want 2 (C is disabled)", len(got))
	}
	if got[0].Title != "A" || got[1].Title != "B" {
		t.Errorf("CalculableUnits = [%s, %s], want [A, B] (disabled C excluded, order preserved)", got[0].Title, got[1].Title)
	}
	for _, u := range got {
		if u.Disabled {
			t.Errorf("CalculableUnits leaked a disabled unit: %s", u.Title)
		}
	}
	if property.CalculableUnits(nil) == nil {
		t.Error("CalculableUnits(nil) should be a non-nil empty slice")
	}
}

// ActiveUnits (the public display gate) must ALSO exclude disabled units (Decision
// #0316): a manager-hidden unit must NEVER leave the server on a public payload,
// even though it remains is_active. Public gate = is_active AND NOT disabled.
func TestActiveUnits_ExcludesDisabled(t *testing.T) {
	in := []property.PropertyUnit{
		{Title: "A", IsActive: true, Disabled: false},  // shown
		{Title: "B", IsActive: true, Disabled: true},   // manager-hidden — must NOT show
		{Title: "C", IsActive: false, Disabled: false}, // deal-closed — not shown (is_active gate)
	}
	got := property.ActiveUnits(in)
	if len(got) != 1 || got[0].Title != "A" {
		t.Fatalf("ActiveUnits = %v, want [A] only (B disabled, C inactive)", got)
	}
	for _, u := range got {
		if u.Disabled {
			t.Errorf("ActiveUnits leaked a disabled unit publicly: %s", u.Title)
		}
	}
}
