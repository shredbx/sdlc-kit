package visitoractivity_test

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	va "github.com/shredbx/sbx-core/pkg/visitoractivity"
)

// testDSNEnv gates the Postgres integration tests. When unset, the store tests
// skip cleanly (matching the sbx-core convention — see pkg/dictionary's
// postgres backend test). Set it to a reachable DSN to actually exercise the
// store, e.g.:
//
//	VISITOR_ACTIVITY_TEST_DSN=postgres://user:pass@localhost:5432/db?sslmode=disable
const testDSNEnv = "VISITOR_ACTIVITY_TEST_DSN"

// testSchemaDDL is the visitor_activity table DDL the integration harness
// applies into a throwaway schema. The PRODUCTION migration is Part B /
// deferred; this copy lets the test stand alone (and avoids a separate *.sql
// fixture, which the repo's .gitignore would drop). Keep it in sync with the
// data model in docs/plans/2026-05-27-visitor-activity-tracking.md (§3).
// %SCHEMA% is substituted with the throwaway schema name at setup.
const testSchemaDDL = `
CREATE TABLE IF NOT EXISTS %SCHEMA%.visitor_activity (
    id           UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    event_type   VARCHAR(40)  NOT NULL,
    target_type  VARCHAR(40)  NOT NULL,
    target_id    UUID,
    visitor_id   CHAR(64)     NOT NULL,
    is_staff     BOOLEAN      NOT NULL DEFAULT false,
    path         VARCHAR(512) NOT NULL,
    referer_origin VARCHAR(255),
    device_type  VARCHAR(16)  NOT NULL DEFAULT 'unknown',
    os_family      VARCHAR(16),
    browser_family VARCHAR(24),
    country_code   VARCHAR(2),
    language       VARCHAR(8),
    is_bot       BOOLEAN      NOT NULL DEFAULT false,
    props        JSONB        NOT NULL DEFAULT '{}'::jsonb,
    CONSTRAINT visitor_activity_event_type_chk  CHECK (event_type IN ('property_view','listing_view','click')),
    CONSTRAINT visitor_activity_target_type_chk CHECK (target_type IN ('property','listing'))
);
CREATE INDEX IF NOT EXISTS idx_visitor_activity_public_views
    ON %SCHEMA%.visitor_activity (target_type, target_id, created_at)
    WHERE is_staff = false AND is_bot = false;
CREATE INDEX IF NOT EXISTS idx_visitor_activity_event_created
    ON %SCHEMA%.visitor_activity (event_type, created_at);
CREATE INDEX IF NOT EXISTS idx_visitor_activity_created
    ON %SCHEMA%.visitor_activity (created_at);
`

// newTestStore spins up a pgxpool against $VISITOR_ACTIVITY_TEST_DSN, creates a
// unique throwaway schema, applies testSchemaDDL into it, and returns a Store
// bound to that schema plus the live pool (for direct row assertions). The
// schema (and everything in it) is dropped on test cleanup, so the harness
// never touches any real application schema.
//
// Skips the test entirely when the DSN env is unset or the DB is unreachable.
func newTestStore(t *testing.T) (va.Store, *pgxpool.Pool, string) {
	t.Helper()

	dsn := os.Getenv(testDSNEnv)
	if dsn == "" {
		t.Skipf("%s not set — skipping Postgres integration test", testDSNEnv)
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Skipf("cannot create pool from %s: %v", testDSNEnv, err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("cannot reach DB at %s: %v", testDSNEnv, err)
	}

	schema := randomSchema(t)
	if _, err := pool.Exec(ctx, fmt.Sprintf("CREATE SCHEMA %s", schema)); err != nil {
		pool.Close()
		t.Fatalf("create schema %s: %v", schema, err)
	}

	if _, err := pool.Exec(ctx, strings.ReplaceAll(testSchemaDDL, "%SCHEMA%", schema)); err != nil {
		_, _ = pool.Exec(ctx, fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", schema))
		pool.Close()
		t.Fatalf("apply schema DDL: %v", err)
	}

	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", schema))
		pool.Close()
	})

	return va.NewPostgresStore(pool, schema), pool, schema
}

// randomSchema returns a unique, allowlist-valid throwaway schema name.
func randomSchema(t *testing.T) string {
	t.Helper()
	b := make([]byte, 8)
	_, err := rand.Read(b)
	require.NoError(t, err)
	return "va_test_" + hex.EncodeToString(b)
}

func ptrUUID(u uuid.UUID) *uuid.UUID { return &u }

