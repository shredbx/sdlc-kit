package calendar_test

import (
	"context"
	"testing"

	"github.com/shredbx/sbx-core/pkg/calendar"
)

// The companion stores (ReminderStore + ReferenceStore) mirror the contact
// note/category companions: a batch Load (no N+1), an Add, and a Remove against the
// child tables. The DB-backed write paths are exercised by the api-chi integration
// tests; here we pin the nil-pool-safe Load contract (a store wired with a nil pool
// returns an empty map, never panics) — the same guard the contact stores carry so a
// DB-less unit context can construct them.

func TestReminderStore_Load_NilPoolSafe(t *testing.T) {
	s := calendar.NewReminderStore(nil, "bestierealestate")
	got, err := s.Load(context.Background(), "11111111-1111-1111-1111-111111111111")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("nil-pool Load: want empty map, got %v", got)
	}
}

func TestReminderStore_Load_EmptyIDs(t *testing.T) {
	s := calendar.NewReminderStore(nil, "bestierealestate")
	got, err := s.Load(context.Background())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("empty-ids Load: want empty map, got %v", got)
	}
}

func TestReferenceStore_Load_NilPoolSafe(t *testing.T) {
	s := calendar.NewReferenceStore(nil, "bestierealestate")
	got, err := s.Load(context.Background(), "11111111-1111-1111-1111-111111111111")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("nil-pool Load: want empty map, got %v", got)
	}
}

func TestReferenceStore_Load_EmptyIDs(t *testing.T) {
	s := calendar.NewReferenceStore(nil, "bestierealestate")
	got, err := s.Load(context.Background())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("empty-ids Load: want empty map, got %v", got)
	}
}
