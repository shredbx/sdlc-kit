package visitoractivity

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrUnknownDimension is returned by the store when a requested breakdown
// dimension is not in the allowlisted dimensionExpr registry. The dimension key
// is rejected BEFORE any SQL is built or run — it is the injection guard for the
// (internal-constant) SQL expressions: a key that is not in the registry can
// never reach a query.
var ErrUnknownDimension = errors.New("visitoractivity: unknown dimension")

// defaultSuppressThreshold is the small-count suppression floor (Decision D3):
// any breakdown bucket whose UniqueViews is below this is folded into a single
// "Other" row so the API never emits a size-1 cell (statistical-disclosure /
// k-anonymity guard). Default 2 hides pure singletons while keeping low-traffic
// dashboards informative; raise as traffic grows.
const defaultSuppressThreshold = 2

// otherLabel is the bucket that suppressed small-count cells fold into.
const otherLabel = "Other"

// TargetStat is a per-target rollup used by leaderboard-style queries.
type TargetStat struct {
	TargetID    uuid.UUID `json:"target_id"`
	UniqueViews int       `json:"unique_views"` // COUNT(DISTINCT visitor_id)
	TotalHits   int       `json:"total_hits"`   // COUNT(*)
}

// LabeledTarget pairs a target stat with a consumer-resolved display label. The
// polymorphic target_id carries no foreign key, so only the consumer can resolve
// a human label for a target — and decide a target no longer exists (a deleted
// entity), in which case the consumer drops it. The leaderboard handler writes
// these instead of raw TargetStats whenever an InsightsDeps.ResolveTop resolver
// is supplied (see http.go).
type LabeledTarget struct {
	TargetStat
	Label string `json:"label"`
}

// DayStat is a per-day rollup (UTC day buckets).
type DayStat struct {
	Day         time.Time `json:"day"`
	UniqueViews int       `json:"unique_views"`
	TotalHits   int       `json:"total_hits"`
}

// DimensionStat is a single bucket of a generic breakdown: a Label (the grouped
// value, e.g. a country code, an OS family, an hour) plus its unique-visitor and
// total-hit counts. It is the one result shape every dimension shares (Decision
// D1 — supersedes the old per-dimension RefererStat/DeviceStat structs).
type DimensionStat struct {
	Label       string `json:"label"`
	UniqueViews int    `json:"unique_views"`
	TotalHits   int    `json:"total_hits"`
}

// PropertyStats is the full per-property breakdown for an analytics detail view.
// Breakdowns is keyed by dimension (e.g. "country", "device", "channel"); see
// the dimensionExpr registry for the supported keys.
type PropertyStats struct {
	UniqueViews int                        `json:"unique_views"`
	TotalHits   int                        `json:"total_hits"`
	ByDay       []DayStat                  `json:"by_day"`
	Breakdowns  map[string][]DimensionStat `json:"breakdowns"`
}

// PathSummary is a compact per-page rollup for a single URL path: the all-time
// total page views and the number of distinct visitors so far today. Mirrors the
// lightweight summary a page header surfaces (no by-day / breakdown detail).
type PathSummary struct {
	TotalViews int `json:"total_views"` // all-time COUNT(*) of page_view hits (staff/bot excluded)
	TodayViews int `json:"today_views"` // COUNT(DISTINCT visitor_id) since UTC start-of-today
}

// Overview is a workspace-wide rollup across event types for a time window.
type Overview struct {
	TotalHits      int            `json:"total_hits"`
	UniqueVisitors int            `json:"unique_visitors"`
	ByEventType    map[string]int `json:"by_event_type"`
	// ByDay is a per-day rollup (UTC day buckets) of unique visitors + total
	// hits across all event types, mirroring PropertyStats.ByDay. Additive and
	// backward-compatible — existing callers ignoring this field are unaffected.
	ByDay []DayStat `json:"by_day"`
	// Breakdowns is keyed by dimension, like PropertyStats.Breakdowns.
	Breakdowns map[string][]DimensionStat `json:"breakdowns"`
}

