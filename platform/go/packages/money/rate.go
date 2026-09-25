package money

import (
	"errors"
	"fmt"
	"math"
	"math/big"
)

// Rate-conversion errors. ErrOverflow (declared in money.go) is reused for a
// converted amount that does not fit in int64.
var (
	// ErrRateUnavailable is returned by a RateProvider when the requested
	// currency pair is unknown (no live or seeded rate).
	ErrRateUnavailable = errors.New("exchange rate unavailable for currency pair")
	// ErrInvalidRate is returned by Convert when the provider yields a nil,
	// zero, or negative rate. A big.Rat can never be NaN/Inf, but a
	// non-positive rate is an attack vector and is rejected.
	ErrInvalidRate = errors.New("invalid exchange rate: must be positive")
)

// RateProvider yields the exchange rate between two currencies. Implementations
// (HTTP/DB/cache adapters) live OUTSIDE pkg/money — rate fetching is out of
// scope for this package (Decision #0292); only THB is a live base today.
type RateProvider interface {
	// Rate returns units of `to` per ONE unit of `from`, as an exact rational.
	// Returns ErrRateUnavailable when the pair is unknown.
	Rate(from, to CurrencyCode) (*big.Rat, error)
}

// Convert converts a Money value into another currency using an INJECTED rate
// provider (never a global) — the package never knows how rates are fetched.
//
// Semantics (Decision #0292):
//   - `to` is normalized; an unknown/invalid code -> ErrUnknownCurrency.
//   - Identity: when m.Currency (normalized) == to, m is returned unchanged
//     without consulting the provider.
//   - The provider's error is propagated (e.g. ErrRateUnavailable on a miss).
//   - The rate is guarded: nil or sign <= 0 -> ErrInvalidRate.
//   - Exact conversion is done with math/big, then rounded HALF-EVEN (banker's)
//     to int64 minor units:
//     to_minor = round_half_even( m.Amount * rate * 10^(to.decimals - from.decimals) )
//   - A result outside [MinInt64, MaxInt64] -> ErrOverflow (never wrapped).
//
// Money is signed: a negative amount converts to a negative amount.
func Convert(m Money, to CurrencyCode, rates RateProvider) (Money, error) {
	toCode := NormalizeCurrencyCode(string(to))
	if !toCode.Valid() {
		return Money{}, fmt.Errorf("%w: %q", ErrUnknownCurrency, string(toCode))
	}

	from := NormalizeCurrencyCode(string(m.Currency))

	// Identity conversion short-circuits — no rate needed.
	if from == toCode {
		return m, nil
	}

	// Target fraction comes from the governed registry. toCode.Valid() above
	// guarantees this lookup succeeds.
	toCur, ok := GetCurrency(string(toCode))
	if !ok {
		return Money{}, fmt.Errorf("%w: %q", ErrUnknownCurrency, string(toCode))
	}

	r, err := rates.Rate(from, toCode)
	if err != nil {
		return Money{}, err
	}
	if r == nil || r.Sign() <= 0 {
		return Money{}, fmt.Errorf("%w: %v", ErrInvalidRate, r)
	}

	// Source fraction: prefer the registry decimals; fall back to the carried
	// Fraction (FromMinor/New always populate it from the registry anyway).
	fromDecimals := m.Fraction
	if fromCur, ok := GetCurrency(string(from)); ok {
		fromDecimals = fromCur.Decimals
	}

	// product = m.Amount * rate * 10^(toDecimals - fromDecimals), as an exact
	// rational. Build the scale factor 10^exp as a *big.Rat so a negative
	// exponent (e.g. THB.2 -> JPY.0) becomes 1/100 with no precision loss.
	product := new(big.Rat).SetInt64(m.Amount)
	product.Mul(product, r)
	product.Mul(product, pow10Rat(int(toCur.Decimals)-int(fromDecimals)))

	rounded := roundHalfEven(product)

	// Range-check before narrowing to int64 — never wrap.
	if rounded.Cmp(minInt64Big) < 0 || rounded.Cmp(maxInt64Big) > 0 {
		return Money{}, fmt.Errorf("%w: converted amount out of int64 range", ErrOverflow)
	}

	return Money{Amount: rounded.Int64(), Currency: toCode, Fraction: toCur.Decimals}, nil
}

// pow10Rat returns 10^exp as an exact rational. exp may be negative (yielding
// 1/10^|exp|), which is the cross-decimals case (e.g. to.dec - from.dec = -2).
func pow10Rat(exp int) *big.Rat {
	ten := big.NewInt(10)
	pow := new(big.Int).Exp(ten, big.NewInt(int64(absInt(exp))), nil)
	if exp < 0 {
		// 1 / 10^|exp|
		return new(big.Rat).SetFrac(big.NewInt(1), pow)
	}
	return new(big.Rat).SetInt(pow)
}

