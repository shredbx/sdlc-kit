package repository

import "context"

// SearchOptions configures a full-text search query with optional structured filters.
// Query is free-text input (passed to websearch_to_tsquery on PostgreSQL).
// Filter provides standard predicate-based narrowing alongside the text search.
type SearchOptions struct {
	Query  string      // free-text search input
	Filter Query       // structured predicate filters (reuses repository.Query)
	Sort   []SortField // ordering (relevance is default when Query is non-empty)
	Limit  int
	Offset int
}

// SearchTier identifies which strategy actually returned results. The pipeline
// runs tiers in order (prefix → fuzzy → ilike) and stops on the first non-empty
// match; TierUsed records which one fired so callers can communicate match
// quality to end users.
type SearchTier string

const (
	TierNone   SearchTier = ""       // no text query supplied
	TierPrefix SearchTier = "prefix" // tier 1: prefix tsquery (`token:*`)
	TierFuzzy  SearchTier = "fuzzy"  // tier 2: word_similarity per-field
	TierIlike  SearchTier = "ilike"  // tier 3: ILIKE substring (last resort)
)

// SearchSection classifies a single result by WHERE the query matched, so the
// caller can split results into provenance buckets (Decision #0272 Phase 2):
//   - SectionMatch — the query hit a structured / high-weight field (title,
//     property_type, location, amenity label, …). Surfaced as "Matches".
//   - SectionAlso — the query matched ONLY the free-text description (the sole
//     weight-D field after Phase 1 re-weighting). Surfaced as "Also mentioned".
type SearchSection string

const (
	SectionMatch SearchSection = "match" // hit a structured/high-weight field
	SectionAlso  SearchSection = "also"  // matched only the free-text description
)

// SearchResult holds the outcome of a Searcher.Search call.
//
// Sections is a parallel slice to Items (same order + length) classifying each
// result by match provenance. It is backward-compatible: consumers that only
// read Items ignore it. Empty/nil when no text query was supplied (a non-text
// list has no provenance to report).
type SearchResult[M any] struct {
	Items     []M
	Sections  []SearchSection // parallel to Items: per-result match provenance
	Total     int
	TierUsed  SearchTier // which tier produced these results
	UsedFuzzy bool       // back-compat alias: TierUsed == TierFuzzy
}

// Searcher performs ranked full-text search with optional structured filters.
// Separate from Repository[M] because full-text search requires operators
// (tsvector @@, trigram similarity) that the predicate Op enum cannot express.
type Searcher[M any] interface {
	Search(ctx context.Context, opts SearchOptions) (SearchResult[M], error)
}
