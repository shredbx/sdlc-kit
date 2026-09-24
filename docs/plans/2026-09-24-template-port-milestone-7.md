# Port process-os's `template` package — Milestone 7 implementation plan

> **For Claude:** same footing as Milestones 2–6 — zero new process-os types/schemas/actions/processes
> modeled. Port #6, third of the four remaining in the recommended order
> (`filesystem → config → template → process`).

**Goal:** port `packages/process-kit/template` (a folder of `template.yaml` + `files/`: renders
Jinja-templated files from data, and checks existing files still conform to a template's `pattern`)
into `platform/python/packages/template`.

**Source:** `/Users/solo/Projects/workspaces/process-os/packages/process-kit/template/` — 3 modules
(`template.py` 226 lines, `engine.py`, `source.py`) + 7 embedded schema files. Depends on
`process-kit-types` + `process-kit-schema` (both ported) + `jinja2`.

---

### Task 1–2: copy source + tests + fixtures verbatim — DONE

Whole-directory `rsync -a`, same method as Milestones 4–6. 32 files: `pyproject.toml`, `README.md`,
`src/process_kit/template/{__init__.py, engine.py, source.py, template.py, schemas/*.yaml}` (7 schema
files), 9 test files (`tests/template/{conform,load,references,register,render}/test_*.py`), 9
fixture files. `pyproject.toml` shape-identical to the other five ports — no alignment edit needed.

---

### Task 3: prove the port — run ported tests as-is — DONE

`uv sync` (pulled in `jinja2`+`markupsafe`) + `uv run --package process-kit-template pytest
packages/template` → **77 passed in 0.43s**, unmodified.

---

### Task 4: ruff + mypy, fix real findings — and *check* for a coverage gap — DONE

**ruff**: 5 findings, all mechanical (`I001` import-sort ×3, `UP035` `Mapping` import location ×1,
plus a cascaded fix once the first was applied). Auto-fixed; behavior unchanged, confirmed by
re-running tests.

**mypy** (`packages/template/src` added to root `pyproject.toml`'s `mypy_path`): **zero findings** —
third clean port in a row (after `filesystem`, `config`); only `action` has needed real fixes so far.

**Coverage check**: read `template.py` in full against its fixtures. `load/invalid.yaml`'s 24 cases
cover every schema/name/files-folder/jinja-syntax/pattern-syntax error path; `render/invalid.yaml`'s
12 cases cover `wrong_type`, `missing` (input validation), `undefined` (bad name and bad file
content), `invalid` (empty/absolute/escaping path, non-UTF-8 content), `duplicate`, and
`unknown_type`; `conform/invalid.yaml`'s 8 cases cover `missing`, `missing_line`, `wrong_type`,
`undefined` and `invalid` on the pattern side. **No real gap found.** Nothing added.

---

### Task 5: prove whatever Task 4 added, full green — DONE

`ruff check .` clean, `ruff format --check .` clean (117 files), `mypy packages/types/src
packages/schema/src packages/action/src packages/filesystem/src packages/config/src
packages/template/src` clean, `pytest packages/template` — 77/77 passing.

---

### Task 6: wrap-up — DONE

- `.github/workflows/python-ci.yml` updated: `packages/template/src` added to the combined `mypy`
  call, its own `pytest packages/template` step added.
- No new process-os type/schema/action/process modeled this round.
- No rule-like governance pattern surfaced worth a `guideline` this port.
- Next in line per the recommended order: `process` (types + schema + action deps, 644 src lines,
  the capstone — depends on everything ported so far).
- Commit: pending user go-ahead.