// dimensionExpr maps an API-safe dimension KEY → a TRUSTED, INTERNAL-CONSTANT
// SQL expression that becomes the GROUP BY label. The expression is NEVER user
// input — exactly the same injection-guard principle as the schema allowlist
// (schema.go). Adding a dimension is ONE entry here (+ a stored column only if
// the value isn't already derivable from an existing column / props / timestamp).
//
// Read-side dimensions (channel/hour/weekday/utm_*) compute their value at query
// time from columns already present — no stored column, no write-side derivation.
var dimensionExpr = map[string]string{
	"referer":  "COALESCE(referer_origin,'')",
	"device":   "device_type",
	"country":  "COALESCE(country_code,'')",
	"language": "COALESCE(language,'')",
	"os":       "COALESCE(os_family,'')",
	"browser":  "COALESCE(browser_family,'')",
	// channel: read-side classification of referer_origin — kept in lockstep
	// with ClassifyChannel (identity.go).
	"channel": "CASE " +
		"WHEN referer_origin IS NULL OR referer_origin = '' THEN 'direct' " +
		"WHEN referer_origin ~* '(google|bing|duckduckgo|yahoo)' THEN 'search' " +
		"WHEN referer_origin ~* '(facebook|instagram|t\\.co|twitter|tiktok|line|lnk)' THEN 'social' " +
		"ELSE 'referral' END",
	"hour":    "to_char(created_at, 'HH24')",
	"weekday": "trim(to_char(created_at, 'Dy'))",
	// utm_*: the beacon (track.ts) folds the UTM allowlist into a nested
	// props.utm object, so read props->'utm'->>'utm_source' (NOT top-level).
	"utm_source":   "props->'utm'->>'utm_source'",
	"utm_medium":   "props->'utm'->>'utm_medium'",
	"utm_campaign": "props->'utm'->>'utm_campaign'",
}

// defaultDimensions is the ordered breakdown set computed when a caller passes no
// explicit dimensions. Deterministic order keeps test output and the JSON map
// population stable.
var defaultDimensions = []string{
	"referer", "device", "country", "language", "os", "browser",
	"channel", "hour", "weekday", "utm_source", "utm_medium", "utm_campaign",
}

// resolveDimensions returns the dimension set to compute and validates every key
// against the allowlist BEFORE any SQL runs. An empty request → the full default
// set. A non-allowlisted key → ErrUnknownDimension (and no query is built).
func resolveDimensions(dims []string) ([]string, error) {
	if len(dims) == 0 {
		return defaultDimensions, nil
	}
	for _, d := range dims {
		if _, ok := dimensionExpr[d]; !ok {
			return nil, fmt.Errorf("%w: %q", ErrUnknownDimension, d)
		}
	}
	return dims, nil
}

// Store persists visitor-activity events and answers aggregation reads. All
// reads exclude staff and bot traffic so public "view" numbers reflect real
// anonymous visitors. Mirrors the hand-written pkg/auth/audit append store.
type Store interface {
	Append(ctx context.Context, e *Event) error

	// TopProperties ranks property targets by unique anonymous viewers since the
	// given time, then by total hits, capped at limit.
	TopProperties(ctx context.Context, since time.Time, limit int) ([]TargetStat, error)

	// PropertyStats returns totals plus by-day and a generic Breakdowns map for
	// one property target since the given time. Optional dims restricts which
	// breakdown dimensions are computed (empty = the full registry). A
	// non-allowlisted dimension returns ErrUnknownDimension and runs no breakdown
	// SQL.
	PropertyStats(ctx context.Context, targetID uuid.UUID, since time.Time, dims ...string) (*PropertyStats, error)

	// Overview returns workspace-wide totals across event types plus a generic
	// Breakdowns map since the given time. Optional dims as PropertyStats.
	Overview(ctx context.Context, since time.Time, dims ...string) (*Overview, error)

	// PathStats returns the same totals + by-day + Breakdowns shape as
	// PropertyStats, but for one page identified by its URL path instead of a
	// target id (page_view events only). Optional dims as PropertyStats; a
	// non-allowlisted dimension returns ErrUnknownDimension and runs no SQL.
	PathStats(ctx context.Context, path string, since time.Time, dims ...string) (*PropertyStats, error)

	// PathSummary returns the all-time total page views and the today-distinct
	// visitor count for one URL path (page_view events, staff/bots excluded).
	PathSummary(ctx context.Context, path string) (PathSummary, error)
}

type postgresStore struct {
	pool   *pgxpool.Pool
	schema string
}

// NewPostgresStore returns a Postgres-backed Store for the visitor_activity
// table in the given schema. Panics if schema is not a valid PostgreSQL
// identifier (SQL-injection guard) — fails loudly at wiring time.
func NewPostgresStore(pool *pgxpool.Pool, schema string) Store {
	mustValidateSchema(schema)
	return &postgresStore{pool: pool, schema: schema}
}

