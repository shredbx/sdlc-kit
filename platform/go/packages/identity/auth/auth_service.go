package auth

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/shredbx/sbx-core/pkg/user"
)

type authService struct {
	cfg        AuthConfig
	users      user.UserStore
	sessions   SessionRepository
	magicLinks MagicLinkRepository
	cache      SessionCache
	audit      AuditRepository // M7: optional; nil = no audit emission
	dummyHash  string          // pre-computed valid bcrypt hash for timing-safe dummy comparison
}

// NewAuthService creates an auth service with all dependencies. audit may be nil
// (no audit emission). M7.
func NewAuthService(cfg AuthConfig, users user.UserStore, sessions SessionRepository, magicLinks MagicLinkRepository, cache SessionCache, audit AuditRepository) AuthService {
	// Normalize safe non-zero defaults (currently the rotation grace window) so a
	// caller that never set RefreshReuseLeeway still gets a working grace instead
	// of the zero-window full-family-logout failure mode.
	cfg.Defaults()
	// Pre-compute a real bcrypt hash so dummy comparisons take the same time
	// as real ones — prevents user enumeration via timing (FIX-1).
	dummyHash, _ := HashPassword("timing-safe-dummy-password", cfg.BcryptCost)
	return &authService{
		cfg:        cfg,
		users:      users,
		sessions:   sessions,
		magicLinks: magicLinks,
		cache:      cache,
		audit:      audit,
		dummyHash:  dummyHash,
	}
}

// emitAudit writes an audit row, swallowing errors (audit failure must not
// block the security-critical operation it's recording).
func (s *authService) emitAudit(ctx context.Context, e *AuditEvent) {
	if s.audit == nil {
		return
	}
	_ = s.audit.Append(ctx, e)
}

func (s *authService) Login(ctx context.Context, email, password string) (*TokenPair, error) {
	u, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		// Timing-safe: always run bcrypt even for nonexistent users
		_ = VerifyPassword(s.dummyHash, password)
		s.emitAudit(ctx, &AuditEvent{
			Action:  AuditActionLoginFailure,
			Payload: map[string]any{"email": email, "reason": "unknown_user"},
		})
		return nil, ErrInvalidCredentials
	}

	if !u.IsActive() {
		_ = VerifyPassword(s.dummyHash, password)
		s.emitAudit(ctx, &AuditEvent{
			ActorID: &u.ID,
			Action:  AuditActionLoginFailure,
			Payload: map[string]any{"email": email, "reason": "inactive_user"},
		})
		return nil, ErrInvalidCredentials
	}

	if err := VerifyPassword(u.PasswordHash, password); err != nil {
		s.emitAudit(ctx, &AuditEvent{
			ActorID: &u.ID,
			Action:  AuditActionLoginFailure,
			Payload: map[string]any{"email": email, "reason": "wrong_password"},
		})
		return nil, ErrInvalidCredentials
	}

	claims := &Claims{
		Sub:   u.ID,
		Email: u.Email,
		Role:  u.Role,
	}

	tp, _, err := s.createSessionAndTokens(ctx, claims)
	if err != nil {
		return nil, err
	}

	_ = s.users.UpdateLastLogin(ctx, u.ID)
	s.emitAudit(ctx, &AuditEvent{
		ActorID: &u.ID,
		Action:  AuditActionLoginSuccess,
	})

	return tp, nil
}

// createSessionAndTokens issues a token pair and persists the session.
// Shared between Login, CompleteOnboarding, and RefreshToken (C4 needs the
// new session ID for the rotation chain, so this returns it explicitly).
func (s *authService) createSessionAndTokens(ctx context.Context, claims *Claims) (*TokenPair, *Session, error) {
	tp, err := NewTokenPair(claims, s.cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("create token pair: %w", err)
	}

	session := &Session{
		ID:     uuid.New(),
		UserID: claims.Sub,
		// F1: store the SHA-256 hash at rest, never the raw bearer token. The
		// client holds the raw token; Refresh/Logout hash on lookup (mirrors the
		// M5 magic-link pattern). A DB leak/backup/log must not yield live tokens.
		RefreshToken: HashToken(tp.RefreshToken),
		JTI:          claims.JTI, // Now correctly set by NewTokenPair
		ExpiresAt:    tp.RefreshExpiresAt,
		CreatedAt:    time.Now(),
	}

	// L7: capture client fingerprint at session creation for later pinning check.
	if cc := ClientContextFromContext(ctx); cc.IP != "" || cc.UserAgent != "" {
		if cc.IP != "" {
			ip := cc.IP
			session.IPAddress = &ip
		}
		if cc.UserAgent != "" {
			ua := cc.UserAgent
			session.UserAgent = &ua
		}
	}

	if err := s.sessions.Create(ctx, session); err != nil {
		return nil, nil, fmt.Errorf("create session: %w", err)
	}

	if s.cache != nil {
		// F1: the session cache is keyed by ID and never looked up by refresh
		// token, so it must not carry the token at all — strip it from the copy.
		cached := *session
		cached.RefreshToken = ""
		_ = s.cache.SetSession(ctx, &cached)
	}

	return tp, session, nil
}

