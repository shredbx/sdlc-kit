package auth

import (
	"strings"
)

// commonWeakPasswords is a small block-list of the most-used weak passwords.
// L3: minimal NIST-flavored breach-list check (full HIBP integration is a
// separate concern outside auth package scope).
var commonWeakPasswords = map[string]struct{}{
	"password": {}, "password1": {}, "password123": {}, "passw0rd": {},
	"12345678": {}, "123456789": {}, "1234567890": {},
	"qwerty": {}, "qwerty123": {}, "qwertyuiop": {},
	"abc12345": {}, "abcd1234": {},
	"iloveyou": {}, "admin": {}, "admin123": {}, "letmein": {},
	"welcome": {}, "welcome123": {}, "monkey123": {},
	"changeme": {}, "test1234": {}, "test-password-123": {},
}

// MinJWTSecretBytes is the minimum HS256 secret length. Shorter secrets are
// offline-bruteable, so config construction rejects them (F2, scope-10).
const MinJWTSecretBytes = 32

// Validate checks security-critical AuthConfig invariants at construction time.
// F2: a present-but-short JWT secret is worse than a clear failure — reject it
// before the service starts signing tokens with it.
func (c AuthConfig) Validate() error {
	if c.JWTSecret == "" {
		return ErrEmptyJWTSecret
	}
	if len(c.JWTSecret) < MinJWTSecretBytes {
		return ErrJWTSecretTooShort
	}
	return nil
}

// ValidateEmail checks if the email has a basic valid format.
func ValidateEmail(email string) error {
	if email == "" {
		return ErrInvalidEmail
	}
	at := strings.Index(email, "@")
	if at < 1 {
		return ErrInvalidEmail
	}
	domain := email[at+1:]
	if domain == "" || !strings.Contains(domain, ".") {
		return ErrInvalidEmail
	}
	return nil
}

// ValidatePassword checks password meets minimum strength requirements.
// Rules: non-empty, at least 8 characters, at most 72 bytes (bcrypt limit),
// not in the common-weak-passwords block-list (L3).
func ValidatePassword(password string) error {
	if password == "" {
		return ErrEmptyPassword
	}
	if len(password) < 8 {
		return ErrWeakPassword
	}
	if len(password) > MaxPasswordLength {
		return ErrPasswordTooLong
	}
	if _, weak := commonWeakPasswords[strings.ToLower(password)]; weak {
		return ErrWeakPassword
	}
	return nil
}

// ValidateRole checks if the role is one of the allowed values. M6: the role
// allowlist is REQUIRED — there is no implicit default. Callers must pass the
// project-specific role list to make the security boundary explicit.
func ValidateRole(role string, allowed []string) error {
	if role == "" {
		return ErrInvalidRole
	}
	if len(allowed) == 0 {
		return ErrInvalidRole
	}
	for _, valid := range allowed {
		if role == valid {
			return nil
		}
	}
	return ErrInvalidRole
}
