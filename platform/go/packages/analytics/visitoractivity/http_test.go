package visitoractivity_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	va "github.com/shredbx/sbx-core/pkg/visitoractivity"
)

// httpStore is a fake Store for the HTTP-handler tests. It captures appended
// Events (for the no-PII assertion) and returns canned aggregation data so the
// insights handlers have a deterministic shape to serialize.
type httpStore struct {
	last *va.Event

	top         []va.TargetStat
	topLimit    int // records the limit the handler asked TopProperties for (over-fetch assertion)
	overview    *va.Overview
	property    *va.PropertyStats
	pathStats   *va.PropertyStats
	pathSummary va.PathSummary
}

func (s *httpStore) Append(_ context.Context, e *va.Event) error {
	cp := *e
	s.last = &cp
	return nil
}

func (s *httpStore) TopProperties(_ context.Context, _ time.Time, limit int) ([]va.TargetStat, error) {
	s.topLimit = limit
	return s.top, nil
}

func (s *httpStore) PropertyStats(context.Context, uuid.UUID, time.Time, ...string) (*va.PropertyStats, error) {
	if s.property == nil {
		return &va.PropertyStats{}, nil
	}
	return s.property, nil
}

func (s *httpStore) Overview(context.Context, time.Time, ...string) (*va.Overview, error) {
	if s.overview == nil {
		return &va.Overview{ByEventType: map[string]int{}}, nil
	}
	return s.overview, nil
}

func (s *httpStore) PathStats(context.Context, string, time.Time, ...string) (*va.PropertyStats, error) {
	if s.pathStats == nil {
		return &va.PropertyStats{}, nil
	}
	return s.pathStats, nil
}

func (s *httpStore) PathSummary(context.Context, string) (va.PathSummary, error) {
	return s.pathSummary, nil
}

// newHTTPService wires a Service over an httpStore + a MemorySaltProvider (a
// real, deterministic-per-day salt — no DB, no Redis).
func newHTTPService() (*va.Service, *httpStore) {
	store := &httpStore{}
	svc := va.NewService(store, va.NewMemorySaltProvider(nil))
	return svc, store
}

// defaultTrackDeps returns deps that produce a fixed IP, non-staff classification.
func defaultTrackDeps() va.TrackDeps {
	return va.TrackDeps{
		ClientIP: func(*http.Request) string { return "203.0.113.50" },
		IsStaff:  func(*http.Request) bool { return false },
	}
}

func postTrack(t *testing.T, h http.HandlerFunc, contentType, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/track", strings.NewReader(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X)")
	rec := httptest.NewRecorder()
	h(rec, req)
	return rec
}

