# Unit model, Scope U0 — design and decisions (plan and log)

Last updated: 2026-09-26
tags: unit-model, scope-U0, decisions, plan

> Design: `docs/proposals/unit-model-design.md`. Approved by the user on 2026-09-26 ("lets implement and see results and
> usability and ux to decide its final structure or refactor more") after the tree below was shown. Introduces **zero**
> definitions and touches **no** code under `platform/`. Nothing is pushed.

## Before → after (new and modified paths only)

```
BEFORE                                                   AFTER
docs/proposals/ (no unit-model doc)                      docs/proposals/unit-model-design.md    NEW
docs/proposals/bos-system-design.md                      MODIFIED  section 3.2 (generic pointer), D16, one decision-log entry
records/sbx-sdlc-kit/architecture/decision/
  platform-position-in-namespace/first.yaml (deferred)   first.yaml   MODIFIED  status: superseded
                                                         second.yaml  NEW  decided for the U1 trial: the stack is a field
  (no such topics)                                       unit-definitions-as-records/first.yaml   NEW
                                                         conform-reaches-output-only/first.yaml   NEW
docs/plans/2026-09-26-unit-model-scope-U0.md             NEW  this file
definitions/ · other records · platform/ · consumers/    UNCHANGED
```

## What differs from the tree that was shown

- The second `platform-position-in-namespace` record is `second.yaml`, not a dated file: `process-cli create` accepts only
  lower-case words joined by hyphens as a file name, and `first.yaml` is the precedent.
- The decisions say "decided for the U1 trial, with a review point", not final, because the user wants to judge usability and
  UX before fixing the structure. The `decision-status` enum has no "trial" value, so the wording carries it.

## Results

| Gate | Result |
|---|---|
| The four decision records against the `decision` schema | all `ok` |
| Negative control: the same record with `topic` removed | rejected, as intended |
| The deferred record | status edited in place to `superseded`; the new record's `supersedes` names its path |
| `process-cli check` | `processos.yaml: ok` |
| Client-name scan over every touched file | no hits |
| Paths the design document cites | all exist |
| `platform/` | 0 files changed, nothing untracked under it |

## A side effect worth recording

While reading library files, a shell command ran inside `processos-workspace/libraries/process-os/`. The protect-mcp hook
wrote `receipts/` and `review-receipts/` there, and `process-cli check` then failed ("only kind folders and namespaces go in a
scope or a namespace"). They were untracked, git-ignored, one unsigned line each, created at the moment of that command; both
were removed and the check passes again. Rule: run shell commands from the repository root.

## Next

U1 (Python): forked schemas, types and the `readme` template, and records for the Python units, with the rendered README
compared byte for byte against the existing one. Its tree is shown before anything is written.
