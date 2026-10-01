# bos Scope 3c — regroup the packages into kits by functionality (plan and log)

Last updated: 2026-09-25
tags: bos, scope-3c, taxonomy, kits, regroup, plan

> Design: `docs/proposals/bos-system-design.md`. Follows Scope 3b
> (`docs/plans/2026-09-25-bos-scope-3b-group-renames.md`). Introduces **zero** process-os definitions and
> **no new code**. Approved 2026-09-25 by the user ("so far yes") after four rounds of review: the flat
> `datatypes` catch-all was rejected ("seo is seo, address-book is address"), then `kits/` as a folder,
> then `seo/seo`-style repetition, then the generic kit name `property`.

## Why

The Scope 2/3b taxonomy grouped by technical shape (`datatypes`, `foundation`, `content`, …), so one
functionality was scattered across groups and one group held unrelated things. The user asked for scopes by
functionality, each a **kit**: a family of packages that will grow (a calendar gains integrations), wrapped
later by frameworks — the same idea as Python's `process-kit`.

Two sources of evidence drove the final shape:

- **The full estate, not just the app's closure.** `sbx-core/pkg` has 89 Go packages: 30 top-level packages
  in the app's closure (34 modules counting sub-packages), 4 more product packages outside it (`authz`,
  `land`, `mapoutline`, `utilityreading`), 8 still to classify (`catalogue`, `csvimport`, `document`, `i18n`,
  `identitydocument`, `person`, `persistence`, `storage`), 2 superseded stubs (`lease`, `leaseparticipant`)
  and **45 SDLC/workspace-tooling packages** (`sdlc`, `vault`, `project`, `workspace`, `trace`, `dokploy`, …)
  that are not bos and never enter the product tree. Svelte: 18 packages under `sbx/packages`, 12 in the
  closure.
- **What the names actually mean**, read from source: `dictionary` is flat code/label lookup lists
  (`property-type`, `land-size-unit`, amenities); `units` is date/number/area/land-unit formatting;
  `text-template` renders text.

## The rules (recorded in `platform/CLAUDE.md`)

1. **The top level of `packages/` is a kit, named by functionality.** Three levels exist: role
   (`packages/` vs `frameworks/`), kit, package.
2. **Naming test:** a kit name must tell a newcomer what is inside without its parent folder. Generic nouns
   (`property`, `transaction`, `collection`, `dictionary`, `units`) are package names only, under a kit that
   qualifies them.
3. **A package with the same name as its kit is the kit folder** (`packages/seo`, not `packages/seo/seo`);
   every other package is a subfolder. A kit grows by adding subfolders; nothing moves.
4. Directory placement is independent of import identity: module paths and npm names stay verbatim until M5.

## Moves (Go: 129 files; Svelte: 16 files)

| From | To | Files |
|---|---|---|
| `datatypes/seo` | `seo` | 3 |
| `datatypes/money` | `money` | 9 |
| `content/dictionary` | `reference-data/dictionary` | 10 |
| `datatypes/language` | `localization/language` | 4 |
| `foundation/notify` | `notifications/notify` | 5 |
| `foundation/httputil` | `http/httputil` | 7 |
| `datatypes/geocoordinate`, `datatypes/address` | `location/…` | 3 + 7 |
| `datatypes/personname`, `phonenumber`, `socialnetwork` | `contacts/…` | 3 + 3 + 3 |
| `foundation/database`, `foundation/repository` (+ nested `postgres`) | `persistence/…` | 4 + 21 |
| `content/rss`, `content/feed` | `news/…` | 40 + 7 |
| Svelte `datatypes/units`, `datatypes/text-template` | `formatting/…` | 11 + 5 |

Stay in place (59 Go files): `identity/{user, rbac}`, `media/{image, video}`, `real-estate/collection`.
Svelte `media/canvas-kit` stays.

## Tasks

