package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/shredbx/sbx-core/pkg/database"
	"github.com/shredbx/sbx-core/pkg/repository"
)

// PostgresSearcher implements Searcher[M] with a 3-tier text search pipeline:
//
//  1. Prefix tsquery — `to_tsquery('simple', 'tok1:* & tok2:*')`, ranked by
//     ts_rank_cd. Handles "Agricul" → "Agricultural" via lexeme prefix.
//  2. Fuzzy — per-field `word_similarity()` (title, area, city, province)
//     above trgmThreshold. Handles typos and substring fragments better than
//     the previous concat-then-similarity approach because the trigram score
//     isn't diluted by long concatenated strings.
//  3. ILIKE — substring case-insensitive last-resort. Catches queries that
//     literally exist in the data but missed both tsvector lexemes and the
//     trigram threshold.
//
// Tiers run in order; the first non-empty tier wins and is recorded in
// SearchResult.TierUsed so callers can show appropriate UI feedback.
type PostgresSearcher[M any] struct {
	db            *database.DB
	mapper        Mapper[M]
	trgmThreshold float64
}

// TextSearchable lets a Mapper drive the text-search tiers. Mappers that do
// NOT implement it keep the legacy property-shaped defaults (alias p, title +
// loc.* fuzzy fields, p.search_vector) so existing behavior is unchanged.
//
//   - SearchVectorColumn — the qualified tsvector column the Tier-1 prefix
//     tsquery matches against (e.g. "p.search_vector").
//   - FuzzyTextFields — the qualified text columns the Tier-2 fuzzy
//     (word_similarity) and Tier-3 (ILIKE) tiers scan; the FIRST field also
//     drives the Tier-3 ORDER BY position(). Order matters for that tiebreak.
type TextSearchable interface {
	SearchVectorColumn() string
	FuzzyTextFields() []string
}

// legacyFuzzyFields is the historical property-shaped fuzzy/ilike field set used
// when a Mapper does NOT implement TextSearchable. `loc.sub_district` is the
// post-2605-095 rename (was loc.area). Keep in EXACT sync with the property
// search trigger + index so non-implementing mappers see zero behavior change.
var legacyFuzzyFields = []string{"p.title", "loc.sub_district", "loc.city", "loc.province"}

// resolveTextSearch returns the search-vector column + fuzzy text fields the
// text-search tiers should use for the given mapper. A mapper implementing
// TextSearchable drives both; any other mapper falls back to the legacy
// property shape (p.search_vector + the 4 hardcoded property/location fields).
func resolveTextSearch[M any](mapper Mapper[M]) (searchVector string, fuzzyFields []string) {
	if ts, ok := any(mapper).(TextSearchable); ok {
		return ts.SearchVectorColumn(), ts.FuzzyTextFields()
	}
	return "p.search_vector", legacyFuzzyFields
}

// buildFuzzyScoreExpr builds the Tier-2 fuzzy score expression — a
// word_similarity() per field, wrapped in GREATEST() when there are 2+ fields.
// idx is the bound parameter position holding the query text. A single field
// is emitted bare (GREATEST of one arg is needless and some planners reject it).
func buildFuzzyScoreExpr(fields []string, idx int) string {
	parts := make([]string, len(fields))
	for i, f := range fields {
		parts[i] = fmt.Sprintf("word_similarity($%d, COALESCE(%s, ''))", idx, f)
	}
	if len(parts) == 1 {
		return parts[0]
	}
	// Indentation matches the original property Tier-2 SQL byte-for-byte (4 tabs
	// per arg, 3 tabs before the closing paren) so non-implementing mappers see
	// no SQL change. Whitespace is semantically irrelevant to Postgres, but
	// keeping it identical makes the no-regression guarantee obvious in diffs.
	return "GREATEST(\n\t\t\t\t" + strings.Join(parts, ",\n\t\t\t\t") + "\n\t\t\t)"
}

// buildIlikeCond builds the Tier-3 ILIKE OR-chain over the fuzzy text fields.
// idx is the bound parameter position holding the query text.
func buildIlikeCond(fields []string, idx int) string {
	parts := make([]string, len(fields))
	for i, f := range fields {
		parts[i] = fmt.Sprintf("COALESCE(%s, '') ILIKE '%%' || $%d || '%%'", f, idx)
	}
	// Indentation matches the original property Tier-3 SQL byte-for-byte (leading
	// newline, 4-tab clauses) so non-implementing mappers see no SQL change.
	return "(\n\t\t\t\t" + strings.Join(parts, "\n\t\t\t\tOR ") + "\n\t\t\t)"
}

