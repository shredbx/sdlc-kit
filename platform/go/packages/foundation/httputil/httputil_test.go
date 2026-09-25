package httputil_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/shredbx/sbx-core/pkg/httputil"
)

type sampleInput struct {
	Code  string `json:"code"  validate:"required,min=1,max=10"`
	Label string `json:"label" validate:"required,max=20"`
	Count int    `json:"count" validate:"gte=0,lte=100"`
}

func decodeJSONResponse(t *testing.T, rec *httptest.ResponseRecorder) httputil.ErrorResponse {
	t.Helper()
	var resp httputil.ErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return resp
}

func newJSONRequest(t *testing.T, body string) *http.Request {
	t.Helper()
	return httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
}

// --- DecodeAndValidate -----------------------------------------------------

func TestDecodeAndValidate_HappyPath(t *testing.T) {
	rec := httptest.NewRecorder()
	req := newJSONRequest(t, `{"code":"abc","label":"hello","count":5}`)
	in, ok := httputil.DecodeAndValidate[sampleInput](rec, req, 1024)
	if !ok {
		t.Fatalf("expected ok=true, got body: %s", rec.Body.String())
	}
	if in.Code != "abc" || in.Label != "hello" || in.Count != 5 {
		t.Fatalf("unexpected decoded values: %+v", in)
	}
}

func TestDecodeAndValidate_MissingRequired(t *testing.T) {
	rec := httptest.NewRecorder()
	req := newJSONRequest(t, `{"count":5}`)
	if _, ok := httputil.DecodeAndValidate[sampleInput](rec, req, 1024); ok {
		t.Fatal("expected ok=false on missing required fields")
	}
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", rec.Code)
	}
	resp := decodeJSONResponse(t, rec)
	if resp.Code != httputil.CodeValidationFailed {
		t.Fatalf("expected code=validation_failed, got %q", resp.Code)
	}
	if len(resp.Details) < 2 {
		t.Fatalf("expected at least 2 field errors (code, label), got %d", len(resp.Details))
	}
	fields := map[string]bool{}
	for _, d := range resp.Details {
		fields[d.Field] = true
	}
	if !fields["code"] || !fields["label"] {
		t.Fatalf("expected fields code+label in details, got %+v", resp.Details)
	}
}

func TestDecodeAndValidate_FieldRangeViolation(t *testing.T) {
	rec := httptest.NewRecorder()
	req := newJSONRequest(t, `{"code":"abc","label":"x","count":999}`)
	if _, ok := httputil.DecodeAndValidate[sampleInput](rec, req, 1024); ok {
		t.Fatal("expected ok=false on out-of-range count")
	}
	resp := decodeJSONResponse(t, rec)
	if len(resp.Details) == 0 || resp.Details[0].Field != "count" || resp.Details[0].Tag != "lte" {
		t.Fatalf("expected count/lte detail, got %+v", resp.Details)
	}
}

func TestDecodeAndValidate_MalformedJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	req := newJSONRequest(t, `{"code":`)
	if _, ok := httputil.DecodeAndValidate[sampleInput](rec, req, 1024); ok {
		t.Fatal("expected ok=false on malformed JSON")
	}
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	resp := decodeJSONResponse(t, rec)
	if resp.Code != httputil.CodeInvalidJSON {
		t.Fatalf("expected code=invalid_json, got %q", resp.Code)
	}
}

func TestDecodeAndValidate_EmptyBody(t *testing.T) {
	rec := httptest.NewRecorder()
	req := newJSONRequest(t, ``)
	if _, ok := httputil.DecodeAndValidate[sampleInput](rec, req, 1024); ok {
		t.Fatal("expected ok=false on empty body")
	}
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestDecodeAndValidate_UnknownField(t *testing.T) {
	rec := httptest.NewRecorder()
	req := newJSONRequest(t, `{"code":"abc","label":"x","count":1,"extra":"nope"}`)
	if _, ok := httputil.DecodeAndValidate[sampleInput](rec, req, 1024); ok {
		t.Fatal("expected ok=false on unknown field")
	}
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 from DisallowUnknownFields, got %d", rec.Code)
	}
}

func TestDecodeAndValidate_BodyTooLarge(t *testing.T) {
	big := bytes.Repeat([]byte("a"), 2048)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/",
		strings.NewReader(`{"code":"abc","label":"x","count":1,"pad":"`+string(big)+`"}`))
	if _, ok := httputil.DecodeAndValidate[sampleInput](rec, req, 256); ok {
		t.Fatal("expected ok=false on oversized body")
	}
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d", rec.Code)
	}
	resp := decodeJSONResponse(t, rec)
	if resp.Code != httputil.CodePayloadTooLarge {
		t.Fatalf("expected code=payload_too_large, got %q", resp.Code)
	}
}