// Append inserts one event via a fully parameterized statement (only the schema
// identifier is interpolated, and it is allowlist-validated in the constructor).
// Mirrors auth.Append: generate id/created_at when zero, default props to '{}'.
func (s *postgresStore) Append(ctx context.Context, e *Event) error {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	if e.CreatedAt.IsZero() {
		e.CreatedAt = time.Now()
	}
	props, err := json.Marshal(e.Props)
	if err != nil {
		return fmt.Errorf("visitoractivity: marshal props: %w", err)
	}
	if len(props) == 0 || string(props) == "null" {
		props = []byte("{}")
	}
	query := fmt.Sprintf(`INSERT INTO %s.visitor_activity
		(id, created_at, event_type, target_type, target_id, visitor_id, is_staff,
		 path, referer_origin, device_type, os_family, browser_family, country_code,
		 language, is_bot, props)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)`, s.schema)
	_, err = s.pool.Exec(ctx, query,
		e.ID, e.CreatedAt, string(e.EventType), string(e.TargetType), e.TargetID,
		e.VisitorID, e.IsStaff, e.Path, nullIfEmpty(e.RefererOrigin),
		string(e.DeviceType), nullIfEmpty(string(e.OSFamily)), nullIfEmpty(string(e.BrowserFamily)),
		nullIfEmpty(e.Country), nullIfEmpty(e.Language), e.IsBot, props)
	if err != nil {
		return fmt.Errorf("visitoractivity: append: %w", err)
	}
	return nil
}

// clampLimit normalizes a client-supplied result limit: a non-positive limit
// defaults to 10, and any limit above 500 is capped at 500 so a runaway query
// can't pull an unbounded grouped set into memory (mirrors pkg/auth/audit.go).
func clampLimit(limit int) int {
	if limit <= 0 {
		return 10
	}
	if limit > 500 {
		return 500
	}
	return limit
}

// TopProperties — see Store. Excludes staff and bots.
func (s *postgresStore) TopProperties(ctx context.Context, since time.Time, limit int) ([]TargetStat, error) {
	limit = clampLimit(limit)
	query := fmt.Sprintf(`SELECT target_id, COUNT(DISTINCT visitor_id) AS unique_views, COUNT(*) AS total_hits
		FROM %s.visitor_activity
		WHERE event_type = 'property_view' AND target_type = 'property' AND target_id IS NOT NULL
		  AND created_at >= $1 AND is_staff = false AND is_bot = false
		GROUP BY target_id
		ORDER BY unique_views DESC, total_hits DESC
		LIMIT $2`, s.schema)
	rows, err := s.pool.Query(ctx, query, since, limit)
	if err != nil {
		return nil, fmt.Errorf("visitoractivity: top properties: %w", err)
	}
	defer rows.Close()

	var out []TargetStat
	for rows.Next() {
		var ts TargetStat
		if err := rows.Scan(&ts.TargetID, &ts.UniqueViews, &ts.TotalHits); err != nil {
			return nil, fmt.Errorf("visitoractivity: scan top property: %w", err)
		}
		out = append(out, ts)
	}
	return out, rows.Err()
}

// PropertyStats — see Store. Excludes staff and bots.
func (s *postgresStore) PropertyStats(ctx context.Context, targetID uuid.UUID, since time.Time, dims ...string) (*PropertyStats, error) {
	// Validate the requested dimensions BEFORE any SQL (injection guard).
	wantDims, err := resolveDimensions(dims)
	if err != nil {
		return nil, err
	}

	base := fmt.Sprintf(`FROM %s.visitor_activity
		WHERE event_type = 'property_view' AND target_type = 'property' AND target_id = $1
		  AND created_at >= $2 AND is_staff = false AND is_bot = false`, s.schema)
	args := []any{targetID, since}

	stats := &PropertyStats{Breakdowns: map[string][]DimensionStat{}}

	// Totals.
	totalsQ := "SELECT COALESCE(COUNT(DISTINCT visitor_id),0), COALESCE(COUNT(*),0) " + base
	if err := s.pool.QueryRow(ctx, totalsQ, args...).
		Scan(&stats.UniqueViews, &stats.TotalHits); err != nil {
		return nil, fmt.Errorf("visitoractivity: property totals: %w", err)
	}

	// By day (UTC).
	byDayQ := "SELECT date_trunc('day', created_at) AS day, COUNT(DISTINCT visitor_id), COUNT(*) " +
		base + " GROUP BY day ORDER BY day"
	if stats.ByDay, err = s.scanDays(ctx, byDayQ, args); err != nil {
		return nil, fmt.Errorf("visitoractivity: property by-day: %w", err)
	}

	// Generic breakdowns — one helper per allowlisted dimension.
	for _, dim := range wantDims {
		rows, err := s.breakdown(ctx, base, args, dim)
		if err != nil {
			return nil, err
		}
		stats.Breakdowns[dim] = rows
	}

	return stats, nil
}

