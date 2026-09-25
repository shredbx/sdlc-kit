package rss_test

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shredbx/sbx-core/pkg/rss"
)

// fixtureTransport maps an upstream URL to either a fixture file's bytes or an
// error, so a real *http.Client can drive the Service offline.
type fixtureTransport struct {
	byURL map[string]string // url -> testdata filename
	errAt map[string]bool   // url -> return a transport error
}

func (t fixtureTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	url := req.URL.String()
	if t.errAt[url] {
		return nil, &transportError{url}
	}
	name, ok := t.byURL[url]
	if !ok {
		return &http.Response{StatusCode: 404, Body: io.NopCloser(strings.NewReader("")), Header: http.Header{}}, nil
	}
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		return nil, err
	}
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(string(data))),
		Header:     http.Header{},
	}, nil
}

type transportError struct{ url string }

func (e *transportError) Error() string { return "transport error for " + e.url }

func clientFor(tr http.RoundTripper) *http.Client {
	return &http.Client{Transport: tr}
}

// allSources is the four-source registry used across service tests.
func allSources() []rss.FeedSource {
	return []rss.FeedSource{
		bangkokpostSource, // en / property / rss2  (3 items, 2026-05-28, 25, 03)
		thaigerSource,     // en / property / wp     (3 items, 2026-05-22, 20, 18)
		prachachatSource,  // th / property / wp     (3 items, 2026-05-27, 26, 25)
		khaosodSource,     // th / general  / wp     (3 items, 2026-05-28, 27, 27)
	}
}

func newTransport() fixtureTransport {
	return fixtureTransport{byURL: map[string]string{
		bangkokpostSource.URL: "bangkokpost-property.xml",
		thaigerSource.URL:     "thethaiger-property.xml",
		prachachatSource.URL:  "prachachat-property.xml",
		khaosodSource.URL:     "khaosod.xml",
	}}
}

func TestService_MergeSortDesc(t *testing.T) {
	svc := rss.NewService(clientFor(newTransport()), rss.NewMemoryCache(), allSources())
	res, err := svc.List(context.Background(), rss.Query{PageSize: 100})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(res.Items) != 12 {
		t.Fatalf("got %d items, want 12 (4 sources × 3)", len(res.Items))
	}
	// Newest first: each item's PublishedAt >= the next.
	for i := 1; i < len(res.Items); i++ {
		if res.Items[i-1].PublishedAt.Before(res.Items[i].PublishedAt) {
			t.Errorf("not sorted desc at %d: %v before %v",
				i, res.Items[i-1].PublishedAt, res.Items[i].PublishedAt)
		}
	}
}

func TestService_FilterByCategoryAndLanguage(t *testing.T) {
	svc := rss.NewService(clientFor(newTransport()), rss.NewMemoryCache(), allSources())
	res, err := svc.List(context.Background(), rss.Query{
		Category: rss.CategoryProperty,
		Language: rss.LanguageEN,
		PageSize: 100,
	})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	// Property+EN = Bangkok Post (3) + Thaiger (3) = 6.
	if len(res.Items) != 6 {
		t.Fatalf("got %d items, want 6 (property+en)", len(res.Items))
	}
	for _, it := range res.Items {
		if it.Category != rss.CategoryProperty || it.Language != rss.LanguageEN {
			t.Errorf("item out of filter: cat=%q lang=%q", it.Category, it.Language)
		}
	}
}

func TestService_FilterBySource(t *testing.T) {
	svc := rss.NewService(clientFor(newTransport()), rss.NewMemoryCache(), allSources())
	res, err := svc.List(context.Background(), rss.Query{
		Sources:  []string{"khaosod"},
		PageSize: 100,
	})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(res.Items) != 3 {
		t.Fatalf("got %d items, want 3 (khaosod only)", len(res.Items))
	}
	for _, it := range res.Items {
		if it.SourceID != "khaosod" {
			t.Errorf("item from wrong source: %q", it.SourceID)
		}
	}
}

