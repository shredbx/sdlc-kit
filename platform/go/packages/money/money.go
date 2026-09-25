// Package money provides type-safe monetary value handling with currency-aware
// storage, display, and operations.
//
// Key Design Decisions:
//   - Storage: All amounts stored as SIGNED int64 in smallest currency unit
//     (satang, cents). Money is a SIGNED value object — it MAY hold a negative
//     amount (fees, debts, adjustments, ledger deltas). Non-negativity is a
//     PRICE-level concern (e.g. Property/PropertyUnit gte=0 validators), never a
//     Money invariant (Decision #0292, Fowler signed-money semantics).
//   - Precision: No floating-point errors - all math is integer-based
//   - Currency: ISO 4217 codes with configurable decimal places
//   - Default: THB (Thai Baht) with 2 decimal places
//
// Example:
//
//	100.00 THB → stored as 10000 (satang)
//	m, _ := money.FromMinor(10000, "THB")
//	m.Display() → "฿100.00"
package money

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"
)

// Common errors
var (
	ErrCurrencyMismatch = errors.New("currency mismatch: cannot operate on different currencies")
	ErrUnknownCurrency  = errors.New("unknown currency code")
	ErrInvalidInput     = errors.New("invalid input format")
	ErrOverflow         = errors.New("operation overflows int64 amount")
)

// Money represents a monetary amount with currency.
// Amount is a SIGNED int64 in the smallest currency unit (satang for THB,
// cents for USD) and MAY be negative. Fraction indicates the number of decimal
// places for this currency (e.g., 2 for THB/USD, 0 for JPY).
type Money struct {
	Amount   int64        `json:"amount"`
	Currency CurrencyCode `json:"currency"`
	Fraction uint8        `json:"fraction"`
}

// Currency defines configuration for a currency.
type Currency struct {
	Code           string `json:"code"`
	Symbol         string `json:"symbol"`
	Name           string `json:"name"`
	Decimals       uint8  `json:"decimals"`
	SymbolPosition string `json:"symbol_position"` // "prefix" or "suffix"
}

// currency registry
var (
	currencies   = make(map[string]Currency)
	currenciesMu sync.RWMutex
)

func init() {
	// Register the governed currency table. generatedCurrencies is projected from
	// .sbx/workspace/packages/core/go/types/money/currencies.yml via
	// `sbx generate currencies` — see currencies.generated.go (DO NOT EDIT).
	for _, c := range generatedCurrencies {
		currencies[c.Code] = c
	}
}

// =============================================================================
// CONSTRUCTORS
// =============================================================================

// New creates a Money value from a SIGNED integer amount in smallest currency
// unit. The amount MAY be negative (Money is a signed value object).
// Example: New(10000, "THB") represents 100.00 THB; New(-2500, "THB") is -25.00 THB.
func New(amount int64, currency CurrencyCode) Money {
	currency = NormalizeCurrencyCode(string(currency))
	if currency == "" {
		currency = "THB"
	}
	fraction := uint8(2) // default
	if cur, ok := GetCurrency(string(currency)); ok {
		fraction = cur.Decimals
	}
	return Money{Amount: amount, Currency: currency, Fraction: fraction}
}

// FromMinor is the canonical boundary constructor: it builds a Money from a
// SIGNED amount in minor units (satang/cents) and an ISO-4217 code, validating
// the currency against the governed registry.
//
//   - empty code            -> defaults to "THB" (the package default currency).
//   - non-empty UNKNOWN code -> ErrUnknownCurrency (NO silent fallback).
//   - valid code            -> Money with the normalized code + registry fraction.
//
// A negative `minor` is ALLOWED and preserved without wrapping — non-negativity
// is a Price-level concern, not a Money invariant (Decision #0292).
func FromMinor(minor int64, currency CurrencyCode) (Money, error) {
	code := NormalizeCurrencyCode(string(currency))
	if code == "" {
		code = "THB"
	}
	cur, ok := GetCurrency(string(code))
	if !ok {
		return Money{}, fmt.Errorf("%w: %q", ErrUnknownCurrency, string(code))
	}
	return Money{Amount: minor, Currency: code, Fraction: cur.Decimals}, nil
}

