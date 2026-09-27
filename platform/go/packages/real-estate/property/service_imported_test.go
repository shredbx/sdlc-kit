package property_test

import (
	"context"
	"testing"
	"time"

	"github.com/shredbx/sbx-core/pkg/property"
)

// CreateImported keeps the caller-supplied ID and legacy timestamps (the bulk
// re-import contract, task 2607-001) and PRESERVES the original published state
// (100% identity) — an originally-published listing imports published, a draft
// imports as a draft.
func TestCreateImported_PreservesIDTimestampsAndPublished(t *testing.T) {
	svc := property.NewPropertyService(nil, nil, "bestierealestate") // fixture mode (nil repo)
	title := "Imported Villa"
	legacy := time.Date(2026, 4, 20, 13, 12, 56, 0, time.UTC)

	// Originally-published source → stays published.
	out, err := svc.CreateImported(context.Background(), property.Property{
		ID:          "0ae9d1f7-82fe-4426-80c3-19b47f0beeab",
		Title:       &title,
		ForSale:     true,
		IsPublished: true,
		CreatedAt:   legacy,
		UpdatedAt:   legacy,
	})
	if err != nil {
		t.Fatalf("CreateImported: %v", err)
	}
	if out.ID != "0ae9d1f7-82fe-4426-80c3-19b47f0beeab" {
		t.Errorf("expected id preserved, got %q", out.ID)
	}
	if !out.CreatedAt.Equal(legacy) {
		t.Errorf("expected created_at preserved %v, got %v", legacy, out.CreatedAt)
	}
	if !out.IsPublished {
		t.Error("expected the original published state (true) to be preserved")
	}

	// Originally-draft source → stays draft.
	draft, err := svc.CreateImported(context.Background(), property.Property{
		ID:          "11111111-1111-1111-1111-111111111111",
		Title:       &title,
		ForLease:    true,
		IsPublished: false,
	})
	if err != nil {
		t.Fatalf("CreateImported (draft): %v", err)
	}
	if draft.IsPublished {
		t.Error("expected the original draft state (false) to be preserved")
	}
}

// CreateImported requires a non-empty ID (unlike Create, which mints one).
func TestCreateImported_RequiresID(t *testing.T) {
	svc := property.NewPropertyService(nil, nil, "bestierealestate")
	title := "No ID"
	if _, err := svc.CreateImported(context.Background(), property.Property{
		Title:   &title,
		ForSale: true,
	}); err == nil {
		t.Fatal("expected an error for an empty id, got nil")
	}
}
