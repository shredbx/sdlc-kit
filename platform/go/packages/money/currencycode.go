package money

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
)

// CurrencyCode is the governed ISO-4217 currency identifier used across money,
// property, and transaction. It is the single typed projection of the currency
// dictionary (currencies.yml → currencies.generated.go): validity is decided by
// the runtime registry, never a hand-maintained switch.
//
// On the wire and in storage it stays a plain string (JSON string, VARCHAR
// column) — the type only adds normalization + validity at the boundaries.
type CurrencyCode string

// NormalizeCurrencyCode canonicalizes a raw code: trim surrounding space and
// upper-case. It does NOT validate — an unknown code normalizes too (so callers
// can report the offending value). Empty normalizes to empty.
func NormalizeCurrencyCode(s string) CurrencyCode {
	return CurrencyCode(strings.ToUpper(strings.TrimSpace(s)))
}

// Valid reports whether the code is a known currency. Empty is NOT valid. The
// code is normalized before lookup, so " usd " and "USD" both resolve. The
// runtime registry (GetCurrency) is the single source of truth — there is no
// duplicate switch to drift.
func (c CurrencyCode) Valid() bool {
	code := NormalizeCurrencyCode(string(c))
	if code == "" {
		return false
	}
	_, ok := GetCurrency(string(code))
	return ok
}

// String returns the bare code.
func (c CurrencyCode) String() string {
	return string(c)
}

// MarshalJSON emits the bare normalized string — the wire format is unchanged
// (a plain JSON string), keeping CurrencyCode a drop-in for the former string.
func (c CurrencyCode) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(NormalizeCurrencyCode(string(c))))
}

// UnmarshalJSON parses a JSON string, normalizes (trim + upper), and validates.
//   - empty string  -> empty CurrencyCode (the caller/constructor applies the
//     default; there is NO silent THB fallback here).
//   - non-empty known code -> the normalized code.
//   - non-empty unknown code -> an error (rejected, not silently defaulted).
func (c *CurrencyCode) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	code := NormalizeCurrencyCode(s)
	if code == "" {
		*c = ""
		return nil
	}
	if !code.Valid() {
		return fmt.Errorf("%w: %q", ErrUnknownCurrency, string(code))
	}
	*c = code
	return nil
}

// Scan implements database/sql.Scanner so CurrencyCode is a drop-in for the
// existing VARCHAR currency columns (pgx / database/sql). The stored value is
// normalized; NULL or empty -> empty CurrencyCode.
func (c *CurrencyCode) Scan(src any) error {
	switch v := src.(type) {
	case nil:
		*c = ""
	case string:
		*c = NormalizeCurrencyCode(v)
	case []byte:
		*c = NormalizeCurrencyCode(string(v))
	default:
		return fmt.Errorf("%w: cannot scan %T into CurrencyCode", ErrInvalidInput, src)
	}
	return nil
}

// Value implements database/sql/driver.Valuer — the column stores the bare
// normalized string (empty stays empty, the column-default/NULL contract is the
// caller's). Returning a string keeps the VARCHAR storage shape unchanged.
func (c CurrencyCode) Value() (driver.Value, error) {
	return string(NormalizeCurrencyCode(string(c))), nil
}
