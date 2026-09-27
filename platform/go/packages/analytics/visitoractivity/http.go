package visitoractivity

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/shredbx/sbx-core/pkg/httputil"
)

// maxTrackBody caps the /track request body read inside the handler as a
// defence-in-depth floor. The app SHOULD also wrap the route in
// httputil.BodySizeLimit; this in-handler limit guarantees a bounded read even
// if a consumer forgets the middleware.
const maxTrackBody = 8 << 10 // 8 KB

// recordTimeout bounds each Service.Record call. The package applies no timeout
// by design (LANDMINE 6) — the handler owns it so a slow DB can't pin a beacon.
const recordTimeout = 3 * time.Second

// trackBody is the wire shape a client POSTs to the track endpoint. Derived
// fields (visitor id, device, is_bot, is_staff, ip, ua, referer_origin,
// created_at) are NEVER accepted from the client — the server fills them.
type trackBody struct {
	Type     string         `json:"type"`
	Target   string         `json:"target"`
	TargetID string         `json:"target_id,omitempty"`
	Path     string         `json:"path"`
	Referer  string         `json:"referer,omitempty"`
	Props    map[string]any `json:"props,omitempty"`
}

// TrackDeps injects the app-specific request derivations the track handler
// needs, keeping the handler router- and app-agnostic. ClientIP yields the real
// client IP (used as HMAC input only — never stored); IsStaff reports whether
// the request carries an authenticated staff session (e.g. claims != nil);
// Country yields the ISO-3166-α2 country code from a trusted edge header (or ""
// for unknown) — never derived from the raw IP. Country is optional: a nil
// Country resolves to "" (unknown), so existing callers keep working.
type TrackDeps struct {
	ClientIP func(*http.Request) string
	IsStaff  func(*http.Request) bool
	Country  func(*http.Request) string
}

// isoCountryRe matches a normalized ISO-3166-α2 code (two uppercase letters).
var isoCountryRe = regexp.MustCompile(`^[A-Z]{2}$`)

// cloudflareUnknownCountries are the CF-IPCountry placeholder values that mean
// "no real country": XX = unknown, T1 = Tor exit. Both map to "" (unknown).
var cloudflareUnknownCountries = map[string]bool{"XX": true, "T1": true}

// CountryFromHeader returns a TrackDeps.Country source that reads a trusted edge
// request header (e.g. Cloudflare's "CF-IPCountry"), uppercases it, and returns
// the ISO-3166-α2 code — or "" when the header is absent, blank, a Cloudflare
// placeholder (XX/T1), or not a valid two-letter code. Zero dependency: the same
// trusted-edge-header pattern as the X-Forwarded-For client-IP work. In dev
// (no edge) the header is absent and country resolves to "" (unknown).
func CountryFromHeader(headerName string) func(*http.Request) string {
	return func(r *http.Request) string {
		return normalizeCountry(r.Header.Get(headerName))
	}
}

// normalizeCountry uppercases/trims a candidate country code and returns the
// ISO-3166-α2 value, or "" when blank, a Cloudflare placeholder (XX/T1), or not
// a valid two-letter code. Shared by CountryFromHeader and Service.Record so a
// direct RecordInput caller (e.g. BS, a non-HTTP path) can never persist a
// non-ISO value.
func normalizeCountry(raw string) string {
	code := strings.ToUpper(strings.TrimSpace(raw))
	if code == "" || cloudflareUnknownCountries[code] || !isoCountryRe.MatchString(code) {
		return ""
	}
	return code
}

