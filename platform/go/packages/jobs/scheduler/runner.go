package scheduler

import (
	"context"
	"fmt"
)

// Handler runs one scheduled job's domain work and returns a human/machine log
// summary plus any error. It is the ONLY place domain knowledge lives — the
// engine never inspects what a handler does. The returned output is persisted
// through Finish, which redacts (DATABASE_URL passwords, presigned signatures,
// secret env assignments) and bounds it, so a handler may freely log a report
// without leaking a secret or an unbounded blob (Decision #0306).
type Handler func(ctx context.Context) (output string, err error)

// Registry maps the jobs an image can actually execute to their handlers. A
// runner asks the engine ONLY for the names in its registry (Due's registered
// filter), so a go-handler image never claims a job it cannot run.
type Registry map[ScheduleJobName]Handler

// keys returns the registered job names — the set a runner offers to Due so the
// engine filters to jobs THIS image can execute.
func (r Registry) keys() []ScheduleJobName {
	names := make([]ScheduleJobName, 0, len(r))
	for name := range r {
		names = append(names, name)
	}
	return names
}

// RunResult is the per-job outcome of a runner pass: which job, the claimed run
// id (empty when no run was claimed), whether THIS runner won the claim, the
// terminal status the run was finished with, and any error encountered along the
// way (claim, handler, or finish). A claim lost to a concurrent tick is a Skipped
// result with no error — not a failure.
type RunResult struct {
	// Name is the job this result is for.
	Name ScheduleJobName
	// RunID is the claimed run's id (empty when the claim was lost or failed).
	RunID string
	// Claimed is true when this runner won the single-flight claim and executed.
	Claimed bool
	// Status is the terminal status the run was finished with (or Skipped/Failed
	// when no run executed).
	Status RunStatus
	// Err carries any claim/handler/finish error (nil on a clean ok or skip).
	Err error
}

// RunDue runs every job that is due on this tick AND registered in reg. It asks
// the engine only for the registry's own job names (so this image never claims a
// job it cannot run), then drives each returned job through the shared single-
// flight runOne. A job that is due but has no handler in reg cannot be returned
// by Due (it was never offered), so every due job here has a handler.
func (s *Scheduler) RunDue(ctx context.Context, reg Registry) ([]RunResult, error) {
	due, err := s.Due(ctx, s.now(), reg.keys())
	if err != nil {
		return nil, fmt.Errorf("scheduler: run due: %w", err)
	}
	results := make([]RunResult, 0, len(due))
	for _, job := range due {
		// Every name Due returned came from reg.keys(), so the lookup is present.
		results = append(results, s.runOne(ctx, job.Name, reg[job.Name]))
	}
	return results, nil
}

// RunOne is the manual single-job path (an operator "Run now"): it claims a
// single-flight run for name and executes its handler through the SAME runOne as
// the due loop, so a manual trigger can never collide with a due tick (the claim
// is the guard). ok is false when name is not registered in reg — the caller
// distinguishes "unknown job" from a real outcome. A claim lost to a concurrent
// tick still returns ok=true with a Skipped result.
func (s *Scheduler) RunOne(ctx context.Context, reg Registry, name ScheduleJobName) (RunResult, bool) {
	h, registered := reg[name]
	if !registered {
		return RunResult{Name: name}, false
	}
	return s.runOne(ctx, name, h), true
}

// runOne is the shared execution unit for both the due loop and the manual path.
// It claims a single-flight run, executes the handler, folds any handler error
// text INTO the output (so Finish redacts it too), and finishes the run with the
// terminal status. The state machine:
//
//   - Start error            → Failed, Err set, no run to finish.
//   - claim lost (!claimed)  → Skipped, handler NOT invoked (another tick owns it).
//   - handler ok             → OK,     output persisted.
//   - handler error          → Failed, error text folded into the persisted output.
//   - Finish error           → Failed, Err set (the run may be left open for the
//     staleness lease to reclaim).
func (s *Scheduler) runOne(ctx context.Context, name ScheduleJobName, h Handler) RunResult {
	runID, _, claimed, err := s.Start(ctx, name)
	if err != nil {
		return RunResult{Name: name, Status: RunStatusFailed, Err: err}
	}
	if !claimed {
		// A concurrent tick already holds the run — a skip, never a failure.
		return RunResult{Name: name, Status: RunStatusSkipped}
	}

	output, runErr := h(ctx)
	status := RunStatusOK
	if runErr != nil {
		status = RunStatusFailed
		// Fold the error text into the output so Finish redacts+bounds it too,
		// and the persisted run carries the failure context (last_error mirrors
		// the failed output in the store).
		if output != "" {
			output += "\n"
		}
		output += "error: " + runErr.Error()
	}

	if finishErr := s.Finish(ctx, runID, status, output); finishErr != nil {
		return RunResult{Name: name, RunID: runID, Claimed: true, Status: RunStatusFailed, Err: finishErr}
	}

	return RunResult{Name: name, RunID: runID, Claimed: true, Status: status, Err: runErr}
}
