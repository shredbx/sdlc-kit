package user

import (
	"context"

	"github.com/google/uuid"
	"github.com/shredbx/sbx-core/pkg/repository"
)

// UserStore persists user entities in Postgres.
type UserStore interface {
	Create(ctx context.Context, input CreateUserInput) (*User, error)
	Get(ctx context.Context, id uuid.UUID) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	// GetByEmailIncludingDeleted returns the user record even when soft-deleted.
	// Used by invite/reactivate flows that need to surface "this email belongs
	// to a deactivated user — reactivate?" instead of opaquely failing on the
	// postgres unique constraint.
	GetByEmailIncludingDeleted(ctx context.Context, email string) (*User, error)
	Update(ctx context.Context, id uuid.UUID, input UpdateUserInput) (*User, error)
	Activate(ctx context.Context, id uuid.UUID, input ActivateUserInput) (*User, error)
	ResetPassword(ctx context.Context, id uuid.UUID, passwordHash string) error
	UpdateLastLogin(ctx context.Context, id uuid.UUID) error
	UpdateRole(ctx context.Context, id uuid.UUID, role string) (*User, error)
	SoftDelete(ctx context.Context, id uuid.UUID) error
	// Reactivate flips a soft-deleted user back to `invited` status, clears the
	// deleted_at tombstone, and resets password_hash so the user must complete
	// the magic-link onboarding flow again. Returns error if the user is not
	// currently in `deleted` state.
	Reactivate(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, opts repository.ListOptions) ([]User, int, error)
	// ListIncludingDeleted returns all users including soft-deleted ones, so
	// the admin UI can surface deactivated rows for reactivation. Filters
	// deleted_at out of the WHERE clause; same pagination contract as List.
	ListIncludingDeleted(ctx context.Context, opts repository.ListOptions) ([]User, int, error)
}
