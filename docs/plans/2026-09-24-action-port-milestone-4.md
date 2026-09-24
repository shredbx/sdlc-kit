# Port process-os's `action` package — Milestone 4 implementation plan

> **For Claude:** same footing as Milestones 2–3 — zero new process-os types/schemas/actions/processes
> modeled. This is port #3. Per the rule of three, once this lands, Task 6 looks seriously at whether
> a real `port-package` action/template is now earned — three real instances of the same shape.

**Goal:** port `packages/process-kit/action` (reads an action's folder, runs its shell scripts under
a fixed hook contract — `check.sh`/`pre.sh`/`action.sh`/`post.sh` — checking inputs/outputs against
their types) into `platform/python/packages/action`.

**Source:** `/Users/solo/Projects/workspaces/process-os/packages/process-kit/action/` — 313 lines
across 5 modules (`action.py`, `context.py`, `executor.py`, `result.py`, `shell.py`) + 7 embedded
schema files (`schemas/*.yaml`, shipped *inside* the package, loaded at runtime by `register()` —
these are package data, not process-os workspace definitions, and stay exactly where they are).
Depends on `process-kit-types` + `process-kit-schema` (both already ported) + `pyyaml`. Longest
source line is 159 chars — fits the existing `line-length = 160` with nothing to spare, no config
change needed.

**Different from Milestones 2–3, worth calling out up front**: this package actually runs
subprocesses (real `bash` scripts, including two `sleep`-based timeout tests) — its own test suite is
already unusually thorough (~80 cases: `action/load/{valid,invalid,wrong-call}`, `action/references`,
`action/register`, `action/run/{inputs,outputs,wrong-call}`, `shell/execute/{check,log,scripts,
timeout,variables}`), unlike `schema`'s real gaps. **No major coverage gap expected this time** —
Task 4 checks for one anyway, the same way every prior port has, but the honest expectation going in
is "port, prove, fix tooling," not "port and then write two dozen new tests."

---

### Task 1: copy `action` package source verbatim — DONE

Done via a whole-directory `rsync -a` (excluding `__pycache__`/`.pytest_cache`/`*.pyc`/`.gitkeep`)
rather than the hand-picked `cp` list originally drafted here — more robust against a layout this
package's `schemas/` subfolder already showed differs from `types`/`schema`. 35 files copied intact.
`action`'s own `pyproject.toml` was already shape-identical to `schema`'s (same `[project]` /
`[build-system]` / `[tool.hatch.build]` / `[tool.pytest.ini_options]` structure) — no alignment edit
needed. Modeling a reusable `copy-package-source` action was considered and explicitly deferred
(decided in conversation, not written up as a decision record — the shape is already fully known for
`process-kit`'s remaining packages; the real test of a generalized action is porting from a
*different* repo's structure, which hasn't happened yet).

---

### Task 2: copy existing tests + fixtures verbatim — DONE

Included in the same `rsync` as Task 1 (tests/fixtures live under the same package root) — 13 test
files + 13 fixture files, all verbatim.

---

### Task 3: prove the port — run ported tests as-is — DONE

`uv sync` + `uv run --package process-kit-action pytest packages/action` → **80 passed in 5.19s**,
unmodified. The `sleep`-based timeout tests accounted for the extra time as expected, not a bug.

---

### Task 4: ruff + mypy, fix real findings — and *check* for a coverage gap — DONE

**ruff**: 10 findings, all mechanical (`UP035` `Mapping` import location ×4, `I001` import sort ×3,
one `UP022` `capture_output` rewrite applied by hand after reading the exact call site rather than
trusting `--unsafe-fixes` blindly). All auto-fixed or manually verified equivalent; zero behavior
change.

**mypy** (`packages/action/src` added to root `pyproject.toml`'s `mypy_path`): 12 findings, all in
`shell.py` — `subprocess.CompletedProcess` needed its type argument (`[str]`, since the function
returns a *re-built* `CompletedProcess` with decoded `str` stdout/stderr, not the raw `bytes` one);
`**captured()` (a `dict[str, str]` unpacked into `Result(...)`, a dataclass with heterogeneous field
types) confused mypy's keyword-matching — fixed by giving `captured()` a precise `TypedDict` return
type instead of a bare `dict[str, str]`, so mypy matches its two keys exactly rather than checking
the value type against every remaining parameter; one bare `errors = []` needed
`errors: list[Error] = []`. Genuinely new pattern this port (not seen in `types`/`schema`): the
`TypedDict`-for-`**kwargs`-unpacking fix.

**Coverage check**: read `action.py`, `context.py`, `executor.py` in full against the fixtures.
Every branch in `Action.load()` (24 cases) and `Action.run()` — including the `stopped`/`skipped`
pass-through — traced to a real case; `check.yaml`'s C3/C4 confirmed even the pass-through runs
through the *same* `built.run(...)` fixture as every other test, not a shell-layer shortcut.
**No real gap found** — matches the plan's own stated expectation. Nothing added.

---

### Task 5: prove whatever Task 4 added, full green — DONE

`ruff check` clean, `ruff format --check` clean, `mypy packages/types/src packages/schema/src
packages/action/src` clean, `pytest packages/action` — 80/80 passing.

---

### Task 6: wrap-up — DONE

- `.github/workflows/python-ci.yml` updated: `packages/action/src` added to the combined `mypy` call,
  its own `pytest packages/action` step added.
- No new process-os type/schema/action/process modeled this round.
- **Rule-of-three verdict**: not yet, and the reason is more specific than "wait for a fourth."
  The copy step is genuinely mechanical and identical in shape across all three ports (proven again
  this round — a whole-directory copy is strictly better than any of the three hand-picked `cp` lists
  written so far). But the *fixing* work — the actual reason a port takes real effort — has been
  different every time: 4 mypy findings (types, pydantic `validate()` collision), 16 (schema, missing
  stubs + `Protocol`/`BaseModel` casting), 12 (action, `CompletedProcess` generics + a `TypedDict`
  fix never seen before). Modeling `port-package` now would template the 20% that's already trivial
  and leave the 80% that's real judgment untouched — not worth it yet. Decided in conversation (not
  written up as a decision record): defer a generalized copy action until porting from a genuinely
  *different* repo's structure, where the generalization would face real diversity instead of three
  copies of the same shape.
- No rule-like governance pattern surfaced this port worth a `guideline` — the new mypy pattern
  (`TypedDict` for `**dict` unpacked into a heterogeneous dataclass constructor) is a technical
  troubleshooting note, not a rule of conduct; it's captured in Task 4 above, not graduated further.
- Commit: pending user go-ahead.
