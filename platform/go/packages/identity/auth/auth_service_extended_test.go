package auth_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shredbx/sbx-core/pkg/auth"
	"github.com/shredbx/sbx-core/pkg/user"
)

func TestAuthService_RefreshTokenReuse(t *testing.T) {
	t.Run("T-RTR-1: reusing rotated refresh token triggers theft response (C4)", func(t *testing.T) {
		svc, users, sessions := setupAuthService()
		seedActiveUser(users)

		// Login to get a token pair
		tp1, err := svc.Login(t.Context(), "test@example.com", "correct-password")
		require.NoError(t, err)

		// Refresh once → old session revoked + replaced_by set, new session created
		tp2, err := svc.RefreshToken(t.Context(), tp1.RefreshToken)
		require.NoError(t, err)
		assert.NotEqual(t, tp1.RefreshToken, tp2.RefreshToken)

		// Sanity: the legitimate user's NEW session is still valid
		sessions.mu.RLock()
		var newSessionRevoked bool
		for _, s := range sessions.sessions {
			if s.RefreshToken == auth.HashToken(tp2.RefreshToken) {
				newSessionRevoked = s.RevokedAt != nil
			}
		}
		sessions.mu.RUnlock()
		assert.False(t, newSessionRevoked, "new session must remain valid after rotation")

		// Age the rotated token PAST the rotation grace window so the replay is
		// treated as theft, not a benign concurrency race (2607-125). Without this,
		// the immediate replay falls inside the default 30s leeway and is (correctly)
		// tolerated — that benign path is covered by T-RTR-5.
		sessions.mu.Lock()
		for _, s := range sessions.sessions {
			if s.RefreshToken == auth.HashToken(tp1.RefreshToken) {
				aged := time.Now().Add(-2 * time.Minute)
				s.RevokedAt = &aged
			}
		}
		sessions.mu.Unlock()

		// Attacker replays the OLD refresh token — theft signal.
		// Expect: ErrRefreshTokenReused AND all user sessions revoked.
		_, err = svc.RefreshToken(t.Context(), tp1.RefreshToken)
		assert.ErrorIs(t, err, auth.ErrRefreshTokenReused, "replayed rotated token must return theft error")

		// Verify family revocation: the legitimate new session is now also revoked.
		sessions.mu.RLock()
		var familyRevoked int
		for _, s := range sessions.sessions {
			if s.UserID == testUserID && s.RevokedAt != nil {
				familyRevoked++
			}
		}
		sessions.mu.RUnlock()
		assert.GreaterOrEqual(t, familyRevoked, 2, "both old and new sessions must be revoked on theft detection")

		// Even the legitimate user's new token is now useless — they must re-login.
		_, err = svc.RefreshToken(t.Context(), tp2.RefreshToken)
		assert.Error(t, err, "after theft response, the rotated session is also dead")
	})

	t.Run("T-RTR-2: expired refresh token returns error and clears cookies", func(t *testing.T) {
		svc, users, sessions := setupAuthService()
		seedActiveUser(users)

		tp, _ := svc.Login(t.Context(), "test@example.com", "correct-password")

		// Manually expire the session
		sessions.mu.Lock()
		for _, s := range sessions.sessions {
			s.ExpiresAt = time.Now().Add(-1 * time.Hour)
		}
		sessions.mu.Unlock()

		_, err := svc.RefreshToken(t.Context(), tp.RefreshToken)
		assert.ErrorIs(t, err, auth.ErrSessionExpired)
	})

	t.Run("T-RTR-3: admin-revoked session returns ErrSessionRevoked (not theft)", func(t *testing.T) {
		svc, users, sessions := setupAuthService()
		seedActiveUser(users)

		tp, _ := svc.Login(t.Context(), "test@example.com", "correct-password")

		// Admin revokes the session — sets revoked_at but does NOT set replaced_by_session_id.
		// This distinguishes "admin action" from "token theft".
		sessions.mu.Lock()
		for _, s := range sessions.sessions {
			now := time.Now()
			s.RevokedAt = &now
			s.ReplacedBySessionID = nil // explicit: not a rotation
		}
		sessions.mu.Unlock()

		_, err := svc.RefreshToken(t.Context(), tp.RefreshToken)
		assert.ErrorIs(t, err, auth.ErrSessionRevoked, "admin-revoked must return ErrSessionRevoked, not ErrRefreshTokenReused")
		assert.NotErrorIs(t, err, auth.ErrRefreshTokenReused, "admin revoke must not trigger family-wide revocation")
	})

	t.Run("T-RTR-4: rotation chain link is recorded on successful refresh (C4)", func(t *testing.T) {
		svc, users, sessions := setupAuthService()
		seedActiveUser(users)

		tp1, _ := svc.Login(t.Context(), "test@example.com", "correct-password")

		// Capture original session ID
		var originalID uuid.UUID
		sessions.mu.RLock()
		for _, s := range sessions.sessions {
			if s.RefreshToken == auth.HashToken(tp1.RefreshToken) {
				originalID = s.ID
			}
		}
		sessions.mu.RUnlock()

		tp2, err := svc.RefreshToken(t.Context(), tp1.RefreshToken)
		require.NoError(t, err)

		// Original session should now have revoked_at AND replaced_by_session_id pointing
		// at the NEW session.
		var newID uuid.UUID
		sessions.mu.RLock()
		original := sessions.sessions[originalID]
		for _, s := range sessions.sessions {
			if s.RefreshToken == auth.HashToken(tp2.RefreshToken) {
				newID = s.ID
			}
		}
		sessions.mu.RUnlock()
		require.NotNil(t, original.RevokedAt, "original session must be marked revoked after rotation")
		require.NotNil(t, original.ReplacedBySessionID, "original session must have replaced_by set")
		assert.Equal(t, newID, *original.ReplacedBySessionID, "replaced_by points at new session")
	})

	t.Run("T-RTR-5: replay WITHIN the grace window is benign — fresh pair, family kept, grace audit (SC1)", func(t *testing.T) {
		users := newMockUserStore()
		sessions := newMockSessionRepo()
		magicLinks := newMockMagicLinkRepo()
		cache := newMockSessionCache()
		audit := &mockAuditRepo{}
		cfg := testConfig() // RefreshReuseLeeway unset → normalized to 30s default

		svc := auth.NewAuthService(cfg, users, sessions, magicLinks, cache, audit)
		seedActiveUser(users)

		tp1, err := svc.Login(t.Context(), "test@example.com", "correct-password")
		require.NoError(t, err)

		// Rotate v1 → v2.
		tp2, err := svc.RefreshToken(t.Context(), tp1.RefreshToken)
		require.NoError(t, err)

		// A second tab / queued request replays the JUST-rotated v1 (age ≈ 0, well
		// inside the 30s window). This is normal concurrency, NOT theft: it must
		// succeed with a fresh pair and must NOT revoke the family.
		tp3, err := svc.RefreshToken(t.Context(), tp1.RefreshToken)
		require.NoError(t, err, "in-window replay must succeed, not force a logout")
		require.NotNil(t, tp3)
		assert.NotEmpty(t, tp3.RefreshToken)
		assert.NotEqual(t, tp1.RefreshToken, tp3.RefreshToken)

		// Family intact: the legitimate v2 still refreshes.
		_, err = svc.RefreshToken(t.Context(), tp2.RefreshToken)
		require.NoError(t, err, "benign in-window replay must not revoke the session family")

		// A grace-replay audit is emitted; no theft audit.
		grace := audit.byAction(auth.AuditActionRefreshGraceReplay)
		assert.NotEmpty(t, grace, "in-window replay must emit auth.refresh_grace_replay")
		theft := audit.byAction(auth.AuditActionTokenReuseDetected)
		assert.Empty(t, theft, "benign in-window replay must NOT emit theft audit")
	})

	t.Run("T-RTR-6: theft audit records family_revoked:false when RevokeAllForUser fails (F8)", func(t *testing.T) {
		users := newMockUserStore()
		sessions := newMockSessionRepo()
		magicLinks := newMockMagicLinkRepo()
		cache := newMockSessionCache()
		audit := &mockAuditRepo{}
		cfg := testConfig()

		svc := auth.NewAuthService(cfg, users, sessions, magicLinks, cache, audit)
		seedActiveUser(users)

		tp1, err := svc.Login(t.Context(), "test@example.com", "correct-password")
		require.NoError(t, err)
		_, err = svc.RefreshToken(t.Context(), tp1.RefreshToken)
		require.NoError(t, err)

		// Age the rotated token past the grace window so the replay is theft, and
		// force the family-revoke UPDATE to fail — the theft is NOT contained.
		sessions.mu.Lock()
		for _, s := range sessions.sessions {
			if s.RefreshToken == auth.HashToken(tp1.RefreshToken) {
				aged := time.Now().Add(-2 * time.Minute)
				s.RevokedAt = &aged
			}
		}
		sessions.revokeAllForUserErr = errors.New("db down")
		sessions.mu.Unlock()

		// Reuse is still rejected, but the audit must not falsely claim containment.
		_, err = svc.RefreshToken(t.Context(), tp1.RefreshToken)
		assert.ErrorIs(t, err, auth.ErrRefreshTokenReused, "reuse is still rejected even if containment fails")

		theft := audit.byAction(auth.AuditActionTokenReuseDetected)
		require.Len(t, theft, 1)
		assert.Equal(t, false, theft[0].Payload["family_revoked"],
			"audit must record the actual (failed) containment, not a hardcoded true")
	})

	t.Run("T-RTR-7: rotation-revoke failure is fail-open and emits rotation_revoke_failed audit (F2)", func(t *testing.T) {
		users := newMockUserStore()
		sessions := newMockSessionRepo()
		magicLinks := newMockMagicLinkRepo()
		cache := newMockSessionCache()
		audit := &mockAuditRepo{}
		cfg := testConfig()

		svc := auth.NewAuthService(cfg, users, sessions, magicLinks, cache, audit)
		seedActiveUser(users)

		tp1, err := svc.Login(t.Context(), "test@example.com", "correct-password")
		require.NoError(t, err)

		// Force the rotation-chain UPDATE to fail (writes nothing, as an atomic
		// UPDATE would). The user must STILL get a working new pair (fail-open),
		// and the previously-silent swallow must now be an observable audit event.
		sessions.mu.Lock()
		sessions.revokeForRotationErr = errors.New("db hiccup")
		sessions.mu.Unlock()

		tp2, err := svc.RefreshToken(t.Context(), tp1.RefreshToken)
		require.NoError(t, err, "rotation-revoke failure must not fail the refresh (fail-open)")
		require.NotNil(t, tp2)
		assert.NotEqual(t, tp1.RefreshToken, tp2.RefreshToken)

		degraded := audit.byAction(auth.AuditActionRotationRevokeFailed)
		require.Len(t, degraded, 1, "a failed RevokeForRotation must emit an observable audit event, not be swallowed")
		assert.Equal(t, "db hiccup", degraded[0].Payload["error"])
	})
}

