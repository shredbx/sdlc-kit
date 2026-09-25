# Port process-os's `process-framework` — Milestone 10 implementation plan

> **For Claude:** second milestone of `2026-09-25-process-os-port-completion-design.md` (M9–M12), and
> the first Python port since M8. Zero new process-os types/schemas/actions/processes modeled. It is
> also the first port that lands outside `platform/python/packages/`: at
> `platform/python/frameworks/process-framework` (the user's call, mirroring upstream's tier). It
> depends on the seven kit packages (all ported) and, for one test case, on M9's definitions library.

**Goal:** port `frameworks/process-framework` (the config, the definitions and the runtime joined into
one `ProcessFramework`, entered through `initialize`) into `platform/python/frameworks/process-framework`.

**Source:** `/Users/solo/Projects/workspaces/process-os/frameworks/process-framework/` — 175 files: 11
in `src/` (7 Python modules plus 4 embedded schema files), 82 in `tests/`, 80 in `fixtures/`, plus
`pyproject.toml` and `README.md`. 1,523 source lines. Depends on `process-kit-{types,schema,config,
action,filesystem,template,process}`, whose distribution names match ours exactly, so its
`pyproject.toml` needed no alignment edit.

---

### Task 1: copy source + tests + fixtures verbatim — DONE

`rsync -a`, excluding `__pycache__`, `.pytest_cache`, `*.pyc`, `.venv`, `.mypy_cache`, `.ruff_cache`.
175 files: `src` 11, `tests` 82, `fixtures` 80, top level 2. `diff -r` against upstream was empty
**before any edit**; `git check-ignore` matched none of the 175 paths.

### Task 2: wire the workspace — DONE

`platform/python/pyproject.toml`: `members` gains `frameworks/*`; `mypy_path` gains
`frameworks/process-framework/src`. `uv lock`: +40 lines, exactly one new workspace member, no external
dependency changes. `uv sync` clean.

### Task 3: prove the port — run the copied tests as-is, then re-point — DONE

As copied: **384 passed, 1 error.** The error is the one live-definition case,
`tests/mcp/schema/test_demo.py::test_input_schema_against_the_real_demo[D1]`: its `catalog` fixture
computes `REPO = Path(__file__).parents[5]`, which at the new depth is `platform/python/`, so it looked
for `platform/python/processos-workspace/definitions` (`FileNotFoundError`). That failure is the
evidence the dependency is real.

