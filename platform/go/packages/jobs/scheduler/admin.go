package scheduler

import "context"

// AdminJob is the admin/web view of a scheduled job: the engine's timing Job plus
// the UI-only `kind` discriminator (go-handler | make-target) that the runner's
// Job deliberately never carries. `kind` is an extra scheduler_jobs column the
// engine never selects (entity schedule-job, Decision #0306) — surfacing it here,
// instead of widening Job/scanJob, keeps the runner path coupling-free: the admin
// queries select `kind` into this thin wrapper, the engine's Job stays unchanged.
type AdminJob struct {
	// Job is the persisted timing state, identical to what the runner reads.
	Job
	// Kind is the UI-only job classifier (go-handler vs make-target). The engine
	// never branches on it; the admin UI uses it to present a job's owner/origin.
	Kind string
}

// AdminStore is the control-plane contract over the scheduler_jobs table — the
// admin/web reads every job and writes ONLY its timing knobs (enabled, interval,
// a queued run). Distinct from the runner's Store (DueJobs/Claim/Finish); a
// consumer's admin API depends on THIS interface, not a concrete store (Rule #9),
// so adding a new scheduled job is a DB row (+ optional UI metadata entry) with
// ZERO handler/store code change. It returns AdminJob (Job + the UI-only kind) so
// the web can classify a job without the engine ever depending on kind.
type AdminStore interface {
	// ListJobs returns ALL jobs (enabled and disabled), ordered by name, for the
	// admin table.
	ListJobs(ctx context.Context) ([]AdminJob, error)

	// UpdateTiming writes ONLY a job's timing knobs (enabled + cadence). The
	// interval is clamped via ClampInterval to the workspace floor before the write
	// (the store-side backstop; a consumer's API should still reject a sub-floor
	// value at its edge). found is false (no error) when no such job row exists.
	UpdateTiming(ctx context.Context, name ScheduleJobName, enabled bool, intervalMinutes int) (AdminJob, bool, error)

	// RequestRun queues a manual "Run now": it sets run_requested_at=NOW() so the
	// job is due on the next tick regardless of interval (then cleared on claim).
	// found is false (no error) when no such job row exists.
	RequestRun(ctx context.Context, name ScheduleJobName) (AdminJob, bool, error)

	// RecentRuns returns a job's most recent execution records, newest first,
	// capped at limit (a non-positive limit yields none). The admin/web reads these
	// to show a job's run history (e.g. the backup page's recent-backups table).
	// The engine stays domain-free: a Run carries only timing + the redacted,
	// bounded output tail — any domain meaning inside output (e.g. an R2 object key
	// echoed by a make-target) is the consumer's to parse, never the engine's.
	// An unknown job (or one with no runs) returns an empty slice, never an error.
	RecentRuns(ctx context.Context, name ScheduleJobName, limit int) ([]Run, error)
}
