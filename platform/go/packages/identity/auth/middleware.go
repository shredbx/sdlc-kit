package auth

import (
	"net/http"
	"strings"
)

// AuthExtract middleware parses JWT from cookie and sets Claims in context.
// NEVER returns 401 — missing/expired tokens result in nil Claims (OK for public routes).
func AuthExtract(svc AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("access_token")
			if err != nil || cookie.Value == "" {
				next.ServeHTTP(w, r)
				return
			}

			claims, err := svc.VerifyToken(r.Context(), cookie.Value)
			if err != nil || claims == nil {
				next.ServeHTTP(w, r)
				return
			}

			ctx := SetClaimsInContext(r.Context(), claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireAuth middleware ensures Claims exist in context. Returns 401 if nil.
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ClaimsFromContext(r.Context()) == nil {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// CSRFCheck middleware verifies X-Requested-With header on state-mutating requests.
func CSRFCheck(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch || r.Method == http.MethodDelete {
			if !strings.EqualFold(r.Header.Get("X-Requested-With"), "XMLHttpRequest") {
				http.Error(w, `{"error":"CSRF check failed"}`, http.StatusForbidden)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
