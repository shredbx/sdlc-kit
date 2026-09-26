# The unit model — one structured definition for every package, framework and kit

**Status:** trial, started 2026-09-26 (scope U0). The structure below is decided *for the U1 trial*, then judged on
usability and UX (section 6) and kept, refined or replaced. Every definition it introduces still needs its own approval.

Last updated: 2026-09-26
tags: proposal, unit-model, modeling, documentation, bos

## 1. What and why

The bos system (`docs/proposals/bos-system-design.md`) is meant to behave like Next.js or NestJS: a fixed structure and
fixed configuration files that the frameworks read, so a new consumer edits a few "hello-world" files and runs `make`.
That only works if every part of the system is described the same way. Today it is not:

- Python: 9 units (7 kit packages, 2 frameworks) plus the `process-cli` product; 8 of the 9 have a README, none has a record.
- Go: 34 modules in 18 kits, **0** READMEs and **0** `doc.go`.
- Svelte: 12 packages, **0** READMEs, 1 `description`.

The goal: one **record** per package or framework holding what a newcomer needs (overview, install, usage, configuration,
dependencies), a README **rendered** from it, and checks that put the whole tree through `process-cli` so drift is
caught, not discovered.

## 2. Two homes (`docs/proposals/infrastructure-services-design.md`)

| Side | Holds | Where |
|---|---|---|
| Workspace | schemas, types, templates, actions, processes (definitions) and the records that describe each unit | `processos-workspace/definitions/sbx-sdlc-kit/<capability>/…`, `processos-workspace/records/sbx-sdlc-kit/<capability>/…` |
| Filesystem | the code, and the files rendered from a record (a README first) | the existing `platform/<lang>/{packages/<kit>/<package>, frameworks/<name>}` folders |

Records never live inside `platform/`; what they render does. The code layout is untouched.

## 3. The choices for the trial

Each is recorded as a decision (`records/sbx-sdlc-kit/architecture/decision/`) and rests on existing governance.

1. **Definitions are records**, one folder per unit: a `package` record (what it is) and a `readme` record (what its README
   says), as upstream process-os does. Capabilities by their own descriptions: `modeling` for the package schema and
   records, `documentation` for the readme schema and template. (`unit-definitions-as-records`)
2. **The stack is a field, not a folder.** Records are flat per unit. A platform folder axis is the grid
   `schema-evolution-principles.md` forbids, and `platform/<lang>/` already gives per-platform navigation.
   (`platform-position-in-namespace`, second record, supersedes the deferred one)
3. **Writing and validating reach `platform/` through staging.** `render --into` and `conform` only accept a folder inside
   `runtime.output` (read in `process_framework/framework.py`). `runtime.output` stays scratch; writing is
   render-then-copy and validating is copy-into-output then `conform`. (`conform-reaches-output-only`)
4. **One `package` schema for all stacks**, forked from `process-os.package` (libraries are never edited in place), with
   `kind` (package, framework, product), `stack` and `kit`. Stack-specific data is a sibling `extension/<stack>.yaml`, added
   only when a real check needs it (mechanism 4). Caution found in the fork: upstream's `kind: kit` means "a package in
   `packages/<module>/`", which collides with our meaning of kit as a family. The fork renames it.
5. **"How to configure" is a `readme` section**, not a field: not every package has configuration (mechanism 1 does not apply).
6. **Python first**, because upstream's records are proven data. The processes (`verify`, `create`) come after three real
   instances (Python, Go, Svelte), by the rule of three.

## 4. Evidence gathered 2026-09-26 (scratchpad probes, nothing written to the repo)

- Rendering upstream's 8 `readme` records through the library's `process-os.readme` template works for all 8, and upstream's
  own README on disk equals the render of its record **exactly**. A README is data.
- Our ported READMEs differ from that render only in whitespace inside code fences (1 to 78 bytes: comment spacing changed by
  the formatter during the port). The records therefore carry the formatted code, and render must equal the README byte for byte.
- The `readme` template's `conform` pattern requires `README.md` to hold `# <name>`, `## Install` and `## Usage`.

## 5. The track

| Scope | What | Gate |
|---|---|---|
| U0 | this document, four decision records, a generic pointer in the bos design | records validate, `process-cli check` ok, `platform/` unchanged |
| U1 Python | forked `package` / `readme` / `readme-section` schemas and their small types, the `readme` template, records for the Python units | 8 renders equal their READMEs byte for byte; negative controls fail as intended |
| U2 Go (about 4 scopes) | 34 records plus real READMEs, by kit family, written from package comments, exported APIs and tests | code and tests untouched; each README approved |
| U3 Svelte (1 or 2) | the same for 12 packages | same |
| U4 processes | `verify-unit` (stage, conform, check the record against the manifest) and `create-package`, dispatching on `stack`; a generated navigation index | each process run over every unit |
| bos level | kit join manifests, framework, preset and consumer-spec records, `bootstrap-consumer` | rides on ladder rungs M2 to M4b |

About 10 scopes, plus or minus 30%. U0 and U1 do not depend on the baseline import.

## 6. Review after U1: usability and UX

Judged with real use, not opinion. For each item, note what happened and what hurt.

1. **Find:** from a package folder, how many steps to its overview, install and usage? Compare with reading the README alone.
2. **Edit:** change one README sentence via its record. Count the commands and the friction; is drift caught if someone edits
   the README directly?
3. **Noise:** two records plus a README per unit. Is the `records/` tree navigable with `process-cli list` and `show`?
4. **Gate:** does byte-equality hold for 8 of 8, and do the negative controls fail as intended?
5. **Vocabulary:** do `kind`, `stack` and `kit` read clearly next to the design doc's own terms?
6. **Staging:** was copy-into-output then `conform` clumsy enough to justify widening `runtime.output`?
7. **Would a person rather edit the README?** If yes for most, the source of truth is in the wrong place.

Outcomes: keep, refine (rename a field, add a section), or restructure (for example records beside the code, which needs
`process-cli` to find records outside one root). The choice is recorded as a decision either way.

## 7. Risks and open points

- **Shared schema with `agent-framework`** (the main session). Two definition schemas would be worse than one: agree the
  `package` schema with that thread before U1 writes it. `agent-framework` has no README today.
- **46 Go and Svelte READMEs are real writing**, not generation. Derive from code, label drafts, approve per kit.
- **Formatting drift:** code fences in ported READMEs were reformatted by the port; records must match them.
- **Hook logs from a stray `cd`:** running a shell inside a process-os scope folder makes the hook drop `receipts/` folders
  there and `process-cli check` then fails. Run from the repository root.

## 8. Not in scope

Kit join manifests, presets, the consumer spec and `bootstrap-consumer` (bos level, later); any change to code under
`platform/`; a README for `agent-framework` (the main session's); a boundary checker for record folders.
