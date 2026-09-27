package scheduler

import (
	"context"
	"time"

	"github.com/shredbx/sbx-core/pkg/feed"
)

// Scheduler is the universal timing engine (Decision #0306). It decides whether a
// named job is due, claims a single-flight run, and records its (redacted,
// bounded) outcome — never executing the job itself. Storage is injected via a
// Store (mirror rss.Service + WithStore); the clock is injectable so Finish is
// deterministically testable.
type Scheduler struct {
	store Store
	now   func() time.Time
}

// Option configures a Scheduler at construction (functional-options, mirroring
// rss.Option).
type Option func(*Scheduler)

// WithNow injects the clock used by Finish to stamp finished_at. Tests pin it for
// determinism; production omits it (defaults to time.Now). A nil fn is ignored.
func WithNow(fn func() time.Time) Option {
	return func(s *Scheduler) {
		if fn != nil {
			s.now = fn
		}
	}
}

// New wires the engine over a Store. The store is the only required dependency;
// the clock defaults to time.Now (override with WithNow in tests).
func New(store Store, opts ...Option) *Scheduler {
	s := &Scheduler{
		store: store,
		now:   time.Now,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// IsDue is the pure due decision for a single job at now. A job is due when a
// manual run was requested (RunRequestedAt != nil) OR it is enabled and its
// interval has elapsed since the last run — reusing pkg/feed.DueNow verbatim (the
// proven refresh-feeds --if-due self-throttle).
func (s *Scheduler) IsDue(job Job, now time.Time) bool {
	if job.RunRequestedAt != nil {
		return true
	}
	return feed.DueNow(job.Enabled, job.LastRunAt, job.IntervalMinutes, now)
}

// Due returns the registered jobs that should run on this tick. The filtering
// (enabled ∧ (due ∨ requested) ∧ registered ∧ no-open-run) lives in the Store so
// it stays a single set-based query (the postgres impl) without N+1 round-trips.
func (s *Scheduler) Due(ctx context.Context, now time.Time, registered []ScheduleJobName) ([]Job, error) {
	return s.store.DueJobs(ctx, now, registered)
}

// Start claims a single-flight run for a job. claimed is false when another tick
// already holds a fresh open run (the run is skipped, not an error) — the runner
// exits with a skip code in that case.
func (s *Scheduler) Start(ctx context.Context, name ScheduleJobName) (runID string, trigger RunTrigger, claimed bool, err error) {
	return s.store.ClaimRun(ctx, name)
}

// Finish records a run's outcome. The raw output is redacted (DATABASE_URL
// passwords, presigned signatures, RCLONE_CONFIG_*/*_SECRET* tokens) and bounded
// (tail-kept to a byte cap) BEFORE it touches the store, so no secret and no
// unbounded blob is ever persisted (Decision #0306 threat T6). finished_at is
// stamped from the injected clock.
func (s *Scheduler) Finish(ctx context.Context, runID string, status RunStatus, output string) error {
	clean := BoundOutput(Redact(output), DefaultMaxOutputBytes)
	return s.store.FinishRun(ctx, runID, s.now(), status, clean)
}
