# Port process-os's `types` package — Milestone 2 implementation plan

> **For Claude:** this port introduces **zero** new process-os types/schemas/actions/processes —
> that's deliberate, per `docs/proposals/porting-and-modeling-process.md`'s "rule of three" (don't
> model a `port-package`/`create-package` action from a single example; this port is data point #1).
> Because nothing new is being *modeled*, `CLAUDE.md`'s per-definition-approval rule doesn't gate
> this the way Milestone 1 did — the fork-in-the-road decisions were already resolved in chat
> (namespace, test-gap, workspace-root). Once this plan itself is approved, Tasks 1–8 execute as one
> reviewed batch, checked at each step, not as individually-approved definitions.

**Goal:** get process-os's `packages/process-kit/types` package running, tested, linted, and
type-checked inside `sdlc-kit`'s own `platform/python/`, proving the port works before anything gets
built on top of it — per the already-approved house rule: "build to prove, then model."

**Source:** `/Users/solo/Projects/workspaces/process-os/packages/process-kit/types/` — 11 source
files, 412 lines, one dependency (`pydantic>=2`). No lint/type-checker/CI anywhere in the source repo
(confirmed, `docs/research/python-implementation-quality-comparison.md`).

**Decisions locked in this session:**
- **Namespace: unchanged.** Ported code stays `process_kit.types` — no `sbx_sdlc_kit` prefix. This
  repo's own identity is not restated inside its own paths ([[no-internal-identity-prefix]] in memory).
- **Test gap: closed now, by hand.** `string`/`integer`/`float`/`sequence` get the same
  fixture-driven test coverage `boolean`/`mapping` already have. This is a one-off manual pass, not
  the standing approach — the long-term direction is generating this from schemas once a real
  process for it is modeled (project memory: codegen-from-schema vision), but there's no generator
  yet and this is one data point, so it's done by hand this time.
- **`platform/python/` becomes a uv workspace root now**, ruff + mypy wired at the root from package
  #1, mirroring process-os's own proven `[tool.uv.workspace] members = [...]` pattern so every future
  package just joins `members`.
- **Testing stays scoped.** Every test run below targets `platform/python/packages/types` specifically
  — never a blind repo-wide `pytest`. Full-suite validation is reserved for CI (Task 8), which itself
  is path-filtered to `platform/python/**` only.

---

### Task 1: `platform/python/pyproject.toml` — workspace root + tooling

**Create:** `platform/python/pyproject.toml`

```toml
[tool.uv.workspace]
members = ["packages/*"]

[tool.ruff]
target-version = "py313"
line-length = 160

[tool.ruff.lint]
select = ["E", "F", "I", "UP", "B"]

[tool.mypy]
python_version = "3.13"
strict = true
```

`line-length = 160`, not a narrower default: the source's own longest line is 151 chars (checked
directly, not assumed) — this is a deliberate "one clause, one line" house style already in the code
being ported verbatim. Widening the config to fit it means the port doesn't get silently reformatted
in the same step as being proven to work; a narrower width is always a later, separate, on-purpose
choice, not a side effect of adding a linter.

---

### Task 2: port the `types` source verbatim

**Create:** `platform/python/packages/types/pyproject.toml` (same shape as source, path adjusted):
```toml
[project]
name = "process-kit-types"
version = "0.1.0"
description = "Types and errors: build a type in code and check a value with it."
requires-python = ">=3.13"
dependencies = ["pydantic>=2"]

[dependency-groups]
dev = ["pytest>=8", "pyyaml>=6"]

[build-system]
requires = ["hatchling"]
build-backend = "hatchling.build"

[tool.hatch.build.targets.wheel]
only-include = ["src/process_kit/types"]
sources = ["src"]

[tool.pytest.ini_options]
testpaths = ["tests"]
addopts = ["--import-mode=importlib"]
```

**Copy verbatim** (no changes — same namespace, same content):
- `src/process_kit/types/{__init__,type,error,field,lookup,yaml_types,boolean_type,string_type,integer_type,float_type,sequence_type,mapping_type}.py`
- `README.md`

Source: `process-os/packages/process-kit/types/src/process_kit/types/*.py` and `README.md`.

---

### Task 3: port existing tests + fixtures verbatim

**Copy verbatim:**
- `tests/conftest.py`
- `tests/boolean/validate/{test_build_valid,test_build_invalid,test_values}.py`
- `tests/mapping/validate/{test_build_invalid,test_values}.py`
- `tests/mapping/references/test_references.py`
- `tests/field/validate/{test_build_valid,test_build_invalid}.py`
- `tests/error/wrong_type/test_built.py`
- matching `fixtures/**/*.yaml` for every test above