// TrackHandler returns an http.HandlerFunc that ingests one visitor-activity
// event. It accepts application/json AND text/plain bodies (beacon clients vary
// the content type), validates via Service.Record, and ALWAYS responds without
// echoing the row:
//
//   - success                          → 204 No Content
//   - ErrInvalidEvent/ErrTargetRequired/ErrPathRequired → 422 Unprocessable Entity
//   - ErrPropsTooLarge                  → 413 Payload Too Large
//   - body over the size limit          → 413 Payload Too Large
//   - any other (internal) error        → logged, then 204 (never leak, never
//     500 to a fire-and-forget beacon)
//
// stdlib net/http only — no chi, no CSRF/rate-limit/RBAC concerns here; those
// stay app-side via route registration so any project can reuse this handler.
func TrackHandler(svc *Service, deps TrackDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Bound the read even without upstream BodySizeLimit middleware.
		r.Body = http.MaxBytesReader(w, r.Body, maxTrackBody)

		var body trackBody
		dec := json.NewDecoder(r.Body)
		if err := dec.Decode(&body); err != nil {
			var maxErr *http.MaxBytesError
			if errors.As(err, &maxErr) {
				httputil.WriteError(w, http.StatusRequestEntityTooLarge, "request body too large", httputil.CodePayloadTooLarge)
				return
			}
			if errors.Is(err, io.EOF) {
				httputil.WriteError(w, http.StatusBadRequest, "request body is empty", httputil.CodeInvalidJSON)
				return
			}
			httputil.WriteError(w, http.StatusBadRequest, "invalid JSON", httputil.CodeInvalidJSON)
			return
		}

		in := RecordInput{
			EventType:      EventType(body.Type),
			TargetType:     TargetType(body.Target),
			Path:           body.Path,
			RefererRaw:     body.Referer,
			IsStaff:        deps.IsStaff(r),
			IP:             deps.ClientIP(r),
			UserAgent:      r.Header.Get("User-Agent"),
			AcceptLanguage: r.Header.Get("Accept-Language"),
			Country:        countryFromDeps(deps, r),
			Props:          body.Props,
		}
		if body.TargetID != "" {
			id, err := uuid.Parse(body.TargetID)
			if err != nil {
				httputil.WriteError(w, http.StatusUnprocessableEntity, "target_id must be a valid UUID", httputil.CodeValidationFailed)
				return
			}
			in.TargetID = &id
		}

		ctx, cancel := context.WithTimeout(r.Context(), recordTimeout)
		defer cancel()

		switch err := svc.Record(ctx, in); {
		case err == nil:
			w.WriteHeader(http.StatusNoContent)
		case errors.Is(err, ErrInvalidEvent), errors.Is(err, ErrTargetRequired), errors.Is(err, ErrPathRequired):
			// All three are malformed-client-request (400-class) validation
			// failures — a page event with a stray target_id or no path is the
			// same shape of error as a property event missing its target_id, so it
			// returns the same 422 (not a misfiled 204 via the internal-error arm).
			httputil.WriteError(w, http.StatusUnprocessableEntity, "invalid event", httputil.CodeValidationFailed)
		case errors.Is(err, ErrPropsTooLarge):
			httputil.WriteError(w, http.StatusRequestEntityTooLarge, "props payload too large", httputil.CodePayloadTooLarge)
		default:
			// Internal failure (DB/salt). Never leak detail and never 500 a
			// fire-and-forget beacon — log server-side and 204.
			log.Printf("visitoractivity track: %v", err)
			w.WriteHeader(http.StatusNoContent)
		}
	}
}

// countryFromDeps resolves the request country via the injected dep, treating a
// nil Country as "" (unknown) so callers that don't wire it keep working.
func countryFromDeps(deps TrackDeps, r *http.Request) string {
	if deps.Country == nil {
		return ""
	}
	return deps.Country(r)
}

// InsightsDeps injects the route-specific property-id extraction so the insights
// handlers stay router-agnostic (the app supplies e.g. chi.URLParam parsing).
type InsightsDeps struct {
	PropertyID func(*http.Request) (uuid.UUID, error)

	// ResolveTop, when non-nil, turns the raw count-ordered top stats into the
	// labeled rows the leaderboard actually writes. The consumer attaches a
	// display label and DROPS targets that no longer resolve (deleted entities) —
	// the polymorphic target_id has no FK, so only the consumer knows which
	// targets still exist — capping the result at `limit`. The handler over-fetches
	// (see topResolveOverfetch) so dropped orphans don't shrink the board below
	// `limit`. nil → the raw []TargetStat is written unchanged (back-compat).
	ResolveTop func(ctx context.Context, stats []TargetStat, limit int) ([]LabeledTarget, error)
}

