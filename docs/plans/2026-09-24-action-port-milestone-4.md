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

### Task 1: copy `action` package source verbatim

```bash
SRC=/Users/solo/Projects/workspaces/process-os/packages/process-kit/action
DST=/Users/solo/Projects/workspaces/sdlc-kit/platform/python/packages/action
mkdir -p "$DST/src/process_kit/action/schemas"
cp "$SRC/pyproject.toml" "$SRC/README.md" "$DST/"
cp "$SRC"/src/process_kit/action/*.py "$DST"/src/process_kit/action/
cp "$SRC"/src/process_kit/action/schemas/*.yaml "$DST"/src/process_kit/action/schemas/
```

---

### Task 2: copy existing tests + fixtures verbatim

`tests/{conftest.py, action/{load/{test_valid,test_invalid,test_wrong_call}, references/test_references,
register/test_ids, run/{test_inputs,test_outputs,test_wrong_call}}, shell/execute/{test_check,test_log,
test_scripts,test_timeout,test_variables}}.py` + matching `fixtures/**/*.yaml`.

---

### Task 3: prove the port — run ported tests as-is

```bash
cd platform/python && uv sync
uv run --package process-kit-action pytest packages/action
```
Expected: same pass count as source, unmodified. `shell/execute/test_timeout.py` genuinely sleeps
(T1–T3 time out at 1s, T4 sleeps 1s) — expect this file alone to take a few real seconds, not a bug.

---

### Task 4: ruff + mypy, fix real findings — and *check* for a coverage gap (don't assume none)

```bash
cd platform/python
uv run ruff check packages/action
uv run ruff format --check packages/action
```
`mypy_path` needs `packages/action/src` added. Expect real findings similar in kind to Milestones 2–3
— `Executor`'s `Protocol` + `ShellExecutor`'s structural conformance, `Result`'s `StrEnum`, dataclass
defaults, subprocess typing (`subprocess.CompletedProcess` generics) are all plausible sources. Fix
each the same way: config or minimal type-precision change, never a behavior change, tests re-run
green after every fix.

After tooling is clean, read `action.py`/`shell.py` against the copied tests once more (the same way
`schema.py` was read against its tests before assuming coverage was fine) — if a real gap turns up,
close it the same way Milestone 3 did; if not, say so plainly rather than manufacturing tests for
their own sake.

---

### Task 5: prove whatever Task 4 added, full green

```bash
uv run --package process-kit-action pytest packages/action
uv run mypy packages/types/src packages/schema/src packages/action/src
```

---

### Task 6: wrap-up

- Update `.github/workflows/python-ci.yml`: add `packages/action` to the combined `mypy` call and its
  own `pytest` step.
- Confirm again: no new process-os type/schema/action/process modeled this round.
- **Rule-of-three check, for real this time**: three ports done (`types`, `schema`, `action`). Look at
  whether `process-os.create-package`/`port-package` (already read as prior art, `docs/research/
  python-implementation-quality-comparison.md` and this session's own exploration) is now a shape
  worth modeling as `sbx-sdlc-kit`'s own action — or whether the three ports still differ enough
  (verbatim copy vs. copy of package-embedded schema data vs. differing gap-closing effort) that it's
  still one more port away from being a real pattern. Bring the answer to the user before modeling
  anything, per the usual discussion-then-approval discipline — this task is "decide whether to ask,"
  not "build."
- Ask the user whether to `git commit`.