func TestDecodeAndValidate_TrailingGarbage(t *testing.T) {
	rec := httptest.NewRecorder()
	req := newJSONRequest(t, `{"code":"abc","label":"x","count":1}{"another":1}`)
	if _, ok := httputil.DecodeAndValidate[sampleInput](rec, req, 1024); ok {
		t.Fatal("expected ok=false on trailing data")
	}
}

// --- ParseUUID -------------------------------------------------------------

func TestParseUUID(t *testing.T) {
	cases := []struct {
		name    string
		raw     string
		wantOK  bool
		wantStatus int
	}{
		{"valid", "550e8400-e29b-41d4-a716-446655440000", true, http.StatusOK},
		{"empty", "", false, http.StatusBadRequest},
		{"garbage", "not-a-uuid", false, http.StatusBadRequest},
		{"truncated", "550e8400-e29b-41d4-a716", false, http.StatusBadRequest},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			_, ok := httputil.ParseUUID(rec, tc.raw, "id")
			if ok != tc.wantOK {
				t.Fatalf("ok=%v, want %v", ok, tc.wantOK)
			}
			if !ok && rec.Code != tc.wantStatus {
				t.Fatalf("status=%d, want %d", rec.Code, tc.wantStatus)
			}
		})
	}
}

// --- ParseCode -------------------------------------------------------------

func TestParseCode(t *testing.T) {
	cases := []struct {
		name   string
		raw    string
		maxLen int
		wantOK bool
	}{
		{"valid lowercase alnum", "property-types", 50, true},
		{"valid with underscore", "for_sale", 50, true},
		{"empty", "", 50, false},
		{"too long", "this-is-way-too-long-for-the-limit", 10, false},
		{"uppercase rejected", "Foo", 50, false},
		{"space rejected", "foo bar", 50, false},
		{"slash rejected", "foo/bar", 50, false},
		{"unlimited length", strings.Repeat("a", 1000), 0, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			_, ok := httputil.ParseCode(rec, tc.raw, "name", tc.maxLen)
			if ok != tc.wantOK {
				t.Fatalf("ok=%v, want %v (body: %s)", ok, tc.wantOK, rec.Body.String())
			}
		})
	}
}

// --- ParseEnum -------------------------------------------------------------

type imagePurpose string

const (
	purposeProperty imagePurpose = "property"
	purposeAgent    imagePurpose = "agent"
)

func TestParseEnum(t *testing.T) {
	allowed := []imagePurpose{purposeProperty, purposeAgent}
	cases := []struct {
		name   string
		raw    string
		wantOK bool
		wantValue imagePurpose
	}{
		{"valid", "property", true, purposeProperty},
		{"valid second", "agent", true, purposeAgent},
		{"empty", "", false, ""},
		{"not allowed", "watermark", false, ""},
		{"case mismatch", "Property", false, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			got, ok := httputil.ParseEnum(rec, tc.raw, "purpose", allowed)
			if ok != tc.wantOK {
				t.Fatalf("ok=%v, want %v", ok, tc.wantOK)
			}
			if ok && got != tc.wantValue {
				t.Fatalf("value=%q, want %q", got, tc.wantValue)
			}
		})
	}
}

// --- ParseIntRange ---------------------------------------------------------

func TestParseIntRange(t *testing.T) {
	cases := []struct {
		name   string
		raw    string
		min    int
		max    int
		def    int
		wantOK bool
		wantN  int
	}{
		{"valid", "50", 0, 100, 10, true, 50},
		{"min boundary", "0", 0, 100, 10, true, 0},
		{"max boundary", "100", 0, 100, 10, true, 100},
		{"empty uses default", "", 0, 100, 10, true, 10},
		{"too small", "-1", 0, 100, 10, false, 0},
		{"too large", "101", 0, 100, 10, false, 0},
		{"not a number", "abc", 0, 100, 10, false, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			n, ok := httputil.ParseIntRange(rec, tc.raw, "page", tc.min, tc.max, tc.def)
			if ok != tc.wantOK {
				t.Fatalf("ok=%v, want %v (body: %s)", ok, tc.wantOK, rec.Body.String())
			}
			if ok && n != tc.wantN {
				t.Fatalf("n=%d, want %d", n, tc.wantN)
			}
		})
	}
}

// --- ParseBool -------------------------------------------------------------

