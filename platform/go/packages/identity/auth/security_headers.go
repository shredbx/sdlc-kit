package auth

import (
	"fmt"
	"net/http"
)

// SecurityHeaders returns middleware that sets security headers on all responses.
// Accepts SecurityHeadersConfig for full control, or use DefaultSecurityHeadersConfig(isProduction).
//
// L2 known limitation: the default CSP includes 'unsafe-inline' for style-src
// because SvelteKit hydration injects inline <style> blocks. A future iteration
// can adopt per-request CSP nonces by:
//  1. Generating a random nonce per request
//  2. Substituting it into the CSP via `style-src 'self' 'nonce-{value}'`
//  3. Plumbing the nonce into SvelteKit's `csp` config so injected styles
//     receive the same nonce attribute
//
// Deferred until SvelteKit integration is needed. 'unsafe-inline' here only
// applies to styles (script-src 'self' is strict), so the XSS amplification is
// minor — an attacker would need a script-execution primitive elsewhere first.
func SecurityHeaders(cfg SecurityHeadersConfig) func(http.Handler) http.Handler {
	// Apply defaults for empty fields
	if cfg.CSP == "" {
		cfg.CSP = "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self'"
	}
	if cfg.ReferrerPolicy == "" {
		cfg.ReferrerPolicy = "strict-origin-when-cross-origin"
	}
	if cfg.HSTSMaxAge == 0 {
		cfg.HSTSMaxAge = 31536000
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Security-Policy", cfg.CSP)
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("Referrer-Policy", cfg.ReferrerPolicy)

			if cfg.IsProduction {
				// L4: add `preload` directive so the domain can be submitted to
				// hstspreload.org for browser-baked HSTS protection (no first-visit
				// downgrade vector). Submission is a manual deploy-time step.
				w.Header().Set("Strict-Transport-Security",
					fmt.Sprintf("max-age=%d; includeSubDomains; preload", cfg.HSTSMaxAge))
			}

			next.ServeHTTP(w, r)
		})
	}
}