// parseDimensions reads an optional comma-separated ?dimensions= query param.
// Empty / unset → nil (the store then computes the full default registry). The
// store validates each key against the allowlist and 422s on an unknown one, so
// no allowlisting is duplicated here — only trimming/splitting.
func parseDimensions(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var out []string
	for _, part := range strings.Split(raw, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// maxPathParam caps the ?path= query param length for the page insights
// handlers. Mirrors the visitor_activity.path column width (VARCHAR(512)) so a
// path that could never have been stored is rejected before any query runs.
const maxPathParam = 512

// topResolveOverfetch widens the top-properties query when an InsightsDeps.ResolveTop
// resolver is present: the resolver may drop targets that no longer exist (deleted
// entities, whose polymorphic events linger), so the handler fetches limit×this so
// the surviving rows can still fill the board to the requested limit. clampLimit
// caps the widened window at the store maximum.
//
// Accepted degradation: if MORE than limit×this of the count-ordered window are
// orphans, the board returns short (fewer than limit rows even though more real
// ones exist deeper). Benign in practice — orphans are a finite deleted-entity
// artifact, not unbounded — but widening this factor trades a larger scan for
// resilience if a workload ever accumulates many high-traffic deletions.
const topResolveOverfetch = 4

// InsightsHandlers returns five read handlers over the analytics aggregations:
//
//	top          GET .../insights/properties/top?period=7d&limit=10  → []TargetStat
//	overview     GET .../insights/overview?period=7d                 → Overview
//	property     GET .../insights/properties/{id}?period=30d         → PropertyStats
//	pages        GET .../insights/pages?path=/&period=30d            → PropertyStats
//	pagesSummary GET .../insights/pages/summary?path=/               → PathSummary
//
// `period` is whitelisted to 7d|30d|90d (else 422); `limit` passes straight
// through to the store (which clamps to ≤500). The page handlers key on the
// ?path= query param (a page is identified by its URL path, NOT a UUID), which
// must be non-empty and ≤512 runes. RBAC/auth gating is applied app-side at
// registration. stdlib net/http only.
func InsightsHandlers(svc *Service, deps InsightsDeps) (top, overview, property, pages, pagesSummary http.HandlerFunc) {
	top = func(w http.ResponseWriter, r *http.Request) {
		since, ok := parsePeriod(r.URL.Query().Get("period"))
		if !ok {
			httputil.WriteError(w, http.StatusUnprocessableEntity, "period must be one of: 7d, 30d, 90d", httputil.CodeValidationFailed)
			return
		}
		// Normalize the limit up-front (clampLimit applies the same floor/ceiling the
		// store would) so the resolver always receives a CONCRETE, capped value — an
		// unset ?limit must not hand ResolveTop a 0 that skips its cap.
		limit := clampLimit(parseLimit(r.URL.Query().Get("limit")))
		fetch := limit
		if deps.ResolveTop != nil {
			// Over-fetch a wider window so the resolver's orphan-drop (deleted
			// targets that no longer resolve) can still fill the board to `limit`.
			// clampLimit caps the widened window at the store maximum.
			fetch = clampLimit(limit * topResolveOverfetch)
		}
		stats, err := svc.TopProperties(r.Context(), since, fetch)
		if err != nil {
			httputil.WriteError(w, http.StatusInternalServerError, "failed to load top properties", httputil.CodeInternal)
			return
		}
		if stats == nil {
			stats = []TargetStat{}
		}
		if deps.ResolveTop != nil {
			rows, err := deps.ResolveTop(r.Context(), stats, limit)
			if err != nil {
				httputil.WriteError(w, http.StatusInternalServerError, "failed to resolve top properties", httputil.CodeInternal)
				return
			}
			if rows == nil {
				rows = []LabeledTarget{}
			}
			writeInsightsJSON(w, rows)
			return
		}
		writeInsightsJSON(w, stats)
	}

	overview = func(w http.ResponseWriter, r *http.Request) {
		since, ok := parsePeriod(r.URL.Query().Get("period"))
		if !ok {
			httputil.WriteError(w, http.StatusUnprocessableEntity, "period must be one of: 7d, 30d, 90d", httputil.CodeValidationFailed)
			return
		}
		dims := parseDimensions(r.URL.Query().Get("dimensions"))
		ov, err := svc.Overview(r.Context(), since, dims...)
		if err != nil {
			if errors.Is(err, ErrUnknownDimension) {
				httputil.WriteError(w, http.StatusUnprocessableEntity, "unknown dimension requested", httputil.CodeValidationFailed)
				return
			}
			httputil.WriteError(w, http.StatusInternalServerError, "failed to load overview", httputil.CodeInternal)
			return
		}
		writeInsightsJSON(w, ov)
	}

	property = func(w http.ResponseWriter, r *http.Request) {
		id, err := deps.PropertyID(r)
		if err != nil {
			httputil.WriteError(w, http.StatusUnprocessableEntity, "id must be a valid UUID", httputil.CodeValidationFailed)
			return
		}
		since, ok := parsePeriod(r.URL.Query().Get("period"))
		if !ok {
			httputil.WriteError(w, http.StatusUnprocessableEntity, "period must be one of: 7d, 30d, 90d", httputil.CodeValidationFailed)
			return
		}
		dims := parseDimensions(r.URL.Query().Get("dimensions"))
		stats, err := svc.PropertyStats(r.Context(), id, since, dims...)
		if err != nil {
			if errors.Is(err, ErrUnknownDimension) {
				httputil.WriteError(w, http.StatusUnprocessableEntity, "unknown dimension requested", httputil.CodeValidationFailed)
				return
			}
			httputil.WriteError(w, http.StatusInternalServerError, "failed to load property stats", httputil.CodeInternal)
			return
		}
		writeInsightsJSON(w, stats)
	}

	pages = func(w http.ResponseWriter, r *http.Request) {
		path, ok := parsePathParam(r.URL.Query().Get("path"))
		if !ok {
			httputil.WriteError(w, http.StatusUnprocessableEntity, "path must be non-empty and at most 512 characters", httputil.CodeValidationFailed)
			return
		}
		since, ok := parsePeriod(r.URL.Query().Get("period"))
		if !ok {
			httputil.WriteError(w, http.StatusUnprocessableEntity, "period must be one of: 7d, 30d, 90d", httputil.CodeValidationFailed)
			return
		}
		dims := parseDimensions(r.URL.Query().Get("dimensions"))
		stats, err := svc.PathStats(r.Context(), path, since, dims...)
		if err != nil {
			if errors.Is(err, ErrUnknownDimension) {
				httputil.WriteError(w, http.StatusUnprocessableEntity, "unknown dimension requested", httputil.CodeValidationFailed)
				return
			}
			httputil.WriteError(w, http.StatusInternalServerError, "failed to load page stats", httputil.CodeInternal)
			return
		}
		writeInsightsJSON(w, stats)
	}

	pagesSummary = func(w http.ResponseWriter, r *http.Request) {
		path, ok := parsePathParam(r.URL.Query().Get("path"))
		if !ok {
			httputil.WriteError(w, http.StatusUnprocessableEntity, "path must be non-empty and at most 512 characters", httputil.CodeValidationFailed)
			return
		}
		summary, err := svc.PathSummary(r.Context(), path)
		if err != nil {
			httputil.WriteError(w, http.StatusInternalServerError, "failed to load page summary", httputil.CodeInternal)
			return
		}
		writeInsightsJSON(w, summary)
	}

	return top, overview, property, pages, pagesSummary
}

// parsePathParam validates a ?path= query value for the page insights handlers:
// it must be non-empty (the URL path IS the page identifier) and at most
// maxPathParam runes (the stored column width). Returns ok=false on either
// failure so the handler can 422. The value is passed straight through to the
// store as a bound parameter — never interpolated into SQL.
func parsePathParam(raw string) (string, bool) {
	if raw == "" || len([]rune(raw)) > maxPathParam {
		return "", false
	}
	return raw, true
}

// parsePeriod maps a whitelisted period token to a "since" time. Only today/7d/30d/90d
// are accepted; anything else returns ok=false so the handler can 422. "today" is the
// start of the current UTC day (matching the store's UTC-day bucketing); 7d/30d/90d are
// rolling N-day windows.
func parsePeriod(period string) (time.Time, bool) {
	if period == "today" {
		// Start of the current UTC day — aligns with the store's UTC day buckets.
		return time.Now().UTC().Truncate(24 * time.Hour), true
	}
	var days int
	switch period {
	case "7d":
		days = 7
	case "30d":
		days = 30
	case "90d":
		days = 90
	default:
		return time.Time{}, false
	}
	return time.Now().Add(-time.Duration(days) * 24 * time.Hour), true
}

// parseLimit reads an optional limit param. The store clamps the value
// (non-positive → default, >500 → 500), so an unparseable value is treated as
// "unset" (0) rather than a hard error — leaderboard reads should not 4xx on a
// stray query param.
func parseLimit(raw string) int {
	if raw == "" {
		return 0
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0
	}
	return n
}

// writeInsightsJSON serializes a 200 OK JSON body.
func writeInsightsJSON(w http.ResponseWriter, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Printf("visitoractivity insights: encode response: %v", err)
	}
}
