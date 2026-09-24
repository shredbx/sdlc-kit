# Port process-os's `filesystem` package — Milestone 5 implementation plan

> **For Claude:** same footing as Milestones 2–4 — zero new process-os types/schemas/actions/processes
> modeled. This is port #4. Order chosen by dependency + size: `filesystem` has zero process-kit
> dependencies (76 src lines) and is the simplest package left, so it goes first of the four
> remaining (`filesystem` → `config` → `template` → `process`), same increasing-complexity shape as
> `types → schema → action`.

**Goal:** port `packages/process-kit/filesystem` (a `Folder` — a root path that refuses any path
outside it, and reads/writes/removes files inside it) into `platform/python/packages/filesystem`.

**Source:** `/Users/solo/Projects/workspaces/process-os/packages/process-kit/filesystem/` — 74 lines
in one module (`folder.py`). Zero process-kit dependencies (`dependencies = []` in its
`pyproject.toml`) — this is the one package in `process-kit` that any other package could depend on
without pulling in anything else.

---

### Task 1–2: copy source + tests + fixtures verbatim — DONE

Whole-directory `rsync -a` (excluding `__pycache__`/`.pytest_cache`/`*.pyc`/`.gitkeep`), same method
as Milestone 4. 25 files: `pyproject.toml`, `README.md`, `src/process_kit/filesystem/{__init__.py,
folder.py}`, 10 test files (`tests/folder/{exists,folder,inside,make,new,outside,path,read,remove,
write}/test_*.py`), 12 fixture files. `pyproject.toml` shape-identical to the other three ports — no
alignment edit needed.

---

### Task 3: prove the port — run ported tests as-is — DONE

`uv sync` + `uv run --package process-kit-filesystem pytest packages/filesystem` → **61 passed in
0.18s**, unmodified.

---

### Task 4: ruff + mypy, fix real findings — and *check* for a coverage gap — DONE

**ruff**: 4 findings, all mechanical — 3× `I001` import-sort in `tests/folder/{make,new,outside}/
test_wrong_call.py`, plus one formatting-only reflow of `README.md`'s embedded Python code fence
(trailing-comment alignment; ruff format's Markdown-code-block formatting, first time seen in this
series of ports — `types`/`schema`/`action`'s READMEs happened not to need it). All auto-fixed,
zero semantic change.

**mypy** (`packages/filesystem/src` added to root `pyproject.toml`'s `mypy_path`): **zero findings**
— the simplest port so far, and the first with nothing to fix.

**Coverage check**: read `folder.py` in full (9 methods: `__post_init__`, `path`, `inside`, `exists`,
`read`, `write`, `remove`, `make`, `folder`) against every fixture. Every raise branch traces to a
real case: `__post_init__`'s `TypeError`/`ValueError` → `new/wrong-call.yaml` K1–K3; `path()`'s
`TypeError`/outside-root/absolute-path branches → `outside/wrong-call.yaml` W1,W6,W7,W8; `write()`'s
own content-type `TypeError` → W9; `remove()`'s delegation to `path()` → W10–W11; `make()`'s
file-as-root case → `make/wrong-call.yaml` MK4. **No real gap found.** Nothing added.

---

### Task 5: prove whatever Task 4 added, full green — DONE

`ruff check .` clean, `ruff format --check .` clean (94 files), `mypy packages/types/src
packages/schema/src packages/action/src packages/filesystem/src` clean, `pytest packages/filesystem`
— 61/61 passing.

---

### Task 6: wrap-up — DONE

- `.github/workflows/python-ci.yml` updated: `packages/filesystem/src` added to the combined `mypy`
  call, its own `pytest packages/filesystem` step added.
- No new process-os type/schema/action/process modeled this round.
- No rule-like governance pattern surfaced worth a `guideline` — the Markdown-code-block ruff
  formatting behavior is a tooling note (captured in Task 4 above), not a rule of conduct.
- Next in line per the recommended order: `config` (types + schema deps, 107 src lines).
- Commit: pending user go-ahead.
