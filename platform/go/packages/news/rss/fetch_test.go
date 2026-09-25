package rss_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/shredbx/sbx-core/pkg/rss"
)

// stubDoer records the last request and returns a scripted response.
type stubDoer struct {
	resp    *http.Response
	err     error
	lastReq *http.Request
}

func (d *stubDoer) Do(req *http.Request) (*http.Response, error) {
	d.lastReq = req
	return d.resp, d.err
}

func makeResp(status int, body string, headers map[string]string) *http.Response {
	h := http.Header{}
	for k, v := range headers {
		h.Set(k, v)
	}
	return &http.Response{
		StatusCode: status,
		Header:     h,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestFetcher_200ReturnsBodyAndValidators(t *testing.T) {
	doer := &stubDoer{resp: makeResp(200, "<rss/>", map[string]string{
		"ETag":          `"abc123"`,
		"Last-Modified": "Wed, 27 May 2026 12:00:00 GMT",
	})}
	f := rss.NewFetcher(doer)

	res, err := f.Fetch(context.Background(), "https://x.test/feed", "", "")
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if res.NotModified {
		t.Error("200 should not be NotModified")
	}
	if string(res.Body) != "<rss/>" {
		t.Errorf("body = %q", res.Body)
	}
	if res.ETag != `"abc123"` {
		t.Errorf("etag = %q", res.ETag)
	}
	if res.LastModified != "Wed, 27 May 2026 12:00:00 GMT" {
		t.Errorf("lastModified = %q", res.LastModified)
	}
}

func TestFetcher_SendsConditionalHeaders(t *testing.T) {
	doer := &stubDoer{resp: makeResp(304, "", nil)}
	f := rss.NewFetcher(doer)

	_, err := f.Fetch(context.Background(), "https://x.test/feed", `"etag-1"`, "Mon, 25 May 2026 00:00:00 GMT")
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if got := doer.lastReq.Header.Get("If-None-Match"); got != `"etag-1"` {
		t.Errorf("If-None-Match = %q, want \"etag-1\"", got)
	}
	if got := doer.lastReq.Header.Get("If-Modified-Since"); got != "Mon, 25 May 2026 00:00:00 GMT" {
		t.Errorf("If-Modified-Since = %q", got)
	}
}

func TestFetcher_304IsNotModified(t *testing.T) {
	doer := &stubDoer{resp: makeResp(304, "", nil)}
	f := rss.NewFetcher(doer)

	res, err := f.Fetch(context.Background(), "https://x.test/feed", `"etag-1"`, "")
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if !res.NotModified {
		t.Error("304 should set NotModified=true")
	}
	if len(res.Body) != 0 {
		t.Errorf("304 body should be empty, got %q", res.Body)
	}
}

func TestFetcher_Non2xxIsError(t *testing.T) {
	doer := &stubDoer{resp: makeResp(500, "boom", nil)}
	f := rss.NewFetcher(doer)
	if _, err := f.Fetch(context.Background(), "https://x.test/feed", "", ""); err == nil {
		t.Error("5xx should return an error")
	}
}

func TestFetcher_CapsBodySize(t *testing.T) {
	big := strings.Repeat("A", 1024)
	doer := &stubDoer{resp: makeResp(200, big, nil)}
	f := rss.NewFetcher(doer)
	f.MaxBytes = 100

	res, err := f.Fetch(context.Background(), "https://x.test/feed", "", "")
	// An oversized body must be REJECTED with an error (never silently truncated
	// into invalid XML, never read in full).
	if err == nil {
		t.Fatalf("oversized body (1024B, cap 100) must error; got nil err, %d body bytes", len(res.Body))
	}
	if bytes.Equal(res.Body, []byte(big)) {
		t.Error("oversized body was read in full despite the cap")
	}
}

func TestDefaultHTTPClient_HasTimeout(t *testing.T) {
	c := rss.DefaultHTTPClient()
	if c == nil {
		t.Fatal("DefaultHTTPClient returned nil")
	}
	if c.Timeout <= 0 {
		t.Error("DefaultHTTPClient should set a positive Timeout")
	}
}
