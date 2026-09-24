# Schema evolution: four mechanisms, not one

**Status:** decided 2026-09-24 — see decision record
`sbx-sdlc-kit/architecture/decision/schema-evolution-mechanism`.

When an entity in this workspace needs new capability, reach for one of these four, in this
preference order. Never invent a fifth ad hoc.

## 1. Extend the schema directly

Add a field to the existing schema. Only when *every* instance needs it.

Example: `decision` gained a required `topic` field because every decision needs one.

## 2. Nest

Narrow scope with a deeper folder path — one real segment at a time, never pre-built.

Example: `sdlc.python` (process-os) is a plain subfolder inside the single `sdlc` scope —
`processos-workspace/definitions/sdlc/python/`, not a second `scope.yaml`. It exists because
real Python-specific content exists, not because every future platform was pre-declared.

## 3. Compose

A field whose type is another mapping schema — structure that's always present, embedded inline
in the same record.

Confirmed against the actual engine, not assumed: `Field.type` is a name, and `types.get(name)`
resolves it the same way whether it names a scalar type or a `mapping` schema
(`process_kit.types.field.Field.validate`). So `field: {type: some-other-mapping-schema}` really
does embed that schema's structure.

## 4. Extend via `requires` — sibling extension file

A *process* declares `requires: {name: schema}` — not the base schema. The data lives in a
sibling `extension/<name>.yaml` beside the base record, present only for instances that actually
go through that process. The base schema never changes for it.

Real example, process-os's own workspace: `sdlc.build-application` requires `build: build`; the
data is `extension/build.yaml` beside the application's own `application.yaml`.
`sdlc/schema/build.yaml`'s own comment: *"The application provides it as extension/build.yaml,
beside its application.yaml, and a process that builds names it in requires. The application
schema does not change for it."*

## The guardrail: never pre-build a grid

shredbx (v1) crossed NFR quality categories with platforms: `nfr.yml` × `nfr-platform-matrix.yml`
— 7 categories × 4 platforms = 28 cells, each needing `{must, should, verification, patterns}`
whether or not a real rule existed for it. sbx.framework (v2) explicitly ported only the flat half
(`guideline.category` + `guideline.tier`) and dropped the platform axis
(`docs/research/2026-08-06-old-sbx-ui-protocols.md:757`).

Mechanisms 2 and 4 above both narrow scope. The rule for both: add one real segment — a folder, a
`requires` extension — only when one real rule or process needs it. Never declare a
category × platform (or any N-dimensional) cross product up front.

## Where each mechanism physically lives

| Mechanism | Lives in | Governed by |
|---|---|---|
| Extend | the schema's own `fields:` | the schema definition itself |
| Nest | a deeper folder path under a scope | CLAUDE.md's namespace discipline |
| Compose | a `Field.type` naming another mapping schema | the schema definition itself |
| Extend via `requires` | a sibling `extension/<name>.yaml` | the consuming process's own `requires:` block |
