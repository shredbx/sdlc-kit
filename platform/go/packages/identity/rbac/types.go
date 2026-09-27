// Package rbac implements Role-Based Access Control with static permission maps.
// Decision #0172: permissions are compile-time constants, not database-backed.
// The Authorizer interface is pure logic (PD layer) — no I/O, no database.
package rbac

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// Permission represents a resource:action pair (e.g., "user:invite").
type Permission string

// RoleCode identifies a role (e.g., "super-admin", "admin", "team-member").
type RoleCode string

// PermissionSet is a set of permissions for O(1) lookup.
type PermissionSet map[Permission]struct{}

// Has checks if the permission is in the set.
func (ps PermissionSet) Has(p Permission) bool {
	_, ok := ps[p]
	return ok
}

// RBACConfig holds project-scoped role-permission configuration.
type RBACConfig struct {
	Roles         map[RoleCode]PermissionSet `json:"roles"`
	DefaultRole   RoleCode                   `json:"default_role"`
	RoleHierarchy []RoleCode                 `json:"role_hierarchy"`
}

// RoleFromContext extracts the role from request context.
// Injected by the caller to avoid importing the auth package.
type RoleFromContext func(ctx context.Context) (RoleCode, bool)

// Authorizer checks permissions for roles. Pure logic, goroutine-safe.
type Authorizer interface {
	HasPermission(role RoleCode, perm Permission) bool
	GetPermissions(role RoleCode) PermissionSet
	ValidateRole(role RoleCode) error
	CheckOwnership(role RoleCode, ownPerm, anyPerm Permission, resourceOwnerID, requestorID uuid.UUID) bool
	AvailableRoles() []RoleCode
	FilterPermissions(role RoleCode, requested []Permission) []Permission
}

// NewPermissionSet creates a PermissionSet from a variadic list of permissions.
func NewPermissionSet(perms ...Permission) PermissionSet {
	ps := make(PermissionSet, len(perms))
	for _, p := range perms {
		ps[p] = struct{}{}
	}
	return ps
}

// ErrInvalidRole is returned when a role code doesn't exist in the configuration.
type ErrInvalidRole struct {
	Role       RoleCode
	ValidRoles []RoleCode
}

func (e *ErrInvalidRole) Error() string {
	return fmt.Sprintf("invalid role %q, valid roles: %v", e.Role, e.ValidRoles)
}
