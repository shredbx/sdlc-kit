package transaction

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shredbx/sbx-core/pkg/money"
	"github.com/shredbx/sbx-core/pkg/repository"
)

// TransactionService is the deal-engine application service. It composes the
// generic repository.Repository[Transaction] (the deal-row CRUD — YAMLStore in
// fixture mode, the generated Postgres mapper in live mode) and adds the
// deal-specific domain operations (Create with validation, Close, Cancel, party
// management). It mirrors pkg/property.PropertyService: the same repo + pool +
// schema composition, with companion money/party persistence the generic repo
// cannot express.
//
// Decoupling invariant: this service NEVER imports a consumer property package.
// Close mutates only the deal and returns an engine-local Outcome (sold|leased);
// the consumer's SI layer maps that Outcome to property.lifecycle_status and
// performs the atomic deal+property write.
type TransactionService struct {
	repo   repository.Repository[Transaction]
	cfg    EnabledTypes
	pool   *pgxpool.Pool
	schema string

	// parties holds TransactionParty rows in fixture mode (no generic-repo slot
	// for a child entity, mirroring how PropertyService owns attribute groups).
	// In live mode the SI layer persists parties to transaction_parties.
	pmu     sync.RWMutex
	parties map[string][]TransactionParty
}

// NewTransactionService constructs the service over a deal repository and the
// consumer's enabled deal-type set. pool + schema are carried for the future
// Postgres companion path (money columns, parties) exactly as PropertyService
// carries them; both may be nil/"" in fixture mode.
func NewTransactionService(repo repository.Repository[Transaction], cfg EnabledTypes, pool *pgxpool.Pool, schema string) *TransactionService {
	return &TransactionService{
		repo:    repo,
		cfg:     cfg,
		pool:    pool,
		schema:  schema,
		parties: make(map[string][]TransactionParty),
	}
}

// =============================================================================
// CRUD + domain operations
// =============================================================================

// Create validates a new deal against the consumer's enabled set and persists
// it. The status is forced to open and OpenedAt stamped here (a deal is born
// open; transitions happen via Close/Cancel, never at create time — mirroring
// PropertyService forcing lifecycle=active at create).
func (s *TransactionService) Create(ctx context.Context, txn Transaction) (Transaction, error) {
	if err := txn.Validate(s.cfg); err != nil {
		return Transaction{}, err
	}
	if txn.ID == "" {
		txn.ID = uuid.NewString()
	}
	txn.Status = StatusOpen
	txn.ClosedAs = nil
	txn.ClosedAt = nil
	if txn.OpenedAt.IsZero() {
		txn.OpenedAt = time.Now()
	}

	if s.repo == nil {
		return txn, nil
	}
	created, err := s.repo.Create(ctx, txn)
	if err != nil {
		return created, err
	}

	// Money columns are companion-persisted (the generated mapper cannot scan a
	// money.Money struct). In fixture mode this is a no-op; in live mode it writes
	// the *_amount/*_currency pairs. Mirrors PropertyService.Save* after Create.
	if err := s.SaveMoney(ctx, created.ID, txn); err != nil {
		return created, fmt.Errorf("create: %w", err)
	}
	return created, nil
}

// Get returns a deal by ID (nil repo → zero value, mirroring PropertyService).
func (s *TransactionService) Get(ctx context.Context, id string) (Transaction, error) {
	if s.repo == nil {
		return Transaction{}, nil
	}
	return s.repo.Get(ctx, id)
}

// List returns deals matching the domain filter (type / status / property),
// paginated by the filter's Limit/Offset. The TransactionFilter compiles to a
// storage-agnostic repository.Query so the same call works against any backend;
// the second return value is the FULL filtered count (before pagination), so a
// paged list never under-reports its total.
//
// A deterministic sort (newest first, id as a stable tiebreaker) is applied so
// successive pages partition the ledger without overlap or gaps — pagination is
// only correct over a stable order. The YAMLStore ignores Sort (its fixture
// result is already deterministic); the Postgres store renders ORDER BY.
func (s *TransactionService) List(ctx context.Context, filter TransactionFilter) ([]Transaction, int, error) {
	if s.repo == nil {
		return nil, 0, nil
	}
	return s.repo.List(ctx, repository.ListOptions{
		Filter: filter.ToQuery(),
		Sort:   []repository.SortField{repository.DescSort("opened_at"), repository.Asc("id")},
		Limit:  filter.Limit,
		Offset: filter.Offset,
	})
}