// TC1 (SE7): a dimension key not in the allowlist registry is rejected with
// ErrUnknownDimension and runs NO SQL. This is a pure unit test — it must hold
// with no DSN: the store validates the requested dimensions BEFORE touching the
// pool, so a nil pool never gets used. (The error proves the injection guard: a
// non-allowlisted key never reaches a query.)
func TestBreakdown_RejectsUnknownDimension(t *testing.T) {
	// nil pool: if validation lets an unknown dimension through to SQL, this
	// panics/errs on pool use — proving the guard runs before any query.
	store := va.NewPostgresStore(nil, "va_test_schema")

	_, err := store.PropertyStats(context.Background(), uuid.New(), time.Now().Add(-time.Hour), "not_a_real_dimension")
	require.Error(t, err)
	assert.ErrorIs(t, err, va.ErrUnknownDimension)

	_, err = store.Overview(context.Background(), time.Now().Add(-time.Hour), "definitely_bogus")
	require.Error(t, err)
	assert.ErrorIs(t, err, va.ErrUnknownDimension)
}

// TC-A4-1: Append round-trips every field, including props and a non-nil target.
func TestStore_Append_RoundTrip(t *testing.T) {
	store, pool, schema := newTestStore(t)
	ctx := context.Background()

	target := uuid.New()
	ev := &va.Event{
		EventType:     va.EventPropertyView,
		TargetType:    va.TargetProperty,
		TargetID:      ptrUUID(target),
		VisitorID:     strings.Repeat("a", 64),
		IsStaff:       true,
		Path:          "/properties/123",
		RefererOrigin: "https://google.com",
		DeviceType:    va.DeviceMobile,
		IsBot:         false,
		Props:         map[string]any{"position": float64(3), "section": "featured"},
	}
	require.NoError(t, store.Append(ctx, ev))
	assert.NotEqual(t, uuid.Nil, ev.ID, "Append must assign an id when zero")
	assert.False(t, ev.CreatedAt.IsZero(), "Append must set created_at when zero")

	// Read the row back directly and assert every column survived the trip.
	var (
		gotEventType, gotTargetType, gotVisitorID, gotPath, gotReferer, gotDevice string
		gotTargetID                                                               *uuid.UUID
		gotIsStaff, gotIsBot                                                      bool
		gotProps                                                                  map[string]any
	)
	row := pool.QueryRow(ctx, fmt.Sprintf(`SELECT event_type, target_type, target_id,
			visitor_id, is_staff, path, referer_origin, device_type, is_bot, props
		FROM %s.visitor_activity WHERE id = $1`, schema), ev.ID)
	require.NoError(t, row.Scan(&gotEventType, &gotTargetType, &gotTargetID, &gotVisitorID,
		&gotIsStaff, &gotPath, &gotReferer, &gotDevice, &gotIsBot, &gotProps))

	assert.Equal(t, string(va.EventPropertyView), gotEventType)
	assert.Equal(t, string(va.TargetProperty), gotTargetType)
	require.NotNil(t, gotTargetID)
	assert.Equal(t, target, *gotTargetID)
	assert.Equal(t, strings.Repeat("a", 64), strings.TrimRight(gotVisitorID, " "))
	assert.True(t, gotIsStaff)
	assert.Equal(t, "/properties/123", gotPath)
	assert.Equal(t, "https://google.com", gotReferer)
	assert.Equal(t, string(va.DeviceMobile), gotDevice)
	assert.False(t, gotIsBot)
	assert.Equal(t, float64(3), gotProps["position"])
	assert.Equal(t, "featured", gotProps["section"])
}

// TC-A4-2: Append accepts a nil target_id (e.g. a listing/index view with no
// specific entity) and defaults empty props to '{}'.
func TestStore_Append_NilTarget(t *testing.T) {
	store, pool, schema := newTestStore(t)
	ctx := context.Background()

	ev := &va.Event{
		EventType:  va.EventListingView,
		TargetType: va.TargetListing,
		TargetID:   nil,
		VisitorID:  strings.Repeat("b", 64),
		Path:       "/listings",
		DeviceType: va.DeviceDesktop,
		Props:      nil, // must default to {} in the column
	}
	require.NoError(t, store.Append(ctx, ev))
	assert.NotEqual(t, uuid.Nil, ev.ID)

	var gotTargetID *uuid.UUID
	var gotProps map[string]any
	row := pool.QueryRow(ctx, fmt.Sprintf(
		`SELECT target_id, props FROM %s.visitor_activity WHERE id = $1`, schema), ev.ID)
	require.NoError(t, row.Scan(&gotTargetID, &gotProps))
	assert.Nil(t, gotTargetID, "nil target must persist as NULL")
	assert.Empty(t, gotProps, "nil props must default to empty {}")
}

