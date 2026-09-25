package transaction

import (
	"time"

	"github.com/shredbx/sbx-core/pkg/money"
)

// =============================================================================
// TRANSACTION — deal entity (containment root; cascade-deletes its parties)
// =============================================================================

// Transaction is a property deal, recorded and progressed to close.
//
// Money fields are minor-units + currency (pkg/money) — never float64; each
// persists as two scalar columns (amount + currency), so there is no array
// codegen gap. Facet-scoped fields (Sale*/Lease*) carry only when that facet is
// part of the deal type.
//
// Nullable timestamps/dates are pointers (nil = not yet set), following the
// property entity idiom. Load-bearing fields stay typed columns (NOT EAV); a
// generic JSONB "data" bag is deliberately omitted (YAGNI) — add a transaction-
// local typed struct if/when a concrete extension need lands.
type Transaction struct {
	ID string `json:"id" yaml:"id"`

	// Type is the deal kind (required). Status defaults to open.
	Type   TransactionType   `json:"type" yaml:"type"`
	Status TransactionStatus `json:"status" yaml:"status"`

	// ClosedAs records which facet a deal closed under; it drives
	// property.lifecycle_status. nil while open, or for a not-yet-closed deal.
	ClosedAs *Facet `json:"closed_as,omitempty" yaml:"closed_as,omitempty"`

	// PropertyID is the subject of the deal (required; references property).
	PropertyID string `json:"property_id" yaml:"property_id"`

	// UnitID optionally narrows the deal to ONE unit of the property (nil =
	// whole-property deal). A deal targets exactly one closeable subject — the
	// property OR one unit, never several. A unit-scoped close deactivates that
	// unit; the property lifecycle is untouched. Set at creation, preserved across
	// terms edits (like PropertyID). References property_units(id), ON DELETE SET NULL.
	UnitID *string `json:"unit_id,omitempty" yaml:"unit_id,omitempty"`

	// Sale facet fields.
	SalePrice money.Money `json:"sale_price,omitempty" yaml:"sale_price,omitempty"`

	// Lease facet fields.
	LeaseRent    money.Money `json:"lease_rent,omitempty" yaml:"lease_rent,omitempty"`
	LeaseDeposit money.Money `json:"lease_deposit,omitempty" yaml:"lease_deposit,omitempty"`
	LeaseTerm    Term        `json:"lease_duration,omitempty" yaml:"lease_duration,omitempty"`
	LeaseStart   *time.Time  `json:"lease_start,omitempty" yaml:"lease_start,omitempty"`
	LeaseEnd     *time.Time  `json:"lease_end,omitempty" yaml:"lease_end,omitempty"`

	// Timestamps.
	OpenedAt time.Time  `json:"opened_at" yaml:"opened_at"`
	ClosedAt *time.Time `json:"closed_at,omitempty" yaml:"closed_at,omitempty"`

	Notes *string `json:"notes,omitempty" yaml:"notes,omitempty"`
}

// TransactionParty is one person's role in a deal (parent = transaction, PK=FK,
// no independent lifecycle — cascade-deletes with the deal).
type TransactionParty struct {
	ID            string    `json:"id" yaml:"id"`
	TransactionID string    `json:"transaction_id" yaml:"transaction_id"`
	Role          PartyRole `json:"role" yaml:"role"`

	// ContactID references the person (G12 Contact Book; required once wired).
	ContactID string `json:"contact_id" yaml:"contact_id"`
}

// =============================================================================
// PURE TRANSITION / CLOSE / LIFECYCLE LOGIC (storage-agnostic)
// =============================================================================

// CanTransition reports whether a deal may move from status `from` to `to`.
// Allowed: open->closed, open->cancelled, and cancelled->open (REOPEN — Decision
// #0314). CLOSED is TERMINAL (no closed->open / closed->cancelled). A no-op
// (from == to) is not a transition.
//
// The asymmetry is principled, not arbitrary: a CLOSE flips the subject property's
// lifecycle to sold/leased (Decision #0313), so reopening a closed deal would have
// to un-sell a published property — semantically wrong and a cross-entity hazard,
// hence closed stays terminal. A CANCEL touches NO property state (the deal's
// parties/terms survive), so reopening is a pure, self-contained status flip with
// nothing downstream to reconcile. This supersedes the cancel-terminal half of
// SC-023-04; SC-022 (closed terminal) is unchanged and reaffirmed.
func CanTransition(from, to TransactionStatus) bool {
	switch from {
	case StatusOpen:
		return to == StatusClosed || to == StatusCancelled
	case StatusCancelled:
		return to == StatusOpen
	default:
		return false
	}
}

// ResolveCloseFacet determines which facet a close runs under. Every deal type
// composes exactly ONE facet, so the facet is implied: `picked` may be empty
// (and if given, must match the implied facet — else ErrFacetNotInType).
//
// An unknown deal type returns ErrTypeRequired (no facets to resolve).
func ResolveCloseFacet(t TransactionType, picked Facet) (Facet, error) {
	facets := FacetsFor(t)
	if len(facets) == 0 {
		// Unknown / unmapped type — nothing to close under.
		return "", ErrTypeRequired
	}
	implied := facets[0]
	if picked != "" && picked != implied {
		return "", ErrFacetNotInType
	}
	return implied, nil
}

// OutcomeFor maps a closed facet to the deal Outcome it drives: sale -> sold,
// lease -> leased (SC-023-01, SC-023-02). Outcome is engine-local — the engine
// never imports a consumer's property package; the consumer maps Outcome to its
// own property lifecycle status (BR: sold/leased). An unknown facet maps to the
// empty Outcome (callers treat "" as "do not change").
func OutcomeFor(f Facet) Outcome {
	switch f {
	case FacetSale:
		return OutcomeSold
	case FacetLease:
		return OutcomeLeased
	}
	return ""
}
