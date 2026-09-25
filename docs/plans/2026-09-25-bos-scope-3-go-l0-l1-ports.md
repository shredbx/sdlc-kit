# bos Scope 3 — the remaining Go L0/L1 packages (plan and log)

Last updated: 2026-09-25
tags: bos, scope-3, port, go, plan

> Design: `docs/proposals/bos-system-design.md` (sections 3.1, 6.3, 8). Follows Scope 2
> (`docs/plans/2026-09-25-bos-scope-2-package-taxonomy-and-regroup.md`), which created the placement
> taxonomy. Introduces **zero** process-os definitions. Approved 2026-09-25 by the user ("so far confirm")
> after the before/after tree; all work stays in this worktree, nothing is pushed.

**Goal:** port every remaining L0/L1 Go package of the `sbx-core` closure — 13 modules — verbatim into
their taxonomy groups, and prove each against the original. After this scope Go layers L0 and L1 are
complete (22 of 34 modules); L2/L3 follow in a later scope.

**Rule applied:** verbatim first, prove with the source's own tests, refactor later. Module paths are
unchanged (`github.com/shredbx/sbx-core/pkg/<name>`), so no import is rewritten.

## Placement (from the taxonomy)

| Group | Modules |
|---|---|
| `values` | `address` |
| `foundation` | `database`, `repository` (+ nested `postgres`), `httputil` |
| `identity` (new) | `user`, `rbac` |
| `content` (new) | `dictionary`, `rss`, `feed` |
| `media` (new) | `image`, `video` |
| `real-estate` (new) | `collection` |

13 modules = 12 top-level + 1 nested (`foundation/repository/postgres`). All in-repo dependencies are
either already ported (`geocoordinate`, `seo`, `money`) or in this set.

## Conventions introduced (authored, confirmed with the tree)

- **In-repo Go dependencies** are declared in `go.mod` as `require github.com/shredbx/sbx-core/pkg/<x> v0.0.0`
  plus a relative `replace … => ../…`, so each module also builds and tidies outside the workspace (the
  consumer's mirror/vendor step will need that). The workspace (`go.work`) resolves them in development.
- **Third-party versions are pinned** to the original `sbx-core/go.mod`; `go.sum` and indirect requirements
  come from `go mod tidy` (needs network).
- **Left behind on purpose:** `image/watermark/` (4 files, 6 tests) — zero importers in BR and in all of
  `sbx-core`.

## Baseline (measured on the original module, 2026-09-25) — pass / skip / fail

| Module | Pass | Skip | Module | Pass | Skip |
|---|---|---|---|---|---|
| address | 82 | 0 | rbac | 44 | 0 |
| collection | 29 | 0 | repository | 56 | 0 |
| database | 43 | 0 | repository/postgres | 53 | 5 |
| dictionary | 21 | 11 | rss | 93 | 0 |
| feed | 13 | 0 | user | 10 | 0 |
| httputil | 56 | 0 | video | 40 | 0 |
| image | 89 | 0 | | | |

Total **629 pass, 16 skip, 0 fail**. The 16 skips are the `dictionary` and `repository/postgres` tests that
need a live Postgres; they stay skipped, exactly as in the original.

## Tasks

| # | Task | Result expected |
|---|---|---|
| 0 | Commit this plan | one `docs:` commit |
| 1 | Copy the 13 packages verbatim into their groups (`image` without `watermark/`; `repository` with its `postgres/` sub-package) | 72 source + 47 test `.go` files, 14 testdata files |
| 2 | Author each `go.mod` (module path, `go 1.26`, pinned third-party, in-repo `require` + `replace`); nested `repository/postgres` gets its own | 13 `go.mod` |
| 3 | `go mod tidy` per module — leaves first, then the modules with in-repo dependencies — for `go.sum` | 13 `go.sum` (or none where dependency-free) |
| 4 | `go.work`: 8 → 21 `use` entries | — |
| 5 | Verify every gate | evidence below |
| 6 | `platform/CLAUDE.md`: record the `require`/`replace` convention; design doc: mark the 13 ported | — |
| 7 | Append results; commit | — |

## Exit gates

- `diff -r` against the original is empty per package, excluding `go.mod`/`go.sum` and the deliberately
  absent `image/watermark/`.
- Per-module pass/skip/fail counts equal the baseline above (**629 / 16 / 0**).
- `go vet` clean; `gofmt -l` empty — any finding is fixed in its own commit, after the verbatim commit.
- The CI commands (`gofmt`, `go list -m`-based `vet` and `test`) run locally first.
- `process-cli check` stays clean. Scoped runs only; the full battery is CI's job.

## Risks and how they are checked

- A `go.mod` `replace` pointing at a module that is also a workspace member could conflict in workspace
  mode. Checked empirically on the first dependent pair before the rest are authored.
- `go mod tidy` needs network; `image` pulls aws-sdk-go-v2, webp and `x/image` (heavy).
- `repository/postgres` is a nested module inside `repository/`; the parent module excludes it
  automatically because it has its own `go.mod`.

## Not in this scope

Go L2/L3 packages, all Svelte ports past Scope 1, kits, frameworks, process-os definitions, any push.
