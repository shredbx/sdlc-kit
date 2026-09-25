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
