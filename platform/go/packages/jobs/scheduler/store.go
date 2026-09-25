package scheduler

import (
	"context"
	"time"
)

// Store is the storage-agnostic persistence backend the scheduler reads job
// timing from and records run outcomes into. It mirrors the rss.Store idiom
// (Rule #9): the engine never hard-couples to PostgreSQL — a consumer injects a
// postgres-backed adapter (NewPostgresStore); tests inject an in-memory stub.
//
// The Store owns the atomicity of a claim: ClaimRun must be single-flight
// (exactly one concurrent caller claims a given job) — the postgres impl uses
// pg_try_advisory_xact_lock; an in-memory impl uses a mutex.
type Store interface {
	// DueJobs returns the jobs that should run on this tick: enabled AND
	// (interval-elapsed OR run-requested) AND name ∈ registered AND no currently
	// open run. The registered filter lets a runner ask only for the jobs it can
	// actually execute (a go-handler registry / the backup gate's single job).
	DueJobs(ctx context.Context, now time.Time, registered []ScheduleJobName) ([]Job, error)

	// JobByName loads one job's timing state. found is false when no such job row
	// exists (not an error) so a caller can distinguish "unknown job" from a fault.
	JobByName(ctx context.Context, name ScheduleJobName) (Job, bool, error)

	// ClaimRun atomically claims the next run for a job: if another tick already
	// holds a fresh open run, it returns claimed=false; otherwise it inserts a new
	// running run, clears the job's run_requested_at, and returns the new run id
	// with claimed=true. This is the single-flight concurrency guard (threat T4).
	// The returned trigger records WHY the run became due at claim time —
	// TriggerManual when run_requested_at was set (the admin "Run now"),
	// TriggerSchedule otherwise — so the executing recipe can label its
	// artifact (e.g. the backup object name) by trigger (2607-004 batch 3 #6c).
	ClaimRun(ctx context.Context, name ScheduleJobName) (runID string, trigger RunTrigger, claimed bool, err error)

	// FinishRun closes a run with its outcome and (already redacted+bounded) output
	// at finishedAt, and rolls the result onto the job's last_run_at / last_status /
	// last_error so the next IsDue decision and the admin view reflect it.
	FinishRun(ctx context.Context, runID string, finishedAt time.Time, status RunStatus, output string) error
}
