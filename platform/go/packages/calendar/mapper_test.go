package calendar_test

import (
	"testing"
	"time"

	"github.com/shredbx/sbx-core/pkg/calendar"
	"github.com/shredbx/sbx-core/pkg/repository/postgres"
)

// TestPostgresMapper_ImplementsInterface verifies the compile-time constraint that
// the hand-written Event mapper satisfies the generic postgres.Mapper[Event] surface
// — so it drops straight into postgres.NewPostgresStore[calendar.Event].
func TestPostgresMapper_ImplementsInterface(t *testing.T) {
	var _ postgres.Mapper[calendar.Event] = calendar.NewPostgresMapper("bestierealestate")
}

func TestPostgresMapper_TableName(t *testing.T) {
	m := calendar.NewPostgresMapper("bestierealestate")
	if got := m.TableName(); got != "bestierealestate.events" {
		t.Errorf("want 'bestierealestate.events', got %q", got)
	}
}

func TestPostgresMapper_TableName_DifferentSchema(t *testing.T) {
	m := calendar.NewPostgresMapper("bestays")
	if got := m.TableName(); got != "bestays.events" {
		t.Errorf("want 'bestays.events', got %q", got)
	}
}

// TestPostgresMapper_Columns_Count pins the flat-column SELECT list. Twelve columns:
// id, start_at, end_at, all_day, title, notes, event_type, status, owner_id,
// visibility, created_at, updated_at, deleted_at = 13.
func TestPostgresMapper_Columns_Count(t *testing.T) {
	m := calendar.NewPostgresMapper("bestierealestate")
	cols := m.Columns()
	if len(cols) != 13 {
		t.Errorf("want 13 columns, got %d: %v", len(cols), cols)
	}
}

// TestPostgresMapper_ToRow_FieldCoverage asserts every persisted attribute lands in
// the INSERT/UPDATE row map with its column name + coerced value. deleted_at is NOT
// in ToRow (DB default NULL; soft-delete is a separate UPDATE), mirroring the inquiry
// mapper.
func TestPostgresMapper_ToRow_FieldCoverage(t *testing.T) {
	m := calendar.NewPostgresMapper("bestierealestate")
	start := time.Date(2026, 6, 12, 10, 0, 0, 0, time.UTC)
	e := calendar.Event{
		ID:         "11111111-1111-1111-1111-111111111111",
		Start:      start,
		End:        start.Add(time.Hour),
		AllDay:     false,
		Title:      "Viewing — Unit A2",
		Notes:      "Bring the keys",
		EventType:  calendar.EventTypeViewing,
		Status:     calendar.StatusScheduled,
		OwnerID:    "22222222-2222-2222-2222-222222222222",
		Visibility: calendar.VisibilityTeam,
		CreatedAt:  start,
		UpdatedAt:  start,
	}
	row, err := m.ToRow(e)
	if err != nil {
		t.Fatalf("ToRow: %v", err)
	}

	want := map[string]any{
		"id":         "11111111-1111-1111-1111-111111111111",
		"start_at":   start,
		"end_at":     start.Add(time.Hour),
		"all_day":    false,
		"title":      "Viewing — Unit A2",
		"notes":      "Bring the keys",
		"event_type": "viewing",
		"status":     "scheduled",
		"owner_id":   "22222222-2222-2222-2222-222222222222",
		"visibility": "team",
	}
	for k, v := range want {
		got, ok := row[k]
		if !ok {
			t.Errorf("ToRow missing column %q", k)
			continue
		}
		if got != v {
			t.Errorf("ToRow[%q]: want %v, got %v", k, v, got)
		}
	}
	// Discriminators must be the STRING form (not the named type) so pgx binds a
	// VARCHAR, never a custom type.
	if _, isStr := row["event_type"].(string); !isStr {
		t.Errorf("event_type must serialize to string, got %T", row["event_type"])
	}
	if _, isStr := row["status"].(string); !isStr {
		t.Errorf("status must serialize to string, got %T", row["status"])
	}
	if _, isStr := row["visibility"].(string); !isStr {
		t.Errorf("visibility must serialize to string, got %T", row["visibility"])
	}
	// deleted_at is never written by ToRow.
	if _, present := row["deleted_at"]; present {
		t.Error("ToRow must NOT include deleted_at (DB default NULL)")
	}
}

// TestPostgresMapper_FieldColumn_Aliases verifies the logical→column mapping the
// time-window + owner + visibility filters depend on. The Go field "start"/"end"
// map to the start_at/end_at columns (end is a SQL keyword).
func TestPostgresMapper_FieldColumn_Aliases(t *testing.T) {
	m := calendar.NewPostgresMapper("bestierealestate")
	cases := map[string]string{
		"id":         "e.id",
		"start":      "e.start_at",
		"end":        "e.end_at",
		"owner_id":   "e.owner_id",
		"visibility": "e.visibility",
		"status":     "e.status",
		"event_type": "e.event_type",
		"deleted_at": "e.deleted_at",
	}
	for field, want := range cases {
		if got := m.FieldColumn(field); got != want {
			t.Errorf("FieldColumn(%q): want %q, got %q", field, want, got)
		}
	}
}

// TestFields_LogicalNames pins the typed Field descriptors the HTTP adapter uses to
// build predicates (start/end range, owner_id eq, visibility eq). The Name on each
// must be the logical key FieldColumn understands.
func TestFields_LogicalNames(t *testing.T) {
	if calendar.Fields.Start.Name != "start" {
		t.Errorf("Fields.Start.Name want 'start', got %q", calendar.Fields.Start.Name)
	}
	if calendar.Fields.End.Name != "end" {
		t.Errorf("Fields.End.Name want 'end', got %q", calendar.Fields.End.Name)
	}
	if calendar.Fields.OwnerID.Name != "owner_id" {
		t.Errorf("Fields.OwnerID.Name want 'owner_id', got %q", calendar.Fields.OwnerID.Name)
	}
	if calendar.Fields.Visibility.Name != "visibility" {
		t.Errorf("Fields.Visibility.Name want 'visibility', got %q", calendar.Fields.Visibility.Name)
	}
}
