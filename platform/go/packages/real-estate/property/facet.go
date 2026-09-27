package property

import (
	"context"

	"github.com/shredbx/sbx-core/pkg/repository"
)

// GroupCount delegates to the repository's optional GroupCounter capability
// (the PostgresStore implements it). When the wired repo does not support
// grouped counts (e.g. the in-memory YAML store used in handler unit tests),
// it degrades gracefully to an empty map + nil error — callers treat absent
// facets as "no counts available", never a failure.
//
// The opts.Filter passed here is the SAME repository.Query the list path
// compiles, so the facet WHERE and the list WHERE stay in lockstep.
func (s *PropertyService) GroupCount(ctx context.Context, opts repository.ListOptions, groupField string) (map[string]int, error) {
	gc, ok := s.repo.(repository.GroupCounter)
	if !ok {
		return map[string]int{}, nil
	}
	return gc.GroupCount(ctx, opts, groupField)
}

// RangeCounts delegates to the repository's optional RangeCounter capability,
// degrading gracefully to an empty map when the repo does not support it.
func (s *PropertyService) RangeCounts(ctx context.Context, opts repository.ListOptions, buckets []repository.RangeBucket) (map[string]int, error) {
	rc, ok := s.repo.(repository.RangeCounter)
	if !ok {
		return map[string]int{}, nil
	}
	return rc.RangeCounts(ctx, opts, buckets)
}