func TestService_Pagination_HasNext(t *testing.T) {
	svc := rss.NewService(clientFor(newTransport()), rss.NewMemoryCache(), allSources())

	page1, err := svc.List(context.Background(), rss.Query{Page: 1, PageSize: 5})
	if err != nil {
		t.Fatalf("List p1: %v", err)
	}
	if len(page1.Items) != 5 {
		t.Errorf("page1 size = %d, want 5", len(page1.Items))
	}
	if !page1.HasNext {
		t.Error("page1 should have a next page (12 items, size 5)")
	}
	if page1.Page != 1 {
		t.Errorf("page1.Page = %d, want 1", page1.Page)
	}

	page3, err := svc.List(context.Background(), rss.Query{Page: 3, PageSize: 5})
	if err != nil {
		t.Fatalf("List p3: %v", err)
	}
	if len(page3.Items) != 2 { // 12 - 5 - 5
		t.Errorf("page3 size = %d, want 2", len(page3.Items))
	}
	if page3.HasNext {
		t.Error("page3 should be the last page")
	}
}

func TestService_DefaultPageSize(t *testing.T) {
	svc := rss.NewService(clientFor(newTransport()), rss.NewMemoryCache(), allSources())
	res, err := svc.List(context.Background(), rss.Query{}) // no page/size
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if res.Page != 1 {
		t.Errorf("default page = %d, want 1", res.Page)
	}
	// 12 items, default size 20 → all on one page, no next.
	if len(res.Items) != 12 || res.HasNext {
		t.Errorf("default page got %d items hasNext=%v, want 12/false", len(res.Items), res.HasNext)
	}
}

func TestService_PartialServe_OneBadSourceAmongGood(t *testing.T) {
	tr := newTransport()
	// Make prachachat return malformed XML; the other 3 must still serve.
	tr.byURL[prachachatSource.URL] = "malformed.xml"

	svc := rss.NewService(clientFor(tr), rss.NewMemoryCache(), allSources())
	res, err := svc.List(context.Background(), rss.Query{PageSize: 100})
	if err != nil {
		t.Fatalf("partial-serve should not error when some sources work: %v", err)
	}
	// 3 good sources × 3 = 9 (prachachat dropped).
	if len(res.Items) != 9 {
		t.Fatalf("got %d items, want 9 (prachachat dropped)", len(res.Items))
	}
	for _, it := range res.Items {
		if it.SourceID == "prachachat-property" {
			t.Error("malformed source should contribute no items")
		}
	}
}

func TestService_PartialServe_TransportErrorSkipped(t *testing.T) {
	tr := newTransport()
	tr.errAt = map[string]bool{thaigerSource.URL: true}

	svc := rss.NewService(clientFor(tr), rss.NewMemoryCache(), allSources())
	res, err := svc.List(context.Background(), rss.Query{PageSize: 100})
	if err != nil {
		t.Fatalf("transport error on one source should not fail List: %v", err)
	}
	if len(res.Items) != 9 { // thaiger (3) dropped
		t.Fatalf("got %d items, want 9 (thaiger errored)", len(res.Items))
	}
}

func TestService_AllSourcesFail_ReturnsError(t *testing.T) {
	tr := fixtureTransport{
		byURL: map[string]string{},
		errAt: map[string]bool{
			bangkokpostSource.URL: true,
			thaigerSource.URL:     true,
			prachachatSource.URL:  true,
			khaosodSource.URL:     true,
		},
	}
	svc := rss.NewService(clientFor(tr), rss.NewMemoryCache(), allSources())
	_, err := svc.List(context.Background(), rss.Query{PageSize: 100})
	if err == nil {
		t.Error("List should return an error when ALL sources fail")
	}
}

func TestService_AllEmpty_ReturnsEmptyResult(t *testing.T) {
	emptySrc := bangkokpostSource
	emptySrc.URL = "https://empty.test/feed"
	tr := fixtureTransport{byURL: map[string]string{emptySrc.URL: "empty-feed.xml"}}

	svc := rss.NewService(clientFor(tr), rss.NewMemoryCache(), []rss.FeedSource{emptySrc})
	res, err := svc.List(context.Background(), rss.Query{PageSize: 100})
	if err != nil {
		t.Fatalf("empty (but valid) feed should not error: %v", err)
	}
	if len(res.Items) != 0 {
		t.Errorf("got %d items, want 0", len(res.Items))
	}
	if res.HasNext {
		t.Error("empty result should not have a next page")
	}
}

