package auth

// Internal (white-box) tests for buildAuditListSQL — the pure SQL builder that
// owns the ORDER BY injection boundary and the ILIKE search group. Living in
// package auth (not auth_test) lets these call the unexported helper directly,
// with no database. The DB-backed flow is exercised by the in-memory mock in
// audit_test.go (package auth_test).

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testAuditSchema = "testschema"

// TestBuildAuditListSQL_OrderBy covers the ORDER BY whitelist + direction logic
// for the zero-opts default, every whitelisted column (ASC + DESC), and the
// injection/unknown fallbacks.
func TestBuildAuditListSQL_OrderBy(t *testing.T) {
	tests := []struct {
		name      string
		opts      AuditListOptions
		wantOrder string // exact ORDER BY clause expected in listSQL
	}{
		{
			name:      "zero opts → created_at DESC",
			opts:      AuditListOptions{},
			wantOrder: "ORDER BY created_at DESC, id DESC",
		},
		{
			name:      "created_at ASC (explicit)",
			opts:      AuditListOptions{Sort: "created_at", Desc: false},
			wantOrder: "ORDER BY created_at ASC, id DESC",
		},
		{
			name:      "created_at DESC (explicit)",
			opts:      AuditListOptions{Sort: "created_at", Desc: true},
			wantOrder: "ORDER BY created_at DESC, id DESC",
		},
		{
			name:      "action ASC adds created_at DESC tiebreak",
			opts:      AuditListOptions{Sort: "action", Desc: false},
			wantOrder: "ORDER BY action ASC, created_at DESC, id DESC",
		},
		{
			name:      "action DESC adds created_at DESC tiebreak",
			opts:      AuditListOptions{Sort: "action", Desc: true},
			wantOrder: "ORDER BY action DESC, created_at DESC, id DESC",
		},
		{
			name:      "actor_id ASC",
			opts:      AuditListOptions{Sort: "actor_id", Desc: false},
			wantOrder: "ORDER BY actor_id ASC, created_at DESC, id DESC",
		},
		{
			name:      "actor_id DESC",
			opts:      AuditListOptions{Sort: "actor_id", Desc: true},
			wantOrder: "ORDER BY actor_id DESC, created_at DESC, id DESC",
		},
		{
			name:      "target_type ASC",
			opts:      AuditListOptions{Sort: "target_type", Desc: false},
			wantOrder: "ORDER BY target_type ASC, created_at DESC, id DESC",
		},
		{
			name:      "target_type DESC",
			opts:      AuditListOptions{Sort: "target_type", Desc: true},
			wantOrder: "ORDER BY target_type DESC, created_at DESC, id DESC",
		},
		{
			name:      "ip_address ASC",
			opts:      AuditListOptions{Sort: "ip_address", Desc: false},
			wantOrder: "ORDER BY ip_address ASC, created_at DESC, id DESC",
		},
		{
			name:      "ip_address DESC",
			opts:      AuditListOptions{Sort: "ip_address", Desc: true},
			wantOrder: "ORDER BY ip_address DESC, created_at DESC, id DESC",
		},
		{
			// Unknown key → created_at fallback. Desc=false but Sort!="" → ASC.
			name:      "bogus key → created_at ASC",
			opts:      AuditListOptions{Sort: "bogus", Desc: false},
			wantOrder: "ORDER BY created_at ASC, id DESC",
		},
		{
			// Injection attempt resolves to created_at; raw text must not appear.
			name:      "injection key → created_at ASC",
			opts:      AuditListOptions{Sort: "created_at);DROP TABLE audit_events;--", Desc: false},
			wantOrder: "ORDER BY created_at ASC, id DESC",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			listSQL, countSQL, _ := buildAuditListSQL(testAuditSchema, tc.opts)
			assert.Contains(t, listSQL, tc.wantOrder, "listSQL ORDER BY")
			// COUNT(*) never carries an ORDER BY.
			assert.NotContains(t, countSQL, "ORDER BY", "countSQL must not order")
		})
	}
}

// TestBuildAuditListSQL_NoInjection asserts hostile Sort text reaches neither query.
func TestBuildAuditListSQL_NoInjection(t *testing.T) {
	for _, sortVal := range []string{
		"bogus",
		"created_at);DROP TABLE audit_events;--",
	} {
		t.Run(sortVal, func(t *testing.T) {
			listSQL, countSQL, _ := buildAuditListSQL(testAuditSchema, AuditListOptions{Sort: sortVal})
			// Fallback column present.
			assert.Contains(t, listSQL, "ORDER BY created_at")
			for _, needle := range []string{"bogus", "DROP", ";"} {
				assert.NotContains(t, listSQL, needle, "listSQL must not contain %q", needle)
				assert.NotContains(t, countSQL, needle, "countSQL must not contain %q", needle)
			}
		})
	}
}