// TC-A4-3: NewPostgresStore panics on a schema name that fails the allowlist
// (SQL-injection guard).
func TestNewPostgresStore_PanicsOnBadSchema(t *testing.T) {
	assert.Panics(t, func() {
		// pool may be nil — the panic must happen before any DB use.
		va.NewPostgresStore(nil, "a;DROP TABLE x")
	})
}

// appendView is a test helper that records one property_view directly through
// the store.
func appendView(t *testing.T, store va.Store, target uuid.UUID, visitorID string, isStaff, isBot bool, device va.DeviceType, referer string) {
	t.Helper()
	tid := target
	require.NoError(t, store.Append(context.Background(), &va.Event{
		EventType:     va.EventPropertyView,
		TargetType:    va.TargetProperty,
		TargetID:      &tid,
		VisitorID:     visitorID,
		IsStaff:       isStaff,
		IsBot:         isBot,
		Path:          "/p",
		RefererOrigin: referer,
		DeviceType:    device,
	}))
}

// appendSignalView records one property_view carrying the full new signal set
// (country/language/os/browser) so the breakdown integration tests can assert
// per-dimension grouping.
func appendSignalView(t *testing.T, store va.Store, target uuid.UUID, visitorID string, sig va.Event) {
	t.Helper()
	tid := target
	sig.EventType = va.EventPropertyView
	sig.TargetType = va.TargetProperty
	sig.TargetID = &tid
	sig.VisitorID = visitorID
	if sig.Path == "" {
		sig.Path = "/p"
	}
	if sig.DeviceType == "" {
		sig.DeviceType = va.DeviceDesktop
	}
	require.NoError(t, store.Append(context.Background(), &sig))
}

// seededVisitor returns a deterministic 64-char visitor id for the given label.
func seededVisitor(label string) string {
	return strings.Repeat(label, 64)[:64]
}

// TC-A6-1: TopProperties ranks by unique viewers then total hits, and excludes
// staff + bot rows. Seed: visitorA×5 + visitorB×1 on propX, visitorA×1 on propY,
// plus a staff row and a bot row on propX (both excluded).
func TestStore_TopProperties(t *testing.T) {
	store, _, _ := newTestStore(t)
	ctx := context.Background()

	propX := uuid.New()
	propY := uuid.New()
	visA := seededVisitor("a")
	visB := seededVisitor("b")

	for i := 0; i < 5; i++ {
		appendView(t, store, propX, visA, false, false, va.DeviceMobile, "https://google.com")
	}
	appendView(t, store, propX, visB, false, false, va.DeviceDesktop, "")
	appendView(t, store, propY, visA, false, false, va.DeviceDesktop, "")
	// excluded:
	appendView(t, store, propX, seededVisitor("s"), true, false, va.DeviceDesktop, "") // staff
	appendView(t, store, propX, seededVisitor("z"), false, true, va.DeviceBot, "")     // bot

	got, err := store.TopProperties(ctx, time.Now().Add(-time.Hour), 10)
	require.NoError(t, err)
	require.Len(t, got, 2)

	assert.Equal(t, propX, got[0].TargetID)
	assert.Equal(t, 2, got[0].UniqueViews, "propX unique = visitorA + visitorB (staff/bot excluded)")
	assert.Equal(t, 6, got[0].TotalHits, "propX hits = 5 + 1 (staff/bot excluded)")

	assert.Equal(t, propY, got[1].TargetID)
	assert.Equal(t, 1, got[1].UniqueViews)
	assert.Equal(t, 1, got[1].TotalHits)
}

