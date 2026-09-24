# Import process-os's applications — Milestones 11 and 12 implementation plan

Last updated: 2026-09-25
tags: process-os, port, plan, import

> **For Claude:** M11 (`process-cli`) and M12 (`process-claude-plugin`) of
> `2026-09-25-process-os-port-completion-design.md`, done together as a **plain import plus a
> does-it-work check**. By the user's direction the lint / type / coverage polish and the CI wiring
> for these applications are **deliberately deferred**: once merged, the repo's own `process-cli`
> validates and refines them, and neither the process MCP nor the upstream-installed binary is needed.
> Everything measured while deciding what to defer is recorded below so it costs nothing to pick up.

**Goal:** bring the two applications of the `process-os` product into
`projects/products/process-os/`, verbatim wherever possible, and prove the CLI works.

**Sources:** `/Users/solo/Projects/workspaces/process-os/products/{process-cli,process-claude-plugin}`.

---

## M11 — `process-cli`

### Task 1: copy verbatim — DONE

`rsync -a` excluding caches. **171 files:** `src` 9 (5 modules plus a `tools/` data folder of 4 files),
`tests` 82, `fixtures` 78, `pyproject.toml`, `README.md`. `diff -r` against upstream was empty before any
edit; `git check-ignore` matched none of the 171 paths.

### Task 2: join the uv workspace — DONE, after a wrong first attempt

Adding `../../projects/products/process-os/*` to `platform/python/pyproject.toml`'s `members` made
`uv lock` **fail**: `process-framework references a workspace in tool.uv.sources … but is not a workspace
member`. uv treats a member that lives outside the workspace root as a standalone project when it reads
that member's own `pyproject.toml`, so `{ workspace = true }` cannot resolve.

**The earlier scratch "verification" was too narrow.** The design doc says a member outside the root
"locks and syncs as an editable install under a single `uv.lock`". That was true only for the dummy member
used, which had **no `workspace = true` dependency**. The real CLI does. Reproduced and fixes tried in
scratch workspaces that do have such a dependency:

| Option | Result | Cost |
|---|---|---|
| E0 — as designed | fails, same error | — |
| F1 — symlink the CLI into `platform/python/apps/` | lock + sync work | a git symlink, and it forces the deferred `apps-vs-applications-naming` decision |
| **F2 — path source in the CLI's `pyproject.toml`** | lock works (then confirmed for real: lock, sync, tests) | one line differs from upstream — **chosen** |
| F3 — uv workspace root moved to the repo root | lock + sync work, `workspace = true` untouched | structural: `pyproject.toml`/`uv.lock` move, CI working directory changes; **deferred**, see below |

Applied: in the copied `pyproject.toml`, `process-framework = { workspace = true }` became
`{ path = "../../../../platform/python/frameworks/process-framework", editable = true }`, with a comment
saying why. Result: `uv lock` 29 → **55 packages** (+26: `process-cli` itself plus 25 external packages, headed by `mcp`
**2.2.0** — the same version upstream's own lock pins, and what `>=1.12` resolves to today; members 8 → 9,
external 21 → 46), `uv sync` clean, and inside the workspace
env `process-cli` resolves to `platform/python/.venv/bin/process-cli` (`0.1.0 (source checkout)`).

### Task 3: prove it works — DONE

**As copied: 12 failed, 206 passed** (218 test cases; 74 was the count of test functions). All 12 failures
are the live-definition cases in `tests/main/demo/` (10) and `tests/main/demo-libraries/` (2): their
`REPO = Path(__file__).parents[5]` lands on `projects/products/`, so they look for
`projects/products/processos-workspace/definitions` (`FileNotFoundError`). That failure is the evidence the
dependency on M9's library is real.

**Deliberate re-point** (three files): `parents[5]` → `parents[7]` (this repo's root, counted from the new
path), and `processos-workspace/definitions` → `processos-workspace/libraries`, docstrings updated. Files:
`tests/main/demo/conftest.py`, `tests/main/demo-libraries/conftest.py`,
`tests/main/demo-libraries/test_digest.py`. **After: 218 passed in ~26 s**, including the demo tests that run
real `uv sync` / `uv add typer` / `uv run pytest` builds and the MCP tests against `mcp` 2.2.0.

**Smoke test against the installed upstream binary**, both pinned to this worktree's `processos.yaml`:
`list` (58 definitions), `list --scope sbx-sdlc-kit`, `list --long --scope std`,
`list records sbx-sdlc-kit/architecture/decision`, `show schema …decision` and `show action
sdlc.resolve-platform` — **6 of 6 outputs identical**; `check` agrees (`ok`; only difference is the printed
config path, relative to the working directory). A write into a readonly library scope is refused by the
ported CLI too (`readonly: scope 'std' is readonly`, scratch workspace).

**Net deviation from upstream, `diff -rq`:** 4 of 171 files — the CLI's `pyproject.toml` and the three tests
above. **167 byte-identical.**

**Existing CI is unaffected:** `ruff check .`, `ruff format --check .` and `mypy` over the 8 existing source
paths were re-run after the new member and its 25 new external packages: all clean. The CLI is simply not
part of CI yet.

## M12 — `process-claude-plugin`

### Task 4: copy verbatim — DONE

12 files (`.claude-plugin/plugin.json`, 5 commands, `README.md`, `skills/using-process-os/` with `SKILL.md`
and 4 `tools/*.md`). `diff -r` empty; none git-ignored.

### Task 5: keep it out of the uv workspace — DONE, negative control on the real repo