Into `platform/python/packages/types/{tests,fixtures}/...`, same relative paths.

---

### Task 4: prove the port — run the ported tests as-is

```bash
cd platform/python && uv sync
uv run --package process-kit-types pytest packages/types
```
Expected: same pass count as the source repo, before anything new is added. This is the "build to
prove" checkpoint — if this doesn't pass unmodified, the port itself is broken and nothing past this
point should proceed.

---

### Task 5: run ruff + mypy against the ported code, fix any real findings

```bash
cd platform/python
uv run ruff check packages/types
uv run ruff format --check packages/types
uv run mypy packages/types
```
Expected, per the quality research's own prediction ("a `mypy --strict` pass is likely to succeed
with few changes — this is a good sign the discipline is real"): clean or near-clean. Any real finding
gets fixed in the ported source directly (not suppressed) before moving on — this is the actual
payoff of wiring the tooling at the point of porting rather than after.

---

### Task 6: close the test gap — `string`, `integer`, `float`, `sequence`

Same convention as `boolean`/`mapping`: `fixtures/<type>/validate/{build-valid,build-invalid,values}.yaml`
+ matching `tests/<type>/validate/test_*.py`, one shared-body test per file (see Task 3's ported files
for the exact pattern being followed). Drafted from reading the actual source
(`string_type.py`, `integer_type.py`, `float_type.py`, `sequence_type.py`) — every case below reflects
real validated behavior, not assumed behavior.

**`string`** — properties: `min_length`, `max_length`, `pattern` (validated as a real regex),
`enum`; `validate()` checks type, then `min_length`, `max_length`, `pattern`, `enum` in that order.

`fixtures/string/validate/build-valid.yaml`:
```yaml
- {id: S1, properties: {}, expect: built}
- {id: S2, properties: {min_length: 2}, expect: built}
- {id: S3, properties: {max_length: 10}, expect: built}
- {id: S4, properties: {pattern: "^[a-z]+$"}, expect: built}
- {id: S5, properties: {enum: [a, b, c]}, expect: built}
- id: S6
  properties: {min_length: 1, max_length: 5, pattern: "^[a-z]+$", enum: [ab, abc]}
  expect: built
```

