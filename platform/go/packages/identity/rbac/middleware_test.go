package rbac_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/shredbx/sbx-core/pkg/rbac"
)

func mockRoleFrom(role rbac.RoleCode) rbac.RoleFromContext {
	return func(_ context.Context) (rbac.RoleCode, bool) {
		if role == "" {
			return "", false
		}
		return role, true
	}
}

func TestRequirePermission(t *testing.T) {
	authz := testAuthorizer()
	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	t.Run("T-RP-1: admin with user:read passes", func(t *testing.T) {
		mw := rbac.RequirePermission(authz, rbac.PermUserRead, mockRoleFrom(rbac.RoleAdmin))
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		mw(ok).ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("T-RP-2: team-member without user:read gets 403", func(t *testing.T) {
		mw := rbac.RequirePermission(authz, rbac.PermUserRead, mockRoleFrom(rbac.RoleTeamMember))
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		mw(ok).ServeHTTP(rec, req)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("T-RP-3: no role in context gets 403", func(t *testing.T) {
		mw := rbac.RequirePermission(authz, rbac.PermUserRead, mockRoleFrom(""))
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		mw(ok).ServeHTTP(rec, req)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	// TC-R2: unknown role (e.g., tampered JWT with role="admin-fake") is rejected
	// with 403. The log line is emitted to stderr — verified via behavior, not log capture.
	t.Run("T-RP-4: unknown role gets 403 (R2: tampering signal)", func(t *testing.T) {
		mw := rbac.RequirePermission(authz, rbac.PermUserRead, mockRoleFrom(rbac.RoleCode("admin-fake")))
		req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
		rec := httptest.NewRecorder()
		mw(ok).ServeHTTP(rec, req)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})
}

func mockUserIDFrom(id uuid.UUID) rbac.UserIDFromContext {
	return func(_ context.Context) (uuid.UUID, bool) {
		return id, true
	}
}

func TestRequireOwnership(t *testing.T) {
	authz := testAuthorizer()
	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	t.Run("T-RO-1: team-member editing own post passes", func(t *testing.T) {
		mw := rbac.RequireOwnership(authz, rbac.PermContentUpdateOwn, rbac.PermContentUpdateAny,
			mockRoleFrom(rbac.RoleTeamMember), mockUserIDFrom(testUserA),
			func(_ *http.Request) (uuid.UUID, error) { return testUserA, nil })
		req := httptest.NewRequest(http.MethodPut, "/", nil)
		rec := httptest.NewRecorder()
		mw(ok).ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("T-RO-2: team-member editing other's post gets 403", func(t *testing.T) {
		mw := rbac.RequireOwnership(authz, rbac.PermContentUpdateOwn, rbac.PermContentUpdateAny,
			mockRoleFrom(rbac.RoleTeamMember), mockUserIDFrom(testUserA),
			func(_ *http.Request) (uuid.UUID, error) { return testUserB, nil })
		req := httptest.NewRequest(http.MethodPut, "/", nil)
		rec := httptest.NewRecorder()
		mw(ok).ServeHTTP(rec, req)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("T-RO-3: admin editing other's post passes (has update-any)", func(t *testing.T) {
		mw := rbac.RequireOwnership(authz, rbac.PermContentUpdateOwn, rbac.PermContentUpdateAny,
			mockRoleFrom(rbac.RoleAdmin), mockUserIDFrom(testUserA),
			func(_ *http.Request) (uuid.UUID, error) { return testUserB, nil })
		req := httptest.NewRequest(http.MethodPut, "/", nil)
		rec := httptest.NewRecorder()
		mw(ok).ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}