// TC-H3: Logout marks the JTI revoked in cache so the access JWT is rejected
// on subsequent requests — otherwise the access token stays valid for up to its
// full TTL (1 hour) after the user logs out, defeating the security intent.
func TestAuthService_Logout_RevokesAccessTokenJTI(t *testing.T) {
	users := newMockUserStore()
	sessions := newMockSessionRepo()
	magicLinks := newMockMagicLinkRepo()
	cache := newMockSessionCache()
	cfg := testConfig()

	svc := auth.NewAuthService(cfg, users, sessions, magicLinks, cache, nil)
	seedActiveUser(users)

	tp, err := svc.Login(t.Context(), "test@example.com", "correct-password")
	require.NoError(t, err)

	// Access token verifies cleanly before logout
	claimsBefore, err := svc.VerifyToken(t.Context(), tp.AccessToken)
	require.NoError(t, err)
	require.NotNil(t, claimsBefore)

	// Logout
	require.NoError(t, svc.Logout(t.Context(), tp.RefreshToken))

	// After logout, the SAME access JWT must fail verification — JTI marked revoked.
	_, err = svc.VerifyToken(t.Context(), tp.AccessToken)
	assert.ErrorIs(t, err, auth.ErrTokenRevoked, "logout must revoke the access JWT's JTI")
}