// Update validates and replaces a deal's mutable terms. Server-owned lifecycle
// fields (status / closed_as / closed_at) are preserved from the stored row so a
// terms edit never silently reopens or re-closes a deal — Close/Cancel are the
// sole writers of lifecycle state (mirrors PropertyService.Update preserving
// is_published / lifecycle_status).
func (s *TransactionService) Update(ctx context.Context, id string, txn Transaction) (Transaction, error) {
	if err := txn.Validate(s.cfg); err != nil {
		return Transaction{}, err
	}
	txn.ID = id

	if s.repo != nil {
		if existing, err := s.repo.Get(ctx, id); err == nil {
			txn.Status = existing.Status
			txn.ClosedAs = existing.ClosedAs
			txn.ClosedAt = existing.ClosedAt
			txn.OpenedAt = existing.OpenedAt
			// The deal's subject + scope are fixed at creation (the update wire
			// shape carries neither). Preserve them from the stored row so a terms
			// edit never wipes property_id (NOT NULL — would fail the write) nor
			// silently drops a unit-scoped deal back to property-level.
			txn.PropertyID = existing.PropertyID
			txn.UnitID = existing.UnitID
		}
	}

	if s.repo == nil {
		return txn, nil
	}
	updated, err := s.repo.Update(ctx, id, txn)
	if err != nil {
		return updated, err
	}
	if err := s.SaveMoney(ctx, id, txn); err != nil {
		return updated, fmt.Errorf("update: %w", err)
	}
	return updated, nil
}

// Close completes an open deal under the resolved facet and returns the deal plus
// the engine-local Outcome (sold | leased) it drives. It does NOT touch the
// property — the consumer's SI layer maps the Outcome to property.lifecycle_status
// and performs the atomic write (this engine is property-decoupled by design).
//
// Flow (per usage-spec):
//   - load the deal; ErrNotFound if absent;
//   - guard the transition: a deal already closed → ErrAlreadyClosed; any other
//     non-open state (cancelled) → ErrInvalidTransition (no reopen, terminal);
//   - resolve the close facet (single-facet implied) — surfaces
//     ErrFacetNotInType / ErrTypeRequired;
//   - set status=closed, closed_as=facet, closed_at=now; persist;
//   - return the closed deal + OutcomeFor(facet).
func (s *TransactionService) Close(ctx context.Context, id string, closedAs Facet) (Transaction, Outcome, error) {
	if s.repo == nil {
		return Transaction{}, "", repository.ErrNotFound
	}
	txn, err := s.repo.Get(ctx, id)
	if err != nil {
		return Transaction{}, "", err
	}

	if txn.Status == StatusClosed {
		return txn, "", ErrAlreadyClosed
	}
	if !CanTransition(txn.Status, StatusClosed) {
		return txn, "", ErrInvalidTransition
	}

	facet, err := ResolveCloseFacet(txn.Type, closedAs)
	if err != nil {
		return txn, "", err
	}

	now := time.Now()
	txn.Status = StatusClosed
	txn.ClosedAs = &facet
	txn.ClosedAt = &now

	updated, err := s.repo.Update(ctx, id, txn)
	if err != nil {
		return updated, "", err
	}
	return updated, OutcomeFor(facet), nil
}

// Cancel abandons an open deal (terminal; the property lifecycle is untouched).
// Returns ErrInvalidTransition if the deal is not open (mirrors the transition
// guard — a closed or already-cancelled deal cannot be cancelled).
func (s *TransactionService) Cancel(ctx context.Context, id string) error {
	if s.repo == nil {
		return repository.ErrNotFound
	}
	txn, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}
	if !CanTransition(txn.Status, StatusCancelled) {
		return ErrInvalidTransition
	}
	txn.Status = StatusCancelled
	_, err = s.repo.Update(ctx, id, txn)
	return err
}

// Reopen revives a CANCELLED deal back to open (Decision #0314). It is the inverse
// of Cancel and, like Cancel, a pure status flip: cancel never deleted the deal's
// parties/terms and touched NO property state, so reopen restores it intact with
// nothing downstream to reconcile. closed_as/closed_at are left untouched (they are
// nil on a cancelled deal — only Close sets them). Returns ErrInvalidTransition if
// the deal is not cancelled (an open deal is a no-op; a CLOSED deal stays terminal —
// a close flipped the property lifecycle and cannot silently un-flip).
func (s *TransactionService) Reopen(ctx context.Context, id string) (Transaction, error) {
	if s.repo == nil {
		return Transaction{}, repository.ErrNotFound
	}
	txn, err := s.repo.Get(ctx, id)
	if err != nil {
		return Transaction{}, err
	}
	if !CanTransition(txn.Status, StatusOpen) {
		return txn, ErrInvalidTransition
	}
	txn.Status = StatusOpen
	return s.repo.Update(ctx, id, txn)
}

