# bos Scope 6a — the first seven Go feature packages (plan and log)

Last updated: 2026-09-26
tags: bos, scope-6a, go, port, plan

> Design: `docs/proposals/bos-system-design.md` (table 6.3, section 8). Follows Scope 5
> (`docs/plans/2026-09-25-bos-scope-5-svelte-ui-packages.md`). Introduces **zero** process-os definitions and
> **no new code**: seven Go modules copied verbatim. Approved by the user on 2026-09-26 ("confirmed") after the
> tree was shown, including the split of the remaining 13 Go modules into 6a / 6b / 6c and the plan file keeping
> the `2026-09-25-` series prefix. Nothing is pushed.

## Why this split, and why these seven

All in-repo dependencies of the 13 remaining Go modules are already ported. The only ones that are not are
the nested ones: `vcard` → `contact`, `ical` → `calendar`, `schedcli` → `scheduler` + `database`. So the split
is by **risk**, not by dependency layer:

| Scope | Modules | Why together |
|---|---|---|
| **6a (this)** | `faq`, `cms`, `contacts/inquiry`, `contacts/contact` (+ `vcard`), `calendar` (+ `ical`) | hermetic tests, small (41 source files), no jwt/redis/env |
| 6b | `identity/auth`, `analytics/visitoractivity`, `jobs/scheduler` (+ `schedcli`) | jwt, redis, bcrypt; env-dependent tests; `scheduler` imports `news/feed` (known violation) |
| 6c | `real-estate/property`, `real-estate/transaction` | largest package (36 files), `testdata`, testcontainers |

## What is copied (verbatim)

Source: `shredbx/projects/sbx/packages/core/go/pkg/<name>` → `platform/go/packages/<kit>/<package>`. The
original is **one** module (`github.com/shredbx/sbx-core`); each package here becomes its own module with the
module path unchanged (`github.com/shredbx/sbx-core/pkg/<name>`), so no import is rewritten.

| Module | Destination | Files (src + test) | Third-party imports |
|---|---|---|---|
| `pkg/faq` | `faq` | 5 (3 + 2) | none |
| `pkg/cms` | `cms` | 14 (9 + 5) | none |
| `pkg/inquiry` | `contacts/inquiry` | 2 (1 + 1) | none |
| `pkg/contact` | `contacts/contact` | 7 (5 + 2; includes `mapper.generated.go`) | `uuid`, `pgx/v5` |
| `pkg/contact/vcard` | `contacts/contact/vcard` (nested module) | 3 (1 + 2) | none |
| `pkg/calendar` | `calendar` | 6 (3 + 3) | `pgx/v5` |
| `pkg/calendar/ical` | `calendar/ical` (nested module) | 4 (1 + 3) | none |

**41 verbatim files**, plus 7 authored `go.mod` and up to 7 `go.sum`. `go.work` grows from 21 to 28 `use` paths.
The original `cms` describes itself as local to the first client; it is ported verbatim anyway and genericized
later, in a scope of its own.

## Conventions (unchanged from Scope 3)

- In-repo dependencies: `require github.com/shredbx/sbx-core/pkg/<x> v0.0.0` plus a relative `replace`, so each
  module tidies and builds outside the workspace too. `replace` directives of a dependency do not propagate to
  the module that uses it, so **transitive in-repo modules also need their own `replace`** (e.g. anything that
  requires `repository/postgres` also replaces `database`). The replace closure is computed from the ported
  `go.mod` files, not guessed.
- Third-party direct versions are pinned to the original `sbx-core/go.mod` (`uuid v1.6.0`, `pgx/v5 v5.9.1`);
  `go.sum` and indirect requirements come from `go mod tidy`, leaves first, nested modules after their parent.
- gofmt findings in verbatim originals get a **separate** style commit after the verbatim one.

## Baseline (measured on the original module, 2026-09-26) — pass / skip / fail

Run with `go test -count=1 -json` in the original `sbx-core` module (read-only; writes only to the Go build
cache), aggregated per package.

| Package | Pass | Skip | Fail |
|---|---|---|---|
| faq | 51 | 0 | 0 |
| cms | 109 | 0 | 0 |
| inquiry | 31 | 0 | 0 |
| contact | 36 | 0 | 0 |
| contact/vcard | 20 | 0 | 0 |
| calendar | 35 | 0 | 0 |
| calendar/ical | 44 | 0 | 0 |
| **Total** | **326** | **0** | **0** |

## Before → after (new and modified paths only)

```
BEFORE                                          AFTER
platform/go/                                    platform/go/
├── go.work            (21 use paths)           ├── go.work            (28 use paths)
└── packages/                                   └── packages/
    ├── seo · money · …    (21 modules)             ├── calendar/            NEW  go.mod go.sum · 3 src · 3 test
    │                                               │   └── ical/            NEW  nested module · 1 src · 3 test
    │                                               ├── cms/                 NEW  go.mod · 9 src · 5 test
    │                                               ├── faq/                 NEW  go.mod go.sum · 3 src · 2 test
    │                                               └── contacts/
    │                                                   ├── contact/         NEW  5 src (1 generated) · 2 test
    │                                                   │   └── vcard/       NEW  nested module · 1 src · 2 test
    │                                                   └── inquiry/         NEW  1 src · 1 test
docs/plans/  (no 6a)                            docs/plans/2026-09-25-bos-scope-6a-go-feature-packages.md   NEW
docs/proposals/bos-system-design.md             modified: table 6.3 ✔, tree, section 8, decision log
```

## Tasks

| # | Task | Result expected |
|---|---|---|
| 0 | Commit this plan | one `docs:` commit |
| 1 | Baseline on the original module (table above) | numbers recorded before anything is ported |
| 2 | Copy the seven packages (files above only) | byte-identical to the source |
| 3 | Author seven `go.mod` (in-repo `require` + `replace` closure, pinned third-party) | seven `go.mod` |
| 4 | `go mod tidy`, leaves first | `go.sum`, indirect requirements |
| 5 | `go.work`: 21 → 28 `use` entries | — |
| 6 | Verify every gate | evidence below |
| 7 | Design doc, results, commits | — |

## Exit gates

- `diff -r` against the source is empty for every package, excluding the authored `go.mod`/`go.sum`.
- Per-module pass/skip/fail equals the baseline above (**326 / 0 / 0**), each module run on its own.
- Every direct third-party requirement in a produced `go.mod` equals the `sbx-core/go.mod` pin.
- `go mod tidy` is a no-op on a second run for every module (idempotent), and a standalone tidy works with
  `GOWORK=off`.
- `go vet` clean across 28 modules; `gofmt -l` empty (findings fixed in their own commit); `go list -m` finds 28.
- The 21 existing modules still give **840 / 16 / 0**.
- Staged file count equals the files on disk; `process-cli check` stays clean. Scoped runs only. Nothing pushed.

## Not in this scope

`identity/auth`, `analytics/visitoractivity`, `jobs/scheduler` (+ `schedcli`) (6b); `real-estate/property`,
`real-estate/transaction` (6c); any refactor, genericizing `cms`, kits' bos wiring (D15), process-os definitions.
Client information is not touched: when needed it lives in the client library the main session is building
(`processos-workspace/definitions/clients/` + `records/clients/`), registered after alignment with that session,
never in `platform/`.
