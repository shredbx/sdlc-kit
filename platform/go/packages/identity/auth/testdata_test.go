package auth_test

import (
	"time"

	"github.com/google/uuid"
	"github.com/shredbx/sbx-core/pkg/auth"
)

// Test fixtures for auth package.

var testUserID = uuid.MustParse("11111111-1111-1111-1111-111111111111")
var testJTI = uuid.MustParse("22222222-2222-2222-2222-222222222222")

func testConfig() auth.AuthConfig {
	return auth.AuthConfig{
		JWTSecret:       "test-secret-minimum-32-bytes!!!!",
		AccessTokenTTL:  time.Hour,
		RefreshTokenTTL: 7 * 24 * time.Hour,
		MagicLinkTTL:    72 * time.Hour,
		BcryptCost:      4, // low cost for fast tests
		IsProduction:    false,
		CookieDomain:    "localhost",
	}
}

func testClaims() auth.Claims {
	return auth.Claims{
		Sub:   testUserID,
		Email: "test@example.com",
		Role:  "admin",
		JTI:   testJTI,
	}
}

func testMagicLink(status auth.MagicLinkStatus) *auth.MagicLink {
	now := time.Now()
	ml := &auth.MagicLink{
		ID:        uuid.New(),
		Token:     "test-token-abc123",
		UserID:    testUserID,
		CreatedAt: now,
	}
	switch status {
	case auth.MagicLinkStatusValid:
		ml.ExpiresAt = now.Add(72 * time.Hour)
	case auth.MagicLinkStatusExpired:
		ml.ExpiresAt = now.Add(-1 * time.Hour)
	case auth.MagicLinkStatusUsed:
		ml.ExpiresAt = now.Add(72 * time.Hour)
		usedAt := now.Add(-30 * time.Minute)
		ml.UsedAt = &usedAt
	}
	return ml
}

func testSession() *auth.Session {
	now := time.Now()
	return &auth.Session{
		ID:           uuid.New(),
		UserID:       testUserID,
		RefreshToken: "test-refresh-token",
		JTI:          testJTI,
		ExpiresAt:    now.Add(7 * 24 * time.Hour),
		CreatedAt:    now,
	}
}
