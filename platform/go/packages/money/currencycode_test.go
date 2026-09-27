package money

import (
	"encoding/json"
	"testing"
)

// TestNormalizeCurrencyCode covers SC1 — Trim+Upper normalization.
func TestNormalizeCurrencyCode(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want CurrencyCode
	}{
		{"lowercase usd", "usd", "USD"},
		{"padded thb", " thb ", "THB"},
		{"already upper", "EUR", "EUR"},
		{"empty stays empty", "", ""},
		{"unknown still normalized", "tbh", "TBH"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeCurrencyCode(tt.in); got != tt.want {
				t.Errorf("NormalizeCurrencyCode(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// TestCurrencyCodeValid covers SC1 — registry-backed validity (no hand-switch).
// Empty is NOT valid; normalized known codes are valid; typos are not.
func TestCurrencyCodeValid(t *testing.T) {
	tests := []struct {
		name string
		code CurrencyCode
		want bool
	}{
		{"TBH typo invalid", "TBH", false},
		{"usd normalizes + valid", "usd", true},
		{"padded thb normalizes + valid", " thb ", true},
		{"uppercase USD valid", "USD", true},
		{"empty invalid", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.code.Valid(); got != tt.want {
				t.Errorf("CurrencyCode(%q).Valid() = %v, want %v", tt.code, got, tt.want)
			}
		})
	}
}

// TestCurrencyCodeUnmarshalJSON covers SC1 — wire parse normalizes; unknown errors;
// empty yields empty (no silent THB fallback).
func TestCurrencyCodeUnmarshalJSON(t *testing.T) {
	t.Run("unknown code errors", func(t *testing.T) {
		var c CurrencyCode
		if err := json.Unmarshal([]byte(`"tbh"`), &c); err == nil {
			t.Errorf("UnmarshalJSON(%q) expected error, got nil (c=%q)", "tbh", c)
		}
	})
	t.Run("padded lowercase normalizes to USD", func(t *testing.T) {
		var c CurrencyCode
		if err := json.Unmarshal([]byte(`"  usd "`), &c); err != nil {
			t.Fatalf("UnmarshalJSON unexpected error: %v", err)
		}
		if c != "USD" {
			t.Errorf("UnmarshalJSON = %q, want USD", c)
		}
	})
	t.Run("empty string yields empty (no THB fallback)", func(t *testing.T) {
		var c CurrencyCode
		if err := json.Unmarshal([]byte(`""`), &c); err != nil {
			t.Fatalf("UnmarshalJSON unexpected error: %v", err)
		}
		if c != "" {
			t.Errorf("UnmarshalJSON empty = %q, want empty", c)
		}
	})
}

// TestCurrencyCodeMarshalJSON ensures the wire stays a bare normalized string.
func TestCurrencyCodeMarshalJSON(t *testing.T) {
	b, err := json.Marshal(CurrencyCode("usd"))
	if err != nil {
		t.Fatalf("MarshalJSON error: %v", err)
	}
	if string(b) != `"USD"` {
		t.Errorf("MarshalJSON = %s, want \"USD\"", b)
	}
}

// TestCurrencyCodeScan covers the DB read boundary — normalizes; NULL/empty -> empty.
func TestCurrencyCodeScan(t *testing.T) {
	t.Run("string normalizes", func(t *testing.T) {
		var c CurrencyCode
		if err := c.Scan("usd"); err != nil {
			t.Fatalf("Scan error: %v", err)
		}
		if c != "USD" {
			t.Errorf("Scan(usd) = %q, want USD", c)
		}
	})
	t.Run("nil yields empty", func(t *testing.T) {
		var c CurrencyCode
		if err := c.Scan(nil); err != nil {
			t.Fatalf("Scan(nil) error: %v", err)
		}
		if c != "" {
			t.Errorf("Scan(nil) = %q, want empty", c)
		}
	})
	t.Run("byte slice normalizes", func(t *testing.T) {
		var c CurrencyCode
		if err := c.Scan([]byte(" thb ")); err != nil {
			t.Fatalf("Scan error: %v", err)
		}
		if c != "THB" {
			t.Errorf("Scan(bytes) = %q, want THB", c)
		}
	})
}

// TestCurrencyCodeValue covers the DB write boundary — emits the normalized string.
func TestCurrencyCodeValue(t *testing.T) {
	v, err := CurrencyCode("usd").Value()
	if err != nil {
		t.Fatalf("Value error: %v", err)
	}
	s, ok := v.(string)
	if !ok || s != "USD" {
		t.Errorf("Value() = %v, want USD", v)
	}
}
