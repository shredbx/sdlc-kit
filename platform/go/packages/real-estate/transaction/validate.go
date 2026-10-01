package transaction

import (
	"errors"
	"time"

	"github.com/shredbx/sbx-core/pkg/money"
)

// =============================================================================
// SENTINEL ERRORS — typed, named exactly per usage-spec (downstream depends on them)
// =============================================================================

var (
	// ErrTypeRequired indicates a deal has no type set (SC-019-02).
	ErrTypeRequired = errors.New("deal kind is required")

	// ErrTypeNotEnabled indicates the deal type is not in the consumer's
	// enabled set (usage-spec 2b).
	ErrTypeNotEnabled = errors.New("deal kind is not enabled for this consumer")

	// ErrCurrencyRequired indicates a money value was set without a currency
	// (SC-020-02).
	ErrCurrencyRequired = errors.New("money currency is required")

	// ErrAmountNotPositive indicates a required money amount was not greater
	// than zero (SC-020-03).
	ErrAmountNotPositive = errors.New("money amount must be greater than zero")

	// ErrInvalidTransition indicates a forbidden status change (e.g. reopening
	// a closed deal) (SC-022-02).
	ErrInvalidTransition = errors.New("invalid status transition")

	// ErrAlreadyClosed indicates an attempt to close a deal that is already
	// closed (SC-023-04).
	ErrAlreadyClosed = errors.New("deal is already closed")

	// ErrFacetNotInType indicates a close facet the deal type does not compose
	// (usage-spec 2c failure twin).
	ErrFacetNotInType = errors.New("facet is not part of this deal type")

	// ErrPartyRoleRequired indicates a party with no role (SC-021-02).
	ErrPartyRoleRequired = errors.New("party role is required")

	// ErrLeaseDatesInverted indicates a lease whose end date precedes its start
	// date (T8 cross-field invariant — checked only when both bounds are set).
	ErrLeaseDatesInverted = errors.New("lease end date must not be before the lease start date")
)

// =============================================================================
// VALIDATION — invariants from domain-model.yml
// =============================================================================

// Validate enforces the transaction invariants against a consumer's enabled set:
//
//   - type is set (ErrTypeRequired) and enabled for this consumer (ErrTypeNotEnabled);
//   - every required, facet-scoped money carries a currency (ErrCurrencyRequired)
//     and a positive amount (ErrAmountNotPositive). The primary money of each
//     composed facet is required: Sale -> SalePrice, Lease -> LeaseRent. Optional
//     monies (e.g. LeaseDeposit) are validated only when set.
//
// Returns the first failing invariant (callers chain if they need full reports).
func (t Transaction) Validate(cfg EnabledTypes) error {
	if t.Type == "" {
		return ErrTypeRequired
	}
	if !cfg.IsEnabled(t.Type) {
		return ErrTypeNotEnabled
	}

	for _, f := range FacetsFor(t.Type) {
		switch f {
		case FacetSale:
			// Sale price is the defining term of a sale facet — required.
			if err := validateMoney(t.SalePrice, true); err != nil {
				return err
			}
		case FacetLease:
			// Rent is the defining term of a lease facet — required.
			if err := validateMoney(t.LeaseRent, true); err != nil {
				return err
			}
			// Deposit is optional — validate only when set.
			if err := validateMoney(t.LeaseDeposit, false); err != nil {
				return err
			}
			// Dates are optional, but when both bounds are present the lease
			// window must not be inverted (T8 cross-field invariant).
			if err := validateLeaseDates(t.LeaseStart, t.LeaseEnd); err != nil {
				return err
			}
		}
	}
	return nil
}

// validateLeaseDates enforces the lease-window invariant: when BOTH bounds are
// set, the end must not precede the start (T8). Either bound alone — or neither —
// is allowed (dates are optional). Equal bounds are valid (a same-day window).
func validateLeaseDates(start, end *time.Time) error {
	if start == nil || end == nil {
		return nil
	}
	if end.Before(*start) {
		return ErrLeaseDatesInverted
	}
	return nil
}

// validateMoney enforces the money invariant for one field.
//   - required: the amount must be positive AND a currency present.
//   - optional: an unset money (zero amount, no currency) is skipped; once any
//     part is set it must still carry a currency and a positive amount.
//
// Currency is checked before amount so an explicit empty-currency money surfaces
// ErrCurrencyRequired even when an amount is present (usage-spec 2b twin).
//
// NOTE: money.New(amount, "") DEFAULTS an empty currency to "THB" (pkg/money), so
// this guard only fires when the money was built WITHOUT money.New (an explicit
// money.Money{Currency: ""}). The SI handler MUST validate the request's currency
// before constructing money via money.New — otherwise a currency-less deal silently
// persists as THB instead of being rejected.
func validateMoney(m money.Money, required bool) error {
	set := m.Amount > 0 || m.Currency != ""
	if !required && !set {
		return nil
	}
	if m.Currency == "" {
		return ErrCurrencyRequired
	}
	if m.Amount == 0 {
		return ErrAmountNotPositive
	}
	return nil
}

// Validate enforces the party invariants: a role is set (ErrPartyRoleRequired)
// and is one of the known roles (also ErrPartyRoleRequired for an unknown code).
func (p TransactionParty) Validate() error {
	if p.Role == "" || !ValidPartyRole(p.Role) {
		return ErrPartyRoleRequired
	}
	return nil
}
