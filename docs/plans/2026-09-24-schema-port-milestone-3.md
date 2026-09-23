# Port process-os's `schema` package — Milestone 3 implementation plan

> **For Claude:** same footing as Milestone 2 — this port models zero new process-os
> types/schemas/actions/processes. This is port #2, though, so Task 9 revisits whether a real
> `port-package` pattern is visible yet across both ports (per the rule of three, two is still not
> three, but it's worth looking).

**Goal:** port `packages/process-kit/schema` (reads a definition — file, text, parsed data, or a
folder of files — and turns it into a usable `Type`) into `platform/python/packages/schema`, proven
the same way `types` was: verbatim first, tooling wired and clean, then real test gaps closed by
hand, all from actually reading the source.

**Source:** `/Users/solo/Projects/workspaces/process-os/packages/process-kit/schema/` — 4 source
files, depends on `process-kit-types` (already ported) + `pyyaml`. No workspace-root changes needed
this time — `platform/python/pyproject.toml`'s `members = ["packages/*"]` already covers a new
package the moment it exists, and ruff/mypy config is already shared at the root.

**Coverage gap found by reading `schema.py`/`definition.py`/`names.py` against the existing 9 test
files** (real, not assumed):
- `Schema.load()` (read from a file path) — never tested; only `Schema.loads()` (text) is.
- `Schema.load_all()` (read every `*.yaml` in a folder) — **entirely untested**: the dot-file/
  non-`.yaml` skip, name-order sorting, per-file error path prefixing, and duplicate-name-across-files
  detection all have zero coverage. This is the single biggest gap — more surface than `types` had.
- Non-mapping schema definitions (`string`/`integer`/`sequence`-based `Schema.loads()`) — every
  existing `loads` test uses a mapping; the other five YAML kinds are never loaded through `Schema`.
- The `fields:` dict-form of a mapping (named fields, as opposed to bare `keys`/`values`) — only its
  *conflict* error is tested (`fields` + `values` together); the happy path is never exercised.
- Several `_name()`/`build()` validation branches: missing `name`, a dotted `name`, a `name` that
  shadows a YAML type, an unknown `type:`, an unexpected property. None appear in the existing fixtures.

---

### Task 1: copy `schema` package source verbatim

```bash
SRC=/Users/solo/Projects/workspaces/process-os/packages/process-kit/schema
DST=/Users/solo/Projects/workspaces/sdlc-kit/platform/python/packages/schema
mkdir -p "$DST/src/process_kit/schema"
cp "$SRC/pyproject.toml" "$SRC/README.md" "$DST/"
cp "$SRC"/src/process_kit/schema/*.py "$DST"/src/process_kit/schema/
```
No changes to `pyproject.toml`'s `[tool.uv.sources] process-kit-types = { workspace = true }` — it
already resolves correctly once both packages sit under `platform/python/packages/`.

---

### Task 2: copy existing tests + fixtures verbatim

`tests/schema/{loads/{test_mapping_description,test_mapping_errors,test_mapping_references},
parse/{test_message,test_text,test_wrong_call}, qualify/{test_id,test_reference,test_wrong_call}}.py`
+ matching `fixtures/schema/**/*.yaml` (9 files each), plus `tests/conftest.py`.

---

### Task 3: prove the port — run ported tests as-is

```bash
cd platform/python && uv sync
uv run --package process-kit-schema pytest packages/schema
```
Expected: same pass count as source, unmodified.

---

### Task 4: ruff + mypy, fix real findings

```bash
cd platform/python
uv run ruff check packages/schema
uv run ruff format --check packages/schema
uv run mypy packages/schema/src
```
`mypy_path` needs `packages/schema/src` added alongside `packages/types/src` (both hold the
`process_kit` namespace) — otherwise the same "shadows library module" misread from Milestone 2
recurs. Any other real finding gets fixed directly, same discipline as Milestone 2 (mechanical
fixes only — import order, formatting, precise type annotations — never behavior changes).

---

### Task 5: close the gap — `Schema.load()` and `FileNotFoundError`

`fixtures/schema/load/from-file.yaml`:
```yaml
- {id: L1, namespace: notif, text: "{name: tag, type: string}", expect: notif.tag}
- {id: L2, namespace: "", text: "{name: tag, type: string}", expect: tag}
```
`tests/schema/load/test_from_file.py`:
```python
from process_kit.schema import Schema


def test_load_reads_the_schema_from_a_file(tmp_path, case):
    source = tmp_path / "definition.yaml"
    source.write_text(case["text"], encoding="utf-8")
    schema = Schema.load(source, case["namespace"])
    assert schema.name == case["expect"]
```

`fixtures/schema/load/missing-file.yaml`:
```yaml
- {id: M1}
```
`tests/schema/load/test_missing_file.py`:
```python
import pytest

from process_kit.schema import Schema


def test_load_raises_file_not_found_for_a_missing_file(tmp_path, case):
    with pytest.raises(FileNotFoundError):
        Schema.load(tmp_path / "does-not-exist.yaml")
```

---

### Task 6: close the gap — `Schema.load_all()`

`fixtures/schema/load_all/success.yaml`:
```yaml
- id: A1
  namespace: ""
  files: {a.yaml: "{name: one, type: string}", b.yaml: "{name: two, type: integer}"}
  expect: [one, two]
- id: A2
  namespace: notif
  files:
    a.yaml: "{name: one, type: string}"
    .hidden.yaml: "{name: bad, type: string}"
    readme.md: "not a definition, ignored by extension"
  expect: [notif.one]
```
`tests/schema/load_all/test_success.py`:
```python
from process_kit.schema import Schema


def test_load_all_reads_every_real_yaml_file_in_the_folder(tmp_path, case):
    for filename, text in case["files"].items():
        (tmp_path / filename).write_text(text, encoding="utf-8")
    result = Schema.load_all(tmp_path, case["namespace"])
    assert sorted(result.keys()) == sorted(case["expect"])
```

`fixtures/schema/load_all/errors.yaml`:
```yaml
- id: E1
  namespace: ""
  files: {a.yaml: "{name: one, type: string}", b.yaml: "{name: one, type: integer}"}
  expect: [{path: [b.yaml, name], code: invalid}]
- id: E2
  namespace: ""
  files: {a.yaml: "{name: one, type: bogus}"}
  expect: [{path: [a.yaml, type], code: unknown_type}]
```
`tests/schema/load_all/test_errors.py`:
```python
from process_kit.schema import Schema


def test_load_all_locates_every_error_by_the_file_it_came_from(tmp_path, case):
    for filename, text in case["files"].items():
        (tmp_path / filename).write_text(text, encoding="utf-8")
    errors = Schema.load_all(tmp_path, case["namespace"])
    assert [(error.path, error.code) for error in errors] == [(tuple(one["path"]), one["code"]) for one in case["expect"]]
    assert all(error.message for error in errors)
```
(`E1` relies on `load_all` processing files in name order — `a.yaml` first, so `b.yaml`'s repeat of
`one` is what's flagged, matching `definition.py`'s own logic read directly.)

---

### Task 7: close the gap — non-mapping definitions, `fields:` happy path, definition-level errors

`fixtures/schema/loads/non-mapping.yaml`:
```yaml
- {id: N1, namespace: notif, text: "{name: greeting, type: string, pattern: '^hi'}", value: "hi there", expect: []}
- {id: N2, namespace: notif, text: "{name: greeting, type: string, pattern: '^hi'}", value: bye, expect: [{path: [], code: pattern}]}
- {id: N3, namespace: notif, text: "{name: nums, type: sequence, items: integer}", value: [1, 2], expect: []}
- {id: N4, namespace: notif, text: "{name: nums, type: sequence, items: integer}", value: [1, "x"], expect: [{path: [1], code: wrong_type}]}
```
`tests/schema/loads/test_non_mapping.py`:
```python
from process_kit.schema import Schema
from process_kit.types import Types


def test_a_non_mapping_schema_loads_and_validates_like_its_base_type(case):
    schema = Schema.loads(case["text"], case["namespace"])
    errors = schema.validate(case["value"], (), Types())
    assert [(error.path, error.code) for error in errors] == [(tuple(one["path"]), one["code"]) for one in case["expect"]]
    assert all(error.message for error in errors)
```
(`N3`/`N4` also prove `items: integer` end-to-end through `Schema.loads()`'s own qualification step —
`integer` is a YAML type name, so `qualify()` leaves it bare and it resolves against the default
`Types()` registry.)

`fixtures/schema/loads/mapping-fields.yaml`:
```yaml
- id: F1
  namespace: notif
  text: |
    name: person
    type: mapping
    fields:
      age: {type: integer, required: true}
      nickname: {type: string}
  value: {age: 30, nickname: abc}
  expect: []
- id: F2
  namespace: notif
  text: |
    name: person
    type: mapping
    fields:
      age: {type: integer, required: true}
  value: {}
  expect: [{path: [age], code: missing}]
```
`tests/schema/loads/test_mapping_fields.py`: same body shape as `test_non_mapping.py` above (schema
loads, `.validate(case["value"], (), Types())`, compare `(path, code)` pairs).

`fixtures/schema/loads/definition-errors.yaml`:
```yaml
- {id: G1, text: "{type: string}", expect: [{path: [name], code: missing}]}
- {id: G2, text: "{name: a.b, type: string}", expect: [{path: [name], code: invalid}]}
- {id: G3, text: "{name: string, type: string}", expect: [{path: [name], code: invalid}]}
- {id: G4, text: "{name: x, type: bogus}", expect: [{path: [type], code: unknown_type}]}
- {id: G5, text: "{name: x, type: string, bogus: 1}", expect: [{path: [bogus], code: unexpected}]}
- {id: G6, text: "{name: x}", expect: [{path: [type], code: missing}]}
```
`tests/schema/loads/test_definition_errors.py`: same body shape as the existing
`test_mapping_errors.py` (`Schema.loads(case["text"])`, compare `(path, code)` pairs) — kept as a
separate file/fixture from `mapping-errors.yaml` since these branches apply to any YAML kind, not
mapping specifically (matching the existing one-concern-per-file convention).

Also **add three cases to the existing `fixtures/schema/loads/mapping-errors.yaml`** (field-definition
errors inside `fields:`, not yet covered):
```yaml
- {id: E6, text: "{name: tags, type: mapping, fields: {a: {type: string, name: x}}}", expect: [{path: [fields, a, name], code: unexpected}]}
- {id: E7, text: "{name: tags, type: mapping, fields: {a: 5}}", expect: [{path: [fields, a], code: wrong_type}]}
- {id: E8, text: "{name: tags, type: mapping, fields: [1, 2]}", expect: [{path: [fields], code: wrong_type}]}
```
No new test file needed — `test_mapping_errors.py` already iterates whatever cases the fixture holds.

---

### Task 8: prove the closed gap — DONE

```bash
cd platform/python
uv run --package process-kit-schema pytest packages/schema
uv run ruff check packages/schema
uv run mypy packages/schema/src
```
Confirmed 2026-09-24: 65 passed (43 ported + 22 new), ruff clean, mypy clean.

**Real findings fixed along the way** (config + minimal type-precision changes, zero behavior
changes — same discipline as Milestone 2):
- `mypy_path` needed `packages/schema/src` added (two packages now share the `process_kit`
  namespace).
- `types-PyYAML` added as a dev dependency — resolved the missing-stubs error and, as a side effect,
  the `SafeLoader` subclassing error too.
- `construct_mapping`'s override needed real parameter/return type annotations.
- `build()`'s final return relies on an invariant mypy can't see across branches (`name`/`built` are
  only `None` when `errors` is already non-empty) — documented with two `assert`s, the same idiom
  used for `mapping_type.py` in Milestone 2.
- `_text`/`_optional_text` needed a local variable (`value := data[key]`) so mypy's `isinstance`
  narrowing actually applies — narrowing doesn't reliably follow a repeated `dict[Any, Any]` subscript
  expression.
- `_build`'s `kind.model_validate(properties)` — `kind: type[Type]` is a `Protocol`, which doesn't
  declare `model_validate` (a pydantic `BaseModel` classmethod); fixed with a targeted
  `cast(Type, ...)` plus `# type: ignore[attr-defined]`, mirroring the `# type: ignore[override]` used
  for the analogous `validate` collision in Milestone 2.
- `_error`'s `problem` parameter was a bare `dict`; retyped as pydantic_core's own `ErrorDetails` —
  accurate, not just silenced.
- Six bare `dict` parameters retyped `dict[Any, Any]`, matching Milestone 2's convention.

**One fixture bug caught by the tests themselves, not assumed**: `N1`'s original pattern `^hi`
against value `"hi there"` failed — `string_type.py` validates with `re.fullmatch()`, not
`re.match()`, so `^hi` only ever matches the literal string `"hi"`. Fixed the fixture to `hi.*`,
not the source (the source's behavior was correct; the test's assumption about regex semantics
wasn't).

**Combined pytest across packages doesn't work — confirmed, not a bug to fix**: running
`uv run pytest packages/types packages/schema` in one call hits pytest collection errors (each
package's own `--import-mode=importlib` config only applies when that package is targeted on its
own). This is exactly why testing stays scoped per package — mypy tolerates a combined target list
fine (`uv run mypy packages/types/src packages/schema/src` works), pytest does not.

---

### Task 9: wrap-up — DONE

- **CI updated**: `mypy` now checks both `packages/types/src packages/schema/src` in one call (mypy
  handles a combined target list fine); `pytest` got its own separate step per package, since a
  combined pytest call across packages doesn't work (Task 8's finding).
- **Confirmed**: no new process-os type/schema/action/process modeled this round either.
- **What the two ports actually had in common**, now that there are two real data points instead of
  zero: copy source+tests+fixtures verbatim into `platform/python/packages/<name>/`; prove unmodified
  before changing anything; run ruff+mypy and fix only real findings (mechanical import/format fixes,
  plus the recurring pattern of a `Protocol`-typed `Type`/pydantic `BaseModel` seam needing a targeted
  `cast(...)`/`# type: ignore` rather than a source rewrite); close any real test-coverage gap found by
  reading the source directly, never assumed; add the package to CI. That is a real, repeatable shape
  — but per the rule of three, two instances is still not three. The call to actually model a
  `port-package`/`create-package` action and template from this shape is for whenever a *third*
  package gets ported, not this one.
- Ask the user whether to `git commit`.