// TC1: a valid event is recorded and the captured Event carries NO raw IP and
// NO raw user-agent — only the derived VisitorID/DeviceType/IsBot. The Event
// type has no IP/UA fields by design; this asserts the serialized row never
// leaks the raw values the handler fed into derivation.
func TestTrackHandler_RecordsNoPII(t *testing.T) {
	svc, store := newHTTPService()
	const rawIP = "198.51.100.23"
	const rawUA = "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) Secret-UA-Token"

	deps := va.TrackDeps{
		ClientIP: func(*http.Request) string { return rawIP },
		IsStaff:  func(*http.Request) bool { return false },
	}
	h := va.TrackHandler(svc, deps)

	tid := uuid.New()
	body := `{"type":"property_view","target":"property","target_id":"` + tid.String() + `","path":"/properties/abc"}`
	req := httptest.NewRequest(http.MethodPost, "/track", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", rawUA)
	rec := httptest.NewRecorder()
	h(rec, req)

	require.Equal(t, http.StatusNoContent, rec.Code)
	require.NotNil(t, store.last, "Record must have appended an Event")

	// The derived visitor id must equal the HMAC, never the raw IP/UA.
	assert.NotEqual(t, rawIP, store.last.VisitorID)
	assert.NotContains(t, store.last.VisitorID, rawIP)
	assert.NotContains(t, store.last.VisitorID, rawUA)

	// Whole-Event invariant: neither the raw IP nor the raw UA may appear
	// anywhere in the serialized row.
	raw, err := json.Marshal(store.last)
	require.NoError(t, err)
	assert.NotContains(t, string(raw), rawIP, "serialized Event must not contain the raw IP")
	assert.NotContains(t, string(raw), rawUA, "serialized Event must not contain the raw user-agent")

	// The handler must never echo the row to the client (204 = empty body).
	assert.Empty(t, rec.Body.String(), "track handler must not echo the recorded row")
}

// TC3: status mapping — valid → 204; invalid event type → 422; oversized props
// (>2KB marshaled) → 413.
func TestTrackHandler_204_422_413(t *testing.T) {
	svc, _ := newHTTPService()
	h := va.TrackHandler(svc, defaultTrackDeps())
	tid := uuid.New()

	// valid → 204
	valid := `{"type":"property_view","target":"property","target_id":"` + tid.String() + `","path":"/p"}`
	rec := postTrack(t, h, "application/json", valid)
	assert.Equal(t, http.StatusNoContent, rec.Code)

	// invalid event type → 422
	bad := `{"type":"bogus","target":"property","target_id":"` + tid.String() + `","path":"/p"}`
	rec = postTrack(t, h, "application/json", bad)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)

	// oversized props (>2KB marshaled) → 413
	big := strings.Repeat("x", 3000)
	oversized := `{"type":"property_view","target":"property","target_id":"` + tid.String() +
		`","path":"/p","props":{"blob":"` + big + `"}}`
	rec = postTrack(t, h, "application/json", oversized)
	assert.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)

	// valid page event (path-keyed, NO target_id) → 204
	validPage := `{"type":"page_view","target":"page","path":"/about"}`
	rec = postTrack(t, h, "application/json", validPage)
	assert.Equal(t, http.StatusNoContent, rec.Code)

	// page event carrying a stray target_id → 422 (a malformed-client request, the
	// same 400-class failure as a property event missing its target_id — must NOT
	// be swallowed as a 204 via the internal-error arm).
	pageWithTargetID := `{"type":"page_view","target":"page","target_id":"` + tid.String() + `","path":"/about"}`
	rec = postTrack(t, h, "application/json", pageWithTargetID)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)

	// page event with no path → 422 (path is the page identifier).
	pageNoPath := `{"type":"page_view","target":"page"}`
	rec = postTrack(t, h, "application/json", pageNoPath)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

// TC3b: text/plain beacons are parsed (defensive: sendBeacon-style content type).
func TestTrackHandler_AcceptsTextPlain(t *testing.T) {
	svc, store := newHTTPService()
	h := va.TrackHandler(svc, defaultTrackDeps())
	tid := uuid.New()
	body := `{"type":"property_view","target":"property","target_id":"` + tid.String() + `","path":"/p"}`
	rec := postTrack(t, h, "text/plain", body)
	assert.Equal(t, http.StatusNoContent, rec.Code)
	require.NotNil(t, store.last)
	assert.Equal(t, va.EventPropertyView, store.last.EventType)
}

