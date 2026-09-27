// Package middleware holds the fixed chain's middleware that no ported package provides.
// Security headers, CSRF and authentication come from the ported auth package, not from here.
package middleware

import (
	"net/http"

	"github.com/shredbx/sbx-core/pkg/httputil"
)

// JSONRecoverer catches panics and returns a JSON 500 instead of plain text.
func JSONRecoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rv := recover(); rv != nil {
				httputil.WriteError(w, http.StatusInternalServerError, "internal server error", httputil.CodeInternal)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
