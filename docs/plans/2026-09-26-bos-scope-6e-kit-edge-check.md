# bos Scope 6e — the kit-edge check (plan and log)

Last updated: 2026-09-26
tags: bos, scope-6e, tools, ci, plan

> Design: `docs/proposals/bos-system-design.md` (section 3.1, "Dependency direction"). Follows Scope 6d
> (`docs/plans/2026-09-25-bos-scope-6d-consumer-info-out.md`). Introduces **zero** process-os definitions and touches **no
> ported code**. Approved by the user on 2026-09-26 ("confirm") after the tree was shown, with one addition: the
> Svelte package count in the design doc is corrected (below). Nothing is pushed. The number 6e was first meant for a bulk
> scrub of client names in the ported code; that idea was dropped, and this scope reuses the number.

## Why

Section 3.1 says shared kits never import feature kits, kits do not form cycles, and a feature kit imports another only
through a declared edge, "an architecture test enforces this later". Every port scope has been recording edges by hand
(the 6a and 6b logs each carry a paragraph). With all 34 Go modules and 12 Svelte packages in, the rule can be
checked mechanically, before kit slicing starts moving code around.

## Before → after (new and modified paths only)

```
BEFORE                                              AFTER
platform/ has no tools/ folder                      platform/tools/check_kit_edges.py    NEW  standard library only, Python 3.11+
                                                    platform/tools/kit-edges.toml        NEW  rules as data
.github/workflows/ (go, svelte, python)             .github/workflows/kit-edges.yml      NEW
platform/CLAUDE.md                                  MODIFIED  the "declared edge" rule points at the check and the toml
docs/proposals/bos-system-design.md                 MODIFIED  section 3.1 wording; Svelte package-count correction;
                                                              M0 "≈14 (12 done)"; scope list; decision-log entry
                                                    docs/plans/2026-09-26-bos-scope-6e-kit-edge-check.md   NEW  this file
platform/go · platform/svelte · go.work · lockfile  UNCHANGED
```

## What the check does

- **Kit of a package** = its folder: `platform/<lang>/packages/<kit>/…`.
- **Edges:** Go, the direct requires (not `// indirect`) of other workspace modules in each `go.mod`. Svelte, the
  `workspace:` entries in `dependencies`, `peerDependencies` and `optionalDependencies`. Edges inside one kit are ignored.
- **Rules,** in `kit-edges.toml`: the 10 shared kits (from section 3.1); every other kit is a feature kit; the declared
  feature-to-feature edges, each with a reason; the known shared-to-feature violations, each with a reason.
- **It exits 1 on:**
  1. a shared kit depending on a feature kit that is not a listed known violation;
  2. a feature kit depending on another that is not a declared edge;
  3. a cycle among kits (per stack);
  4. a stale entry (an edge listed that no package depends on any more), a declared edge that involves a shared kit, a
     known violation that is not shared-to-feature, or a shared kit with no package on either stack.
- **It does not** read Go source imports (one module per package makes `go.mod` the source of truth, and `go mod tidy`
  keeps it honest), and it says nothing about kit wiring, which does not exist yet (D15).

## The Svelte package count, corrected

Section 3.1 said "18 packages under `sbx/packages`" and listed `receipts` and `review-receipts` as unported ones. Each of
those two folders holds one `receipts.jsonl`: hook logs, not packages. The real count is 17: 15 top-level folders plus
`core/svelte` (which is `@sbx/core-ui`) and `core/ts/animations`. 12 are in the app's closure; the other 5 (`ui-video`,
`i18n-svelte`, `ui-code`, `ui-diagram`, `games`) are not imported by the app. Checked read-only against the source
(2026-09-26): every `@sbx/*` package the app needs is ported, and no ported package is outside its closure.

## Results

| Gate | Result |
|---|---|
| On today's tree | exit 0; **19 Go and 8 Svelte kit edges**, 4 declared, 1 known violation, 0 errors |
| Edge counts vs. the ones recorded by hand in this session | equal (19 and 8); adding `peerDependencies` and `optionalDependencies` to the Svelte side changed nothing |
| Negative controls (scratch copy of the manifests, the repo untouched) | 9 scenarios, each as intended: baseline exit 0; `money` requiring `cms` fails; the `cms`→`contacts` line deleted fails; a stale `faq`→`cms` entry fails; a `money`⇄`persistence` cycle fails; `units` depending on `ui-seo` fails; a known violation filed as declared fails; a typo in the shared list fails; `jobs`→`news` removed from the code with its entry left fails |
| `ruff check` and `ruff format --check`, with the repo's own settings (`platform/python/pyproject.toml`) | clean (the formatter changed two cosmetic lines, then the check and the controls were re-run) |
| Workflow file | parses as YAML; its one command is the one run locally. It can only run on GitHub after a push, so its first real run is still ahead |
| `platform/go`, `platform/svelte` | 0 files changed; `process-cli check` ok; nothing pushed |

**One earlier decision is changed on purpose.** The 6a log said the feature-to-feature imports (`faq`→`seo`, `cms`→`seo`,
`cms`→`contacts/socialnetwork`) would be declared "when kit wiring lands (D15), not before". The user's approval of this
scope's tree included allow-listing them now, so they are declared in `kit-edges.toml` with their reasons; kit wiring may
later move or replace them. The 6b log recorded `jobs`→`news` (`scheduler` imports `news/feed`), the one known violation.
`real-estate`→`seo` (`collection` and `property`) was not in any log; the edge scan found it. Every other edge is
feature-to-shared or shared-to-shared and needs no declaration.
