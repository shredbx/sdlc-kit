package contact

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// =============================================================================
// CATEGORY STORE — companion for the contact_category_links join table
// =============================================================================
//
// The contact↔category relation is a normalized many-to-many link table
// (contact_category_links), NOT a column on the contacts row, so the generated
// mapper cannot map it (codegen has no join support — same constraint that
// drives Property.Amenities' LoadAmenities/SaveAmenities companion). This
// hand-written store is that companion: one batch read for list pages (no N+1),
// a transactional delete-all+insert-set write, idempotent add, min-one-guarded
// remove, and the EXISTS directory-filter compiler.
//
// The set is always read ORDERED BY the contact_categories dictionary sort_order,
// so the first code is the primary one — Contact.PrimaryCategory() relies on this.

// ErrEmptyCategorySet is returned by CategoryStore.Save when an empty code set is
// submitted, and by CategoryStore.Remove when the caller tries to remove the last
// remaining code. It encodes the min-one invariant at the store boundary so the
// HTTP layer can map it to 400 (Save) / 409 (Remove). Test with
// errors.Is(err, contact.ErrEmptyCategorySet).
var ErrEmptyCategorySet = errors.New("contact category set must not be empty (min-one invariant)")

// execQuerier is the minimal command surface the write helpers need. Both
// *pgxpool.Pool and pgx.Tx satisfy it, so Save/Add/Remove accept EITHER — the
// handler passes a pgx.Tx to thread the category write into the SAME transaction
// as the contact row write, while tests (or non-transactional callers) may pass
// the pool directly.
type execQuerier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// Compile-time checks that the two concrete pgx types satisfy execQuerier.
var (
	_ execQuerier = (*pgxpool.Pool)(nil)
	_ execQuerier = (pgx.Tx)(nil)
)

// CategoryStore reads and writes a contact's category set against the
// {schema}.contact_category_links table. Construct with NewCategoryStore.
type CategoryStore struct {
	pool   *pgxpool.Pool
	schema string
}

// NewCategoryStore wires the store to a pool + schema. Pass a nil pool only in
// tests that exercise the pure helpers (CategoryFilterClause) without a DB.
func NewCategoryStore(pool *pgxpool.Pool, schema string) CategoryStore {
	return CategoryStore{pool: pool, schema: schema}
}

