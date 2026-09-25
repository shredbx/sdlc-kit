package transaction_test

import (
	"context"
	"errors"
	"testing"

	"github.com/shredbx/sbx-core/pkg/money"
	"github.com/shredbx/sbx-core/pkg/transaction"
)

// MD-layer tests for the deal engine (TransactionService over the YAMLStore
// fixture repository — Decision #0167 fixture mode, NO database).
//
// Two classes of test live here:
//
//   - ENGINE-TESTABLE (fleshed green below): create / parties / list-filter /
//     close-transition / outcome. These exercise the decoupled engine end-to-end
//     against deterministic YAML fixtures, no DB.
//   - PROPERTY-AWARE (kept as explicit skips): the property.lifecycle_status flip
//     a close drives is the BR CONSUMER's job at the SI layer — the engine is
//     property-decoupled by design (it returns an engine-local Outcome and never
//     imports pkg/property). Those are verified at the SI layer, not here.
//
// See .sbx/.runtime/tasks/2606-001/artifacts/test-strategy.md §1.

// brConfig (FX-CFG-BR: sale, lease) is defined in validate_test.go and shared
// across the transaction_test package.

// newFixtureService loads the deterministic seed into a YAMLStore and wires a
// TransactionService over it (no pool, no schema — pure fixture mode).
func newFixtureService(t *testing.T) *transaction.TransactionService {
	t.Helper()
	store, err := transaction.NewYAMLStore("testdata/seed.yml")
	if err != nil {
		t.Fatalf("NewYAMLStore: %v", err)
	}
	return transaction.NewTransactionService(store, brConfig(), nil, "")
}

// Stable fixture IDs (see testdata/seed.yml).
const (
	dealSaleOpen   = "d0000000-0000-0000-0000-000000000001"
	dealLeaseOpen  = "d0000000-0000-0000-0000-000000000002"
	dealClosed     = "d0000000-0000-0000-0000-000000000004"
	propActive     = "11111111-1111-1111-1111-111111111111"
	contactSomchai = "c0000000-0000-0000-0000-0000000000a1"
)

// TestService_Create_Sale — TC-019-01 / SC-019-01: create a sale deal on a
// property. The deal is born open, OpenedAt stamped, and the sale price round-
// trips through the fixture store.
func TestService_Create_Sale(t *testing.T) {
	svc := newFixtureService(t)

	in := transaction.Transaction{
		Type:       transaction.TypeSale,
		PropertyID: propActive,
		SalePrice:  money.New(2000000000, "THB"),
	}
	created, err := svc.Create(context.Background(), in)
	if err != nil {
		t.Fatalf("Create sale: %v", err)
	}
	if created.ID == "" {
		t.Error("Create: expected a generated ID")
	}
	if created.Status != transaction.StatusOpen {
		t.Errorf("Create: status = %q, want open", created.Status)
	}
	if created.OpenedAt.IsZero() {
		t.Error("Create: OpenedAt was not stamped")
	}
	if created.ClosedAs != nil || created.ClosedAt != nil {
		t.Error("Create: a new deal must not be closed")
	}
	if created.SalePrice.Amount != 2000000000 || created.SalePrice.Currency != "THB" {
		t.Errorf("Create: sale price = %d %s, want 2000000000 THB",
			created.SalePrice.Amount, created.SalePrice.Currency)
	}

	// It is retrievable and counted.
	got, err := svc.Get(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("Get after Create: %v", err)
	}
	if got.ID != created.ID {
		t.Errorf("Get after Create: id = %q, want %q", got.ID, created.ID)
	}
}

// TestService_Create_NoType_Rejected — failure twin (SC-019-02): a deal with no
// type is rejected by validation before persistence.
func TestService_Create_NoType_Rejected(t *testing.T) {
	svc := newFixtureService(t)
	_, err := svc.Create(context.Background(), transaction.Transaction{PropertyID: propActive})
	if !errors.Is(err, transaction.ErrTypeRequired) {
		t.Fatalf("Create no-type: err = %v, want ErrTypeRequired", err)
	}
}

