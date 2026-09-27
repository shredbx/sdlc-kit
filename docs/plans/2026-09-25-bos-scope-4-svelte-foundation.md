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

## Results (2026-09-26) — done

| Task | Commit | Evidence |
|---|---|---|
| 0 plan | `0e2df9d` | this document |
| 1–3 copy, lockfile, CI | `4f23f16` | 258 files: **256 new** (animations 25, core-ui 231), `pnpm-lock.yaml`, `svelte-ci.yml` |
| 4 design doc | `58e73e3` | table 6.4 and the end-state tree marked ✔, section 8, decision log |
| 5 results | (this commit) | — |

**Gates**
- `diff -r` is empty for `animations/src` (23 files) and `core-ui/src` (228 files); `diff` is empty for the five
  root files (`package.json`, `tsconfig.json`; `package.json`, `svelte.config.js`, `vitest.config.ts`). No
  stray `.DS_Store`.
- **core-ui:** 21 test files, **251 tests** pass, equal to the original (run twice: directly and through the
  exact CI command `pnpm --filter @sbx/core-ui test`, 40–44 s).
- **animations:** `tsc --noEmit -p tsconfig.json` exits 0 (directly and through the CI command).
- **Workspace links:** `core-ui/node_modules/@sbx/{animations, units}` are symlinks to the workspace packages;
  the lockfile records `link:../animations` and `link:../../formatting/units`.
- `pnpm install --frozen-lockfile` passes. The existing three packages still give **81 / 10 / 396**.
- Staged file count (256) equals the files on disk (256): the `lib/` ignore trap did not bite.
- `process-cli check`: ok. Nothing pushed.

**Findings on the way**
- The lockfile gained **100** packages (pnpm's "145" counts reused store entries; my estimate of about 145 was
  wrong). It also changed three existing lines and removed one: `core-ui`'s `jsdom` dev dependency changed the
  resolved *peer suffix* of the existing packages' vitest entry (`vitest@4.1.11(vite@8.3.1)` →
  `vitest@4.1.11(jsdom@25.0.1)(vite@8.3.1)`). No version changed, and their suites are unchanged — but the plan
  said their entries would not change, and strictly they did.
- Under the newer Svelte compiler (5.57.1; the original pins 5.50.0) `ModelControls.svelte` prints two
  `state_referenced_locally` warnings (lines 61–62). The file is byte-identical to the source; whether the
  original emits them under its own pin was not compared. Tests are unaffected.
- pnpm reports two deprecations (`lucide-svelte@0.562.0`, sub-dependency `whatwg-encoding@3.1.1`) and
  peer-dependency warnings; none affect the gates. No dependency was changed to silence them (verbatim first).

**Result on disk**

```
platform/svelte/packages/
├── ui/          animations · core-ui
├── formatting/  units · text-template
└── media/       canvas-kit
```

Next: Scope 5 (Svelte `ui-*` packages), its own approval.
