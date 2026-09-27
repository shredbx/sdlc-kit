package transaction

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMapper_UnitID_NilAndSet locks the per-unit-deal column wiring in the
// generated mapper (2606-002): unit_id round-trips through ToRow as a nullable
// *string, and its SELECT position immediately follows property_id so Columns()
// stays aligned with FromRow's scan order (a misaligned scan would silently read
// the wrong column). The deep DB round-trip is covered by the deal integration test.
func TestMapper_UnitID_NilAndSet(t *testing.T) {
	m := NewPostgresMapper("bestierealestate")

	// Columns() order: unit_id must sit right after property_id (matches FromRow).
	cols := m.Columns()
	pi := indexOf(cols, "t.property_id")
	require.GreaterOrEqual(t, pi, 0, "property_id must be a selected column")
	require.Less(t, pi+1, len(cols))
	assert.Equal(t, "t.unit_id", cols[pi+1], "unit_id must immediately follow property_id in Columns()")

	unit := "11111111-1111-1111-1111-111111111111"
	cases := []struct {
		name string
		unit *string
	}{
		{"whole-property deal (nil unit_id)", nil},
		{"unit-scoped deal (set unit_id)", &unit},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := Transaction{
				ID:         "22222222-2222-2222-2222-222222222222",
				Type:       TypeSale,
				Status:     StatusOpen,
				PropertyID: "33333333-3333-3333-3333-333333333333",
				UnitID:     tc.unit,
				OpenedAt:   time.Now().UTC(),
			}
			row, err := m.ToRow(in)
			require.NoError(t, err)
			require.Contains(t, row, "unit_id")
			if tc.unit == nil {
				assert.Nil(t, row["unit_id"], "nil unit_id must map to a NULL column value")
			} else {
				got, ok := row["unit_id"].(*string)
				require.True(t, ok, "unit_id must map as *string")
				require.NotNil(t, got)
				assert.Equal(t, unit, *got)
			}
		})
	}
}

func indexOf(ss []string, want string) int {
	for i, s := range ss {
		if s == want {
			return i
		}
	}
	return -1
}
