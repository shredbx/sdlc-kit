package repository

// ListOptions configures a List query with filtering, sorting, and pagination.
type ListOptions struct {
	// Filter is a composable query tree. Empty filter matches everything.
	Filter Query

	// Sort specifies ordering. Applied in order (primary, secondary, ...).
	Sort []SortField

	// Limit is the maximum number of items to return. 0 means use default.
	Limit int

	// Offset is the number of items to skip (for pagination).
	Offset int

	// OnlyDeleted flips the soft-delete gate: instead of the default
	// deleted_at IS NULL, List matches ONLY soft-deleted (archived) rows —
	// the listing behind an "Archived" tab with restore (2607-102). Zero value
	// keeps today's behavior byte-identical for every existing consumer.
	// Facet counters (GroupCount/RangeCounts) stay live-only regardless.
	OnlyDeleted bool
}

// RangeBucket labels a sub-population of a faceted count: every row matching
// Where is tallied under Label. Buckets are evaluated against the SAME base
// WHERE (ListOptions.Filter) as the owning facet, so a set of overlapping
// thresholds (e.g. bedrooms >= 1, >= 2, …) yields cumulative counts in ONE
// query. Where is compiled by the identical QueryBuilder as the list filter, so
// a bucket can never drift from the list semantics it refines.
type RangeBucket struct {
	// Label is the bucket key in the returned map (e.g. "1", "2", …).
	Label string

	// Where is the bucket-specific predicate tree, AND-ed onto the facet's base
	// filter at count time. An empty Where counts the whole base population.
	Where Query
}

// SortField specifies a field to sort by and its direction.
type SortField struct {
	// Field is the logical field name (mapped to column by Mapper).
	Field string

	// Desc is true for descending order, false for ascending.
	Desc bool
}

// Asc creates an ascending sort field.
func Asc(field string) SortField {
	return SortField{Field: field, Desc: false}
}

// Desc creates a descending sort field.
func DescSort(field string) SortField {
	return SortField{Field: field, Desc: true}
}
