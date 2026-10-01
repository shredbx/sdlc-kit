package auth_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/shredbx/sbx-core/pkg/auth"
)

func TestSecurityHeaders(t *testing.T) {
	handler := auth.SecurityHeaders(auth.DefaultSecurityHeadersConfig(false))(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	t.Run("T-SH-1: CSP header present on response", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		csp := rec.Header().Get("Content-Security-Policy")
		assert.Contains(t, csp, "default-src 'self'")
		assert.Contains(t, csp, "script-src 'self'")
	})

	t.Run("T-SH-2: X-Content-Type-Options nosniff", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"))
	})

	t.Run("T-SH-3: X-Frame-Options DENY", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, "DENY", rec.Header().Get("X-Frame-Options"))
	})

	t.Run("T-SH-4: Referrer-Policy set", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.NotEmpty(t, rec.Header().Get("Referrer-Policy"))
	})

	t.Run("T-SH-5: HSTS absent in development", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Empty(t, rec.Header().Get("Strict-Transport-Security"))
	})

	t.Run("T-SH-6: HSTS present in production with preload (TC-L4)", func(t *testing.T) {
		prodHandler := auth.SecurityHeaders(auth.DefaultSecurityHeadersConfig(true))(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		prodHandler.ServeHTTP(rec, req)
		hsts := rec.Header().Get("Strict-Transport-Security")
		assert.Contains(t, hsts, "max-age=31536000")
		assert.Contains(t, hsts, "includeSubDomains")
		assert.Contains(t, hsts, "preload",
			"L4: HSTS must include 'preload' so domain can be submitted to hstspreload.org")
	})
}
