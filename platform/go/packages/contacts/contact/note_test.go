package contact_test

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/shredbx/sbx-core/pkg/contact"
)

// Notes are an ordered child collection on Contact (created_at ascending — newest
// at the bottom). This locks the field shape + ordering assumption the NoteStore /
// handler rely on: the slice order IS the display order, oldest first.
func TestContact_Notes_OrderedOldestFirst(t *testing.T) {
	t0 := time.Date(2026, 6, 6, 9, 0, 0, 0, time.UTC)
	c := contact.NewContact("Somchai", "Jaidee", "landlord")
	c.Notes = []contact.Note{
		{ID: "n1", ContactID: c.ID, Body: "first", CreatedAt: t0},
		{ID: "n2", ContactID: c.ID, Body: "second", CreatedAt: t0.Add(time.Hour)},
	}
	if len(c.Notes) != 2 {
		t.Fatalf("want 2 notes, got %d", len(c.Notes))
	}
	if c.Notes[0].Body != "first" || c.Notes[1].Body != "second" {
		t.Errorf("notes must preserve oldest-first order, got %q then %q", c.Notes[0].Body, c.Notes[1].Body)
	}
	if !c.Notes[0].CreatedAt.Before(c.Notes[1].CreatedAt) {
		t.Error("note[0].CreatedAt must precede note[1].CreatedAt (ascending order)")
	}
}

// A fresh contact legitimately has ZERO notes — the min-one rule applies only to
// delete (1→0), never forcing ≥1 at rest. A contact with name only stays IsZero=false
// regardless of notes; an empty note slice does not by itself make a contact non-zero.
func TestContact_Notes_ZeroIsLegitimate(t *testing.T) {
	c := contact.NewContact("Somchai", "Jaidee", "landlord")
	if len(c.Notes) != 0 {
		t.Errorf("a freshly constructed contact must have zero notes, got %d", len(c.Notes))
	}
	// IsZero must not be tripped by a nil notes slice.
	if c.IsZero() {
		t.Error("a named contact with no notes should NOT be IsZero")
	}
}

// IsZero counts notes via len() — a Contact carrying ONLY a note (no other data) is
// not IsZero, proving the notes slice participates in the empties check.
func TestContact_IsZero_WithNoteOnly(t *testing.T) {
	c := contact.Contact{Notes: []contact.Note{{ID: "n1", Body: "x"}}}
	if c.IsZero() {
		t.Error("a Contact carrying a note should NOT be IsZero")
	}
}

// The Note wire shape carries id/contact_id/body/created_at — the exact fields the
// AddNote 201 body and the Get response notes array serialize.
func TestNote_JSONShape(t *testing.T) {
	n := contact.Note{
		ID:        "11111111-1111-1111-1111-111111111111",
		ContactID: "22222222-2222-2222-2222-222222222222",
		Body:      "Called, left voicemail",
		CreatedAt: time.Date(2026, 6, 6, 9, 0, 0, 0, time.UTC),
	}
	raw, err := json.Marshal(n)
	if err != nil {
		t.Fatalf("marshal note: %v", err)
	}
	var back map[string]any
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatalf("unmarshal note: %v", err)
	}
	for _, key := range []string{"id", "contact_id", "body", "created_at"} {
		if _, ok := back[key]; !ok {
			t.Errorf("note JSON missing %q key; got %s", key, raw)
		}
	}
	if back["body"] != "Called, left voicemail" {
		t.Errorf("note body round-trip mismatch: %v", back["body"])
	}
}

// The two note sentinels are distinct from each other and from the category set
// sentinel, so errors.Is at the HTTP boundary maps each to the right status
// (ErrEmptyNote → 400, ErrLastNote → 409) without cross-matching.
func TestNote_SentinelsDistinct(t *testing.T) {
	if errors.Is(contact.ErrLastNote, contact.ErrEmptyNote) {
		t.Error("ErrLastNote must not match ErrEmptyNote")
	}
	if errors.Is(contact.ErrLastNote, contact.ErrEmptyCategorySet) {
		t.Error("ErrLastNote must not match ErrEmptyCategorySet")
	}
	if errors.Is(contact.ErrEmptyNote, contact.ErrEmptyCategorySet) {
		t.Error("ErrEmptyNote must not match ErrEmptyCategorySet")
	}
}