// NewPostgresSearcher creates a PostgreSQL-backed Searcher[M].
// trgmThreshold controls when fuzzy results are included (0.3 is the
// workspace default — tuned for short prefix-like queries).
func NewPostgresSearcher[M any](db *database.DB, mapper Mapper[M], trgmThreshold float64) *PostgresSearcher[M] {
	if trgmThreshold <= 0 {
		trgmThreshold = 0.3
	}
	return &PostgresSearcher[M]{db: db, mapper: mapper, trgmThreshold: trgmThreshold}
}

func (s *PostgresSearcher[M]) Search(ctx context.Context, opts repository.SearchOptions) (repository.SearchResult[M], error) {
	selectFrom := s.mapper.SelectFrom()
	cols := strings.Join(s.mapper.Columns(), ", ")
	fieldMapper := s.mapper.FieldColumn

	qb := NewQueryBuilder()
	baseFilter := repository.And(repository.Predicate{
		FieldName: "deleted_at",
		Op:        repository.OpIsNull,
	})

	var combinedFilter repository.Query
	if opts.Filter.IsEmpty() {
		combinedFilter = baseFilter
	} else {
		combinedFilter = repository.Query{
			And: []repository.Query{baseFilter, opts.Filter},
		}
	}

	whereClause, whereArgs := qb.Build(combinedFilter, fieldMapper)

	if strings.TrimSpace(opts.Query) != "" {
		return s.searchWithText(ctx, opts, selectFrom, cols, fieldMapper, whereClause, whereArgs)
	}

	return s.searchWithoutText(ctx, opts, selectFrom, cols, fieldMapper, whereClause, whereArgs)
}

// searchWithText runs the 3-tier pipeline. Each tier's args slice is
// freshly-built from the structured-filter args + that tier's text params, so
// arg indices line up with the bound parameter positions in the SQL.
func (s *PostgresSearcher[M]) searchWithText(
	ctx context.Context,
	opts repository.SearchOptions,
	selectFrom, cols string,
	fieldMapper func(string) string,
	whereClause string,
	whereArgs []any,
) (repository.SearchResult[M], error) {
	// Tier 1 — prefix tsquery. Unlike tiers 2/3, this tier searches the full
	// search_vector, which includes the free-text description (the sole weight-D
	// field after Phase-1 re-weighting). So Tier-1 results need per-row match
	// provenance: a row whose hit is description-only is "also mentioned", any
	// row hitting a structured field (title/property_type/location/amenity) is a
	// real "match". structRankExpr re-ranks each row with the description (D)
	// weight zeroed — > 0 means a structured field matched (Decision #0272).
	// Resolve the text-search shape: a TextSearchable mapper drives the
	// search-vector column + fuzzy fields; any other mapper keeps the legacy
	// property defaults (p.search_vector + title/loc.*) — zero behavior change.
	searchVector, fuzzyFields := resolveTextSearch[M](s.mapper)

	tsq := buildPrefixTsquery(opts.Query)
	if tsq != "" {
		result, ok, err := s.tryTierClassified(
			ctx, opts, selectFrom, cols, whereClause, whereArgs,
			func(idx int) (cond string, extra []any, order string, structRank string) {
				cond = fmt.Sprintf("%s @@ to_tsquery('simple', $%d)", searchVector, idx)
				order = fmt.Sprintf("ts_rank_cd(%s, to_tsquery('simple', $%d)) DESC", searchVector, idx)
				// D=0 zeroes the description/body contribution — a positive value means a
				// structured (weight C/B/A) field matched. Weight array is {D,C,B,A}
				// (Decision #0272), unchanged across entities.
				structRank = fmt.Sprintf("ts_rank_cd('{0,0.2,0.4,1.0}'::float4[], %s, to_tsquery('simple', $%d))", searchVector, idx)
				return cond, []any{tsq}, order, structRank
			},
			repository.TierPrefix,
		)
		if err != nil {
			return result, err
		}
		if ok {
			return result, nil
		}
	}

	// Tier 2 — per-field word_similarity (fuzzy) over the resolved fuzzy fields.
	// For the legacy property mapper these are title + loc.* (loc.sub_district is
	// the post-2605-095 rename of loc.area); a TextSearchable mapper supplies its
	// own (e.g. cms content_entries → just p.title).
	result, ok, err := s.tryTier(
		ctx, opts, selectFrom, cols, whereClause, whereArgs,
		func(idx int) (string, []any, string) {
			scoreExpr := buildFuzzyScoreExpr(fuzzyFields, idx)
			cond := fmt.Sprintf("%s > $%d", scoreExpr, idx+1)
			order := fmt.Sprintf("%s DESC", scoreExpr)
			return cond, []any{opts.Query, s.trgmThreshold}, order
		},
		repository.TierFuzzy,
	)
	if err != nil {
		return result, err
	}
	if ok {
		return result, nil
	}

	// Tier 3 — ILIKE substring (last resort)
	result, ok, err = s.tryTier(
		ctx, opts, selectFrom, cols, whereClause, whereArgs,
		func(idx int) (string, []any, string) {
			// ILIKE fallback over the resolved fuzzy fields. The ORDER BY uses the
			// FIRST fuzzy field for position() (title for both property + cms).
			cond := buildIlikeCond(fuzzyFields, idx)
			order := fmt.Sprintf("position(LOWER($%d) IN LOWER(COALESCE(%s, ''))) ASC NULLS LAST", idx, fuzzyFields[0])
			return cond, []any{opts.Query}, order
		},
		repository.TierIlike,
	)
	if err != nil {
		return result, err
	}
	if ok {
		return result, nil
	}

	// No tier matched — return empty (TierUsed records last attempt = ilike).
	result.TierUsed = repository.TierIlike
	if result.Items == nil {
		result.Items = make([]M, 0)
	}
	return result, nil
}