// PathStats — see Store. Identical aggregation to PropertyStats, but keyed on
// the URL path + page_view event type instead of target_id/target_type. Excludes
// staff and bots. Returns the SAME *PropertyStats shape so the frontend reuses
// the per-property detail rendering for a page.
func (s *postgresStore) PathStats(ctx context.Context, path string, since time.Time, dims ...string) (*PropertyStats, error) {
	// Validate the requested dimensions BEFORE any SQL (injection guard).
	wantDims, err := resolveDimensions(dims)
	if err != nil {
		return nil, err
	}

	base := fmt.Sprintf(`FROM %s.visitor_activity
		WHERE event_type = 'page_view' AND path = $1
		  AND created_at >= $2 AND is_staff = false AND is_bot = false`, s.schema)
	args := []any{path, since}

	stats := &PropertyStats{Breakdowns: map[string][]DimensionStat{}}

	// Totals.
	totalsQ := "SELECT COALESCE(COUNT(DISTINCT visitor_id),0), COALESCE(COUNT(*),0) " + base
	if err := s.pool.QueryRow(ctx, totalsQ, args...).
		Scan(&stats.UniqueViews, &stats.TotalHits); err != nil {
		return nil, fmt.Errorf("visitoractivity: path totals: %w", err)
	}

	// By day (UTC).
	byDayQ := "SELECT date_trunc('day', created_at) AS day, COUNT(DISTINCT visitor_id), COUNT(*) " +
		base + " GROUP BY day ORDER BY day"
	if stats.ByDay, err = s.scanDays(ctx, byDayQ, args); err != nil {
		return nil, fmt.Errorf("visitoractivity: path by-day: %w", err)
	}

	// Generic breakdowns — one helper per allowlisted dimension.
	for _, dim := range wantDims {
		rows, err := s.breakdown(ctx, base, args, dim)
		if err != nil {
			return nil, err
		}
		stats.Breakdowns[dim] = rows
	}

	return stats, nil
}

// PathSummary — see Store. TotalViews is the all-time COUNT(*) of page_view hits
// for the path; TodayViews is the distinct-visitor count since the start of the
// current day. The today boundary uses date_trunc('day', now()) — the SAME
// date_trunc('day', …) treatment the by-day rollups apply to created_at, so a
// session running in UTC (as the store does) buckets "today" on the UTC day.
func (s *postgresStore) PathSummary(ctx context.Context, path string) (PathSummary, error) {
	var sum PathSummary
	query := fmt.Sprintf(`SELECT
			COALESCE(COUNT(*),0) AS total_views,
			COALESCE(COUNT(DISTINCT visitor_id) FILTER (
				WHERE created_at >= date_trunc('day', now())
			),0) AS today_views
		FROM %s.visitor_activity
		WHERE event_type = 'page_view' AND path = $1
		  AND is_staff = false AND is_bot = false`, s.schema)
	if err := s.pool.QueryRow(ctx, query, path).Scan(&sum.TotalViews, &sum.TodayViews); err != nil {
		return PathSummary{}, fmt.Errorf("visitoractivity: path summary: %w", err)
	}
	return sum, nil
}

// scanDays runs a per-day grouping query and collects DayStat rows. Shared by
// PropertyStats and Overview so the by-day scan loop lives in one place.
func (s *postgresStore) scanDays(ctx context.Context, query string, args []any) ([]DayStat, error) {
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DayStat
	for rows.Next() {
		var d DayStat
		if err := rows.Scan(&d.Day, &d.UniqueViews, &d.TotalHits); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// breakdown runs ONE allowlisted dimension grouping over the (already
// staff/bot-filtered) base query and returns small-count-suppressed buckets.
//
// The dimension key is rejected (ErrUnknownDimension, no SQL) if it is not in
// the dimensionExpr registry — the expression is a trusted internal constant, so
// interpolating it is safe; the values stay in bound parameters (args). This is
// the single generic helper that replaces every per-dimension query block: a new
// dimension is a one-line registry entry, nothing here changes.
func (s *postgresStore) breakdown(ctx context.Context, base string, args []any, dim string) ([]DimensionStat, error) {
	expr, ok := dimensionExpr[dim]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnknownDimension, dim)
	}

	q := "SELECT " + expr + " AS label, COUNT(DISTINCT visitor_id), COUNT(*) " +
		base + " GROUP BY label ORDER BY COUNT(*) DESC NULLS LAST"
	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("visitoractivity: breakdown %q: %w", dim, err)
	}
	defer rows.Close()

	var stats []DimensionStat
	for rows.Next() {
		var (
			label *string // read-side exprs (utm_*) can yield NULL → treat as ""
			st    DimensionStat
		)
		if err := rows.Scan(&label, &st.UniqueViews, &st.TotalHits); err != nil {
			return nil, fmt.Errorf("visitoractivity: scan breakdown %q: %w", dim, err)
		}
		if label != nil {
			st.Label = *label
		}
		stats = append(stats, st)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return suppressSmallCounts(stats, defaultSuppressThreshold), nil
}

