package user

import "net/mail"

const maxEmailLength = 255

// ValidateEmail validates email format per RFC 5322.
func ValidateEmail(email string) error {
	if email == "" {
		return ErrInvalidEmail
	}
	if len(email) > maxEmailLength {
		return ErrEmailTooLong
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return ErrInvalidEmail
	}
	return nil
}
