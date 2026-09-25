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

## Results (2026-09-26) — done

| Task | Commit | Evidence |
|---|---|---|
| 0 plan | `7d6f98a` | this document, with the baseline |
| 1 baseline | (scratchpad `go_baseline_6b.json`, table above) | measured on the original module before anything was ported |
| 2–5 copy, `go.mod`, tidy, `go.work` | `dade4a6` | 72 files: **71 new** (63 verbatim + 4 `go.mod` + 4 `go.sum`) and `go.work` modified |
| gofmt (kept separate) | `70d9f9f` | 4 files, 11 insertions / 12 deletions, formatting only |
| 7 `platform/CLAUDE.md` rule, design doc | `2a05921` | table 6.3 and the tree ✔, section 8, M0 9 of about 12, the `replace`-closure rule, decision log |
| results | (this commit) | — |

**Gates**
- **Byte-identity:** `diff -r -x go.mod -x go.sum` against the source is empty for all three top-level trees
  (auth 35, visitoractivity 14, scheduler 14 incl. `schedcli` = 63 files), so nothing was missed either.
  (Superseded for the 4 files gofmt touched, by `70d9f9f`.)
- **Tests, each module on its own, equal to the baseline both before and after gofmt:** auth 213, visitoractivity
  87 pass + 13 skip, scheduler 32, schedcli no test files = **332 pass, 13 skip, 0 fail**, with
  `VISITOR_ACTIVITY_TEST_DSN` unset in both runs. The 32-module run gives **1,498 / 29 / 0**; the 28 earlier
  modules are unchanged (compared programmatically: 1,166 / 16 / 0), and the per-module JSON is identical before
  and after gofmt.
- **Pins:** every direct third-party requirement equals the original `sbx-core/go.mod` (`jwt/v5` v5.3.1,
  `go-redis/v9` v9.18.0, `x/crypto` v0.49.0, `uuid` v1.6.0, `pgx/v5` v5.9.1, `testify` v1.11.1); checked by script.
- **Tidy:** a standalone `go mod tidy` (`GOWORK=off`) succeeds and is a no-op on a second run for all 4 modules.
- `go vet` clean across 32 modules; `gofmt -l` empty after `70d9f9f`; `go list -m` finds 32.
- Staged file count (71 new + `go.work`) equals the files on disk (71). `go.work.sum` did not change.
- `process-cli check`: ok. Nothing pushed. `go-ci.yml` is `go list -m`-based and needed no change.

**`go.mod` shape per module** (direct in-repo requires → full `replace` closure)

| Module | Direct in-repo | `replace` lines | Third-party direct |
|---|---|---|---|
| auth | 2 | 2 | 6 (`jwt/v5`, `uuid`, `pgx/v5`, `go-redis/v9`, `testify`, `x/crypto`) |
| visitoractivity | 1 | 2 | 4 (`uuid`, `pgx/v5`, `go-redis/v9`, `testify`) |
| scheduler | 2 | 3 | 2 (`uuid`, `pgx/v5`) |
| schedcli | 2 | 4 | none |

**Findings on the way**
- **A defect in the original, found by the background security review of `dade4a6`, confirmed by reading and
  then reproduced in a scratch copy (2026-09-26).** `auth_service.go` stores `HashToken(tp.RefreshToken)` in the
  session (`createSessionAndTokens`, the original's "F1" fix), and `Logout` and `RefreshToken` look sessions up by
  `HashToken(presented)`. But `magic_link_service.go` `CompleteOnboarding` builds its session with the **raw**
  `tp.RefreshToken` (line 218) and caches it too. Running it showed the effect is functional, not an exposed
  credential: the stored raw value can never match a lookup, so a magic-link session cannot be refreshed
  (`ErrSessionNotFound`) and `Logout` for it silently does nothing (no JTI revocation). It is also an F1 hygiene
  violation, a raw token in the DB and the cache. (Reading alone suggested "a live token at rest"; that overstated
  it.) Read only: the magic-link path records no IP or user agent, so session pinning never applies to those
  sessions. The port is byte-identical to the original, so the defect is the original's, not introduced here. It
  is **not fixed in the port**: the fix changes behavior and the oracle must equal the original. It is recorded in
  the design doc; the user fixes it in the original first.
- **Tidy resolved two indirect test dependencies** for `visitoractivity` and `scheduler` (`kr/text` v0.2.0,
  `go-internal` v1.16.0), the same versions `identity/user` already has. Indirect only.
- **gofmt:** the same four `auth` files are not gofmt-clean in the original (`interfaces.go`,
  `reset_request_test.go`, `schema_validation_test.go`, `security_headers.go`).
- **`jobs/scheduler` still imports `news/feed`,** ported as is, the one known shared-imports-feature edge.
- **The 29 skips in the workspace (at 6b), each read on 2026-09-26:** 13 need a DSN
  (`VISITOR_ACTIVITY_TEST_DSN`, `visitoractivity`); 13 are unconditional "RED placeholder" skips for unimplemented
  features (`dictionary` 8, `repository/postgres` 5); 3 are unconditional pointers at app-level tests
  (`dictionary`). A live Postgres can enable only the 13. The repo has a postgres dev bundle
  (`projects/services/postgres-dev/`), so that run is possible; it is not part of any planned scope.

**Result on disk**

```
platform/go/packages/
├── identity/    user · rbac · auth
├── analytics/   visitoractivity                     NEW kit
├── jobs/        scheduler (+schedcli)               NEW kit
└── … (28 earlier modules unchanged)
```

32 of 34 Go modules and all 12 Svelte packages are ported. Next: Scope 6c (`real-estate/property`,
`real-estate/transaction`), its own approval.