// tryTier runs one tier and returns (result, hasItems, error). The buildCond
// closure takes the next free $N param index and returns:
//   - the SQL WHERE condition fragment
//   - the extra args to append (in $N, $N+1, … order)
//   - the ORDER BY expression
func (s *PostgresSearcher[M]) tryTier(
	ctx context.Context,
	opts repository.SearchOptions,
	selectFrom, cols string,
	baseWhere string,
	baseArgs []any,
	buildCond func(startIdx int) (cond string, extra []any, order string),
	tier repository.SearchTier,
) (repository.SearchResult[M], bool, error) {
	var result repository.SearchResult[M]

	startIdx := len(baseArgs) + 1
	cond, extra, order := buildCond(startIdx)
	args := append(append([]any{}, baseArgs...), extra...)

	var fullWhere string
	if baseWhere != "" {
		fullWhere = baseWhere + " AND " + cond
	} else {
		fullWhere = cond
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s", selectFrom, fullWhere)
	var total int
	if err := s.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return result, false, fmt.Errorf("searcher tier=%s count: %w", tier, err)
	}
	if total == 0 {
		return result, false, nil
	}

	dataQuery := fmt.Sprintf("SELECT %s FROM %s WHERE %s ORDER BY %s", cols, selectFrom, fullWhere, order)
	if opts.Limit > 0 {
		dataQuery += fmt.Sprintf(" LIMIT %d", opts.Limit)
	}
	if opts.Offset > 0 {
		dataQuery += fmt.Sprintf(" OFFSET %d", opts.Offset)
	}

	items, err := s.queryItems(ctx, dataQuery, args)
	if err != nil {
		return result, false, err
	}

	result.Items = items
	// Tiers 2 (fuzzy) and 3 (ilike) search only structured fields
	// (title + location), never the description — so every hit is a real
	// "match" (Decision #0272).
	result.Sections = make([]repository.SearchSection, len(items))
	for i := range result.Sections {
		result.Sections[i] = repository.SectionMatch
	}
	result.Total = total
	result.TierUsed = tier
	result.UsedFuzzy = tier == repository.TierFuzzy
	return result, true, nil
}

// tryTierClassified is tryTier for the Tier-1 path: in addition to running the
// tier, it computes a per-row struct_rank (the buildCond closure's 4th return)
// and classifies each item — SectionMatch when struct_rank > 0 (a structured
// field matched), SectionAlso when struct_rank == 0 (description-only hit).
// Ordering is preserved by appending struct_rank as a trailing SELECT column
// scanned via a wrapper that strips it before the mapper sees the row.
func (s *PostgresSearcher[M]) tryTierClassified(
	ctx context.Context,
	opts repository.SearchOptions,
	selectFrom, cols string,
	baseWhere string,
	baseArgs []any,
	buildCond func(startIdx int) (cond string, extra []any, order string, structRank string),
	tier repository.SearchTier,
) (repository.SearchResult[M], bool, error) {
	var result repository.SearchResult[M]

	startIdx := len(baseArgs) + 1
	cond, extra, order, structRank := buildCond(startIdx)
	args := append(append([]any{}, baseArgs...), extra...)

	var fullWhere string
	if baseWhere != "" {
		fullWhere = baseWhere + " AND " + cond
	} else {
		fullWhere = cond
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s", selectFrom, fullWhere)
	var total int
	if err := s.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return result, false, fmt.Errorf("searcher tier=%s count: %w", tier, err)
	}
	if total == 0 {
		return result, false, nil
	}

	// Append struct_rank as a trailing column; the scan wrapper captures it.
	dataQuery := fmt.Sprintf("SELECT %s, %s AS struct_rank FROM %s WHERE %s ORDER BY %s",
		cols, structRank, selectFrom, fullWhere, order)
	if opts.Limit > 0 {
		dataQuery += fmt.Sprintf(" LIMIT %d", opts.Limit)
	}
	if opts.Offset > 0 {
		dataQuery += fmt.Sprintf(" OFFSET %d", opts.Offset)
	}

	items, sections, err := s.queryItemsClassified(ctx, dataQuery, args)
	if err != nil {
		return result, false, err
	}

	result.Items = items
	result.Sections = sections
	result.Total = total
	result.TierUsed = tier
	result.UsedFuzzy = tier == repository.TierFuzzy
	return result, true, nil
}