// TC-A6-2: PropertyStats returns totals + by-device + by-referer for one target,
// excluding staff/bot.
func TestStore_PropertyStats(t *testing.T) {
	store, _, _ := newTestStore(t)
	ctx := context.Background()

	propX := uuid.New()
	visA := seededVisitor("a")
	visB := seededVisitor("b")

	for i := 0; i < 5; i++ {
		appendView(t, store, propX, visA, false, false, va.DeviceMobile, "https://google.com")
	}
	appendView(t, store, propX, visB, false, false, va.DeviceDesktop, "")
	appendView(t, store, propX, seededVisitor("s"), true, false, va.DeviceDesktop, "") // staff excluded
	appendView(t, store, propX, seededVisitor("z"), false, true, va.DeviceBot, "")     // bot excluded

	stats, err := store.PropertyStats(ctx, propX, time.Now().Add(-time.Hour))
	require.NoError(t, err)
	require.NotNil(t, stats)

	assert.Equal(t, 2, stats.UniqueViews)
	assert.Equal(t, 6, stats.TotalHits)
	assert.NotEmpty(t, stats.ByDay)

	// D1: device + referer are now generic breakdowns keyed in the map. Default
	// dimension set (no explicit dims) must include both. Suppression default
	// threshold = 2 folds singletons → no per-bucket assertion below crosses it
	// because the mobile bucket has 5 hits / 1 unique → it would be suppressed,
	// so we assert via TotalHits at the aggregate (device mobile=5, desktop=1)
	// is NOT safe under suppression; instead assert the map is populated and the
	// referer "https://google.com" bucket (unique=1) folds with default threshold.
	devs := byLabel(stats.Breakdowns["device"])
	require.NotEmpty(t, stats.Breakdowns["device"], "device breakdown must populate")
	// visA (5 hits) is a single unique visitor on mobile; visB (1 hit) single
	// unique on desktop. With suppression threshold 2 both are singletons and
	// fold to Other. Assert the folded Other row carries the totals.
	assert.Equal(t, 6, devs["Other"].TotalHits, "both device singletons fold to Other under threshold 2")

	refs := byLabel(stats.Breakdowns["referer"])
	require.NotEmpty(t, stats.Breakdowns["referer"], "referer breakdown must populate")
	assert.Equal(t, 6, refs["Other"].TotalHits, "both referer singletons fold to Other under threshold 2")
}

// byLabel indexes a DimensionStat slice by its label for assertion lookups.
func byLabel(stats []va.DimensionStat) map[string]va.DimensionStat {
	m := map[string]va.DimensionStat{}
	for _, s := range stats {
		m[s.Label] = s
	}
	return m
}

// TC-A6-3: Overview totals across event types, excluding staff/bot.
func TestStore_Overview(t *testing.T) {
	store, _, _ := newTestStore(t)
	ctx := context.Background()

	propX := uuid.New()
	visA := seededVisitor("a")
	visB := seededVisitor("b")

	appendView(t, store, propX, visA, false, false, va.DeviceMobile, "")
	appendView(t, store, propX, visB, false, false, va.DeviceDesktop, "")
	appendView(t, store, propX, seededVisitor("s"), true, false, va.DeviceDesktop, "") // staff excluded

	// a listing_view and a click via direct Append
	require.NoError(t, store.Append(ctx, &va.Event{
		EventType: va.EventListingView, TargetType: va.TargetListing,
		VisitorID: visA, Path: "/listings", DeviceType: va.DeviceMobile,
	}))
	require.NoError(t, store.Append(ctx, &va.Event{
		EventType: va.EventClick, TargetType: va.TargetProperty, TargetID: &propX,
		VisitorID: visB, Path: "/p", DeviceType: va.DeviceDesktop,
	}))

	ov, err := store.Overview(ctx, time.Now().Add(-time.Hour))
	require.NoError(t, err)
	require.NotNil(t, ov)

	assert.Equal(t, 4, ov.TotalHits, "2 property_view + 1 listing_view + 1 click (staff excluded)")
	assert.Equal(t, 2, ov.ByEventType[string(va.EventPropertyView)])
	assert.Equal(t, 1, ov.ByEventType[string(va.EventListingView)])
	assert.Equal(t, 1, ov.ByEventType[string(va.EventClick)])
	assert.GreaterOrEqual(t, ov.UniqueVisitors, 2)
}

// appendViewAt records one property_view at an explicit created_at (so day-bucket
// assertions are deterministic across UTC day boundaries).
func appendViewAt(t *testing.T, store va.Store, target uuid.UUID, visitorID string, isStaff, isBot bool, createdAt time.Time) {
	t.Helper()
	tid := target
	require.NoError(t, store.Append(context.Background(), &va.Event{
		EventType:  va.EventPropertyView,
		TargetType: va.TargetProperty,
		TargetID:   &tid,
		VisitorID:  visitorID,
		IsStaff:    isStaff,
		IsBot:      isBot,
		Path:       "/p",
		DeviceType: va.DeviceMobile,
		CreatedAt:  createdAt,
	}))
}

