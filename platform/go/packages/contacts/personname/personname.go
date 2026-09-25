package personname

import (
	"fmt"
	"strings"
)

// PersonName represents a structured person name with optional title and middle name.
// It is an immutable value object — create new instances rather than mutating.
type PersonName struct {
	Title      string `json:"title,omitempty" yaml:"title,omitempty"`
	GivenName  string `json:"given_name" yaml:"given_name"`
	MiddleName string `json:"middle_name,omitempty" yaml:"middle_name,omitempty"`
	Surname    string `json:"surname" yaml:"surname"`
}

// NewPersonName creates a PersonName with the required given name and surname.
func NewPersonName(given, surname string) PersonName {
	return PersonName{GivenName: given, Surname: surname}
}

// Validate checks that the name has non-empty given name and surname.
func (n PersonName) Validate() error {
	if strings.TrimSpace(n.GivenName) == "" {
		return fmt.Errorf("given_name must not be empty")
	}
	if strings.TrimSpace(n.Surname) == "" {
		return fmt.Errorf("surname must not be empty")
	}
	return nil
}

// IsZero returns true if all name fields are empty.
func (n PersonName) IsZero() bool {
	return n.Title == "" && n.GivenName == "" && n.MiddleName == "" && n.Surname == ""
}

// FullName returns the complete name as "GivenName MiddleName Surname".
// Middle name is omitted if empty.
func (n PersonName) FullName() string {
	parts := []string{n.GivenName}
	if n.MiddleName != "" {
		parts = append(parts, n.MiddleName)
	}
	parts = append(parts, n.Surname)
	return strings.Join(parts, " ")
}

// FormalName returns the formal name as "Title Surname".
// If title is empty, returns just the surname.
func (n PersonName) FormalName() string {
	if n.Title != "" {
		return n.Title + " " + n.Surname
	}
	return n.Surname
}
