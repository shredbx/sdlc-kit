package rbac

// Permission constants for shredbx project (from FDD1.FL permission_map).
const (
	// User management
	PermUserInvite      Permission = "user:invite"
	PermUserRead        Permission = "user:read"
	PermUserManageRoles Permission = "user:manage-roles"
	PermUserDelete      Permission = "user:delete"

	// Session management
	PermSessionRevoke Permission = "session:revoke"

	// Settings
	PermSettingsManage Permission = "settings:manage"

	// Content management
	PermContentCreateDraft            Permission = "content:create-draft"
	PermContentUpdateOwn              Permission = "content:update-own"
	PermContentUpdateAny              Permission = "content:update-any"
	PermContentPublish                Permission = "content:publish"
	PermContentManageDictionaries     Permission = "content:manage-dictionaries"
	PermContentHardDeleteDictionaries Permission = "content:hard-delete-dictionaries"

	// Profile
	PermProfileEditOwn Permission = "profile:edit-own"

	// Dashboard
	PermDashboardView    Permission = "dashboard:view"
	PermDashboardViewAll Permission = "dashboard:view-all"
)

// Role constants for shredbx project.
const (
	RoleSuperAdmin RoleCode = "super-admin"
	RoleAdmin      RoleCode = "admin"
	RoleTeamMember RoleCode = "team-member"
)
