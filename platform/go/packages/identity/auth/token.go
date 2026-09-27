package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// HashToken returns the lowercase hex SHA-256 of token. Used to store magic
// link tokens in hashed form (M5) so a DB leak cannot be replayed as token
// theft. Matches the PostgreSQL backfill: encode(digest(token,'sha256'),'hex').
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// GenerateSecureToken generates a cryptographically random base64url string (32 bytes).
func GenerateSecureToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// NewTokenPair creates an access JWT + refresh token string from claims and config.
// Mutates claims.JTI, claims.IAT, claims.EXP in-place so the caller has the final values.
func NewTokenPair(claims *Claims, cfg AuthConfig) (*TokenPair, error) {
	if cfg.JWTSecret == "" {
		return nil, ErrEmptyJWTSecret
	}
	if claims.Sub == uuid.Nil {
		return nil, ErrEmptySubject
	}

	now := time.Now()
	accessExp := now.Add(cfg.AccessTokenTTL)
	refreshExp := now.Add(cfg.RefreshTokenTTL)

	// Generate JTI if not set
	if claims.JTI == uuid.Nil {
		claims.JTI = uuid.New()
	}

	claims.IAT = now.Unix()
	claims.EXP = accessExp.Unix()

	accessToken, err := signJWT(*claims, cfg.JWTSecret)
	if err != nil {
		return nil, fmt.Errorf("sign access token: %w", err)
	}

	refreshToken, err := GenerateSecureToken()
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:      accessToken,
		RefreshToken:     refreshToken,
		AccessExpiresAt:  accessExp,
		RefreshExpiresAt: refreshExp,
		TokenType:        "Bearer",
	}, nil
}

// NewAccessToken creates a new access JWT only (for refresh flow).
func NewAccessToken(claims Claims, cfg AuthConfig) (string, time.Time, error) {
	if cfg.JWTSecret == "" {
		return "", time.Time{}, ErrEmptyJWTSecret
	}

	now := time.Now()
	exp := now.Add(cfg.AccessTokenTTL)

	// New JTI for the refreshed token
	claims.JTI = uuid.New()
	claims.IAT = now.Unix()
	claims.EXP = exp.Unix()

	token, err := signJWT(claims, cfg.JWTSecret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign access token: %w", err)
	}
	return token, exp, nil
}

// jwtClaims is the typed JWT claims struct used with ParseWithClaims.
// Embeds jwt.RegisteredClaims for automatic exp/iat/nbf/iss validation.
type jwtClaims struct {
	Email string `json:"email"`
	Role  string `json:"role"`
	jwt.RegisteredClaims
}

// ParseToken parses and validates a JWT token string.
// Security: rejects alg:none, requires exp claim, validates issuer, allows 5s clock skew.
func ParseToken(tokenString string, secret string) (*Claims, error) {
	parsed, err := jwt.ParseWithClaims(tokenString, &jwtClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("invalid signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	},
		jwt.WithValidMethods([]string{"HS256"}),
		jwt.WithExpirationRequired(),
		jwt.WithLeeway(5*time.Second),
		jwt.WithIssuer(TokenIssuer),
	)

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, fmt.Errorf("parse token: %w", err)
	}

	jc, ok := parsed.Claims.(*jwtClaims)
	if !ok || !parsed.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	claims := &Claims{
		Email: jc.Email,
		Role:  jc.Role,
	}
	// L1: Subject (user ID) MUST parse to a valid UUID. A token with a
	// malformed subject is not safely usable downstream (could map to uuid.Nil
	// and cause unexpected DB lookups). Fail closed.
	if jc.Subject == "" {
		return nil, ErrEmptySubject
	}
	sub, subErr := uuid.Parse(jc.Subject)
	if subErr != nil {
		return nil, fmt.Errorf("invalid subject: %w", subErr)
	}
	claims.Sub = sub
	if jc.ID != "" {
		jti, jtiErr := uuid.Parse(jc.ID)
		if jtiErr != nil {
			return nil, fmt.Errorf("invalid jti: %w", jtiErr)
		}
		claims.JTI = jti
	}
	if jc.IssuedAt != nil {
		claims.IAT = jc.IssuedAt.Unix()
	}
	if jc.ExpiresAt != nil {
		claims.EXP = jc.ExpiresAt.Unix()
	}

	return claims, nil
}

func signJWT(claims Claims, secret string) (string, error) {
	jc := jwtClaims{
		Email: claims.Email,
		Role:  claims.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   claims.Sub.String(),
			ID:        claims.JTI.String(),
			Issuer:    TokenIssuer,
			IssuedAt:  jwt.NewNumericDate(time.Unix(claims.IAT, 0)),
			ExpiresAt: jwt.NewNumericDate(time.Unix(claims.EXP, 0)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jc)
	return token.SignedString([]byte(secret))
}
