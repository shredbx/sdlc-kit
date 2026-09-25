# bos Scope 6d — corrections, and consumer information out of sdlc-kit (plan and log)

Last updated: 2026-09-26
tags: bos, scope-6d, docs, plan

> Design: `docs/proposals/bos-system-design.md`. Follows Scope 6c
> (`docs/plans/2026-09-25-bos-scope-6c-go-real-estate.md`). Docs only: **no code, no process-os definitions**.
> Approved by the user on 2026-09-26 ("confirm") after the tree was shown. Nothing is pushed.

## Why

Two rules, both from the user on 2026-09-26.

1. **sdlc-kit holds no consumer information.** Everything about a consumer (its evidence, layout, oracle numbers, scope
   plans, recon) lives in that consumer's own repo. The design doc and a recon file had grown consumer-specific parts.
2. **Softened for ported code:** the repo is private and never shared, so client names in the ported code's comments and
   test fixtures are not scrubbed in a bulk scope. They are scrubbed along the way, in each kit's own refinement slice.
   The earlier plan for a "Scope 6e" bulk scrub is dropped.

Two recorded claims were also wrong and needed correcting (found by checking, not by a later change):

- "All 29 skips need a live Postgres." Read one by one, the 33 skips at the end of 6c are: 13 that need a DSN
  (`VISITOR_ACTIVITY_TEST_DSN`, `analytics/visitoractivity` only), 13 unconditional "RED placeholder" skips for
  unimplemented features (`dictionary` 8, `repository/postgres` 5), and 7 unconditional pointers at app-level tests
  (`dictionary` 3, `transaction` 4). A live Postgres could enable only the 13.
- "A live bearer token at rest" for the magic-link session defect in `identity/auth`. Reproduced in a scratch copy, it is a
  functional bug (magic-link sessions cannot be refreshed; `Logout` for them is a silent no-op) plus an F1 hygiene
  violation, not an exposed credential.

## Before → after (new, modified and removed paths only)

```
sdlc-kit (this worktree)
docs/plans/…scope-3-go-l0-l1-ports.md                      MODIFIED  skip claim
docs/plans/…scope-6b-go-identity-analytics-jobs.md         MODIFIED  skip claim + auth defect wording
docs/proposals/bos-system-design.md                        MODIFIED  skip claim, auth wording (commit 1); consumer parts
                                                                     moved out, the rest made generic (commit 2)
docs/research/<first-consumer>-decomposition-recon.md      DELETED   moved to the consumer's repo
docs/research/README.md                                    MODIFIED  index entry of the moved file removed
platform/CLAUDE.md                                         MODIFIED  rule restated to the real policy
docs/plans/2026-09-25-bos-scope-6d-consumer-info-out.md    NEW       this file
platform/ code · .gitmodules · the submodule pointer       UNCHANGED (the mount moves in the sync, not here)

the first consumer's repo (one local commit, not pushed)
docs/research/decomposition-recon.md                       NEW       moved, byte-identical (sha256 checked)
docs/bos-consumer-plan.md                                  NEW       the moved design-doc parts + the Scope 7 recon
README.md                                                  MODIFIED  points at docs/
```

Two commits in sdlc-kit: the corrections first, then the moves. The corrections are in `ca8551c`.

## What moved out of the design doc, and what stayed

| Was in the design doc | Now |
|---|---|
| section 2, Evidence (route counts, LOC splits, prior attempt) | consumer plan, section 1; a 7-line stub stays |
| section 6.2, the consumer's layout | consumer plan, section 2; a generic consumer layout stays |
| section 6.5, app-owned code → kits | consumer plan, section 3; a pointer stays |
| section 7: suite counts, the do-not-"fix" list | consumer plan, section 4; a generic sentence stays |
| D1, D3, the ladder's M0, M4b and M5 rows, the "worth checking" bullet | consumer plan, section 5; generic wording stays |
| — (never written down) | consumer plan, section 6: the Scope 7 recon, the proposed 7a to 7d split, the two binding spikes, the traps |

Sections 3 to 5, 6.1, 6.3, 6.4, 8.1 and the method stay. The auth defect (section 7, item 1) stays: it is about a shared
library, not a consumer.

## Gates

1. **Name scan** over the sdlc-kit files touched by commit 2 (design doc, `platform/CLAUDE.md`, this plan) for the
   consumer's and the project's names, the `BR` abbreviation, and its app-internal paths: no hits. The research README's
   remaining hits are entries for two research files the main session already removed; the sync clears them.
   Informational, not blocking, per the softened rule.
2. **Move check:** the moved recon is sha256-identical in both places; every part cut from the design doc is present in
   the consumer plan, and the consumer plan's section bodies are cut from the design doc by anchor, not retyped.
3. **Skip claim** now reads 13 + 13 + 7 = 33.
4. **Trial merge** with the main session's branch (`git merge-tree`).
5. `platform/` code unchanged, `process-cli check` ok, nothing pushed.