// TC10: Overview.ByDay buckets unique visitors + total hits per UTC day,
// excluding staff and bots. Seed: day1 (visA×2 + visB×1) + day2 (visA×1), plus a
// staff row and a bot row on day1 (both excluded from every figure).
func TestOverview_PopulatesByDay(t *testing.T) {
	store, _, _ := newTestStore(t)
	ctx := context.Background()

	propX := uuid.New()
	visA := seededVisitor("a")
	visB := seededVisitor("b")

	// Two distinct UTC days, well inside the query window, away from "now" so a
	// midnight boundary never reclassifies a row.
	day1 := time.Date(2026, 1, 10, 9, 0, 0, 0, time.UTC)
	day2 := time.Date(2026, 1, 11, 14, 0, 0, 0, time.UTC)

	// day1: visA twice + visB once → unique=2, hits=3
	appendViewAt(t, store, propX, visA, false, false, day1)
	appendViewAt(t, store, propX, visA, false, false, day1.Add(2*time.Hour))
	appendViewAt(t, store, propX, visB, false, false, day1)
	// day1 excluded rows:
	appendViewAt(t, store, propX, seededVisitor("s"), true, false, day1) // staff
	appendViewAt(t, store, propX, seededVisitor("z"), false, true, day1) // bot
	// day2: visA once → unique=1, hits=1
	appendViewAt(t, store, propX, visA, false, false, day2)

	// since well before day1.
	ov, err := store.Overview(ctx, day1.Add(-24*time.Hour))
	require.NoError(t, err)
	require.NotNil(t, ov)
	require.Len(t, ov.ByDay, 2, "two distinct UTC days seeded (staff/bot excluded)")

	byDay := map[string]va.DayStat{}
	for _, d := range ov.ByDay {
		byDay[d.Day.UTC().Format("2006-01-02")] = d
	}

	d1 := byDay["2026-01-10"]
	assert.Equal(t, 2, d1.UniqueViews, "day1 unique = visA + visB (staff/bot excluded)")
	assert.Equal(t, 3, d1.TotalHits, "day1 hits = 3 (staff/bot excluded)")

	d2 := byDay["2026-01-11"]
	assert.Equal(t, 1, d2.UniqueViews)
	assert.Equal(t, 1, d2.TotalHits)
}

// distinctVisitor returns a deterministic 64-char visitor id keyed by an integer
// so each seeded row can be a distinct unique visitor (to clear the suppression
// threshold in the breakdown integration tests).
func distinctVisitor(n int) string {
	s := fmt.Sprintf("v%063d", n)
	return s[:64]
}

// TC6 (SE6): the generic breakdown helper groups by an allowlisted dimension and
// returns correct unique/total counts per bucket. Seeds 3 distinct visitors per
// country/os bucket so each bucket clears the default suppression threshold (2).
func TestBreakdown_GroupsByDimension(t *testing.T) {
	store, _, _ := newTestStore(t)
	ctx := context.Background()
	propX := uuid.New()

	// TH: 3 distinct visitors, 4 hits (one repeats). US: 2 distinct, 2 hits.
	appendSignalView(t, store, propX, distinctVisitor(1), va.Event{Country: "TH", Language: "th", OSFamily: va.OSiOS, BrowserFamily: va.BrowserSafari, DeviceType: va.DeviceMobile})
	appendSignalView(t, store, propX, distinctVisitor(1), va.Event{Country: "TH", Language: "th", OSFamily: va.OSiOS, BrowserFamily: va.BrowserSafari, DeviceType: va.DeviceMobile})
	appendSignalView(t, store, propX, distinctVisitor(2), va.Event{Country: "TH", Language: "th", OSFamily: va.OSiOS, BrowserFamily: va.BrowserSafari, DeviceType: va.DeviceMobile})
	appendSignalView(t, store, propX, distinctVisitor(3), va.Event{Country: "TH", Language: "en", OSFamily: va.OSAndroid, BrowserFamily: va.BrowserChrome, DeviceType: va.DeviceMobile})
	appendSignalView(t, store, propX, distinctVisitor(4), va.Event{Country: "US", Language: "en", OSFamily: va.OSWindows, BrowserFamily: va.BrowserEdge, DeviceType: va.DeviceDesktop})
	appendSignalView(t, store, propX, distinctVisitor(5), va.Event{Country: "US", Language: "en", OSFamily: va.OSWindows, BrowserFamily: va.BrowserChrome, DeviceType: va.DeviceDesktop})

	stats, err := store.PropertyStats(ctx, propX, time.Now().Add(-time.Hour))
	require.NoError(t, err)
	require.NotNil(t, stats)

	// TH has 3 unique visitors and 4 hits (visitor 1 repeats); US has 2 unique, 2 hits.
	countries := byLabel(stats.Breakdowns["country"])
	assert.Equal(t, 3, countries["TH"].UniqueViews)
	assert.Equal(t, 4, countries["TH"].TotalHits)
	assert.Equal(t, 2, countries["US"].UniqueViews)
	assert.Equal(t, 2, countries["US"].TotalHits)

	langs := byLabel(stats.Breakdowns["language"])
	assert.Equal(t, 3, langs["en"].UniqueViews, "en: visitors 3,4,5")
	assert.Equal(t, 2, langs["th"].UniqueViews, "th: visitors 1,2 (visitor 1 repeats)")

	oses := byLabel(stats.Breakdowns["os"])
	assert.Equal(t, 2, oses["windows"].UniqueViews)
}