// Load fetches the category codes for one or more contacts in a SINGLE query
// (no N+1), returning a map of contactID → []code. Each contact's codes are
// ordered by the contact_categories dictionary sort_order (then code as a stable
// tie-breaker), so the first element is that contact's primary category. A
// contact with no links is simply absent from the map (callers treat a missing
// key as "no categories yet"). An empty id list yields an empty map.
func (s CategoryStore) Load(ctx context.Context, contactIDs ...string) (map[string][]string, error) {
	out := make(map[string][]string, len(contactIDs))
	if s.pool == nil || len(contactIDs) == 0 {
		return out, nil
	}

	rows, err := s.pool.Query(ctx,
		`SELECT l.contact_id, l.category_code
		   FROM `+s.schema+`.contact_category_links l
		   JOIN `+s.schema+`.contact_categories d ON d.code = l.category_code
		  WHERE l.contact_id = ANY($1)
		  ORDER BY l.contact_id, d.sort_order, l.category_code`,
		contactIDs)
	if err != nil {
		return nil, fmt.Errorf("load contact categories: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var contactID, code string
		if err := rows.Scan(&contactID, &code); err != nil {
			return nil, fmt.Errorf("scan contact category: %w", err)
		}
		out[contactID] = append(out[contactID], code)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("load contact categories: rows: %w", err)
	}
	return out, nil
}

// Save replaces a contact's entire category set (delete-all + insert-set) using
// the supplied execQuerier — pass a pgx.Tx to keep this write in the same
// transaction as the contact row write. REJECTS an empty set with
// ErrEmptyCategorySet (the min-one invariant; the contacts row would otherwise be
// left with zero categories). Duplicate codes in the input are de-duplicated so a
// repeated code never violates the link table's composite primary key.
func (s CategoryStore) Save(ctx context.Context, q execQuerier, contactID string, codes []string) error {
	clean := dedupeNonBlank(codes)
	if len(clean) == 0 {
		return ErrEmptyCategorySet
	}

	if _, err := q.Exec(ctx,
		`DELETE FROM `+s.schema+`.contact_category_links WHERE contact_id = $1`,
		contactID); err != nil {
		return fmt.Errorf("save contact categories: delete: %w", err)
	}
	for _, code := range clean {
		if _, err := q.Exec(ctx,
			`INSERT INTO `+s.schema+`.contact_category_links (contact_id, category_code) VALUES ($1, $2)`,
			contactID, code); err != nil {
			return fmt.Errorf("save contact categories: insert %s: %w", code, err)
		}
	}
	return nil
}

// Add idempotently links one category code to a contact. ON CONFLICT DO NOTHING
// makes a repeat add a no-op (the detail-page selector may re-send a code). Pass
// a pgx.Tx to keep the add in a larger transaction.
func (s CategoryStore) Add(ctx context.Context, q execQuerier, contactID, code string) error {
	if _, err := q.Exec(ctx,
		`INSERT INTO `+s.schema+`.contact_category_links (contact_id, category_code)
		 VALUES ($1, $2) ON CONFLICT (contact_id, category_code) DO NOTHING`,
		contactID, code); err != nil {
		return fmt.Errorf("add contact category %s: %w", code, err)
	}
	return nil
}

// Remove unlinks one category code from a contact, enforcing the min-one
// invariant: if code is the LAST remaining link it is NOT removed and
// ErrEmptyCategorySet is returned (the HTTP layer maps this to 409). The count
// and delete run on the SAME execQuerier, so passing a pgx.Tx makes the
// check-then-delete atomic against concurrent removes.
func (s CategoryStore) Remove(ctx context.Context, q execQuerier, contactID, code string) error {
	// Atomic guard: delete the row ONLY when the contact has more than one link.
	// A single DELETE … WHERE (SELECT COUNT) > 1 avoids a check-then-act race when
	// q is a transaction. RowsAffected()==0 then means either the code was absent
	// or it was the last remaining one — Remove treats both as the min-one block,
	// the same outcome the caller needs (the contact still has ≥1 category).
	tag, err := q.Exec(ctx,
		`DELETE FROM `+s.schema+`.contact_category_links
		  WHERE contact_id = $1 AND category_code = $2
		    AND (SELECT COUNT(*) FROM `+s.schema+`.contact_category_links WHERE contact_id = $1) > 1`,
		contactID, code)
	if err != nil {
		return fmt.Errorf("remove contact category %s: %w", code, err)
	}
	if tag.RowsAffected() == 0 {
		return ErrEmptyCategorySet
	}
	return nil
}

// CategoryFilterClause compiles the directory's `?category=<code>` filter to an
// EXISTS subquery against the link table — a contact matches if ANY of its codes
// equals the bound parameter. alias is the contacts-row alias already in scope in
// the outer query (e.g. "c"); param is the bound placeholder for the wanted code
// (e.g. "$1"). The schema is the store's schema. Returned as a string so callers
// splice it into their WHERE builder exactly like Property's clause helpers.
//
//	clause := store.CategoryFilterClause("c", "$1")
//	// → EXISTS (SELECT 1 FROM <schema>.contact_category_links l
//	//           WHERE l.contact_id = c.id AND l.category_code = $1)
func (s CategoryStore) CategoryFilterClause(alias, param string) string {
	return fmt.Sprintf(
		"EXISTS (SELECT 1 FROM %s.contact_category_links l WHERE l.contact_id = %s.id AND l.category_code = %s)",
		s.schema, alias, param,
	)
}

// dedupeNonBlank trims, drops blanks, and de-duplicates codes preserving first-
// seen order. Used by Save so the insert set never collides on the link table's
// composite PK and never persists a blank code.
func dedupeNonBlank(codes []string) []string {
	seen := make(map[string]struct{}, len(codes))
	out := make([]string, 0, len(codes))
	for _, code := range codes {
		c := strings.TrimSpace(code)
		if c == "" {
			continue
		}
		if _, dup := seen[c]; dup {
			continue
		}
		seen[c] = struct{}{}
		out = append(out, c)
	}
	return out
}
