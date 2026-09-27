package money

import (
	"errors"
	"math"
	"math/big"
	"testing"
)

// TestConvert covers SC2: a THB Money converted via an injected RateProvider.
// A valid rate yields a correctly half-even rounded result; an invalid (<=0)
// rate returns ErrInvalidRate and never an overflowed/garbage amount; a missing
// pair returns ErrRateUnavailable; an unknown target code returns
// ErrUnknownCurrency; the identity conversion returns the input unchanged.
func TestConvert(t *testing.T) {
	tests := []struct {
		name     string
		amount   int64        // source minor units
		from     CurrencyCode // source currency
		to       CurrencyCode // target currency
		rate     *big.Rat     // units of `to` per ONE unit of `from`
		seedPair bool         // whether to seed (from,to) into the table at all
		want     int64        // expected target minor units (when wantErr == nil)
		wantCur  CurrencyCode // expected target currency
		wantFrac uint8        // expected target fraction
		wantErr  error
	}{
		// --- same-decimals (THB.2 -> USD.2) ---
		{
			name:     "THB->USD at 1/34 rounds half-even down",
			amount:   10000, // 100.00 THB
			from:     "THB",
			to:       "USD",
			rate:     big.NewRat(1, 34), // 10000/34 = 294.1176... -> 294
			seedPair: true,
			want:     294,
			wantCur:  "USD",
			wantFrac: 2,
		},
		{
			name:     "tie rounds to even (12.5 -> 12)",
			amount:   100, // 1.00 THB
			from:     "THB",
			to:       "USD",
			rate:     big.NewRat(1, 8), // 100/8 = 12.5 -> tie -> even -> 12
			seedPair: true,
			want:     12,
			wantCur:  "USD",
			wantFrac: 2,
		},
		{
			name:     "tie rounds to even (37.5 -> 38)",
			amount:   300, // 3.00 THB
			from:     "THB",
			to:       "USD",
			rate:     big.NewRat(1, 8), // 300/8 = 37.5 -> tie -> even -> 38
			seedPair: true,
			want:     38,
			wantCur:  "USD",
			wantFrac: 2,
		},
		// --- cross-decimals (THB.2 -> JPY.0), 10^(0-2) = 1/100 factor ---
		{
			name:     "THB->JPY cross-decimals non-tie",
			amount:   25000, // 250.00 THB
			from:     "THB",
			to:       "JPY",
			rate:     big.NewRat(4, 1), // 25000 * 4 / 100 = 1000 JPY
			seedPair: true,
			want:     1000,
			wantCur:  "JPY",
			wantFrac: 0,
		},
		{
			name:     "THB->JPY cross-decimals tie rounds to even",
			amount:   100, // 1.00 THB
			from:     "THB",
			to:       "JPY",
			rate:     big.NewRat(5, 2), // 100 * 2.5 / 100 = 2.5 -> tie -> even -> 2
			seedPair: true,
			want:     2,
			wantCur:  "JPY",
			wantFrac: 0,
		},
		// --- signed Money: a negative converts to a negative ---
		{
			name:     "negative Money converts to negative",
			amount:   -10000, // -100.00 THB
			from:     "THB",
			to:       "USD",
			rate:     big.NewRat(1, 34), // -10000/34 = -294.117... -> -294
			seedPair: true,
			want:     -294,
			wantCur:  "USD",
			wantFrac: 2,
		},
		// --- invalid rate guards ---
		{
			name:     "zero rate -> ErrInvalidRate",
			amount:   10000,
			from:     "THB",
			to:       "USD",
			rate:     big.NewRat(0, 1),
			seedPair: true,
			wantErr:  ErrInvalidRate,
		},
		{
			name:     "negative rate -> ErrInvalidRate",
			amount:   10000,
			from:     "THB",
			to:       "USD",
			rate:     big.NewRat(-1, 34),
			seedPair: true,
			wantErr:  ErrInvalidRate,
		},
		// --- missing pair ---
		{
			name:     "missing pair -> ErrRateUnavailable",
			amount:   10000,
			from:     "THB",
			to:       "USD",
			seedPair: false,
			wantErr:  ErrRateUnavailable,
		},
		// --- unknown target code (registry-gated) ---
		{
			name:     "unknown target code -> ErrUnknownCurrency",
			amount:   10000,
			from:     "THB",
			to:       "ZZZ",
			seedPair: false,
			wantErr:  ErrUnknownCurrency,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New(tt.amount, tt.from)
			rates := NewRateTable()
			if tt.seedPair && tt.rate != nil {
				rates.Set(tt.from, tt.to, tt.rate)
			}

			got, err := Convert(m, tt.to, rates)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Convert() error = %v, want %v", err, tt.wantErr)
				}
				// On error the result must be the zero Money, never garbage.
				if got != (Money{}) {
					t.Errorf("Convert() on error = %v, want zero Money", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("Convert() unexpected error: %v", err)
			}
			if got.Amount != tt.want {
				t.Errorf("Convert() Amount = %d, want %d", got.Amount, tt.want)
			}
			if got.Currency != tt.wantCur {
				t.Errorf("Convert() Currency = %q, want %q", got.Currency, tt.wantCur)
			}
			if got.Fraction != tt.wantFrac {
				t.Errorf("Convert() Fraction = %d, want %d", got.Fraction, tt.wantFrac)
			}
		})
	}
}