// TC7: insights handlers parse the period whitelist and serialize the correct
// JSON shape from stub data; a bad period (5d) → 422.
func TestInsightsHandlers_PeriodAndShape(t *testing.T) {
	svc, store := newHTTPService()
	propID := uuid.New()
	store.top = []va.TargetStat{{TargetID: propID, UniqueViews: 7, TotalHits: 12}}
	store.overview = &va.Overview{
		TotalHits:      12,
		UniqueVisitors: 7,
		ByEventType:    map[string]int{"property_view": 12},
		ByDay:          []va.DayStat{{Day: time.Now(), UniqueViews: 7, TotalHits: 12}},
	}
	store.property = &va.PropertyStats{
		UniqueViews: 7,
		TotalHits:   12,
		ByDay:       []va.DayStat{{Day: time.Now(), UniqueViews: 7, TotalHits: 12}},
		Breakdowns: map[string][]va.DimensionStat{
			"device": {{Label: "mobile", UniqueViews: 5, TotalHits: 9}},
		},
	}

	deps := va.InsightsDeps{
		PropertyID: func(*http.Request) (uuid.UUID, error) { return propID, nil },
	}
	top, overview, property, _, _ := va.InsightsHandlers(svc, deps)

	// top — valid period → 200, []TargetStat shape
	rec := httptest.NewRecorder()
	top(rec, httptest.NewRequest(http.MethodGet, "/insights/properties/top?period=7d&limit=10", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	var gotTop []va.TargetStat
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &gotTop))
	require.Len(t, gotTop, 1)
	assert.Equal(t, propID, gotTop[0].TargetID)
	assert.Equal(t, 7, gotTop[0].UniqueViews)

	// overview — valid period → 200, Overview shape (incl ByDay)
	rec = httptest.NewRecorder()
	overview(rec, httptest.NewRequest(http.MethodGet, "/insights/overview?period=30d", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	var gotOv va.Overview
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &gotOv))
	assert.Equal(t, 12, gotOv.TotalHits)
	assert.Equal(t, 7, gotOv.UniqueVisitors)
	assert.Len(t, gotOv.ByDay, 1)

	// property — valid period → 200, PropertyStats shape
	rec = httptest.NewRecorder()
	property(rec, httptest.NewRequest(http.MethodGet, "/insights/properties/"+propID.String()+"?period=90d", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	var gotPS va.PropertyStats
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &gotPS))
	assert.Equal(t, 7, gotPS.UniqueViews)
	assert.Len(t, gotPS.Breakdowns["device"], 1)

	// today — valid period (start-of-UTC-day window) → 200 (2606-073)
	rec = httptest.NewRecorder()
	overview(rec, httptest.NewRequest(http.MethodGet, "/insights/overview?period=today", nil))
	require.Equal(t, http.StatusOK, rec.Code)

	// bad period (5d) → 422 on every handler
	for name, h := range map[string]http.HandlerFunc{"top": top, "overview": overview, "property": property} {
		rec = httptest.NewRecorder()
		h(rec, httptest.NewRequest(http.MethodGet, "/insights?period=5d", nil))
		assert.Equalf(t, http.StatusUnprocessableEntity, rec.Code, "%s with bad period must be 422", name)
	}
}

// TC9: the factories invoke the injected Deps — proves the wiring is reusable
// without app coupling. Custom ClientIP/IsStaff/PropertyID closures must be
// called.
func TestHandlerFactories_InjectedDeps(t *testing.T) {
	svc, store := newHTTPService()

	var ipCalled, staffCalled, propIDCalled bool
	const injectedIP = "192.0.2.99"

	trackDeps := va.TrackDeps{
		ClientIP: func(*http.Request) string { ipCalled = true; return injectedIP },
		IsStaff:  func(*http.Request) bool { staffCalled = true; return true },
	}
	th := va.TrackHandler(svc, trackDeps)

	tid := uuid.New()
	body := `{"type":"property_view","target":"property","target_id":"` + tid.String() + `","path":"/p"}`
	rec := postTrack(t, th, "application/json", body)
	require.Equal(t, http.StatusNoContent, rec.Code)
	assert.True(t, ipCalled, "TrackDeps.ClientIP must be invoked")
	assert.True(t, staffCalled, "TrackDeps.IsStaff must be invoked")
	require.NotNil(t, store.last)
	assert.True(t, store.last.IsStaff, "IsStaff result must flow into the recorded Event")
	firstVisitorID := store.last.VisitorID

	// The injected IP must drive the visitor id: a handler built with a
	// DIFFERENT ClientIP closure (same salt provider, same UA) must yield a
	// different visitor id — proving the ClientIP dep actually feeds derivation.
	otherDeps := va.TrackDeps{
		ClientIP: func(*http.Request) string { return "10.0.0.1" },
		IsStaff:  func(*http.Request) bool { return true },
	}
	th2 := va.TrackHandler(svc, otherDeps)
	rec = postTrack(t, th2, "application/json", body)
	require.Equal(t, http.StatusNoContent, rec.Code)
	assert.NotEqual(t, firstVisitorID, store.last.VisitorID, "injected IP must influence the visitor id")

	propID := uuid.New()
	insightsDeps := va.InsightsDeps{
		PropertyID: func(*http.Request) (uuid.UUID, error) { propIDCalled = true; return propID, nil },
	}
	_, _, property, _, _ := va.InsightsHandlers(svc, insightsDeps)
	rec = httptest.NewRecorder()
	property(rec, httptest.NewRequest(http.MethodGet, "/insights/properties/x?period=30d", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, propIDCalled, "InsightsDeps.PropertyID must be invoked")
}

// TC5 (SE10): CountryFromHeader reads the configured edge header and uppercases
// the ISO-3166-α2 code. Absent / blank / placeholder (XX, T1) → "" (unknown), so
// a local/dev request with no edge header resolves country to unknown.
func TestCountryFromHeader(t *testing.T) {
	fn := va.CountryFromHeader("CF-IPCountry")

	cases := []struct {
		name   string
		header string
		value  string
		want   string
	}{
		{"present uppercase", "CF-IPCountry", "TH", "TH"},
		{"present lowercase normalized", "CF-IPCountry", "th", "TH"},
		{"absent header -> unknown", "", "", ""},
		{"blank value -> unknown", "CF-IPCountry", "", ""},
		{"placeholder XX -> unknown", "CF-IPCountry", "XX", ""},
		{"tor T1 -> unknown", "CF-IPCountry", "T1", ""},
		{"non-iso garbage -> unknown", "CF-IPCountry", "NOTACODE", ""},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/track", nil)
			if c.header != "" {
				req.Header.Set(c.header, c.value)
			}
			assert.Equal(t, c.want, fn(req))
		})
	}
}