func (s *authService) Logout(ctx context.Context, refreshToken string) error {
	// F1: sessions are stored by hashed refresh token — hash the raw token to look up.
	session, err := s.sessions.GetByRefreshToken(ctx, HashToken(refreshToken))
	if err != nil {
		return nil // idempotent — already logged out
	}

	// H3: mark JTI revoked in cache BEFORE deleting the session row. Otherwise
	// the access JWT (60-min TTL) stays valid after logout — defeating the
	// user's "log me out now" intent (e.g., stolen laptop scenario).
	if s.cache != nil {
		_ = s.cache.SetRevoked(ctx, session.JTI)
	}

	if err := s.sessions.Delete(ctx, session.ID); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}

	if s.cache != nil {
		_ = s.cache.DeleteSession(ctx, session.ID)
	}

	return nil
}

// checkSessionPinning enforces optional L7 IP/UA pinning. When SessionPinning is
// enabled and the request's IP/UA diverges from the values captured at login it
// emits an auth.session_mismatch audit and returns ErrSessionMismatch; otherwise
// it returns nil. Applied on BOTH the normal rotation path and the grace-window
// replay path (F4) so a stolen just-rotated token replayed from a foreign client
// inside the grace window is not waved through.
func (s *authService) checkSessionPinning(ctx context.Context, session *Session) error {
	if !s.cfg.SessionPinning {
		return nil
	}
	cc := ClientContextFromContext(ctx)
	mismatch := false
	if session.IPAddress != nil && cc.IP != "" && *session.IPAddress != cc.IP {
		mismatch = true
	}
	if session.UserAgent != nil && cc.UserAgent != "" && *session.UserAgent != cc.UserAgent {
		mismatch = true
	}
	if !mismatch {
		return nil
	}
	s.emitAudit(ctx, &AuditEvent{
		ActorID:    &session.UserID,
		Action:     "auth.session_mismatch",
		TargetType: "session",
		TargetID:   &session.ID,
		IPAddress:  cc.IP,
		UserAgent:  cc.UserAgent,
	})
	return ErrSessionMismatch
}

