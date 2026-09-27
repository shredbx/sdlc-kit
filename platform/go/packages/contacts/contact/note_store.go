package contact

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// queryQuerier is the row-set query surface NoteStore.Load needs; rowQuerier is the
// single-row surface NoteStore.Add needs for its INSERT … RETURNING. Both
// *pgxpool.Pool and pgx.Tx satisfy them, so Load runs on the pool and Add accepts
// either a pool or a tx. Returned-value types (pgx.Rows / pgx.Row) keep the
// interfaces aligned with the concrete pgx signatures.
type queryQuerier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

type rowQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// Compile-time checks that the two concrete pgx types satisfy the note query
// surfaces (execQuerier is asserted in category_store.go).
var (
	_ pgxQuerier = (*pgxpool.Pool)(nil)
	_ rowQuerier = (pgx.Tx)(nil)
)

// =============================================================================
// NOTE STORE — companion for the contact_notes child table
// =============================================================================
//
// A contact's notes are an ordered child collection (contact_notes), NOT a column
// on the contacts row, so the generated mapper cannot map it (codegen has no child-
// collection support — the same constraint that drives the CategoryStore companion
// for the contact_category_links join table, and Property.Amenities). This hand-
// written store is that companion: one batch read for detail/list pages (no N+1),
// an append (Add), and a min-one-guarded delete (Delete).
//
// Notes are append + delete ONLY — there is no edit and no bulk Save. The set is
// always read ORDERED BY created_at ASCENDING (then id as a stable tie-breaker for
// notes created in the same instant), so the newest note is last — callers append
// at the bottom.

// ErrLastNote is returned by NoteStore.Delete when the caller tries to delete the
// LAST remaining note on a contact. It encodes the min-one-on-delete invariant at
// the store boundary so the HTTP layer can map it to 409 Conflict. A contact that
// has notes keeps ≥1; a fresh contact with zero notes is unaffected (this rule
// fires only on the 1→0 transition). Test with errors.Is(err, contact.ErrLastNote).
var ErrLastNote = errors.New("the last remaining note cannot be deleted (min-one invariant)")

// ErrEmptyNote is returned by NoteStore.Add when the supplied body is blank
// (empty or whitespace-only). A note must carry text. Test with
// errors.Is(err, contact.ErrEmptyNote).
var ErrEmptyNote = errors.New("note body must not be empty")

// NoteStore reads and writes a contact's note collection against the
// {schema}.contact_notes table. Construct with NewNoteStore. It shares the
// execQuerier surface (defined in category_store.go) so Add/Delete accept EITHER a
// *pgxpool.Pool or a pgx.Tx — the handler passes a pgx.Tx to make the delete's
// count-then-delete atomic; tests may pass the pool directly.
type NoteStore struct {
	pool   pgxQuerier
	schema string
}

// pgxQuerier is the minimal read+command surface NoteStore needs from the pool
// (Query for batch Load, QueryRow for the Add RETURNING). *pgxpool.Pool satisfies
// it. Kept distinct from execQuerier (command-only) so Load can run on the pool
// while Add/Delete accept either pool or tx via execQuerier.
type pgxQuerier interface {
	queryQuerier
	rowQuerier
}

// NewNoteStore wires the store to a pool + schema. Pass a nil pool only in tests
// that exercise pure helpers without a DB.
func NewNoteStore(pool pgxQuerier, schema string) NoteStore {
	return NoteStore{pool: pool, schema: schema}
}

// Load fetches the notes for one or more contacts in a SINGLE query (no N+1),
// returning a map of contactID → []Note. Each contact's notes are ordered by
// created_at ascending (then id), so the newest note is last. A contact with no
// notes is simply absent from the map (callers treat a missing key as "no notes
// yet"). An empty id list yields an empty map.
func (s NoteStore) Load(ctx context.Context, contactIDs ...string) (map[string][]Note, error) {
	out := make(map[string][]Note, len(contactIDs))
	if s.pool == nil || len(contactIDs) == 0 {
		return out, nil
	}

	rows, err := s.pool.Query(ctx,
		`SELECT id, contact_id, body, created_at
		   FROM `+s.schema+`.contact_notes
		  WHERE contact_id = ANY($1)
		  ORDER BY contact_id, created_at, id`,
		contactIDs)
	if err != nil {
		return nil, fmt.Errorf("load contact notes: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var n Note
		if err := rows.Scan(&n.ID, &n.ContactID, &n.Body, &n.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan contact note: %w", err)
		}
		out[n.ContactID] = append(out[n.ContactID], n)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("load contact notes: rows: %w", err)
	}
	return out, nil
}

// Add appends one note to a contact and returns the created Note (with its
// server-assigned id + created_at), so the handler can echo it in the 201 body.
// REJECTS a blank body with ErrEmptyNote (the HTTP layer maps it to 400). Pass a
// pgx.Tx to keep the append in a larger transaction. The body is trimmed before
// insert.
func (s NoteStore) Add(ctx context.Context, q rowQuerier, contactID, body string) (Note, error) {
	clean := strings.TrimSpace(body)
	if clean == "" {
		return Note{}, ErrEmptyNote
	}

	var n Note
	err := q.QueryRow(ctx,
		`INSERT INTO `+s.schema+`.contact_notes (contact_id, body)
		 VALUES ($1, $2)
		 RETURNING id, contact_id, body, created_at`,
		contactID, clean).Scan(&n.ID, &n.ContactID, &n.Body, &n.CreatedAt)
	if err != nil {
		return Note{}, fmt.Errorf("add contact note: %w", err)
	}
	return n, nil
}

// Delete removes one note from a contact, enforcing the min-one-on-delete
// invariant: if noteID is the contact's LAST remaining note it is NOT removed and
// ErrLastNote is returned (the HTTP layer maps this to 409). The count and delete
// run on the SAME execQuerier, so passing a pgx.Tx makes the check-then-delete
// atomic against concurrent deletes.
func (s NoteStore) Delete(ctx context.Context, q execQuerier, contactID, noteID string) error {
	// Atomic guard: delete the row ONLY when the contact has more than one note.
	// A single DELETE … WHERE (SELECT COUNT) > 1 avoids a check-then-act race when
	// q is a transaction. RowsAffected()==0 then means either the note was absent
	// (wrong id / wrong contact) or it was the last remaining one — Delete treats
	// both as the min-one block (the contact still has ≥1 note, the outcome the
	// caller needs).
	tag, err := q.Exec(ctx,
		`DELETE FROM `+s.schema+`.contact_notes
		  WHERE id = $1 AND contact_id = $2
		    AND (SELECT COUNT(*) FROM `+s.schema+`.contact_notes WHERE contact_id = $2) > 1`,
		noteID, contactID)
	if err != nil {
		return fmt.Errorf("delete contact note %s: %w", noteID, err)
	}
	if tag.RowsAffected() == 0 {
		return ErrLastNote
	}
	return nil
}