// TC10 (SE3): the beacon (track.ts) folds the UTM allowlist into a NESTED
// props.utm object, so the utm_* registry must read props->'utm'->>'utm_source'
// (not top-level props->>'utm_source'). This test seeds nested-utm props and
// asserts campaign grouping — it guards that frontend↔store contract.
func TestBreakdown_GroupsByUTM(t *testing.T) {
	store, _, _ := newTestStore(t)
	ctx := context.Background()
	propX := uuid.New()

	utmProps := func(source string) map[string]any {
		return map[string]any{"utm": map[string]any{
			"utm_source": source, "utm_medium": "cpc", "utm_campaign": "spring",
		}}
	}
	// google ×2 distinct, facebook ×2 distinct — both clear the suppression threshold.
	appendSignalView(t, store, propX, distinctVisitor(1), va.Event{Props: utmProps("google")})
	appendSignalView(t, store, propX, distinctVisitor(2), va.Event{Props: utmProps("google")})
	appendSignalView(t, store, propX, distinctVisitor(3), va.Event{Props: utmProps("facebook")})
	appendSignalView(t, store, propX, distinctVisitor(4), va.Event{Props: utmProps("facebook")})

	stats, err := store.PropertyStats(ctx, propX, time.Now().Add(-time.Hour))
	require.NoError(t, err)
	require.NotNil(t, stats)

	sources := byLabel(stats.Breakdowns["utm_source"])
	assert.Equal(t, 2, sources["google"].UniqueViews, "nested props.utm.utm_source must be read, not top-level")
	assert.Equal(t, 2, sources["facebook"].UniqueViews)
	campaigns := byLabel(stats.Breakdowns["utm_campaign"])
	assert.Equal(t, 4, campaigns["spring"].UniqueViews, "all four hits share the spring campaign")
}

// TC7 (SE8): staff and bot rows are excluded from EVERY dimension bucket. Seeds 3
// clean visitors plus a staff row and a bot row in the same country/os bucket;
// the breakdown counts must reflect only the 3 clean visitors.
func TestBreakdown_ExcludesStaffBot(t *testing.T) {
	store, _, _ := newTestStore(t)
	ctx := context.Background()
	propX := uuid.New()

	for i := 1; i <= 3; i++ {
		appendSignalView(t, store, propX, distinctVisitor(i), va.Event{Country: "TH", Language: "th", OSFamily: va.OSiOS, BrowserFamily: va.BrowserSafari})
	}
	// excluded staff + bot in the SAME bucket:
	appendSignalView(t, store, propX, distinctVisitor(98), va.Event{Country: "TH", Language: "th", OSFamily: va.OSiOS, BrowserFamily: va.BrowserSafari, IsStaff: true})
	appendSignalView(t, store, propX, distinctVisitor(99), va.Event{Country: "TH", Language: "th", OSFamily: va.OSiOS, BrowserFamily: va.BrowserSafari, IsBot: true})

	stats, err := store.PropertyStats(ctx, propX, time.Now().Add(-time.Hour))
	require.NoError(t, err)
	require.NotNil(t, stats)

	countries := byLabel(stats.Breakdowns["country"])
	assert.Equal(t, 3, countries["TH"].UniqueViews, "staff + bot excluded from country bucket")
	assert.Equal(t, 3, countries["TH"].TotalHits)
}

