package money

import (
	"errors"
	"math"
	"testing"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name     string
		amount   int64
		currency CurrencyCode
		expected Money
	}{
		{"100 THB in satang", 10000, "THB", Money{Amount: 10000, Currency: "THB", Fraction: 2}},
		{"0 USD", 0, "USD", Money{Amount: 0, Currency: "USD", Fraction: 2}},
		{"empty currency defaults to THB", 10000, "", Money{Amount: 10000, Currency: "THB", Fraction: 2}},
		{"lowercase currency", 10000, "thb", Money{Amount: 10000, Currency: "THB", Fraction: 2}},
		{"negative amount allowed", -2500, "THB", Money{Amount: -2500, Currency: "THB", Fraction: 2}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New(tt.amount, tt.currency)
			if m.Amount != tt.expected.Amount || m.Currency != tt.expected.Currency || m.Fraction != tt.expected.Fraction {
				t.Errorf("New(%d, %q) = %v, want %v", tt.amount, tt.currency, m, tt.expected)
			}
		})
	}
}

func TestFromFloat(t *testing.T) {
	tests := []struct {
		name           string
		amount         float64
		currency       CurrencyCode
		expectedAmount int64
	}{
		{"100.00 THB", 100.00, "THB", 10000},
		{"100.50 THB", 100.50, "THB", 10050},
		{"1250.99 THB", 1250.99, "THB", 125099},
		{"0.01 USD", 0.01, "USD", 1},
		{"1000 JPY (no decimals)", 1000, "JPY", 1000},
		{"negative amount allowed", -100.00, "THB", -10000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := FromFloat(tt.amount, tt.currency)
			if err != nil {
				t.Fatalf("FromFloat(%v, %q) unexpected error: %v", tt.amount, tt.currency, err)
			}
			if m.Amount != tt.expectedAmount {
				t.Errorf("FromFloat(%v, %q).Amount = %d, want %d", tt.amount, tt.currency, m.Amount, tt.expectedAmount)
			}
		})
	}
}

// TestFromFloatNonFinite verifies NaN/Inf are rejected before int64 narrowing.
func TestFromFloatNonFinite(t *testing.T) {
	for _, in := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		if _, err := FromFloat(in, "THB"); !errors.Is(err, ErrInvalidInput) {
			t.Errorf("FromFloat(%v, THB) error = %v, want ErrInvalidInput", in, err)
		}
	}
}

func TestFromFloatFraction(t *testing.T) {
	m, err := FromFloat(12900000.00, "THB")
	if err != nil {
		t.Fatalf("FromFloat THB unexpected error: %v", err)
	}
	if m.Fraction != 2 {
		t.Errorf("FromFloat THB fraction = %d, want 2", m.Fraction)
	}
	if m.Amount != 1290000000 {
		t.Errorf("FromFloat ฿12,900,000.00 amount = %d, want 1290000000", m.Amount)
	}

	jpy, err := FromFloat(1000, "JPY")
	if err != nil {
		t.Fatalf("FromFloat JPY unexpected error: %v", err)
	}
	if jpy.Fraction != 0 {
		t.Errorf("FromFloat JPY fraction = %d, want 0", jpy.Fraction)
	}
}

func TestToFloat(t *testing.T) {
	tests := []struct {
		name     string
		money    Money
		expected float64
	}{
		{"10000 satang = 100 THB", Money{Amount: 10000, Currency: "THB", Fraction: 2}, 100.00},
		{"10050 satang = 100.50 THB", Money{Amount: 10050, Currency: "THB", Fraction: 2}, 100.50},
		{"1 cent = 0.01 USD", Money{Amount: 1, Currency: "USD", Fraction: 2}, 0.01},
		{"1000 JPY = 1000 JPY", Money{Amount: 1000, Currency: "JPY", Fraction: 0}, 1000.00},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.money.ToFloat()
			if result != tt.expected {
				t.Errorf("%v.ToFloat() = %v, want %v", tt.money, result, tt.expected)
			}
		})
	}
}

func TestDisplay(t *testing.T) {
	tests := []struct {
		name     string
		money    Money
		expected string
	}{
		{"100 THB", Money{Amount: 10000, Currency: "THB", Fraction: 2}, "฿100.00"},
		{"1,250.50 THB", Money{Amount: 125050, Currency: "THB", Fraction: 2}, "฿1,250.50"},
		{"100 USD", Money{Amount: 10000, Currency: "USD", Fraction: 2}, "$100.00"},
		{"1000 JPY", Money{Amount: 1000, Currency: "JPY", Fraction: 0}, "¥1,000"},
		{"100 EUR (suffix)", Money{Amount: 10000, Currency: "EUR", Fraction: 2}, "100.00€"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.money.Display()
			if result != tt.expected {
				t.Errorf("%v.Display() = %q, want %q", tt.money, result, tt.expected)
			}
		})
	}
}