// FromFloat creates a Money value from a float64. The float is converted to the
// smallest currency unit based on decimal places. A negative amount is allowed.
//
// Hardening: NaN/±Inf are rejected with ErrInvalidInput BEFORE int64 narrowing,
// and the rounded product is range-checked against [MinInt64, MaxInt64] (an
// out-of-range value errors with ErrOverflow rather than silently wrapping).
// Example: FromFloat(100.00, "THB") returns Money{Amount: 10000, Currency: "THB"}.
func FromFloat(amount float64, currency CurrencyCode) (Money, error) {
	if !isFiniteFloat(amount) {
		return Money{}, fmt.Errorf("%w: amount is not finite (NaN/Inf)", ErrInvalidInput)
	}

	currency = NormalizeCurrencyCode(string(currency))
	if currency == "" {
		currency = "THB"
	}

	decimals := uint8(2) // default
	if cur, ok := GetCurrency(string(currency)); ok {
		decimals = cur.Decimals
	}

	multiplier := math.Pow(10, float64(decimals))
	scaled := math.Round(amount * multiplier)
	if !isFiniteFloat(scaled) || scaled < math.MinInt64 || scaled >= float64(math.MaxInt64) {
		return Money{}, fmt.Errorf("%w: amount %v exceeds int64 minor-unit range", ErrOverflow, amount)
	}

	return Money{Amount: int64(scaled), Currency: currency, Fraction: decimals}, nil
}

// isFiniteFloat reports whether f is a real finite number (not NaN, not ±Inf).
func isFiniteFloat(f float64) bool {
	return !math.IsNaN(f) && !math.IsInf(f, 0)
}

// FromString parses a money value from a string.
// Supports formats: "100", "100.00", "1,000.00", "฿100.00"
func FromString(input string, currency CurrencyCode) (Money, error) {
	currency = NormalizeCurrencyCode(string(currency))
	if currency == "" {
		currency = "THB"
	}

	// Get currency config
	cur, ok := GetCurrency(string(currency))
	if !ok {
		return Money{}, ErrUnknownCurrency
	}

	// Clean input: remove currency symbols, spaces, and thousands separators
	cleaned := input
	cleaned = strings.ReplaceAll(cleaned, cur.Symbol, "")
	cleaned = strings.ReplaceAll(cleaned, ",", "")
	cleaned = strings.ReplaceAll(cleaned, " ", "")
	cleaned = strings.TrimSpace(cleaned)

	if cleaned == "" {
		return Money{}, ErrInvalidInput
	}

	// Parse as float
	f, err := strconv.ParseFloat(cleaned, 64)
	if err != nil {
		return Money{}, fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}

	// Hardening: reject NaN/±Inf (ParseFloat yields ±Inf for over-large inputs).
	if !isFiniteFloat(f) {
		return Money{}, fmt.Errorf("%w: amount is not finite (NaN/Inf)", ErrInvalidInput)
	}

	// FromString is PRICE input — a user-entered price is non-negative. (Money
	// itself is signed; this guard is a parse-layer policy, not a Money rule.)
	if f < 0 {
		return Money{}, fmt.Errorf("%w: negative amounts not allowed", ErrInvalidInput)
	}

	return FromFloat(f, currency)
}

// Zero returns a zero Money value for the given currency.
func Zero(currency CurrencyCode) Money {
	currency = NormalizeCurrencyCode(string(currency))
	if currency == "" {
		currency = "THB"
	}
	fraction := uint8(2)
	if cur, ok := GetCurrency(string(currency)); ok {
		fraction = cur.Decimals
	}
	return Money{Amount: 0, Currency: currency, Fraction: fraction}
}

// =============================================================================
// DISPLAY METHODS
// =============================================================================

// ToFloat converts the Money amount to a float64.
// Warning: Use only for display purposes, not for calculations.
func (m Money) ToFloat() float64 {
	decimals := uint8(2) // default
	if cur, ok := GetCurrency(string(m.Currency)); ok {
		decimals = cur.Decimals
	}

	divisor := math.Pow(10, float64(decimals))
	return float64(m.Amount) / divisor
}

// Display formats the money value for display with currency symbol.
// Example: Money{10000, "THB"}.Display() → "฿100.00"
func (m Money) Display() string {
	cur, ok := GetCurrency(string(m.Currency))
	if !ok {
		cur = Currency{Code: string(m.Currency), Symbol: string(m.Currency), Decimals: 2, SymbolPosition: "prefix"}
	}

	f := m.ToFloat()
	formatted := formatNumber(f, cur.Decimals)

	if cur.SymbolPosition == "suffix" {
		return formatted + cur.Symbol
	}
	return cur.Symbol + formatted
}

// DisplayCode formats the money value with currency code instead of symbol.
// Example: Money{10000, "THB"}.DisplayCode() → "100.00 THB"
func (m Money) DisplayCode() string {
	cur, ok := GetCurrency(string(m.Currency))
	if !ok {
		cur = Currency{Code: string(m.Currency), Decimals: 2}
	}

	f := m.ToFloat()
	formatted := formatNumber(f, cur.Decimals)

	return formatted + " " + string(m.Currency)
}

