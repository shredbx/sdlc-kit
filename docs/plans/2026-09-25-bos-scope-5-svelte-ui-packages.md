# bos Scope 5 — the remaining Svelte packages (plan and log)

Last updated: 2026-09-26
tags: bos, scope-5, svelte, ui, port, plan

> Design: `docs/proposals/bos-system-design.md` (table 6.4, section 8). Follows Scope 4
> (`docs/plans/2026-09-25-bos-scope-4-svelte-foundation.md`). Introduces **zero** process-os definitions and
> **no new code**: seven packages copied verbatim. Approved by the user on 2026-09-26 ("confirm") after the
> tree was shown, including one scope for all seven and acceptance of newer resolved tool versions.

## Why one scope, and the order inside it

Every dependency these packages declare is already ported (`core-ui`, `units`, `canvas-kit`, `text-template`),
except one inside the scope: **`canvas-ui` depends on `ui-source-picker`** (`workspace:*`). `workspace:*` breaks
`pnpm install` when its target is missing, so all seven are copied before the one install; the order matters
only for reading the log (source-picker before canvas-ui). After this scope all 12 Svelte packages are ported.

## What is copied (verbatim); left behind is `node_modules` everywhere

Source: `shredbx/projects/sbx/packages/<name>` → destination `platform/svelte/packages/<kit>/<name>`.

| Package | Destination kit | Copied | Files | Tests |
|---|---|---|---|---|
| `ui-seo` | `seo` | `package.json`, `src/` (6 `.svelte`, 6 `.ts`) | 13 | 1 file |
| `ui-contact` | `contacts` | `package.json`, `vitest.config.ts`, `src/` (1 `.svelte`, 6 `.ts`) | 9 | 1 file |
| `ui-calendar` | `calendar` | `package.json`, `vitest.config.ts`, `src/` (6 `.svelte`, 16 `.ts`) | 24 | 6 files |
| `ui-map` | `location` | `package.json`, `vitest.config.ts`, `src/` (9 `.svelte`, 28 `.ts`) | 39 | 9 files |
| `ui-image` | `media` | `package.json`, `vitest.config.ts`, `src/` (9 `.svelte`, 20 `.ts`) | 31 | 5 files |
| `ui-source-picker` | `media` | `package.json`, `src/` (2 `.svelte`, 1 `.ts`) | 4 | none |
| `canvas-ui` | `media` | `package.json`, `src/` (35 `.svelte`, 8 `.ts`) | 44 | none |

164 files. `pnpm-workspace.yaml` (`packages/*/*`) already matches every destination; it is not edited.

## Tasks

| # | Task | Result expected |
|---|---|---|
| 0 | Commit this plan | one `docs:` commit |
| 1 | Baseline: the five tested packages' vitest pass/skip/fail counts, measured in a throwaway copy of the *original* source (scratchpad, never the shredbx tree) | numbers recorded below before anything is ported |
| 2 | Copy the seven packages (files above only) | byte-identical to the source |
| 3 | `pnpm install` to extend `pnpm-lock.yaml`; verify `--frozen-lockfile` | new packages added; versions of the existing five packages unchanged |
| 4 | `svelte-ci.yml`: five vitest steps, each run locally first | CI mirrors the local gates |
| 5 | Design doc (table 6.4, tree, section 8, decision log) | earlier scope logs left as written |
| 6 | Verify every gate; append results; commit | evidence, not assertion |

## Exit gates

- `diff -r` against the source is empty for every package (excluding `node_modules`).
- Tested packages (`ui-seo`, `ui-contact`, `ui-calendar`, `ui-map`, `ui-image`): pass/skip/fail counts equal the
  task-1 baseline, per package. The baseline is a copy of the original source under the default tool
  resolution, not the original workspace's own pins; that difference is recorded, not hidden.
- Untested packages (`ui-source-picker`, `canvas-ui`; 48 files): the originals have no tests and no type-check
  config, so there is **no runtime gate**. Their gates are byte-identity, workspace links resolving and
  `--frozen-lockfile`. Runtime proof comes at Scope 7, when the app imports them.
- Workspace links: `@sbx/core-ui`, `@sbx/units`, `@sbx/canvas-kit`, `@sbx/text-template` and
  `@sbx/ui-source-picker` resolve from the packages that declare them.
- `pnpm install --frozen-lockfile` passes; the existing suites still give 81 / 10 / 396 and `core-ui` 251.
- Staged file count equals the files on disk (the `.gitignore` `lib/` trap).
- `process-cli check` stays clean. Scoped runs only. Nothing is pushed.