// TC8 (SE9): a bucket whose unique_views is below the suppression threshold is
// folded into a single Other row; totals are unaffected. Seeds one country bucket
// well above the threshold (TH×3) and two singleton buckets (LV×1, EE×1) that
// must collapse into Other (2) — never emitted size-1.
func TestBreakdown_SuppressesSmallCounts(t *testing.T) {
	store, _, _ := newTestStore(t)
	ctx := context.Background()
	propX := uuid.New()

	for i := 1; i <= 3; i++ {
		appendSignalView(t, store, propX, distinctVisitor(i), va.Event{Country: "TH"})
	}
	appendSignalView(t, store, propX, distinctVisitor(50), va.Event{Country: "LV"}) // singleton
	appendSignalView(t, store, propX, distinctVisitor(51), va.Event{Country: "EE"}) // singleton

	stats, err := store.PropertyStats(ctx, propX, time.Now().Add(-time.Hour))
	require.NoError(t, err)
	require.NotNil(t, stats)

	countries := byLabel(stats.Breakdowns["country"])
	assert.Equal(t, 3, countries["TH"].UniqueViews, "TH above threshold stays visible")
	_, hasLV := countries["LV"]
	_, hasEE := countries["EE"]
	assert.False(t, hasLV, "singleton LV must NOT be emitted size-1")
	assert.False(t, hasEE, "singleton EE must NOT be emitted size-1")
	require.Contains(t, countries, "Other", "singletons fold into an Other row")
	assert.Equal(t, 2, countries["Other"].UniqueViews, "LV + EE folded into Other")
	assert.Equal(t, 2, countries["Other"].TotalHits)

	// Totals must be unaffected by suppression.
	assert.Equal(t, 5, stats.UniqueViews, "5 distinct visitors total")
	assert.Equal(t, 5, stats.TotalHits)
}

// TC11 (SE5): read-side hour + weekday dimensions group rows by created_at with
// no stored column. Seeds rows at distinct UTC hours/weekdays with 2 distinct
// visitors per bucket so they clear the suppression threshold.
func TestBreakdown_ByHourWeekday(t *testing.T) {
	store, _, _ := newTestStore(t)
	ctx := context.Background()
	propX := uuid.New()

	// 2026-01-12 is a Monday; 2026-01-13 a Tuesday (UTC).
	mon09 := time.Date(2026, 1, 12, 9, 30, 0, 0, time.UTC)
	tue14 := time.Date(2026, 1, 13, 14, 15, 0, 0, time.UTC)

	appendSignalView(t, store, propX, distinctVisitor(1), va.Event{CreatedAt: mon09})
	appendSignalView(t, store, propX, distinctVisitor(2), va.Event{CreatedAt: mon09})
	appendSignalView(t, store, propX, distinctVisitor(3), va.Event{CreatedAt: tue14})
	appendSignalView(t, store, propX, distinctVisitor(4), va.Event{CreatedAt: tue14})

	stats, err := store.PropertyStats(ctx, propX, mon09.Add(-24*time.Hour))
	require.NoError(t, err)
	require.NotNil(t, stats)

	hours := byLabel(stats.Breakdowns["hour"])
	assert.Equal(t, 2, hours["09"].UniqueViews, "two visitors at hour 09")
	assert.Equal(t, 2, hours["14"].UniqueViews, "two visitors at hour 14")

	weekdays := byLabel(stats.Breakdowns["weekday"])
	assert.Equal(t, 2, weekdays["Mon"].UniqueViews)
	assert.Equal(t, 2, weekdays["Tue"].UniqueViews)
}

// appendPageViewAt records one page_view for a URL path at an explicit
// created_at (the path is the identifier — no target id). Mirrors appendViewAt
// but for the path-keyed page target.
func appendPageViewAt(t *testing.T, store va.Store, path, visitorID string, isStaff, isBot bool, createdAt time.Time, referer string) {
	t.Helper()
	require.NoError(t, store.Append(context.Background(), &va.Event{
		EventType:     va.EventPageView,
		TargetType:    va.TargetPage,
		TargetID:      nil,
		VisitorID:     visitorID,
		IsStaff:       isStaff,
		IsBot:         isBot,
		Path:          path,
		RefererOrigin: referer,
		DeviceType:    va.DeviceDesktop,
		CreatedAt:     createdAt,
	}))
}