func TestDisplayWithDecimals(t *testing.T) {
	tests := []struct {
		name     string
		money    Money
		decimals uint8
		expected string
	}{
		// decimals=0 → whole-baht (the BR manage-list/picker look), thousands-separated, no cents.
		{"12M THB whole", Money{Amount: 1200000000, Currency: "THB", Fraction: 2}, 0, "฿12,000,000"},
		{"1,250.50 THB whole", Money{Amount: 125050, Currency: "THB", Fraction: 2}, 0, "฿1,250"},
		{"100 USD whole (prefix)", Money{Amount: 10000, Currency: "USD", Fraction: 2}, 0, "$100"},
		{"100 EUR whole (suffix)", Money{Amount: 10000, Currency: "EUR", Fraction: 2}, 0, "100€"},
		// decimals override still works upward.
		{"explicit 2 decimals", Money{Amount: 1200000000, Currency: "THB", Fraction: 2}, 2, "฿12,000,000.00"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.money.DisplayWithDecimals(tt.decimals)
			if result != tt.expected {
				t.Errorf("%v.DisplayWithDecimals(%d) = %q, want %q", tt.money, tt.decimals, result, tt.expected)
			}
		})
	}
}

func TestDisplayWithLocale(t *testing.T) {
	m := New(1290000000, "THB")
	result := m.DisplayWithLocale("th-TH")
	if result != "฿12,900,000.00" {
		t.Errorf("DisplayWithLocale = %q, want ฿12,900,000.00", result)
	}
}

func TestFromString(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		currency       CurrencyCode
		expectedAmount int64
		expectError    bool
	}{
		{"simple integer", "100", "THB", 10000, false},
		{"with decimals", "100.50", "THB", 10050, false},
		{"with thousands", "1,250.50", "THB", 125050, false},
		{"with symbol", "฿100.00", "THB", 10000, false},
		{"with spaces", " 100.00 ", "THB", 10000, false},
		{"empty string", "", "THB", 0, true},
		{"invalid", "abc", "THB", 0, true},
		{"negative", "-100", "THB", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := FromString(tt.input, tt.currency)
			if tt.expectError {
				if err == nil {
					t.Errorf("FromString(%q, %q) expected error, got nil", tt.input, tt.currency)
				}
			} else {
				if err != nil {
					t.Errorf("FromString(%q, %q) unexpected error: %v", tt.input, tt.currency, err)
				} else if m.Amount != tt.expectedAmount {
					t.Errorf("FromString(%q, %q).Amount = %d, want %d", tt.input, tt.currency, m.Amount, tt.expectedAmount)
				}
			}
		})
	}
}

func TestAdd(t *testing.T) {
	tests := []struct {
		name        string
		a           Money
		b           Money
		expected    Money
		expectError bool
	}{
		{"same currency", Money{Amount: 10000, Currency: "THB", Fraction: 2}, Money{Amount: 5000, Currency: "THB", Fraction: 2}, Money{Amount: 15000, Currency: "THB", Fraction: 2}, false},
		{"different currency", Money{Amount: 10000, Currency: "THB", Fraction: 2}, Money{Amount: 5000, Currency: "USD", Fraction: 2}, Money{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.a.Add(tt.b)
			if tt.expectError {
				if err == nil {
					t.Errorf("Add expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Add unexpected error: %v", err)
				} else if result.Amount != tt.expected.Amount {
					t.Errorf("Add result = %d, want %d", result.Amount, tt.expected.Amount)
				}
			}
		})
	}
}

