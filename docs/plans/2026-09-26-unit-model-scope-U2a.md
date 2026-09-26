# Unit model, Scope U2a — Go, first slice (plan and log)

Last updated: 2026-09-26
tags: unit-model, scope-U2a, go, records, readme, plan

> Design: `docs/proposals/unit-model-design.md`. Follows `docs/plans/2026-09-26-unit-model-scope-U1b.md` and
> `docs/plans/2026-09-26-bos-scope-C0-convention-and-roadmap.md`. The three README texts were shown to the user and approved
> ("so far looks good") before anything was copied into `platform/`. Touches **no** Go code, `go.mod`, test or definition. Nothing is pushed.

## What differs from the tree first proposed

- **The slice changed.** `calendar/ical` was proposed as a leaf. It is not: it requires `calendar`, which requires the repository
  modules. The first dependency parser missed the single-line `require` form. The corrected levels are 14, 7, 10 and 3 (not 17, 5,
  10 and 2). The slice is `seo`, `money` and `location/geocoordinate`, which are true leaves.
- **No `requirement` change.** The three modules have no third-party requirements, so `dependencies: []` passes as it is. The type
  is loosened when the first module with third-party dependencies is recorded.

## Before → after (new and modified paths only)

```
BEFORE                                                       AFTER
records/.../modeling/package/ (8 Python units)               {seo, money, geocoordinate}/package.yaml   NEW × 3
                                                             {seo, money, geocoordinate}/readme.yaml    NEW × 3
platform/go/packages/{seo, money, location/geocoordinate}    README.md  NEW × 3  (rendered from the records, copied in)
platform/CLAUDE.md                                           MODIFIED  one rule: a unit's README is rendered from its record
docs/plans/ (this file) NEW · unit-model-design.md MODIFIED (section 9) · bos design: one log line
definitions/ · Go code · go.mod · tests · go.work            UNCHANGED (0 files)
```

## How the READMEs were made

Each README's prose comes from the package's own comments, exported API and tests. The usage example is a complete program that was
compiled and run against the workspace; the program is embedded in the record, and the output the README shows is what it printed.
One correction came from reading the code: the `money` package comment says "no floating-point errors", but `Multiply` and `Divide`
use a `float64` factor and round, so the README says what the code does.

## Results

| Gate | Result |
|---|---|
| `process-cli check` | `processos.yaml: ok` |
| Records against their schemas | 11 of 11 `package`, 11 of 11 `readme` (8 Python + 3 Go) |
| `name` equals the last segment of `path`; `path` holds a README; `kit` matches its folder | 11 of 11; 11 of 11; 11 of 11 |
| Reference integrity: every `uses` id has a record | ok |
| Render vs the README in the record's own `path`, byte for byte | **11 of 11**; `conform` 11 of 11 |
| The examples in the committed Go READMEs compile, run and print exactly what the README says | 3 of 3 |
| `go test` on the three modules vs the baseline taken before | equal (`seo` 2, `money` 120 with 34 top-level, `geocoordinate` 17 with 4 top-level, none failing) |
| Negative controls, run on bad input | a hand-edited README differs from the render; a package without `stack` rejected; a README claiming the wrong output makes the example check fail; a `uses` id with no record is flagged (the schema accepts it, by design) |
| `platform/` | exactly 3 new `README.md` files and `platform/CLAUDE.md` modified; no modified file under `platform/go` |
| Kit-edge check | exit 0, unchanged (4 declared, 1 known violation) |
| Client-name scan of every changed file | no hits |

## Next

Scope U2b: the remaining level-0 Go modules (11 of the 14), in dependency order. Before it, one decision: how to author the many new
READMEs (see the design document, section 9, U2a).