| # | Task | Result expected |
|---|---|---|
| 0 | Commit this plan | one `docs:` commit |
| 1 | `git mv` the 15 Go package folders and the 2 Svelte package folders | 145 R100 renames |
| 2 | `go.work`: 21 paths. Four `replace` paths (`user`, `collection`, `httputil`, `dictionary`); the other three (`address`, `postgres`, `feed`) point at siblings that move together | module paths unchanged |
| 3 | `pnpm install` to regenerate `pnpm-lock.yaml` (importer paths only); verify `--frozen-lockfile` | no version changes |
| 4 | Design doc (sections 1, 3, 5, 6, 8; D12, D13, D15 open; decision log) and `platform/CLAUDE.md` | Scope 2/3/3b logs left as written (history) |
| 5 | Verify every gate; append results; commit | evidence, not assertion |

## Exit gates

- `git diff -M100%` shows 145 pure renames (129 Go + 16 TS); the only other changed files are the ones in
  tasks 2–4.
- Go: every module's pass/skip/fail equals the baseline captured before the moves across all 21 modules
  (**840 pass, 16 skip, 0 fail**); `go vet` clean; `gofmt -l` empty; `go list -m` finds 21.
- TS: vitest 81 + 10 + 396 unchanged; `pnpm install --frozen-lockfile` passes; the lockfile diff is importer
  paths only.
- No path under `platform/*/packages` repeats its parent folder's name; no kit name is a generic noun.
- `process-cli check` stays clean. Scoped runs only; the full battery is CI's job. Nothing is pushed.

## Not in this scope

Any port, any new code, Scope 4, wiring (where a kit's handlers/migrations/routes live is open decision D15,
settled by the FAQ slice in M2).

## Results (2026-09-25) — done

| Task | Commit | Evidence |
|---|---|---|
| 0 plan | `7dae5fb` | this document |
| 1–3 moves, `go.work`, four `replace` paths, lockfile | `41437bc` | 149 files: **145 renames** (129 Go + 16 TS; 143 are R100, `httputil/go.mod` is R088 and `dictionary/go.mod` R089 because each carries its one `replace` line), plus `go.work`, `user/go.mod`, `collection/go.mod` and `pnpm-lock.yaml`; 23 insertions, 23 deletions in total |
| 4 design doc, `platform/CLAUDE.md` | `e7e7ab7` | sections 1, 3, 5, 6, 8 and the decision log rewritten; D12 revised, D13 replaced, D15 opened |
| 5 results | (this commit) | — |

**Gates**
- The four `replace` edits are exactly: `user` → `../../persistence/repository`, `collection` → `../../seo`,
  `httputil` → `../../money`, `dictionary` → `../../persistence/database`. The other three in-repo `replace`
  lines (`address`, `postgres`, `feed`) point at siblings that moved together and did not change.
- Lockfile diff: three lines — two importer paths and one relative link. No version changed.
- Go, all 21 modules, counted with the same script before and after the moves: **840 pass, 16 skip, 0 fail**;
  the two per-module result files are identical (`diff` empty). `go vet` clean on 21 modules; `gofmt -l`
  reports 0 files; `go list -m` finds 21.
- TS vitest unchanged: units 81 (4 files), text-template 10 (1 file), canvas-kit 396 (21 files) = 487.
  `pnpm install --frozen-lockfile` passes.
- Path rule: no path under `platform/*/packages` repeats its parent folder's name and no kit name is a
  generic noun (13 Go kits, 2 Svelte kits).
- `process-cli check`: ok. `git grep` finds no old-group path (`packages/datatypes|foundation|content|values`)
  outside the plan logs, which keep the names of their time. CI needed no change: it names no group path.
- Nothing pushed. The four emptied group folders were removed with `rmdir` (they held no files).

**Result on disk**

```
platform/go/packages/
├── seo · money
├── real-estate/     collection
├── reference-data/  dictionary
├── identity/        user · rbac
├── contacts/        personname · phonenumber · socialnetwork
├── location/        geocoordinate · address
├── media/           image · video
├── news/            rss · feed
├── persistence/     database · repository (+ postgres)
├── localization/language · notifications/notify · http/httputil
platform/svelte/packages/
├── formatting/      units · text-template
└── media/           canvas-kit
```

Open: **D15**, where a kit's bos wiring lives; the FAQ slice in M2 settles it. Next: Scope 4 (Svelte
foundation), destination `platform/svelte/packages/ui/{animations, core-ui}`, its own approval.