The plugin has no `pyproject.toml`. With the `../../projects/products/process-os/*` glob and no exclusion,
`uv lock` fails: `Workspace member … is missing a pyproject.toml (matches: ../../projects/products/process-os/*)`
— reproduced on the real repo before the fix. Fix: `exclude = ["../../projects/products/process-os/process-claude-plugin"]`
in `platform/python/pyproject.toml`. After it, `uv lock` is unchanged (0 diff lines) and `uv sync` is clean.

### Task 6: check it — DONE (structure only)

`plugin.json` is valid JSON (`process-claude-plugin` 0.2.0) and declares the stdio MCP server
(`mcpServers` → `process-cli mcp`); `SKILL.md` has its frontmatter; all 5 command files open with
frontmatter. **Not** registered or enabled anywhere: this repo has no `.claude-plugin/marketplace.json`.
(`.claude/settings.json` enables `process-claude-plugin@process-os` — upstream's own marketplace copy,
unrelated to this one.) Whether it *behaves* as a plugin was not tested; that needs activating it.

## Decision records — hand-written, unvalidated

Four records under `processos-workspace/records/sbx-sdlc-kit/architecture/decision/`, each headed by a
`# NOTE` saying so: `process-os-port-scope`, `mcp-transport`, `definitions-as-verbatim-library`,
`process-os-product-layout` (all `decided`). A plain-Python parse confirmed each has exactly the `decision`
schema's fields, in the same key order as the existing records, with a valid status and a string date. That
is **not** `process-cli` schema validation, which is deferred to after the merge.

---

## Deferred refinements — pick up after merge, with our own `process-cli`

Measured on a scratch copy of upstream's CLI with this repo's exact ruff/mypy config, so none of it needs
rediscovering.

1. **ruff on the CLI: 44 findings** — 29 `B905` (`zip()` without `strict=`; all in 25 test files, none in
   `src`; ruff's own unsafe fix inserts `strict=False`, which is the current default behavior), 10 `I001`
   (7 of them only because `process_cli` is not classified first-party under `--config`; consistent
   either way), 5 `E501`; 4 files need `ruff format`. Expect the README code fence to be reformatted too (as
   in M10).
2. **mypy strict on the CLI's `src`: 16 errors in 4 files.** 13 are `CallToolResult(isError=…,
   structuredContent=…)` — mypy not understanding pydantic aliases; `plugins = ["pydantic.mypy"]` in
   `[tool.mypy]` clears all 13 (tested). The other 3 are real: `printer.py:85` (`shown(str | int)`),
   `commands.py:105` (`content` reused as `str` and `dict`; annotate `str | dict[str, str]`), and
   `main.py:92` — `Printer.report(event: Event)` vs the framework's `Host.report(event: object)`, a
   contravariance violation that only works because the runner always passes an `Event`. Fix either by
   narrowing in `Printer` or by tightening `Host.report` to `Event` (touches the M10 framework).
3. **CI wiring for the CLI** — none done. Needed: the workflow's trigger `paths` add
   `projects/products/process-os/**` (today a change there does not start CI); ruff must run on the products
   path with `--config pyproject.toml` (by reasoning, not demonstrated here: outside `platform/python/` there is
   no `[tool.ruff]` above the files, so ruff would fall back to its defaults); `mypy_path` and the mypy command
   gain `../../projects/products/process-os/process-cli/src`; and one pytest step for the CLI.
4. **M10's coverage hypothesis** — `framework.py` is 41% covered by its own suite; the uncovered methods
   (`create`, `edit`, `show`, `write`, `remove`, …) match the CLI's fixtures folders one to one. Measure
   `process_framework` coverage under the CLI's suite (`coverage run --branch` via `uv run --with coverage`),
   including whether the `readonly` branch in `Catalog.writable` is reached. Behavior is already confirmed by
   hand (see Task 3); coverage is not.
5. **Structural option F3** — moving the uv workspace root to the repo root would restore
   `workspace = true` for the CLI and give one ruff/mypy config, one lock and one CI scope across
   `platform/` and `projects/`. A real decision (moves `pyproject.toml`/`uv.lock`, changes CI's working
   directory and the `.venv` location); not made here.
6. **Plugin** — not registered anywhere. Before registering: it has the same name as the upstream plugin
   already enabled here (`process-claude-plugin@process-os`), and its `tools/*.md` reference docs are static
   copies (the generating actions live in the `process-os` library and carry upstream's paths).
7. **Validate the four decision records** with `process-cli`, then refine the wording.

## Corrections to the design doc

`2026-09-25-process-os-port-completion-design.md` was **not** edited in this step (outside the approved
tree). Corrections to carry into it:

- **uv:** the "verified" outside-root member claim holds only for a member with no `workspace = true`
  dependency (see Task 2). A CLI-shaped member needs a path source, a symlink shim, or a repo-root workspace.
- **Counts are test cases, not functions:** framework 385 (not 79); CLI 218 (not 74).
- **Live-definition tests:** framework 1 parametrized case; CLI **12** cases (not "four tests").
- **Layout table / M11 row:** the CI step, `mypy_path` and ruff/mypy reach for the CLI were *not* done here
  (item 3 above); the `exclude` for the plugin arrived with M12, as planned.

## Retrospective note

- **A scratch check must copy every property of the real case that could matter.** The outside-root
  workspace test used a dummy member with no workspace dependency, so it "passed" and then failed on the real
  thing. It cost one failed `uv lock`, not a broken commit, only because the wiring was tried before committing.
- All commands ran through `uv --directory platform/python …` and `process-cli --config <this worktree>`;
  the process MCP was not used, and no `cd` into another repo was needed.
