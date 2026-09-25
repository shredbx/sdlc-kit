package auth

import (
	"encoding/json"
	"net"
	"net/http"
	"strconv"
	"strings"
)

// statusCapturingWriter wraps ResponseWriter to record the HTTP status code so
// the rate-limit middleware can decide whether to RecordFailure post-handler.
type statusCapturingWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (w *statusCapturingWriter) WriteHeader(code int) {
	if !w.wroteHeader {
		w.status = code
		w.wroteHeader = true
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusCapturingWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.status = http.StatusOK
		w.wroteHeader = true
	}
	return w.ResponseWriter.Write(b)
}

// RateLimitMiddleware returns HTTP middleware that enforces rate limiting.
// keyFunc extracts the rate limit key from the request (e.g., IP address, user ID).
// Auto-records a failure with the limiter when the downstream handler returns
// 401 or 423 — typical auth/lockout responses worth counting against the user
// or IP. 4xx responses outside that range (404, 400) are NOT counted, since they
// don't represent credential-stuffing or brute-force signals.
func RateLimitMiddleware(limiter RateLimiter, cfg RateLimitConfig, keyFunc func(*http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := keyFunc(r)
			result, err := limiter.Allow(r.Context(), key, cfg)
			if err != nil {
				http.Error(w, `{"error":"internal server error","code":"rate_limit_error"}`, http.StatusInternalServerError)
				return
			}

			if !result.Allowed {
				w.Header().Set("Retry-After", strconv.FormatInt(result.RetryAfter, 10))
				w.Header().Set("X-RateLimit-Remaining", "0")
				w.Header().Set("Content-Type", "application/json")

				if result.Locked {
					w.WriteHeader(http.StatusLocked)
					json.NewEncoder(w).Encode(map[string]interface{}{
						"error":       "Account temporarily locked",
						"code":        "account_locked",
						"retry_after": result.RetryAfter,
					})
				} else {
					w.WriteHeader(http.StatusTooManyRequests)
					json.NewEncoder(w).Encode(map[string]interface{}{
						"error":       "Rate limit exceeded",
						"code":        "rate_limited",
						"retry_after": result.RetryAfter,
					})
				}
				return
			}

			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(result.Remaining))
			if result.BackoffSeconds > 0 {
				w.Header().Set("X-RateLimit-Backoff", strconv.Itoa(result.BackoffSeconds))
			}

			capturing := &statusCapturingWriter{ResponseWriter: w}
			next.ServeHTTP(capturing, r)

			// Bug-fix: middleware used to only call Allow, never RecordFailure —
			// so failed logins never incremented the counter. Now we count 401
			// (auth failure) and 423 (downstream lockout) against the key.
			if capturing.status == http.StatusUnauthorized || capturing.status == http.StatusLocked {
				_ = limiter.RecordFailure(r.Context(), key, cfg)
			}
		})
	}
}

// IPKeyFunc extracts client IP from X-Forwarded-For (first value) or RemoteAddr.
func IPKeyFunc(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.SplitN(xff, ",", 2)
		if ip := strings.TrimSpace(parts[0]); ip != "" {
			return ip
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// UserKeyFunc extracts user ID from auth claims in context. Falls back to IPKeyFunc.
func UserKeyFunc(r *http.Request) string {
	claims := ClaimsFromContext(r.Context())
	if claims != nil {
		return "user:" + claims.Sub.String()
	}
	return IPKeyFunc(r)
}