// =============================================================================
// PARTIES — child-entity ops (fixture-mode in-memory; live-mode → transaction_parties)
// =============================================================================

// AddParty attaches a validated party (role + contact) to an existing deal. The
// contact_id is a plain reference string (G12 Contact Book is not merged yet, so
// there is no FK to validate against — US-021 is a "should"). Returns the party
// validation error or repository.ErrNotFound if the deal does not exist.
func (s *TransactionService) AddParty(ctx context.Context, txID string, party TransactionParty) error {
	if err := party.Validate(); err != nil {
		return err
	}
	if s.repo != nil {
		if _, err := s.repo.Get(ctx, txID); err != nil {
			return err
		}
	}
	if party.ID == "" {
		party.ID = uuid.NewString()
	}
	party.TransactionID = txID

	s.pmu.Lock()
	s.parties[txID] = append(s.parties[txID], party)
	s.pmu.Unlock()
	return nil
}

// RemoveParty detaches a party from a deal by party ID. Returns
// repository.ErrNotFound if no such party is attached to the deal.
func (s *TransactionService) RemoveParty(_ context.Context, txID string, partyID string) error {
	s.pmu.Lock()
	defer s.pmu.Unlock()

	list := s.parties[txID]
	for i, p := range list {
		if p.ID == partyID {
			s.parties[txID] = append(list[:i], list[i+1:]...)
			return nil
		}
	}
	return repository.ErrNotFound
}

// Parties returns the parties attached to a deal (deterministic insertion order).
// Used by the SI/read layer and by fixture tests to assert AddParty.
func (s *TransactionService) Parties(_ context.Context, txID string) []TransactionParty {
	s.pmu.RLock()
	defer s.pmu.RUnlock()

	src := s.parties[txID]
	out := make([]TransactionParty, len(src))
	copy(out, src)
	return out
}

// =============================================================================
// MONEY — companion persistence (the generated mapper cannot scan money.Money)
// =============================================================================

// SaveMoney writes a deal's facet money values to their *_amount (BIGINT,
// minor-units) + *_currency (TEXT) column pairs. money.Money is a 3-field struct
// with no sql.Scanner/driver.Valuer, so it is intentionally outside the generated
// mapper's `columns:` block — this companion mirrors PropertyService.SaveTags /
// SaveAmenities for the same codegen reason. No-op when pool is nil (fixture mode).
func (s *TransactionService) SaveMoney(ctx context.Context, id string, txn Transaction) error {
	if s.pool == nil {
		return nil
	}
	_, err := s.pool.Exec(ctx,
		`UPDATE `+s.schema+`.transactions SET
			sale_price_amount     = $1, sale_price_currency     = $2,
			lease_rent_amount     = $3, lease_rent_currency     = $4,
			lease_deposit_amount  = $5, lease_deposit_currency  = $6
		 WHERE id = $7`,
		txn.SalePrice.Amount, txn.SalePrice.Currency,
		txn.LeaseRent.Amount, txn.LeaseRent.Currency,
		txn.LeaseDeposit.Amount, txn.LeaseDeposit.Currency,
		id,
	)
	if err != nil {
		return fmt.Errorf("save money: %w", err)
	}
	return nil
}

// LoadMoney reads a deal's facet money columns back into the three money.Money
// fields. Companion to SaveMoney (the generated mapper does not touch these
// columns). No-op when pool is nil (fixture mode). Currency defaults to THB via
// money.New when the stored currency is empty (matches pkg/money's default).
func (s *TransactionService) LoadMoney(ctx context.Context, txn *Transaction) error {
	if s.pool == nil || txn == nil {
		return nil
	}
	var (
		saleAmt, rentAmt, depAmt int64
		saleCur, rentCur, depCur string
	)
	err := s.pool.QueryRow(ctx,
		`SELECT
			COALESCE(sale_price_amount, 0),    COALESCE(sale_price_currency, ''),
			COALESCE(lease_rent_amount, 0),    COALESCE(lease_rent_currency, ''),
			COALESCE(lease_deposit_amount, 0), COALESCE(lease_deposit_currency, '')
		 FROM `+s.schema+`.transactions WHERE id = $1`,
		txn.ID).Scan(&saleAmt, &saleCur, &rentAmt, &rentCur, &depAmt, &depCur)
	if err != nil {
		return fmt.Errorf("load money: %w", err)
	}
	txn.SalePrice = money.New(saleAmt, money.NormalizeCurrencyCode(saleCur))
	txn.LeaseRent = money.New(rentAmt, money.NormalizeCurrencyCode(rentCur))
	txn.LeaseDeposit = money.New(depAmt, money.NormalizeCurrencyCode(depCur))
	return nil
}
