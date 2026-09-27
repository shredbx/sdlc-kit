package rbac

import (
	"context"
	"log"
	"net/http"

	"github.com/google/uuid"
)

// RequirePermission creates middleware that checks if the user's role has a specific permission.
// roleFrom extracts the role from context (injected to avoid auth import).
// R2: logs unknown role attempts (role present in claims but not in RBACConfig)
// — these are tampered-JWT or stale-role-in-token signals worth alerting on.
func RequirePermission(authz Authorizer, perm Permission, roleFrom RoleFromContext) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, ok := roleFrom(r.Context())
			if !ok {
				http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
				return
			}

			// R2: distinguish "unknown role" (tampering signal) from "role lacks
			// permission" (normal authz denial) and emit a structured log line.
			if err := authz.ValidateRole(role); err != nil {
				log.Printf("rbac: unknown_role role=%q path=%s method=%s remote=%s",
					role, r.URL.Path, r.Method, r.RemoteAddr)
				http.Error(w, `{"error":"insufficient permissions"}`, http.StatusForbidden)
				return
			}

			if !authz.HasPermission(role, perm) {
				http.Error(w, `{"error":"insufficient permissions"}`, http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// UserIDFromContext extracts the user ID from request context.
// Injected by the caller (same pattern as RoleFromContext).
type UserIDFromContext func(ctx context.Context) (uuid.UUID, bool)

// RequireOwnership creates middleware for resource-level authorization.
// Checks "any" permission first, falls back to "own" + ownership match.
//
// R3: This middleware is intended for domains that have an explicit ownership
// model — i.e., resources tied to a creator/owner where role-based any/own
// permission pairs are meaningful. When using it, define paired permissions
// like content:update-own + content:update-any and an ownerExtractor that
// fetches the resource's owner_id from the DB (NOT from request body or URL
// params — those are attacker-controlled).
//
// Apps that have NO ownership model (e.g., BR where all agents can manage all
// properties) should NOT use this middleware — RequirePermission with the
// permission set per role is sufficient and clearer. Mixing the two creates
// silent gaps in authorization coverage.
func RequireOwnership(authz Authorizer, ownPerm, anyPerm Permission, roleFrom RoleFromContext, userIDFrom UserIDFromContext, ownerExtractor func(r *http.Request) (uuid.UUID, error)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, ok := roleFrom(r.Context())
			if !ok {
				http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
				return
			}

			// Check "any" permission first
			if authz.HasPermission(role, anyPerm) {
				next.ServeHTTP(w, r)
				return
			}

			// Need "own" permission + ownership check
			if !authz.HasPermission(role, ownPerm) {
				http.Error(w, `{"error":"insufficient permissions"}`, http.StatusForbidden)
				return
			}

			ownerID, err := ownerExtractor(r)
			if err != nil {
				http.Error(w, `{"error":"failed to determine resource owner"}`, http.StatusInternalServerError)
				return
			}

			requestorID, ok := userIDFrom(r.Context())
			if !ok {
				http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
				return
			}

			if ownerID != requestorID {
				http.Error(w, `{"error":"not resource owner"}`, http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