`fixtures/string/validate/build-invalid.yaml`:
```yaml
- {id: S7, properties: {min_length: "two"}, expect: {raises: [min_length]}}
- {id: S8, properties: {max_length: "ten"}, expect: {raises: [max_length]}}
- {id: S9, properties: {enum: "a"}, expect: {raises: [enum]}}
- {id: S10, properties: {pattern: "("}, expect: {raises: [pattern]}}
- {id: S11, properties: {unexpected: 1}, expect: {raises: [unexpected]}}
```
(`S10` exercises `string_type.py`'s own custom validator that rejects a non-compiling regex.)

`fixtures/string/validate/values.yaml`:
```yaml
- {id: V1, properties: {}, value: hello, expect: []}
- {id: V2, properties: {}, value: 3, expect: [{path: [], code: wrong_type}]}
- {id: V3, properties: {}, value: true, expect: [{path: [], code: wrong_type}]}
- {id: V4, properties: {min_length: 3}, value: hi, expect: [{path: [], code: min_length}]}
- {id: V5, properties: {min_length: 3}, value: hey, expect: []}
- {id: V6, properties: {max_length: 3}, value: heyy, expect: [{path: [], code: max_length}]}
- {id: V7, properties: {max_length: 3}, value: hey, expect: []}
- {id: V8, properties: {pattern: "^[a-z]+$"}, value: Hello, expect: [{path: [], code: pattern}]}
- {id: V9, properties: {pattern: "^[a-z]+$"}, value: hello, expect: []}
- {id: V10, properties: {enum: [a, b]}, value: c, expect: [{path: [], code: enum}]}
- {id: V11, properties: {enum: [a, b]}, value: a, expect: []}
- id: V12
  properties: {min_length: 3, pattern: "^[a-z]+$", enum: [abcd]}
  value: AB
  expect: [{path: [], code: min_length}, {path: [], code: pattern}, {path: [], code: enum}]
- {id: V13, properties: {min_length: 2}, value: h, path: [name], expect: [{path: [name], code: min_length}]}
```

Test files (same shared-body pattern as `boolean`/`mapping`):
```python
# tests/string/validate/test_build_valid.py
from process_kit.types import StringType


def test_a_string_type_is_built_and_holds_the_properties_it_was_given(case):
    properties = case["properties"]
    string_type = StringType(**properties)
    for name, value in properties.items():
        assert getattr(string_type, name) == value
```
```python
# tests/string/validate/test_build_invalid.py
import pytest
from pydantic import ValidationError

from process_kit.types import StringType


def test_building_a_string_type_raises_and_names_each_wrong_property(case):
    with pytest.raises(ValidationError) as raised:
        StringType(**case["properties"])
    assert sorted(problem["loc"][0] for problem in raised.value.errors()) == sorted(case["expect"]["raises"])
```
```python
# tests/string/validate/test_values.py
from process_kit.types import StringType, Types


def test_a_value_gets_exactly_the_errors_in_the_case_and_nothing_else(case):
    errors = StringType(**case["properties"]).validate(case["value"], tuple(case.get("path", [])), Types())
    assert [(error.path, error.code) for error in errors] == [(tuple(one["path"]), one["code"]) for one in case["expect"]]
    assert all(error.message for error in errors)
```

**`integer`** — properties: `minimum`, `maximum` (both `StrictInt`); `validate()` rejects
non-`int` and rejects `bool` explicitly (`isinstance(value, bool)` check).

`fixtures/integer/validate/build-valid.yaml`:
```yaml
- {id: I1, properties: {}, expect: built}
- {id: I2, properties: {minimum: 1}, expect: built}
- {id: I3, properties: {maximum: 10}, expect: built}
- {id: I4, properties: {minimum: 1, maximum: 10}, expect: built}
```
`fixtures/integer/validate/build-invalid.yaml`:
```yaml
- {id: I5, properties: {minimum: "one"}, expect: {raises: [minimum]}}
- {id: I6, properties: {maximum: "ten"}, expect: {raises: [maximum]}}
- {id: I7, properties: {minimum: 1.5}, expect: {raises: [minimum]}}
- {id: I8, properties: {unexpected: 1}, expect: {raises: [unexpected]}}
```
`fixtures/integer/validate/values.yaml`:
```yaml
- {id: V1, properties: {}, value: 5, expect: []}
- {id: V2, properties: {}, value: "5", expect: [{path: [], code: wrong_type}]}
- {id: V3, properties: {}, value: true, expect: [{path: [], code: wrong_type}]}
- {id: V4, properties: {}, value: 5.5, expect: [{path: [], code: wrong_type}]}
- {id: V5, properties: {minimum: 3}, value: 2, expect: [{path: [], code: minimum}]}
- {id: V6, properties: {minimum: 3}, value: 3, expect: []}
- {id: V7, properties: {maximum: 3}, value: 4, expect: [{path: [], code: maximum}]}
- {id: V8, properties: {maximum: 3}, value: 3, expect: []}
- id: V9
  properties: {minimum: 3, maximum: 5}
  value: 1
  expect: [{path: [], code: minimum}]
- {id: V10, properties: {minimum: 2}, value: 1, path: [count], expect: [{path: [count], code: minimum}]}
```
Test files: identical shape to `string`'s three, importing `IntegerType`.

**`float`** — properties: `minimum`, `maximum` (both `StrictFloat`); `validate()` accepts `int` or
`float` but rejects `bool`. To avoid any ambiguity about whether `StrictFloat` accepts a plain `int`
at construction time, `build-valid`'s properties use real floats only (`1.5`, not `1`); `values.yaml`
passes an int (`3`) as the value under test, which is fine — `validate()` itself does the isinstance
check, unrelated to pydantic's strict construction.

`fixtures/float/validate/build-valid.yaml`:
```yaml
- {id: F1, properties: {}, expect: built}
- {id: F2, properties: {minimum: 1.5}, expect: built}
- {id: F3, properties: {maximum: 10.5}, expect: built}
- {id: F4, properties: {minimum: 1.0, maximum: 10.0}, expect: built}
```
`fixtures/float/validate/build-invalid.yaml`:
```yaml
- {id: F5, properties: {minimum: "one"}, expect: {raises: [minimum]}}
- {id: F6, properties: {maximum: "ten"}, expect: {raises: [maximum]}}
- {id: F7, properties: {unexpected: 1}, expect: {raises: [unexpected]}}
```
`fixtures/float/validate/values.yaml`:
```yaml
- {id: V1, properties: {}, value: 1.5, expect: []}
- {id: V2, properties: {}, value: 3, expect: []}
- {id: V3, properties: {}, value: "3", expect: [{path: [], code: wrong_type}]}
- {id: V4, properties: {}, value: true, expect: [{path: [], code: wrong_type}]}
- {id: V5, properties: {minimum: 2.0}, value: 1.5, expect: [{path: [], code: minimum}]}
- {id: V6, properties: {minimum: 2.0}, value: 2, expect: []}
- {id: V7, properties: {maximum: 2.0}, value: 2.5, expect: [{path: [], code: maximum}]}
- {id: V8, properties: {maximum: 2.0}, value: 2, expect: []}
- id: V9
  properties: {minimum: 2.0, maximum: 5.0}
  value: 1
  expect: [{path: [], code: minimum}]
- {id: V10, properties: {minimum: 1.0}, value: 0.5, path: [score], expect: [{path: [score], code: minimum}]}
```
Test files: identical shape, importing `FloatType`.

**`sequence`** — bare `items: StrictStr | None`, a type *name* looked up in `Types` at validate
time; `validate()` rejects non-`list`, resolves `items` via `types.get()` (an unregistered name is an
`unknown_type` error at the sequence's own path, not the item's), and validates each item at its own
index appended to `path`.

`fixtures/sequence/validate/build-valid.yaml`:
```yaml
- {id: Q1, properties: {}, expect: built}
- {id: Q2, properties: {items: string}, expect: built}
- {id: Q3, properties: {items: integer}, expect: built}
```
`fixtures/sequence/validate/build-invalid.yaml`:
```yaml
- {id: Q4, properties: {items: 1}, expect: {raises: [items]}}
- {id: Q5, properties: {unexpected: 1}, expect: {raises: [unexpected]}}
```
`fixtures/sequence/validate/values.yaml`:
```yaml
- {id: V1, properties: {}, value: [], expect: []}
- {id: V2, properties: {}, value: [1, "a", true], expect: []}
- {id: V3, properties: {}, value: "not-a-list", expect: [{path: [], code: wrong_type}]}
- {id: V4, properties: {items: integer}, value: [1, 2, 3], expect: []}
- {id: V5, properties: {items: integer}, value: [1, "x", 3], expect: [{path: [1], code: wrong_type}]}
- id: V6
  properties: {items: integer}
  value: [1, "x", "y"]
  expect: [{path: [1], code: wrong_type}, {path: [2], code: wrong_type}]
- {id: V7, properties: {items: nothing}, value: [1], expect: [{path: [], code: unknown_type}]}
- id: V8
  properties: {items: integer}
  value: ["x"]
  path: [tags]
  expect: [{path: [tags, 0], code: wrong_type}]
```
Test files: identical shape, importing `SequenceType`. (`SequenceType.references()` is a trivial
one-liner already exercised indirectly by `V7`'s `unknown_type` path; a dedicated `references` test
suite — the way `mapping` has one — is left out here as disproportionate to what the research
actually flagged as the gap. Worth adding later if it turns out to matter.)

---

### Task 7: prove the closed gap

```bash
cd platform/python
uv run --package process-kit-types pytest packages/types
uv run ruff check packages/types
uv run mypy packages/types
```
Expected: every test green (ported + new), ruff and mypy both clean.

---

### Task 8: CI — scoped to `platform/python`, full validation only on that gate

**Create:** `.github/workflows/python-ci.yml`

```yaml
name: python-ci

on:
  push:
    paths: ["platform/python/**", ".github/workflows/python-ci.yml"]
  pull_request:
    paths: ["platform/python/**", ".github/workflows/python-ci.yml"]

jobs:
  test:
    runs-on: ubuntu-latest
    defaults:
      run:
        working-directory: platform/python
    steps:
      - uses: actions/checkout@v4
      - uses: astral-sh/setup-uv@v3
      - run: uv sync
      - run: uv run ruff check .
      - run: uv run ruff format --check .
      - run: uv run mypy .
      - run: uv run pytest
```

`paths:`-filtered so this never fires on unrelated changes, and running the *full* `platform/python`
suite here is intentional — CI is exactly the "explicit gate" [[scoped-testing-not-full-suite]] carves
out as the place full validation belongs, as opposed to every local edit.

---

### Task 9: wrap-up

- Confirm explicitly: **no new process-os type/schema/action/process was modeled this milestone** —
  by design, per the rule of three. Worth revisiting `process-os.create-package`'s own shape (already
  read as prior art) once a *second* package is ported and a real pattern exists to generalize from.
- Ask the user whether to `git commit` this milestone.
- Next in line after this: decide with the user what the second package/port is — that's what would
  actually earn a `port-package` action/schema, and what resolves the still-deferred
  `platform-position-in-namespace` decision with two real data points instead of zero.
