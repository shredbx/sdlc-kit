// Package repository provides a storage-agnostic generic CRUD interface.
//
// The Repository[M] interface defines standard CRUD operations that any
// storage backend can implement. Domain code depends only on this package
// (PD layer) — it never imports database drivers or storage-specific types.
//
// Architecture:
//
//	PD Layer (this package): Repository[M], Field[T], Query, ListOptions
//	MD Layer (postgres/):    PostgresStore[M], Mapper[M], QueryBuilder
//
// Usage:
//
//	// Domain code — zero storage imports
//	func ListPublished(ctx context.Context, repo repository.Repository[Property]) ([]Property, error) {
//	    opts := repository.ListOptions{
//	        Filter: repository.And(PropertyFields.IsPublished.Eq(true)),
//	        Sort:   []repository.SortField{{Field: "created_at", Desc: true}},
//	        Limit:  20,
//	    }
//	    items, _, err := repo.List(ctx, opts)
//	    return items, err
//	}
package repository

import "context"

// Repository is the generic storage-agnostic CRUD interface.
// M is the domain model type — it has no storage constraints.
type Repository[M any] interface {
	// Get retrieves a single entity by its ID.
	// Returns ErrNotFound if the entity does not exist.
	Get(ctx context.Context, id string) (M, error)

	// List retrieves entities matching the given options.
	// Returns the matching items and total count (for pagination).
	List(ctx context.Context, opts ListOptions) ([]M, int, error)

	// Create inserts a new entity and returns it with generated fields
	// (ID, timestamps). Returns ErrConflict on unique constraint violation.
	Create(ctx context.Context, model M) (M, error)

	// Update modifies an existing entity by ID and returns the updated version.
	// Returns ErrNotFound if the entity does not exist.
	Update(ctx context.Context, id string, model M) (M, error)

	// Delete removes an entity by ID.
	// Returns ErrNotFound if the entity does not exist.
	Delete(ctx context.Context, id string) error
}
