# Port process-os's definitions as a verbatim library — Milestone 9 implementation plan

> **For Claude:** first milestone of `2026-09-25-process-os-port-completion-design.md` (M9–M12). No
> code is ported and zero new types/schemas/actions/processes are modeled — this copies three
> existing scopes byte-for-byte and points `processos.yaml` at them. It goes ahead of the framework
> (M10) and CLI (M11) because the framework's `catalog`-fixture test file and four CLI tests read
> these definitions live.

**Goal:** make process-os's `sdlc`, `std` and `process-os` scopes available to this workspace as
**libraries** — preserved 100%, refined later if needed — without touching our own
`definitions/sbx-sdlc-kit/` namespace.

**Source:** `/Users/solo/Projects/workspaces/process-os/processos-workspace/definitions/{std,sdlc,process-os}`
— 98 files (`std` 4, `sdlc` 37 incl. the nested `sdlc/python/` namespace, `process-os` 57), plus 2
stray `.pyc` files that were deliberately not copied.

**Lands at:** `processos-workspace/libraries/{std,sdlc,process-os}/`, referenced by three `libraries:`
entries in `processos.yaml`, with a guard note at `processos-workspace/libraries/CLAUDE.md`.

---

### Task 1: copy the three scopes verbatim — DONE

`rsync -a --exclude '__pycache__' --exclude '*.pyc'` per scope. 98 files copied. **Proof of "100%":**
`diff -r` against upstream is empty for all three scopes, and the 3 executable scripts in
`process-os` kept their mode bits. Before copying, `git check-ignore` was run on all 98 would-be
paths (this repo's `.gitignore` has broad Python patterns such as `lib/`, `build/`, `dist/`): none
match, and after copying all 98 + the guard note show as untracked-not-ignored.

### Task 2: register the libraries in `processos.yaml` — DONE

Three entries, `name` + `path` only — the same shape as upstream's own `libraries/std` entry.
`access` defaults to `readonly` in the ported `config` package (`entry.get("access", "readonly")`),
so it is not spelled out; `version` is optional and not pinned.

### Task 3: guard note — DONE

`processos-workspace/libraries/CLAUDE.md`, approved as its own item. It sits one level **above** the
scope folders on purpose: checked in a scratch workspace with a negative control, a `CLAUDE.md` there
passes `process-cli check`, while one inside `libraries/sdlc/` fails with "only kind folders and
namespaces go in a scope or a namespace" (same constraint as
`architecture/decision/nested-claude-md-governance`).

### Task 4: prove it — DONE

| Check | Result |
|---|---|
| `process-cli check`, real repo, three entries | `processos.yaml: ok` |
| Negative control: one library path wrong (scratch config via `--config`) | fails: `libraries[1].path: not_found: there is no folder …` |
| Same config, path fixed | `ok` — the failure was the path, nothing else |
| `process-cli list` per scope | `process-os` 27, `sdlc` 22, `std` 3 (library) + `sbx-sdlc-kit` 6 (ours) |
| `process-cli list --scope sbx-sdlc-kit` | unchanged: 4 schemas + 2 types |
| `readonly` enforced (scratch workspace) | `process-cli write type std.version …` → `readonly: scope 'std' is readonly`; library file untouched. Control: same command into a writable scope → `written` |

**A first negative control was invalid and is recorded on purpose.** It failed on
`runtime.root: not_found` (the scratch runtime folder didn't exist), which says nothing about library
paths. It was redone with only the library path wrong, plus a fixed-path counterpart.

No Python was touched, so there is no ruff/mypy/pytest step and no CI step for M9; the coverage-gap
check does not apply (nothing to cover). No deviation from verbatim — the `REPO` path edits come with
M10/M11.

---

## What this changes for everyday work in this repo

- **52 library definitions are now visible** in `process-cli list` and, through it, in the MCP
  `tools/list` that the plugin serves — including `process-os.create-package`,
  `process-os.create-plugin`, `process-os.build-cli`. Those hard-code upstream's layout
  (`products/`, `packages/process-kit/`, `frameworks/`, `dist/`) and would write to wrong paths here:
  **do not run them**. The guard note says so; nothing technical prevents it.
- `sdlc` is layout-agnostic (build target comes from each application's `extension/build.yaml`).
- `process-cli write` into a library is refused (`readonly`). Only `write` was exercised — `remove`
  was not — and only through the CLI; the MCP write/remove tools run on the same framework but were
  not exercised separately.

## Retrospective note (process-os tooling)

All M9 proof ran through `process-cli` directly, not the MCP round-trip. The CLI's `--config FILE`
was what made a scratch negative control possible without editing the real `processos.yaml`; whether
the MCP tools offer an equivalent was not checked. Leaning CLI-first for proof work that needs a
throwaway config.

## Draft notes for wrap-up (not yet records)

Per `porting-and-modeling-process.md`, noted here and graduated only with the user's approval:

- `definitions-as-verbatim-library` — now has evidence behind it: `check` passes, content is visible,
  `readonly` is enforced, ours stays untouched. Candidate for a decided record.
- The three remaining candidates from the design (`process-os-port-scope`, `mcp-transport`,
  `process-os-product-layout`) wait for M10–M12 to add their own evidence.
