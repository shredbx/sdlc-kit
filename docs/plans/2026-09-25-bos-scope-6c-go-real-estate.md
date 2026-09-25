# bos Scope 6c — property and transaction (plan and log)

Last updated: 2026-09-26
tags: bos, scope-6c, go, port, plan

> Design: `docs/proposals/bos-system-design.md` (table 6.3, section 8). Follows Scope 6b
> (`docs/plans/2026-09-25-bos-scope-6b-go-identity-analytics-jobs.md`). Introduces **zero** process-os definitions and
> **no new code**: two Go modules copied verbatim; no interface is changed or refined (that happens when a kit is
> sliced, section 8.1 of the design). Approved by the user on 2026-09-26 ("proceed as recommended") after the tree was
> shown. Nothing is pushed. This is the last Go scope: it completes all 34 Go modules.

## What is copied (verbatim)

Source: `shredbx/projects/sbx/packages/core/go/pkg/<name>` → `platform/go/packages/real-estate/<name>`. Module paths
stay `github.com/shredbx/sbx-core/pkg/<name>`, so no import is rewritten. The `real-estate` kit already exists
(`collection`); both packages join it.

| Module | Destination | Files | Direct in-repo | Third-party direct |
|---|---|---|---|---|
| `pkg/property` | `real-estate/property` | 36 (13 src + 19 test + 4 `testdata/*.yml`) | `address`, `money`, `repository`, `repository/postgres`, `seo` | `uuid`, `gosimple/slug` v1.15.0, `pgx/v5`, `yaml.v3` v3.0.1 |
| `pkg/transaction` | `real-estate/transaction` | 15 (8 src + 6 test + 1 `testdata/seed.yml`) | `money`, `repository`, `repository/postgres` | `uuid`, `pgx/v5`, `yaml.v3`, `testify` |

**51 verbatim files**, plus 2 authored `go.mod` and up to 2 `go.sum`. `go.work` grows from 32 to 34 `use` paths.

Planned `replace` closure (`replace` does not propagate, see `platform/CLAUDE.md`): `property` = `address`,
`geocoordinate`, `money`, `repository`, `repository/postgres`, `database`, `seo`; `transaction` = `money`,
`repository`, `repository/postgres`, `database`. The script computes the closure from the ported `go.mod` files and
asserts every dependency is a ported module.

## Things to watch

- **`testdata/` is new to the copy step.** Every earlier scope copied flat directories. The copy script now accepts
  a `testdata` sub-directory (byte-verified, recursive) and still rejects any other unexpected sub-directory. The tests
  read `testdata/seed.yml` by relative path, so the gate is the tests themselves passing inside the new folders.
- **The `.gitignore` trap.** The staged-versus-on-disk file count is the guard; the five `.yml` files are the ones a
  broad ignore rule would swallow.
- **`mapper.generated.go`** exists in both packages. It is ported as is, not regenerated: regeneration belongs to the
  kit slice, where the generator's input is known.
- **The 4 skipped `transaction` tests** are unconditional `t.Skip` calls that point at app-level tests in
  `bestierealestate`. They cannot run until the app is imported (Scope 7). (My earlier notes said "testcontainers";
  that word is only a comment in the source. Corrected.)
- **`gosimple/slug` and `rainycape/unidecode`** are new to the workspace. They enter at the original pins.
- **`property` is the largest package** (4,939 source lines) and `repository/postgres` is in its closure, so `pgx` and
  `go.sum` are expected for both modules.

## Baseline (measured on the original module, 2026-09-26) — pass / skip / fail

`go test -count=1 -json` in the original `sbx-core` module, aggregated per package. No environment variable gates any
test in these two packages.

| Package | Pass | Skip | Fail |
|---|---|---|---|
| property | 232 | 0 | 0 |
| transaction | 60 | 4 | 0 |
| **Total** | **292** | **4** | **0** |

## Before → after (new and modified paths only)

```
BEFORE                                          AFTER
platform/go/go.work        (32 use paths)       platform/go/go.work        (34 use paths)
platform/go/packages/real-estate/               platform/go/packages/real-estate/
└── collection/                                 ├── collection/           (unchanged)
                                                ├── property/             NEW  go.mod go.sum · 13 src · 19 test · testdata/ (4 yml)
                                                └── transaction/          NEW  go.mod go.sum · 8 src · 6 test · testdata/ (1 yml)
docs/plans/  (no 6c)                            docs/plans/2026-09-25-bos-scope-6c-go-real-estate.md   NEW
docs/proposals/bos-system-design.md             modified: table 6.3, tree, section 7 (defect decision), section 8, decision log
```

## Tasks

| # | Task | Result expected |
|---|---|---|
| 0 | Commit this plan | one `docs:` commit |
| 1 | Baseline on the original module (table above) | numbers recorded before anything is ported |
| 2 | Copy the two packages, `testdata/` included | byte-identical to the source |
| 3 | Author two `go.mod` (in-repo `require` + `replace` closure, pinned third-party) | two `go.mod` |
| 4 | `go mod tidy`, standalone | `go.sum`, indirect requirements |
| 5 | `go.work`: 32 → 34 `use` entries | — |
| 6 | Verify every gate | evidence below |
| 7 | Design doc, results, commits | — |

## Exit gates

- `diff -r` against the source is empty for both packages, excluding the authored `go.mod`/`go.sum`.
- Per-module pass/skip/fail equals the baseline above (**292 / 4 / 0**), each module run on its own.
- Every direct third-party requirement equals the `sbx-core/go.mod` pin; a standalone tidy (`GOWORK=off`) is a no-op
  on a second run.
- `go vet` clean across 34 modules; `gofmt -l` empty (findings in their own commit); `go list -m` finds 34.
- The 32 existing modules still give **1,498 / 29 / 0**.
- Staged file count equals the files on disk; `process-cli check` stays clean. Scoped runs only. Nothing pushed.

## Not in this scope

Fixing anything in the originals; the `identity/auth` defect (its decision is recorded in the design doc, section 7);
importing the apps or capturing the oracle (Scope 7); any interface change or refinement; wiring to a framework or an
app; declaring kit edges (D15); process-os definitions. Client information is not touched: it lives in the client
library the main session is building, registered after alignment with that session, never in `platform/`.
