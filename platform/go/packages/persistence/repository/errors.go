package repository

import (
	"errors"
	"fmt"
)

// Domain errors — storage-agnostic. Backends wrap storage-specific errors
// into these sentinel errors so callers can use errors.Is().
var (
	// ErrNotFound indicates the requested entity does not exist.
	ErrNotFound = errors.New("not found")

	// ErrConflict indicates a unique constraint violation (duplicate key).
	ErrConflict = errors.New("conflict")

	// ErrValidation indicates the entity failed validation before persistence.
	ErrValidation = errors.New("validation failed")
)

// NotFoundError wraps ErrNotFound with entity context.
type NotFoundError struct {
	Entity string
	ID     string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("%s with id %q not found", e.Entity, e.ID)
}

func (e *NotFoundError) Unwrap() error {
	return ErrNotFound
}

// NewNotFoundError creates a NotFoundError for the given entity and ID.
func NewNotFoundError(entity, id string) error {
	return &NotFoundError{Entity: entity, ID: id}
}
