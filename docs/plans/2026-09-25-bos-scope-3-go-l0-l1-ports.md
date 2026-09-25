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

## Results (2026-09-25) — done

| Task | Commit | Evidence |
|---|---|---|
| 0 plan | `9777056` | this document |
| 1–4 verbatim ports, `go.mod`, tidy, `go.work` | `17e53e5` | 157 files: 119 `.go` (72 src + 47 test), 14 testdata, 13 `go.mod`, 9 `go.sum`, `go.work`, `go.work.sum` |
| gofmt (kept separate) | `1ceec13` | 14 files, 83 insertions / 82 deletions, formatting only |
| 6–7 conventions, design doc, results | (this commit) | `platform/CLAUDE.md`, design doc |

**Gates**
- **Byte-identity:** `diff -r` against the original is empty for all 12 top-level packages (13 modules;
  `repository` includes its nested `postgres/`), excluding `go.mod`/`go.sum` and the deliberately absent
  `image/watermark/`. (Superseded for the 14 files gofmt touched, by commit `1ceec13`.)
- **Tests, per module, equal to the baseline both before and after gofmt:** address 82, database 43, image 89,
  rbac 44, repository 56, repository/postgres 53 pass + 5 skip, rss 93, video 40, collection 29,
  dictionary 21 pass + 11 skip, feed 13, httputil 56, user 10 = **629 pass, 16 skip, 0 fail**.
- `go vet` clean across all 21 modules; `gofmt -l` empty after `1ceec13`; `go list -m` finds 21 modules;
  `process-cli check` ok.

**Result on disk**

```
platform/go/packages/
├── values/        address + the 7 from Scope 1
├── foundation/    database · repository (+ postgres, nested) · httputil · notify
├── identity/      user · rbac
├── content/       dictionary · rss · feed
├── media/         image (no watermark/) · video
└── real-estate/   collection                                       → 21 of 34 modules; L0 and L1 complete
```

## Found on the way

1. **The `require` + `replace` convention coexists with `go.work`.** Tested on the first dependent pair
   (`address` → `geocoordinate`) before authoring the rest: no "conflicting replacements", tests pass, and
   a standalone `go mod tidy` succeeds.
2. **My import scan was wrong about `feed`.** The facts script listed no third-party import for `feed`;
   `go mod tidy` correctly added `github.com/google/uuid` as a direct requirement. Tidy is authoritative —
   recorded as a rule in `platform/CLAUDE.md`.
3. **`go.work.sum` appeared** once third-party modules entered the workspace, and is committed. The CI
   cache key (`platform/go/**/go.sum`) does not cover it — harmless (a cache-key precision only), left for a
   later CI touch-up.
4. **Indirect versions.** The pinned direct versions reproduced the original's aws-sdk indirect versions
   exactly; `golang.org/x/text` resolves lower (v0.29.0) in the pgx-based modules than the original's
   v0.35.0, which came from other dependencies of the whole `sbx-core` module. Indirect only; results are
   identical.
5. **gofmt:** 14 of the ported files were not gofmt-clean in the original; fixed in their own commit, as in
   Scope 1.

Next: the Svelte foundation (`animations`, then `core-ui`) — its own before/after tree and approval.
