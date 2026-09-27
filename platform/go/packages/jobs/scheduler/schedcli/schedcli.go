// scheduler is the tiny, universal timing CLI embedded in ANY image that owns a
// cron-controllable job (Decision #0306). It NEVER runs the job — the owning
// image's recipe / runner does. It communicates purely via exit codes + ids so a
// Makefile recipe can gate itself:
//
//	scheduler is-due <job>                         exit 0 = due, 1 = not due
//	scheduler start  <job>                         claimed → print run id, exit 0
//	                                               not claimed (already running) → exit 75
//	scheduler finish <job> --run <id> --status ok|failed [--output -]
//	                                               --output - reads the run log from stdin
//
// A backup recipe gates itself with:
//
//	scheduler is-due db-backup || exit 0
//	RUN=$(scheduler start db-backup) || exit 0     # exit 75 = skip (already claimed)
//	make backup-r2 2>&1 | scheduler finish db-backup --run "$RUN" --status ok --output -
//
// DB wiring mirrors cmd/refresh-feeds: DATABASE_URL + SCHEMA env, fail fast with a
// clear stderr message + non-zero exit on a missing/invalid value.
package schedcli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/shredbx/sbx-core/pkg/database"
	"github.com/shredbx/sbx-core/pkg/scheduler"
)

// exitSkip is the conventional "skipped — nothing to do" exit code (EX_TEMPFAIL,
// sysexits.h 75): a `start` that did not claim the run (another tick holds it, or
// it is not due). A gating recipe treats it as a clean no-op, distinct from a real
// failure (non-zero, non-75).
const exitSkip = 75

// Main is the scheduler CLI entrypoint — both shells (core cmd/scheduler and a
// consumer app's mirror-buildable cmd/scheduler, e.g. BR's) delegate here so the
// subcommand contract lives in exactly one place. It os.Exits like any CLI body.
func Main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	sub := os.Args[1]
	args := os.Args[2:]

	switch sub {
	case "is-due":
		runIsDue(args)
	case "start":
		runStart(args)
	case "finish":
		runFinish(args)
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: scheduler <is-due|start|finish> <job> [flags]")
	fmt.Fprintln(os.Stderr, "  is-due <job>                                    exit 0 if due, 1 if not")
	fmt.Fprintln(os.Stderr, "  start  <job>                                    print `<run id> <trigger>` (trigger: schedule|manual) + exit 0 if claimed, exit 75 if skipped")
	fmt.Fprintln(os.Stderr, "  finish <job> --run <id> --status ok|failed [--output -]")
}

// connect builds the scheduler over a PostgresStore from DATABASE_URL + SCHEMA.
// It fails fast (stderr + exit 1) on a missing env or a connection error so a
// gating recipe surfaces a setup problem distinctly from a skip.
func connect(ctx context.Context) (*scheduler.Scheduler, *database.DB) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		fmt.Fprintln(os.Stderr, "Error: DATABASE_URL is required")
		os.Exit(1)
	}
	schema := os.Getenv("SCHEMA")
	if schema == "" {
		fmt.Fprintln(os.Stderr, "Error: SCHEMA is required")
		os.Exit(1)
	}

	db, err := database.New(ctx, database.Config{URL: dbURL, Schema: schema})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: database connection failed: %v\n", err)
		os.Exit(1)
	}
	return scheduler.New(scheduler.NewPostgresStore(db, schema)), db
}

// jobArg pulls the positional <job> name (the first non-flag arg) from args.
func jobArg(args []string) scheduler.ScheduleJobName {
	for _, a := range args {
		if len(a) > 0 && a[0] != '-' {
			return scheduler.ScheduleJobName(a)
		}
	}
	fmt.Fprintln(os.Stderr, "Error: <job> name is required")
	os.Exit(2)
	return ""
}

// runIsDue loads the job and exits 0 when due, 1 when not. An unknown job is a
// usage error (exit 2).
func runIsDue(args []string) {
	job := jobArg(args)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	sched, db := connect(ctx)
	defer db.Close()

	store := scheduler.NewPostgresStore(db, os.Getenv("SCHEMA"))
	j, found, err := store.JobByName(ctx, job)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: load job %q: %v\n", job, err)
		os.Exit(1)
	}
	if !found {
		fmt.Fprintf(os.Stderr, "Error: no such job %q\n", job)
		os.Exit(2)
	}

	if sched.IsDue(j, time.Now()) {
		fmt.Println("due")
		os.Exit(0)
	}
	fmt.Println("not due")
	os.Exit(1)
}

// runStart claims a run: on a claim it prints the run id to stdout and exits 0; if
// the run was not claimed (already running / not due) it exits 75 (skip).
func runStart(args []string) {
	job := jobArg(args)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	sched, db := connect(ctx)
	defer db.Close()

	runID, trigger, claimed, err := sched.Start(ctx, job)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: start %q: %v\n", job, err)
		os.Exit(1)
	}
	if !claimed {
		fmt.Fprintf(os.Stderr, "skip: %q already running or not claimable\n", job)
		os.Exit(exitSkip)
	}
	// `<run_id> <trigger>` — the gating recipe splits on the space; trigger is
	// "manual" (admin Run-now claim) or "schedule" (interval), so the executed
	// artifact (e.g. the backup object name) can carry WHY it ran (#6c).
	fmt.Println(runID + " " + trigger.String())
	os.Exit(0)
}

// runFinish closes a run with its status + output. --output - reads the run log
// from stdin (the piped recipe output); the engine redacts + bounds it.
func runFinish(args []string) {
	job := jobArg(args)
	fs := flag.NewFlagSet("finish", flag.ExitOnError)
	runID := fs.String("run", "", "the run id returned by `start`")
	statusStr := fs.String("status", "", "ok | failed")
	output := fs.String("output", "", "run output; '-' reads from stdin")
	// Parse only the flag args (skip the positional job).
	_ = fs.Parse(flagArgs(args))

	if *runID == "" {
		fmt.Fprintln(os.Stderr, "Error: --run <id> is required")
		os.Exit(2)
	}

	status, ok := parseStatus(*statusStr)
	if !ok {
		fmt.Fprintf(os.Stderr, "Error: --status must be ok|failed, got %q\n", *statusStr)
		os.Exit(2)
	}

	out := *output
	if out == "-" {
		raw, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: read stdin: %v\n", err)
			os.Exit(1)
		}
		out = string(raw)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	sched, db := connect(ctx)
	defer db.Close()

	if err := sched.Finish(ctx, *runID, status, out); err != nil {
		fmt.Fprintf(os.Stderr, "Error: finish %q run %s: %v\n", job, *runID, err)
		os.Exit(1)
	}
	os.Exit(0)
}

// parseStatus maps the finish --status flag to a RunStatus. Only the two terminal
// outcomes a runner reports are accepted (ok|failed); running/skipped are internal.
func parseStatus(s string) (scheduler.RunStatus, bool) {
	switch s {
	case "ok":
		return scheduler.RunStatusOK, true
	case "failed":
		return scheduler.RunStatusFailed, true
	default:
		return "", false
	}
}

// flagArgs drops the leading positional <job> arg so flag parsing sees only flags.
func flagArgs(args []string) []string {
	out := make([]string, 0, len(args))
	skippedJob := false
	for _, a := range args {
		if !skippedJob && len(a) > 0 && a[0] != '-' {
			skippedJob = true
			continue
		}
		out = append(out, a)
	}
	return out
}