// TestService_AddParty_Buyer — TC-021-01 / SC-021-01: assign a contact as buyer.
// contact_id is a plain reference string — G12 Contact Book is not merged yet, so
// the engine validates only the role, not the contact's existence.
func TestService_AddParty_Buyer(t *testing.T) {
	svc := newFixtureService(t)

	err := svc.AddParty(context.Background(), dealSaleOpen, transaction.TransactionParty{
		Role:      transaction.RoleBuyer,
		ContactID: contactSomchai,
	})
	if err != nil {
		t.Fatalf("AddParty buyer: %v", err)
	}

	parties := svc.Parties(context.Background(), dealSaleOpen)
	if len(parties) != 1 {
		t.Fatalf("Parties: got %d, want 1", len(parties))
	}
	p := parties[0]
	if p.Role != transaction.RoleBuyer {
		t.Errorf("party role = %q, want buyer", p.Role)
	}
	if p.ContactID != contactSomchai {
		t.Errorf("party contact_id = %q, want %q", p.ContactID, contactSomchai)
	}
	if p.TransactionID != dealSaleOpen {
		t.Errorf("party transaction_id = %q, want %q", p.TransactionID, dealSaleOpen)
	}
	if p.ID == "" {
		t.Error("party: expected a generated ID")
	}
}

// TestService_AddParty_NoRole_Rejected — failure twin (SC-021-02): a party with
// no role is rejected.
func TestService_AddParty_NoRole_Rejected(t *testing.T) {
	svc := newFixtureService(t)
	err := svc.AddParty(context.Background(), dealSaleOpen, transaction.TransactionParty{
		ContactID: contactSomchai,
	})
	if !errors.Is(err, transaction.ErrPartyRoleRequired) {
		t.Fatalf("AddParty no-role: err = %v, want ErrPartyRoleRequired", err)
	}
}

// TestService_List_FilterByType — TC-024-01 / SC-024-01: list deals filtered by
// kind. Filtering by `lease` returns ONLY the lease deal.
func TestService_List_FilterByType(t *testing.T) {
	svc := newFixtureService(t)

	leaseType := transaction.TypeLease
	items, total, err := svc.List(context.Background(), transaction.TransactionFilter{Type: &leaseType})
	if err != nil {
		t.Fatalf("List by type: %v", err)
	}
	if total != 1 || len(items) != 1 {
		t.Fatalf("List lease: total=%d items=%d, want 1/1", total, len(items))
	}
	if items[0].ID != dealLeaseOpen {
		t.Errorf("List lease: id = %q, want %q", items[0].ID, dealLeaseOpen)
	}
	if items[0].Type != transaction.TypeLease {
		t.Errorf("List lease: type = %q, want lease", items[0].Type)
	}

	// Unfiltered returns the whole ledger (3 fixtures).
	_, allTotal, err := svc.List(context.Background(), transaction.TransactionFilter{})
	if err != nil {
		t.Fatalf("List all: %v", err)
	}
	if allTotal != 3 {
		t.Errorf("List all: total = %d, want 3", allTotal)
	}
}