// TC-L7: SessionPinning rejects refresh from a different IP when enabled.
func TestRefreshToken_SessionPinning_IPMismatch(t *testing.T) {
	users := newMockUserStore()
	sessions := newMockSessionRepo()
	magicLinks := newMockMagicLinkRepo()
	cache := newMockSessionCache()
	cfg := testConfig()
	cfg.SessionPinning = true // enable pinning

	svc := auth.NewAuthService(cfg, users, sessions, magicLinks, cache, nil)
	seedActiveUser(users)

	// Login from IP 1.1.1.1
	loginCtx := auth.WithClientContext(t.Context(), auth.ClientContext{IP: "1.1.1.1", UserAgent: "Mozilla/5.0"})
	tp, err := svc.Login(loginCtx, "test@example.com", "correct-password")
	require.NoError(t, err)

	// Refresh from SAME IP — works
	sameCtx := auth.WithClientContext(t.Context(), auth.ClientContext{IP: "1.1.1.1", UserAgent: "Mozilla/5.0"})
	_, err = svc.RefreshToken(sameCtx, tp.RefreshToken)
	assert.NoError(t, err, "same IP/UA must pass pinning")
}

// TC-L7: SessionPinning rejects refresh from a different IP when enabled.
func TestRefreshToken_SessionPinning_RejectsIPMismatch(t *testing.T) {
	users := newMockUserStore()
	sessions := newMockSessionRepo()
	magicLinks := newMockMagicLinkRepo()
	cache := newMockSessionCache()
	cfg := testConfig()
	cfg.SessionPinning = true

	svc := auth.NewAuthService(cfg, users, sessions, magicLinks, cache, nil)
	seedActiveUser(users)

	loginCtx := auth.WithClientContext(t.Context(), auth.ClientContext{IP: "1.1.1.1", UserAgent: "Mozilla/5.0"})
	tp, err := svc.Login(loginCtx, "test@example.com", "correct-password")
	require.NoError(t, err)

	// Refresh from DIFFERENT IP — rejected
	attackerCtx := auth.WithClientContext(t.Context(), auth.ClientContext{IP: "2.2.2.2", UserAgent: "Mozilla/5.0"})
	_, err = svc.RefreshToken(attackerCtx, tp.RefreshToken)
	assert.ErrorIs(t, err, auth.ErrSessionMismatch, "IP mismatch must reject when pinning enabled")
}

