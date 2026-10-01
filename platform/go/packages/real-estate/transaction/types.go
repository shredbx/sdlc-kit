// Package transaction provides the shared, storage-agnostic problem-domain (PD)
// layer of the property deal engine (G13).
//
// A deal (sale | lease) is recorded and progressed
// to close, where close drives property.lifecycle_status (sold | leased).
// bestierealestate is the first consumer; a future Bestays adds a Rent facet with
// zero engine change (facet-composition, governance rule #9 / GoF strategy).
//
// Architecture (mirrors pkg/property — a self-contained type package that owns
// both its domain logic AND its own persistence):
//
//	PD layer: named-type enums, Transaction/TransactionParty structs, pure
//	    transition/close/lifecycle logic (CanTransition, ResolveCloseFacet,
//	    OutcomeFor), Validate, and the TransactionRepository interface. The
//	    domain TYPES + LOGIC depend on NO consumer package and NO datastore.
//	MD layer (this package): a YAML (fixture) store for DB-free development and a
//	    generated Postgres mapper (mapper.generated.go, via `sbx generate mapper`)
//	    plus the TransactionService that composes a repository.Repository[Transaction].
//	    The mapper pulls in database/sql — exactly as pkg/property's does — so the
//	    PACKAGE provides persistence, while the domain layer stays portable.
//
// Decoupling invariant (load-bearing): this package imports NO consumer domain
// package (in particular NOT pkg/property). Close mutates only the DEAL (status,
// closed_as, closed_at) and returns an engine-local Outcome (sold | leased); the
// consumer's SI layer maps that Outcome to its own property.lifecycle_status and
// performs the atomic deal+property write. The engine never reaches into property.
//
// Discriminators are named types (never raw string), mirroring property.go:
//
//	TransactionType, TransactionStatus, Facet, PartyRole
//
// Type codes are CODE-managed (each carries facet behaviour) — they live as a
// typed enum here, and consumers ENABLE a subset via EnabledSet config (NOT a
// dictionary table; Decision: transaction-type = code enum). All money is
// minor-units + currency via pkg/money — never float64.
package transaction

// =============================================================================
// NAMED TYPES — discriminators (never raw string), mirroring property.go
// =============================================================================

// TransactionType is the built-in deal kind. Each code carries facet behaviour
// (see FacetsFor), so types are a code enum — not a dictionary FK.
type TransactionType string

const (
	// TypeSale is a freehold sale — composes the Sale facet only.
	TypeSale TransactionType = "sale"
	// TypeLease is a lease — composes the Lease facet only.
	TypeLease TransactionType = "lease"
	// TypeRent — deferred (Bestays consumer); additive, zero engine change.
)

// TransactionStatus is where a deal sits in its lifecycle.
type TransactionStatus string

const (
	// StatusOpen is the initial, in-progress state.
	StatusOpen TransactionStatus = "open"
	// StatusClosed is a completed deal — drives property.lifecycle_status.
	StatusClosed TransactionStatus = "closed"
	// StatusCancelled is an abandoned deal — terminal, lifecycle untouched.
	StatusCancelled TransactionStatus = "cancelled"
)

// Facet is a behaviour group the engine branches on. A new variant (Rent for
// Bestays) is a new facet, zero engine change.
type Facet string

const (
	// FacetSale is the sale behaviour group; closing under it marks the
	// property sold.
	FacetSale Facet = "sale"
	// FacetLease is the lease behaviour group; closing under it marks the
	// property leased.
	FacetLease Facet = "lease"
	// FacetRent — deferred (Bestays); additive.
)

// Outcome is the deal result a close drives. It is engine-local so the engine
// never imports a consumer's property package (keeping this PD layer storage-
// agnostic and reusable across consumers); each consumer maps an Outcome to its
// own property lifecycle status (BR: sold/leased). See OutcomeFor.
type Outcome string

const (
	// OutcomeSold is the result of closing under the Sale facet.
	OutcomeSold Outcome = "sold"
	// OutcomeLeased is the result of closing under the Lease facet.
	OutcomeLeased Outcome = "leased"
)

// PartyRole is one person's role in a deal.
type PartyRole string

const (
	// RoleBuyer is the buyer in a sale.
	RoleBuyer PartyRole = "buyer"
	// RoleSeller is the seller in a sale.
	RoleSeller PartyRole = "seller"
	// RoleLandlord is the landlord in a lease.
	RoleLandlord PartyRole = "landlord"
	// RoleTenant is the tenant in a lease.
	RoleTenant PartyRole = "tenant"
)

// ValidPartyRole reports whether code matches one of the known party roles.
// Used as a parse-time guard; mirrors property.ValidLifecycleStatus.
func ValidPartyRole(code PartyRole) bool {
	switch code {
	case RoleBuyer, RoleSeller, RoleLandlord, RoleTenant:
		return true
	}
	return false
}

// Term is a lease duration in whole months. A small named type keeps the unit
// explicit in signatures (never a bare int).
type Term uint16

// Months constructs a Term from a month count.
func Months(n uint16) Term {
	return Term(n)
}
