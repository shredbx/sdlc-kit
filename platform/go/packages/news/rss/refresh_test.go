package rss_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/shredbx/sbx-core/pkg/rss"
)

// stubStore is an in-memory rss.Store for the engine refresh/list unit tests.
// It records upserts (dedup on source_id+guid), serves visible items newest-first,
// and captures the per-source fetch state the Refresh run writes.
type stubStore struct {
	items     map[string]rss.FeedItem // key: sourceID + "\x00" + guid
	order     []string                // insertion order of keys (for stable newest-first when times tie)
	fetch     map[string]stubFetchState
	enabled   []rss.FeedSource
	upsertErr error
}

type stubFetchState struct {
	status       string
	etag         string
	lastModified string
	fetchedAt    time.Time
}

func newStubStore(enabled ...rss.FeedSource) *stubStore {
	return &stubStore{
		items:   map[string]rss.FeedItem{},
		fetch:   map[string]stubFetchState{},
		enabled: enabled,
	}
}

func key(sourceID, guid string) string { return sourceID + "\x00" + guid }

func (s *stubStore) UpsertItemByGUID(_ context.Context, it rss.FeedItem) (bool, error) {
	if s.upsertErr != nil {
		return false, s.upsertErr
	}
	k := key(it.SourceID, it.GUID)
	if _, ok := s.items[k]; ok {
		s.items[k] = it // update in place — no duplicate
		return false, nil
	}
	s.items[k] = it
	s.order = append(s.order, k)
	return true, nil
}

func (s *stubStore) ListItems(_ context.Context, q rss.Query) (rss.Result, error) {
	// Apply the query filters as a DB WHERE would, in insertion order.
	srcSet := map[string]bool{}
	for _, id := range q.Sources {
		srcSet[id] = true
	}
	var out []rss.FeedItem
	for _, k := range s.order {
		it := s.items[k]
		if len(srcSet) > 0 && !srcSet[it.SourceID] {
			continue
		}
		if q.Category.Valid() && it.Category != q.Category {
			continue
		}
		if q.Language.Valid() && it.Language != q.Language {
			continue
		}
		out = append(out, it)
	}
	return rss.Result{Items: out, Page: 1, HasNext: false}, nil
}

func (s *stubStore) NewerCount(_ context.Context, since time.Time, _ rss.Query) (int, error) {
	n := 0
	for _, it := range s.items {
		if it.PublishedAt.After(since) {
			n++
		}
	}
	return n, nil
}

func (s *stubStore) SetSourceFetchState(_ context.Context, sourceID, status, etag, lastModified string, fetchedAt time.Time) error {
	s.fetch[sourceID] = stubFetchState{status: status, etag: etag, lastModified: lastModified, fetchedAt: fetchedAt}
	return nil
}

func (s *stubStore) EnabledSources(_ context.Context) ([]rss.FeedSource, error) {
	return s.enabled, nil
}

func (s *stubStore) ItemImageState(_ context.Context, sourceID, guid string) (rss.ItemImageState, bool, error) {
	it, ok := s.items[key(sourceID, guid)]
	if !ok {
		return rss.ItemImageState{}, false, nil
	}
	return rss.ItemImageState{ImageURL: it.ImageURL, ImageR2URL: it.ImageR2URL}, true, nil
}

func (s *stubStore) count() int { return len(s.items) }

// errorTransport returns a non-2xx status for the configured URLs so a source
// errors mid-run (partial-serve proof), while others fetch normally.
type errorTransport struct {
	base   fixtureTransport
	failAt map[string]int // url -> status code to return (e.g. 402)
}

func (t errorTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if code, ok := t.failAt[req.URL.String()]; ok {
		return &http.Response{StatusCode: code, Body: http.NoBody, Header: http.Header{}}, nil
	}
	return t.base.RoundTrip(req)
}

