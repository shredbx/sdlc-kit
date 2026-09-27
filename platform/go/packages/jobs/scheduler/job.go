// Package scheduler is the UNIVERSAL timing tool for cron-controllable jobs
// (Decision #0306). Its only concern is WHEN: "is this named job due?", claim a
// run, record the outcome. It NEVER executes domain work and carries ZERO
// exec.Command — execution stays tier-owned in the image that owns the capability
// (the backup image's make-target, an api runner's Go handler). The web controls
// timings by writing scheduler_jobs rows; this engine reads them.
//
// The due decision reuses pkg/feed.DueNow verbatim (the proven
// refresh-feeds --if-due self-throttle). Storage is injected via a Store interface
// (mirror rss.Store + WithStore) so the engine never hard-couples to postgres —
// a consumer injects NewPostgresStore; tests inject an in-memory stub.
package scheduler

import (
	"encoding/json"
	"time"
)

// ScheduleJobName is the stable identifier of a scheduled job (e.g.
// "news-refresh", "db-backup"). A named type — never a raw string — so the job
// vocabulary is explicit and a registry / store can key on it without string
// literals scattered across consumers.
type ScheduleJobName string

// String returns the underlying job name (for logging, SQL params, hashtext keys).
func (n ScheduleJobName) String() string { return string(n) }

// RunStatus is the small, dictionary-like vocabulary a run's lifecycle moves
// through. A named type (never a raw string) so the status set is closed and
// consumers branch on the consts, never on literals.
type RunStatus string

const (
	// RunStatusRunning marks a claimed-but-unfinished run (the open-run guard).
	RunStatusRunning RunStatus = "running"
	// RunStatusOK marks a run that finished successfully.
	RunStatusOK RunStatus = "ok"
	// RunStatusFailed marks a run that finished with an error.
	RunStatusFailed RunStatus = "failed"
	// RunStatusSkipped marks a tick that intentionally did no work (not due / disabled).
	RunStatusSkipped RunStatus = "skipped"
)

// String returns the underlying status code (for SQL params / logging).
func (s RunStatus) String() string { return string(s) }

// RunTrigger is WHY a claimed run became due — a named discriminator (never a
// raw string) the executing recipe can label its artifact by (2607-004 batch 3
// #6c: backup object names carry their trigger — deploy/schedule/manual).
type RunTrigger string

const (
	// TriggerSchedule — the job's interval elapsed (the ordinary cron tick).
	TriggerSchedule RunTrigger = "schedule"
	// TriggerManual — an admin "Run now" (run_requested_at was set at claim).
	TriggerManual RunTrigger = "manual"
)

// String returns the underlying trigger code (for recipe env / logging).
func (t RunTrigger) String() string { return string(t) }

// Job is one cron-controllable job's persisted timing state. The admin/web
// controls ONLY the timing knobs (Enabled, IntervalMinutes, RunRequestedAt) —
// never a command, target, or executable params — so no admin input ever reaches
// an executor (Decision #0306, threat T1 removed by construction). Params is an
// opaque, image-interpreted payload (e.g. which feed set), never a shell string.
type Job struct {
	// Name is the stable job identifier the runner/registry keys on.
	Name ScheduleJobName
	// Enabled gates whether the job participates in the interval schedule.
	Enabled bool
	// IntervalMinutes is the configured cadence floor (clamp via ClampInterval on save).
	IntervalMinutes int
	// LastRunAt is when the job last completed a run (nil = never run).
	LastRunAt *time.Time
	// LastStatus is the outcome of the most recent finished run.
	LastStatus RunStatus
	// LastError is the (redacted, bounded) error from the most recent failed run.
	LastError string
	// RunRequestedAt records a manual "Run now" request; non-nil forces the job
	// due on the next tick regardless of interval, then is cleared on claim.
	RunRequestedAt *time.Time
	// Params is an opaque, image-interpreted JSON payload (never an executable
	// string). The scheduler carries it but never inspects or runs it.
	Params json.RawMessage
	// UpdatedAt is the last time the job's timing config changed.
	UpdatedAt time.Time
}

// Run is one execution record of a job: a claimed run, later finished with an
// outcome and a (redacted, bounded) output.
type Run struct {
	// ID is the run's unique identifier (assigned by the store on claim).
	ID string
	// JobName is the job this run belongs to.
	JobName ScheduleJobName
	// StartedAt is when the run was claimed.
	StartedAt time.Time
	// FinishedAt is when the run completed (nil while running).
	FinishedAt *time.Time
	// Status is the run's current lifecycle status.
	Status RunStatus
	// Output is the redacted, bounded run output/log tail.
	Output string
}

// MinIntervalMinutes is the workspace floor for a job's cadence. It mitigates the
// schedule-frequency DoS (Decision #0306 threat T11): a cadence is clamped to at
// least this many minutes on save, so no admin can drive a job to run unboundedly.
const MinIntervalMinutes = 1

// ClampInterval raises any cadence below the workspace floor to MinIntervalMinutes.
// Consumers call this when persisting a job's IntervalMinutes so the schedule can
// never be driven below the floor.
func ClampInterval(n int) int {
	if n < MinIntervalMinutes {
		return MinIntervalMinutes
	}
	return n
}