// TestService_List_LimitOffset — pagination contract (A1-backend FIX #1 + FIX #8).
// With N (3) fixtures and a page smaller than N, List must:
//   - return at most `limit` items (the data page is capped),
//   - keep `total` equal to the FULL filtered count (pagination never shrinks the
//     total — the web loader's "N deals to export" reads this number, and a shrunk
//     total would under/over-count),
//   - skip exactly `offset` rows (a page beyond the data returns zero rows, never a
//     wrap-around re-fetch of the whole set — the root cause of the ~2× export
//     over-count),
//   - partition the ledger: walking the pages yields every deal exactly once
//     (DISTINCT deals across pages == total), so accumulating pages can never
//     double-count.
func TestService_List_LimitOffset(t *testing.T) {
	svc := newFixtureService(t)
	ctx := context.Background()

	const fullTotal = 3 // the seed ledger size (see testdata/seed.yml)

	// A page smaller than the ledger caps the items but reports the full total.
	page1, total, err := svc.List(ctx, transaction.TransactionFilter{Limit: 2, Offset: 0})
	if err != nil {
		t.Fatalf("List page1: %v", err)
	}
	if total != fullTotal {
		t.Errorf("List page1: total = %d, want %d (pagination must NOT shrink total)", total, fullTotal)
	}
	if len(page1) != 2 {
		t.Errorf("List page1: items = %d, want 2 (limit must cap the page)", len(page1))
	}

	// The next page skips the first `offset` rows; total is still the full count.
	// With 3 fixtures and limit 2, the second page holds the single remaining deal.
	page2, total2, err := svc.List(ctx, transaction.TransactionFilter{Limit: 2, Offset: 2})
	if err != nil {
		t.Fatalf("List page2: %v", err)
	}
	if total2 != fullTotal {
		t.Errorf("List page2: total = %d, want %d", total2, fullTotal)
	}
	if len(page2) != 1 {
		t.Errorf("List page2: items = %d, want 1", len(page2))
	}

	// A page whose offset is at/after the end returns ZERO rows — never the whole
	// set again (the export double-count was a wrap-around re-fetch on the page
	// past the data). Total stays the full count.
	beyond, totalBeyond, err := svc.List(ctx, transaction.TransactionFilter{Limit: 2, Offset: fullTotal})
	if err != nil {
		t.Fatalf("List beyond: %v", err)
	}
	if totalBeyond != fullTotal {
		t.Errorf("List beyond: total = %d, want %d", totalBeyond, fullTotal)
	}
	if len(beyond) != 0 {
		t.Errorf("List beyond end: items = %d, want 0 (offset>=total must not wrap)", len(beyond))
	}

	// Walking the pages yields every deal exactly once — DISTINCT deals across the
	// pages equals total, so an accumulating consumer can never double-count.
	seen := make(map[string]bool, fullTotal)
	for _, it := range append(append([]transaction.Transaction{}, page1...), page2...) {
		if seen[it.ID] {
			t.Errorf("pagination returned deal %s twice — pages overlap", it.ID)
		}
		seen[it.ID] = true
	}
	if len(seen) != fullTotal {
		t.Errorf("DISTINCT deals across pages = %d, want %d (must equal total)", len(seen), fullTotal)
	}

	// limit=0 means "no cap" (defaults are applied at the SI/handler boundary, not
	// here) — the service must return the whole filtered set when no limit is set.
	all, allTotal, err := svc.List(ctx, transaction.TransactionFilter{})
	if err != nil {
		t.Fatalf("List all: %v", err)
	}
	if allTotal != fullTotal || len(all) != fullTotal {
		t.Errorf("List unpaged: items=%d total=%d, want %d/%d", len(all), allTotal, fullTotal, fullTotal)
	}
}

// TestService_List_FilterByStatus narrows the ledger to open deals (2 of 3).
func TestService_List_FilterByStatus(t *testing.T) {
	svc := newFixtureService(t)
	open := transaction.StatusOpen
	_, total, err := svc.List(context.Background(), transaction.TransactionFilter{Status: &open})
	if err != nil {
		t.Fatalf("List by status: %v", err)
	}
	if total != 2 {
		t.Errorf("List open: total = %d, want 2", total)
	}
}

// TestService_Close_SetsClosedAsAndOutcome — closing an open sale deal sets
// status=closed, closed_as=sale, stamps closed_at, and returns the engine-local
// Outcome (sold) the consumer maps to property lifecycle (SC-023-01). The engine
// does NOT touch any property — that flip is the SI consumer's job.
func TestService_Close_SetsClosedAsAndOutcome(t *testing.T) {
	svc := newFixtureService(t)

	closed, outcome, err := svc.Close(context.Background(), dealSaleOpen, "")
	if err != nil {
		t.Fatalf("Close sale: %v", err)
	}
	if closed.Status != transaction.StatusClosed {
		t.Errorf("Close: status = %q, want closed", closed.Status)
	}
	if closed.ClosedAs == nil || *closed.ClosedAs != transaction.FacetSale {
		t.Errorf("Close: closed_as = %v, want sale", closed.ClosedAs)
	}
	if closed.ClosedAt == nil {
		t.Error("Close: closed_at was not stamped")
	}
	if outcome != transaction.OutcomeSold {
		t.Errorf("Close: outcome = %q, want sold", outcome)
	}

	// Persisted: a re-fetch reflects the closed state.
	got, err := svc.Get(context.Background(), dealSaleOpen)
	if err != nil {
		t.Fatalf("Get after Close: %v", err)
	}
	if got.Status != transaction.StatusClosed {
		t.Errorf("Get after Close: status = %q, want closed", got.Status)
	}
}

