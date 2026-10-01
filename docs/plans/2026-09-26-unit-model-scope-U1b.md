# Unit model, Scope U1b — cleanup after the first review (plan and log)

Last updated: 2026-09-26
tags: unit-model, scope-U1b, python, records, identity, plan

> Design: `docs/proposals/unit-model-design.md` (sections 3 and 9). Follows `docs/plans/2026-09-26-unit-model-scope-U1.md`. Approved by
> the user on 2026-09-26 ("ok lets proceed") after the tree below was shown, with `name` kept alongside `path`. Touches **no** code
> under `platform/` and no README. Nothing is pushed.

## Before → after (new and modified paths only)

```
BEFORE                                                       AFTER
records/.../documentation/readme/<unit>/readme.yaml × 8      moved to records/.../modeling/package/<unit>/readme.yaml (git mv); the empty folder is gone
records/.../modeling/package/<unit>/package.yaml × 8         MODIFIED  + path: platform/python/...
definitions/.../modeling/type/                               unit-path.yaml   NEW
definitions/.../modeling/schema/package.yaml                 MODIFIED  + required `path`
records/.../decision/unit-definitions-as-records/            second.yaml NEW; first.yaml status: superseded
docs/proposals/unit-model-design.md                          MODIFIED  section 2, item 7, section 9 (U1b)
docs/plans/ this file NEW · bos design: one decision-log line
platform/ · code · READMEs · other definitions               UNCHANGED (0 files)
```

## Results

| Gate | Result |
|---|---|
| `process-cli check` | `processos.yaml: ok` |
| Records against their schemas, from the new places | 8 of 8 `package`, 8 of 8 `readme` |
| One folder per unit, two records each; the old records folder | true; gone |
| `path` exists and holds a README; `name` equals the last segment of `path` | 8 of 8; 8 of 8 |
| Render of each `readme` record vs the README in the folder its **own** `path` names | **identical byte for byte, 8 of 8**; `conform` on the staged real README 8 of 8 |
| The gate script's lookup table from unit to folder | none: units are found by walking the records folder, folders come from `path` |
| Negative controls | no `path` rejected; an uppercase `path` rejected; a `path` to a folder that does not exist is accepted by the schema, caught by a folder-exists check (the verify process's job) |
| `platform/` | 0 files changed or added |
| Client-name scan | no hits |

## Next

Scope U2a (Go, first slice): loosen `requirement`, three Go records and their three new READMEs, and one rule in `platform/CLAUDE.md`.