// TC-01 (SC-NEWS-13) — Refresh imports items and is idempotent: a first run
// inserts every item; a second run over the SAME feed inserts ZERO (dedup on
// (source_id, guid)) and updates in place.
func TestRefresh_UpsertByGUID_Idempotent(t *testing.T) {
	store := newStubStore(bangkokpostSource) // 3 items in the fixture
	svc := rss.NewService(clientFor(newTransport()), rss.NewMemoryCache(), nil,
		rss.WithStore(store))

	rep1, err := svc.Refresh(context.Background())
	if err != nil {
		t.Fatalf("Refresh #1: %v", err)
	}
	if rep1.Imported != 3 {
		t.Errorf("first run imported = %d, want 3", rep1.Imported)
	}
	if store.count() != 3 {
		t.Errorf("store has %d items after first run, want 3", store.count())
	}

	rep2, err := svc.Refresh(context.Background())
	if err != nil {
		t.Fatalf("Refresh #2: %v", err)
	}
	if rep2.Imported != 0 {
		t.Errorf("second run imported = %d, want 0 (idempotent dedup)", rep2.Imported)
	}
	if rep2.Deduped != 3 {
		t.Errorf("second run deduped = %d, want 3", rep2.Deduped)
	}
	if store.count() != 3 {
		t.Errorf("store has %d items after second run, want 3 (no duplicates)", store.count())
	}
}

// TC-02 (SC-NEWS-14) — partial-serve: one source erroring (402) does NOT abort
// the run; the other source still imports and the error is collected in the
// report with the failing source + status.
func TestRefresh_PartialServe_SourceError(t *testing.T) {
	tr := errorTransport{
		base:   newTransport(),
		failAt: map[string]int{bangkokpostSource.URL: http.StatusPaymentRequired},
	}
	store := newStubStore(bangkokpostSource, thaigerSource) // bkk fails, thaiger imports
	svc := rss.NewService(clientFor(tr), rss.NewMemoryCache(), nil, rss.WithStore(store))

	rep, err := svc.Refresh(context.Background())
	if err != nil {
		t.Fatalf("Refresh must not error on partial failure: %v", err)
	}
	if rep.Imported != 3 {
		t.Errorf("imported = %d, want 3 (thaiger still imports)", rep.Imported)
	}
	if len(rep.Errors) != 1 {
		t.Fatalf("errors = %d, want 1", len(rep.Errors))
	}
	if rep.Errors[0].Source != bangkokpostSource.ID {
		t.Errorf("error source = %q, want %q", rep.Errors[0].Source, bangkokpostSource.ID)
	}
	// The erroring source's fetch state is recorded as an error.
	if st := store.fetch[bangkokpostSource.ID].status; st == "" || st == "ok" {
		t.Errorf("bkk fetch status = %q, want an error status", st)
	}
	// The working source's fetch state is ok.
	if st := store.fetch[thaigerSource.ID].status; st != "ok" {
		t.Errorf("thaiger fetch status = %q, want ok", st)
	}
}

// TC-14 (R-validator) — Validate probes a source via the SAME fetch+parse path
// and reports reachability, item count, sample titles, image coverage, parser_ok.
func TestValidate_Probe(t *testing.T) {
	svc := rss.NewService(clientFor(newTransport()), rss.NewMemoryCache(), nil)

	res, err := svc.Validate(context.Background(), thaigerSource)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if !res.Reachable {
		t.Errorf("reachable = false, want true")
	}
	if res.HTTPStatus != http.StatusOK {
		t.Errorf("httpStatus = %d, want 200", res.HTTPStatus)
	}
	if res.ItemCount != 3 {
		t.Errorf("itemCount = %d, want 3", res.ItemCount)
	}
	if !res.ParserOK {
		t.Errorf("parserOK = false, want true")
	}
	if len(res.SampleTitles) == 0 {
		t.Errorf("sampleTitles empty, want some")
	}
	if res.LanguageHint != thaigerSource.Language {
		t.Errorf("languageHint = %q, want %q", res.LanguageHint, thaigerSource.Language)
	}
	if res.CategoryHint != thaigerSource.Category {
		t.Errorf("categoryHint = %q, want %q", res.CategoryHint, thaigerSource.Category)
	}
}

// TC-14b — Validate reports an unreachable / error source without erroring the call.
func TestValidate_Unreachable(t *testing.T) {
	tr := errorTransport{base: newTransport(), failAt: map[string]int{bangkokpostSource.URL: http.StatusPaymentRequired}}
	svc := rss.NewService(clientFor(tr), rss.NewMemoryCache(), nil)

	res, err := svc.Validate(context.Background(), bangkokpostSource)
	if err != nil {
		t.Fatalf("Validate must report, not error: %v", err)
	}
	if res.Reachable {
		t.Errorf("reachable = true, want false for a 402 source")
	}
	if res.HTTPStatus != http.StatusPaymentRequired {
		t.Errorf("httpStatus = %d, want 402", res.HTTPStatus)
	}
}