func TestSubtract(t *testing.T) {
	tests := []struct {
		name        string
		a           Money
		b           Money
		expected    Money
		expectError bool
	}{
		{"valid subtraction", Money{Amount: 10000, Currency: "THB", Fraction: 2}, Money{Amount: 3000, Currency: "THB", Fraction: 2}, Money{Amount: 7000, Currency: "THB", Fraction: 2}, false},
		{"subtract to zero", Money{Amount: 10000, Currency: "THB", Fraction: 2}, Money{Amount: 10000, Currency: "THB", Fraction: 2}, Money{Amount: 0, Currency: "THB", Fraction: 2}, false},
		// Signed semantics: a negative difference is a valid Money, NOT an error.
		{"negative result allowed", Money{Amount: 3000, Currency: "THB", Fraction: 2}, Money{Amount: 10000, Currency: "THB", Fraction: 2}, Money{Amount: -7000, Currency: "THB", Fraction: 2}, false},
		{"different currency", Money{Amount: 10000, Currency: "THB", Fraction: 2}, Money{Amount: 5000, Currency: "USD", Fraction: 2}, Money{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.a.Subtract(tt.b)
			if tt.expectError {
				if err == nil {
					t.Errorf("Subtract expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Subtract unexpected error: %v", err)
				} else if result.Amount != tt.expected.Amount {
					t.Errorf("Subtract result = %d, want %d", result.Amount, tt.expected.Amount)
				}
			}
		})
	}
}

func TestMultiply(t *testing.T) {
	tests := []struct {
		name     string
		money    Money
		factor   float64
		expected int64
	}{
		{"multiply by 2", Money{Amount: 10000, Currency: "THB", Fraction: 2}, 2, 20000},
		{"multiply by 0.5", Money{Amount: 10000, Currency: "THB", Fraction: 2}, 0.5, 5000},
		{"multiply by 0", Money{Amount: 10000, Currency: "THB", Fraction: 2}, 0, 0},
		{"multiply by 1.5", Money{Amount: 10000, Currency: "THB", Fraction: 2}, 1.5, 15000},
		// Signed semantics: a negative factor now yields a negative amount (no clamp).
		{"multiply by -1 honored", Money{Amount: 10000, Currency: "THB", Fraction: 2}, -1, -10000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.money.Multiply(tt.factor)
			if result.Amount != tt.expected {
				t.Errorf("Multiply(%v) = %d, want %d", tt.factor, result.Amount, tt.expected)
			}
		})
	}
}

func TestCompare(t *testing.T) {
	tests := []struct {
		name        string
		a           Money
		b           Money
		expected    int
		expectError bool
	}{
		{"equal", Money{Amount: 10000, Currency: "THB", Fraction: 2}, Money{Amount: 10000, Currency: "THB", Fraction: 2}, 0, false},
		{"less than", Money{Amount: 5000, Currency: "THB", Fraction: 2}, Money{Amount: 10000, Currency: "THB", Fraction: 2}, -1, false},
		{"greater than", Money{Amount: 10000, Currency: "THB", Fraction: 2}, Money{Amount: 5000, Currency: "THB", Fraction: 2}, 1, false},
		{"different currency", Money{Amount: 10000, Currency: "THB", Fraction: 2}, Money{Amount: 10000, Currency: "USD", Fraction: 2}, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.a.Compare(tt.b)
			if tt.expectError {
				if err == nil {
					t.Errorf("Compare expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Compare unexpected error: %v", err)
				} else if result != tt.expected {
					t.Errorf("Compare = %d, want %d", result, tt.expected)
				}
			}
		})
	}
}

func TestIsZero(t *testing.T) {
	if !Zero("THB").IsZero() {
		t.Error("Zero() should return money with IsZero() == true")
	}
	m := New(10000, "THB")
	if m.IsZero() {
		t.Error("Non-zero money should have IsZero() == false")
	}
}

// TestFromMinor covers the canonical boundary constructor (SC3): negative minor
// units are allowed, an unknown non-empty code errors (no silent fallback), and
// an empty code defaults to THB.
func TestFromMinor(t *testing.T) {
	t.Run("negative minor allowed (no wrap)", func(t *testing.T) {
		m, err := FromMinor(-2500, "THB")
		if err != nil {
			t.Fatalf("FromMinor(-2500, THB) unexpected error: %v", err)
		}
		if m.Amount != -2500 {
			t.Errorf("FromMinor(-2500, THB).Amount = %d, want -2500", m.Amount)
		}
		if !m.IsNegative() {
			t.Errorf("FromMinor(-2500, THB).IsNegative() = false, want true")
		}
		if m.Currency != "THB" || m.Fraction != 2 {
			t.Errorf("FromMinor(-2500, THB) = %+v, want THB/fraction 2", m)
		}
	})

	t.Run("unknown code errors (no silent fallback)", func(t *testing.T) {
		if _, err := FromMinor(1000, "TBH"); !errors.Is(err, ErrUnknownCurrency) {
			t.Errorf("FromMinor(1000, TBH) error = %v, want ErrUnknownCurrency", err)
		}
	})

	t.Run("empty code defaults to THB", func(t *testing.T) {
		m, err := FromMinor(1000, "")
		if err != nil {
			t.Fatalf("FromMinor(1000, \"\") unexpected error: %v", err)
		}
		if m.Currency != "THB" {
			t.Errorf("FromMinor(1000, \"\").Currency = %q, want THB", m.Currency)
		}
	})

	t.Run("valid code normalized", func(t *testing.T) {
		m, err := FromMinor(500, "usd")
		if err != nil {
			t.Fatalf("FromMinor(500, usd) unexpected error: %v", err)
		}
		if m.Currency != "USD" {
			t.Errorf("FromMinor(500, usd).Currency = %q, want USD", m.Currency)
		}
	})
}

