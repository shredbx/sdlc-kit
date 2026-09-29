// Package authfake is a fake AuthService that always authenticates as a fixed identity,
// regardless of what a caller presents. It exists to preserve the RBAC seam (RequireAuth,
// Claims.Role) before real authentication (magic link / password, roadmap M6) is built: a
// consumer wires this in during development, and swapping it for the real Postgres-backed
// AuthService later touches only which implementation is constructed — nothing in the routes
// that call RequireAuth changes.
package authfake

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/shredbx/sbx-core/pkg/auth"
)

// Service is an auth.AuthService that always succeeds as the same fixed identity.
type Service struct {
	claims *auth.Claims
}

// New returns a Service that always authenticates as role (default "admin").
func New(role string) *Service {
	if role == "" {
		role = "admin"
	}
	now := time.Now()
	return &Service{claims: &auth.Claims{
		Sub:   uuid.Nil,
		Email: "dev@bos-demo.local",
		Role:  role,
		JTI:   uuid.Nil,
		IAT:   now.Unix(),
		EXP:   now.Add(24 * time.Hour).Unix(),
	}}
}

func (s *Service) Login(ctx context.Context, email, password string) (*auth.TokenPair, error) {
	return s.tokenPair(), nil
}

func (s *Service) Logout(ctx context.Context, refreshToken string) error { return nil }

func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (*auth.TokenPair, error) {
	return s.tokenPair(), nil
}

// VerifyToken always returns the fixed claims, regardless of accessToken — the point of this
// adapter (see the package doc).
func (s *Service) VerifyToken(ctx context.Context, accessToken string) (*auth.Claims, error) {
	return s.claims, nil
}

func (s *Service) RevokeSession(ctx context.Context, sessionID uuid.UUID) error { return nil }

func (s *Service) ListSessions(ctx context.Context, userID uuid.UUID) ([]auth.Session, error) {
	return nil, nil
}

func (s *Service) tokenPair() *auth.TokenPair {
	now := time.Now()
	return &auth.TokenPair{
		AccessToken:      "fake",
		RefreshToken:     "fake",
		AccessExpiresAt:  now.Add(24 * time.Hour),
		RefreshExpiresAt: now.Add(24 * time.Hour),
		TokenType:        "Bearer",
	}
}

// RoleCookie is the dev-only role override authfake reads (see Always). There is no real login to
// switch identities with yet, so an admin UI can set this cookie to act as a different fake role
// without restarting the process. Real auth has no equivalent — a real session's role comes from
// the database, not a cookie a client controls.
const RoleCookie = "bosdemo_role"

// Always is middleware that puts a fixed dev identity into every request's context
// unconditionally, so a downstream auth.RequireAuth resolves without a client needing to
// present any cookie or header. auth.AuthExtract is not used for this: it only calls
// VerifyToken when a caller already sent an access_token cookie, which a fixed dev identity
// has no reason to require. If RoleCookie is present, its value overrides Service's own role for
// this request only — nothing else about the fixed identity changes.
func Always(svc *Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, _ := svc.VerifyToken(r.Context(), "")
			if c, err := r.Cookie(RoleCookie); err == nil && c.Value != "" {
				override := *claims
				override.Role = c.Value
				claims = &override
			}
			ctx := auth.SetClaimsInContext(r.Context(), claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
