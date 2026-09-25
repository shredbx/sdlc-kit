package auth

import (
	"context"

	"github.com/google/uuid"
)

// AuthService handles authentication — login, logout, token refresh, verification.
type AuthService interface {
	Login(ctx context.Context, email, password string) (*TokenPair, error)
	Logout(ctx context.Context, refreshToken string) error
	RefreshToken(ctx context.Context, refreshToken string) (*TokenPair, error)
	VerifyToken(ctx context.Context, accessToken string) (*Claims, error)
	RevokeSession(ctx context.Context, sessionID uuid.UUID) error
	ListSessions(ctx context.Context, userID uuid.UUID) ([]Session, error)
}

// MagicLinkService handles magic link onboarding — generate, validate, activate.
type MagicLinkService interface {
	Generate(ctx context.Context, email, role string) (*MagicLink, error)
	// GenerateForPurpose creates a magic link for an existing user with an
	// explicit purpose (invite, reset, login). Used by super-admin-initiated
	// flows (reactivate, password-reset) where the email-lookup branching
	// inside Generate is not what we want — we already know the user ID and
	// the desired purpose.
	GenerateForPurpose(ctx context.Context, userID uuid.UUID, purpose MagicLinkPurpose) (*MagicLink, error)
	Validate(ctx context.Context, token string) (*MagicLink, MagicLinkStatus, error)
	CompleteOnboarding(ctx context.Context, token, password, fullName string) (*TokenPair, error)
}

// SessionRepository persists sessions in Postgres.
type SessionRepository interface {
	Create(ctx context.Context, session *Session) error
	Get(ctx context.Context, id uuid.UUID) (*Session, error)
	GetByRefreshToken(ctx context.Context, token string) (*Session, error)
	ListByUser(ctx context.Context, userID uuid.UUID) ([]Session, error)
	Revoke(ctx context.Context, id uuid.UUID) error
	// RevokeForRotation atomically marks oldID as revoked and records that it was
	// replaced by newID. Used during refresh-token rotation so that a future replay
	// of the old refresh token can be distinguished from an admin-revoked session
	// (theft signal — triggers RevokeAllForUser via the service layer).
	RevokeForRotation(ctx context.Context, oldID, newID uuid.UUID) error
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteExpired(ctx context.Context) (int64, error)
	RevokeAllForUser(ctx context.Context, userID uuid.UUID) (int64, error)
}

// MagicLinkRepository persists magic links in Postgres.
type MagicLinkRepository interface {
	Create(ctx context.Context, link *MagicLink) error
	GetByToken(ctx context.Context, token string) (*MagicLink, error)
	GetActiveByUserID(ctx context.Context, userID uuid.UUID) (*MagicLink, error)
	MarkUsed(ctx context.Context, id uuid.UUID) error
	// InvalidateUnusedForUser atomically marks all unused links for a user as used.
	// Called before creating a fresh link so two valid links never coexist (H5).
	// Returns the count of links invalidated.
	InvalidateUnusedForUser(ctx context.Context, userID uuid.UUID) (int64, error)
}

// SessionCache provides Redis-backed session caching.
type SessionCache interface {
	GetSession(ctx context.Context, id uuid.UUID) (*Session, error)
	SetSession(ctx context.Context, session *Session) error
	DeleteSession(ctx context.Context, id uuid.UUID) error
	IsRevoked(ctx context.Context, jti uuid.UUID) (bool, error)
	SetRevoked(ctx context.Context, jti uuid.UUID) error
}


// RateLimiter provides rate limiting for auth endpoints.
type RateLimiter interface {
	Allow(ctx context.Context, key string, cfg RateLimitConfig) (*RateLimitResult, error)
	RecordFailure(ctx context.Context, key string, cfg RateLimitConfig) error
	Reset(ctx context.Context, key string) error
}

// ResetRequestRepository persists password reset requests in PostgreSQL.
type ResetRequestRepository interface {
	Create(ctx context.Context, req *PasswordResetRequest) error
	Get(ctx context.Context, id uuid.UUID) (*PasswordResetRequest, error)
	GetPendingByUser(ctx context.Context, userID uuid.UUID) (*PasswordResetRequest, error)
	ListPending(ctx context.Context) ([]PasswordResetRequest, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status ResetRequestStatus, resolvedBy uuid.UUID) error
	SetMagicLink(ctx context.Context, id, magicLinkID uuid.UUID) error
	ExpireOld(ctx context.Context) (int64, error)
}

// ResetRequestService handles password reset request lifecycle.
type ResetRequestService interface {
	RequestReset(ctx context.Context, email, reason string) error
	ListPending(ctx context.Context) ([]PasswordResetRequest, error)
	Approve(ctx context.Context, requestID, adminID uuid.UUID) (*MagicLink, error)
	Reject(ctx context.Context, requestID, adminID uuid.UUID) error
	ExpireOld(ctx context.Context) (int64, error)
}