// TestBuildAuditListSQL_NoWhereWhenUnfiltered: zero opts → no WHERE, empty args.
func TestBuildAuditListSQL_NoWhereWhenUnfiltered(t *testing.T) {
	listSQL, countSQL, args := buildAuditListSQL(testAuditSchema, AuditListOptions{})
	assert.NotContains(t, listSQL, "WHERE")
	assert.NotContains(t, countSQL, "WHERE")
	assert.Empty(t, args, "no filter/search args before limit/offset")
	// LIMIT/OFFSET placeholders are $1/$2 since args is empty.
	assert.Contains(t, listSQL, "LIMIT $1 OFFSET $2")
}

// TestBuildAuditListSQL_Search: Query adds one %arg% referenced 5× in both queries.
func TestBuildAuditListSQL_Search(t *testing.T) {
	listSQL, countSQL, args := buildAuditListSQL(testAuditSchema, AuditListOptions{Query: "foo"})

	require.Len(t, args, 1, "exactly one search arg")
	assert.Equal(t, "%foo%", args[0])

	// The search arg is placeholder $1; the 5-way ILIKE OR group references the
	// SAME $1 five times.
	wantGroup := "(action ILIKE $1 OR target_type ILIKE $1 OR ip_address ILIKE $1" +
		" OR actor_id::text ILIKE $1 OR target_id::text ILIKE $1)"
	assert.Contains(t, listSQL, wantGroup)
	assert.Equal(t, 5, strings.Count(listSQL, "$1"), "$1 referenced 5× in listSQL")

	// WHERE present in BOTH queries, sharing the same predicate.
	assert.Contains(t, listSQL, "WHERE "+wantGroup)
	assert.Contains(t, countSQL, "WHERE "+wantGroup)
}

// TestBuildAuditListSQL_SearchEscapesWildcards: %, _, \ escaped in the arg value.
func TestBuildAuditListSQL_SearchEscapesWildcards(t *testing.T) {
	_, _, args := buildAuditListSQL(testAuditSchema, AuditListOptions{Query: "50%_x"})
	require.Len(t, args, 1)
	assert.Equal(t, `%50\%\_x%`, args[0])

	// Backslash itself is escaped first (no double-escape of the % that follows).
	_, _, args = buildAuditListSQL(testAuditSchema, AuditListOptions{Query: `a\b`})
	require.Len(t, args, 1)
	assert.Equal(t, `%a\\b%`, args[0])
}

// TestBuildAuditListSQL_Combined: equality filter + search + sort together.
func TestBuildAuditListSQL_Combined(t *testing.T) {
	opts := AuditListOptions{Action: "user.invited", Query: "foo", Sort: "action", Desc: true}
	listSQL, countSQL, args := buildAuditListSQL(testAuditSchema, opts)

	// Arg order: action equality first ($1), then the search arg ($2).
	require.Len(t, args, 2)
	assert.Equal(t, "user.invited", args[0])
	assert.Equal(t, "%foo%", args[1])

	// WHERE has the action equality AND the ILIKE group (group references $2).
	assert.Contains(t, listSQL, "WHERE action = $1")
	assert.Contains(t, listSQL, "AND (action ILIKE $2 OR target_type ILIKE $2 OR ip_address ILIKE $2"+
		" OR actor_id::text ILIKE $2 OR target_id::text ILIKE $2)")
	assert.Contains(t, countSQL, "WHERE action = $1")
	assert.Contains(t, countSQL, "AND (action ILIKE $2")

	// ORDER BY uses the literal whitelisted column + literal DESC.
	assert.Contains(t, listSQL, "ORDER BY action DESC, created_at DESC, id DESC")

	// LIMIT/OFFSET placeholders follow the two filter/search args → $3/$4.
	assert.Contains(t, listSQL, "LIMIT $3 OFFSET $4")
}

// TestBuildAuditListSQL_EqualityFiltersUnchanged: existing filters still AND-combine,
// parameterized, in declared order.
func TestBuildAuditListSQL_EqualityFiltersUnchanged(t *testing.T) {
	actor := uuid.New()
	target := uuid.New()
	opts := AuditListOptions{
		Action:     "user.role_changed",
		ActorID:    &actor,
		TargetType: "user",
		TargetID:   &target,
	}
	listSQL, countSQL, args := buildAuditListSQL(testAuditSchema, opts)

	require.Len(t, args, 4)
	assert.Equal(t, "user.role_changed", args[0])
	assert.Equal(t, actor, args[1])
	assert.Equal(t, "user", args[2])
	assert.Equal(t, target, args[3])

	wantWhere := "WHERE action = $1 AND actor_id = $2 AND target_type = $3 AND target_id = $4"
	assert.Contains(t, listSQL, wantWhere)
	assert.Contains(t, countSQL, wantWhere)
	// No search arg → no ILIKE group.
	assert.NotContains(t, listSQL, "ILIKE")
}

// TestEscapeLike documents the escaping contract directly.
func TestEscapeLike(t *testing.T) {
	assert.Equal(t, "plain", escapeLike("plain"))
	assert.Equal(t, `\%`, escapeLike("%"))
	assert.Equal(t, `\_`, escapeLike("_"))
	assert.Equal(t, `\\`, escapeLike(`\`))
	// Backslash escaped before % so the result is \\ then \% (not \\%).
	assert.Equal(t, `\\\%`, escapeLike(`\%`))
}
