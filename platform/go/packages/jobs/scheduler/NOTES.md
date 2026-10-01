# pkg/scheduler — notes & deferred hardening

Universal cron-control timing tool (Decision #0306). Deferred items from the D4-A code
review (2026-06-19) — non-blocking, recorded for a later pass.

## Redaction (redact.go) — lower-probability secret shapes still pass through
The two highest-probability shapes (libpq `password=…` in a pg error, `?X-Amz-Signature=…`
in a presigned URL, slash/plus DB passwords, and SigV4 *header* forms) are covered + tested.
Still uncaught (MINOR — unlikely in the two in-tree jobs `pg_dump|gzip|rclone` and news
`Refresh()`):
- Space-delimited KV from an rclone *config* dump: `pass = some_value`, `key = value`.
- JSON-quoted secrets: `{"secret_access_key":"…"}`.
- A bare `Signature=…` / `Credential=…` NOT behind an `Authorization:` header or `?&` query.
Fix when a job is added whose output uses these shapes.

## postgres.go
- Schema identifier is interpolated via Go `%q`, not Postgres identifier quoting. Safe today
  (schema is env-sourced/operator-controlled, never request input). If schema ever becomes
  less trusted, switch to `pgx.Identifier{schema, "scheduler_jobs"}.Sanitize()`.
- `PostgresStore` has no live-DB unit coverage (engine gates SC1–SC6 covered by the in-memory
  stub). Add a DB-backed integration test alongside the D4-B migration (exercise the
  `pg_try_advisory_xact_lock` claim + the 30-min stale-run lease + the open-run guard).
- D4-B index suggestion: `(job_name, status, started_at)` to keep the open-run check cheap.

## cmd/scheduler/main.go
- `runIsDue` opens a `PostgresStore` twice (once in `connect()`, once directly) — harmless
  duplication; thread the store through instead.
- CLI **stderr** (`fmt.Fprintf(os.Stderr, …err)`) is NOT redacted. It is redacted on *persist*
  because the documented recipe pipes it into `scheduler finish --output -`. A raw pg error
  (which can embed the DSN) could still surface in raw container logs if stderr is captured
  separately. Wrap the CLI's stderr error print through `Redact` for defense in depth.

## D4-C runner + consumers (2026-06-20) — deferred items

### runner.go (the generic execution loop)
- A `Finish` error on a successful run is reported as `Status=Failed` with the Finish err, but
  the persisted run may be left OPEN (FinishRun didn't commit). The postgres store's stale-run
  lease (30 min) reclaims it on a later tick, so it self-heals — but the immediate RunResult
  says Failed while the job's last_run_at was NOT advanced. Acceptable (a failed Finish IS a
  failure to record), noted so a future "retry Finish" path is a conscious choice, not a gap.
- No per-job panic recovery: a handler that panics takes down the whole RunDue loop (and the
  claimed run is left open for the lease to reclaim). The two in-tree handlers (news Refresh,
  backup recipe) don't panic; add a `defer recover()` in `runOne` only if a riskier handler is
  registered.

### BR cmd/scheduler-run (the api go-handler runner)
- **stderr is intentionally terse** (`job <name> failed`, never the raw error) so a DSN-bearing
  error can't leak to a log scraper; the FULL (redacted+bounded) error is persisted by the
  engine on Finish. Same defense-in-depth posture as the CLI note above — verify no future edit
  prints `r.Err` to stdout/stderr.
- `internal/newsfeed` is the de-dup target for refresh-feeds' wiring (the `// transient` is
  resolved by EXTRACTION, not inline). When refresh-feeds is deleted in D4-F, its
  `newsR2ObjectStore()` helper + service construction are already homed here — just delete the
  old binary, no wiring move needed.

### backup.mk backup-r2-scheduled (the make gate)
- `make -n` (dry-run) EXECUTES the recipe for real because it contains `$(MAKE)` (GNU make's
  dry-run-recursion semantics). So `-n` validated syntax AND exercised the skip/claim-fail
  guards live — but it is NOT a pure no-op. A future reader debugging this target with `-n`
  should expect the `is-due`/`start` shell branch to actually run.
- The gate shells to `$(SCHEDULER_BIN)` (a Makefile recipe → CLI binary, allowed; NOT Go
  exec.Command). It does not flock independently — backup-r2 itself holds the FD-200 lock, and
  the scheduler's single-flight claim is the cross-process guard, so a double-claim is a
  Skipped no-op rather than a corrupt dump.
- `is-due` exit-code branching (review H1 fix): the gate now treats ONLY CLI exit 1 as a clean
  "not due → skip"; exit 2 (unknown/unseeded `db-backup` job) fails LOUD (exit 1) so a missing
  seed / typo / wrong-SCHEMA can never silently drop every backup. RESIDUAL: `cmd/scheduler`'s
  `connect()` also exits 1 on a DB-connection failure (it shares the "not due" code), so a
  TRANSIENT DB outage at is-due time is treated as a skip and retried on the next base-rate
  tick — acceptable (self-healing). If a distinct connect-failure exit code is ever added to the
  CLI, branch it here as a loud failure too.

## D4-D admin control plane (2026-06-20) — deferred items

### AdminJob.Kind is a loose string, intentionally (engine vs UI coupling)
`AdminJob.Kind` is a bare `string`, not a named type, and the consumer's `resolveJobMeta`
(BR `schedule_manage.go`) switches on the raw literals `"go-handler"` / `"make-target"`.
This is DELIBERATE: `kind` is a UI-only `scheduler_jobs` column the engine never selects or
branches on (entity schedule-job) — promoting it to a named `ScheduleJobKind` in `pkg/scheduler`
would couple the generic engine to a discriminator it doesn't use. RESIDUAL drift risk: the
`go-handler|make-target` vocabulary now lives in THREE places that must stay in sync — the
entity.yml `kind.values`, the BR migration's `kind` CHECK (D4-B), and the consumer's
`resolveJobMeta` switch. A future task that adds a third kind must touch all three. If a fourth
consumer ever needs to branch on kind, promote it to a named type in `pkg/scheduler` then (one
owner, not three) — not before (YAGNI; today only the BR admin presentation reads it).

### jobMeta presentation map lives in the consumer (BR), not the engine
The human owner/description per job (`jobMeta` in BR `schedule_manage.go`) is consumer-owned, so
adding a scheduled job is one map line in the BR handler + a seed row — zero engine change. An
UNKNOWN job (no jobMeta entry) degrades to a kind-derived owner + empty description, never an
error, so a newly-seeded job renders before its metadata line is added. Acceptable; noted so the
empty-description fallback is a conscious default, not a gap.
