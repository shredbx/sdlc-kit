# Complete the process-os port — design (Milestones 9–12)

Last updated: 2026-09-25
tags: process-os, port, design, plan

> **Status:** design agreed with the user in a brainstorming pass on 2026-09-25. Nothing below is
> implemented yet — no code, no definitions, no config edits. Each milestone gets its own
> before/after file tree and explicit go-ahead when it starts, per CLAUDE.md's per-definition
> approval rule. This design settles the three open decisions in
> `docs/proposals/product-example-scaffold.md` (see "Decisions").

**Goal:** finish porting the runnable part of process-os — `process-framework`, `process-cli`,
`process-claude-plugin`, and the definitions they run against — verbatim, so new functionality is
added on top of a complete base. `process-kit` (7 packages) is already ported, Milestones 2–8.

---

## Decisions

| # | Question | Resolution |
|---|---|---|
| 1 | What is the "example product"? | The rest of process-os itself — not a new domain. A port of proven code, so the shape is already known to work (`porting-and-modeling-process.md`). |
| 2 | stdio vs HTTP MCP | **stdio only.** `process-cli mcp` is a local stdio server and the plugin's `plugin.json` spawns it that way. Nothing in this port needs network reach, so HTTP/FastAPI is deferred until a real product does. |
| 3 | Reuse `sdlc.python` cross-repo vs. adapt fresh | **Neither.** Port process-os's definitions (`sdlc`, `std`, `process-os`) **verbatim as a library**, referenced through `libraries:` entries in `processos.yaml`. No live link to upstream, no collision with our `sbx-sdlc-kit` namespace rules; refine later by forking into `definitions/` or through `uses:`. |
| 4 | Layout | Product **`process-os`** at `projects/products/process-os/`, its applications `process-cli` and `process-claude-plugin` directly inside it (no extra `applications/` folder). `process-framework` at `platform/python/frameworks/process-framework`, mirroring upstream's tier. |
| 5 | Excluded | `tools/process-os/cli` (a ~3 MB built binary) and `dist/` (wheels) — build outputs, not source. |

## What was left upstream (inventory)

| Tier | Upstream location | Size | Depends on |
|---|---|---|---|
| Framework | `frameworks/process-framework` | 1,523 src + 800 test LOC, 79 tests | the 7 kit packages |
| CLI | `products/process-cli` | 747 src + 704 test LOC, 74 tests, YAML fixtures | process-framework, `mcp>=1.12` |
| Plugin | `products/process-claude-plugin` | 12 files, no Python | process-cli at runtime |
| Definitions | `processos-workspace/definitions/{process-os, sdlc (incl. sdlc.python), std}` | 98 files (`std` 4, `sdlc` 37, `process-os` 57) | — |

## Layout and workspace changes

- `platform/python/pyproject.toml`: `members` grows to `packages/*`, `frameworks/*`,
  `../../projects/products/process-os/*`, with the plugin excluded (it has no `pyproject.toml`,
  as upstream also excludes it). **Verified in a scratch workspace with uv 0.8.24:** a member
  outside the workspace root, reached through `../../`, locks and syncs as an editable install
  under a single `uv.lock`. Not yet tried: the exact deeper glob plus `exclude` in the real tree.
- `mypy_path` extended for the framework. For the CLI, whether ruff/mypy config reaches a project
  outside `platform/python/` is **unverified** — resolved in M11.
- CI: one step per package, as for the kit packages (a combined `pytest` run collides at
  collection — expected, see CLAUDE.md).
- `processos.yaml`: three `libraries:` entries (`std`, `sdlc`, `process-os`), added in M9.

## Milestones

Every milestone follows the routine of Milestones 2–8: verbatim copy, scoped pytest (not the full
suite), ruff, mypy, one CI step, a plan doc in `docs/plans/`, and a coverage-gap check.

| M | Work | Lands at | Proof |
|---|---|---|---|
| 9 | Definitions library, verbatim | `processos-workspace/libraries/{sdlc,std,process-os}/` + `libraries:` entries + a guard note | `process-cli check` |
| 10 | `process-framework` | `platform/python/frameworks/process-framework` | its scoped pytest, ruff, mypy, CI step |
| 11 | `process-cli` | `projects/products/process-os/process-cli` | same |
| 12 | `process-claude-plugin` | `projects/products/process-os/process-claude-plugin` | manifest and file check; no Python |

The library goes first because tests in M10 and M11 read live definitions.

**M9 details.** Copy the three scope folders (strip `__pycache__`). **Verified in a scratch
workspace:** with the copies given as `libraries:` entries, `process-cli check` reports
`processos.yaml: ok` and the library's types appear in `process-cli list`. The guard note lives at
`processos-workspace/libraries/CLAUDE.md`, one level above the scopes — **verified with a negative
control:** there it passes `check`, while a `CLAUDE.md` inside `libraries/sdlc/` fails with "only
kind folders and namespaces go in a scope or a namespace" (the same constraint recorded in
`architecture/decision/nested-claude-md-governance`). The note says: verbatim upstream copy; refine
by forking into `definitions/`, never in place; do not run the `process-os.*` actions here until
refined (see unknown 3).

**M10 details.** One test file, `tests/mcp/schema/test_demo.py`, uses a `catalog` fixture that opens
the live definitions folder.

