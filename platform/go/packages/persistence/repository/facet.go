package repository

import "context"

// GroupCounter is the optional faceted-count capability: tally COUNT(*) grouped
// by a single logical field, scoped by ListOptions.Filter. It is an OPTIONAL
// capability (not part of Repository[M]) so in-memory / non-grouping backends
// stay unaffected — consumers type-assert and degrade gracefully when absent.
//
// The filter passed in is the SAME repository.Query the list path compiles, so
// the facet WHERE and the list WHERE are produced by identical machinery and
// can never drift. Callers achieve "exclude-own-dimension" faceting by omitting
// the grouped dimension's predicate from opts.Filter and supplying it as the
// groupField instead.
//
// SECURITY CONTRACT: groupField (and RangeBucket.Where below) name a COLUMN, and
// column identifiers CANNOT be bound as SQL parameters — backends interpolate
// them into the statement. They MUST therefore be developer-supplied constants
// (or values validated against a field allow-list), NEVER request-derived input.
// All VALUES remain parameter-bound; only the column identifier is interpolated.
type GroupCounter interface {
	// GroupCount returns a map of group-value → COUNT(*) for rows matching
	// opts.Filter (plus the backend's own soft-delete gate), grouped by the
	// column that groupField maps to. Rows whose group value is NULL are
	// omitted (they have no bucket). opts.Sort / Limit / Offset are ignored.
	GroupCount(ctx context.Context, opts ListOptions, groupField string) (map[string]int, error)
}

// RangeCounter is the optional bucketed-count capability: tally COUNT(*) for
// each RangeBucket in ONE query (via FILTER), all sharing opts.Filter as the
// base WHERE. Like GroupCounter it is OPTIONAL and reuses the same QueryBuilder,
// so bucket predicates can never drift from list semantics.
type RangeCounter interface {
	// RangeCounts returns a map of bucket.Label → COUNT(*) for rows matching
	// opts.Filter AND the bucket's Where (plus the backend soft-delete gate),
	// computed in a single pass. opts.Sort / Limit / Offset are ignored.
	RangeCounts(ctx context.Context, opts ListOptions, buckets []RangeBucket) (map[string]int, error)
}
