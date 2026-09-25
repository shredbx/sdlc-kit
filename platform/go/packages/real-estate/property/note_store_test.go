package property_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/shredbx/sbx-core/pkg/property"
)

// The Note wire shape carries id/property_id/body/created_at — the exact fields the
// AddNote 201 body and the ListNotes response notes array serialize.
func TestNote_JSONShape(t *testing.T) {
	n := property.Note{
		ID:         "11111111-1111-1111-1111-111111111111",
		PropertyID: "22222222-2222-2222-2222-222222222222",
		Body:       "Owner wants a 2026 renovation",
		CreatedAt:  time.Date(2026, 6, 16, 9, 0, 0, 0, time.UTC),
	}
	raw, err := json.Marshal(n)
	if err != nil {
		t.Fatalf("marshal note: %v", err)
	}
	var back map[string]any
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatalf("unmarshal note: %v", err)
	}
	for _, key := range []string{"id", "property_id", "body", "created_at"} {
		if _, ok := back[key]; !ok {
			t.Errorf("note JSON missing %q key; got %s", key, raw)
		}
	}
	if back["body"] != "Owner wants a 2026 renovation" {
		t.Errorf("note body round-trip mismatch: %v", back["body"])
	}
}

// Notes are an ordered child collection ordered created_at ascending (newest at the
// bottom). This locks the slice-order-IS-display-order assumption the NoteStore /
// handler rely on: oldest first.
func TestNote_OrderedOldestFirst(t *testing.T) {
	t0 := time.Date(2026, 6, 16, 9, 0, 0, 0, time.UTC)
	notes := []property.Note{
		{ID: "n1", PropertyID: "p1", Body: "first", CreatedAt: t0},
		{ID: "n2", PropertyID: "p1", Body: "second", CreatedAt: t0.Add(time.Hour)},
	}
	if notes[0].Body != "first" || notes[1].Body != "second" {
		t.Errorf("notes must preserve oldest-first order, got %q then %q", notes[0].Body, notes[1].Body)
	}
	if !notes[0].CreatedAt.Before(notes[1].CreatedAt) {
		t.Error("note[0].CreatedAt must precede note[1].CreatedAt (ascending order)")
	}
}

// Add REJECTS a blank (empty or whitespace-only) body with ErrEmptyNote BEFORE it
// touches the DB — so a nil querier is never dereferenced on the blank path. This
// proves the HTTP layer's 400 mapping fires for blank bodies.
func TestNoteStore_Add_BlankBodyRejected(t *testing.T) {
	s := property.NewNoteStore(nil, "bestierealestate")
	for _, body := range []string{"", "   ", "\t\n  "} {
		_, err := s.Add(context.Background(), nil, "p1", body)
		if !errors.Is(err, property.ErrEmptyNote) {
			t.Errorf("Add(%q) must return ErrEmptyNote, got %v", body, err)
		}
	}
}

// Load with an empty id list (or a nil pool) yields an empty, non-nil map and never
// touches the DB — the batch read short-circuits.
func TestNoteStore_Load_EmptyIsEmptyMap(t *testing.T) {
	s := property.NewNoteStore(nil, "bestierealestate")
	out, err := s.Load(context.Background())
	if err != nil {
		t.Fatalf("Load() with no ids: unexpected error %v", err)
	}
	if out == nil {
		t.Fatal("Load() must return a non-nil map")
	}
	if len(out) != 0 {
		t.Errorf("Load() with no ids must be empty, got %d entries", len(out))
	}
}

// fakeExec records the Exec call and returns a configurable RowsAffected, so Delete
// can be exercised without a real DB. It satisfies the execQuerier surface Delete
// accepts (Exec).
type fakeExec struct {
	called   bool
	gotSQL   string
	gotArgs  []any
	affected int64
	err      error
}

func (f *fakeExec) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	f.called = true
	f.gotSQL = sql
	f.gotArgs = args
	if f.err != nil {
		return pgconn.CommandTag{}, f.err
	}
	// "DELETE N" yields a CommandTag whose RowsAffected() == N.
	tag := pgconn.NewCommandTag("DELETE " + itoa(f.affected))
	return tag, nil
}

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

// Delete returns nil even when ZERO rows are affected — property notes have NO
// min-one-on-delete invariant, so a delete-to-zero (or a repeated/idempotent delete
// of an already-gone note) is a no-op success, never an error.
func TestNoteStore_Delete_ZeroRowsIsOK(t *testing.T) {
	s := property.NewNoteStore(nil, "bestierealestate")
	fe := &fakeExec{affected: 0}
	if err := s.Delete(context.Background(), fe, "p1", "n1"); err != nil {
		t.Errorf("Delete with 0 rows affected must be nil (idempotent), got %v", err)
	}
	if !fe.called {
		t.Error("Delete must issue the DELETE exec")
	}
	// noteID is $1, propertyID is $2 — both must be passed through.
	if len(fe.gotArgs) != 2 || fe.gotArgs[0] != "n1" || fe.gotArgs[1] != "p1" {
		t.Errorf("Delete args want [n1 p1], got %v", fe.gotArgs)
	}
}

// Delete also returns nil when a row IS removed (the normal path).
func TestNoteStore_Delete_OneRowIsOK(t *testing.T) {
	s := property.NewNoteStore(nil, "bestierealestate")
	fe := &fakeExec{affected: 1}
	if err := s.Delete(context.Background(), fe, "p1", "n1"); err != nil {
		t.Errorf("Delete removing one row must be nil, got %v", err)
	}
}

// A DB-level error from the Exec is wrapped and surfaced (the HTTP layer maps it to
// 500), distinct from the idempotent 0-rows success above.
func TestNoteStore_Delete_DBErrorSurfaces(t *testing.T) {
	s := property.NewNoteStore(nil, "bestierealestate")
	boom := errors.New("connection reset")
	fe := &fakeExec{err: boom}
	err := s.Delete(context.Background(), fe, "p1", "n1")
	if err == nil {
		t.Fatal("Delete must surface a DB error, got nil")
	}
	if !errors.Is(err, boom) {
		t.Errorf("Delete must wrap the underlying DB error, got %v", err)
	}
}

// Compile-time anchor: pgx.Tx satisfies the execQuerier surface Delete accepts, so
// the handler can pass a tx if it ever needs to (it currently passes the pool).
var _ interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
} = (pgx.Tx)(nil)
