// Package phonenumber provides a reusable embedded type for international phone
// numbers with country code separation and type classification.
//
// PhoneNumber is a value object with embedded reference semantics — consuming
// entities gain typed columns ({attr}_country_code, {attr}_number, {attr}_type)
// via 3-column expansion, following the same pattern as GeoCoordinate.
//
// Example:
//
//	ph := phonenumber.NewPhoneNumber("+66", "812345678")
//	if err := ph.Validate(); err != nil { ... }
//	fmt.Println(ph.Format()) // "+66 812345678"
package phonenumber

import (
	"fmt"
	"strings"
)

// PhoneNumber represents an international phone number with country code and
// optional type classification. It is an immutable value object — create new
// instances rather than mutating existing ones.
type PhoneNumber struct {
	CountryCode string `json:"country_code" yaml:"country_code"`
	Number      string `json:"number" yaml:"number"`
	PhoneType   string `json:"phone_type,omitempty" yaml:"phone_type,omitempty"`
}

// NewPhoneNumber creates a PhoneNumber with the given country code and number.
func NewPhoneNumber(countryCode, number string) PhoneNumber {
	return PhoneNumber{CountryCode: countryCode, Number: number}
}

// Validate checks that the phone number has a valid country code and non-empty number.
func (p PhoneNumber) Validate() error {
	if strings.TrimSpace(p.CountryCode) == "" {
		return fmt.Errorf("country_code must not be empty")
	}
	if !strings.HasPrefix(p.CountryCode, "+") {
		return fmt.Errorf("country_code must start with '+', got %q", p.CountryCode)
	}
	if strings.TrimSpace(p.Number) == "" {
		return fmt.Errorf("number must not be empty")
	}
	return nil
}

// IsZero returns true if all phone number fields are empty.
func (p PhoneNumber) IsZero() bool {
	return p.CountryCode == "" && p.Number == "" && p.PhoneType == ""
}

// Format returns the phone number in international format: "+66 812345678".
func (p PhoneNumber) Format() string {
	return p.CountryCode + " " + p.Number
}