## Not in this scope

Any Go package, decomposing `core-ui` (M1), authoring tests for the untested packages, and any client
information. When client details are needed they live in the client library the main session is building
(`processos-workspace/definitions/clients/` scope + `records/clients/`, no `libraries:` entry); registering
`bestierealestate` there is done after aligning with the main session, never in `platform/`.

## Results (2026-09-26) — done

| Task | Commit | Evidence |
|---|---|---|
| 0 plan | `ffc0e25` | this document |
| 1 baseline | (scratchpad, not committed) | counts below, measured before anything was ported |
| 2–4 copy, lockfile, CI | `593d45f` | 167 files: **164 new** package files, `pnpm-lock.yaml`, `pnpm-workspace.yaml`, `svelte-ci.yml` |
| 5 design doc, `platform/CLAUDE.md` | `77d8702` | table 6.4 and the tree ✔ (all 12 Svelte packages), section 8, decision log, the pnpm build-script rule |
| 6 results | (this commit) | — |

**Baseline and result, per tested package** (a copy of the original source in the scratchpad, default tool
resolution; the port is identical):

| Package | Test files | Tests passed / failed / skipped |
|---|---|---|
| `ui-seo` | 1 | 10 / 0 / 0 |
| `ui-contact` | 1 | 12 / 0 / 0 |
| `ui-calendar` | 6 | 55 / 0 / 0 |
| `ui-map` | 9 | 160 / 0 / 0 |
| `ui-image` | 5 | 45 / 0 / 0 |

22 files, 282 tests; the two count files are equal (compared programmatically).

**Gates**
- `diff -r -x node_modules` of each whole package against its source is empty for all seven, so nothing
  was missed either (164 files: 13 + 9 + 24 + 39 + 31 + 4 + 44).
- Tested packages equal the baseline (table above), directly and through each exact CI command
  (`pnpm --filter @sbx/<name> test`).
- Existing suites unchanged: units 81, text-template 10, canvas-kit 396, core-ui 251.
- Workspace links resolve on disk: `canvas-ui` → `canvas-kit`, `core-ui`, `text-template`, `ui-source-picker`;
  `ui-source-picker` → `canvas-kit`, `core-ui`; `ui-map` → `core-ui`, `units`.
- `pnpm install --frozen-lockfile` passes. Staged file count (164) equals the files on disk (164).
- `process-cli check`: ok. Nothing pushed.
- **No runtime gate** for `ui-source-picker` and `canvas-ui` (48 files, about 15.4k LOC), as planned: byte-identity,
  workspace links and the frozen lockfile only. Scope 7 gives their first runtime proof.

**Findings on the way**
- **A change outside the approved tree:** `pnpm install` exited 1 with `ERR_PNPM_IGNORED_BUILDS` — pnpm 11 blocks
  the build scripts of `core-js@3.50.0` (via `canvas-ui` → `jspdf` → `canvg`) and `esbuild@0.21.5` (via a vite 5
  dev dependency of `canvas-ui` and `ui-source-picker`), and a non-zero install would fail CI. I tested the fix in
  the scratchpad copy first, then added an explicit `allowBuilds` deny for both to `platform/svelte/pnpm-workspace.yaml`.
  It keeps pnpm's default (scripts do not run); it only records the decision. `core-js` only prints a banner and
  esbuild's native binary ships as a platform package. The rule is now in `platform/CLAUDE.md`.
- The lockfile gained 64 packages and lost one entry: `vitefu@1.1.3` in its unsuffixed form became
  peer-suffixed. No version changed and no existing importer entry changed (unlike Scope 4).
- The baseline is a copy under default tool resolution (Svelte 5.57.1, vitest 4.1.11), not the original
  workspace's own pins; a run in the original tree was avoided so nothing there is written.
- Two dependency deprecations are reported by pnpm (`lucide-svelte@0.562.0`, `whatwg-encoding@3.1.1`); nothing
  was changed to silence them (verbatim first).

**Result on disk**

```
platform/svelte/packages/
├── ui/          animations · core-ui
├── formatting/  units · text-template
├── media/       canvas-kit · ui-image · ui-source-picker · canvas-ui
├── seo/ui-seo · contacts/ui-contact · calendar/ui-calendar · location/ui-map
```

All 12 Svelte packages are ported. Next: Scope 6 (Go domain packages), its own approval.
