package dictionary_test

// Reorder unit tests — validation paths that do NOT require a live DB.
// Happy-path + DB-touching scenarios live in the BR integration test suite
// where seeded property_furnished data exists (see test-red-result.yml).

import (
	"context"
	"errors"
	"testing"

	"github.com/shredbx/sbx-core/pkg/dictionary"
)

// TestReorder_EmptyCodes verifies that an empty codes array returns
// ErrInvalidPayload without touching the database. Validation runs before
// any tx is begun.
func TestReorder_EmptyCodes(t *testing.T) {
	b := dictionary.NewPostgresBackend(nil, "test")

	err := b.Reorder(context.Background(), "property_furnished", []string{})
	if err == nil {
		t.Fatal("expected ErrInvalidPayload for empty codes, got nil")
	}
	if !errors.Is(err, dictionary.ErrInvalidPayload) {
		t.Errorf("expected ErrInvalidPayload, got: %v", err)
	}
}

// TestReorder_DuplicateCodes verifies that duplicate codes are rejected
// before any DB operation occurs.
func TestReorder_DuplicateCodes(t *testing.T) {
	b := dictionary.NewPostgresBackend(nil, "test")

	err := b.Reorder(context.Background(), "property_furnished", []string{"fully", "fully", "partially"})
	if err == nil {
		t.Fatal("expected ErrInvalidPayload for duplicate codes, got nil")
	}
	if !errors.Is(err, dictionary.ErrInvalidPayload) {
		t.Errorf("expected ErrInvalidPayload, got: %v", err)
	}
}

// TestReorder_HappyPath documents the happy-path contract. The full DB
// roundtrip is covered by BR integration tests; this test verifies the
// validation prelude does NOT reject a well-formed payload.
//
// Skipped: requires DB pool. Will run in BR's integration suite where
// property_furnished is seeded (FX-DICT-FURNISHED-3 fixture).
func TestReorder_HappyPath(t *testing.T) {
	t.Skip("DB-dependent: covered by BR integration test in clients/bestie/.../api-chi tests")
}

// TestReorder_UnknownCode — DB-dependent (requires existing dict). Skipped
// here, covered by BR integration test.
func TestReorder_UnknownCode(t *testing.T) {
	t.Skip("DB-dependent: covered by BR integration test")
}

// TestReorder_MissingCode — DB-dependent. Skipped here, covered by BR
// integration test.
func TestReorder_MissingCode(t *testing.T) {
	t.Skip("DB-dependent: covered by BR integration test")
}