// TestService_Close_Lease_SetsClosedAsAndOutcome — closing an open lease deal
// implies the lease facet, sets closed_as=lease, and returns the engine-local
// Outcome (leased) the consumer maps to property lifecycle (SC-023-02).
func TestService_Close_Lease_SetsClosedAsAndOutcome(t *testing.T) {
	svc := newFixtureService(t)

	closed, outcome, err := svc.Close(context.Background(), dealLeaseOpen, "")
	if err != nil {
		t.Fatalf("Close lease: %v", err)
	}
	if closed.ClosedAs == nil || *closed.ClosedAs != transaction.FacetLease {
		t.Errorf("Close lease: closed_as = %v, want lease", closed.ClosedAs)
	}
	if outcome != transaction.OutcomeLeased {
		t.Errorf("Close lease: outcome = %q, want leased", outcome)
	}
}

// TestService_Close_AlreadyClosed — TC-023-04 / SC-023-04: re-closing a closed
// deal is rejected, and the deal's closed state is unchanged.
func TestService_Close_AlreadyClosed(t *testing.T) {
	svc := newFixtureService(t)

	before, err := svc.Get(context.Background(), dealClosed)
	if err != nil {
		t.Fatalf("Get closed fixture: %v", err)
	}

	_, _, err = svc.Close(context.Background(), dealClosed, transaction.FacetSale)
	if !errors.Is(err, transaction.ErrAlreadyClosed) {
		t.Fatalf("re-close: err = %v, want ErrAlreadyClosed", err)
	}

	after, _ := svc.Get(context.Background(), dealClosed)
	if after.Status != transaction.StatusClosed {
		t.Errorf("re-close mutated status to %q, want closed", after.Status)
	}
	if after.ClosedAt == nil || before.ClosedAt == nil || !after.ClosedAt.Equal(*before.ClosedAt) {
		t.Error("re-close changed closed_at; the closed deal must be untouched")
	}
}

// TestService_Cancel_Open marks an open deal cancelled (terminal); a re-cancel is
// then rejected by the transition guard.
func TestService_Cancel_Open(t *testing.T) {
	svc := newFixtureService(t)

	if err := svc.Cancel(context.Background(), dealLeaseOpen); err != nil {
		t.Fatalf("Cancel open: %v", err)
	}
	got, _ := svc.Get(context.Background(), dealLeaseOpen)
	if got.Status != transaction.StatusCancelled {
		t.Errorf("Cancel: status = %q, want cancelled", got.Status)
	}

	if err := svc.Cancel(context.Background(), dealLeaseOpen); !errors.Is(err, transaction.ErrInvalidTransition) {
		t.Fatalf("re-cancel: err = %v, want ErrInvalidTransition", err)
	}
}