// TestConvertIdentity verifies Convert(m, m.Currency) returns the input unchanged
// without consulting the rate provider (no THB->THB rate needs to be seeded).
func TestConvertIdentity(t *testing.T) {
	m := New(12345, "THB")
	// An empty table would error on any lookup; identity must short-circuit.
	got, err := Convert(m, "thb", NewRateTable())
	if err != nil {
		t.Fatalf("Convert identity unexpected error: %v", err)
	}
	if got != m {
		t.Errorf("Convert identity = %v, want %v (unchanged)", got, m)
	}
}

// TestConvertOverflow verifies that a conversion whose rounded result exceeds the
// int64 range returns ErrOverflow rather than wrapping/narrowing blindly.
func TestConvertOverflow(t *testing.T) {
	m := New(math.MaxInt64, "THB")
	rates := NewRateTable().Set("THB", "USD", big.NewRat(1000000, 1))
	if _, err := Convert(m, "USD", rates); !errors.Is(err, ErrOverflow) {
		t.Errorf("Convert overflow expected ErrOverflow, got %v", err)
	}
}

// TestRateTableLookup verifies the in-memory reference adapter: a seeded pair is
// returned exactly, a missing pair yields ErrRateUnavailable, and Set is chainable.
func TestRateTableLookup(t *testing.T) {
	r := big.NewRat(1, 34)
	table := NewRateTable().
		Set("THB", "USD", r).
		Set("THB", "JPY", big.NewRat(45, 10))

	got, err := table.Rate("thb", "usd") // lookup normalizes codes
	if err != nil {
		t.Fatalf("Rate(THB,USD) unexpected error: %v", err)
	}
	if got.Cmp(r) != 0 {
		t.Errorf("Rate(THB,USD) = %s, want %s", got.RatString(), r.RatString())
	}

	if _, err := table.Rate("USD", "THB"); !errors.Is(err, ErrRateUnavailable) {
		t.Errorf("Rate(USD,THB) on missing pair = %v, want ErrRateUnavailable", err)
	}
}

// TestRateTableReturnsCopy verifies the table hands back a defensive copy — a
// caller mutating the returned *big.Rat must not corrupt the stored rate.
func TestRateTableReturnsCopy(t *testing.T) {
	table := NewRateTable().Set("THB", "USD", big.NewRat(1, 34))
	got, err := table.Rate("THB", "USD")
	if err != nil {
		t.Fatalf("Rate unexpected error: %v", err)
	}
	got.Add(got, big.NewRat(100, 1)) // mutate the returned value

	again, err := table.Rate("THB", "USD")
	if err != nil {
		t.Fatalf("Rate unexpected error: %v", err)
	}
	if again.Cmp(big.NewRat(1, 34)) != 0 {
		t.Errorf("stored rate corrupted by caller mutation: got %s", again.RatString())
	}
}
