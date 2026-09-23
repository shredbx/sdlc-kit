# process-kit-types

Types and errors. Build a type in code, check a value with it, and get back every problem with its place. No files and no schema — this package is what `process-kit-schema` is built on, and it can be used without it.

## Install

Not installed standalone: pulled in as a dependency by every other `process-kit` package
(`uses: [process-kit-types]`, from its own `package.yaml`). From a checkout of this repository,
`uv sync --all-packages --group dev` builds it into the workspace for its own tests.

## Usage

```python
from process_kit.types import Field, MappingType, StringType, Types

types = Types()
types.add("email", StringType(pattern="[^@]+@[^@]+"))
types.add("contact", MappingType(fields=[Field(name="email", type="email", required=True)]))

for error in types.validate({"email": "nope", "phone": "555"}, "contact"):
    print(error.path, error.code, error.message)
```

```text
('phone',) unexpected "phone" is not a declared field
('email',) pattern must match [^@]+@[^@]+, got 'nope'
```

## What it holds

| | |
|---|---|
| `Type` | the protocol: `validate(value, path, types)` returns the list of `Error`, empty when the value is fine, and `references()` says which type names it refers to. A class is a `Type` by having both; it inherits nothing |
| `StringType` `IntegerType` `FloatType` `BooleanType` `SequenceType` `MappingType` | the types YAML itself has. Every property is optional and narrows the type. `YAML_TYPES` maps their names to the classes. A `MappingType` is bare (any mapping), has `fields` (only those keys), or has `keys` and `values`, each the name of a type, for free keys checked one by one. `fields` never comes with the other two |
| `Field` | one named entry of a mapping: `name`, `type` (the *name* of its type, looked up when it validates), `required` (`False` by default) and `description` (or `None`) |
| `Types` | every type by name. Starts with the six above. `add(name, type)`, `get(name)`, `validate(data, name, path=())`, and `check()`, which finds every name a held type refers to that nothing holds |
| `Error` | `path` (a tuple of keys and item positions), `code`, `message` |
| `wrong_type(path, expected, value)` | the `Error` a type gives for a value of the wrong kind, code `wrong_type`, so a type written in code says it as the six do |

## Two rules

- **Wrong data is returned as errors.** `validate` never raises.
- **A wrong type definition raises.** A type checks its own properties when it is built, so a
  type with a `min_length` of `"two"` or a `pattern` that is not a regular expression cannot exist.

```python
types.add("counts", MappingType(keys="string", values="integer"))
types.validate({"a": 1, "b": "x", 2: 3}, "counts")  # b's value, and the key 2: each at its own path
```

A key is checked first, then its value, both at the key's path. A key that is a boolean shows in a path as
text (`"True"`), because YAML reads `yes` and `on` as `true`. A `keys` or `values` that names no type gives one
`unknown_type` at the mapping, before any key is read.

A `pattern` is a Python regular expression and the whole value must match it. Nothing is coerced:
`"30"` and `true` are not integers.
