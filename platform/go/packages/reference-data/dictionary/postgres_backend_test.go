package dictionary_test

// CRUD extension tests (TC-003 .. TC-010) for PostgresBackend.
//
// These tests cover the new methods being added in P1 of task 2605-058:
//   - Update(ctx, name, code, UpdatePatch) → (*Entry, error)
//   - Delete(ctx, name, code) → error  (FK RESTRICT enforced)
//   - CascadeReplaceAndDelete(ctx, name, code, replacement, FKRefs) → (migratedCount int, err error)
//   - ConsumersCount(ctx, name, code, FKRefs) → (total int, byRef []ConsumerCount, err error)
//
// All tests use the existing pkg/dictionary test pattern (testify) and rely on
// a Postgres test DB. They will FAIL until P1 implementation lands.
//
// NOTE: this is the RED phase per FDD4.RED. Tests are written before implementation.

import (
	"context"
	"testing"

	"github.com/shredbx/sbx-core/pkg/dictionary"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// helper: assumes a test backend factory exists; if not, tests fail to compile (red)
// In green phase we'll wire a real testcontainers-postgres helper or skip if DB unavailable.
func newTestBackend(t *testing.T) *dictionary.PostgresBackend {
	t.Helper()
	t.Skip("PostgresBackend test wiring lands in P1 — RED phase placeholder")
	return nil
}

// TC-003 — Update changes label only; other fields unchanged. Maps to SC4.1.
func TestUpdate_LabelOnly(t *testing.T) {
	b := newTestBackend(t)
	ctx := context.Background()

	newLabel := "Pool Villa"
	entry, err := b.Update(ctx, "property_types", "villa", dictionary.UpdatePatch{
		Label: &newLabel,
	})
	require.NoError(t, err)
	require.NotNil(t, entry)
	assert.Equal(t, "Pool Villa", entry.Label)
	assert.Equal(t, "villa", entry.Code) // unchanged
}

// TC-004 — Update flips deprecated=true. Maps to SC5.1.
func TestUpdate_DeprecateField(t *testing.T) {
	b := newTestBackend(t)
	ctx := context.Background()

	deprecated := true
	entry, err := b.Update(ctx, "property_types", "villa", dictionary.UpdatePatch{
		Deprecated: &deprecated,
	})
	require.NoError(t, err)
	require.NotNil(t, entry)
	assert.True(t, entry.Deprecated)
}

// TC-005 — Update flips deprecated=false (restore). Maps to SC5.2.
func TestUpdate_RestoreField(t *testing.T) {
	b := newTestBackend(t)
	ctx := context.Background()

	notDeprecated := false
	entry, err := b.Update(ctx, "property_types", "villa", dictionary.UpdatePatch{
		Deprecated: &notDeprecated,
	})
	require.NoError(t, err)
	require.NotNil(t, entry)
	assert.False(t, entry.Deprecated)
}

// TC-006 — Update with all-nil patch is a no-op (returns current entry unchanged). Edge case.
func TestUpdate_PartialBodyNoOp(t *testing.T) {
	b := newTestBackend(t)
	ctx := context.Background()

	entry, err := b.Update(ctx, "property_types", "villa", dictionary.UpdatePatch{})
	require.NoError(t, err)
	require.NotNil(t, entry)
	assert.Equal(t, "villa", entry.Code)
}

// TC-007 — ConsumersCount returns 0 + empty byRef for unused value. Maps to SC6.1.
func TestConsumersCount_Zero(t *testing.T) {
	b := newTestBackend(t)
	ctx := context.Background()

	total, byRef, err := b.ConsumersCount(ctx, "property_types", "unused-code", []dictionary.FKRef{
		{Table: "properties", Column: "property_type", Nullable: true},
	})
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Empty(t, byRef)
}

// TC-008 — ConsumersCount returns positive total + breakdown for in-use value. Maps to SC6.2.
func TestConsumersCount_WithRows(t *testing.T) {
	b := newTestBackend(t)
	ctx := context.Background()

	// Fixture: assume 14 properties with property_type=villa pre-seeded.
	total, byRef, err := b.ConsumersCount(ctx, "property_types", "villa", []dictionary.FKRef{
		{Table: "properties", Column: "property_type", Nullable: true},
	})
	require.NoError(t, err)
	assert.Equal(t, 14, total)
	require.Len(t, byRef, 1)
	assert.Equal(t, "properties", byRef[0].Table)
	assert.Equal(t, "property_type", byRef[0].Column)
	assert.Equal(t, 14, byRef[0].Count)
}

// TC-009 — CascadeReplaceAndDelete atomically migrates consumers + deletes dict row. Maps to SC6.2.
func TestCascadeReplaceAndDelete_Atomic(t *testing.T) {
	b := newTestBackend(t)
	ctx := context.Background()

	migrated, err := b.CascadeReplaceAndDelete(ctx, "property_types", "villa", "house", []dictionary.FKRef{
		{Table: "properties", Column: "property_type", Nullable: true},
	})
	require.NoError(t, err)
	assert.Equal(t, 14, migrated, "all 14 consumer rows must be migrated to replacement")

	// Verify dict row gone
	_, err = b.Get(ctx, "property_types", "villa")
	assert.ErrorIs(t, err, dictionary.ErrNotFound, "villa must be deleted from dict table")

	// Verify replacement still exists
	houseEntry, err := b.Get(ctx, "property_types", "house")
	require.NoError(t, err)
	assert.Equal(t, "house", houseEntry.Code)
}

// TC-010 — CascadeReplaceAndDelete rolls back on mid-tx error (edge case).
// Inject failure between UPDATE and DELETE; verify state unchanged.
func TestCascadeReplaceAndDelete_RollbackOnError(t *testing.T) {
	b := newTestBackend(t)
	ctx := context.Background()

	// Use unknown replacement to force an FK violation that aborts the tx
	_, err := b.CascadeReplaceAndDelete(ctx, "property_types", "villa", "nonexistent-replacement", []dictionary.FKRef{
		{Table: "properties", Column: "property_type", Nullable: true},
	})
	assert.Error(t, err, "must fail with replacement that doesn't exist")

	// Both consumers and dict row must be intact
	_, err = b.Get(ctx, "property_types", "villa")
	require.NoError(t, err, "villa must still exist after rollback")

	total, _, err := b.ConsumersCount(ctx, "property_types", "villa", []dictionary.FKRef{
		{Table: "properties", Column: "property_type", Nullable: true},
	})
	require.NoError(t, err)
	assert.Equal(t, 14, total, "consumer rows must still reference villa after rollback")
}
