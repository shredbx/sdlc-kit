package rss

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"
)

// errNoStore is returned by Refresh/NewerCount when no Store is wired — these
// are the persisted-feed entry points and need a backend to write/read.
var errNoStore = errors.New("feed: no store configured (use rss.WithStore)")

// Refresh fetches every enabled source (conditional GET via the persisted
// validators), parses it, and UpsertByGUID-imports its items — deduplicated on
// (source_id, guid). It is the import side of the persisted-feed split (List is
// the read side).
//
// Partial-serve: a source that errors, times out, or returns 304 never aborts
// the run; its outcome is recorded via SetSourceFetchState and (for errors)
// collected into the RefreshReport. The returned error is non-nil ONLY when no
// Store is wired — a per-source failure is reported, not returned.
//
// Idempotent: re-running over an unchanged feed inserts zero new rows (every
// item dedups to an in-place update) — the (source_id, guid) unique key holds.
func (s *Service) Refresh(ctx context.Context) (RefreshReport, error) {
	if s.store == nil {
		return RefreshReport{}, errNoStore
	}

	sources, err := s.store.EnabledSources(ctx)
	if err != nil {
		return RefreshReport{}, fmt.Errorf("feed: load enabled sources: %w", err)
	}

	report := RefreshReport{Sources: len(sources), Errors: []RefreshError{}}
	now := time.Now()

	for _, src := range sources {
		s.refreshOne(ctx, src, &report, now)
	}

	return report, nil
}

// RefreshSource runs the SAME fetch+parse+import path as Refresh for exactly one
// enabled source (matched by ID) — the per-source "fetch now" admin trigger. The
// returned RefreshReport.Sources is 1 when the source was found+enabled, else 0.
// A disabled or unknown source is NOT an error (Sources=0, no Errors) — only a
// missing Store or an EnabledSources load failure returns a non-nil error.
func (s *Service) RefreshSource(ctx context.Context, sourceID string) (RefreshReport, error) {
	if s.store == nil {
		return RefreshReport{}, errNoStore
	}

	sources, err := s.store.EnabledSources(ctx)
	if err != nil {
		return RefreshReport{}, fmt.Errorf("feed: load enabled sources: %w", err)
	}

	report := RefreshReport{Errors: []RefreshError{}}
	now := time.Now()
	for _, src := range sources {
		if src.ID != sourceID {
			continue
		}
		report.Sources = 1
		s.refreshOne(ctx, src, &report, now)
		break
	}
	return report, nil
}

// refreshOne fetches+parses+imports a single source and folds the outcome into
// report (partial-serve: a per-source failure is recorded, never returned). It
// is the shared body of Refresh (all sources) and RefreshSource (one source).
func (s *Service) refreshOne(ctx context.Context, src FeedSource, report *RefreshReport, now time.Time) {
	items, status, etag, lastModified, ferr := s.refreshFetch(ctx, src)
	if ferr != nil {
		report.Errors = append(report.Errors, RefreshError{Source: src.ID, Status: ferr.Error()})
		_ = s.store.SetSourceFetchState(ctx, src.ID, errorStatus(ferr), src.ETag, src.LastModified, now)
		log.Printf("feed: refresh source %q error: %v", src.ID, ferr)
		return
	}

	report.Fetched++

	// 304 / no body to import: keep validators, count as skipped.
	if status == statusNotModified {
		report.Skipped++
		_ = s.store.SetSourceFetchState(ctx, src.ID, string(statusNotModified), etag, lastModified, now)
		return
	}

	for _, it := range items {
		// P3: resolve the best image (incl. og:image fallback) and cache it to R2
		// in the background. resolveImage returns the resolved upstream URL (the
		// one persisted, so idempotency holds across runs) and the cached R2 URL.
		// Per-item failure is non-fatal — the item is still imported, just without
		// a cached thumbnail. No-op when no cacher is wired (R2 URL stays "").
		it.ImageURL, it.ImageR2URL = s.resolveAndCacheImage(ctx, it)

		inserted, uerr := s.store.UpsertItemByGUID(ctx, it)
		if uerr != nil {
			report.Errors = append(report.Errors, RefreshError{Source: src.ID, Status: uerr.Error()})
			log.Printf("feed: upsert item %q/%q: %v", src.ID, it.GUID, uerr)
			continue
		}
		if inserted {
			report.Imported++
		} else {
			report.Deduped++
		}
	}

	_ = s.store.SetSourceFetchState(ctx, src.ID, string(statusOK), etag, lastModified, now)
}

