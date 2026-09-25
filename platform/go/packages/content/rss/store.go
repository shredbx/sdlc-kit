package rss

import (
	"context"
	"time"
)

// Store is the storage-agnostic persistence backend the engine writes imported
// items into and reads the public listing from. It mirrors the Cache interface
// style (Rule #9): the shared engine never hard-couples to PostgreSQL — a
// consumer (e.g. BR) injects a postgres-backed adapter via WithStore; tests
// inject an in-memory stub. With a Store wired, Refresh persists items and List
// reads them from the DB (zero upstream RSS on a page load); without one, the
// engine keeps its legacy live-fetch List behavior.
type Store interface {
	// UpsertItemByGUID inserts a new item or updates the existing row matched on
	// (SourceID, GUID). inserted is true only when a new row was created (so a
	// refresh run can count imported-vs-deduped). The dedup key is the natural
	// (source_id, guid) unique constraint.
	UpsertItemByGUID(ctx context.Context, item FeedItem) (inserted bool, err error)

	// ListItems returns a filtered, newest-first, paginated page of VISIBLE items
	// (hidden excluded) honoring the Query's source/category/language filters.
	ListItems(ctx context.Context, q Query) (Result, error)

	// NewerCount reports how many visible items match the Query and were published
	// strictly after since — the incremental-reveal count.
	NewerCount(ctx context.Context, since time.Time, q Query) (int, error)

	// SetSourceFetchState records the outcome of a refresh fetch for one source:
	// the conditional-GET validators (etag/lastModified) to replay next run, a
	// human-readable status (ok | error:<msg> | not-modified), and the fetch time.
	SetSourceFetchState(ctx context.Context, sourceID, status, etag, lastModified string, fetchedAt time.Time) error

	// EnabledSources returns the sources a refresh run should fetch (enabled only),
	// each carrying its persisted conditional-GET validators.
	EnabledSources(ctx context.Context) ([]FeedSource, error)

	// ItemImageState returns the currently persisted image state for one item
	// (matched on (sourceID, guid)) so the refresh job can decide whether to
	// re-cache: found is false when the item is new. It powers the P3 idempotency
	// rule — skip re-upload when the R2 thumbnail already exists and the upstream
	// image URL is unchanged. A storage backend with no items yet returns
	// found=false, never an error.
	ItemImageState(ctx context.Context, sourceID, guid string) (state ItemImageState, found bool, err error)
}

// ItemImageState is the persisted image bookkeeping for one item, used by the
// refresh job's idempotency check (re-cache only when the R2 thumbnail is absent
// or the upstream image URL changed).
type ItemImageState struct {
	// ImageURL is the upstream image URL last persisted for the item ("" if none).
	ImageURL string
	// ImageR2URL is the cached R2 thumbnail URL last persisted ("" if not cached).
	ImageR2URL string
}

// fetchStatus is the small, dictionary-like vocabulary SetSourceFetchState
// records. It is a named type (never a raw string) so the status set is explicit
// and consumers can branch on it without string literals scattered across code.
type fetchStatus string

const (
	// statusOK marks a successful fetch+parse for a source.
	statusOK fetchStatus = "ok"
	// statusNotModified marks a 304 conditional-GET reuse.
	statusNotModified fetchStatus = "not-modified"
)

// errorStatus formats the per-source error status (error:<msg>) recorded when a
// source fails mid-run; partial-serve keeps the run going and surfaces this.
func errorStatus(err error) string {
	return "error:" + err.Error()
}

// RefreshError is one source's failure within a partial-serve Refresh run.
type RefreshError struct {
	// Source is the failing source's ID.
	Source string `json:"source"`
	// Status is a short outcome string (e.g. "402", "timeout", "parse error").
	Status string `json:"status"`
}

// RefreshReport summarizes one Refresh run (usage-spec §3 counts). A source that
// errors is recorded in Errors but never aborts the run (partial-serve).
type RefreshReport struct {
	// Sources is the number of enabled sources the run attempted.
	Sources int `json:"sources"`
	// Fetched is the number of sources that fetched successfully (incl. 304).
	Fetched int `json:"fetched"`
	// Imported is the number of NEW items inserted across all sources.
	Imported int `json:"imported"`
	// Deduped is the number of already-present items updated in place.
	Deduped int `json:"deduped"`
	// Skipped is the number of sources skipped (e.g. 304 not-modified, no body).
	Skipped int `json:"skipped"`
	// Errors collects per-source failures (partial-serve never aborts the run).
	Errors []RefreshError `json:"errors"`
}

// ValidationResult is the probe report for a single source (research-plan
// validator_contract): it reuses the SAME Fetcher+parser path as Refresh and
// lets a manager assess a feed before enabling it.
type ValidationResult struct {
	// Reachable is true when the fetch returned a usable 2xx (or 304) response.
	Reachable bool `json:"reachable"`
	// HTTPStatus is the upstream HTTP status code (0 on a transport error).
	HTTPStatus int `json:"httpStatus"`
	// ItemCount is the number of items the parser yielded.
	ItemCount int `json:"itemCount"`
	// SampleTitles is up to validationSampleSize item titles for eyeballing.
	SampleTitles []string `json:"sampleTitles"`
	// ItemsWithImageCount is how many items carried a non-empty ImageURL.
	ItemsWithImageCount int `json:"itemsWithImageCount"`
	// ParserOK is true when the configured parser decoded the body without error.
	ParserOK bool `json:"parserOk"`
	// LanguageHint echoes the source's assigned language (the engine does not
	// auto-detect per item — out of P1 scope).
	LanguageHint FeedLanguage `json:"languageHint"`
	// CategoryHint echoes the source's assigned category.
	CategoryHint FeedCategory `json:"categoryHint"`
	// Error is a human-readable failure note when the probe could not complete
	// (transport error or parse error); empty on success.
	Error string `json:"error,omitempty"`
}

// validationSampleSize caps SampleTitles so a Validate probe stays small.
const validationSampleSize = 5
