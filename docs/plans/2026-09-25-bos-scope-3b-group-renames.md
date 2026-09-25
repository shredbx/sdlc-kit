# bos Scope 3b — group renames for clarity (plan and log)

Last updated: 2026-09-25
tags: bos, scope-3b, taxonomy, rename, plan

> Design: `docs/proposals/bos-system-design.md` (section 3.1, D13). Follows Scope 3
> (`docs/plans/2026-09-25-bos-scope-3-go-l0-l1-ports.md`). Introduces **zero** process-os definitions and
> **no new code**. Approved 2026-09-25 by the user ("ok, lets use it so far this way") after the review of
> all eight group names; "so far" means the names stay revisable.

**Why:** the user, reading `platform/go/packages/`, found `values` unclear and asked for the other names to
be checked. Review verdicts: `values` is ambiguous (the word is used in about 50 other tracked files in
unrelated senses — YAML values, config values — and its DDD meaning is jargon); `engagement` is marketing
jargon with no folder yet; `foundation`, `identity`, `content`, `media`, `analytics`, `real-estate` are
clear enough and stay. `foundation` was weighed against `core` (collides with `core-ui` and the old
`sbx-core`) and `infra` (collides with the `infrastructure` SDLC capability and does not fit UI) and kept.

## Renames

| Was | Becomes | Meaning (goes into `platform/CLAUDE.md`) |
|---|---|---|
| `values` | **`datatypes`** | pure data types and small logic: no I/O, no product vocabulary (money, phone number, address, units, text template) |
| `engagement` | **`crm`** | contacts, inquiries (leads) and appointments — no folder exists yet, docs only |

Unchanged, with the one-line meaning now written down: `foundation` (generic building blocks everything
else stands on — backend plumbing and the UI base), `identity` (users, roles, login), `content` (managed
content and reference data), `media`, `analytics`, `real-estate`.

## Tasks

| # | Task | Result expected |
|---|---|---|
| 0 | Commit this plan | one `docs:` commit |
| 1 | `git mv` `platform/go/packages/values` → `datatypes` (8 modules, 35 files) and `platform/svelte/packages/values` → `datatypes` (2 packages, 16 files) | 51 100%-similarity renames |
| 2 | `go.work`: 8 paths; `foundation/httputil/go.mod` and `real-estate/collection/go.mod`: the two `replace` paths | module paths unchanged |
| 3 | Regenerate `pnpm-lock.yaml` (importer paths only); verify `--frozen-lockfile` | no version changes |
| 4 | Design doc and `platform/CLAUDE.md`: the two renames, the one-line meanings, the decision log | Scope 2 and 3 logs left as written (history) |
| 5 | Verify every gate; append results; commit | evidence, not assertion |

## Exit gates

- `git diff -M100%` shows 51 pure renames; the only other changed files are the ones in tasks 2–4.
- Go: every module's pass/skip/fail equals its baseline across all 21 modules (**840 pass, 16 skip, 0 fail**:
  211 from Scope 1 + 629 from Scope 3); `go vet` clean; `gofmt -l` empty; `go list -m` finds 21.
- TS: vitest 81 + 10 + 396 unchanged; `pnpm install --frozen-lockfile` passes; the lockfile diff is
  importer paths and one link only.
- `process-cli check` stays clean. Scoped runs only; the full battery is CI's job. Nothing is pushed.

## Not in this scope

Any port, any new group folder, any code change, Scope 4.