// refreshFetch fetches+parses one source using its PERSISTED conditional-GET
// validators (from the DB), returning the parsed items plus the fresh validators
// and status. On 304 it returns statusNotModified and no items.
func (s *Service) refreshFetch(ctx context.Context, src FeedSource) (items []FeedItem, status fetchStatus, etag, lastModified string, err error) {
	parser, perr := ParserFor(src.Parser)
	if perr != nil {
		return nil, "", "", "", perr
	}

	res, ferr := s.fetcher.Fetch(ctx, src.URL, src.ETag, src.LastModified)
	if ferr != nil {
		return nil, "", "", "", ferr
	}

	if res.NotModified {
		return nil, statusNotModified, res.ETag, res.LastModified, nil
	}

	parsed, parseErr := parser.Parse(res.Body, src)
	if parseErr != nil {
		return nil, "", "", "", parseErr
	}
	return parsed, statusOK, res.ETag, res.LastModified, nil
}

// NewerCount reports how many visible items match q and were published strictly
// after since — the incremental-reveal count. Requires a Store.
func (s *Service) NewerCount(ctx context.Context, since time.Time, q Query) (int, error) {
	if s.store == nil {
		return 0, errNoStore
	}
	return s.store.NewerCount(ctx, since, q)
}

// Validate probes a source via the SAME Fetcher + parser path as Refresh (no new
// fetch logic) and reports reachability, item count, sample titles, image
// coverage, parser status, and the source's language/category hints — so a
// manager can assess a feed before enabling it (research-plan validator_contract).
//
// It never returns an error for an unreachable or malformed source; that outcome
// is captured in the ValidationResult (Reachable=false, Error set). The returned
// error is reserved for an unknown parser kind (a misconfiguration).
func (s *Service) Validate(ctx context.Context, src FeedSource) (ValidationResult, error) {
	out := ValidationResult{
		LanguageHint: src.Language,
		CategoryHint: src.Category,
		SampleTitles: []string{},
	}

	parser, perr := ParserFor(src.Parser)
	if perr != nil {
		return out, perr
	}

	// Probe with no conditional validators so we always get a body to inspect.
	res, ferr := s.fetcher.Fetch(ctx, src.URL, "", "")
	if ferr != nil {
		out.Error = ferr.Error()
		out.HTTPStatus = httpStatusFromFetchError(ferr)
		return out, nil
	}

	out.Reachable = true
	out.HTTPStatus = 200

	parsed, parseErr := parser.Parse(res.Body, src)
	if parseErr != nil {
		out.Error = parseErr.Error()
		return out, nil
	}

	out.ParserOK = true
	out.ItemCount = len(parsed)
	for _, it := range parsed {
		if it.ImageURL != "" {
			out.ItemsWithImageCount++
		}
		if len(out.SampleTitles) < validationSampleSize && it.Title != "" {
			out.SampleTitles = append(out.SampleTitles, it.Title)
		}
	}
	return out, nil
}

// httpStatusFromFetchError extracts the upstream HTTP status from a fetch error
// when it is a *StatusError (a non-2xx response); a transport-level error (no
// HTTP response at all) yields 0.
func httpStatusFromFetchError(err error) int {
	var se *StatusError
	if errors.As(err, &se) {
		return se.StatusCode
	}
	return 0
}
