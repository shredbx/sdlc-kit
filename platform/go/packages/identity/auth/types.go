// Package auth handles authentication — JWT tokens, sessions, magic links, passwords.
// PD layer types are pure data structures with no I/O dependencies.
package auth

import (
	"time"

	"github.com/google/uuid"
)

// Session represents an authenticated session bound to a user and device.
type Session struct {
	ID                  uuid.UUID  `json:"id"`
	UserID              uuid.UUID  `json:"user_id"`
	RefreshToken        string     `json:"-"`
	JTI                 uuid.UUID  `json:"jti"`
	UserAgent           *string    `json:"user_agent,omitempty"`
	IPAddress           *string    `json:"ip_address,omitempty"`
	ExpiresAt           time.Time  `json:"expires_at"`
	RevokedAt           *time.Time `json:"revoked_at,omitempty"`
	ReplacedBySessionID *uuid.UUID `json:"replaced_by_session_id,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
}

// MagicLink is a one-time token for invite or password-reset flows.
type MagicLink struct {
	ID        uuid.UUID        `json:"id"`
	Token     string           `json:"token"`
	UserID    uuid.UUID        `json:"user_id"`
	Purpose   MagicLinkPurpose `json:"purpose"`
	ExpiresAt time.Time        `json:"expires_at"`
	UsedAt    *time.Time       `json:"used_at,omitempty"`
	CreatedAt time.Time        `json:"created_at"`
}

// MagicLinkStatus is the derived status of a magic link token.
type MagicLinkStatus string

const (
	MagicLinkStatusValid   MagicLinkStatus = "valid"
	MagicLinkStatusUsed    MagicLinkStatus = "used"
	MagicLinkStatusExpired MagicLinkStatus = "expired"
)

// MagicLinkPurpose distinguishes invite vs reset flows so a reset-purpose token
// cannot be replayed as an invite token (and vice versa) — H2 defense-in-depth.
type MagicLinkPurpose string

const (
	MagicLinkPurposeInvite MagicLinkPurpose = "invite"
	MagicLinkPurposeReset  MagicLinkPurpose = "reset"
	MagicLinkPurposeLogin  MagicLinkPurpose = "login"
)

// Status returns the derived status of the magic link.
func (ml *MagicLink) Status() MagicLinkStatus {
	if ml.UsedAt != nil {
		return MagicLinkStatusUsed
	}
	if time.Now().After(ml.ExpiresAt) {
		return MagicLinkStatusExpired
	}
	return MagicLinkStatusValid
}

// TokenPair contains access + refresh tokens returned on login/activate/refresh.
type TokenPair struct {
	AccessToken      string    `json:"access_token"`
	RefreshToken     string    `json:"refresh_token"`
	AccessExpiresAt  time.Time `json:"access_expires_at"`
	RefreshExpiresAt time.Time `json:"refresh_expires_at"`
	TokenType        string    `json:"token_type"`
}

// TokenIssuer is the JWT issuer claim value for shredbx tokens.
const TokenIssuer = "shredbx"

// Claims represents the JWT payload.
type Claims struct {
	Sub   uuid.UUID `json:"sub"`
	Email string    `json:"email"`
	Role  string    `json:"role"`
	JTI   uuid.UUID `json:"jti"`
	IAT   int64     `json:"iat"`
	EXP   int64     `json:"exp"`
}

// AuthConfig holds constructor-injected configuration for auth services.
type AuthConfig struct {
	JWTSecret       string        `json:"-"`
	AccessTokenTTL  time.Duration `json:"access_token_ttl"`
	RefreshTokenTTL time.Duration `json:"refresh_token_ttl"`
	MagicLinkTTL    time.Duration `json:"magic_link_ttl"`
	// MagicLinkResetTTL overrides MagicLinkTTL for reset-purpose links so a
	// shorter window (default 1h) bounds the blast radius of a leaked reset
	// link. If zero, falls back to MagicLinkTTL.
	MagicLinkResetTTL  time.Duration `json:"magic_link_reset_ttl"`
	ResetRequestExpiry time.Duration `json:"reset_request_expiry"` // default 7 days
	BcryptCost         int           `json:"bcrypt_cost"`
	IsProduction       bool          `json:"is_production"`
	CookieDomain       string        `json:"cookie_domain"`
	ValidRoles         []string      `json:"valid_roles"` // configurable role list
	// SessionPinning: when true, refresh token validation compares the
	// requesting IP and User-Agent against the values captured at login.
	// Mismatches return ErrSessionMismatch + emit an audit event. Off by
	// default — opt-in via config because legitimate users behind mobile
	// networks (changing IP) or browser updates (changing UA) would be
	// punished by aggressive pinning. L7.
	SessionPinning bool `json:"session_pinning"`

	// RefreshReuseLeeway is the refresh-token rotation grace window. A rotated
	// token replayed within this window of its rotation is treated as a BENIGN
	// concurrency race — a second tab, a queued request, a lost/discarded
	// Set-Cookie, or a rolling-deploy two-container overlap replaying the
	// just-rotated token — and issued a fresh token pair WITHOUT revoking the
	// session family. A replay AFTER the window is treated as theft (the whole
	// family is revoked; RFC 6819 §5.2.2.3 / RFC 9700). Mirrors Auth0's "Rotation
	// Overlap Period" and Okta's "Grace period for token rotation" (default 30s,
	// range 0-60s). Defaults() normalizes a zero value UP to the 30s default so the
	// grace is always on — a zero window makes every benign concurrent refresh a
	// full-family logout.
	RefreshReuseLeeway time.Duration `json:"refresh_reuse_leeway"`
}

// DefaultRefreshReuseLeeway is the out-of-the-box refresh-token rotation grace
// window, matching Okta's documented default. See AuthConfig.RefreshReuseLeeway.
const DefaultRefreshReuseLeeway = 30 * time.Second

// Defaults fills zero-value AuthConfig fields that carry a safe non-zero default.
// Today that is only the rotation grace window: a zero RefreshReuseLeeway would
// turn every benign concurrent refresh into a full-family logout, so it is
// normalized UP to DefaultRefreshReuseLeeway rather than treated as "disabled".
func (c *AuthConfig) Defaults() {
	if c.RefreshReuseLeeway == 0 {
		c.RefreshReuseLeeway = DefaultRefreshReuseLeeway
	}
}

// RateLimitConfig holds rate limiting configuration for auth endpoints.
type RateLimitConfig struct {
	MaxAttempts      int           `json:"max_attempts"`
	Window           time.Duration `json:"window"`
	LockoutThreshold int           `json:"lockout_threshold"` // 0 = use MaxAttempts
	LockoutDuration  time.Duration `json:"lockout_duration"`  // 0 = 30min default
	BackoffBase      int           `json:"backoff_base"`      // 0 = 2 (exponential)
	BackoffMax       int           `json:"backoff_max"`       // 0 = 300 seconds
}

// Defaults fills zero-value fields with sensible defaults.
func (c *RateLimitConfig) Defaults() {
	if c.LockoutThreshold == 0 {
		c.LockoutThreshold = c.MaxAttempts
	}
	if c.LockoutDuration == 0 {
		c.LockoutDuration = 30 * time.Minute
	}
	if c.BackoffBase == 0 {
		c.BackoffBase = 2
	}
	if c.BackoffMax == 0 {
		c.BackoffMax = 300
	}
}

// Sentinel errors for auth operations.
var (
	ErrInvalidCredentials    = &AuthError{Code: "invalid_credentials", Message: "Invalid credentials"}
	ErrTokenExpired          = &AuthError{Code: "token_expired", Message: "Token expired"}
	ErrTokenRevoked          = &AuthError{Code: "token_revoked", Message: "Token revoked"}
	ErrSessionNotFound       = &AuthError{Code: "session_not_found", Message: "Session not found"}
	ErrSessionExpired        = &AuthError{Code: "session_expired", Message: "Session expired"}
	ErrSessionRevoked        = &AuthError{Code: "session_revoked", Message: "Session revoked"}
	ErrMagicLinkNotFound     = &AuthError{Code: "magic_link_not_found", Message: "Magic link not found"}
	ErrMagicLinkExpired      = &AuthError{Code: "magic_link_expired", Message: "Magic link expired"}
	ErrMagicLinkUsed         = &AuthError{Code: "magic_link_used", Message: "Magic link already used"}
	ErrMagicLinkWrongPurpose = &AuthError{Code: "magic_link_wrong_purpose", Message: "Magic link purpose does not match the requested flow"}
	ErrEmptyPassword         = &AuthError{Code: "empty_password", Message: "Password cannot be empty"}
	ErrPasswordTooLong       = &AuthError{Code: "password_too_long", Message: "Password exceeds maximum length"}
	ErrEmptyJWTSecret        = &AuthError{Code: "empty_jwt_secret", Message: "JWT secret is required"}
	ErrJWTSecretTooShort     = &AuthError{Code: "jwt_secret_too_short", Message: "JWT secret must be at least 32 bytes"}
	ErrEmptySubject          = &AuthError{Code: "empty_subject", Message: "Token subject (user ID) is required"}
	ErrResetRequestNotFound  = &AuthError{Code: "reset_not_found", Message: "Reset request not found"}
	ErrResetRequestExpired   = &AuthError{Code: "reset_expired", Message: "Reset request expired"}
	ErrWeakPassword          = &AuthError{Code: "weak_password", Message: "Password does not meet strength requirements"}
	ErrInvalidEmail          = &AuthError{Code: "invalid_email", Message: "Invalid email format"}
	ErrInvalidRole           = &AuthError{Code: "invalid_role", Message: "Invalid role"}
	ErrRateLimited           = &AuthError{Code: "rate_limited", Message: "Too many requests"}
	ErrAccountLocked         = &AuthError{Code: "account_locked", Message: "Account temporarily locked"}
	ErrRefreshTokenReused    = &AuthError{Code: "refresh_token_reused", Message: "Refresh token reuse detected — all sessions revoked"}
	ErrSessionMismatch       = &AuthError{Code: "session_mismatch", Message: "Session client context (IP/UA) does not match login fingerprint"}
	ErrUserInactive          = &AuthError{Code: "user_inactive", Message: "User account is inactive"}
)

// AuthError is a typed error for auth operations.
type AuthError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *AuthError) Error() string {
	return e.Message
}

// PasswordResetRequest represents a user's request to reset their password.
// Admin reviews and approves (generating a magic link) or rejects.
type PasswordResetRequest struct {
	ID          uuid.UUID          `json:"id"`
	UserID      uuid.UUID          `json:"user_id"`
	Email       string             `json:"email,omitempty"` // denormalized for admin display
	Status      ResetRequestStatus `json:"status"`
	Reason      string             `json:"reason,omitempty"`
	ResolvedBy  *uuid.UUID         `json:"resolved_by,omitempty"`
	ResolvedAt  *time.Time         `json:"resolved_at,omitempty"`
	MagicLinkID *uuid.UUID         `json:"magic_link_id,omitempty"`
	ExpiresAt   time.Time          `json:"expires_at"`
	CreatedAt   time.Time          `json:"created_at"`
}

// ResetRequestStatus is the lifecycle status of a password reset request.
type ResetRequestStatus string

const (
	ResetStatusPending  ResetRequestStatus = "pending"
	ResetStatusApproved ResetRequestStatus = "approved"
	ResetStatusRejected ResetRequestStatus = "rejected"
	ResetStatusExpired  ResetRequestStatus = "expired"
)

// RateLimitResult is the outcome of a rate limit check.
type RateLimitResult struct {
	Allowed        bool  `json:"allowed"`
	Remaining      int   `json:"remaining"`
	Locked         bool  `json:"locked"`
	RetryAfter     int64 `json:"retry_after,omitempty"`
	BackoffSeconds int   `json:"backoff_seconds,omitempty"`
}

// SecurityHeadersConfig holds security header configuration.
type SecurityHeadersConfig struct {
	CSP            string `json:"csp"`             // default: "default-src 'self'; script-src 'self'..."
	ReferrerPolicy string `json:"referrer_policy"` // default: "strict-origin-when-cross-origin"
	IsProduction   bool   `json:"is_production"`
	HSTSMaxAge     int    `json:"hsts_max_age"` // default: 31536000 (1 year)
}

// DefaultSecurityHeadersConfig returns production-safe defaults.
func DefaultSecurityHeadersConfig(isProduction bool) SecurityHeadersConfig {
	return SecurityHeadersConfig{
		CSP:            "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self'",
		ReferrerPolicy: "strict-origin-when-cross-origin",
		IsProduction:   isProduction,
		HSTSMaxAge:     31536000,
	}
}
