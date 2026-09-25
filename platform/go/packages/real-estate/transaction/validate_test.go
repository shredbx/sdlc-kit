package transaction_test

import (
	"errors"
	"testing"
	"time"

	"github.com/shredbx/sbx-core/pkg/money"
	"github.com/shredbx/sbx-core/pkg/transaction"
)

// brConfig is the BR enablement used across validate tests (FX-CFG-BR):
// sale, lease are enabled.
func brConfig() transaction.EnabledTypes {
	return transaction.EnabledSet(
		transaction.TypeSale,
		transaction.TypeLease,
	)
}

// TestValidate_TypeRequired covers SC-019-02: a deal with no type is rejected.
func TestValidate_TypeRequired(t *testing.T) {
	cfg := brConfig()
	tx := transaction.Transaction{
		Type:       "", // unset
		Status:     transaction.StatusOpen,
		PropertyID: "prop-1",
	}
	err := tx.Validate(cfg)
	if !errors.Is(err, transaction.ErrTypeRequired) {
		t.Fatalf("Validate(no type) err = %v, want ErrTypeRequired", err)
	}
}

// TestValidate_TypeNotEnabled covers the enablement invariant (usage-spec 2b):
// a type outside the consumer's enabled set is rejected.
func TestValidate_TypeNotEnabled(t *testing.T) {
	// Enable only sale; a lease deal must then be rejected.
	cfg := transaction.EnabledSet(transaction.TypeSale)
	tx := transaction.Transaction{
		Type:       transaction.TypeLease,
		Status:     transaction.StatusOpen,
		PropertyID: "prop-1",
		LeaseRent:  money.New(2_500_000, "THB"),
	}
	err := tx.Validate(cfg)
	if !errors.Is(err, transaction.ErrTypeNotEnabled) {
		t.Fatalf("Validate(lease, only sale enabled) err = %v, want ErrTypeNotEnabled", err)
	}
}

// TestValidate_CurrencyRequired covers SC-020-02: money set with no currency is rejected.
func TestValidate_CurrencyRequired(t *testing.T) {
	cfg := brConfig()
	tx := transaction.Transaction{
		Type:       transaction.TypeSale,
		Status:     transaction.StatusOpen,
		PropertyID: "prop-1",
		// money.New(amount, "") defaults currency to THB, so to model a missing
		// currency we set the Money explicitly with an empty currency + positive amount.
		SalePrice: money.Money{Amount: 2_100_000_000, Currency: ""},
	}
	err := tx.Validate(cfg)
	if !errors.Is(err, transaction.ErrCurrencyRequired) {
		t.Fatalf("Validate(sale, no currency) err = %v, want ErrCurrencyRequired", err)
	}
}

// TestValidate_AmountPositive covers SC-020-03: zero rent where a Lease facet
// term is set is rejected.
func TestValidate_AmountPositive(t *testing.T) {
	cfg := brConfig()
	tx := transaction.Transaction{
		Type:       transaction.TypeLease,
		Status:     transaction.StatusOpen,
		PropertyID: "prop-1",
		LeaseRent:  money.New(0, "THB"), // zero rent — not positive
	}
	err := tx.Validate(cfg)
	if !errors.Is(err, transaction.ErrAmountNotPositive) {
		t.Fatalf("Validate(lease, zero rent) err = %v, want ErrAmountNotPositive", err)
	}
}

// TestValidate_PartyRoleRequired covers SC-021-02: a party with no role is rejected.
func TestValidate_PartyRoleRequired(t *testing.T) {
	p := transaction.TransactionParty{
		Role:      "", // unset
		ContactID: "contact-1",
	}
	err := p.Validate()
	if !errors.Is(err, transaction.ErrPartyRoleRequired) {
		t.Fatalf("Party.Validate(no role) err = %v, want ErrPartyRoleRequired", err)
	}
}

// TestValidate_PartyRole_Valid is the success twin for party validation.
func TestValidate_PartyRole_Valid(t *testing.T) {
	p := transaction.TransactionParty{
		Role:      transaction.RoleBuyer,
		ContactID: "contact-1",
	}
	if err := p.Validate(); err != nil {
		t.Fatalf("Party.Validate(buyer) err = %v, want nil", err)
	}
}

// TestValidate_LeaseDatesInverted covers T8: a lease whose end date precedes its
// start date is rejected (cross-field invariant; fires only when BOTH are set).
func TestValidate_LeaseDatesInverted(t *testing.T) {
	cfg := brConfig()
	start := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC) // before start
	tx := transaction.Transaction{
		Type:       transaction.TypeLease,
		Status:     transaction.StatusOpen,
		PropertyID: "prop-1",
		LeaseRent:  money.New(2_500_000, "THB"),
		LeaseStart: &start,
		LeaseEnd:   &end,
	}
	err := tx.Validate(cfg)
	if !errors.Is(err, transaction.ErrLeaseDatesInverted) {
		t.Fatalf("Validate(lease, end<start) err = %v, want ErrLeaseDatesInverted", err)
	}
}

// TestValidate_LeaseDates_Valid is the success twin: end on/after start validates.
// Equal dates are allowed (a same-day lease window is not inverted).
func TestValidate_LeaseDates_Valid(t *testing.T) {
	cfg := brConfig()
	start := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	for _, end := range []time.Time{start, start.AddDate(1, 0, 0)} {
		end := end
		tx := transaction.Transaction{
			Type:       transaction.TypeLease,
			Status:     transaction.StatusOpen,
			PropertyID: "prop-1",
			LeaseRent:  money.New(2_500_000, "THB"),
			LeaseStart: &start,
			LeaseEnd:   &end,
		}
		if err := tx.Validate(cfg); err != nil {
			t.Fatalf("Validate(lease, end>=start) err = %v, want nil", err)
		}
	}
}

// TestValidate_LeaseDates_PartialOK covers that a single date (only start, only
// end, or neither) is allowed — dates are optional and the cross-field check
// only fires when BOTH bounds are present.
func TestValidate_LeaseDates_PartialOK(t *testing.T) {
	cfg := brConfig()
	start := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	base := transaction.Transaction{
		Type:       transaction.TypeLease,
		Status:     transaction.StatusOpen,
		PropertyID: "prop-1",
		LeaseRent:  money.New(2_500_000, "THB"),
	}
	onlyStart := base
	onlyStart.LeaseStart = &start
	onlyEnd := base
	onlyEnd.LeaseEnd = &start
	for name, tx := range map[string]transaction.Transaction{"only-start": onlyStart, "only-end": onlyEnd, "neither": base} {
		if err := tx.Validate(cfg); err != nil {
			t.Fatalf("Validate(lease, %s) err = %v, want nil", name, err)
		}
	}
}

// TestValidate_Sale_Success is the baseline success case: an enabled sale deal
// with a positive, currency-bearing price validates clean.
func TestValidate_Sale_Success(t *testing.T) {
	cfg := brConfig()
	tx := transaction.Transaction{
		Type:       transaction.TypeSale,
		Status:     transaction.StatusOpen,
		PropertyID: "prop-1",
		SalePrice:  money.New(2_100_000_000, "THB"),
	}
	if err := tx.Validate(cfg); err != nil {
		t.Fatalf("Validate(sale, valid) err = %v, want nil", err)
	}
}
