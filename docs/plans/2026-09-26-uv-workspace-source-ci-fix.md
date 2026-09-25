# python-ci fix: `process-framework`'s uv source, and adding `agent-framework` to the gate

Last updated: 2026-09-26
tags: process-os, ci, uv, agent-framework, fix

> **For Claude:** `python-ci` had been failing on every push to `feature/agent-framework` since
> `243ab38` (Milestone 11, see `2026-09-25-process-os-apps-import-milestones-11-12.md`'s Task 2 /
> deferred-item 3) — the CLI was never wired into CI, so nobody noticed. This doc records the root
> cause, the fix, and what's still deferred.

## Root cause

`platform/python/pyproject.toml`'s workspace glob (`../../projects/products/process-os/*`) makes
`process-cli` a workspace member. Its own dependency on `process-framework` (also a member, under
`frameworks/`) was declared as a `path` source — the F2 workaround chosen in M11, because uv <0.12
could not resolve `{ workspace = true }` for a member reached only via a glob outside the workspace
root (`platform/python`).

uv >=0.12 fixed that resolution *and* started rejecting the old workaround: a real workspace member
must now declare a same-workspace dependency as `{ workspace = true }`, full stop. CI's
`astral-sh/setup-uv@v3` pins `version: latest`, so every CI run since uv crossed that line has died
at `uv sync` — before ruff, mypy, or a single test ever runs. Confirmed both directions locally:

| uv version | `path = "..."` (old) | `{ workspace = true }` (upstream's own shape) |
|---|---|---|
| 0.8.24 (this machine's prior install) | works | fails: "not a workspace member" |
| 0.12.19 (CI's `latest`, and this machine's after the fix) | fails: "must use workspace = true" | works |

## Fix applied

1. `projects/products/process-os/process-cli/pyproject.toml`: `process-framework` source back to
   `{ workspace = true }`, matching upstream. Comment updated to explain the version-dependent
   history instead of the now-stale "path source is required" claim.
2. Upgraded the machine's global `uv` (`~/.local/bin/uv`, standalone install) from 0.8.24 to 0.12.19
   via the official installer script, so local `uv sync` / `process-cli`'s editable install keep
   working under the corrected config. `platform/python/uv.lock` did not need regenerating content
   (the lock doesn't encode which source-declaration shape was used, only resolved versions).
3. `.github/workflows/python-ci.yml`: added `frameworks/agent-framework/src` to the `mypy` path list
   and a `uv run pytest frameworks/agent-framework` step — it had never been part of the gate since
   its own introduction (M1, `d2b67f0`). mypy passed clean on first run (71 source files); `ruff
   format --check` found 4 files that had never been checked either (agent-framework's own code),
   fixed with a plain `ruff format .` — confirmed purely mechanical (magic-trailing-comma
   line-per-item explosion, one docstring/args wrap), no logic changes.
4. Verified every CI step locally end to end after the fix: `ruff check .` clean, `ruff format
   --check .` clean, `mypy` clean (71 files), and all nine `pytest` steps green (types 111,
   schema 65, action 80, filesystem 61, config 38, template 77, process 118, process-framework 385,
   agent-framework 8 passed + 3 skipped).

## Deferred / not done here

- Structural option F3 from M11 (moving the uv workspace root to the repo root, one lock/CI scope
  for `platform/` and `projects/`) is still not taken — `{ workspace = true }` alone was enough
  once uv's own resolution improved. Revisit only if another cross-boundary member hits a similar
  wall F3 would have prevented generically.
- CI still doesn't trigger on changes under `projects/products/process-os/**` (only
  `platform/python/**` and the workflow file itself) — a change to `process-cli` alone (no
  `platform/` touch) still won't run CI. Same gap M11 already flagged, still open.
- Pinning `setup-uv`'s version was considered and dropped: now that the config matches uv's own
  current/forward direction (upstream's shape), floating `latest` should stay correct going
  forward, unlike the workaround it replaced.
