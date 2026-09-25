// mapper_test.go exercises the MD layer of pkg/faq (FDD4 red→green, SDLC 2607-001,
// test-plan MD unit >80%). Each PostgresMapper is proven by a ToRow → (replay-scan)
// FromRow round-trip: the replay scanner maps each Columns() name to its ToRow value
// (or zero for the ToRow-omitted deleted_at), proving the column ordering + value
// mapping are mutually consistent — NO database required (mirrors pkg/cms/entry_test.go).
package faq

import (
	"strings"
	"testing"
	"time"

	"github.com/shredbx/sbx-core/pkg/seo"
)

// =============================================================================
// CATEGORY MAPPER
// =============================================================================

func TestCategoryPostgresMapper_ColumnsMatchFromRow(t *testing.T) {
	m := NewCategoryPostgresMapper("bestierealestate")

	if got := m.TableName(); got != "bestierealestate.faq_categories" {
		t.Fatalf("TableName = %q, want bestierealestate.faq_categories", got)
	}
	if !strings.HasSuffix(m.SelectFrom(), " p") {
		t.Fatalf("SelectFrom must alias the table p: %q", m.SelectFrom())
	}
	for _, c := range m.Columns() {
		if !strings.HasPrefix(c, "p.") {
			t.Fatalf("every column must be p.-prefixed: %q", c)
		}
	}
	var n int
	if _, err := m.FromRow(func(dest ...any) error { n = len(dest); return nil }); err != nil {
		t.Fatalf("FromRow counting scan error: %v", err)
	}
	if n != len(m.Columns()) {
		t.Fatalf("Columns() count %d != FromRow dest count %d", len(m.Columns()), n)
	}
}

func TestCategoryPostgresMapper_RoundTrip(t *testing.T) {
	m := NewCategoryPostgresMapper("bestierealestate")
	createdBy := "admin@bestierealestate.com"
	cat := Category{
		ID:       "11111111-1111-1111-1111-111111111111",
		Slug:     Slug("buyers-guide"),
		Title:    "Properties Buyer Guide",
		Blurb:    "The honest 0→6 journey.",
		Position: 0,
		Featured: true,
		SEO:      seo.SeoMeta{MetaTitle: "Buyers Guide | Bestie"},
		Status:   StatusPublished,
		Version:  3,
		CreatedBy: &createdBy,
		CreatedAt: time.Date(2026, 7, 7, 12, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 7, 7, 12, 30, 0, 0, time.UTC),
	}
	out := roundTripCategory(t, m, cat)

	if out.ID != cat.ID || out.Slug != cat.Slug || out.Title != cat.Title || out.Blurb != cat.Blurb {
		t.Fatalf("core fields diverged: %+v vs %+v", out, cat)
	}
	if out.Position != cat.Position || out.Featured != cat.Featured || out.Status != cat.Status {
		t.Fatalf("flags/status diverged: %+v vs %+v", out, cat)
	}
	if out.SEO.MetaTitle != cat.SEO.MetaTitle {
		t.Fatalf("SEO diverged: %+v vs %+v", out.SEO, cat.SEO)
	}
	if out.Version != cat.Version {
		t.Fatalf("version diverged: %d vs %d", out.Version, cat.Version)
	}
	if out.CreatedBy == nil || *out.CreatedBy != *cat.CreatedBy {
		t.Fatalf("created_by diverged: %v vs %v", out.CreatedBy, cat.CreatedBy)
	}
	if !out.CreatedAt.Equal(cat.CreatedAt) || !out.UpdatedAt.Equal(cat.UpdatedAt) {
		t.Fatalf("timestamps diverged: %+v vs %+v", out, cat)
	}
	if out.DeletedAt != nil {
		t.Fatalf("DeletedAt must be nil (ToRow omits it): %v", out.DeletedAt)
	}
}

// =============================================================================
// STEP MAPPER
// =============================================================================

func TestStepPostgresMapper_ColumnsMatchFromRow(t *testing.T) {
	m := NewStepPostgresMapper("bestierealestate")
	if got := m.TableName(); got != "bestierealestate.faq_steps" {
		t.Fatalf("TableName = %q, want bestierealestate.faq_steps", got)
	}
	if !strings.HasSuffix(m.SelectFrom(), " p") {
		t.Fatalf("SelectFrom must alias p: %q", m.SelectFrom())
	}
	var n int
	if _, err := m.FromRow(func(dest ...any) error { n = len(dest); return nil }); err != nil {
		t.Fatalf("FromRow counting scan error: %v", err)
	}
	if n != len(m.Columns()) {
		t.Fatalf("Columns() count %d != FromRow dest count %d", len(m.Columns()), n)
	}
}

