# process-kit-schema

Definitions. Read a schema definition — from a file, from text, from data already parsed, or a folder of them — and use it as a type. Built on `process-kit-types`.

## Overview

A definition has a `name`, a `type` (one of the six YAML types) and the properties of that type:

```yaml
name: email
type: string
pattern: "[^@]+@[^@]+"
```

## Install

Not installed standalone: pulled in as a dependency by every package that reads a definition file
(`uses: [process-kit-schema]`). From a checkout of this repository, `uv sync --all-packages --group dev`
builds it into the workspace for its own tests.

## Usage

```python
from process_kit.schema import Schema
from process_kit.types import Types

email = Schema.load("schemas/email.yaml")  # a Schema — or a list of Error
types = Types()
types.add(email.name, email)

for error in types.validate("nope", "email"):
    print(error.path, error.code, error.message)
```

```text
() pattern must match [^@]+@[^@]+, got 'nope'
```

## What it holds

| | |
|---|---|
| `Schema` | a named type: `name` (the full name, see below), `base`, the configured YAML type it is based on, and `description` — the definition's own optional sentence, `None` for one that never wrote it. It *is* a `Type`, so a field can name it exactly the way it names `string` |
| `Schema.load(source)` | the schema in a file |
| `Schema.loads(text)` | the schema in YAML text |
| `Schema.from_data(data)` | the schema in a definition already parsed — a dict from JSON, a database, code |
| `Schema.load_all(location)` | every `*.yaml` file directly in a folder, as a `dict` of name to `Schema` |
| `qualify(name, namespace)` | the name rule, for a **reference to a type**: a bare name gets the namespace, and a dotted name or the name of a YAML type stays as written |
| `qualify_id(name, namespace)` | the name rule, for an **id**: a bare name gets the namespace, a YAML type name too, and a dotted name stays. For a package that loads a file of its own, such as an action |
| `parse(text)` | the YAML reader the loaders use, for any package that reads a YAML file: `(data, errors)`. A syntax error, or a mapping key written twice, is one `Error` with code `invalid`, whose message is one line that says what is wrong and ends with the line it is on, and `data` is `None`. Anything but text raises `TypeError` |

Each returns the schema, **or** the list of every error in it — never both, never half a schema.
The first three give the same errors for the same content. A missing file or folder raises
`FileNotFoundError`, and a key written twice is an error rather than a silent override.
`load_all` reads the files in name order, ignores names starting with a dot and other files, and
starts the path of each error with the file name.

## Names

A definition never says which package it belongs to. `name: email` and `type: email` are local,
and only another package's type is written with a dot: `type: core.tags`. The package comes in as
a parameter, `namespace`, on every way to load:

```python
from importlib.resources import files

Schema.load_all(files(__package__) / "schemas", __package__)  # notification_kit.email, notification_kit.contact …
```

With it, the schema's name becomes `notification_kit.email`, and each bare name that is not a YAML
type becomes `notification_kit.<name>` — once, when the schema is built. It is empty by default,
which changes nothing. Rename the package and nothing inside it changes. A wrong namespace — not
text, or `.a`, `a..b` — raises; note that `__package__` is `None` in a script.

`qualify` and `qualify_id` are that rule, public, so a package that reads another file with names in it
(an action, a template, a process) applies the same one. Both raise for a name that is not text, and for a
wrong namespace.

A mapping with free keys says what its keys and values are, by name, and the names are qualified the way
`items` and a field's `type` are:

```yaml
name: tags
type: mapping
keys: string
values: tag        # notif.tag when the namespace is notif; core.tag stays as written
```

A type name inside a definition is not looked up when the file is read, only when data is checked
or `types.check()` is asked, so schemas can be loaded and added in any order, and a schema can
name itself.
