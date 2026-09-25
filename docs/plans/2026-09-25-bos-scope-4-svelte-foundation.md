# bos Scope 4 — Svelte foundation: `ui/animations` and `ui/core-ui` (plan and log)

Last updated: 2026-09-26
tags: bos, scope-4, svelte, ui, port, plan

> Design: `docs/proposals/bos-system-design.md` (tables 6.4 and section 8). Follows Scope 3c
> (`docs/plans/2026-09-25-bos-scope-3c-kits-by-functionality.md`). Introduces **zero** process-os definitions
> and **no new code**: two packages copied verbatim. Approved by the user on 2026-09-26 ("ok") after the
> tree was shown twice, including leaving `animations/package-lock.json` behind and accepting newer
> resolved tool versions.

## Why these two, and in this order

`workspace:*` breaks `pnpm install` for a package whose declared workspace dependency is missing, so the
Svelte port order is dependency-driven. `@sbx/core-ui` declares `@sbx/animations` and `@sbx/units`
(`workspace:*`); `units` is already ported (Scope 1). `animations` has no in-repo dependency, so it comes
first. Every remaining `ui-*` package (Scope 5) imports or declares `core-ui`, so nothing else can port before
these two.

Both go into a new **`ui`** kit: the UI base every other Svelte package stands on. `core-ui` is ported as ONE
package, verbatim; its decomposition into `ui-*` packages is M1.

## What is copied (verbatim) and what is left behind

| Source (`shredbx/projects/sbx/packages/core/…`) | Destination (`platform/svelte/packages/ui/…`) | Copied | Left behind |
|---|---|---|---|
| `ts/animations` | `animations` | `package.json`, `tsconfig.json`, `src/` (23 files, 1,821 LOC, no tests) | `package-lock.json` (a stray npm lockfile; this workspace is pnpm), `node_modules` |
| `svelte` | `core-ui` | `package.json`, `svelte.config.js`, `vitest.config.ts`, `src/` (228 files under `src/lib`: 75 `.svelte`, 125 non-test `.ts`, 21 test `.ts`, 5 `.css`, 2 `.yml`; about 44.7k LOC) | `node_modules` |

`pnpm-workspace.yaml` (`packages/*/*`) already matches `ui/animations` and `ui/core-ui`; it is not edited.

## Tasks

| # | Task | Result expected |
|---|---|---|
| 0 | Commit this plan | one `docs:` commit |
| 1 | Copy the two packages (files above only) | byte-identical to the source |
| 2 | `pnpm install` to extend `pnpm-lock.yaml`; verify `--frozen-lockfile` | about 145 packages added; the existing three packages' entries unchanged |
| 3 | `svelte-ci.yml`: an `animations` type-check step and a `core-ui` vitest step, both run locally first | CI mirrors the local gates |
| 4 | Design doc (table 6.4 status, section 8, decision log) | Scope 1–3c logs left as written |
| 5 | Verify every gate; append results; commit | evidence, not assertion |

## Exit gates

- `diff -r` against the source is empty for both packages (excluding the left-behind files).
- **core-ui:** vitest gives **21 test files, 251 tests**, equal to the original.
- **animations:** it has no tests, so the gate is `tsc --noEmit -p tsconfig.json` exiting 0, as in the original.
- **Workspace links:** `@sbx/core-ui` resolves `@sbx/animations` and `@sbx/units` (`workspace:*`).
- `pnpm install --frozen-lockfile` passes; the existing three packages still give **81 / 10 / 396**.
- Staged file count equals the files on disk (`.gitignore` carries a `lib/` rule that would swallow
  `src/lib/`; the negation for `platform/svelte` is verified, not assumed).
- Tool versions: a throwaway spike showed the gates reproduce under default tool resolution (Svelte 5.57.1,
  vitest 4.1.11, against the original's 5.50.0 and 4.1.0), so no version overrides; the committed lockfile
  pins whatever resolves. Recorded here as an accepted difference.
- `process-cli check` stays clean. Scoped runs only. Nothing is pushed.

## Not in this scope

Any `ui-*` package (Scope 5), decomposing `core-ui` (M1), authoring tests for `animations`, replacing its
stray `package-lock.json`, Go.