// TC5b (SE10): the track handler passes Accept-Language and the injected Country
// dep into the recorded Event; a request with no edge header records unknown
// country while all other signals still record.
func TestTrackHandler_AcceptLanguageAndCountry(t *testing.T) {
	svc, store := newHTTPService()
	deps := va.TrackDeps{
		ClientIP: func(*http.Request) string { return "203.0.113.7" },
		IsStaff:  func(*http.Request) bool { return false },
		Country:  va.CountryFromHeader("CF-IPCountry"),
	}
	h := va.TrackHandler(svc, deps)
	tid := uuid.New()
	body := `{"type":"property_view","target":"property","target_id":"` + tid.String() + `","path":"/p"}`

	// With an edge country header + Accept-Language.
	req := httptest.NewRequest(http.MethodPost, "/track", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept-Language", "th-TH,th;q=0.9,en;q=0.8")
	req.Header.Set("CF-IPCountry", "TH")
	rec := httptest.NewRecorder()
	h(rec, req)
	require.Equal(t, http.StatusNoContent, rec.Code)
	require.NotNil(t, store.last)
	assert.Equal(t, "TH", store.last.Country)
	assert.Equal(t, "th", store.last.Language)
	assert.Equal(t, va.OSWindows, store.last.OSFamily)
	assert.Equal(t, va.BrowserChrome, store.last.BrowserFamily)

	// No edge header → country unknown, other signals still record.
	req2 := httptest.NewRequest(http.MethodPost, "/track", strings.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/120.0.0.0 Safari/537.36")
	req2.Header.Set("Accept-Language", "en-US,en;q=0.9")
	rec2 := httptest.NewRecorder()
	h(rec2, req2)
	require.Equal(t, http.StatusNoContent, rec2.Code)
	require.NotNil(t, store.last)
	assert.Equal(t, "", store.last.Country, "no edge header → unknown country")
	assert.Equal(t, "en", store.last.Language)
}

// TC1 (2606-114): an InsightsDeps.ResolveTop resolver turns the raw count-ordered
// top stats into labeled rows — dropping a target that no longer resolves (a
// deleted property: the orphan-analytics class) and labeling the survivors. The
// handler over-fetches so the drop can't shrink the board below the requested
// limit. With no resolver the raw []TargetStat shape is unchanged (back-compat,
// proven by TestInsightsHandlers_PeriodAndShape).
func TestInsightsHandlers_TopResolver(t *testing.T) {
	svc, store := newHTTPService()
	realID, orphanID := uuid.New(), uuid.New()
	// The orphan sits ABOVE the real one in count order — proving a deleted target
	// can't keep its top slot just because it accrued the most lingering hits.
	store.top = []va.TargetStat{
		{TargetID: orphanID, UniqueViews: 40, TotalHits: 40},
		{TargetID: realID, UniqueViews: 8, TotalHits: 9},
	}

	deps := va.InsightsDeps{
		PropertyID: func(*http.Request) (uuid.UUID, error) { return uuid.Nil, nil },
		// Simulates the BR properties join: only realID resolves; orphanID is absent
		// (deleted) and therefore dropped.
		ResolveTop: func(_ context.Context, stats []va.TargetStat, limit int) ([]va.LabeledTarget, error) {
			out := make([]va.LabeledTarget, 0, limit)
			for _, s := range stats {
				if s.TargetID == realID {
					out = append(out, va.LabeledTarget{TargetStat: s, Label: "Sea View Villa"})
				}
			}
			return out, nil
		},
	}
	top, _, _, _, _ := va.InsightsHandlers(svc, deps)

	rec := httptest.NewRecorder()
	top(rec, httptest.NewRequest(http.MethodGet, "/insights/properties/top?period=7d&limit=8", nil))
	require.Equal(t, http.StatusOK, rec.Code)

	var got []va.LabeledTarget
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Len(t, got, 1) // orphan dropped server-side
	assert.Equal(t, realID, got[0].TargetID)
	assert.Equal(t, "Sea View Villa", got[0].Label)
	assert.Equal(t, 8, got[0].UniqueViews)
	assert.Equal(t, 9, got[0].TotalHits)

	// The label rides in JSON as "label" alongside the flattened TargetStat fields.
	assert.Contains(t, rec.Body.String(), `"label":"Sea View Villa"`)
	assert.Contains(t, rec.Body.String(), `"target_id":"`+realID.String()+`"`)

	// Over-fetch: with a resolver present, the handler asks the store for MORE than
	// the requested limit so dropped orphans don't leave the board short.
	assert.Greater(t, store.topLimit, 8, "handler should over-fetch when a resolver may drop rows")
}

// 2606-114 (review nit #1): an UNSET ?limit must still cap the resolved board. The
// handler normalizes the limit BEFORE invoking the resolver, so ResolveTop never
// receives 0 (which would skip the per-row cap and dump the whole over-fetch
// window). Caps at the store's clampLimit default.
func TestInsightsHandlers_TopResolver_UnsetLimitCaps(t *testing.T) {
	svc, store := newHTTPService()
	// 12 real rows, all resolvable — more than the clampLimit default.
	store.top = make([]va.TargetStat, 12)
	for i := range store.top {
		store.top[i] = va.TargetStat{TargetID: uuid.New(), UniqueViews: 12 - i, TotalHits: 12 - i}
	}
	deps := va.InsightsDeps{
		PropertyID: func(*http.Request) (uuid.UUID, error) { return uuid.Nil, nil },
		// A resolver that honours the cap contract (cap at limit when limit > 0).
		ResolveTop: func(_ context.Context, stats []va.TargetStat, limit int) ([]va.LabeledTarget, error) {
			out := make([]va.LabeledTarget, 0, len(stats))
			for _, s := range stats {
				out = append(out, va.LabeledTarget{TargetStat: s, Label: "x"})
				if limit > 0 && len(out) >= limit {
					break
				}
			}
			return out, nil
		},
	}
	top, _, _, _, _ := va.InsightsHandlers(svc, deps)

	rec := httptest.NewRecorder()
	// NO limit param — the handler must still hand the resolver a concrete cap.
	top(rec, httptest.NewRequest(http.MethodGet, "/insights/properties/top?period=7d", nil))
	require.Equal(t, http.StatusOK, rec.Code)

	var got []va.LabeledTarget
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.LessOrEqual(t, len(got), 10, "unset limit must still cap the resolved board (clampLimit default), not dump the whole window")
}

// TC9b: a PropertyID extractor that returns an error → 422 (bad id).
func TestInsightsHandlers_BadPropertyID(t *testing.T) {
	svc, _ := newHTTPService()
	deps := va.InsightsDeps{
		PropertyID: func(*http.Request) (uuid.UUID, error) {
			return uuid.Nil, uuid.Validate("not-a-uuid")
		},
	}
	_, _, property, _, _ := va.InsightsHandlers(svc, deps)
	rec := httptest.NewRecorder()
	property(rec, httptest.NewRequest(http.MethodGet, "/insights/properties/not-a-uuid?period=30d", nil))
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

// TC-P4: the pages handler reads ?path=, validates period, and serializes the
// PropertyStats JSON shape from stub data. A valid request → 200 with the stats.
func TestInsightsHandlers_Pages_Shape(t *testing.T) {
	svc, store := newHTTPService()
	store.pathStats = &va.PropertyStats{
		UniqueViews: 9,
		TotalHits:   15,
		ByDay:       []va.DayStat{{Day: time.Now(), UniqueViews: 9, TotalHits: 15}},
		Breakdowns: map[string][]va.DimensionStat{
			"referer": {{Label: "https://google.com", UniqueViews: 6, TotalHits: 10}},
		},
	}
	_, _, _, pages, _ := va.InsightsHandlers(svc, va.InsightsDeps{})

	rec := httptest.NewRecorder()
	pages(rec, httptest.NewRequest(http.MethodGet, "/insights/pages?path=/sell&period=30d&dimensions=referer", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	var got va.PropertyStats
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, 9, got.UniqueViews)
	assert.Equal(t, 15, got.TotalHits)
	require.Len(t, got.Breakdowns["referer"], 1)
	assert.Equal(t, "https://google.com", got.Breakdowns["referer"][0].Label)
}

// TC-P5: the pages handler 422s on a missing path and on a bad period.
func TestInsightsHandlers_Pages_Validation(t *testing.T) {
	svc, _ := newHTTPService()
	_, _, _, pages, _ := va.InsightsHandlers(svc, va.InsightsDeps{})

	// missing path → 422
	rec := httptest.NewRecorder()
	pages(rec, httptest.NewRequest(http.MethodGet, "/insights/pages?period=30d", nil))
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code, "missing path must 422")

	// bad period → 422
	rec = httptest.NewRecorder()
	pages(rec, httptest.NewRequest(http.MethodGet, "/insights/pages?path=/sell&period=5d", nil))
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code, "bad period must 422")

	// path over 512 runes → 422
	rec = httptest.NewRecorder()
	long := "/" + strings.Repeat("a", 600)
	pages(rec, httptest.NewRequest(http.MethodGet, "/insights/pages?path="+long+"&period=7d", nil))
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code, "over-length path must 422")
}

// TC-P6: the root path "/" is a valid ?path= value (it is non-empty).
func TestInsightsHandlers_Pages_RootPath(t *testing.T) {
	svc, store := newHTTPService()
	store.pathStats = &va.PropertyStats{UniqueViews: 3, TotalHits: 3}
	_, _, _, pages, _ := va.InsightsHandlers(svc, va.InsightsDeps{})

	rec := httptest.NewRecorder()
	pages(rec, httptest.NewRequest(http.MethodGet, "/insights/pages?path=/&period=7d", nil))
	require.Equal(t, http.StatusOK, rec.Code, "root path / must be accepted")
	var got va.PropertyStats
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, 3, got.UniqueViews)
}

// TC-P7: the pages/summary handler reads ?path= and serializes the PathSummary
// JSON ({"total_views":N,"today_views":N}); a missing path → 422.
func TestInsightsHandlers_PagesSummary(t *testing.T) {
	svc, store := newHTTPService()
	store.pathSummary = va.PathSummary{TotalViews: 42, TodayViews: 7}
	_, _, _, _, pagesSummary := va.InsightsHandlers(svc, va.InsightsDeps{})

	rec := httptest.NewRecorder()
	pagesSummary(rec, httptest.NewRequest(http.MethodGet, "/insights/pages/summary?path=/sell", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	var got va.PathSummary
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, 42, got.TotalViews)
	assert.Equal(t, 7, got.TodayViews)

	// Confirm the exact wire keys.
	var raw map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &raw))
	assert.Contains(t, raw, "total_views")
	assert.Contains(t, raw, "today_views")

	// missing path → 422
	rec = httptest.NewRecorder()
	pagesSummary(rec, httptest.NewRequest(http.MethodGet, "/insights/pages/summary", nil))
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code, "missing path must 422")
}
