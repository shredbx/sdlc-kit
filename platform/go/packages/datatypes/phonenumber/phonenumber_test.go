package phonenumber

import (
	"testing"
)

func TestNewPhoneNumber(t *testing.T) {
	t.Run("creates with country code and number", func(t *testing.T) {
		p := NewPhoneNumber("+66", "812345678")
		if p.CountryCode != "+66" || p.Number != "812345678" {
			t.Errorf("expected {+66, 812345678}, got {%s, %s}", p.CountryCode, p.Number)
		}
		if p.PhoneType != "" {
			t.Errorf("expected empty PhoneType, got %q", p.PhoneType)
		}
	})
}

func TestPhoneNumber_Validate(t *testing.T) {
	tests := []struct {
		name        string
		countryCode string
		number      string
		wantErr     bool
		errContains string
	}{
		{"valid Thai mobile", "+66", "812345678", false, ""},
		{"valid German landline", "+49", "3012345678", false, ""},
		{"valid US number", "+1", "2125551234", false, ""},
		{"empty country_code", "", "812345678", true, "country_code must not be empty"},
		{"whitespace country_code", "  ", "812345678", true, "country_code must not be empty"},
		{"missing plus prefix", "66", "812345678", true, "country_code must start with '+'"},
		{"empty number", "+66", "", true, "number must not be empty"},
		{"whitespace number", "+66", "  ", true, "number must not be empty"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := PhoneNumber{CountryCode: tt.countryCode, Number: tt.number}
			err := p.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil {
				if got := err.Error(); !contains(got, tt.errContains) {
					t.Errorf("Validate() error = %q, want containing %q", got, tt.errContains)
				}
			}
		})
	}
}

func TestPhoneNumber_IsZero(t *testing.T) {
	t.Run("zero value", func(t *testing.T) {
		p := PhoneNumber{}
		if !p.IsZero() {
			t.Error("expected IsZero() = true for zero value")
		}
	})

	t.Run("non-zero with all fields", func(t *testing.T) {
		p := PhoneNumber{CountryCode: "+66", Number: "812345678", PhoneType: "mobile"}
		if p.IsZero() {
			t.Error("expected IsZero() = false for non-zero phone")
		}
	})

	t.Run("non-zero with only type", func(t *testing.T) {
		p := PhoneNumber{PhoneType: "mobile"}
		if p.IsZero() {
			t.Error("expected IsZero() = false when PhoneType is set")
		}
	})
}

func TestPhoneNumber_Format(t *testing.T) {
	tests := []struct {
		name        string
		countryCode string
		number      string
		expected    string
	}{
		{"Thai mobile", "+66", "812345678", "+66 812345678"},
		{"German landline", "+49", "3012345678", "+49 3012345678"},
		{"US number", "+1", "2125551234", "+1 2125551234"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewPhoneNumber(tt.countryCode, tt.number)
			got := p.Format()
			if got != tt.expected {
				t.Errorf("Format() = %q, want %q", got, tt.expected)
			}
		})
	}
}

// contains checks if s contains substr.
func contains(s, substr string) bool {
	return len(substr) == 0 || len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
