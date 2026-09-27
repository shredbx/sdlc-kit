package property

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// =============================================================================
// NOTE STORE — companion for the property_notes child table
// =============================================================================
//
// This is the property-side PARALLEL of pkg/contact.NoteStore. A property's notes
// are an ordered child collection (property_notes), NOT a column on the properties
// row, so the generated mapper cannot map it (codegen has no child-collection
// support). This hand-written store is that companion: one batch read for detail/
// list pages (no N+1), an append (Add), and a delete (Delete).
//
// Notes are append + delete ONLY — there is no edit and no bulk Save. The set is
// always read ORDERED BY created_at ASCENDING (then id as a stable tie-breaker for
// notes created in the same instant), so the newest note is last — callers append
// at the bottom.
//
// DELIBERATE DIFFERENCE FROM pkg/contact.NoteStore: property notes have NO
// min-one-on-delete invariant. A property may legitimately have ZERO notes, and a
// note may be deleted down to zero, so there is no ErrLastNote and no count-then-
// delete guard here — Delete is a plain idempotent DELETE.
//
// The two stores (contact + property) are kept SEPARATE on purpose: unifying them
// behind a future generic pkg/notes would require touching the contact area, which
// is owned by another worktree. When that consolidation is scheduled, this file and
// pkg/contact/note_store.go are the two call sites to fold together.

// Note is a single free-form note attached to a property. Notes form an ordered
// collection (by CreatedAt ascending — newest appended at the bottom), each one its
// own row in the property_notes table. They are append + delete only (no edit, no
// bulk save): a note is added via NoteStore.Add and removed via NoteStore.Delete,
// never mutated in place. The set is a normalized child collection (not a properties
// column), so it is absent from the generated mapper and loaded via NoteStore.
type Note struct {
	ID         string    `json:"id"`
	PropertyID string    `json:"property_id"`
	Body       string    `json:"body"`
	CreatedAt  time.Time `json:"created_at"`
}

// ErrEmptyNote is returned by NoteStore.Add when the supplied body is blank
// (empty or whitespace-only). A note must carry text. Test with
// errors.Is(err, property.ErrEmptyNote).
var ErrEmptyNote = errors.New("note body must not be empty")

// queryQuerier is the row-set query surface NoteStore.Load needs; rowQuerier is the
// single-row surface NoteStore.Add needs for its INSERT … RETURNING; execQuerier is
// the command surface NoteStore.Delete needs. Both *pgxpool.Pool and pgx.Tx satisfy
// them, so Load runs on the pool and Add/Delete accept either a pool or a tx.
type queryQuerier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

type rowQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type execQuerier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// pgxQuerier is the minimal read+command surface NoteStore needs from the pool
// (Query for batch Load, QueryRow for the Add RETURNING). *pgxpool.Pool satisfies
// it. Kept distinct from execQuerier (command-only) so Load can run on the pool
// while Add/Delete accept either pool or tx.
type pgxQuerier interface {
	queryQuerier
	rowQuerier
}

// Compile-time checks that the concrete pgx types satisfy the note query surfaces.
var (
	_ pgxQuerier  = (*pgxpool.Pool)(nil)
	_ rowQuerier  = (pgx.Tx)(nil)
	_ execQuerier = (*pgxpool.Pool)(nil)
	_ execQuerier = (pgx.Tx)(nil)
)

// NoteStore reads and writes a property's note collection against the
// {schema}.property_notes table. Construct with NewNoteStore.
type NoteStore struct {
	pool   pgxQuerier
	schema string
}

// NewNoteStore wires the store to a pool + schema. Pass a nil pool only in tests
// that exercise pure helpers without a DB.
func NewNoteStore(pool pgxQuerier, schema string) NoteStore {
	return NoteStore{pool: pool, schema: schema}
}

// Load fetches the notes for one or more properties in a SINGLE query (no N+1),
// returning a map of propertyID → []Note. Each property's notes are ordered by
// created_at ascending (then id), so the newest note is last. A property with no
// notes is simply absent from the map (callers treat a missing key as "no notes
// yet"). An empty id list yields an empty map.
func (s NoteStore) Load(ctx context.Context, propertyIDs ...string) (map[string][]Note, error) {
	out := make(map[string][]Note, len(propertyIDs))
	if s.pool == nil || len(propertyIDs) == 0 {
		return out, nil
	}

	rows, err := s.pool.Query(ctx,
		`SELECT id, property_id, body, created_at
		   FROM `+s.schema+`.property_notes
		  WHERE property_id = ANY($1)
		  ORDER BY property_id, created_at, id`,
		propertyIDs)
	if err != nil {
		return nil, fmt.Errorf("load property notes: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var n Note
		if err := rows.Scan(&n.ID, &n.PropertyID, &n.Body, &n.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan property note: %w", err)
		}
		out[n.PropertyID] = append(out[n.PropertyID], n)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("load property notes: rows: %w", err)
	}
	return out, nil
}

// Add appends one note to a property and returns the created Note (with its
// server-assigned id + created_at), so the handler can echo it in the 201 body.
// REJECTS a blank body with ErrEmptyNote (the HTTP layer maps it to 400). Pass a
// pgx.Tx to keep the append in a larger transaction. The body is trimmed before
// insert.
func (s NoteStore) Add(ctx context.Context, q rowQuerier, propertyID, body string) (Note, error) {
	clean := strings.TrimSpace(body)
	if clean == "" {
		return Note{}, ErrEmptyNote
	}

	var n Note
	err := q.QueryRow(ctx,
		`INSERT INTO `+s.schema+`.property_notes (property_id, body)
		 VALUES ($1, $2)
		 RETURNING id, property_id, body, created_at`,
		propertyID, clean).Scan(&n.ID, &n.PropertyID, &n.Body, &n.CreatedAt)
	if err != nil {
		return Note{}, fmt.Errorf("add property note: %w", err)
	}
	return n, nil
}

// Delete removes one note from a property. UNLIKE pkg/contact.NoteStore.Delete there
// is NO min-one-on-delete invariant: property notes may be deleted down to zero. The
// delete is a plain idempotent statement — RowsAffected()==0 (the note was absent /
// wrong id / wrong property) is NOT an error, so a repeated delete is a no-op that
// returns nil. The HTTP layer returns 204 regardless.
func (s NoteStore) Delete(ctx context.Context, q execQuerier, propertyID, noteID string) error {
	_, err := q.Exec(ctx,
		`DELETE FROM `+s.schema+`.property_notes
		  WHERE id = $1 AND property_id = $2`,
		noteID, propertyID)
	if err != nil {
		return fmt.Errorf("delete property note %s: %w", noteID, err)
	}
	return nil
}