// TestService_Reopen_Cancelled — Decision #0314: a CANCELLED deal reopens to open
// (a pure status flip — cancel deleted nothing, touched no property state). The
// deal's terms/parties survive a cancel, so reopen restores it intact.
func TestService_Reopen_Cancelled(t *testing.T) {
	svc := newFixtureService(t)

	// Bring an open deal to cancelled first (no cancelled fixture exists; this
	// mirrors TestService_Cancel_Open and needs no seed change).
	if err := svc.Cancel(context.Background(), dealLeaseOpen); err != nil {
		t.Fatalf("Cancel open: %v", err)
	}

	reopened, err := svc.Reopen(context.Background(), dealLeaseOpen)
	if err != nil {
		t.Fatalf("Reopen cancelled: %v", err)
	}
	if reopened.Status != transaction.StatusOpen {
		t.Errorf("Reopen: status = %q, want open", reopened.Status)
	}
	// Reopen is a pure status flip — closed_as/closed_at stay nil (they were never
	// set on a cancelled deal).
	if reopened.ClosedAs != nil || reopened.ClosedAt != nil {
		t.Errorf("Reopen: closed_as/closed_at must stay nil, got %v / %v", reopened.ClosedAs, reopened.ClosedAt)
	}

	// Persisted: a re-fetch reflects the reopened (open) state.
	got, err := svc.Get(context.Background(), dealLeaseOpen)
	if err != nil {
		t.Fatalf("Get after Reopen: %v", err)
	}
	if got.Status != transaction.StatusOpen {
		t.Errorf("Get after Reopen: status = %q, want open", got.Status)
	}
}

// TestService_Reopen_NotCancelled_Rejected — failure twin: only a CANCELLED deal
// may reopen. An OPEN deal (no-op) and a CLOSED deal (terminal — a close flipped
// the property lifecycle) both return ErrInvalidTransition, unchanged.
func TestService_Reopen_NotCancelled_Rejected(t *testing.T) {
	svc := newFixtureService(t)

	// Reopening an already-open deal is not a transition.
	if _, err := svc.Reopen(context.Background(), dealSaleOpen); !errors.Is(err, transaction.ErrInvalidTransition) {
		t.Fatalf("Reopen open: err = %v, want ErrInvalidTransition", err)
	}
	openGot, _ := svc.Get(context.Background(), dealSaleOpen)
	if openGot.Status != transaction.StatusOpen {
		t.Errorf("Reopen open mutated status to %q, want open", openGot.Status)
	}

	// A CLOSED deal stays terminal — reopen must NOT un-close it.
	if _, err := svc.Reopen(context.Background(), dealClosed); !errors.Is(err, transaction.ErrInvalidTransition) {
		t.Fatalf("Reopen closed: err = %v, want ErrInvalidTransition", err)
	}
	closedGot, _ := svc.Get(context.Background(), dealClosed)
	if closedGot.Status != transaction.StatusClosed {
		t.Errorf("Reopen closed mutated status to %q, want closed", closedGot.Status)
	}
}

// =============================================================================
// PROPERTY-AWARE — verified at the SI (BR consumer) layer, NOT here.
//
// These assert the property.lifecycle_status side effect of a close/create. The
// shared engine is property-decoupled (it returns an Outcome and does not import
// pkg/property), so the property flip is verified where it lives — the BR
// consumer seam — NOT in this engine package. The coverage is NOT dropped: each
// skip below names the BR integration test that now exercises it against a real
// PostgreSQL (testcontainers), in:
//
//	clients/bestie/projects/bestierealestate/apps/api-chi/internal/deal/
//	    orchestrator_integration_test.go   (build tag: integration)
//
// Kept as explicit skips (not deletions) so the engine's fixture pass stays
// honest — no false greens, and no pkg/property import leaks into the engine.
// =============================================================================

func TestService_Close_Sale_MarksSold(t *testing.T) {
	t.Skip("property flip lives at the BR seam — see internal/deal.TestClose_Sale_MarksSold (integration); the decoupled engine returns Outcome and never imports pkg/property")
}

func TestService_Close_Lease_MarksLeased(t *testing.T) {
	t.Skip("property flip lives at the BR seam — see internal/deal.TestClose_Lease_MarksLeased (integration); the decoupled engine returns Outcome and never imports pkg/property")
}

func TestService_Open_LifecycleUnchanged(t *testing.T) {
	t.Skip("property flip lives at the BR seam — see internal/deal.TestOpen_LifecycleUnchanged (integration); the decoupled engine returns Outcome and never imports pkg/property")
}

func TestService_Create_WarnsWhenSold(t *testing.T) {
	t.Skip("property flip lives at the BR seam — see internal/deal.TestCreate_OnSoldProperty_StillRecords (integration); the decoupled engine returns Outcome and never imports pkg/property")
}
