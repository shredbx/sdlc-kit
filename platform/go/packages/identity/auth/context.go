package auth

import "context"

type contextKey struct{}

// SetClaimsInContext stores Claims in the context.
func SetClaimsInContext(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, contextKey{}, claims)
}

// ClaimsFromContext extracts Claims from the context. Returns nil if not authenticated.
// Callers that proceed without a nil check on the returned pointer risk a panic
// — prefer RequireClaims for handlers that already expect an authenticated user.
func ClaimsFromContext(ctx context.Context) *Claims {
	claims, _ := ctx.Value(contextKey{}).(*Claims)
	return claims
}

// RequireClaims returns the claims from context or a non-nil error if no
// authenticated user is present. L8: panic-safe alternative to ClaimsFromContext
// for handler code that should never deref a nil claims pointer.
func RequireClaims(ctx context.Context) (*Claims, error) {
	claims := ClaimsFromContext(ctx)
	if claims == nil {
		return nil, ErrTokenRevoked // any caller of RequireClaims is past AuthExtract,
		// so missing claims here means token was revoked or middleware was bypassed.
	}
	return claims, nil
}

// ClientContext carries IP and User-Agent for session pinning (L7). Callers
// (typically a Chi middleware) populate this from the HTTP request before
// invoking AuthService.Login or AuthService.RefreshToken. When empty, pinning
// is skipped — pinning is opt-in via AuthConfig.SessionPinning.
type ClientContext struct {
	IP        string
	UserAgent string
}

type clientCtxKey struct{}

// WithClientContext stores client IP/UA on the context. Idempotent — overrides
// any prior value.
func WithClientContext(ctx context.Context, c ClientContext) context.Context {
	return context.WithValue(ctx, clientCtxKey{}, c)
}

// ClientContextFromContext extracts client IP/UA from context. Returns the
// zero value when not set.
func ClientContextFromContext(ctx context.Context) ClientContext {
	c, _ := ctx.Value(clientCtxKey{}).(ClientContext)
	return c
}