func (s *authService) RefreshToken(ctx context.Context, refreshToken string) (*TokenPair, error) {
	// F1: sessions are stored by hashed refresh token — hash the raw token to look up.
	session, err := s.sessions.GetByRefreshToken(ctx, HashToken(refreshToken))
	if err != nil {
		return nil, ErrSessionNotFound
	}

	// C4: theft detection with a rotation grace window. A session that has been
	// rotated (ReplacedBySessionID set) being presented again is EITHER a benign
	// concurrency race OR token theft.
	//
	// Within RefreshReuseLeeway of the rotation it is benign — a second tab, a
	// queued request, a lost/discarded Set-Cookie, or a rolling-deploy overlap
	// replaying the just-rotated token — so issue a fresh pair and DO NOT revoke
	// the family. Mirrors Auth0's "Rotation Overlap Period" / Okta's grace period.
	// (The DB stores only the token HASH at rest, so the child's raw token can't be
	// re-issued; a fresh independent session for the racer is the equivalent, and
	// converges as the client settles on the newest cookie.)
	//
	// After the window it is theft: revoke ALL sessions for this user
	// (RFC 6819 §5.2.2.3 / RFC 9700).
	if session.RevokedAt != nil && session.ReplacedBySessionID != nil {
		if s.cfg.RefreshReuseLeeway > 0 && time.Since(*session.RevokedAt) <= s.cfg.RefreshReuseLeeway {
			// F4: a pin mismatch INSIDE the grace window is a strong theft signal —
			// the benign-replay shortcut must not skip the L7 IP/UA check the normal
			// rotation path enforces below.
			if err := s.checkSessionPinning(ctx, session); err != nil {
				return nil, err
			}
			u, err := s.users.Get(ctx, session.UserID)
			if err != nil {
				return nil, fmt.Errorf("get user for grace refresh: %w", err)
			}
			// F5: refresh is the liveness checkpoint — a deactivated / soft-deleted
			// user must not keep minting tokens via the grace path either.
			if !u.IsActive() {
				s.emitAudit(ctx, &AuditEvent{
					ActorID:    &session.UserID,
					Action:     AuditActionRefreshInactiveUser,
					TargetType: "session",
					TargetID:   &session.ID,
				})
				return nil, ErrUserInactive
			}
			claims := &Claims{Sub: u.ID, Email: u.Email, Role: u.Role}
			tp, _, err := s.createSessionAndTokens(ctx, claims)
			if err != nil {
				return nil, err
			}
			s.emitAudit(ctx, &AuditEvent{
				ActorID:    &session.UserID,
				Action:     AuditActionRefreshGraceReplay,
				TargetType: "session",
				TargetID:   &session.ID,
				Payload:    map[string]any{"leeway_seconds": int(s.cfg.RefreshReuseLeeway.Seconds())},
			})
			return tp, nil
		}
		revokedCount, revokeErr := s.sessions.RevokeAllForUser(ctx, session.UserID)
		if revokeErr != nil {
			// The family-revoke UPDATE failed, so the sibling sessions are still
			// live — the theft was NOT contained. Do not let the audit assert a
			// containment that did not happen; record family_revoked:false so an
			// investigator is not misled. (Retry/alerting on this failure is a
			// separate owner decision.)
			log.Printf("auth: theft-response RevokeAllForUser failed for user %s: %v", session.UserID, revokeErr)
		}
		if s.cache != nil {
			_ = s.cache.SetRevoked(ctx, session.JTI)
		}
		s.emitAudit(ctx, &AuditEvent{
			ActorID:    &session.UserID,
			Action:     AuditActionTokenReuseDetected,
			TargetType: "session",
			TargetID:   &session.ID,
			Payload:    map[string]any{"family_revoked": revokeErr == nil, "sessions_revoked": revokedCount},
		})
		return nil, ErrRefreshTokenReused
	}

	// Admin-revoked (no rotation marker) — keep the old "session revoked" semantics,
	// don't escalate to family-revocation since this wasn't a theft signal.
	if session.RevokedAt != nil {
		return nil, ErrSessionRevoked
	}
	if time.Now().After(session.ExpiresAt) {
		return nil, ErrSessionExpired
	}

	// L7: optional session pinning (IP/UA), enforced identically on the grace path.
	if err := s.checkSessionPinning(ctx, session); err != nil {
		return nil, err
	}

	// Look up user for fresh claims
	u, err := s.users.Get(ctx, session.UserID)
	if err != nil {
		return nil, fmt.Errorf("get user for refresh: %w", err)
	}

	// F5: refresh is the liveness checkpoint — reject a deactivated / soft-deleted
	// user so a revoked account cannot keep rotating a fresh 1h access token
	// indefinitely (middleware/RBAC trust the JWT claims alone).
	if !u.IsActive() {
		s.emitAudit(ctx, &AuditEvent{
			ActorID:    &session.UserID,
			Action:     AuditActionRefreshInactiveUser,
			TargetType: "session",
			TargetID:   &session.ID,
		})
		return nil, ErrUserInactive
	}

	// Create new session FIRST so we have its ID to record in the rotation chain.
	claims := &Claims{
		Sub:   u.ID,
		Email: u.Email,
		Role:  u.Role,
	}
	tp, newSession, err := s.createSessionAndTokens(ctx, claims)
	if err != nil {
		return nil, err
	}

	// Mark the old session as revoked AND point it at the new session. If this
	// fails the new session is still valid, so we do NOT fail the request — the
	// user already has working tokens (fail-open by design). BUT the old refresh
	// token is now left un-rotated (valid until its TTL) and single-use reuse-
	// detection is disarmed for this lineage, so the degradation must not be
	// silent: log it and emit an audit event. (Failing CLOSED instead — refusing
	// the new pair / atomic revoke-then-mint — is a separate owner decision.)
	if revErr := s.sessions.RevokeForRotation(ctx, session.ID, newSession.ID); revErr != nil {
		log.Printf("auth: RevokeForRotation failed (old session %s left live): %v", session.ID, revErr)
		s.emitAudit(ctx, &AuditEvent{
			ActorID:    &session.UserID,
			Action:     AuditActionRotationRevokeFailed,
			TargetType: "session",
			TargetID:   &session.ID,
			Payload:    map[string]any{"error": revErr.Error(), "new_session_id": newSession.ID.String()},
		})
	}
	if s.cache != nil {
		// Invalidate the OLD session's cached entry + mark its JTI revoked so
		// any in-flight access token validations fail closed.
		_ = s.cache.DeleteSession(ctx, session.ID)
		_ = s.cache.SetRevoked(ctx, session.JTI)
	}

	return tp, nil
}

func (s *authService) VerifyToken(ctx context.Context, accessToken string) (*Claims, error) {
	if accessToken == "" {
		return nil, nil // no token = unauthenticated (OK for public routes)
	}

	claims, err := ParseToken(accessToken, s.cfg.JWTSecret)
	if err != nil {
		return nil, nil // expired/invalid = treat as unauthenticated on public routes
	}

	// Check revocation in cache
	if s.cache != nil {
		revoked, err := s.cache.IsRevoked(ctx, claims.JTI)
		if err == nil && revoked {
			return nil, ErrTokenRevoked
		}
	}

	return claims, nil
}

func (s *authService) ListSessions(ctx context.Context, userID uuid.UUID) ([]Session, error) {
	return s.sessions.ListByUser(ctx, userID)
}

func (s *authService) RevokeSession(ctx context.Context, sessionID uuid.UUID) error {
	session, err := s.sessions.Get(ctx, sessionID)
	if err != nil {
		return ErrSessionNotFound
	}

	if err := s.sessions.Revoke(ctx, sessionID); err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}

	if s.cache != nil {
		_ = s.cache.SetRevoked(ctx, session.JTI)
		_ = s.cache.DeleteSession(ctx, sessionID)
	}

	return nil
}
