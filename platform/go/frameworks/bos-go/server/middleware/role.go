package middleware

import (
	"encoding/json"
	"net/http"

	"github.com/shredbx/sbx-core/pkg/auth"
	"github.com/shredbx/sbx-core/pkg/httputil"
)

// RequireRole returns middleware that 403s unless the request's Claims (set by an earlier
// RequireAuth/AuthExtract, real or authfake) has one of allowed as its Role. Role-gating is not
// part of the ported auth package (see this package's own doc comment) — this is the framework
// glue a route tree needs to keep two admin areas apart, real or fake auth alike: only how a
// Claims.Role gets set differs.
func RequireRole(allowed ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := auth.ClaimsFromContext(r.Context())
			if claims == nil {
				httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", httputil.CodeUnauthorized)
				return
			}
			for _, role := range allowed {
				if claims.Role == role {
					next.ServeHTTP(w, r)
					return
				}
			}
			httputil.WriteError(w, http.StatusForbidden, "forbidden", httputil.CodeForbidden)
		})
	}
}

// WhoAmI writes the current request's identity as JSON ({} if unauthenticated) — lets an admin UI
// show or hide role-gated areas without knowing whether Claims came from real or fake auth.
func WhoAmI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	claims := auth.ClaimsFromContext(r.Context())
	if claims == nil {
		_ = json.NewEncoder(w).Encode(map[string]any{})
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]string{"role": claims.Role, "email": claims.Email})
}
