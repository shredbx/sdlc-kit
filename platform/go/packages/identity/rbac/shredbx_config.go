package rbac

// NewShredbxConfig returns the default 3-role hierarchy used by shredbx-pattern
// consumers (team-member → admin → super-admin). Each role inherits the prior
// role's permissions (R1: flattening happens in NewAuthorizer).
//
// Project-specific role configs (e.g., BR's [agent, admin, super-admin]) should
// be defined in the consuming app's own config package; this is a sensible
// default for new projects following the shredbx pattern.
func NewShredbxConfig() RBACConfig {
	teamMember := NewPermissionSet(
		PermContentCreateDraft,
		PermContentUpdateOwn,
		PermProfileEditOwn,
		PermDashboardView,
	)

	admin := NewPermissionSet(
		// inherits team-member via flattening, but declared explicitly for clarity
		PermContentCreateDraft,
		PermContentUpdateOwn,
		PermProfileEditOwn,
		PermDashboardView,
		// admin-specific:
		PermUserRead,
		PermContentUpdateAny,
		PermContentPublish,
		PermContentManageDictionaries,
		PermDashboardViewAll,
		PermSessionRevoke,
	)

	superAdmin := NewPermissionSet(
		PermContentCreateDraft,
		PermContentUpdateOwn,
		PermProfileEditOwn,
		PermDashboardView,
		PermUserRead,
		PermContentUpdateAny,
		PermContentPublish,
		PermContentManageDictionaries,
		PermDashboardViewAll,
		PermSessionRevoke,
		// super-admin-specific:
		PermUserInvite,
		PermUserManageRoles,
		PermUserDelete,
		PermSettingsManage,
		PermContentHardDeleteDictionaries,
	)

	return RBACConfig{
		Roles: map[RoleCode]PermissionSet{
			RoleTeamMember: teamMember,
			RoleAdmin:      admin,
			RoleSuperAdmin: superAdmin,
		},
		DefaultRole:   RoleTeamMember,
		RoleHierarchy: []RoleCode{RoleTeamMember, RoleAdmin, RoleSuperAdmin},
	}
}