// DisplayWithDecimals formats the money value with the currency symbol but an
// EXPLICIT number of fractional digits, overriding the currency's default. The
// thousands-separated integer part is always shown; the decimal part is omitted
// when decimals == 0 (the BR whole-baht look: Money{1200000000, "THB"}.
// DisplayWithDecimals(0) → "฿12,000,000", matching the web formatPriceMinor
// {decimals:'off'} manage-list/picker convention). Additive — Display() is
// unchanged for all existing callers.
func (m Money) DisplayWithDecimals(decimals uint8) string {
	cur, ok := GetCurrency(string(m.Currency))
	if !ok {
		cur = Currency{Code: string(m.Currency), Symbol: string(m.Currency), Decimals: decimals, SymbolPosition: "prefix"}
	}

	formatted := formatNumber(m.ToFloat(), decimals)

	if cur.SymbolPosition == "suffix" {
		return formatted + cur.Symbol
	}
	return cur.Symbol + formatted
}

// String implements fmt.Stringer for debugging.
func (m Money) String() string {
	return fmt.Sprintf("Money{%d %s}", m.Amount, m.Currency)
}

// formatNumber formats a float with the specified decimal places and thousands separators.
func formatNumber(f float64, decimals uint8) string {
	// Format with decimals
	format := fmt.Sprintf("%%.%df", decimals)
	s := fmt.Sprintf(format, f)

	// Split integer and decimal parts
	parts := strings.Split(s, ".")
	intPart := parts[0]

	// Add thousands separators
	var result strings.Builder
	for i, c := range intPart {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			result.WriteRune(',')
		}
		result.WriteRune(c)
	}

	if len(parts) > 1 && decimals > 0 {
		result.WriteRune('.')
		result.WriteString(parts[1])
	}

	return result.String()
}

// =============================================================================
// OPERATIONS
// =============================================================================

// Add returns the sum of two Money values.
// Returns ErrCurrencyMismatch if currencies differ, or ErrOverflow if the signed
// int64 sum would overflow (checked, never silently wrapped).
func (m Money) Add(other Money) (Money, error) {
	if m.Currency != other.Currency {
		return Money{}, ErrCurrencyMismatch
	}
	sum, ok := addInt64(m.Amount, other.Amount)
	if !ok {
		return Money{}, fmt.Errorf("%w: %d + %d", ErrOverflow, m.Amount, other.Amount)
	}
	return Money{Amount: sum, Currency: m.Currency, Fraction: m.Fraction}, nil
}

// Subtract returns the SIGNED difference of two Money values. A negative result
// is a valid Money (Fowler signed-money semantics, Decision #0292) — callers
// needing non-negativity check IsNegative() or clamp at their own layer.
// The only error is ErrCurrencyMismatch (or ErrOverflow on int64 wrap).
func (m Money) Subtract(other Money) (Money, error) {
	if m.Currency != other.Currency {
		return Money{}, ErrCurrencyMismatch
	}
	// a - b = a + (-b); negate via subtraction to keep the same overflow check.
	diff, ok := subInt64(m.Amount, other.Amount)
	if !ok {
		return Money{}, fmt.Errorf("%w: %d - %d", ErrOverflow, m.Amount, other.Amount)
	}
	return Money{Amount: diff, Currency: m.Currency, Fraction: m.Fraction}, nil
}

// Multiply returns the Money value multiplied by a factor. A negative factor now
// yields a negative amount (Money is signed — no clamping). The rounded product
// is range-checked against int64; an out-of-range product SATURATES to
// MinInt64/MaxInt64 (documented behavior — Multiply has no error return, so it
// clamps to the representable bound rather than silently wrapping).
func (m Money) Multiply(factor float64) Money {
	if !isFiniteFloat(factor) {
		return Money{Amount: 0, Currency: m.Currency, Fraction: m.Fraction}
	}
	scaled := math.Round(float64(m.Amount) * factor)
	return Money{Amount: saturateToInt64(scaled), Currency: m.Currency, Fraction: m.Fraction}
}

// Divide returns the Money value divided by a divisor. A negative divisor now
// yields a negative amount (Money is signed). A zero/NaN/Inf divisor returns the
// original value unchanged (no-op, as before). The quotient saturates to int64
// bounds rather than wrapping.
func (m Money) Divide(divisor float64) Money {
	if !isFiniteFloat(divisor) || divisor == 0 {
		return m // Return original if invalid divisor
	}
	scaled := math.Round(float64(m.Amount) / divisor)
	return Money{Amount: saturateToInt64(scaled), Currency: m.Currency, Fraction: m.Fraction}
}

