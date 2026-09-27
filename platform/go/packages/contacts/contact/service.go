package contact

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shredbx/sbx-core/pkg/repository"
)

// ErrValidation wraps every validation failure. Callers should test with
// errors.Is(err, contact.ErrValidation).
var ErrValidation = errors.New("validation")

// ContactService is the application-service for the Contact entity.
// All persistence goes through repo (Repository[Contact]); pool is used for
// search and any ad-hoc SQL the typed Repository doesn't expose.
type ContactService struct {
	repo   repository.Repository[Contact]
	pool   *pgxpool.Pool
	schema string
}

// NewContactService wires the service. Pass nil repo only in tests that don't
// touch the database (the service still validates and normalizes).
func NewContactService(repo repository.Repository[Contact], pool *pgxpool.Pool, schema string) *ContactService {
	return &ContactService{repo: repo, pool: pool, schema: schema}
}

// Create validates the input, stamps id/timestamps, persists the contact row,
// then persists its category set. Returns the stored Contact with its assigned
// UUID, CreatedAt/UpdatedAt, and CategoryCodes set.
//
// The category set is a normalized link table, not a contacts column, so it is
// written by the CategoryStore companion AFTER the primary row (the same
// primary-then-companion shape PropertyService uses for highlights/amenities).
// The companion Save is itself transactional (delete-all + insert-set in one tx),
// so the category set is written atomically. Validate already enforced the
// min-one invariant, so Save will not reject the set here.
func (s *ContactService) Create(ctx context.Context, c Contact) (Contact, error) {
	if err := c.Validate(); err != nil {
		return Contact{}, fmt.Errorf("%w: %v", ErrValidation, err)
	}
	c.ID = uuid.NewString()
	now := time.Now()
	c.CreatedAt = now
	c.UpdatedAt = now

	if s.repo == nil {
		return c, nil
	}
	result, err := s.repo.Create(ctx, c)
	if err != nil {
		return result, err
	}
	if err := s.saveCategories(ctx, result.ID, c.CategoryCodes); err != nil {
		return result, fmt.Errorf("create: %w", err)
	}
	result.CategoryCodes = c.CategoryCodes
	return result, nil
}

// Update validates, stamps UpdatedAt, and persists. Soft-deleted contacts are
// not updatable; consumers must reactivate first (not yet implemented — for v1
// soft-delete is treated as terminal).
func (s *ContactService) Update(ctx context.Context, id string, c Contact) (Contact, error) {
	if err := c.Validate(); err != nil {
		return Contact{}, fmt.Errorf("%w: %v", ErrValidation, err)
	}
	c.ID = id
	c.UpdatedAt = time.Now()

	if s.repo == nil {
		return c, nil
	}
	result, err := s.repo.Update(ctx, id, c)
	if err != nil {
		return result, err
	}
	// The primary repo.Update writes only the contacts row (the mapper's ToRow no
	// longer carries category_code). Replace the category set via the companion —
	// a PUT always submits the full set (Validate guaranteed ≥1), so this is a
	// full-replace, mirroring how PropertyService replaces highlights on update.
	if err := s.saveCategories(ctx, id, c.CategoryCodes); err != nil {
		return result, fmt.Errorf("update: %w", err)
	}
	result.CategoryCodes = c.CategoryCodes
	return result, nil
}

// Get fetches a single contact by id, with its category set AND its note
// collection attached. Returns repository.ErrNotFound if absent.
//
// Neither the category set (contact_category_links join table) nor the notes
// (contact_notes child table) are contacts columns, so each is loaded via its
// companion store after the primary row — CategoryStore.Load for the codes and
// NoteStore.Load for the ordered notes (created_at ascending).
func (s *ContactService) Get(ctx context.Context, id string) (Contact, error) {
	if s.repo == nil {
		return Contact{}, errors.New("repo not configured")
	}
	c, err := s.repo.Get(ctx, id)
	if err != nil {
		return c, err
	}
	if s.pool != nil {
		byID, lerr := NewCategoryStore(s.pool, s.schema).Load(ctx, id)
		if lerr != nil {
			return c, fmt.Errorf("get: load categories: %w", lerr)
		}
		c.CategoryCodes = byID[id]

		notesByID, nerr := NewNoteStore(s.pool, s.schema).Load(ctx, id)
		if nerr != nil {
			return c, fmt.Errorf("get: load notes: %w", nerr)
		}
		c.Notes = notesByID[id]
	}
	return c, nil
}

