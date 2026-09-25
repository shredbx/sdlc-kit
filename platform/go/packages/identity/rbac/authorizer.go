package rbac

import "github.com/google/uuid"

type authorizer struct {
	roles     map[RoleCode]PermissionSet
	hierarchy []RoleCode
}

// NewAuthorizer creates an Authorizer from an RBACConfig.
// R1: hierarchy is flattened at init time. Each role in RoleHierarchy inherits
// the cumulative permissions of all preceding roles in the list. Future configs
// can be written incrementally — admin only needs to declare the NEW perms it
// grants over the prior role. Configs that already duplicate full permission
// sets per role (like BR's current config) are flattening-idempotent: the
// flatten step adds nothing new.
//
// Roles NOT present in RoleHierarchy keep their declared permission set as-is
// (no inheritance applied). This supports flat configs that opt out of hierarchy.
func NewAuthorizer(cfg RBACConfig) Authorizer {
	flattened := make(map[RoleCode]PermissionSet, len(cfg.Roles))

	// Walk hierarchy in order, accumulating permissions.
	accumulated := PermissionSet{}
	for _, roleName := range cfg.RoleHierarchy {
		if perms, ok := cfg.Roles[roleName]; ok {
			for p := range perms {
				accumulated[p] = struct{}{}
			}
		}
		// Snapshot accumulated state into this role's flattened set.
		snapshot := make(PermissionSet, len(accumulated))
		for p := range accumulated {
			snapshot[p] = struct{}{}
		}
		flattened[roleName] = snapshot
	}

	// Preserve roles not in the hierarchy (flat-config opt-out).
	for r, perms := range cfg.Roles {
		if _, alreadyFlattened := flattened[r]; !alreadyFlattened {
			flattened[r] = perms
		}
	}

	return &authorizer{
		roles:     flattened,
		hierarchy: cfg.RoleHierarchy,
	}
}

func (a *authorizer) HasPermission(role RoleCode, perm Permission) bool {
	ps, ok := a.roles[role]
	if !ok {
		return false
	}
	return ps.Has(perm)
}

func (a *authorizer) GetPermissions(role RoleCode) PermissionSet {
	ps, ok := a.roles[role]
	if !ok {
		return PermissionSet{}
	}
	return ps
}

func (a *authorizer) ValidateRole(role RoleCode) error {
	if _, ok := a.roles[role]; ok {
		return nil
	}
	return &ErrInvalidRole{
		Role:       role,
		ValidRoles: a.AvailableRoles(),
	}
}

func (a *authorizer) CheckOwnership(role RoleCode, ownPerm, anyPerm Permission, resourceOwnerID, requestorID uuid.UUID) bool {
	// Check "any" permission first — if role can edit any resource, ownership doesn't matter.
	if a.HasPermission(role, anyPerm) {
		return true
	}
	// Fall back to "own" permission + ownership match.
	if a.HasPermission(role, ownPerm) && resourceOwnerID == requestorID {
		return true
	}
	return false
}

func (a *authorizer) AvailableRoles() []RoleCode {
	roles := make([]RoleCode, 0, len(a.roles))
	for r := range a.roles {
		roles = append(roles, r)
	}
	return roles
}

func (a *authorizer) FilterPermissions(role RoleCode, requested []Permission) []Permission {
	ps := a.GetPermissions(role)
	result := make([]Permission, 0, len(requested))
	for _, p := range requested {
		if ps.Has(p) {
			result = append(result, p)
		}
	}
	return result
}