// Compare compares two Money values.
// Returns -1 if m < other, 0 if equal, 1 if m > other.
// Returns error if currencies don't match.
func (m Money) Compare(other Money) (int, error) {
	if m.Currency != other.Currency {
		return 0, ErrCurrencyMismatch
	}
	if m.Amount < other.Amount {
		return -1, nil
	}
	if m.Amount > other.Amount {
		return 1, nil
	}
	return 0, nil
}

// addInt64 returns a+b and whether it stayed within int64 (no overflow). The
// sign-aware check needs no wider integer type and is stdlib-pure.
func addInt64(a, b int64) (int64, bool) {
	sum := a + b
	// Overflow iff a and b share a sign and the result's sign differs.
	if (a > 0 && b > 0 && sum < 0) || (a < 0 && b < 0 && sum >= 0) {
		return 0, false
	}
	return sum, true
}

// subInt64 returns a-b and whether it stayed within int64 (no overflow).
func subInt64(a, b int64) (int64, bool) {
	diff := a - b
	// Overflow iff a and b have different signs and the result's sign differs from a.
	if (a >= 0 && b < 0 && diff < 0) || (a < 0 && b > 0 && diff >= 0) {
		return 0, false
	}
	return diff, true
}

// saturateToInt64 clamps a rounded float magnitude to the int64 range. Inputs are
// expected pre-rounded; NaN clamps to 0 (defensive). Used by Multiply/Divide which
// have no error channel and therefore saturate rather than wrap.
func saturateToInt64(f float64) int64 {
	switch {
	case math.IsNaN(f):
		return 0
	case f >= float64(math.MaxInt64):
		return math.MaxInt64
	case f <= float64(math.MinInt64):
		return math.MinInt64
	default:
		return int64(f)
	}
}

// Equals returns true if both Money values are equal.
func (m Money) Equals(other Money) bool {
	return m.Amount == other.Amount && m.Currency == other.Currency
}

// IsZero returns true if the amount is zero.
func (m Money) IsZero() bool {
	return m.Amount == 0
}

// IsPositive returns true if the amount is greater than zero.
func (m Money) IsPositive() bool {
	return m.Amount > 0
}

// IsNegative returns true if the amount is below zero. Money is a signed value
// object — a negative amount is valid (Decision #0292); consumers that require
// non-negativity (e.g. a price) gate on this at their own layer.
func (m Money) IsNegative() bool {
	return m.Amount < 0
}

// =============================================================================
// CURRENCY REGISTRY
// =============================================================================

// GetCurrency returns the currency configuration for the given code.
func GetCurrency(code string) (Currency, bool) {
	currenciesMu.RLock()
	defer currenciesMu.RUnlock()
	c, ok := currencies[strings.ToUpper(code)]
	return c, ok
}

// RegisterCurrency registers a custom currency.
func RegisterCurrency(c Currency) error {
	if c.Code == "" {
		return errors.New("currency code is required")
	}
	currenciesMu.Lock()
	defer currenciesMu.Unlock()
	currencies[strings.ToUpper(c.Code)] = c
	return nil
}

// ListCurrencies returns all registered currencies.
func ListCurrencies() []Currency {
	currenciesMu.RLock()
	defer currenciesMu.RUnlock()
	result := make([]Currency, 0, len(currencies))
	for _, c := range currencies {
		result = append(result, c)
	}
	return result
}

// =============================================================================
// DATABASE HELPERS
// =============================================================================

// ToMinor returns the raw SIGNED amount in minor units (satang/cents). It is the
// boundary accessor paired with FromMinor — the canonical way to read a Money's
// storable integer value.
func (m Money) ToMinor() int64 {
	return m.Amount
}

// ToDatabase returns the (signed) amount and currency for database storage.
// Use separate columns: price_amount BIGINT (signed), price_currency VARCHAR(3).
func (m Money) ToDatabase() (int64, CurrencyCode) {
	return m.Amount, m.Currency
}

// FromDatabase creates a Money value from database columns. The BIGINT amount is
// signed; a negative stored value is preserved.
func FromDatabase(amount int64, currency CurrencyCode) Money {
	return New(amount, currency)
}

// DisplayWithLocale formats the money value for a specific locale.
// Currently delegates to Display() — locale-specific formatting (CLDR)
// will be added when bojanz/currency integration is needed.
func (m Money) DisplayWithLocale(locale string) string {
	return m.Display()
}