// suppressSmallCounts folds every bucket whose UniqueViews is below threshold
// into a single "Other" row (Decision D3 — statistical-disclosure guard so the
// API never emits a size-1 cell). Buckets at/above the threshold are kept as-is;
// the Other row carries the summed unique + total of all folded buckets. Result
// is sorted by TotalHits desc with Other pushed last, so the visible ranking is
// stable and the catch-all sits at the bottom.
func suppressSmallCounts(stats []DimensionStat, threshold int) []DimensionStat {
	var kept []DimensionStat
	var otherUnique, otherTotal int
	for _, st := range stats {
		if st.UniqueViews < threshold {
			otherUnique += st.UniqueViews
			otherTotal += st.TotalHits
			continue
		}
		kept = append(kept, st)
	}
	sort.SliceStable(kept, func(i, j int) bool {
		return kept[i].TotalHits > kept[j].TotalHits
	})
	if otherTotal > 0 {
		kept = append(kept, DimensionStat{Label: otherLabel, UniqueViews: otherUnique, TotalHits: otherTotal})
	}
	return kept
}

// Overview — see Store. Excludes staff and bots.
func (s *postgresStore) Overview(ctx context.Context, since time.Time, dims ...string) (*Overview, error) {
	// Validate the requested dimensions BEFORE any SQL (injection guard).
	wantDims, err := resolveDimensions(dims)
	if err != nil {
		return nil, err
	}

	ov := &Overview{ByEventType: map[string]int{}, Breakdowns: map[string][]DimensionStat{}}

	// Workspace-wide base: all event types, staff/bots excluded. The breakdown
	// helper appends its GROUP BY to this exact predicate.
	base := fmt.Sprintf(`FROM %s.visitor_activity
		WHERE created_at >= $1 AND is_staff = false AND is_bot = false`, s.schema)
	args := []any{since}

	totalsQ := "SELECT COALESCE(COUNT(*),0), COALESCE(COUNT(DISTINCT visitor_id),0) " + base
	if err := s.pool.QueryRow(ctx, totalsQ, args...).Scan(&ov.TotalHits, &ov.UniqueVisitors); err != nil {
		return nil, fmt.Errorf("visitoractivity: overview totals: %w", err)
	}

	byTypeQ := "SELECT event_type, COUNT(*) " + base + " GROUP BY event_type"
	rows, err := s.pool.Query(ctx, byTypeQ, args...)
	if err != nil {
		return nil, fmt.Errorf("visitoractivity: overview by-type: %w", err)
	}
	for rows.Next() {
		var et string
		var n int
		if err := rows.Scan(&et, &n); err != nil {
			rows.Close()
			return nil, fmt.Errorf("visitoractivity: scan by-type: %w", err)
		}
		ov.ByEventType[et] = n
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// By day (UTC) — mirrors PropertyStats.ByDay: unique visitors + total hits
	// per UTC day across all event types, excluding staff/bots.
	byDayQ := "SELECT date_trunc('day', created_at) AS day, COUNT(DISTINCT visitor_id), COUNT(*) " +
		base + " GROUP BY day ORDER BY day"
	if ov.ByDay, err = s.scanDays(ctx, byDayQ, args); err != nil {
		return nil, fmt.Errorf("visitoractivity: overview by-day: %w", err)
	}

	// Generic breakdowns — same helper as PropertyStats, workspace-wide base.
	for _, dim := range wantDims {
		stats, err := s.breakdown(ctx, base, args, dim)
		if err != nil {
			return nil, err
		}
		ov.Breakdowns[dim] = stats
	}

	return ov, nil
}

// nullIfEmpty converts an empty string to a SQL NULL (so referer_origin stays
// NULL rather than ” when there is no referer).
func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
