package property

import (
	"errors"
	"strings"
	"time"

	"github.com/shredbx/sbx-core/pkg/money"
)

// =============================================================================
// PROPERTY UNIT — 1:many containment child (Decision #0019)
// =============================================================================

// ErrUnitTitleRequired indicates a PropertyUnit was saved without a title. Title
// is the unit's human label and is UNIQUE within its parent property.
var ErrUnitTitleRequired = errors.New("property unit title is required")

// ErrInvalidCurrency indicates a PriceCurrency was set to a code the governed
// currency registry does not know (after normalization). Use nil to inherit the
// parent property's currency.
var ErrInvalidCurrency = errors.New("price currency must be a known ISO-4217 code when set")

// PropertyUnit is a sellable/leasable sub-listing within a property — a villa in
// a development, a condo unit in a building, a plot in a subdivision. It is a
// containment child (parent = property): own UUID identity, 1:many, cascade-
// deletes with the parent (FK ON DELETE CASCADE). Mirrors the transaction_party
// pattern — a sibling struct + own table + companion methods, NOT the generated
// mapper (codegen cannot emit a 1:many child) and NOT a formal standalone entity.
//
// Structural protocol conformance (no Go interface to implement):
//   - Priced:      SalePrice / LeasePrice (satang, *int64) + PriceCurrency.
//   - Publishable: IsActive gate + stable id (addressable on public surfaces).
//   - Bindable:    registered as a canvas Source provider (kind: property-unit).
//
// Cover + gallery images are pkg/image assets with owner_id = the unit id (no
// new image table) — CoverImageID is the FK to images(id) for the unit cover.
type PropertyUnit struct {
	ID         string `json:"id" yaml:"id"`
	PropertyID string `json:"property_id" yaml:"property_id"`

	// Title is the unit label (required); unique within the parent property.
	Title string `json:"title" yaml:"title"`

	Description  *string `json:"description,omitempty" yaml:"description,omitempty"`
	CoverImageID *string `json:"cover_image_id,omitempty" yaml:"cover_image_id,omitempty"`

	// Per-mode prices in satang (THB × 100). nil = "Price on Application".
	SalePrice  *int64 `json:"sale_price,omitempty" yaml:"sale_price,omitempty"`
	LeasePrice *int64 `json:"lease_price,omitempty" yaml:"lease_price,omitempty"`

	// PriceCurrency applies to both unit prices; nil inherits the parent property's.
	PriceCurrency *money.CurrencyCode `json:"price_currency,omitempty" yaml:"price_currency,omitempty"`

	// IsActive is the deal-closed gate (Decision #0316), system-owned: the deal
	// orchestrator sets it false when a unit's deal closes. Together with Disabled
	// it forms the public Publishable gate (active AND not disabled). Behavioral
	// (always present), defaults TRUE at the DB layer.
	IsActive bool `json:"is_active" yaml:"is_active"`

	// Disabled is the manager "hide from listing" flag (Decision #0316),
	// manager-owned: a disabled unit is excluded from ALL availability/transaction
	// math (as if it does not exist) AND never leaves the server on a public
	// payload. DISTINCT from IsActive — a unit may be active yet disabled
	// (manager-hidden) or inactive yet not disabled (deal-closed). Behavioral
	// (always present), defaults FALSE at the DB layer.
	Disabled bool `json:"disabled" yaml:"disabled"`

	// SortOrder is the ascending display order within the property.
	SortOrder int `json:"sort_order" yaml:"sort_order"`

	// Sizes / Rooms mirror the parent Property's attribute-groups (same *Sizes / *Rooms
	// types, same fields, same json/yaml tags) so a multi-house listing's units each carry
	// their OWN structured size + room data. Unlike Property (which stores these in the
	// property_sizes / property_rooms sub-tables via the generated mapper LEFT JOIN), the
	// unit flattens both groups into FLAT inline columns on property_units, round-tripped by
	// the LoadUnits / SaveUnit companion methods. nil = no data (mirrors the parent contract).
	Sizes *Sizes `json:"sizes,omitempty" yaml:"sizes,omitempty"`
	Rooms *Rooms `json:"rooms,omitempty" yaml:"rooms,omitempty"`

	CreatedAt time.Time `json:"created_at" yaml:"created_at"`
	UpdatedAt time.Time `json:"updated_at" yaml:"updated_at"`
}

// SourceKindPropertyUnit is the Bindable Source-kind code a PropertyUnit reports
// via BindKind() — the "kind" half of the (kind, id) pair other modules (Canvas,
// Calendar, pickers) use to reference ONE unit, distinct from a whole property's
// "property" kind. A plain string (NOT calendar.SourceKind): the property package
// is Foundation and must not import the Office/calendar module, and every consumer
// (Reference.RefType, SourceRef.kind) treats the kind as an open string anyway.
const SourceKindPropertyUnit = "property-unit"

// BindKind reports the unit's Bindable Source kind. Part of the structural Bindable
// surface (FI-1 / FI-2): a PropertyUnit IS a Source identified by (BindKind, BindID),
// exactly as calendar.Event conforms. No interface is implemented — conformance is
// the method set — so the return is a plain string, keeping Foundation import-free.
func (u PropertyUnit) BindKind() string { return SourceKindPropertyUnit }

// BindID reports the unit's stable Source identity (its UUID). Part of the
// structural Bindable surface; survives in a consumer's cached reference label.
func (u PropertyUnit) BindID() string { return u.ID }

// Validate enforces the unit invariants: a non-empty title, non-negative prices
// (nil = POA), and a registry-known currency when set. A nil receiver is valid
// (no unit).
func (u *PropertyUnit) Validate() error {
	if u == nil {
		return nil
	}
	if strings.TrimSpace(u.Title) == "" {
		return ErrUnitTitleRequired
	}
	if u.SalePrice != nil && *u.SalePrice < 0 {
		return ErrNegativePrice
	}
	if u.LeasePrice != nil && *u.LeasePrice < 0 {
		return ErrNegativePrice
	}
	if u.PriceCurrency != nil && !u.PriceCurrency.Valid() {
		return ErrInvalidCurrency
	}
	return nil
}

// ActiveUnits returns the units that render on PUBLIC surfaces, preserving input
// order. It is the Publishable gate: a unit shows only when it is active AND not
// disabled (Decision #0316) — an inactive (deal-closed) OR a disabled
// (manager-hidden) unit must NEVER leave the server on a public payload. A
// nil/empty input yields a non-nil empty slice. Kept here (domain layer) so the
// gate is one tested function reused by every public consumer, never
// re-implemented (and re-bugged) per handler.
func ActiveUnits(units []PropertyUnit) []PropertyUnit {
	active := make([]PropertyUnit, 0, len(units))
	for _, u := range units {
		if u.IsActive && !u.Disabled {
			active = append(active, u)
		}
	}
	return active
}

// CalculableUnits is the listing-status MATH set (Decision #0316): every unit a
// manager has NOT disabled, preserving input order — regardless of deal state. A
// disabled unit is excluded from availability math entirely (as if it does not
// exist); a deal-closed (IsActive=false) unit still counts toward the math as a
// closed unit. This is the set resolveListingStatus reasons over (open / partial
// / closed). A nil/empty input yields a non-nil empty slice.
func CalculableUnits(units []PropertyUnit) []PropertyUnit {
	calc := make([]PropertyUnit, 0, len(units))
	for _, u := range units {
		if !u.Disabled {
			calc = append(calc, u)
		}
	}
	return calc
}
