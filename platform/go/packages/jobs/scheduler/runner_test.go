package scheduler

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// RC1 — a due, registered job is claimed, its handler runs ok, and the result is
// Status=ok with a finished run recording the handler's output.
func TestRunDue_DueJobClaimedHandlerOK(t *testing.T) {
	now := time.Date(2026, 6, 19, 12, 0, 0, 0, time.UTC)
	store := newMemStore(now)
	store.put(Job{Name: "news-refresh", Enabled: true, IntervalMinutes: 10, LastRunAt: ptrTime(now.Add(-20 * time.Minute))})
	s := New(store, WithNow(func() time.Time { return now }))

	var calls int
	reg := Registry{
		"news-refresh": func(ctx context.Context) (string, error) {
			calls++
			return "imported=3 deduped=1", nil
		},
	}

	results, err := s.RunDue(context.Background(), reg)
	if err != nil {
		t.Fatalf("RunDue error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if calls != 1 {
		t.Fatalf("expected handler invoked once, got %d", calls)
	}

	got := results[0]
	if got.Name != "news-refresh" {
		t.Errorf("result name = %q, want news-refresh", got.Name)
	}
	if got.Status != RunStatusOK {
		t.Errorf("result status = %q, want ok", got.Status)
	}
	if !got.Claimed {
		t.Errorf("expected Claimed=true")
	}
	if got.Err != nil {
		t.Errorf("expected no error, got %v", got.Err)
	}

	// The store recorded a finished run with the handler output present.
	if len(store.outputs) != 1 {
		t.Fatalf("expected one finished run output, got %d", len(store.outputs))
	}
	if !strings.Contains(store.outputs[0], "imported=3 deduped=1") {
		t.Errorf("finished output missing handler output: %q", store.outputs[0])
	}
}

// RC2 — a handler that returns an error yields Status=failed with the error text
// folded into the persisted output.
func TestRunDue_HandlerErrorIsFailed(t *testing.T) {
	now := time.Date(2026, 6, 19, 12, 0, 0, 0, time.UTC)
	store := newMemStore(now)
	store.put(Job{Name: "news-refresh", Enabled: true, IntervalMinutes: 10, LastRunAt: ptrTime(now.Add(-20 * time.Minute))})
	s := New(store, WithNow(func() time.Time { return now }))

	reg := Registry{
		"news-refresh": func(ctx context.Context) (string, error) {
			return "partial work", errors.New("upstream feed unreachable")
		},
	}

	results, err := s.RunDue(context.Background(), reg)
	if err != nil {
		t.Fatalf("RunDue error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	got := results[0]
	if got.Status != RunStatusFailed {
		t.Errorf("result status = %q, want failed", got.Status)
	}
	if got.Err == nil {
		t.Errorf("expected the handler error surfaced on the result")
	}

	if len(store.outputs) != 1 {
		t.Fatalf("expected one finished run output, got %d", len(store.outputs))
	}
	if !strings.Contains(store.outputs[0], "upstream feed unreachable") {
		t.Errorf("expected the error text folded into the persisted output, got %q", store.outputs[0])
	}
}

// RC3 — when the claim is lost (another tick holds the run), the result is
// Skipped and the handler is NOT invoked.
func TestRunOne_ClaimLostIsSkippedNoHandler(t *testing.T) {
	now := time.Date(2026, 6, 19, 12, 0, 0, 0, time.UTC)
	store := newMemStore(now)
	store.put(Job{Name: "db-backup", Enabled: true, IntervalMinutes: 60})
	// Simulate a concurrent tick already holding an open run → ClaimRun returns
	// claimed=false (the single-flight guard).
	store.open["db-backup"] = true
	s := New(store, WithNow(func() time.Time { return now }))

	var invoked bool
	reg := Registry{
		"db-backup": func(ctx context.Context) (string, error) {
			invoked = true
			return "should not run", nil
		},
	}

	got, ok := s.RunOne(context.Background(), reg, "db-backup")
	if !ok {
		t.Fatalf("expected ok=true for a registered job")
	}
	if got.Status != RunStatusSkipped {
		t.Errorf("result status = %q, want skipped", got.Status)
	}
	if got.Claimed {
		t.Errorf("expected Claimed=false when the claim is lost")
	}
	if invoked {
		t.Errorf("handler must NOT be invoked when the claim is lost")
	}
	if got.Err != nil {
		t.Errorf("a lost claim is a skip, not an error; got %v", got.Err)
	}
	if len(store.outputs) != 0 {
		t.Errorf("expected no finished run on a skip, got %d", len(store.outputs))
	}
}

// RC4 — RunOne for a name that is not in the registry returns ok=false and
// creates no run.
func TestRunOne_UnknownNameNoRun(t *testing.T) {
	now := time.Date(2026, 6, 19, 12, 0, 0, 0, time.UTC)
	store := newMemStore(now)
	store.put(Job{Name: "news-refresh", Enabled: true, IntervalMinutes: 10})
	s := New(store, WithNow(func() time.Time { return now }))

	reg := Registry{
		"news-refresh": func(ctx context.Context) (string, error) { return "", nil },
	}

	got, ok := s.RunOne(context.Background(), reg, "no-such-job")
	if ok {
		t.Fatalf("expected ok=false for an unregistered name")
	}
	if got.Name != "no-such-job" {
		t.Errorf("result name = %q, want no-such-job", got.Name)
	}
	if got.Claimed {
		t.Errorf("expected no claim for an unknown job")
	}
	if len(store.outputs) != 0 {
		t.Errorf("expected no finished run for an unknown job, got %d", len(store.outputs))
	}
}

// RC5 — RunDue with a registry whose only job is NOT due returns zero results
// (Due returns nothing) and never invokes the handler.
func TestRunDue_NotDueZeroResults(t *testing.T) {
	now := time.Date(2026, 6, 19, 12, 0, 0, 0, time.UTC)
	store := newMemStore(now)
	// last run only 5m ago, interval 30m → not due.
	store.put(Job{Name: "news-refresh", Enabled: true, IntervalMinutes: 30, LastRunAt: ptrTime(now.Add(-5 * time.Minute))})
	s := New(store, WithNow(func() time.Time { return now }))

	var invoked bool
	reg := Registry{
		"news-refresh": func(ctx context.Context) (string, error) {
			invoked = true
			return "", nil
		},
	}

	results, err := s.RunDue(context.Background(), reg)
	if err != nil {
		t.Fatalf("RunDue error: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected zero results for a not-due job, got %d", len(results))
	}
	if invoked {
		t.Errorf("handler must NOT be invoked when the job is not due")
	}
}

// finishErrStore wraps memStore but makes FinishRun fail, to exercise runOne's
// record-failure transition (a handler that SUCCEEDED but whose outcome cannot be
// persisted). DueJobs/JobByName/ClaimRun are promoted from the embedded *memStore;
// only FinishRun is overridden.
type finishErrStore struct {
	*memStore
	finishErr error
}

func (s finishErrStore) FinishRun(ctx context.Context, runID string, finishedAt time.Time, status RunStatus, output string) error {
	return s.finishErr
}

// RC6 — a handler that SUCCEEDS but whose run cannot be RECORDED (FinishRun errors)
// yields Status=failed with the finish error surfaced; the run is left OPEN (the
// store's stale-run lease reclaims it). Locks the documented record-failure
// contract (NOTES.md M2) so a future edit can't silently turn it into a false ok.
func TestRunOne_FinishErrorIsFailed(t *testing.T) {
	now := time.Date(2026, 6, 19, 12, 0, 0, 0, time.UTC)
	mem := newMemStore(now)
	mem.put(Job{Name: "news-refresh", Enabled: true, IntervalMinutes: 10, LastRunAt: ptrTime(now.Add(-20 * time.Minute))})
	store := finishErrStore{memStore: mem, finishErr: errors.New("store unavailable")}
	s := New(store, WithNow(func() time.Time { return now }))

	var calls int
	reg := Registry{
		"news-refresh": func(ctx context.Context) (string, error) {
			calls++
			return "imported=5", nil // handler SUCCEEDS — only the recording fails
		},
	}

	got, ok := s.RunOne(context.Background(), reg, "news-refresh")
	if !ok {
		t.Fatalf("expected ok=true for a registered job")
	}
	if calls != 1 {
		t.Fatalf("expected handler invoked once, got %d", calls)
	}
	if got.Status != RunStatusFailed {
		t.Errorf("a run we could not record IS a failed run, even on a successful handler; status = %q, want failed", got.Status)
	}
	if got.Err == nil {
		t.Errorf("expected the FinishRun error surfaced on the result")
	}
	if !got.Claimed {
		t.Errorf("expected Claimed=true (the run WAS claimed and the handler executed)")
	}
	if got.RunID == "" {
		t.Errorf("expected the claimed run id on the result")
	}
	// FinishRun failed → nothing persisted, and the open run is left for the lease.
	if len(mem.outputs) != 0 {
		t.Errorf("expected no persisted output when FinishRun fails, got %d", len(mem.outputs))
	}
	if !mem.open["news-refresh"] {
		t.Errorf("expected the run left OPEN when FinishRun fails (stale-run lease reclaims it)")
	}
}
