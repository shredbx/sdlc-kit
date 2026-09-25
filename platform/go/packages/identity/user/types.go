// Package user manages the User entity — identity, profile, credentials, role.
package user

import (
	"time"

	"github.com/google/uuid"
)

// UserStatus represents the lifecycle state of a user account.
type UserStatus string

const (
	UserStatusInvited UserStatus = "invited"
	UserStatusActive  UserStatus = "active"
	UserStatusDeleted UserStatus = "deleted"
)

// User is the aggregate root — core identity entity.
type User struct {
	ID           uuid.UUID  `json:"id"`
	Email        string     `json:"email"`
	FullName     string     `json:"full_name"`
	PasswordHash string     `json:"-"`
	Role         string     `json:"role"`
	Status       UserStatus `json:"status"`
	AvatarURL    *string    `json:"avatar_url,omitempty"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	DeletedAt    *time.Time `json:"deleted_at,omitempty"`
}

// IsActive returns true if the user can authenticate.
func (u *User) IsActive() bool {
	return u.Status == UserStatusActive && u.DeletedAt == nil
}

// CreateUserInput is the input for admin-initiated user creation.
type CreateUserInput struct {
	Email    string `json:"email"`
	FullName string `json:"full_name"`
	Role     string `json:"role"`
}

// UpdateUserInput is the input for profile updates (partial — pointer fields).
type UpdateUserInput struct {
	FullName  *string `json:"full_name,omitempty"`
	AvatarURL *string `json:"avatar_url,omitempty"`
}

// ActivateUserInput is the input for magic link activation.
type ActivateUserInput struct {
	PasswordHash string `json:"-"`
	FullName     string `json:"full_name"`
}

// Sentinel errors for user operations.
var (
	ErrUserNotFound      = &UserError{Code: "user_not_found", Message: "User not found"}
	ErrEmailExists       = &UserError{Code: "email_exists", Message: "Email already exists"}
	ErrInvalidEmail      = &UserError{Code: "invalid_email", Message: "Invalid email format"}
	ErrEmailTooLong      = &UserError{Code: "email_too_long", Message: "Email exceeds maximum length"}
	ErrUserAlreadyActive = &UserError{Code: "user_already_active", Message: "User is already active"}
	ErrUserNotActive     = &UserError{Code: "user_not_active", Message: "User is not active"}
)

// UserError is a typed error for user operations.
type UserError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *UserError) Error() string {
	return e.Message
}