// buildPrefixTsquery converts free-text input into a safe tsquery string.
// Returns empty string when input has no usable tokens.
//
// Sanitization rules:
//   - whitespace splits tokens
//   - each tsquery operator (& | ! ( ) : ' " \) is stripped from tokens
//   - empty/short-after-strip tokens are dropped
//   - each surviving token gets `:*` appended (lexeme prefix match)
//   - tokens are joined with ` & ` (AND)
//
// This makes the result safe to pass as a parameterized to_tsquery argument:
// no operators leak from user input, and parameterization handles SQL injection
// at the driver level. Caller MUST still bind it via $N (never concat).
func buildPrefixTsquery(q string) string {
	tokens := strings.Fields(q)
	if len(tokens) == 0 {
		return ""
	}

	const stripChars = `&|!():'"\\`
	cleaned := make([]string, 0, len(tokens))
	for _, t := range tokens {
		for _, c := range stripChars {
			t = strings.ReplaceAll(t, string(c), "")
		}
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		cleaned = append(cleaned, t+":*")
	}
	if len(cleaned) == 0 {
		return ""
	}
	return strings.Join(cleaned, " & ")
}

func (s *PostgresSearcher[M]) searchWithoutText(
	ctx context.Context,
	opts repository.SearchOptions,
	selectFrom, cols string,
	fieldMapper func(string) string,
	whereClause string,
	whereArgs []any,
) (repository.SearchResult[M], error) {
	var result repository.SearchResult[M]

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s", selectFrom)
	if whereClause != "" {
		countQuery += " WHERE " + whereClause
	}

	var total int
	if err := s.db.QueryRow(ctx, countQuery, whereArgs...).Scan(&total); err != nil {
		return result, fmt.Errorf("searcher count: %w", err)
	}

	dataQuery := fmt.Sprintf("SELECT %s FROM %s", cols, selectFrom)
	if whereClause != "" {
		dataQuery += " WHERE " + whereClause
	}

	dataQuery += BuildOrderBy(opts.Sort, fieldMapper)

	if opts.Limit > 0 {
		dataQuery += fmt.Sprintf(" LIMIT %d", opts.Limit)
	}
	if opts.Offset > 0 {
		dataQuery += fmt.Sprintf(" OFFSET %d", opts.Offset)
	}

	items, err := s.queryItems(ctx, dataQuery, whereArgs)
	if err != nil {
		return result, err
	}

	result.Items = items
	result.Total = total
	return result, nil
}

// queryItemsClassified runs a Tier-1 data query whose SELECT ends in a trailing
// `struct_rank` column, returning the items alongside a parallel Sections slice.
// The mapper's FromRow scans only its own columns, so each row's scan is wrapped:
// the wrapper appends a *float32 for struct_rank to the dest list, runs the real
// scan, then classifies (struct_rank > 0 → match, else → also). The mapper never
// sees the extra column.
func (s *PostgresSearcher[M]) queryItemsClassified(ctx context.Context, query string, args []any) ([]M, []repository.SearchSection, error) {
	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("searcher query: %w", err)
	}
	defer rows.Close()

	var items []M
	var sections []repository.SearchSection
	for rows.Next() {
		var structRank float32
		item, err := s.mapper.FromRow(func(dest ...any) error {
			return rows.Scan(append(dest, &structRank)...)
		})
		if err != nil {
			return nil, nil, fmt.Errorf("searcher scan: %w", err)
		}
		items = append(items, item)
		if structRank > 0 {
			sections = append(sections, repository.SectionMatch)
		} else {
			sections = append(sections, repository.SectionAlso)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("searcher rows: %w", err)
	}

	if items == nil {
		items = make([]M, 0)
	}
	if sections == nil {
		sections = make([]repository.SearchSection, 0)
	}

	return items, sections, nil
}

func (s *PostgresSearcher[M]) queryItems(ctx context.Context, query string, args []any) ([]M, error) {
	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("searcher query: %w", err)
	}
	defer rows.Close()

	var items []M
	for rows.Next() {
		item, err := s.mapper.FromRow(rows.Scan)
		if err != nil {
			return nil, fmt.Errorf("searcher scan: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("searcher rows: %w", err)
	}

	if items == nil {
		items = make([]M, 0)
	}

	return items, nil
}
