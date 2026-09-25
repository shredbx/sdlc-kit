package scheduler

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/shredbx/sbx-core/pkg/database"
)

// PostgresStore is a GENERIC, schema-parameterized Store so cmd/scheduler is
// self-contained for ANY consumer: it queries the agreed
// <schema>.scheduler_jobs / <schema>.scheduler_runs shape that the consumer
// creates in its own migration (D4-B). It carries ZERO domain knowledge.
type PostgresStore struct {
	db     *database.DB
	schema string
}

// compile-time assertion that PostgresStore satisfies Store.
var _ Store = (*PostgresStore)(nil)

// StaleRunLease is how long an OPEN run may sit before ClaimRun treats it as
// crashed and reclaimable. The advisory lock makes the check-and-insert atomic and
// the open-run row guards the run DURATION; this lease is the backstop that lets a
// genuinely crashed run (process died before FinishRun) be re-claimed instead of
// blocking the job forever.
const StaleRunLease = 30 * time.Minute

// NewPostgresStore wires the store to a database connection + the schema that owns
// the scheduler_jobs / scheduler_runs tables. The schema is a code-controlled
// value (the consumer's own schema name), never user input; it is quoted into the
// table identifiers (which cannot be parameterized) while every value is a bound
// parameter.
func NewPostgresStore(db *database.DB, schema string) *PostgresStore {
	return &PostgresStore{db: db, schema: schema}
}

// jobsTable / runsTable return the quoted, schema-qualified table identifiers.
func (s *PostgresStore) jobsTable() string {
	return fmt.Sprintf(`%q.scheduler_jobs`, s.schema)
}

func (s *PostgresStore) runsTable() string {
	return fmt.Sprintf(`%q.scheduler_runs`, s.schema)
}

// DueJobs returns enabled jobs that are interval-due OR run-requested, whose name
// is in registered, and that have no currently-open (fresh) run. The open-run
// exclusion uses the same staleness lease as ClaimRun so a crashed run does not
// hide its job from the next tick. The interval/requested gate is expressed in SQL
// (NOW() - last_run_at >= interval) so it stays a single set-based query.
func (s *PostgresStore) DueJobs(ctx context.Context, now time.Time, registered []ScheduleJobName) ([]Job, error) {
	if len(registered) == 0 {
		return nil, nil
	}
	names := make([]string, len(registered))
	for i, n := range registered {
		names[i] = n.String()
	}

	sql := fmt.Sprintf(`
		SELECT j.name, j.enabled, j.interval_minutes, j.last_run_at,
		       j.last_status, j.last_error, j.run_requested_at, j.params, j.updated_at
		FROM %s j
		WHERE j.name = ANY($1)
		  AND j.enabled
		  AND (
		        j.run_requested_at IS NOT NULL
		     OR j.last_run_at IS NULL
		     OR ($2::timestamptz - j.last_run_at) >= make_interval(mins => j.interval_minutes)
		  )
		  AND NOT EXISTS (
		        SELECT 1 FROM %s r
		        WHERE r.job_name = j.name
		          AND r.status = $3
		          AND r.finished_at IS NULL
		          AND r.started_at > ($2::timestamptz - make_interval(mins => $4))
		  )
		ORDER BY j.name
	`, s.jobsTable(), s.runsTable())

	rows, err := s.db.Query(ctx, sql, names, now, RunStatusRunning.String(), int(StaleRunLease.Minutes()))
	if err != nil {
		return nil, fmt.Errorf("scheduler: due jobs: %w", err)
	}
	defer rows.Close()

	var out []Job
	for rows.Next() {
		j, scanErr := scanJob(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, j)
	}
	return out, rows.Err()
}

// JobByName loads one job. found is false (no error) when no such row exists.
func (s *PostgresStore) JobByName(ctx context.Context, name ScheduleJobName) (Job, bool, error) {
	sql := fmt.Sprintf(`
		SELECT name, enabled, interval_minutes, last_run_at,
		       last_status, last_error, run_requested_at, params, updated_at
		FROM %s
		WHERE name = $1
	`, s.jobsTable())

	row := s.db.QueryRow(ctx, sql, name.String())
	j, err := scanJob(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Job{}, false, nil
		}
		return Job{}, false, fmt.Errorf("scheduler: job by name: %w", err)
	}
	return j, true, nil
}

