# Port process-os's `config` package — Milestone 6 implementation plan

> **For Claude:** same footing as Milestones 2–5 — zero new process-os types/schemas/actions/processes
> modeled. Port #5, second of the four remaining in the recommended order
> (`filesystem → config → template → process`), chosen by dependency + size.

**Goal:** port `packages/process-kit/config` (reads `processos.yaml`, checks it against its own
shipped schema, resolves every path in it to absolute) into `platform/python/packages/config`.

**Source:** `/Users/solo/Projects/workspaces/process-os/packages/process-kit/config/` — one module
(`config.py`, 104 lines) + 9 embedded schema files (`schemas/*.yaml`, package data loaded by
`register()`, same pattern as `action`'s). Depends on `process-kit-types` + `process-kit-schema`
(both already ported).

---

### Task 1–2: copy source + tests + fixtures verbatim — DONE

Whole-directory `rsync -a`, same method as Milestones 4–5. 20 files: `pyproject.toml`, `README.md`,
`src/process_kit/config/{__init__.py, config.py, schemas/*.yaml}` (9 schema files), 4 test files
(`tests/config/{load,register}/test_*.py`), 4 fixture files. `pyproject.toml` shape-identical to the
other four ports — no alignment edit needed.

---

### Task 3: prove the port — run ported tests as-is — DONE

`uv sync` + `uv run --package process-kit-config pytest packages/config` → **38 passed in 0.35s**,
unmodified.

---

### Task 4: ruff + mypy, fix real findings — and *check* for a coverage gap — DONE

**ruff**: 1 finding — `I001` import-sort in `tests/config/load/test_wrong_call.py`, same mechanical
pattern as every prior port. Auto-fixed; `ruff format` also reflowed 2 files (no semantic change,
confirmed by re-running tests after).

**mypy** (`packages/config/src` added to root `pyproject.toml`'s `mypy_path`): **zero findings** —
second port in a row with nothing to fix (after `filesystem`).

**Coverage check**: read `config.py` in full against its fixtures. `load/invalid.yaml`'s 24 cases
(I1–I24) cover every schema-validation code path the file can produce (`wrong_type`, `invalid`,
`missing`, `unexpected`, `min_length`, `not_found`, `outside_root`, `enum`, `pattern`, `minimum`);
`load/valid.yaml` covers path resolution (relative root, nested config file, custom `records`/
`output`/`runs`); `load/wrong-call.yaml`'s W1–W2 cover the two raises from `file.read_text()`
(`FileNotFoundError`, `IsADirectoryError`). **No real gap found.** Nothing added.

---

### Task 5: prove whatever Task 4 added, full green — DONE

`ruff check .` clean, `ruff format --check .` clean (102 files), `mypy packages/types/src
packages/schema/src packages/action/src packages/filesystem/src packages/config/src` clean,
`pytest packages/config` — 38/38 passing.

---

### Task 6: wrap-up — DONE

- `.github/workflows/python-ci.yml` updated: `packages/config/src` added to the combined `mypy`
  call, its own `pytest packages/config` step added.
- No new process-os type/schema/action/process modeled this round.
- No rule-like governance pattern surfaced worth a `guideline` this port.
- Next in line per the recommended order: `template` (types + schema + jinja2 deps, 259 src lines).
- Commit: pending user go-ahead.
