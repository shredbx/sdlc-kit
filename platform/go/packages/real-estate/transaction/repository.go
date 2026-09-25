package transaction

import (
	"context"

	"github.com/shredbx/sbx-core/pkg/repository"
)

// =============================================================================
// FILTER — domain-shaped List filter (type / status / property)
// =============================================================================
//
// The typed `Fields` descriptors this filter compiles against now live in the
// generated mapper (mapper.generated.go, from the entity.yml postgres block's
// `filterable_fields`) — the MD layer owns that struct so there is exactly one
// declaration. Discriminators are bound as their string form (the column type),
// consistent with how property exposes LifecycleStatus as Field[string].

// TransactionFilter narrows a transaction List by the deal's facet-relevant
// axes. A nil field means "no constraint on that axis". It compiles to a
// repository.Query via ToQuery (so the MD/storage layer stays generic).
//
// Limit / Offset carry pagination through to the storage layer's
// repository.ListOptions. They are storage-agnostic: the YAMLStore slices its
// result and the Postgres store emits LIMIT/OFFSET — both leave `total` (the
// full filtered count) untouched, so a paged list never shrinks its reported
// total. A zero Limit means "no cap" (the SI/handler boundary applies the page
// default); a non-positive Offset is treated as 0.
type TransactionFilter struct {
	Type       *TransactionType
	Status     *TransactionStatus
	PropertyID *string
	UnitID     *string
	Limit      int
	Offset     int
}

// ToQuery renders the filter as a storage-agnostic repository.Query (AND of the
// set axes). An empty filter yields an empty Query that matches everything.
func (f TransactionFilter) ToQuery() repository.Query {
	preds := make([]repository.Predicate, 0, 4)
	if f.Type != nil {
		preds = append(preds, Fields.Type.Eq(string(*f.Type)))
	}
	if f.Status != nil {
		preds = append(preds, Fields.Status.Eq(string(*f.Status)))
	}
	if f.PropertyID != nil {
		preds = append(preds, Fields.PropertyID.Eq(*f.PropertyID))
	}
	if f.UnitID != nil {
		preds = append(preds, Fields.UnitID.Eq(*f.UnitID))
	}
	return repository.And(preds...)
}

// =============================================================================
// REPOSITORY — interface the MD layer implements (defined BEFORE implementation)
// =============================================================================

// TransactionRepository is the storage interface for deals. It composes the
// generic CRUD (repository.Repository[Transaction] — Get/List/Create/Update/
// Delete, reused as-is) and ADDS the deal-specific domain operations. The MD
// layer implements it with an atomic Close that updates the deal AND
// property.lifecycle_status in one db transaction.
//
// This is an interface ONLY — no implementation lives in the PD layer.
type TransactionRepository interface {
	repository.Repository[Transaction]

	// Close completes a deal under closedAs and drives the subject property's
	// lifecycle_status (sold|leased) atomically. Returns ErrAlreadyClosed if the
	// deal is already closed, ErrFacetNotInType per the close-facet resolution
	// rules.
	Close(ctx context.Context, id string, closedAs Facet) (Transaction, error)

	// Cancel abandons an open deal. The subject property's lifecycle is
	// untouched. Returns ErrInvalidTransition if the deal is not open.
	Cancel(ctx context.Context, id string) error

	// AddParty attaches a party (role + contact) to a deal.
	AddParty(ctx context.Context, txID string, party TransactionParty) error

	// RemoveParty detaches a party from a deal by party id.
	RemoveParty(ctx context.Context, txID string, partyID string) error
}
