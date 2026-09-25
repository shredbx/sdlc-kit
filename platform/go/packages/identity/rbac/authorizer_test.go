package rbac_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shredbx/sbx-core/pkg/rbac"
)

func TestHasPermission(t *testing.T) {
	authz := testAuthorizer()

	tests := []struct {
		name string
		role rbac.RoleCode
		perm rbac.Permission
		want bool
	}{
		{"T-HP-1: super-admin has user:invite", rbac.RoleSuperAdmin, rbac.PermUserInvite, true},
		{"T-HP-2: team-member lacks user:invite", rbac.RoleTeamMember, rbac.PermUserInvite, false},
		{"T-HP-3: unknown role has nothing", rbac.RoleCode("unknown"), rbac.PermUserInvite, false},
		{"T-HP-4: admin inherits content:update-own from team-member", rbac.RoleAdmin, rbac.PermContentUpdateOwn, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := authz.HasPermission(tt.role, tt.perm)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetPermissions(t *testing.T) {
	authz := testAuthorizer()

	t.Run("T-GP-1: super-admin has all 15 permissions", func(t *testing.T) {
		ps := authz.GetPermissions(rbac.RoleSuperAdmin)
		assert.Len(t, ps, 15)
	})

	t.Run("T-GP-2: team-member has 4 permissions", func(t *testing.T) {
		ps := authz.GetPermissions(rbac.RoleTeamMember)
		assert.Len(t, ps, 4)
	})

	t.Run("T-GP-3: unknown role returns empty set", func(t *testing.T) {
		ps := authz.GetPermissions(rbac.RoleCode("unknown"))
		assert.Len(t, ps, 0)
	})
}

func TestValidateRole(t *testing.T) {
	authz := testAuthorizer()

	t.Run("T-VR-1: admin is valid", func(t *testing.T) {
		err := authz.ValidateRole(rbac.RoleAdmin)
		assert.NoError(t, err)
	})

	t.Run("T-VR-2: superuser is invalid", func(t *testing.T) {
		err := authz.ValidateRole(rbac.RoleCode("superuser"))
		require.Error(t, err)
		var invalidErr *rbac.ErrInvalidRole
		require.ErrorAs(t, err, &invalidErr)
		assert.Len(t, invalidErr.ValidRoles, 3)
	})

	t.Run("T-VR-3: empty string is invalid", func(t *testing.T) {
		err := authz.ValidateRole(rbac.RoleCode(""))
		assert.Error(t, err)
	})
}

func TestCheckOwnership(t *testing.T) {
	authz := testAuthorizer()

	t.Run("T-CO-1: team-member owns resource", func(t *testing.T) {
		ok := authz.CheckOwnership(rbac.RoleTeamMember, rbac.PermContentUpdateOwn, rbac.PermContentUpdateAny, testUserA, testUserA)
		assert.True(t, ok)
	})

	t.Run("T-CO-2: team-member doesn't own resource", func(t *testing.T) {
		ok := authz.CheckOwnership(rbac.RoleTeamMember, rbac.PermContentUpdateOwn, rbac.PermContentUpdateAny, testUserA, testUserB)
		assert.False(t, ok)
	})

	t.Run("T-CO-3: admin has update-any", func(t *testing.T) {
		ok := authz.CheckOwnership(rbac.RoleAdmin, rbac.PermContentUpdateOwn, rbac.PermContentUpdateAny, testUserA, testUserB)
		assert.True(t, ok)
	})

	t.Run("T-CO-4: admin skips ownership check", func(t *testing.T) {
		ok := authz.CheckOwnership(rbac.RoleAdmin, rbac.PermContentUpdateOwn, rbac.PermContentUpdateAny, testUserA, testUserA)
		assert.True(t, ok)
	})
}

func TestAvailableRoles(t *testing.T) {
	authz := testAuthorizer()
	roles := authz.AvailableRoles()
	assert.Len(t, roles, 3)
}

func TestFilterPermissions(t *testing.T) {
	authz := testAuthorizer()

	t.Run("T-FP-1: admin filters to what they have", func(t *testing.T) {
		result := authz.FilterPermissions(rbac.RoleAdmin, []rbac.Permission{rbac.PermUserRead, rbac.PermUserInvite})
		assert.Equal(t, []rbac.Permission{rbac.PermUserRead}, result)
	})

	t.Run("T-FP-2: empty requested returns empty", func(t *testing.T) {
		result := authz.FilterPermissions(rbac.RoleAdmin, []rbac.Permission{})
		assert.Empty(t, result)
	})
}

// TC-R1: NewAuthorizer flattens RoleHierarchy at init.
// An incremental config (each role declares only NEW perms) should grant
// inherited perms to higher-tier roles automatically.
func TestNewAuthorizer_FlattensHierarchy(t *testing.T) {
	// Incremental config: each role declares ONLY the perms it adds to its parent.
	cfg := rbac.RBACConfig{
		Roles: map[rbac.RoleCode]rbac.PermissionSet{
			rbac.RoleTeamMember: rbac.NewPermissionSet(rbac.PermDashboardView),       // ONE perm
			rbac.RoleAdmin:      rbac.NewPermissionSet(rbac.PermUserRead),            // ONE perm
			rbac.RoleSuperAdmin: rbac.NewPermissionSet(rbac.PermUserManageRoles),     // ONE perm
		},
		RoleHierarchy: []rbac.RoleCode{rbac.RoleTeamMember, rbac.RoleAdmin, rbac.RoleSuperAdmin},
	}

	authz := rbac.NewAuthorizer(cfg)

	// team-member: just its declared perm
	assert.True(t, authz.HasPermission(rbac.RoleTeamMember, rbac.PermDashboardView))
	assert.False(t, authz.HasPermission(rbac.RoleTeamMember, rbac.PermUserRead))

	// admin: inherits team-member's perms PLUS its own
	assert.True(t, authz.HasPermission(rbac.RoleAdmin, rbac.PermDashboardView), "admin should inherit team-member's PermDashboardView")
	assert.True(t, authz.HasPermission(rbac.RoleAdmin, rbac.PermUserRead))
	assert.False(t, authz.HasPermission(rbac.RoleAdmin, rbac.PermUserManageRoles))

	// super-admin: inherits everything below + its own
	assert.True(t, authz.HasPermission(rbac.RoleSuperAdmin, rbac.PermDashboardView), "super-admin should inherit team-member's")
	assert.True(t, authz.HasPermission(rbac.RoleSuperAdmin, rbac.PermUserRead), "super-admin should inherit admin's")
	assert.True(t, authz.HasPermission(rbac.RoleSuperAdmin, rbac.PermUserManageRoles))
}

// Flattening must be idempotent for already-fully-declared configs (BR pattern).
func TestNewAuthorizer_FlatteningIsIdempotentForFullConfig(t *testing.T) {
	authz := rbac.NewAuthorizer(rbac.NewShredbxConfig())
	// super-admin's 15 perms are explicitly declared in NewShredbxConfig.
	// After flattening, it should still have exactly those 15 (no duplicates).
	perms := authz.GetPermissions(rbac.RoleSuperAdmin)
	assert.Len(t, perms, 15)
}

// Roles outside the hierarchy keep their declared perm set as-is.
func TestNewAuthorizer_NonHierarchicalRolesPreserved(t *testing.T) {
	specialRole := rbac.RoleCode("readonly")
	cfg := rbac.RBACConfig{
		Roles: map[rbac.RoleCode]rbac.PermissionSet{
			rbac.RoleAdmin: rbac.NewPermissionSet(rbac.PermUserInvite, rbac.PermUserRead),
			specialRole:    rbac.NewPermissionSet(rbac.PermDashboardView),
		},
		RoleHierarchy: []rbac.RoleCode{rbac.RoleAdmin}, // readonly NOT in hierarchy
	}
	authz := rbac.NewAuthorizer(cfg)

	// readonly keeps just dashboard:view — does NOT inherit from admin
	assert.True(t, authz.HasPermission(specialRole, rbac.PermDashboardView))
	assert.False(t, authz.HasPermission(specialRole, rbac.PermUserInvite))
}

func TestNewShredbxConfig(t *testing.T) {
	cfg := rbac.NewShredbxConfig()

	t.Run("T-NSC-1: 3 roles", func(t *testing.T) {
		assert.Len(t, cfg.Roles, 3)
	})

	t.Run("T-NSC-2: default is team-member", func(t *testing.T) {
		assert.Equal(t, rbac.RoleTeamMember, cfg.DefaultRole)
	})

	t.Run("T-NSC-3: super-admin has all 13 permissions", func(t *testing.T) {
		ps := cfg.Roles[rbac.RoleSuperAdmin]
		allPerms := []rbac.Permission{
			rbac.PermUserInvite, rbac.PermUserRead, rbac.PermUserManageRoles, rbac.PermUserDelete,
			rbac.PermSessionRevoke, rbac.PermSettingsManage,
			rbac.PermContentCreateDraft, rbac.PermContentUpdateOwn, rbac.PermContentUpdateAny, rbac.PermContentPublish,
			rbac.PermProfileEditOwn, rbac.PermDashboardView, rbac.PermDashboardViewAll,
		}
		for _, p := range allPerms {
			assert.True(t, ps.Has(p), "super-admin should have %s", p)
		}
	})
}

func TestPermissionSet_Has(t *testing.T) {
	t.Run("T-PSH-1: has existing permission", func(t *testing.T) {
		ps := rbac.NewPermissionSet(rbac.PermUserRead)
		assert.True(t, ps.Has(rbac.PermUserRead))
	})

	t.Run("T-PSH-2: lacks missing permission", func(t *testing.T) {
		ps := rbac.NewPermissionSet(rbac.PermUserRead)
		assert.False(t, ps.Has(rbac.PermUserInvite))
	})

	t.Run("T-PSH-3: empty set has nothing", func(t *testing.T) {
		ps := rbac.NewPermissionSet()
		assert.False(t, ps.Has(rbac.PermUserRead))
	})
}

func TestNewAuthorizer_EmptyConfig(t *testing.T) {
	authz := testEmptyAuthorizer()

	t.Run("T-NA-2: denies everything", func(t *testing.T) {
		assert.False(t, authz.HasPermission(rbac.RoleSuperAdmin, rbac.PermUserInvite))
	})
}

func BenchmarkHasPermission(b *testing.B) {
	authz := testAuthorizer()
	for i := 0; i < b.N; i++ {
		authz.HasPermission(rbac.RoleSuperAdmin, rbac.PermUserInvite)
	}
}