// saveCategories persists a contact's category set via the CategoryStore companion
// inside its own transaction (the multi-statement delete-all + insert-set must be
// atomic). A nil pool (test wiring) is a no-op. Save enforces the min-one
// invariant, but Validate already guaranteed ≥1 on every Create/Update path.
func (s *ContactService) saveCategories(ctx context.Context, contactID string, codes []string) error {
	if s.pool == nil {
		return nil
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("save categories: begin: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := NewCategoryStore(s.pool, s.schema).Save(ctx, tx, contactID, codes); err != nil {
		return fmt.Errorf("save categories: %w", err)
	}
	return tx.Commit(ctx)
}

// Delete soft-deletes the contact. Consuming entities keep their FK; UI surfaces
// the contact as "(deactivated)" via the deleted_at timestamp on subsequent reads.
func (s *ContactService) Delete(ctx context.Context, id string) error {
	if s.repo == nil {
		return errors.New("repo not configured")
	}
	return s.repo.Delete(ctx, id)
}

// List returns paginated contacts subject to the filter in opts.
//
// Callers typically build filters via repository.Field[T].Eq() / In() against
// the Fields struct in mapper.generated.go. Category is NOT a contacts column
// anymore (it lives in the contact_category_links join table) — filter by
// category via Search (which compiles CategoryStore.CategoryFilterClause) and
// attach codes via CategoryStore.Load.
func (s *ContactService) List(ctx context.Context, opts repository.ListOptions) ([]Contact, int, error) {
	if s.repo == nil {
		return nil, 0, nil
	}
	return s.repo.List(ctx, opts)
}

// SearchOptions is the input shape for Search.
//
// Query is matched against given_name, surname, phone_number, and email
// (case-insensitive substring). Category narrows to one contact category
// when set. Limit caps result count; default 50 if zero, hard cap 200.
type SearchOptions struct {
	Query    string
	Category string
	Limit    int
	Offset   int
}

// Search performs a free-text search across name/phone/email columns plus an
// optional category filter. Implemented in Go because Postgres pg_trgm is not
// guaranteed across all consumer schemas and the contact dataset stays small
// enough that ILIKE on a B-tree is performant.
//
// Order: surname ASC, given_name ASC, id ASC (deterministic tie-breaker).
func (s *ContactService) Search(ctx context.Context, opts SearchOptions) ([]Contact, int, error) {
	if s.repo == nil || s.pool == nil {
		return nil, 0, nil
	}
	limit := opts.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	conds := []string{"c.deleted_at IS NULL"}
	args := []any{}

	if cat := strings.TrimSpace(opts.Category); cat != "" {
		// The category set is now a normalized link table (contact_category_links),
		// not a contacts column — match via an EXISTS subquery (ANY-of): a contact
		// matches when ANY of its codes equals the requested one.
		args = append(args, cat)
		store := NewCategoryStore(s.pool, s.schema)
		conds = append(conds, store.CategoryFilterClause("c", fmt.Sprintf("$%d", len(args))))
	}
	if q := strings.TrimSpace(opts.Query); q != "" {
		args = append(args, "%"+q+"%")
		i := len(args)
		conds = append(conds, fmt.Sprintf(
			"(c.given_name ILIKE $%d OR c.surname ILIKE $%d OR c.phone_number ILIKE $%d OR c.email ILIKE $%d)",
			i, i, i, i,
		))
	}

	where := "WHERE " + strings.Join(conds, " AND ")
	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM %s.contacts c %s", s.schema, where)

	var total int
	if err := s.pool.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("contact search: count: %w", err)
	}
	if total == 0 {
		return []Contact{}, 0, nil
	}

	// Mapper Columns() already include the primary `c.` alias prefix.
	listSQL := fmt.Sprintf(`SELECT %s FROM %s.contacts c %s ORDER BY c.surname ASC, c.given_name ASC, c.id ASC LIMIT $%d OFFSET $%d`,
		strings.Join(PostgresMapper{}.Columns(), ", "),
		s.schema, where, len(args)+1, len(args)+2,
	)
	args = append(args, limit, opts.Offset)

	rows, err := s.pool.Query(ctx, listSQL, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("contact search: query: %w", err)
	}
	defer rows.Close()

	mapper := NewPostgresMapper(s.schema)
	var contacts []Contact
	for rows.Next() {
		c, err := mapper.FromRow(rows.Scan)
		if err != nil {
			return nil, 0, fmt.Errorf("contact search: from row: %w", err)
		}
		contacts = append(contacts, c)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("contact search: rows.Err: %w", err)
	}
	return contacts, total, nil
}
