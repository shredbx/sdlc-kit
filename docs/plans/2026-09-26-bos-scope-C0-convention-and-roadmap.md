# bos Scope C0 — the convention and the roadmap (plan and log)

Last updated: 2026-09-26
tags: bos, scope-C0, convention, roadmap, docs, plan

> Design: `docs/proposals/bos-system-design.md` (sections 6.6 and 8.2, decision D17). Approved by the user on 2026-09-26 ("yes": the
> four configuration layers, and C0 before U2a) after the tree below was shown. **Docs only**: no definitions, no records, nothing under
> `platform/`. Nothing is pushed.

## Why

The user asked for the milestone roadmap of what the frameworks give a consumer, and for the structure that configures static pages,
branding, styles and settings, so that the first consumer becomes as thin as the agent application is meant to be. The analysis lived
only in chat. Recording it makes it something later scopes can be checked against.

## Before → after (new and modified paths only)

```
BEFORE                                                       AFTER
docs/proposals/bos-system-design.md                          MODIFIED  D17; section 6.6 (the convention: four layers, six rules);
                                                                       section 8.2 (what a consumer uses at each rung); one log entry
docs/proposals/unit-model-design.md                          MODIFIED  one pointer line
consumers/clients/<client>/<project>/docs/bos-consumer-plan.md   MODIFIED in the consumer's own repo: section 7, today and tomorrow
docs/plans/ (this file)                                      NEW
platform/ · definitions/ · records/                          UNCHANGED (0 files)
```

## What it says (short)

- **Four layers:** spec (YAML in the consumer, validated by `process-cli` against a schema composed from the enabled kits), seed (readable
  content keyed by slug, applied to fresh environments only), runtime (the database, edited in the admin), environment (secrets, named
  through `connections.yaml`).
- **A static page is content, not code.** A page that needs its own layout is an explicit route override.
- **SvelteKit routes are files**, so bos-svelte generates shim routes from the kit manifests (the Svelte half of D15, settled in M2).
- **Development and production profiles** in the spec, which the earlier consumer layout lacked.
- **Same pattern as the agent framework:** registry, spec, shims, bootstrap process, fake adapters selectable by configuration.

## Results

| Gate | Result |
|---|---|
| Client-name scan over every sdlc-kit file touched | no hits |
| Anchors: D17, section 6.6, section 8.2 each present exactly once | yes |
| Sections the new text points at exist (6.2, 8.1, D15, D11) | yes |
| `process-cli check` | `processos.yaml: ok` |
| `platform/`, `definitions/`, `records/` | 0 files changed |
| Consumer repo | one file modified, `docs/bos-consumer-plan.md`; committed locally, not pushed |

## Next

Scope U2a (Go, first slice): three leaf modules, their records, and their READMEs. The README texts are shown for approval before
anything is copied into `platform/`.
