package transaction

import (
	"context"
	"fmt"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/shredbx/sbx-core/pkg/repository"
	"gopkg.in/yaml.v3"
)

// YAMLStore is a fixture-mode (Decision #0167) implementation of the generic
// repository.Repository[Transaction] half of TransactionRepository. It loads a
// deterministic set of deals from a YAML seed file and serves CRUD + filtered
// List entirely in memory — no database. Mirrors pkg/property.YAMLStore.
//
// The deal-domain operations (Close/Cancel/AddParty/RemoveParty) are NOT on the
// store; they live on TransactionService, which composes this store. Parties are
// likewise owned by the service (the generic repo only knows the deal row),
// mirroring how PropertyService owns the attribute-group Save*/Load* methods.
type YAMLStore struct {
	mu    sync.RWMutex
	items map[string]Transaction
}

// Compile-time check: YAMLStore satisfies the generic CRUD interface.
var _ repository.Repository[Transaction] = (*YAMLStore)(nil)

// NewYAMLStore loads deals from a YAML seed file into an in-memory store. Each
// deal missing an ID is assigned a UUID; a missing OpenedAt defaults to now so
// fixtures need only carry the fields a test asserts on.
func NewYAMLStore(seedPath string) (*YAMLStore, error) {
	data, err := os.ReadFile(seedPath)
	if err != nil {
		return nil, fmt.Errorf("read seed file: %w", err)
	}

	var list []Transaction
	if err := yaml.Unmarshal(data, &list); err != nil {
		return nil, fmt.Errorf("parse seed YAML: %w", err)
	}

	items := make(map[string]Transaction, len(list))
	for _, txn := range list {
		if txn.ID == "" {
			txn.ID = uuid.NewString()
		}
		if txn.Status == "" {
			txn.Status = StatusOpen
		}
		if txn.OpenedAt.IsZero() {
			txn.OpenedAt = time.Now()
		}
		items[txn.ID] = txn
	}

	return &YAMLStore{items: items}, nil
}

// Get retrieves a deal by ID. Returns repository.ErrNotFound when absent.
func (s *YAMLStore) Get(_ context.Context, id string) (Transaction, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	txn, ok := s.items[id]
	if !ok {
		return Transaction{}, repository.ErrNotFound
	}
	return txn, nil
}

// List returns deals matching the filter, with offset/limit pagination. The
// second return value is the total matched count (before pagination), mirroring
// the generic repository contract.
//
// The matched set is sorted deterministically (newest opened first, id as a
// stable tiebreaker) BEFORE slicing, so successive offset/limit pages partition
// the ledger without overlap or gaps — pagination over a map's random iteration
// order would otherwise re-shuffle each call and double-count across pages. This
// mirrors TransactionService.List's default Sort (which the Postgres store
// renders as ORDER BY), keeping the two backends' page order consistent.
func (s *YAMLStore) List(_ context.Context, opts repository.ListOptions) ([]Transaction, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []Transaction
	for _, txn := range s.items {
		if !matchQuery(txn, opts.Filter) {
			continue
		}
		result = append(result, txn)
	}

	sort.Slice(result, func(i, j int) bool {
		if !result[i].OpenedAt.Equal(result[j].OpenedAt) {
			return result[i].OpenedAt.After(result[j].OpenedAt) // newest first
		}
		return result[i].ID < result[j].ID // stable tiebreaker
	})

	total := len(result)

	if opts.Offset > 0 && opts.Offset < len(result) {
		result = result[opts.Offset:]
	} else if opts.Offset >= len(result) {
		result = nil
	}
	if opts.Limit > 0 && opts.Limit < len(result) {
		result = result[:opts.Limit]
	}

	return result, total, nil
}

// Create inserts a new deal. Assigns an ID and stamps OpenedAt when unset; the
// status defaults to open. (TransactionService.Create owns validation + the
// domain defaults; this is the raw persistence step.)
func (s *YAMLStore) Create(_ context.Context, txn Transaction) (Transaction, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if txn.ID == "" {
		txn.ID = uuid.NewString()
	}
	if txn.Status == "" {
		txn.Status = StatusOpen
	}
	if txn.OpenedAt.IsZero() {
		txn.OpenedAt = time.Now()
	}
	s.items[txn.ID] = txn
	return txn, nil
}

// Update replaces an existing deal by ID. Returns repository.ErrNotFound when
// the deal does not exist.
func (s *YAMLStore) Update(_ context.Context, id string, txn Transaction) (Transaction, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.items[id]; !ok {
		return Transaction{}, repository.ErrNotFound
	}
	txn.ID = id
	s.items[id] = txn
	return txn, nil
}

// Delete removes a deal by ID. Transactions are an append-mostly ledger with no
// soft-delete column, so this is a hard delete from the fixture map. Returns
// repository.ErrNotFound when the deal does not exist.
func (s *YAMLStore) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.items[id]; !ok {
		return repository.ErrNotFound
	}
	delete(s.items, id)
	return nil
}

// matchQuery evaluates a repository.Query tree against one deal (AND/OR/leaf),
// mirroring pkg/property.matchQuery. An empty query matches everything.
func matchQuery(txn Transaction, q repository.Query) bool {
	if q.IsEmpty() {
		return true
	}
	if q.Pred != nil {
		return matchPredicate(txn, *q.Pred)
	}
	if len(q.And) > 0 {
		for _, sub := range q.And {
			if !matchQuery(txn, sub) {
				return false
			}
		}
		return true
	}
	if len(q.Or) > 0 {
		for _, sub := range q.Or {
			if matchQuery(txn, sub) {
				return true
			}
		}
		return false
	}
	return true
}

// matchPredicate evaluates one equality predicate against a deal. Only the
// filterable axes (type / status / property_id) are matched; discriminators are
// compared in their string form, consistent with how Fields binds them as
// Field[string] and TransactionFilter.ToQuery renders them. Unknown fields and
// non-Eq operators are permissive (match) — the generic List contract narrows
// only on what it understands.
func matchPredicate(txn Transaction, pred repository.Predicate) bool {
	if pred.Op != repository.OpEq {
		return true
	}
	v, ok := pred.Value.(string)
	if !ok {
		return true
	}
	switch pred.FieldName {
	case "type":
		return string(txn.Type) == v
	case "status":
		return string(txn.Status) == v
	case "property_id":
		return txn.PropertyID == v
	}
	return true
}