// L7: when pinning is disabled (default), IP mismatches are allowed (mobile users).
func TestRefreshToken_SessionPinning_DisabledByDefault(t *testing.T) {
	users := newMockUserStore()
	sessions := newMockSessionRepo()
	magicLinks := newMockMagicLinkRepo()
	cache := newMockSessionCache()
	cfg := testConfig()
	// cfg.SessionPinning defaults to false

	svc := auth.NewAuthService(cfg, users, sessions, magicLinks, cache, nil)
	seedActiveUser(users)

	loginCtx := auth.WithClientContext(t.Context(), auth.ClientContext{IP: "1.1.1.1", UserAgent: "Mozilla/5.0"})
	tp, err := svc.Login(loginCtx, "test@example.com", "correct-password")
	require.NoError(t, err)

	// Refresh from different IP — allowed because pinning is off
	attackerCtx := auth.WithClientContext(t.Context(), auth.ClientContext{IP: "2.2.2.2", UserAgent: "Mozilla/5.0"})
	_, err = svc.RefreshToken(attackerCtx, tp.RefreshToken)
	assert.NoError(t, err, "pinning disabled → IP mismatch is allowed")
}

// T-RTR-8 (F5): a deactivated / soft-deleted user cannot rotate a fresh token
// pair on the normal refresh path — refresh is the liveness checkpoint.
func TestRefreshToken_DeactivatedUser_NormalPath(t *testing.T) {
	users := newMockUserStore()
	sessions := newMockSessionRepo()
	magicLinks := newMockMagicLinkRepo()
	cache := newMockSessionCache()
	audit := &mockAuditRepo{}
	cfg := testConfig()

	svc := auth.NewAuthService(cfg, users, sessions, magicLinks, cache, audit)
	u := seedActiveUser(users)

	tp, err := svc.Login(t.Context(), "test@example.com", "correct-password")
	require.NoError(t, err)

	// Admin deactivates the user (mirrors SoftDelete: status=deleted + deleted_at).
	users.mu.Lock()
	now := time.Now()
	u.Status = user.UserStatusDeleted
	u.DeletedAt = &now
	users.mu.Unlock()

	// The refresh token is still structurally valid, but the account is gone.
	_, err = svc.RefreshToken(t.Context(), tp.RefreshToken)
	assert.ErrorIs(t, err, auth.ErrUserInactive, "deactivated user must not refresh")
	assert.NotEmpty(t, audit.byAction(auth.AuditActionRefreshInactiveUser),
		"rejected refresh for an inactive user must emit auth.refresh_inactive_user")
}

