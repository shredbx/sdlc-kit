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
- **`gosimple/slug` and its indirect `gosimple/unidecode`** are new to the workspace. They enter at the original pins.
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

## Results (2026-09-26) — done

| Task | Commit | Evidence |
|---|---|---|
| 0 plan | `72cdeb9` | this document, with the baseline |
| 1 baseline | (scratchpad `go_baseline_6c.json`, table above) | measured on the original module before anything was ported |
| 2–5 copy, `go.mod`, tidy, `go.work` | `1f1072c` | 56 files: **55 new** (51 verbatim + 2 `go.mod` + 2 `go.sum`) and `go.work` modified |
| gofmt (kept separate) | `3caa817` | 5 files, 10 insertions / 10 deletions, formatting only |
| 7 design doc | `b0c7d7e` | table 6.3 and the tree ✔, section 7 decision, section 8, M0 10 of about 12, decision log |
| results | (this commit) | — |

**Gates**
- **Byte-identity:** `diff -r -x go.mod -x go.sum` against the source is empty for both packages (property 36 files,
  transaction 15 = 51), `testdata/` included, so nothing was missed either. (Superseded for the 5 files gofmt touched,
  by `3caa817`.)
- **Tests, each module on its own, equal to the baseline both before and after gofmt:** property 232 / 0 / 0,
  transaction 60 pass + 4 skip = **292 pass, 4 skip, 0 fail**. The 34-module run gives **1,790 / 33 / 0**; the 32
  earlier modules are unchanged (compared programmatically: 1,498 / 29 / 0, no module differs), and the per-module
  JSON is identical before and after gofmt. The `testdata/*.yml` fixtures load from the new folders.
- **Pins:** every direct third-party requirement equals the original `sbx-core/go.mod` (`uuid` v1.6.0, `gosimple/slug`
  v1.15.0, `pgx/v5` v5.9.1, `yaml.v3` v3.0.1, `testify` v1.11.1); checked by script. The indirect `gosimple/unidecode`
  v1.0.1 equals the original too.
- **Tidy:** a standalone `go mod tidy` (`GOWORK=off`) succeeds and is a no-op on a second run for both modules.
- `go vet` clean across 34 modules; `gofmt -l` empty after `3caa817`; `go list -m` finds 34.
- Staged file count (55 new + `go.work`) equals the files on disk (55). No file is ignored (`git check-ignore` finds
  none of the `.yml` fixtures). `go.work.sum` did not change.
- `process-cli check`: ok. Nothing pushed. `go-ci.yml` is `go list -m`-based and needed no change.

**`go.mod` shape per module** (direct in-repo requires → full `replace` closure)

| Module | Direct in-repo | `replace` lines | Third-party direct |
|---|---|---|---|
| property | 5 | 7 | 4 (`uuid`, `gosimple/slug`, `pgx/v5`, `yaml.v3`) |
| transaction | 3 | 4 | 4 (`uuid`, `pgx/v5`, `testify`, `yaml.v3`) |

**Findings on the way**
- **Tidy resolved two indirect test dependencies** for both modules (`kr/text` v0.2.0, `go-internal` v1.16.0), the same
  versions the earlier ports have. The original's `go.mod` lists `go-internal` at v1.14.1 (indirect) and has no
  `kr/text` line, so these two indirect versions differ from the original's; indirect only, and the tests pass.
- **gofmt:** five files are not gofmt-clean in the original (`property/image_collection_test.go`,
  `property/property.go`, `property/service_test.go`, `property/slug_test.go`, `transaction/service.go`).
- **`mapper.generated.go`** is ported as is in both packages; nothing was regenerated.
- **The 4 skipped `transaction` tests** are unconditional and point at `internal/deal` tests in the app; they first
  run at Scope 7.
- **The copy script needed one change:** it now copies a `testdata/` sub-directory recursively and byte-verifies it,
  and still rejects any other unexpected sub-directory. Scratchpad tooling only; nothing in the repo.
- **Decision on the 6b `identity/auth` defect** (recorded in the design doc): fix it in the original first, then
  re-sync. Not part of this scope's code.

**Result on disk**

```
platform/go/packages/real-estate/
├── collection/      (unchanged)
├── property/        NEW
└── transaction/     NEW
```

All 34 Go modules and all 12 Svelte packages are ported. Next: Scope 7 (baseline import of the apps and oracle
capture), its own approval.