func TestStepPostgresMapper_RoundTrip(t *testing.T) {
	m := NewStepPostgresMapper("bestierealestate")
	step := Step{
		ID:          "22222222-2222-2222-2222-222222222222",
		CategoryID:  "11111111-1111-1111-1111-111111111111",
		Position:    1,
		Title:       "Can I Buy?",
		Description: "What foreigners may own.",
		Status:      StatusPublished,
		Version:     1,
	}
	out := roundTripStep(t, m, step)
	if out.ID != step.ID || out.CategoryID != step.CategoryID || out.Title != step.Title {
		t.Fatalf("core fields diverged: %+v vs %+v", out, step)
	}
	if out.Position != step.Position || out.Description != step.Description || out.Status != step.Status {
		t.Fatalf("fields diverged: %+v vs %+v", out, step)
	}
}

// =============================================================================
// ITEM MAPPER
// =============================================================================

func TestItemPostgresMapper_ColumnsMatchFromRow(t *testing.T) {
	m := NewItemPostgresMapper("bestierealestate")
	if got := m.TableName(); got != "bestierealestate.faq_items" {
		t.Fatalf("TableName = %q, want bestierealestate.faq_items", got)
	}
	if !strings.HasSuffix(m.SelectFrom(), " p") {
		t.Fatalf("SelectFrom must alias p: %q", m.SelectFrom())
	}
	var n int
	if _, err := m.FromRow(func(dest ...any) error { n = len(dest); return nil }); err != nil {
		t.Fatalf("FromRow counting scan error: %v", err)
	}
	if n != len(m.Columns()) {
		t.Fatalf("Columns() count %d != FromRow dest count %d", len(m.Columns()), n)
	}
}

func TestItemPostgresMapper_RoundTrip(t *testing.T) {
	m := NewItemPostgresMapper("bestierealestate")
	item := Item{
		ID:       "33333333-3333-3333-3333-333333333333",
		StepID:   "22222222-2222-2222-2222-222222222222",
		Position: 2,
		Question: "Can foreigners buy property in Thailand?",
		Answer:   "Yes, via a condominium.",
		Tip:      "Use an independent lawyer.",
		Status:   StatusPublished,
		Version:  1,
	}
	out := roundTripItem(t, m, item)
	if out.ID != item.ID || out.StepID != item.StepID || out.Question != item.Question {
		t.Fatalf("core fields diverged: %+v vs %+v", out, item)
	}
	if out.Position != item.Position || out.Answer != item.Answer || out.Tip != item.Tip || out.Status != item.Status {
		t.Fatalf("fields diverged: %+v vs %+v", out, item)
	}
}

// =============================================================================
// round-trip helpers — replay ToRow values through FromRow (mirrors entry_test.go).
// =============================================================================

func roundTripCategory(t *testing.T, m CategoryPostgresMapper, in Category) Category {
	t.Helper()
	row, err := m.ToRow(in)
	if err != nil {
		t.Fatalf("ToRow error: %v", err)
	}
	cols := m.Columns()
	out, err := m.FromRow(func(dest ...any) error {
		if len(dest) != len(cols) {
			t.Fatalf("dest %d != cols %d", len(dest), len(cols))
		}
		for i, c := range cols {
			assignScan(t, dest[i], row[strings.TrimPrefix(c, "p.")])
		}
		return nil
	})
	if err != nil {
		t.Fatalf("FromRow error: %v", err)
	}
	return out
}

func roundTripStep(t *testing.T, m StepPostgresMapper, in Step) Step {
	t.Helper()
	row, err := m.ToRow(in)
	if err != nil {
		t.Fatalf("ToRow error: %v", err)
	}
	cols := m.Columns()
	out, err := m.FromRow(func(dest ...any) error {
		for i, c := range cols {
			assignScan(t, dest[i], row[strings.TrimPrefix(c, "p.")])
		}
		return nil
	})
	if err != nil {
		t.Fatalf("FromRow error: %v", err)
	}
	return out
}

func roundTripItem(t *testing.T, m ItemPostgresMapper, in Item) Item {
	t.Helper()
	row, err := m.ToRow(in)
	if err != nil {
		t.Fatalf("ToRow error: %v", err)
	}
	cols := m.Columns()
	out, err := m.FromRow(func(dest ...any) error {
		for i, c := range cols {
			assignScan(t, dest[i], row[strings.TrimPrefix(c, "p.")])
		}
		return nil
	})
	if err != nil {
		t.Fatalf("FromRow error: %v", err)
	}
	return out
}

// assignScan emulates a sql driver assigning a stored value to a scan dest pointer
// for the faq named types (mirrors entry_test.go assignScan).
func assignScan(t *testing.T, dest, val any) {
	t.Helper()
	if val == nil {
		return // ToRow-omitted column (deleted_at) → leave dest at its zero value
	}
	switch d := dest.(type) {
	case *string:
		*d = val.(string)
	case *Slug:
		*d = Slug(val.(string))
	case *Status:
		*d = Status(val.(string))
	case *bool:
		*d = val.(bool)
	case *int:
		*d = val.(int)
	case *seo.SeoMeta:
		*d = val.(seo.SeoMeta)
	case **string:
		s := val.(*string)
		*d = s
	case *time.Time:
		*d = val.(time.Time)
	case **time.Time:
		*d = val.(*time.Time)
	default:
		t.Fatalf("assignScan: unhandled dest type %T", dest)
	}
}