func TestParseBool(t *testing.T) {
	cases := []struct {
		name   string
		raw    string
		def    bool
		wantOK bool
		wantV  bool
	}{
		{"true", "true", false, true, true},
		{"True", "True", false, true, true},
		{"1", "1", false, true, true},
		{"yes", "yes", false, true, true},
		{"false", "false", true, true, false},
		{"0", "0", true, true, false},
		{"no", "no", true, true, false},
		{"empty uses default true", "", true, true, true},
		{"empty uses default false", "", false, true, false},
		{"garbage", "maybe", false, false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			v, ok := httputil.ParseBool(rec, tc.raw, "flag", tc.def)
			if ok != tc.wantOK {
				t.Fatalf("ok=%v, want %v", ok, tc.wantOK)
			}
			if ok && v != tc.wantV {
				t.Fatalf("v=%v, want %v", v, tc.wantV)
			}
		})
	}
}

// --- BodySizeLimit middleware ---------------------------------------------

func TestBodySizeLimit_AllowsSmall(t *testing.T) {
	mw := httputil.BodySizeLimit(1024)
	called := false
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		if len(body) != 10 {
			t.Fatalf("len=%d, want 10", len(body))
		}
	}))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("0123456789"))
	h.ServeHTTP(rec, req)
	if !called {
		t.Fatal("handler not invoked")
	}
}

func TestBodySizeLimit_RejectsLarge(t *testing.T) {
	mw := httputil.BodySizeLimit(8)
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := io.ReadAll(r.Body)
		// MaxBytesReader returns *http.MaxBytesError once the limit is crossed.
		if err == nil {
			t.Fatal("expected error reading oversized body")
		}
		var maxErr *http.MaxBytesError
		if !isMaxBytesError(err, &maxErr) {
			t.Fatalf("expected MaxBytesError, got %T: %v", err, err)
		}
	}))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("0123456789abcdef"))
	h.ServeHTTP(rec, req)
}

func isMaxBytesError(err error, target **http.MaxBytesError) bool {
	for e := err; e != nil; {
		if me, ok := e.(*http.MaxBytesError); ok {
			*target = me
			return true
		}
		// best-effort unwrap
		type unwrapper interface{ Unwrap() error }
		if u, ok := e.(unwrapper); ok {
			e = u.Unwrap()
			continue
		}
		break
	}
	return false
}

// --- WriteError + WriteValidationError -------------------------------------

func TestWriteError(t *testing.T) {
	rec := httptest.NewRecorder()
	httputil.WriteError(rec, http.StatusForbidden, "forbidden", httputil.CodeForbidden)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d, want 403", rec.Code)
	}
	if rec.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("missing content-type header")
	}
	resp := decodeJSONResponse(t, rec)
	if resp.Error != "forbidden" || resp.Code != httputil.CodeForbidden {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

// ParseCodeList — the shared multi-code grammar (comma-split, trim, blank-skip,
// per-code ParseCode) with the maxItems cap (2607-004 S1: an unbounded comma list
// on a public endpoint would become an unbounded SQL IN parameter set).
func TestParseCodeList(t *testing.T) {
	t.Run("empty -> nil ok (optional param)", func(t *testing.T) {
		rec := httptest.NewRecorder()
		codes, ok := httputil.ParseCodeList(rec, "", "property_type", 50, 3)
		if !ok || codes != nil {
			t.Fatalf("empty: got %v ok=%v, want nil ok=true", codes, ok)
		}
	})
	t.Run("splits, trims, skips blanks", func(t *testing.T) {
		rec := httptest.NewRecorder()
		codes, ok := httputil.ParseCodeList(rec, " villa , ,condo", "property_type", 50, 3)
		if !ok || len(codes) != 2 || codes[0] != "villa" || codes[1] != "condo" {
			t.Fatalf("got %v ok=%v, want [villa condo] ok=true", codes, ok)
		}
	})
	t.Run("bad grammar -> 400", func(t *testing.T) {
		rec := httptest.NewRecorder()
		if _, ok := httputil.ParseCodeList(rec, "villa,BAD CODE", "property_type", 50, 3); ok {
			t.Fatal("bad code must fail")
		}
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rec.Code)
		}
	})
	t.Run("over maxItems -> 400, never silently dropped", func(t *testing.T) {
		rec := httptest.NewRecorder()
		if _, ok := httputil.ParseCodeList(rec, "a,b,c,d", "property_type", 50, 3); ok {
			t.Fatal("4 codes with cap 3 must fail")
		}
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rec.Code)
		}
	})
	t.Run("exactly maxItems passes", func(t *testing.T) {
		rec := httptest.NewRecorder()
		codes, ok := httputil.ParseCodeList(rec, "a,b,c", "property_type", 50, 3)
		if !ok || len(codes) != 3 {
			t.Fatalf("got %v ok=%v, want 3 codes ok=true", codes, ok)
		}
	})
}
