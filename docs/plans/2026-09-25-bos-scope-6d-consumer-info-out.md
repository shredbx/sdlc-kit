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

## Results

Commits in sdlc-kit: `ca8551c` (corrections) and `128b779` (moves). In the consumer's repo, local and unpushed:
`4f15efe` (the plan and the recon) and `e3e5798` (the regroup, below).

| Gate | Result |
|---|---|
| Name scan, commit 2's files | design doc and `platform/CLAUDE.md`: 0 hits. This plan: 1, the abbreviation `BR` named in the gate's own description |
| Move check | recon sha256 `c42cee1c…037dc6` identical before deleting the original; design doc 588 → 562 lines; the consumer plan's sections 1 to 5 are cut from the design doc by anchor |
| Skip claim | three files corrected; a search for the old wording finds only the quotations in this file |
| Trial merge | one conflict, `.gitmodules` |
| `platform/` code | 0 files under `platform/go` or `platform/svelte` changed by 6d or by the merge |

Left as they are, by the softened rule: three historical plan logs (5 lines that name the consumer), the main session's
own research files and clients registry, and 73 ported files that name the app they came from in comments and fixtures.

## Sync with the main session

The user said the main session had finished refactoring where client repos are mounted. Its shape is
`consumers/clients/<client>/<project>/` (one repo per project; a client may have several).

1. **Merge** `feature/agent-framework` (`d8440d9`) into this branch: `a353b16`. One conflict, `.gitmodules`: the main
   session's nested mount for its own client repo plus this branch's consumer mount, both kept. Everything else merged
   cleanly, including the research README, which both sides had edited in different places. The merge changed no file
   under `platform/go` or `platform/svelte`.
2. **Move the consumer mount** under `consumers/clients/<client>/<project>` with `git mv`, and rename its `.gitmodules`
   section to match its path, as the main session did for its own: `d94184c`. The pointer stays at the consumer repo's
   bootstrap commit `9715beb`; its later commits are local, so recording them would point at commits nobody else has.
3. **One more `../`** in the consumer's `processos.yaml` `libraries:` path (four levels up to sdlc-kit's root instead of
   three), and its README names the new mount: `e3e5798`. Negative control: with the old three-level path
   `process-cli check` fails (`libraries[0].path: not_found`); with the new one it passes.
4. **Gates re-run after the merge**, all equal to before: Go 34 modules 1,790 pass / 33 skip / 0 fail, identical per
   module to the 6c baseline; `go vet` 34 modules 0 failures; `gofmt -l` 0 files; `pnpm install --frozen-lockfile` up to
   date; Svelte suites 81 / 10 / 396 / 251 / 10 / 12 / 55 / 160 / 45; `process-cli check` ok from sdlc-kit's root and from
   inside the consumer's repo. Python was not re-run: nothing in it is this branch's.
5. **Known local wart:** git's own `.git/config` in this checkout still registers the mount under its old submodule name
   (the working tree and history are unaffected). `git submodule status` therefore shows it with a `-` here; a fresh clone
   initialises it under the new name. Nothing is pushed.