// TestToMinor verifies the boundary accessor returns the raw signed amount.
func TestToMinor(t *testing.T) {
	m := New(-7000, "THB")
	if m.ToMinor() != -7000 {
		t.Errorf("ToMinor() = %d, want -7000", m.ToMinor())
	}
}

// TestIsNegative covers the signed-amount sign predicates.
func TestIsNegative(t *testing.T) {
	if !New(-1, "THB").IsNegative() {
		t.Error("New(-1).IsNegative() = false, want true")
	}
	if New(0, "THB").IsNegative() {
		t.Error("New(0).IsNegative() = true, want false")
	}
	if New(1, "THB").IsNegative() {
		t.Error("New(1).IsNegative() = true, want false")
	}
	if !New(1, "THB").IsPositive() {
		t.Error("New(1).IsPositive() = false, want true")
	}
	if New(-1, "THB").IsPositive() {
		t.Error("New(-1).IsPositive() = true, want false")
	}
}

// TestAddOverflow verifies Add reports an error rather than silently wrapping.
func TestAddOverflow(t *testing.T) {
	a := New(math.MaxInt64, "THB")
	b := New(1, "THB")
	if _, err := a.Add(b); err == nil {
		t.Error("Add(MaxInt64, 1) expected overflow error, got nil")
	}
}

func TestDatabaseRoundtrip(t *testing.T) {
	original, err := FromFloat(1250.50, "THB")
	if err != nil {
		t.Fatalf("FromFloat unexpected error: %v", err)
	}

	// To database
	amount, currency := original.ToDatabase()

	// From database
	restored := FromDatabase(amount, currency)

	if !original.Equals(restored) {
		t.Errorf("Database roundtrip failed: %v != %v", original, restored)
	}
}

func TestCurrencyRegistry(t *testing.T) {
	// Test built-in currency
	cur, ok := GetCurrency("THB")
	if !ok {
		t.Error("THB should be registered")
	}
	if cur.Symbol != "฿" {
		t.Errorf("THB symbol = %q, want ฿", cur.Symbol)
	}

	// Test custom currency registration
	err := RegisterCurrency(Currency{
		Code:           "BTC",
		Symbol:         "₿",
		Name:           "Bitcoin",
		Decimals:       8,
		SymbolPosition: "prefix",
	})
	if err != nil {
		t.Errorf("RegisterCurrency failed: %v", err)
	}

	cur, ok = GetCurrency("BTC")
	if !ok {
		t.Error("BTC should be registered after RegisterCurrency")
	}
	if cur.Decimals != 8 {
		t.Errorf("BTC decimals = %d, want 8", cur.Decimals)
	}
}

// TestBRPriceFormat validates the exact price display from the BR website screenshot.
func TestBRPriceFormat(t *testing.T) {
	tests := []struct {
		name     string
		price    float64
		expected string
	}{
		{"Akiraya Luxury Pool Villas", 12900000.00, "฿12,900,000.00"},
		{"Commercial Building", 9750000.00, "฿9,750,000.00"},
		{"Sea View Mountain Villa", 21000000.00, "฿21,000,000.00"},
		{"Wellness Business", 7000000.00, "฿7,000,000.00"},
		{"Seaview Land Chanot", 14000000.00, "฿14,000,000.00"},
		{"Slope Land Paradise", 4000000.00, "฿4,000,000.00"},
		{"Land Lease Hin Kong", 20000.00, "฿20,000.00"},
		{"Business Partner", 550000.00, "฿550,000.00"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := FromFloat(tt.price, "THB")
			if err != nil {
				t.Fatalf("FromFloat(%v, THB) unexpected error: %v", tt.price, err)
			}
			result := m.Display()
			if result != tt.expected {
				t.Errorf("FromFloat(%v, THB).Display() = %q, want %q", tt.price, result, tt.expected)
			}
		})
	}
}
