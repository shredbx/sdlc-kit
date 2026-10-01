package scheduler

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// compile-time assertion that PostgresStore satisfies the admin contract too — the
// same store backs both the runner (Store) and the control plane (AdminStore).
var _ AdminStore = (*PostgresStore)(nil)

// adminColumns is the admin SELECT/RETURNING column list: the full scanJob shape
// PLUS the UI-only `kind` column the runner's queries deliberately omit. Kept as a
// single const so ListJobs/UpdateTiming/RequestRun stay in lockstep with scanAdminJob.
const adminColumns = `name, kind, enabled, interval_minutes, last_run_at,
	last_status, last_error, run_requested_at, params, updated_at`

// ListJobs returns ALL jobs (enabled + disabled), ordered by name, for the admin
// table — including the UI-only kind discriminator.
func (s *PostgresStore) ListJobs(ctx context.Context) ([]AdminJob, error) {
	sql := fmt.Sprintf(`SELECT %s FROM %s ORDER BY name`, adminColumns, s.jobsTable())
	rows, err := s.db.Query(ctx, sql)
	if err != nil {
		return nil, fmt.Errorf("scheduler: list jobs: %w", err)
	}
	defer rows.Close()

	var out []AdminJob
	for rows.Next() {
		j, scanErr := scanAdminJob(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, j)
	}
	return out, rows.Err()
}

// UpdateTiming writes ONLY the timing knobs (enabled + clamped interval) for one
// job and returns the updated row. The interval is clamped to the workspace floor
// via ClampInterval BEFORE the UPDATE (the store-side backstop — a sub-floor value
// can never reach the table). A no-row RETURNING (unknown job) → found=false.
func (s *PostgresStore) UpdateTiming(ctx context.Context, name ScheduleJobName, enabled bool, intervalMinutes int) (AdminJob, bool, error) {
	interval := ClampInterval(intervalMinutes)
	sql := fmt.Sprintf(`
		UPDATE %s
		SET enabled = $2, interval_minutes = $3, updated_at = NOW()
		WHERE name = $1
		RETURNING %s
	`, s.jobsTable(), adminColumns)

	row := s.db.QueryRow(ctx, sql, name.String(), enabled, interval)
	j, err := scanAdminJob(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AdminJob{}, false, nil
		}
		return AdminJob{}, false, fmt.Errorf("scheduler: update timing: %w", err)
	}
	return j, true, nil
}

// RequestRun queues a manual "Run now" for one job (run_requested_at=NOW()) and
// returns the updated row. The next tick forces the job due regardless of interval,
// then clears the flag on claim. A no-row RETURNING (unknown job) → found=false.
func (s *PostgresStore) RequestRun(ctx context.Context, name ScheduleJobName) (AdminJob, bool, error) {
	sql := fmt.Sprintf(`
		UPDATE %s
		SET run_requested_at = NOW(), updated_at = NOW()
		WHERE name = $1
		RETURNING %s
	`, s.jobsTable(), adminColumns)

	row := s.db.QueryRow(ctx, sql, name.String())
	j, err := scanAdminJob(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AdminJob{}, false, nil
		}
		return AdminJob{}, false, fmt.Errorf("scheduler: request run: %w", err)
	}
	return j, true, nil
}

// runColumns is the scheduler_runs SELECT column list for the admin run-history
// read — the full scheduler.Run shape, kept as a single const so RecentRuns stays
// in lockstep with scanRun.
const runColumns = `id, job_name, started_at, finished_at, status, output`

// RecentRuns returns a job's most recent runs (newest first), capped at limit.
// A non-positive limit yields none (no query). An unknown job (or one with no
// runs) returns an empty slice — the read is domain-free, so the consumer parses
// any meaning out of a run's output tail.
func (s *PostgresStore) RecentRuns(ctx context.Context, name ScheduleJobName, limit int) ([]Run, error) {
	if limit <= 0 {
		return nil, nil
	}
	sql := fmt.Sprintf(
		`SELECT %s FROM %s WHERE job_name = $1 ORDER BY started_at DESC LIMIT $2`,
		runColumns, s.runsTable(),
	)
	rows, err := s.db.Query(ctx, sql, name.String(), limit)
	if err != nil {
		return nil, fmt.Errorf("scheduler: recent runs: %w", err)
	}
	defer rows.Close()

	var out []Run
	for rows.Next() {
		r, scanErr := scanRun(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// scanRun reads one scheduler_runs row into a Run, mapping the nullable
// finished_at safely (nil while a run is still open).
func scanRun(row rowScanner) (Run, error) {
	var (
		r          Run
		jobName    string
		status     string
		finishedAt *time.Time
	)
	if err := row.Scan(&r.ID, &jobName, &r.StartedAt, &finishedAt, &status, &r.Output); err != nil {
		return Run{}, err
	}
	r.JobName = ScheduleJobName(jobName)
	r.FinishedAt = finishedAt
	r.Status = RunStatus(status)
	return r, nil
}

// scanAdminJob reads one admin row (the scanJob shape + the UI-only kind column)
// into an AdminJob, mapping nullable columns safely — mirroring scanJob but with
// `kind` interleaved after `name`.
func scanAdminJob(row rowScanner) (AdminJob, error) {
	var (
		j          AdminJob
		name       string
		kind       string
		lastStatus *string
		lastError  *string
		lastRunAt  *time.Time
		requested  *time.Time
		params     []byte
	)
	if err := row.Scan(
		&name, &kind, &j.Enabled, &j.IntervalMinutes, &lastRunAt,
		&lastStatus, &lastError, &requested, &params, &j.UpdatedAt,
	); err != nil {
		return AdminJob{}, err
	}
	j.Name = ScheduleJobName(name)
	j.Kind = kind
	j.LastRunAt = lastRunAt
	j.RunRequestedAt = requested
	if lastStatus != nil {
		j.LastStatus = RunStatus(*lastStatus)
	}
	if lastError != nil {
		j.LastError = *lastError
	}
	if len(params) > 0 {
		j.Params = params
	}
	return j, nil
}
