# Port process-os's `process` package — Milestone 8 implementation plan

> **For Claude:** same footing as Milestones 2–7 — zero new process-os types/schemas/actions/processes
> modeled. Port #7, the capstone of the recommended order
> (`filesystem → config → template → process`) — depends on `types`, `schema` and `action`, all
> already ported, and its own test suite exercises every one of them end to end (real actions,
> real `ShellExecutor`, nested processes, resume).

**Goal:** port `packages/process-kit/process` (reads a process's `<name>.yaml`, checks its whole
graph before anything runs, then walks it one node at a time so a run can resume) into
`platform/python/packages/process`.

**Source:** `/Users/solo/Projects/workspaces/process-os/packages/process-kit/process/` — 8 modules
(`process.py`, `graph.py`, `resolver.py`, `run.py`, `runner.py`, `steps.py`, `store.py`, `__init__.py`
— 644 lines total, the largest port so far) + 2 embedded schema files. Depends on
`process-kit-types` + `process-kit-schema` + `process-kit-action` (all three already ported).

---

### Task 1–2: copy source + tests + fixtures verbatim — DONE

Whole-directory `rsync -a`, same method as Milestones 4–7. 51 files: `pyproject.toml`, `README.md`,
8 source files + 2 schema files, 20 test files (`tests/process/{check,data,load,of_action,references,
register,run,step}/...`), 19 fixture files. `pyproject.toml` shape-identical to `action`'s (three
workspace deps) — no alignment edit needed.

---

### Task 3: prove the port — run ported tests as-is — DONE

`uv sync` + `uv run --package process-kit-process pytest packages/process` → **118 passed in 1.77s**,
unmodified — the largest test count of any port so far.

---

### Task 4: ruff + mypy, fix real findings — and *check* for a coverage gap — DONE

**ruff**: 19 findings — 14 mechanical (`I001` import-sort, `UP035` `Mapping` import location, `F401`
unused import), auto-fixed; **5 genuinely new `E501`** (lines over `line-length = 160`, first time
this limit was actually exceeded across all seven ports) — resolved not by hand-wrapping but by
running `ruff format`, which itself breaks a long function signature across lines once it would
overflow the configured limit (confirmed via `--diff` before applying). Zero semantic change,
confirmed by re-running the 118 tests after.

**mypy** (`packages/process/src` added to root `pyproject.toml`'s `mypy_path`): **4 real findings**,
all genuinely new patterns:
- `steps.py`: three bare `dict` parameters (`_stop`, `_switch`, `_unexpected`) needed
  `dict[str, Any]` — mypy strict mode requires type arguments even where the dict only flows through
  to more `dict`/`Any`-typed calls.
- `runner.py:43`: `Runner.resume`'s optional `process` parameter (typed `Process | None`) was
  reassigned the resolver's wider `Action | Process | list[Error]` return before narrowing it —
  mypy correctly rejects narrowing a parameter's own declared type with a wider union. Fixed by
  resolving into a new local (`node`), narrowing that, then assigning the already-narrowed value to
  `process` — same runtime behavior (confirmed by `wrong-process.yaml`'s W2/W3 cases, which exercise
  exactly this raise's two branches), cleaner for the type checker.

**Coverage check**: read all 8 source modules in full against the fixtures. `graph.py`'s every error
code (`unknown_id`, `missing`, `type_mismatch`, `not_produced`, `unknown_field`, `invalid`,
`unhandled`, `unreachable`, `cycle`, `duplicate`) traces to a real case in `check/invalid.yaml`'s 18
cases; `steps.py`'s `StepType` validation traces to `step/validate.yaml`'s 22 cases (`wrong_type`,
`pattern`, `invalid`, `missing`, `min_length`, `unexpected`); `runner.py`'s full lifecycle — nested
processes, resume, skip propagation via `check.sh`'s `SKIP` exit code, switch branching, both of
`resume()`'s wrong-call raises — traces to `run/started.yaml`'s 13 cases plus `run/wrong-call.yaml`
and `run/wrong-process.yaml`. `resolver.py` and `store.py` are thin `Protocol`s with no branches of
their own. **No real gap found.** Nothing added.

---

### Task 5: prove whatever Task 4 added, full green — DONE

`ruff check .` clean, `ruff format --check .` clean (146 files), `mypy packages/types/src
packages/schema/src packages/action/src packages/filesystem/src packages/config/src
packages/template/src packages/process/src` clean, `pytest packages/process` — 118/118 passing.

A combined `pytest packages/` across all seven ported packages was tried once, as a capstone
sanity check now that every `process-kit` package is ported — it fails collection (44 errors) because
several packages share identical relative test paths (e.g. `tests/mapping/validate/...`) under one
collection root. This is expected, not a regression: it is exactly why this repo's CI has always run
`pytest` once per package rather than combined (confirmed: every package already passes on its own).
No fix applied — this confirms the existing per-package CI structure is correct, not a gap in it.

---

### Task 6: wrap-up — DONE

- `.github/workflows/python-ci.yml` updated: `packages/process/src` added to the combined `mypy`
  call, its own `pytest packages/process` step added. All seven `process-kit` packages are now
  ported and wired into CI.
- No new process-os type/schema/action/process modeled this round.
- No rule-like governance pattern surfaced worth a `guideline` — the `E501`-resolved-by-`ruff format`
  and the `Process | None` parameter-narrowing patterns are tooling/typing notes (captured above),
  not rules of conduct.
- All four packages named in the recommended order (`filesystem → config → template → process`) are
  now ported. `process-kit` in full is ported into `platform/python`.
- Commit: pending user go-ahead.