**M11 details.** Four tests (`tests/main/demo/`, `tests/main/demo-libraries/`) read live
definitions. They also run `uv sync`, `uv add typer` and `uv run pytest` on a generated project, so
they need `uv` and network access; upstream skips them when `uv` is absent.

**M12 details.** `plugin.json` declares `mcpServers` → `process-cli mcp`. The skill's
`tools/*.md` reference docs are generated upstream by `process-os.build-skill-reference`, whose
paths are upstream's — so they arrive as static copies.

## Deliberate deviations from verbatim

Only the `REPO` lines in four test files, plus the definitions folder they open:

- `products/process-cli/tests/main/demo/conftest.py`
- `products/process-cli/tests/main/demo-libraries/conftest.py`
- `products/process-cli/tests/main/demo-libraries/test_digest.py`
- `frameworks/process-framework/tests/mcp/schema/conftest.py`

Each computes the repo root as `Path(__file__).parents[5]` (upstream's depth). At both new
locations the repo root is `parents[7]` (counted from the intended paths), and
`processos-workspace/definitions` becomes `processos-workspace/libraries`. The libraries folder
holds `sdlc`, `std` and `process-os` as sibling scopes, so tests that open it as a folder of scopes
should behave as before — **this is not yet verified against the real tests**; M10 and M11 will
show it.

## Known unknowns (resolved inside the milestones, not here)

1. **ruff/mypy outside `platform/python/`** — will the config reach `projects/products/process-os/*`?
   (M11)
2. **Plugin name collision** — this repo already consumes upstream's `process-claude-plugin` under
   the same name. (M12)
3. **`process-os.*` actions carry upstream's layout** — `create-plugin` writes `products/<name>/`,
   `build-skill-reference` targets `products/process-claude-plugin/...`, `create-package` and
   `build-cli` assume `packages/process-kit/`, `frameworks/`, `dist/`. Preserved verbatim,
   guarded, not fixed. By contrast `sdlc` is layout-agnostic: its build target comes from each
   application's own `extension/build.yaml`, so it does not touch `apps-vs-applications-naming`.
4. **The library is live** — it appears in `process-cli list` and the MCP `tools/list`, so
   `process-os.create-package` becomes callable in this repo.
5. **"framework" has two senses** — process-os's `frameworks/` tier vs. our
   `namespace-dispatch-within-capability` (a third-party framework such as `nextjs`). Upstream's
   folder is kept, per the user's call; watch for confusion.

## Parallel work with `main`, and syncing back

State checked 2026-09-25:

- This branch (`worktree-agent-a60b8643af6cd3393`) and `main` share merge-base `ebfd15b` and are one
  commit apart each. `main` added `docs/proposals/infrastructure-services-design.md`,
  `processos-workspace/definitions/CLAUDE.md` and the `nested-claude-md-governance` decision. This
  branch added `docs/proposals/product-example-scaffold.md`.
- `git merge-tree --write-tree HEAD main` produced a tree with no conflict output — a clean merge.
- No path overlap: this work creates files under `docs/plans/`, `processos-workspace/libraries/`,
  `projects/products/process-os/` and `platform/python/frameworks/`. `main`'s services design
  creates `projects/services/<bundle>/` — a sibling under `projects/`, a different subtree.
- `main`'s checkout has uncommitted changes (`.claude/settings.json`, untracked `.claude/skills/`,
  `skills-lock.json`) unrelated to any path here. Its `.claude/worktrees/` is untracked and not
  ignored, so a `git add -A` there would try to pick this worktree up.

**Files both sides could plausibly touch:** `CLAUDE.md` ("Where things stand"), `processos.yaml`,
`platform/python/uv.lock`, `.github/workflows/python-ci.yml`, and the decision-record folders (a
conflict only if both add the same topic). `uv.lock` is generated — on conflict, run `uv lock`
instead of hand-merging.

**Sync protocol:**
1. **Integration happens from `main`'s side** (user's instruction, 2026-09-25): this branch does not
   merge `main` in, and nothing here touches `main`'s checkout. A read-only
   `git merge-tree --write-tree HEAD main` at each milestone boundary is still worth running to see
   drift early (at M9: one commit behind, clean).
2. Whoever integrates runs `process-cli check` and the scoped tests of what was ported.
3. Use `process-cli` with `--config` pinned to this worktree's `processos.yaml` — not the process
   MCP, which loads its catalog once at startup and goes stale mid-session (see the M9 plan's
   retrospective note).
4. Never use bare `git stash` (the stash stack is shared across worktrees).

## Decision-record candidates

Not created here. Each needs its own approval, and per `porting-and-modeling-process.md` they
graduate at a milestone's wrap-up, not mid-task:

- `process-os-port-scope` — decided (framework + CLI + plugin + definitions library; build outputs
  excluded).
- `mcp-transport` — stdio now, HTTP deferred.
- `definitions-as-verbatim-library` — decided.
- `process-os-product-layout` — decided.

## Out of scope

HTTP MCP / FastAPI; `apps-vs-applications-naming` (no `apps/` folder is created here);
`platform-position-in-namespace`; refining the ported definitions; correcting the `process-os.*`
action paths; `tools/` and `dist/`.
