# Unit model, Scope U2c — Go, three modules with third-party requirements (plan and log)

Last updated: 2026-09-26
tags: unit-model, scope-U2c, go, records, readme, requirement, dependencies, plan

> Design: `docs/proposals/unit-model-design.md`. Follows `docs/plans/2026-09-26-unit-model-scope-U2b.md`. The one definition change (the
> `requirement` type) and the three README texts were shown to the user and approved before anything was written; the texts before anything
> was copied into `platform/`. Touches **no** Go code, `go.mod`, test or other definition. Nothing is pushed.

## The slice

`localization/language` (60 lines), `identity/rbac` (374), `persistence/database` (965): level-0 modules with third-party requirements and no
in-repo ones. The other two, `news/rss` and `media/image`, are larger (1,937 and 2,265 lines) and form U2d.

## Before → after (new and modified paths only)

```
BEFORE                                                      AFTER
definitions/sbx-sdlc-kit/modeling/type/requirement.yaml     MODIFIED  the pattern gains a Go alternative (path@version); comment updated
records/.../modeling/package/ (17 units)                    {language, rbac, database}/{package.yaml, readme.yaml}       NEW × 6
platform/go/packages/{localization/language, identity/rbac, persistence/database}   README.md  NEW × 3  (rendered from the records, copied in)
docs/plans/ (this file) NEW · unit-model-design.md MODIFIED (section 9) · bos design: one log line
platform/CLAUDE.md · every other definition · Go code · go.mod · go.sum · tests · go.work   UNCHANGED (0 files)
```

## The `requirement` type: one form per stack

A dependency is written as its stack's manifest writes it, without spaces: Python `pyyaml>=6`, Go `path@version` (as `go get` writes it,
`github.com/jackc/pgx/v5@v5.9.1`). The pattern is an alternation and gains one alternative per stack, when that stack's first record needs
it; the npm form comes with U3. Checked through `process-cli`: all 17 existing records still validate, 9 real forms are accepted (Go paths,
a pseudo-version, `+incompatible`, Python forms) and 7 bad ones rejected (no version, a space, `uuid@1.6.0`, empty, the npm form until U3).
A string of one stack can pass on a record of another; the verify step, which knows `stack`, is where that is caught, not the schema (a
per-stack type keyed on `stack` would be the platform grid the governance forbids).

## What `dependencies` and `uses` hold

- `uses`: in-repo units, by id. `dependencies`: third-party **runtime** requirements. Dev and test requirements are not recorded.
- I first proposed recording every direct `require`, `testify` included. That contradicted the rule the eight Python records already follow
  (their dev group is left out), so it was corrected before anything was written.
- Measured over all 34 Go modules with `go list -deps`: 47 direct third-party requires, 7 test-only (all `testify`), 15 distinct third-party
  modules, none required at two versions.
- **The gate** compares a record's `dependencies` with the third-party modules that non-test code imports (`go list -deps`), at the version
  `go.mod` states, not with the text of `go.mod`: `go.mod` lists `testify` as a direct require, and only `go list` shows that no non-test
  code imports it.

## How the READMEs were made

As in U2b: markdown drafts, a scratch converter, a byte round trip (render equals the approved draft). New in this scope:

- **Units with third-party requirements** are wired the way the README says (`require` + `replace`), then `go mod tidy` runs offline from the
  module cache with the real go1.26 binary (`GOSUMDB=off`, `GOPROXY=off`); it made no lookup.
- **A second, compile-only code block** for a unit that needs a service: `persistence/database` has one runnable program (what needs no
  server: schema resolution, the failure path of `New`, version numbering) and one connected program (connect, `Migrate`, `WithTx`) that is
  vetted and built with the network denied but never run.
- Claims were proven with throwaway programs before they went into a README (network denied, no lookups).

## What reading the code found (in the READMEs; no code was changed)

- `rbac`: `GetPermissions` returns the authorizer's own set (for a role outside the hierarchy, the caller's own config map); `DefaultRole` is
  stored and never used; `AvailableRoles` has no fixed order (4 orders in 300 runs); the middleware writes JSON with `http.Error`, so the
  `Content-Type` is `text/plain`; a role in the hierarchy but missing from `Roles` still exists and inherits.
- `database`: the schema name goes into `search_path` and `CREATE SCHEMA` unquoted; errors keep a sentinel but flatten the pgx cause to text
  (`errors.As` cannot reach it); versions sort as text; the advisory lock is taken and released through the pool; `MigrateDown` is not under
  the lock; `WrapPool`'s comment says the `DB` does not own the pool but `Close` closes it; nothing that touches a database is tested.
- `language`: nothing to correct.
- Fixes belong to each kit's own refinement slice.

## Results

| Gate | Result |
|---|---|
| `process-cli check` | `processos.yaml: ok` |
| Records against their schemas | 20 of 20 `package`, 20 of 20 `readme` (8 Python + 12 Go) |
| `name` equals the last segment of `path`; `path` holds a README; `kit` matches its folder | 20 of 20 each |
| Reference integrity: every `uses` id has a record | ok |
| Render vs the README in the record's own `path`, byte for byte | **20 of 20**; `conform` 20 of 20 |
| Draft → record → render equals the approved draft, before anything was copied | 3 of 3 |
| `dependencies` equals what non-test code imports (`go list`); `uses` empty exactly when there are no in-repo requires | 12 of 12 |
| README code blocks: `gofmt`, offline `tidy`, 0 DNS lookups, identical with the network denied; the first block of each README run and equal to the printed output, the second block of `database` compile-only | 13 blocks, 12 of 12 READMEs |
| `go test` on the twelve Go modules vs the baselines taken before | equal (`language` 1/1, `rbac` 44/14, `database` 43/23, and the nine from U2a and U2b), 0 non-pass, resolver canary: no name looked up |
| Negative controls, run on bad input | test-only dependency added, wrong version and dropped dependency each fail the dependencies gate; a one-word record edit makes the render differ; a compile-only program with a type error fails; a README claiming `zz` is a language fails; a Go requirement with a space is rejected by the type; a package without `stack` rejected; a dangling `uses` flagged |
| `platform/` | exactly 3 new `README.md` files; no modified file under `platform/go` |
| Kit-edge check | exit 0, unchanged (4 declared, 1 known violation) |
| Client-name scan of every changed file | no hits |

## Next

Scope U2d: `news/rss` and `media/image` (level 0, larger; `media/image` has nine direct requires, one test-only). Then the level-1 modules
(`http/httputil`, `identity/user`, `location/address`, `news/feed`, `persistence/repository/postgres`, `real-estate/collection`,
`reference-data/dictionary`), whose records carry the first non-empty `uses`.
