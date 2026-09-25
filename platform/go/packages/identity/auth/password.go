package auth

import "golang.org/x/crypto/bcrypt"

// MaxPasswordLength is the bcrypt limit (72 bytes).
const MaxPasswordLength = 72

// HashPassword hashes a plaintext password using bcrypt with the given cost factor.
func HashPassword(password string, cost int) (string, error) {
	if password == "" {
		return "", ErrEmptyPassword
	}
	if len(password) > MaxPasswordLength {
		return "", ErrPasswordTooLong
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// VerifyPassword compares a plaintext password against a bcrypt hash.
// Returns nil on match, non-nil on mismatch. Constant-time comparison.
func VerifyPassword(hash string, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}