// ClaimRun atomically claims a single-flight run for a job. It runs inside a
// transaction so the advisory lock is xact-scoped:
//
//  1. pg_try_advisory_xact_lock(hashtext('scheduler:'||name)) — non-blocking. If
//     false, another tick holds the lock → claimed=false (skip).
//  2. With the lock held, verify there is no FRESH open run (status=running,
//     finished_at IS NULL, started_at within the staleness lease). A run older
//     than the lease is a crash and is reclaimable.
//  3. INSERT a new running run, clear the job's run_requested_at, commit.
//
// The advisory lock makes the check-and-insert atomic; the open-run row guards the
// run duration; the lease reclaims crashed runs (Decision #0306 threat T4).
func (s *PostgresStore) ClaimRun(ctx context.Context, name ScheduleJobName) (string, RunTrigger, bool, error) {
	var runID string
	var claimed bool
	trigger := TriggerSchedule

	err := s.db.WithTx(ctx, func(tx pgx.Tx) error {
		// (1) Non-blocking advisory lock, scoped to this transaction.
		var locked bool
		if err := tx.QueryRow(ctx,
			`SELECT pg_try_advisory_xact_lock(hashtext($1))`,
			"scheduler:"+name.String(),
		).Scan(&locked); err != nil {
			return fmt.Errorf("scheduler: advisory lock: %w", err)
		}
		if !locked {
			claimed = false
			return nil
		}

		// (2) Guard the run DURATION: bail if a fresh open run already exists.
		var openRuns int
		openSQL := fmt.Sprintf(`
			SELECT COUNT(*) FROM %s
			WHERE job_name = $1 AND status = $2 AND finished_at IS NULL
			  AND started_at > (NOW() - make_interval(mins => $3))
		`, s.runsTable())
		if err := tx.QueryRow(ctx, openSQL,
			name.String(), RunStatusRunning.String(), int(StaleRunLease.Minutes()),
		).Scan(&openRuns); err != nil {
			return fmt.Errorf("scheduler: open-run check: %w", err)
		}
		if openRuns > 0 {
			claimed = false
			return nil
		}

		// (3) Learn the TRIGGER before clearing it (2607-004 #6c): a set
		// run_requested_at means this claim serves an admin "Run now" (manual);
		// otherwise the interval made it due (schedule). Read + clear happen in
		// this same advisory-locked transaction, so the answer can't race a
		// concurrent RequestRun into a lost label.
		var wasRequested bool
		requestedSQL := fmt.Sprintf(`
			SELECT run_requested_at IS NOT NULL FROM %s WHERE name = $1
		`, s.jobsTable())
		if err := tx.QueryRow(ctx, requestedSQL, name.String()).Scan(&wasRequested); err != nil {
			return fmt.Errorf("scheduler: read run-requested: %w", err)
		}
		if wasRequested {
			trigger = TriggerManual
		}

		// (4) Insert the running run and clear the manual request flag.
		runID = uuid.NewString()
		insertSQL := fmt.Sprintf(`
			INSERT INTO %s (id, job_name, started_at, status)
			VALUES ($1, $2, NOW(), $3)
		`, s.runsTable())
		if _, err := tx.Exec(ctx, insertSQL, runID, name.String(), RunStatusRunning.String()); err != nil {
			return fmt.Errorf("scheduler: insert run: %w", err)
		}

		clearSQL := fmt.Sprintf(`
			UPDATE %s SET run_requested_at = NULL, updated_at = NOW()
			WHERE name = $1
		`, s.jobsTable())
		if _, err := tx.Exec(ctx, clearSQL, name.String()); err != nil {
			return fmt.Errorf("scheduler: clear run-requested: %w", err)
		}

		claimed = true
		return nil
	})
	if err != nil {
		return "", TriggerSchedule, false, err
	}
	return runID, trigger, claimed, nil
}

// FinishRun closes a run with its outcome and (already redacted+bounded) output,
// and rolls the result onto the job (last_run_at, last_status, last_error). Both
// writes happen in one transaction so the run record and the job's denormalized
// last-state never diverge. last_error is set from the output only on a failed
// status; a successful run clears it.
func (s *PostgresStore) FinishRun(ctx context.Context, runID string, finishedAt time.Time, status RunStatus, output string) error {
	return s.db.WithTx(ctx, func(tx pgx.Tx) error {
		// Close the run and learn which job it belongs to.
		var jobName string
		finishSQL := fmt.Sprintf(`
			UPDATE %s
			SET finished_at = $2, status = $3, output = $4
			WHERE id = $1
			RETURNING job_name
		`, s.runsTable())
		if err := tx.QueryRow(ctx, finishSQL, runID, finishedAt, status.String(), output).Scan(&jobName); err != nil {
			return fmt.Errorf("scheduler: finish run: %w", err)
		}

		lastError := ""
		if status == RunStatusFailed {
			lastError = output
		}
		jobSQL := fmt.Sprintf(`
			UPDATE %s
			SET last_run_at = $2, last_status = $3, last_error = $4, updated_at = NOW()
			WHERE name = $1
		`, s.jobsTable())
		if _, err := tx.Exec(ctx, jobSQL, jobName, finishedAt, status.String(), lastError); err != nil {
			return fmt.Errorf("scheduler: roll job state: %w", err)
		}
		return nil
	})
}

// rowScanner is the single-row Scan surface shared by pgx.Row and pgx.Rows, so
// scanJob serves both JobByName (Row) and DueJobs (Rows).
type rowScanner interface {
	Scan(dest ...any) error
}

// scanJob reads one scheduler_jobs row into a Job, mapping nullable columns
// (last_run_at, run_requested_at, last_status, last_error, params) safely.
func scanJob(row rowScanner) (Job, error) {
	var (
		j          Job
		name       string
		lastStatus *string
		lastError  *string
		lastRunAt  *time.Time
		requested  *time.Time
		params     []byte
	)
	if err := row.Scan(
		&name, &j.Enabled, &j.IntervalMinutes, &lastRunAt,
		&lastStatus, &lastError, &requested, &params, &j.UpdatedAt,
	); err != nil {
		return Job{}, err
	}
	j.Name = ScheduleJobName(name)
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
