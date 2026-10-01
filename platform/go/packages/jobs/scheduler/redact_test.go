package scheduler

import (
	"context"
	"strings"
	"testing"
	"time"
)

// SC6 — Finish redacts secrets and bounds the run output before persist. A run's
// captured output (a DATABASE_URL with a password, a presigned X-Amz-Signature,
// and a long body) must be redacted and tail-bounded when stored.
func TestFinish_RedactsAndBoundsOutput(t *testing.T) {
	now := time.Date(2026, 6, 19, 12, 0, 0, 0, time.UTC)
	store := newMemStore(now)
	store.put(Job{Name: "db-backup", Enabled: true, IntervalMinutes: 60})

	// Pin Finish's clock so the test is deterministic.
	s := New(store, WithNow(func() time.Time { return now }))

	runID, _, claimed, err := s.Start(context.Background(), "db-backup")
	if err != nil || !claimed {
		t.Fatalf("expected to claim run: claimed=%v err=%v", claimed, err)
	}

	// A long body so the bounding kicks in: pad past the default cap.
	longBody := strings.Repeat("x", DefaultMaxOutputBytes+4096)
	raw := "connecting postgres://br:s3cr3tpw@db:5432/app\n" +
		"GET https://r2.example.com/backup.sql?X-Amz-Signature=deadbeefcafef00d&X-Amz-Date=20260619\n" +
		longBody + "\nTAIL-MARKER-DONE"

	if err := s.Finish(context.Background(), runID, RunStatusOK, raw); err != nil {
		t.Fatalf("Finish error: %v", err)
	}

	if len(store.outputs) != 1 {
		t.Fatalf("expected one stored output, got %d", len(store.outputs))
	}
	got := store.outputs[0]

	// Redaction: the DB password must be gone.
	if strings.Contains(got, "s3cr3tpw") {
		t.Errorf("stored output still contains the DATABASE_URL password: %q", got)
	}
	// Redaction: the presigned signature value must be gone.
	if strings.Contains(got, "deadbeefcafef00d") {
		t.Errorf("stored output still contains the X-Amz-Signature value")
	}
	// Bounding: the stored output must be within the cap.
	if len(got) > DefaultMaxOutputBytes {
		t.Errorf("stored output not bounded: %d bytes > cap %d", len(got), DefaultMaxOutputBytes)
	}
	// Tail-keeping: the TAIL of the output is what we keep (the most recent lines).
	if !strings.Contains(got, "TAIL-MARKER-DONE") {
		t.Errorf("expected the TAIL of the output to be kept")
	}
	// A truncation marker is present when content was cut.
	if !strings.Contains(got, "truncated") {
		t.Errorf("expected a truncation marker when output is cut")
	}
}

func TestRedact(t *testing.T) {
	cases := []struct {
		name       string
		in         string
		mustGone   []string // substrings that must NOT survive
		mustRemain []string // substrings that must survive (non-secret context)
	}{
		{
			name:       "database url password",
			in:         "postgres://user:supersecret@host:5432/db",
			mustGone:   []string{"supersecret"},
			mustRemain: []string{"postgres://user:", "@host:5432/db"},
		},
		{
			name:     "presigned X-Amz-Signature query param",
			in:       "https://r2/obj?X-Amz-Signature=abc123def456&X-Amz-Expires=900",
			mustGone: []string{"abc123def456"},
		},
		{
			name:     "generic sig query param",
			in:       "https://cdn/file?sig=topsecretsig&v=2",
			mustGone: []string{"topsecretsig"},
		},
		{
			name:     "rclone config secret env",
			in:       "RCLONE_CONFIG_R2_SECRET_ACCESS_KEY=abcd1234secretvalue rclone copy ...",
			mustGone: []string{"abcd1234secretvalue"},
		},
		{
			name:     "generic SECRET env assignment",
			in:       "BESTIEREALESTATE_R2_SECRET_ACCESS_KEY=zzz999secretkey next",
			mustGone: []string{"zzz999secretkey"},
		},
		{
			// Regression (code review 2026-06-19): generated Postgres passwords
			// contain '/' and '+'; the old class [^@/\s]+ stopped at '/' and leaked.
			name:       "database url password with slash and plus",
			in:         "connecting postgres://app:Xq9/Lm2+abcDEF@db.internal:5432/bestie?sslmode=require",
			mustGone:   []string{"Xq9/Lm2+abcDEF"},
			mustRemain: []string{"postgres://app:", "@db.internal:5432"},
		},
		{
			// Regression: SigV4 HEADER form (rclone -vv / aws debug), not a query param.
			name:     "x-amz-security-token header",
			in:       "x-amz-security-token: FwoGZ1longopaquetoken2345",
			mustGone: []string{"FwoGZ1longopaquetoken2345"},
		},
		{
			name:     "x-amz-signature header",
			in:       "< X-Amz-Signature: 9f8e7d6c5b4a3210ffee",
			mustGone: []string{"9f8e7d6c5b4a3210ffee"},
		},
		{
			name:     "authorization bearer header",
			in:       "Authorization: Bearer eyJhbGciOiJIUzI1NiJ9.payloadpart.signaturepart",
			mustGone: []string{"eyJhbGciOiJIUzI1NiJ9.payloadpart.signaturepart"},
		},
		{
			name:       "clean text untouched",
			in:         "backup ok: 1 file, 42 MB in 3.1s",
			mustRemain: []string{"backup ok: 1 file, 42 MB in 3.1s"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Redact(c.in)
			for _, g := range c.mustGone {
				if strings.Contains(got, g) {
					t.Errorf("Redact(%q) leaked %q: got %q", c.in, g, got)
				}
			}
			for _, r := range c.mustRemain {
				if !strings.Contains(got, r) {
					t.Errorf("Redact(%q) dropped non-secret %q: got %q", c.in, r, got)
				}
			}
		})
	}
}

func TestBoundOutput(t *testing.T) {
	// Under the cap → unchanged.
	short := "all good"
	if got := BoundOutput(short, 100); got != short {
		t.Errorf("BoundOutput unchanged for short input: got %q", got)
	}

	// Over the cap → keep the TAIL, prefix a truncation marker, stay within cap.
	body := strings.Repeat("a", 50) + "TAIL"
	got := BoundOutput(body, 20)
	if len(got) > 20 {
		t.Errorf("BoundOutput exceeded cap: %d > 20", len(got))
	}
	if !strings.HasPrefix(got, "…[truncated]") {
		t.Errorf("expected truncation prefix, got %q", got)
	}
	if !strings.HasSuffix(got, "TAIL") {
		t.Errorf("expected the tail kept, got %q", got)
	}

	// A non-positive cap is treated defensively (no panic; returns within a sane bound).
	_ = BoundOutput(body, 0)
}