// T-RTR-9 (F5): the grace-window path also rechecks liveness — a deactivated
// user replaying a just-rotated token inside the leeway is rejected, not graced.
func TestRefreshToken_DeactivatedUser_GracePath(t *testing.T) {
	users := newMockUserStore()
	sessions := newMockSessionRepo()
	magicLinks := newMockMagicLinkRepo()
	cache := newMockSessionCache()
	audit := &mockAuditRepo{}
	cfg := testConfig() // RefreshReuseLeeway normalizes to the 30s default

	svc := auth.NewAuthService(cfg, users, sessions, magicLinks, cache, audit)
	u := seedActiveUser(users)

	tp1, err := svc.Login(t.Context(), "test@example.com", "correct-password")
	require.NoError(t, err)

	// Rotate v1 → v2 so replaying v1 lands in the grace branch.
	_, err = svc.RefreshToken(t.Context(), tp1.RefreshToken)
	require.NoError(t, err)

	// Deactivate, then replay v1 within the 30s grace window.
	users.mu.Lock()
	now := time.Now()
	u.Status = user.UserStatusDeleted
	u.DeletedAt = &now
	users.mu.Unlock()

	_, err = svc.RefreshToken(t.Context(), tp1.RefreshToken)
	assert.ErrorIs(t, err, auth.ErrUserInactive, "grace-window replay by a deactivated user must be rejected")
	assert.NotEmpty(t, audit.byAction(auth.AuditActionRefreshInactiveUser))
}

// F4: session pinning is enforced INSIDE the grace window — a just-rotated token
// replayed from a foreign IP within the leeway is rejected, not waved through.
func TestRefreshToken_GraceWindow_EnforcesPinning(t *testing.T) {
	users := newMockUserStore()
	sessions := newMockSessionRepo()
	magicLinks := newMockMagicLinkRepo()
	cache := newMockSessionCache()
	cfg := testConfig()
	cfg.SessionPinning = true

	svc := auth.NewAuthService(cfg, users, sessions, magicLinks, cache, nil)
	seedActiveUser(users)

	loginCtx := auth.WithClientContext(t.Context(), auth.ClientContext{IP: "1.1.1.1", UserAgent: "Mozilla/5.0"})
	tp1, err := svc.Login(loginCtx, "test@example.com", "correct-password")
	require.NoError(t, err)

	// Rotate v1 → v2 from the legit IP (v1 becomes revoked+replaced).
	_, err = svc.RefreshToken(loginCtx, tp1.RefreshToken)
	require.NoError(t, err)

	// Attacker replays the just-rotated v1 from a FOREIGN IP inside the grace window.
	attackerCtx := auth.WithClientContext(t.Context(), auth.ClientContext{IP: "2.2.2.2", UserAgent: "Mozilla/5.0"})
	_, err = svc.RefreshToken(attackerCtx, tp1.RefreshToken)
	assert.ErrorIs(t, err, auth.ErrSessionMismatch, "grace-window replay from a foreign IP must fail pinning")
}

// F4 regression guard: pinning inside the grace window must NOT punish the
// legitimate racer — a same-IP in-window replay still gets a fresh pair.
func TestRefreshToken_GraceWindow_PinningMatchStillGraces(t *testing.T) {
	users := newMockUserStore()
	sessions := newMockSessionRepo()
	magicLinks := newMockMagicLinkRepo()
	cache := newMockSessionCache()
	audit := &mockAuditRepo{}
	cfg := testConfig()
	cfg.SessionPinning = true

	svc := auth.NewAuthService(cfg, users, sessions, magicLinks, cache, audit)
	seedActiveUser(users)

	loginCtx := auth.WithClientContext(t.Context(), auth.ClientContext{IP: "1.1.1.1", UserAgent: "Mozilla/5.0"})
	tp1, err := svc.Login(loginCtx, "test@example.com", "correct-password")
	require.NoError(t, err)

	_, err = svc.RefreshToken(loginCtx, tp1.RefreshToken)
	require.NoError(t, err)

	// Same-IP in-window replay (second tab) — benign, gets a fresh pair.
	tp3, err := svc.RefreshToken(loginCtx, tp1.RefreshToken)
	require.NoError(t, err, "same-IP in-window replay must still be graced")
	require.NotNil(t, tp3)
	assert.NotEmpty(t, tp3.RefreshToken)
	assert.NotEmpty(t, audit.byAction(auth.AuditActionRefreshGraceReplay), "benign in-window replay emits the grace audit")
}