func TestService_DisabledSourceSkipped(t *testing.T) {
	srcs := allSources()
	srcs[3].Enabled = false // disable khaosod

	svc := rss.NewService(clientFor(newTransport()), rss.NewMemoryCache(), srcs)
	res, err := svc.List(context.Background(), rss.Query{PageSize: 100})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(res.Items) != 9 { // khaosod (3) excluded
		t.Fatalf("got %d items, want 9 (khaosod disabled)", len(res.Items))
	}
	for _, it := range res.Items {
		if it.SourceID == "khaosod" {
			t.Error("disabled source contributed items")
		}
	}
}

func TestService_Sources_ReturnsRegistry(t *testing.T) {
	srcs := allSources()
	srcs[3].Enabled = false
	svc := rss.NewService(clientFor(newTransport()), rss.NewMemoryCache(), srcs)
	got := svc.Sources()
	if len(got) != 4 {
		t.Fatalf("Sources() = %d, want 4 (includes disabled)", len(got))
	}
}

func TestService_UnknownFilterIgnored(t *testing.T) {
	svc := rss.NewService(clientFor(newTransport()), rss.NewMemoryCache(), allSources())
	// An out-of-range category that is not a valid dictionary value must be
	// treated as "no category filter" — never a 500 / empty surprise.
	res, err := svc.List(context.Background(), rss.Query{
		Category: rss.FeedCategory("sports"),
		PageSize: 100,
	})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(res.Items) != 12 {
		t.Errorf("invalid category should not filter; got %d, want 12", len(res.Items))
	}
}

// TC-05 (SC-NEWS-02) — when a Store is wired, List reads from the Store (DB),
// not upstream RSS, and the source/category/language filters are honored. The
// stub store applies the filters as a DB WHERE would.
func TestList_Filters_SourceCategoryLanguage(t *testing.T) {
	store := newStubStore()
	seed := []rss.FeedItem{
		{SourceID: "s-prop-en", GUID: "1", Title: "Prop EN", Category: rss.CategoryProperty, Language: rss.LanguageEN},
		{SourceID: "s-prop-th", GUID: "2", Title: "Prop TH", Category: rss.CategoryProperty, Language: rss.LanguageTH},
		{SourceID: "s-biz-en", GUID: "3", Title: "Biz EN", Category: rss.CategoryBusiness, Language: rss.LanguageEN},
	}
	for _, it := range seed {
		if _, err := store.UpsertItemByGUID(context.Background(), it); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	// A store-backed Service: no live fetch, List delegates to the Store.
	svc := rss.NewService(nil, rss.NewMemoryCache(), nil, rss.WithStore(store))

	// Category filter.
	res, err := svc.List(context.Background(), rss.Query{Category: rss.CategoryProperty})
	if err != nil {
		t.Fatalf("List category: %v", err)
	}
	if len(res.Items) != 2 {
		t.Errorf("category=property → %d items, want 2", len(res.Items))
	}

	// Language filter.
	res, err = svc.List(context.Background(), rss.Query{Language: rss.LanguageEN})
	if err != nil {
		t.Fatalf("List language: %v", err)
	}
	if len(res.Items) != 2 {
		t.Errorf("lang=en → %d items, want 2", len(res.Items))
	}

	// Source filter.
	res, err = svc.List(context.Background(), rss.Query{Sources: []string{"s-biz-en"}})
	if err != nil {
		t.Fatalf("List source: %v", err)
	}
	if len(res.Items) != 1 {
		t.Errorf("source=s-biz-en → %d items, want 1", len(res.Items))
	}

	// Combined category + language.
	res, err = svc.List(context.Background(), rss.Query{Category: rss.CategoryProperty, Language: rss.LanguageEN})
	if err != nil {
		t.Fatalf("List combined: %v", err)
	}
	if len(res.Items) != 1 {
		t.Errorf("property+en → %d items, want 1", len(res.Items))
	}
}
