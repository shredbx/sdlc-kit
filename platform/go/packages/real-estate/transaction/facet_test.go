package transaction_test

import (
	"testing"

	"github.com/shredbx/sbx-core/pkg/transaction"
)

// facetSetEqual reports whether two facet slices hold the same facets in order.
func facetSetEqual(a, b []transaction.Facet) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestFacetsFor covers the code mapper TransactionType -> facet SET (usage-spec 2a).
// SUCCESS: sale/lease each resolve to their compiled single facet.
// FAILURE/EDGE: an unknown code (rent — not compiled, and the now-removed
// freehold-leasehold) resolves to nil.
func TestFacetsFor_Sale(t *testing.T) {
	got := transaction.FacetsFor(transaction.TypeSale)
	want := []transaction.Facet{transaction.FacetSale}
	if !facetSetEqual(got, want) {
		t.Fatalf("FacetsFor(sale) = %v, want %v", got, want)
	}
}

func TestFacetsFor_Lease(t *testing.T) {
	got := transaction.FacetsFor(transaction.TypeLease)
	want := []transaction.Facet{transaction.FacetLease}
	if !facetSetEqual(got, want) {
		t.Fatalf("FacetsFor(lease) = %v, want %v", got, want)
	}
}

func TestFacetsFor_FreeholdLeasehold_Removed(t *testing.T) {
	// freehold-leasehold is no longer a compiled code (the composed type was
	// removed) — it is now an unknown code with no facet set.
	got := transaction.FacetsFor(transaction.TransactionType("freehold-leasehold"))
	if got != nil {
		t.Fatalf("FacetsFor(freehold-leasehold) = %v, want nil (removed composed code)", got)
	}
}

func TestFacetsFor_Unknown(t *testing.T) {
	got := transaction.FacetsFor(transaction.TransactionType("rent"))
	if got != nil {
		t.Fatalf("FacetsFor(rent) = %v, want nil (uncompiled code has no facet set)", got)
	}
}

// TestEnabledSet_IsEnabled covers consumer enablement (usage-spec 2a/2d).
// SUCCESS: an enabled code reports true; EDGE: a not-enabled but compiled code reports false.
func TestEnabledSet_IsEnabled(t *testing.T) {
	cfg := transaction.EnabledSet(transaction.TypeSale, transaction.TypeLease)

	if !cfg.IsEnabled(transaction.TypeSale) {
		t.Errorf("IsEnabled(sale) = false, want true")
	}
	if !cfg.IsEnabled(transaction.TypeLease) {
		t.Errorf("IsEnabled(lease) = false, want true")
	}
	// rent is not compiled and not enabled.
	if cfg.IsEnabled(transaction.TransactionType("rent")) {
		t.Errorf("IsEnabled(rent) = true, want false")
	}
}

// TestEnabledSet_PanicsOnUnknown is the config-time failure twin (usage-spec 2d):
// enabling a code with no compiled facet set fails fast at startup.
func TestEnabledSet_PanicsOnUnknown(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("EnabledSet(rent) did not panic; want fail-fast on uncompiled code")
		}
	}()
	transaction.EnabledSet(transaction.TransactionType("rent"))
}

// TestClose_FacetNotInType is the failure twin for close-facet resolution
// (usage-spec 2c): picking a facet the deal type does not compose is rejected.
// A sale deal closed under the lease facet must surface ErrFacetNotInType.
func TestClose_FacetNotInType(t *testing.T) {
	_, err := transaction.ResolveCloseFacet(transaction.TypeSale, transaction.FacetLease)
	if err == nil {
		t.Fatalf("ResolveCloseFacet(sale, lease) err = nil, want ErrFacetNotInType")
	}
	if err != transaction.ErrFacetNotInType {
		t.Fatalf("ResolveCloseFacet(sale, lease) err = %v, want ErrFacetNotInType", err)
	}
}