// roundHalfEven rounds a rational to the nearest integer, ties-to-even (banker's
// rounding). It works on the exact numerator/denominator with big.Int only:
// quotient = num/den (truncated toward zero), remainder r = num - q*den; compare
// 2*|r| to den — below half rounds toward zero, above half rounds away, exactly
// half rounds to make the quotient even. Sign is handled explicitly so a
// negative value rounds symmetrically (e.g. -2.5 -> -2, -3.5 -> -4).
func roundHalfEven(x *big.Rat) *big.Int {
	num := new(big.Int).Set(x.Num())
	den := new(big.Int).Set(x.Denom()) // always positive for a normalized big.Rat

	// Work on the magnitude, then re-apply the sign.
	neg := num.Sign() < 0
	num.Abs(num)

	q := new(big.Int)
	rem := new(big.Int)
	q.QuoRem(num, den, rem) // q = num/den, rem = num - q*den, 0 <= rem < den

	// twiceRem = 2*rem; compare to den.
	twiceRem := new(big.Int).Lsh(rem, 1)
	switch twiceRem.Cmp(den) {
	case 1: // remainder > half -> round up
		q.Add(q, bigOne)
	case 0: // exactly half -> ties to even
		if q.Bit(0) == 1 { // q is odd -> step to the even neighbor
			q.Add(q, bigOne)
		}
	case -1: // remainder < half -> truncate (round down)
	}

	if neg {
		q.Neg(q)
	}
	return q
}

// absInt returns the absolute value of an int.
func absInt(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// Shared big constants for range checks and rounding (computed once).
var (
	bigOne      = big.NewInt(1)
	maxInt64Big = new(big.Int).SetInt64(math.MaxInt64)
	minInt64Big = new(big.Int).SetInt64(math.MinInt64)
)

// =============================================================================
// RateTable — static in-memory RateProvider (test + reference adapter)
// =============================================================================

// RateTable is a static in-memory RateProvider — a test/reference adapter. A
// LIVE fetching adapter (HTTP/DB/cache) is a separate downstream module and is
// OUT OF SCOPE here (Decision #0292); today only THB is a live base.
//
// Seed example (THB is the live base; the USD pair below is illustrative):
//
//	rates := money.NewRateTable().
//	    Set("THB", "USD", big.NewRat(1, 34)).   // ~1/34 USD per THB (live base)
//	    Set("THB", "JPY", big.NewRat(45, 10))   // ~4.5 JPY per THB (stub pair)
//	usd, err := money.Convert(thb, "USD", rates)
//
// EXTENSION POINT:
//
//	A live fetcher (HTTP/DB/cache adapter) implements money.RateProvider in its
//	OWN package and is injected wherever a RateTable is used here — Convert never
//	changes. Such an adapter MUST validate that the fetched rate is finite and
//	strictly positive BEFORE returning it (the same guard Convert applies via
//	ErrInvalidRate); a non-positive or unparsable upstream rate must surface as
//	an error, never a zero/garbage *big.Rat. Caching/TTL, source selection, and
//	pair triangulation all live in that adapter, not in pkg/money.
type RateTable struct {
	rates map[rateKey]*big.Rat
}

// rateKey identifies a directed currency pair (from -> to) with normalized codes.
type rateKey struct {
	from CurrencyCode
	to   CurrencyCode
}

// NewRateTable returns an empty in-memory rate provider.
func NewRateTable() *RateTable {
	return &RateTable{rates: make(map[rateKey]*big.Rat)}
}

// Set stores the rate (units of `to` per ONE unit of `from`) and returns the
// table for chaining. Codes are normalized; a nil rate is ignored. A defensive
// copy of the rate is stored so a later caller mutation cannot corrupt it.
func (t *RateTable) Set(from, to CurrencyCode, rate *big.Rat) *RateTable {
	if rate == nil {
		return t
	}
	key := rateKey{
		from: NormalizeCurrencyCode(string(from)),
		to:   NormalizeCurrencyCode(string(to)),
	}
	t.rates[key] = new(big.Rat).Set(rate)
	return t
}

// Rate returns units of `to` per ONE unit of `from`. Codes are normalized before
// lookup. A defensive copy is returned so the caller cannot mutate the stored
// rate. Returns ErrRateUnavailable when the pair was never seeded.
func (t *RateTable) Rate(from, to CurrencyCode) (*big.Rat, error) {
	key := rateKey{
		from: NormalizeCurrencyCode(string(from)),
		to:   NormalizeCurrencyCode(string(to)),
	}
	r, ok := t.rates[key]
	if !ok {
		return nil, fmt.Errorf("%w: %s->%s", ErrRateUnavailable, key.from, key.to)
	}
	return new(big.Rat).Set(r), nil
}
