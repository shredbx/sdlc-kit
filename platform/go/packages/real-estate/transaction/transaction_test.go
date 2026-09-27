package transaction_test

import (
	"testing"

	"github.com/shredbx/sbx-core/pkg/transaction"
)

// TestTransition_OpenToClosed covers the allowed status transitions (SC-022-01).
// SUCCESS: open->closed and open->cancelled are allowed.
func TestTransition_OpenToClosed(t *testing.T) {
	if !transaction.CanTransition(transaction.StatusOpen, transaction.StatusClosed) {
		t.Errorf("CanTransition(open, closed) = false, want true")
	}
	if !transaction.CanTransition(transaction.StatusOpen, transaction.StatusCancelled) {
		t.Errorf("CanTransition(open, cancelled) = false, want true")
	}
}

// TestTransition_ReopenAndTerminal asserts the asymmetric reopen rule (Decision
// #0314): cancelled->open is ALLOWED (Reopen — a cancel touches no property state,
// so reopening is a pure status flip), while CLOSED stays TERMINAL (closed->open
// and closed->cancelled forbidden — a close flips the subject property lifecycle,
// so it cannot silently un-flip). A no-op (closed->closed) is not a transition.
func TestTransition_ReopenAndTerminal(t *testing.T) {
	cases := []struct {
		name     string
		from, to transaction.TransactionStatus
		want     bool
	}{
		{"cancelled->open (reopen)", transaction.StatusCancelled, transaction.StatusOpen, true},
		{"closed->open (terminal)", transaction.StatusClosed, transaction.StatusOpen, false},
		{"closed->cancelled (terminal)", transaction.StatusClosed, transaction.StatusCancelled, false},
		{"closed->closed (no-op)", transaction.StatusClosed, transaction.StatusClosed, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := transaction.CanTransition(tc.from, tc.to); got != tc.want {
				t.Errorf("CanTransition(%s, %s) = %v, want %v", tc.from, tc.to, got, tc.want)
			}
		})
	}
}

// TestClose_ImpliesFacet covers facet resolution on close. Every deal type
// composes exactly ONE facet, so the facet is implied (an explicit pick that
// matches is accepted; an empty pick resolves to the implied facet).
func TestClose_ImpliesFacet(t *testing.T) {
	cases := []struct {
		name   string
		typ    transaction.TransactionType
		picked transaction.Facet
		want   transaction.Facet
	}{
		{"sale implies sale", transaction.TypeSale, "", transaction.FacetSale},
		{"lease implies lease", transaction.TypeLease, "", transaction.FacetLease},
		{"sale matching pick", transaction.TypeSale, transaction.FacetSale, transaction.FacetSale},
		{"lease matching pick", transaction.TypeLease, transaction.FacetLease, transaction.FacetLease},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := transaction.ResolveCloseFacet(tc.typ, tc.picked)
			if err != nil {
				t.Fatalf("ResolveCloseFacet(%s, %q) unexpected err: %v", tc.typ, tc.picked, err)
			}
			if got != tc.want {
				t.Fatalf("ResolveCloseFacet(%s, %q) = %q, want %q", tc.typ, tc.picked, got, tc.want)
			}
		})
	}
}

// TestClose_AlreadyClosed is the failure twin (SC-023-04): re-closing an
// already-closed deal is rejected; the transition guard forbids closed->closed.
func TestClose_AlreadyClosed(t *testing.T) {
	if transaction.CanTransition(transaction.StatusClosed, transaction.StatusClosed) {
		t.Fatalf("CanTransition(closed, closed) = true, want false (already closed)")
	}
}

// TestOutcomeFor maps a closed facet to the engine-local Outcome it drives,
// which the consumer maps to its property lifecycle (SC-023-01, SC-023-02).
func TestOutcomeFor(t *testing.T) {
	cases := []struct {
		facet transaction.Facet
		want  transaction.Outcome
	}{
		{transaction.FacetSale, transaction.OutcomeSold},
		{transaction.FacetLease, transaction.OutcomeLeased},
	}
	for _, tc := range cases {
		t.Run(string(tc.facet), func(t *testing.T) {
			if got := transaction.OutcomeFor(tc.facet); got != tc.want {
				t.Fatalf("OutcomeFor(%q) = %q, want %q", tc.facet, got, tc.want)
			}
		})
	}
}

// TestMonths_Term confirms the Term constructor stores the month count.
func TestMonths_Term(t *testing.T) {
	if got := transaction.Months(360); got != transaction.Term(360) {
		t.Fatalf("Months(360) = %d, want 360", got)
	}
}
