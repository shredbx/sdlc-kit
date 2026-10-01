package rbac_test

import (
	"github.com/google/uuid"
	"github.com/shredbx/sbx-core/pkg/rbac"
)

// Test fixtures for rbac package.

func testConfig() rbac.RBACConfig {
	return rbac.NewShredbxConfig()
}

func testAuthorizer() rbac.Authorizer {
	return rbac.NewAuthorizer(testConfig())
}

func testEmptyAuthorizer() rbac.Authorizer {
	return rbac.NewAuthorizer(rbac.RBACConfig{
		Roles:       map[rbac.RoleCode]rbac.PermissionSet{},
		DefaultRole: "",
	})
}

var (
	testUserA = uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	testUserB = uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
)