// TC-P1: PathStats returns totals + by-day + breakdowns for one URL path,
// keyed on path + page_view, excluding staff and bots. Seeds the same path with
// clean visitors across two UTC days plus a staff and a bot row (both excluded),
// and a page_view for a DIFFERENT path that must not leak in.
func TestStore_PathStats(t *testing.T) {
	store, _, _ := newTestStore(t)
	ctx := context.Background()

	const path = "/sell"
	day1 := time.Date(2026, 2, 10, 9, 0, 0, 0, time.UTC)
	day2 := time.Date(2026, 2, 11, 14, 0, 0, 0, time.UTC)

	// day1: 3 distinct visitors (one repeats) → unique=3, hits=4.
	appendPageViewAt(t, store, path, distinctVisitor(1), false, false, day1, "https://google.com")
	appendPageViewAt(t, store, path, distinctVisitor(1), false, false, day1.Add(time.Hour), "https://google.com")
	appendPageViewAt(t, store, path, distinctVisitor(2), false, false, day1, "https://google.com")
	appendPageViewAt(t, store, path, distinctVisitor(3), false, false, day1, "https://google.com")
	// day2: 1 distinct visitor → unique=1, hits=1.
	appendPageViewAt(t, store, path, distinctVisitor(4), false, false, day2, "https://google.com")
	// excluded staff + bot on the same path/day1:
	appendPageViewAt(t, store, path, distinctVisitor(98), true, false, day1, "")
	appendPageViewAt(t, store, path, distinctVisitor(99), false, true, day1, "")
	// a different path's page_view must not bleed into /sell.
	appendPageViewAt(t, store, "/", distinctVisitor(50), false, false, day1, "")

	stats, err := store.PathStats(ctx, path, day1.Add(-24*time.Hour))
	require.NoError(t, err)
	require.NotNil(t, stats)

	assert.Equal(t, 4, stats.UniqueViews, "4 distinct clean visitors on /sell (staff/bot + other path excluded)")
	assert.Equal(t, 5, stats.TotalHits, "5 clean hits on /sell")
	require.Len(t, stats.ByDay, 2, "two distinct UTC days seeded for /sell")

	byDay := map[string]va.DayStat{}
	for _, d := range stats.ByDay {
		byDay[d.Day.UTC().Format("2006-01-02")] = d
	}
	assert.Equal(t, 3, byDay["2026-02-10"].UniqueViews, "day1 unique = visitors 1,2,3")
	assert.Equal(t, 4, byDay["2026-02-10"].TotalHits, "day1 hits = 4 (visitor 1 repeats)")
	assert.Equal(t, 1, byDay["2026-02-11"].UniqueViews)
	assert.Equal(t, 1, byDay["2026-02-11"].TotalHits)

	// referer breakdown: all clean hits share google → above threshold, visible.
	refs := byLabel(stats.Breakdowns["referer"])
	require.NotEmpty(t, stats.Breakdowns["referer"], "referer breakdown must populate")
	assert.Equal(t, 4, refs["https://google.com"].UniqueViews, "4 distinct clean google referers")
	assert.Equal(t, 5, refs["https://google.com"].TotalHits)
}

// TC-P2: PathStats rejects a non-allowlisted dimension with ErrUnknownDimension
// before any SQL runs (injection guard) — holds with a nil pool.
func TestStore_PathStats_RejectsUnknownDimension(t *testing.T) {
	store := va.NewPostgresStore(nil, "va_test_schema")
	_, err := store.PathStats(context.Background(), "/sell", time.Now().Add(-time.Hour), "not_a_real_dimension")
	require.Error(t, err)
	assert.ErrorIs(t, err, va.ErrUnknownDimension)
}

// TC-P3: PathSummary returns the all-time total and today-distinct visitor count
// for one path, excluding staff and bots. Seeds an old hit, two today hits from
// one visitor, one today hit from a second visitor, plus a today staff and a
// today bot hit (both excluded). Total = clean all-time count; TodayViews =
// distinct clean visitors today.
func TestStore_PathSummary(t *testing.T) {
	store, _, _ := newTestStore(t)
	ctx := context.Background()

	const path = "/"
	// now()-based today boundary: seed "today" rows close to now and an older row
	// well before today so it counts toward Total but not Today.
	now := time.Now().UTC()
	earlierToday := now.Add(-30 * time.Minute)
	yesterday := now.Add(-26 * time.Hour)

	// all-time clean hits: 1 yesterday + 3 today = 4 total.
	appendPageViewAt(t, store, path, distinctVisitor(1), false, false, yesterday, "")
	appendPageViewAt(t, store, path, distinctVisitor(1), false, false, earlierToday, "")
	appendPageViewAt(t, store, path, distinctVisitor(1), false, false, now.Add(-5*time.Minute), "")
	appendPageViewAt(t, store, path, distinctVisitor(2), false, false, earlierToday, "")
	// excluded today: staff + bot.
	appendPageViewAt(t, store, path, distinctVisitor(98), true, false, earlierToday, "")
	appendPageViewAt(t, store, path, distinctVisitor(99), false, true, earlierToday, "")
	// a different path must not bleed in.
	appendPageViewAt(t, store, "/sell", distinctVisitor(3), false, false, earlierToday, "")

	sum, err := store.PathSummary(ctx, path)
	require.NoError(t, err)

	assert.Equal(t, 4, sum.TotalViews, "all-time clean page_view hits on / (staff/bot + other path excluded)")
	assert.Equal(t, 2, sum.TodayViews, "distinct clean visitors today = visitor 1 + visitor 2")
}
