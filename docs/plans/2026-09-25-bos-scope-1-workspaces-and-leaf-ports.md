# bos Scope 1 — workspaces and Wave-1 leaf ports (implementation log)

Last updated: 2026-09-25
tags: bos, scope-1, port, go, svelte, plan

> Design: `docs/proposals/bos-system-design.md` (section 9 is this scope). Evidence base:
> `docs/research/bestierealestate-decomposition-recon.md`. Introduces **zero** process-os
> definitions — per the "build to prove, then model" rule, this is the first data point on both stacks.

**Goal:** stand up the two platform workspaces and the client repo, and prove the port loop on both
stacks with the safest packages — before anything is built on top.

**Rule applied:** port verbatim first, prove with the source's own tests, refactor later. Ported sources are
byte-identical to the originals; the only authored files are workspace roots, `go.mod`/`go.sum`, CI and docs.
Module paths and npm names are unchanged (`github.com/shredbx/sbx-core/pkg/<name>`, `@sbx/*`), so no import is
rewritten; final names are decided at M5.

## Tasks and evidence

| # | Task | Commit | Evidence |
|---|---|---|---|
| 0 | Save and commit the plan | `25a4419` | `docs/proposals/bos-system-design.md` |
| 1 | Bootstrap the client repo (initial commit, push) | client repo `9715beb` | README, `.gitignore`, `processos.yaml` (`libraries:` → `sbx-sdlc-kit`, readonly), `processos-workspace/{definitions,records}/.gitkeep` |
| 2 | Mount it as a submodule | `ef5e50f` | `consumers/clients/bestie-bestierealestate` @ `9715beb` (since regrouped to `consumers/clients/bestie/bestierealestate`); `process-cli check` is clean from inside it and the `sbx-sdlc-kit` scope resolves via `../../../` |
| 3 | `platform/go`: `go.work` + 8 leaf packages | `ea11a9e` | see gates |
| 4 | `platform/svelte`: pnpm workspace + 3 pure-TS packages, `.gitignore` | `13454d8` | see gates |
| 5 | gofmt fix (kept separate) | `6c866ed` | `money/rate.go`, comment-only |
| 6 | `go-ci.yml`, `svelte-ci.yml`, pnpm pin | `f46bd3a` | every CI command run locally first |
| 7 | Recon + this log + index | (this commit) | — |

## Exit gates — results

**Byte-identity.** `diff -r` of each ported package against its original is empty (Go: excluding the authored
`go.mod`/`go.sum`; TS: `src/`, `package.json`, `vitest.config.ts`). One deliberate later exception:
`money/rate.go` (task 5, whitespace inside a doc comment).

**Go — per-package `go test` pass counts, ported vs original module** (identical; 0 failures; 211 total):

| Package | Ported | Original |
|---|---|---|
| geocoordinate | 17 | 17 |
| language | 1 | 1 |
| money | 120 | 120 |
| notify | 4 | 4 |
| personname | 21 | 21 |
| phonenumber | 19 | 19 |
| seo | 2 | 2 |
| socialnetwork | 27 | 27 |

`go vet` clean; `gofmt -l` empty after task 5.

**TS — vitest, ported vs original** (identical): `@sbx/units` 4 files / 81 tests, `@sbx/text-template`
1 / 10, `@sbx/canvas-kit` 21 / 396. `pnpm install --frozen-lockfile` reproduces from the committed lockfile.

**Scoped runs.** Every run above targeted one package (or the explicit list); nothing ran a blind
repo-wide suite. The full battery is CI's job.

## Found on the way

1. **A `.gitignore` rule was silently swallowing the sources.** The Python template's `lib/` rule
   (line 17) also matches `src/lib/`, where these packages keep all their code. The first staged commit held
   10 files instead of 72 — it would have shipped three empty packages. Caught by comparing the staged count
   with the files on disk; fixed with a narrow negation (`!platform/svelte/**/src/lib/`), verified with
   `git check-ignore`. Lesson: after adding a language tree to a repo with a broad ignore template, compare
   staged-vs-on-disk counts before committing.
2. **One gofmt finding, comment-only.** `money/rate.go` — a Go 1.19+ doc-comment list line was indented
   two extra spaces. Fixed in its own commit so the verbatim commit stays byte-identical.
3. **`./...` at a `go.work` root matches nothing** when every package is its own module ("directory prefix .
   does not contain modules listed in go.work"). CI therefore uses `go list -m | sed 's|$|/...|'` to turn each
   workspace module into a package pattern; verified locally with the expanded patterns. Nested modules (a
   later `repository/postgres` inside `repository/`) will be listed by `go list -m` too.
4. **The toolchain auto-fetched.** The source declares `go 1.26`; the local Go is 1.25.6 and switched
   automatically on first use (needs network). CI's `setup-go` reads the version from `go.work`.

## Not done (by design)

No framework code, no kits, no process-os definitions, no CI type-check for TS (the packages have no
`tsconfig`; adding one is authored code, so it belongs to the M1 package-conventions decision), no
`README` per package. Waves 2–5 of M0 (the remaining 26 Go packages, the remaining 9 Svelte/TS packages,
the baseline import of the app, the equivalence oracle) each get their own before/after tree and approval.

## Next scope (proposal, needs its own approval)

Wave 2: Go L0/L1 infrastructure and value packages (`database`, `repository` + `repository/postgres`,
`httputil`, `rss`, `feed`, `video`, `image`, `rbac`, `address`, `collection`, `dictionary`, `user`) and the
adapter-bearing Svelte packages (`ui-map`, `ui-calendar`, `ui-image`, `ui-contact`). This is where the
first nested modules, the first heavy third-party dependencies (`image`: aws-sdk, webp; pgx) and — by the
second ported package on each stack — the first honest case for a `port-package` action and package
templates appear.
