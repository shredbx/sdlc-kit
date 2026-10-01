# Unit model, Scope U1 — Python (plan and log)

Last updated: 2026-09-26
tags: unit-model, scope-U1, python, records, readme, plan

> Design: `docs/proposals/unit-model-design.md`. Follows `docs/plans/2026-09-26-unit-model-scope-U0.md`. Approved by the user on
> 2026-09-26 ("ok") after the tree below was shown. Touches **no** code under `platform/` and no README. Nothing is pushed.

## Before → after (new and modified paths only)

```
BEFORE                                                     AFTER
definitions/sbx-sdlc-kit/scope.yaml   (uses: [])           scope.yaml  MODIFIED  uses: [std]   (std.version)
definitions/sbx-sdlc-kit/ (architecture/, infrastructure/)
                                                           modeling/type/         package-kind · package-name · package-list · requirement · requirement-list · stack   NEW (6)
                                                           modeling/schema/       package                                                                             NEW
                                                           documentation/type/    readme-command-list · readme-section-list                                           NEW (2)
                                                           documentation/schema/  readme · readme-section                                                             NEW (2)
                                                           documentation/template/readme/   template.yaml · files/README.md.jinja                                     NEW
records/sbx-sdlc-kit/                                      modeling/package/<unit>/package.yaml         NEW × 8
                                                           documentation/readme/<unit>/readme.yaml      NEW × 8
docs/                                                      this file NEW · unit-model-design.md MODIFIED (section 9) · bos design: one log line
platform/ · agent-framework · process-cli · other definitions   UNCHANGED (0 files)
```

The units are `process-kit-{types, schema, action, filesystem, config, template, process}` and `process-framework`.
`agent-framework` (the main session's) and `process-cli` (a product outside `platform/`) are left out on purpose.

## What was forked, and how

- **Definitions** are forks of the `process-os` library's (never edited in place), written with `process-cli write`, which
  verifies each one as it lands. Changes from upstream: `package-kind`'s value `kit` is `package` here (a kit is a family, not one
  package); `package` gains a required `stack` and renames `module` to `kit`; the new `stack` type is `[python, go, svelte]`; the
  `readme` template's `files/` are copied byte for byte.
- **Records** are upstream's data ported: `package.yaml` gets `stack: python` and `kit: process-kit` (a framework has none).
  `readme.yaml` is upstream's, with the whitespace in its code fences changed to what the ported README holds (the port's formatter
  had changed comment spacing). A build loop patched each record until the render equalled the existing README.

## Results

| Gate | Result |
|---|---|
| `process-cli check` | `processos.yaml: ok` |
| Records against their schemas | 8 of 8 `package`, 8 of 8 `readme` valid |
| Render of each `readme` record vs the README in the package folder | **identical byte for byte, 8 of 8** |
| `conform` on the real README, staged into `output/` | 8 of 8 ok |
| Negative controls | `kind: kit` rejected; a `readme` without `install` rejected; a `package` without `stack` rejected; a README without `## Install` rejected by `conform`; one changed character in a README **not** caught by `conform`, caught by the byte compare |
| Extra: record vs the package's `pyproject.toml` (name, version, dependencies) | 8 of 8 agree |
| `platform/` | 0 files changed or added; no README changed |
| Client-name scan of every new file | no hits |

## Usability findings (checklist, design doc section 6)

See `docs/proposals/unit-model-design.md`, section 9, which carries the table and the recommendation.