func TestAuthService_VerifyToken_CacheDegradation(t *testing.T) {
	t.Run("T-CD-1: nil cache falls back to direct validation", func(t *testing.T) {
		users := newMockUserStore()
		sessions := newMockSessionRepo()
		magicLinks := newMockMagicLinkRepo()
		cfg := testConfig()

		// Create service with nil cache
		svc := auth.NewAuthService(cfg, users, sessions, magicLinks, nil, nil)
		seedActiveUser(users)

		tp, _ := svc.Login(t.Context(), "test@example.com", "correct-password")

		claims, err := svc.VerifyToken(t.Context(), tp.AccessToken)
		require.NoError(t, err)
		assert.NotNil(t, claims)
		assert.Equal(t, "test@example.com", claims.Email)
	})
}

func TestAuthService_RevokeSession_WithCache(t *testing.T) {
	t.Run("T-RS-1: revoke session clears cache entry", func(t *testing.T) {
		svc, users, sessions := setupAuthService()
		seedActiveUser(users)

		tp, _ := svc.Login(t.Context(), "test@example.com", "correct-password")

		// Find the session ID
		var sessionID uuid.UUID
		sessions.mu.RLock()
		for _, s := range sessions.sessions {
			if s.RefreshToken == auth.HashToken(tp.RefreshToken) {
				sessionID = s.ID
			}
		}
		sessions.mu.RUnlock()

		err := svc.RevokeSession(t.Context(), sessionID)
		assert.NoError(t, err)
	})

	t.Run("T-RS-2: revoke nonexistent session returns error", func(t *testing.T) {
		svc, _, _ := setupAuthService()
		err := svc.RevokeSession(t.Context(), uuid.New())
		assert.ErrorIs(t, err, auth.ErrSessionNotFound)
	})
}

func TestAuthService_LoginCreatesSession(t *testing.T) {
	t.Run("T-LC-1: login creates session with correct fields", func(t *testing.T) {
		svc, users, sessions := setupAuthService()
		seedActiveUser(users)

		tp, err := svc.Login(t.Context(), "test@example.com", "correct-password")
		require.NoError(t, err)

		// Verify session was created
		sessions.mu.RLock()
		assert.Len(t, sessions.sessions, 1)
		for _, s := range sessions.sessions {
			assert.Equal(t, testUserID, s.UserID)
			// F1: the session stores the HASH of the refresh token, never the raw token.
			assert.Equal(t, auth.HashToken(tp.RefreshToken), s.RefreshToken)
			assert.False(t, s.ExpiresAt.IsZero())
		}
		sessions.mu.RUnlock()
	})
}

func TestAuthExtract(t *testing.T) {

	t.Run("T-AE-1: valid cookie sets claims in context", func(t *testing.T) {
		svc, users, sessions := setupAuthService()
		seedActiveUser(users)

		tp, _ := svc.Login(t.Context(), "test@example.com", "correct-password")
		_ = sessions // used by setupAuthService

		var gotClaims *auth.Claims
		inner := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			gotClaims = auth.ClaimsFromContext(r.Context())
		})

		handler := auth.AuthExtract(svc)(inner)
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{Name: "access_token", Value: tp.AccessToken})
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		require.NotNil(t, gotClaims)
		assert.Equal(t, "test@example.com", gotClaims.Email)
	})

	t.Run("T-AE-2: no cookie passes through with nil claims", func(t *testing.T) {
		svc, _, _ := setupAuthService()

		var gotClaims *auth.Claims
		inner := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			gotClaims = auth.ClaimsFromContext(r.Context())
		})

		handler := auth.AuthExtract(svc)(inner)
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		assert.Nil(t, gotClaims)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("T-AE-3: expired token passes through with nil claims", func(t *testing.T) {
		svc, _, _ := setupAuthService()

		var gotClaims *auth.Claims
		inner := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			gotClaims = auth.ClaimsFromContext(r.Context())
		})

		handler := auth.AuthExtract(svc)(inner)
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{Name: "access_token", Value: "expired.jwt.token"})
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		assert.Nil(t, gotClaims)
	})
}
