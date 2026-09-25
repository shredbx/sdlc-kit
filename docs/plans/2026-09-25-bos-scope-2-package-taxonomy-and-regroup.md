# bos Scope 2 — package taxonomy and regroup (plan and log)

Last updated: 2026-09-25
tags: bos, scope-2, taxonomy, regroup, plan

> Design: `docs/proposals/bos-system-design.md` (section 3.1 is the taxonomy, D13/D14 the decisions).
> Follows Scope 1 (`docs/plans/2026-09-25-bos-scope-1-workspaces-and-leaf-ports.md`). Introduces **zero**
> process-os definitions and **no new code**: it moves what Scope 1 ported into the placement taxonomy,
> before Scope 3 adds 13 more Go modules.

**Why:** the Scope 1 layout put every package in one flat `packages/` folder — a gap in the design, since
the repo's own namespace discipline says to nest, not to grow flat names. Directory placement and import
identity are independent, so grouping costs nothing now and much more after 34+ modules and their importers.

**Approved:** 2026-09-25 by the user ("commit plan then lets go, working along this worktree"). All work
stays in this worktree; nothing is pushed.

## Taxonomy (from the design doc, section 3.1)

`platform/<lang>/packages/<group>/<package>`; groups `values`, `foundation`, and domain areas (`identity`,
`content`, `media`, `engagement`, `analytics`, `real-estate`) created lazily. Placement test in order: pure
value → `values`; plumbing/UI foundation → `foundation`; product vocabulary → its domain area; used by two or
more domains → moves down, never sideways. Direction: `values` ← `foundation` ← domains.

## Tasks

| # | Task | Result expected |
|---|---|---|
| 0 | Commit this plan (design-doc taxonomy + this document) | one `docs:` commit |
| 1 | Move Go: `money`, `phonenumber`, `language`, `geocoordinate`, `personname`, `socialnetwork`, `seo` → `packages/values/`; `notify` → `packages/foundation/` | 100%-similarity renames |
| 2 | Move Svelte: `units`, `text-template` → `packages/values/`; `canvas-kit` → `packages/media/` | 100%-similarity renames |
| 3 | `go.work` paths → `packages/<group>/<name>`; `pnpm-workspace.yaml` → `packages/*/*`; regenerate `pnpm-lock.yaml` | same module paths, same npm names |
| 4 | `go-ci.yml` cache path → `platform/go/**/go.sum` (nested modules) | CI unchanged otherwise (`go list -m` finds every module) |
| 5 | New `platform/CLAUDE.md`: placement test, dependency direction, group map | auto-loaded when working under `platform/` |
| 6 | `CLAUDE.md`: add the completion-report-shape paragraph under "Workflow with the user" | — |
| 7 | Verify every gate; append the results below; commit | evidence, not assertion |

## Exit gates

- `git diff -M` shows the 11 moves as 100% renames; the only other changed files are the ones listed above.
- Module paths and npm names are unchanged, so no source file is touched.
- Go: `go vet` clean, `gofmt -l` empty, per-package `go test` pass counts equal Scope 1 (17 / 1 / 120 / 4 / 21 /
  19 / 2 / 27 = 211). TS: vitest 81 + 10 + 396 = 487 tests (4 / 1 / 21 files).
- `pnpm install --frozen-lockfile` passes on the regenerated lockfile.
- The `go.work`-root CI commands (`go list -m`-based vet/test, gofmt check) are run locally first.
- `process-cli check` stays clean. Scoped runs only; the full battery is CI's job.

## Not in this scope

The 13 remaining Go L0/L1 ports (Scope 3), all Svelte ports past Scope 1, any framework or kit code, any
process-os definition, any push.
