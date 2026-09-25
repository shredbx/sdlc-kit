# bos Scope 6b — auth, visitor activity and the job scheduler (plan and log)

Last updated: 2026-09-26
tags: bos, scope-6b, go, port, plan

> Design: `docs/proposals/bos-system-design.md` (table 6.3, section 8). Follows Scope 6a
> (`docs/plans/2026-09-25-bos-scope-6a-go-feature-packages.md`). Introduces **zero** process-os definitions and
> **no new code**: four Go modules copied verbatim; no interface is changed or refined (that happens when a kit
> is sliced, section 8.1 of the design). Approved by the user on 2026-09-26 ("proceed as per planned then") after
> the tree was shown. Nothing is pushed.

## What is copied (verbatim)

Source: `shredbx/projects/sbx/packages/core/go/pkg/<name>` → `platform/go/packages/<kit>/<package>`. Module
paths stay `github.com/shredbx/sbx-core/pkg/<name>`, so no import is rewritten. Two new kits appear when their
first package lands (`analytics`, `jobs`); `identity` already exists.

| Module | Destination | Files (src + test) | Direct in-repo | Third-party direct |
|---|---|---|---|---|
| `pkg/auth` | `identity/auth` | 35 (19 + 16) | `user`, `repository` | `jwt/v5`, `uuid`, `pgx/v5`, `go-redis/v9`, `x/crypto` (bcrypt), `testify` |
| `pkg/visitoractivity` | `analytics/visitoractivity` | 14 (7 + 7) | `httputil` | `uuid`, `pgx/v5`, `go-redis/v9`, `testify` |
| `pkg/scheduler` | `jobs/scheduler` | 13 (8 + 4 + `NOTES.md`) | `database`, `feed` | `uuid`, `pgx/v5` |
| `pkg/scheduler/schedcli` | `jobs/scheduler/schedcli` (nested module) | 1 | `database`, `scheduler` | none |

**63 verbatim files**, plus 4 authored `go.mod` and up to 4 `go.sum`. `go.work` grows from 28 to 32 `use` paths.
`NOTES.md` is documentation that lives in the original package; it is copied like every other file.

## Known edge, ported as is

`jobs/scheduler` imports `news/feed`: a shared kit importing a feature kit, against the dependency rule in
`platform/CLAUDE.md`. It is ported verbatim and inverted in a later scope (the design's known violation).

## Conventions (unchanged from 6a, one addition)

- In-repo dependencies: `require … v0.0.0` plus a relative `replace`; third-party direct versions pinned to the
  original `sbx-core/go.mod`; `go.sum` and indirect requirements from a standalone `go mod tidy`; gofmt findings
  in a separate style commit.
- **Added to `platform/CLAUDE.md` in this scope:** `replace` does not propagate, so a module replaces its whole
  in-repo dependency closure (direct and transitive), computed from the ported `go.mod` files. This was
  discovered in 6a (`vcard` requires two in-repo modules and needs nine `replace` lines).

## Baseline (measured on the original module, 2026-09-26) — pass / skip / fail

`go test -count=1 -json` in the original `sbx-core` module, aggregated per package. `VISITOR_ACTIVITY_TEST_DSN` is
unset, so `visitoractivity`'s 13 Postgres integration tests skip; the port is measured under the same condition.

| Package | Pass | Skip | Fail |
|---|---|---|---|
| auth | 213 | 0 | 0 |
| visitoractivity | 87 | 13 | 0 |
| scheduler | 32 | 0 | 0 |
| scheduler/schedcli | 0 | 0 | 0 (no test files) |
| **Total** | **332** | **13** | **0** |

## Before → after (new and modified paths only)

```
BEFORE                                          AFTER
platform/go/go.work        (28 use paths)       platform/go/go.work        (32 use paths)
platform/go/packages/                           platform/go/packages/
├── identity/  user · rbac                      ├── identity/
│                                               │   └── auth/                NEW  go.mod go.sum · 19 src · 16 test
│                                               ├── analytics/               NEW kit
│                                               │   └── visitoractivity/     NEW  go.mod go.sum · 7 src · 7 test
│                                               └── jobs/                    NEW kit
│                                                   └── scheduler/           NEW  go.mod go.sum · 8 src · 4 test · NOTES.md
│                                                       └── schedcli/        NEW  nested module · 1 src
docs/plans/  (no 6b)                            docs/plans/2026-09-25-bos-scope-6b-go-identity-analytics-jobs.md   NEW
docs/proposals/bos-system-design.md             modified: table 6.3 ✔, tree, section 8, decision log
platform/CLAUDE.md                              modified: the `replace`-closure rule
```

## Tasks

| # | Task | Result expected |
|---|---|---|
| 0 | Commit this plan | one `docs:` commit |
| 1 | Baseline on the original module (table above) | numbers recorded before anything is ported |
| 2 | Copy the four packages | byte-identical to the source |
| 3 | Author four `go.mod` (in-repo `require` + `replace` closure, pinned third-party) | four `go.mod` |
| 4 | `go mod tidy`, parents before nested | `go.sum`, indirect requirements |
| 5 | `go.work`: 28 → 32 `use` entries | — |
| 6 | Verify every gate | evidence below |
| 7 | `platform/CLAUDE.md` rule, design doc, results, commits | — |

## Exit gates

- `diff -r` against the source is empty for every package, excluding the authored `go.mod`/`go.sum`.
- Per-module pass/skip/fail equals the baseline above (**332 / 13 / 0**), each module run on its own, under the same
  environment (DSN unset).
- Every direct third-party requirement equals the `sbx-core/go.mod` pin; a standalone tidy (`GOWORK=off`) is a no-op
  on a second run.
- `go vet` clean across 32 modules; `gofmt -l` empty (findings in their own commit); `go list -m` finds 32.
- The 28 existing modules still give **1,166 / 16 / 0**.
- Staged file count equals the files on disk; `process-cli check` stays clean. Scoped runs only. Nothing pushed.

## Not in this scope

`real-estate/property` and `real-estate/transaction` (6c); inverting `scheduler` → `feed`; any interface change or
refinement; wiring to a framework or an app; declaring kit edges (D15); process-os definitions. Client information
is not touched: it lives in the client library the main session is building, registered after alignment with that
session, never in `platform/`.