Deliberate re-point in `tests/mcp/schema/conftest.py` (the only framework file with a `REPO` line):
`parents[5]` → `parents[7]` (this repo's root, counted from the new path), and
`processos-workspace/definitions` → `processos-workspace/libraries` (M9's verbatim copy), with the
docstring updated to say so. After: **385 passed.**

**Count correction to the design doc:** "79 tests" counted `def test_` functions; parametrization makes
it 385 test cases. And the live-definition dependency is one parametrized case, not "one test file"
worth of tests.

### Task 4: ruff + mypy, fix real findings — DONE

**ruff (before):** 44 findings — 37 `I001` (import order), 4 `E501` (over `line-length = 160`), 3
`UP035` (deprecated import) — 40 auto-fixable; and 9 files needing `ruff format`. All in the framework;
the kit packages stayed clean (0 changes under `packages/`).

**mypy (before):** 12 errors in 4 files. Fixed by hand, each chosen to keep runtime behavior identical
except where noted:

| Site | Finding | Fix | Runtime effect |
|---|---|---|---|
| `runtime.py` ×2 (4 errors) | `file` was bound as a `Path` earlier in `gather`, then re-bound to `Records.file()`'s `str \| None` | new local `located`, plus `assert located is not None` | none on any reachable path. `Records.file` returns `None` only when the folder leaves the records folder, which `Records.read` already reports as an `outside_root` error just before — so it is not `None` there. The assert makes that invariant explicit. |
| `mcp.py:60` | `types.get()` returns `Type \| None` into `_map(kind: Type, …)` | `_map`'s parameter widened to `Type \| None` | none: `_map` already ends `raise TypeError("not a YAML type: …")`, so `None` already raised there |
| `catalog.py:157` | early-return conditions left `folder` typed `Path \| None` | reordered into nested `if folder is None: … ` so each branch narrows | none: truth table of `(folder, file)` × {None, set} checked for all four cases — same result in each |
| `catalog.py:194` | `locate(...) or locate(...)` is `Path \| None` | local `located` + `assert located is not None` (a name from `types.known` has a type or schema file) | none on reachable paths; an impossible state now fails where it arises |
| `catalog.py:279` | `checked` needed an annotation | `graph = []` / `checked: set[str] = set()` (matches `check`'s `set[str] \| None`) | none |
| `catalog.py:389` | bare `tuple` in `_calls(steps: tuple)` | `tuple[Step, ...]`; `Step` is a public export of `process_kit.process` | none |
| `framework.py` ×3 | `ProcessFramework.list` (a method) shadows the builtin inside the class body, so annotations `list[...]` written after it resolve to the method | `import builtins`; three annotations spelled `builtins.list[...]` | none: the file has `from __future__ import annotations`, which is why it ran at all |

**ruff (after the hand fixes):** 47 findings (three more, from the edits above), `--fix` cleared 43,
and `ruff format` reformatted 9 files and cleared the remaining 4 `E501` by wrapping. Run scoped to
`frameworks/process-framework` only.

**One non-obvious one: `README.md`.** `ruff format` (0.16.8) also formats Python code fences inside
Markdown. It changed only comment spacing in one code block (two spaces before `#`). Confirmed by
running `ruff format --diff` on a scratch copy of the untouched upstream README: identical diff. CI's
`ruff format --check .` would have failed on the original file.

**Net deviation from upstream, measured with `diff -rq`:** 45 of 175 files differ — 6 in `src/`
(`__init__`, `catalog`, `entry`, `framework`, `mcp`, `runtime`; `host.py` is untouched), 38 in `tests/`
(ruff import order/format, plus the one deliberate `conftest.py` edit), and `README.md`. **130 are
byte-identical.** No file exists in the copy that is not upstream.

### Task 5: check for a coverage gap — DONE (measured, not read)

Earlier ports read the source against the fixtures. Here it was measured: `coverage run --branch` in an
ephemeral `uv run --with coverage` environment (no dependency added to the project; data file in the
scratchpad). 385 tests, **74% total** (1,079 statements, 251 missed; 538 branches, 23 partial):

| Module | Stmts | Missed | Cover |
|---|---|---|---|
| `__init__.py`, `entry.py`, `host.py` | 71 | 0 | 100% |
| `catalog.py` | 359 | 21 | 93% |
| `mcp.py` | 80 | 4 | 92% |
| `runtime.py` | 188 | 22 | 88% |
| **`framework.py`** | 381 | **204** | **41%** |

`framework.py`'s misses are contiguous and name-able: the **authoring / CRUD surface** — `create`,
`edit`, `_cast_record`, `show`, `write`, `remove`, `_write_action`, `_verify_action`, `_verify`,
`description`, `list_records`, `_walk_action`. What its own suite does cover: `check`, `validate`,
`render`, `conform`, `run`, `resume`, `list_runs`, `list`, `tools`, `call`.

**Hypothesis, not a finding:** process-cli's suite exercises these through the framework — its
fixtures folders include `create/`, `edit/`, `show/`, `write/`, `remove/` and `list/records.yaml`,
which match this list one to one. **Unverified until M11**, where `process_framework` coverage will be
measured under the CLI's suite; if those methods are still uncovered then, that is a real gap and gets
tests. "No real gap found" is deliberately *not* claimed here.

Smaller misses: `Catalog.writable` (`catalog.py:126-139`) — including the `readonly` enforcement at
137-138 that M9 relied on and exercised by hand in a scratch workspace, which this suite never reaches;
`runtime.py` records-given-as-a-file-path (69-73) and `cast_value`'s sequence/float cases (163-178);
`catalog.py` digest with type errors (188), reached-set dedupe (226), `_calls`' `Switch` branch
(399-401); and two defensive `RuntimeError` paths for the package's own shipped schemas being wrong
(476, 491), unreachable without corrupting the package. Same downstream-verification note applies to
all but the last.

### Task 6: final gate — DONE

Run in CI order, from `platform/python`, after the CI edit below:

| Command | Result |
|---|---|
| `uv sync` (as CI runs it, not `--all-packages`) | `Audited 28 packages` |
| `ruff check .` | All checks passed |
| `ruff format --check .` | 236 files already formatted |
| `mypy` over 8 src paths | Success: no issues found in 45 source files |
| `pytest frameworks/process-framework` | 385 passed |
| `process-cli --config <this worktree>/processos.yaml check` | `processos.yaml: ok` |

Only the framework's own suite was run; the seven kit packages are untouched (0 changes under
`packages/`), so their suites were not re-run — the scoped-testing rule.

### Task 7: CI — DONE

`.github/workflows/python-ci.yml`: `frameworks/process-framework/src` added to the `mypy` line, and one
new step, `uv run pytest frameworks/process-framework`, after the `packages/process` step. `ruff check .`
and `ruff format --check .` already cover the new folder (they run on `.`).

---

## Notes for the next milestones

- **M11 must measure, not assume:** `process_framework` coverage under process-cli's suite (Task 5's
  hypothesis), including whether the `readonly` branch in `Catalog.writable` is reached.
- The design doc's Inventory row and M10 details carry the two count corrections above (79 functions /
  385 cases; one parametrized case, not a test file). **Not edited in this step** — it was outside the
  approved file tree; proposed as a follow-up.
- The same functions-vs-cases distinction likely applies to process-cli's "74 tests"; recount in M11.

## Retrospective note (process-os tooling / working method)

- All commands ran through `process-cli` (config pinned with `--config`) and `uv --directory
  platform/python …`; the process MCP was not used (its catalog is a startup snapshot — see the M9 plan).
- Shell working directory persists between Bash calls: twice this session a `cd` into the upstream
  process-os checkout leaked into the following commands. Both were read-only listings — nothing was
  written to process-os — but it is the same wrong-folder hazard the MCP finding was about. From here
  on: `uv --directory`, and paths relative to the worktree, with no `cd` into another repo.
