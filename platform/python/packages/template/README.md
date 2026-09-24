# process-kit-template

Templates. Read a template from its folder and check it, make its files from data, and check that files still conform to it. Built on `process-kit-types`, `process-kit-schema` and Jinja2.

## Overview

A template is a folder. `template.yaml` names it, says what type of data it is rendered with, and says
what `conform` will require; `files/` holds the files it makes:

```text
definitions/process-os/template/package/
├── template.yaml
└── files/
    ├── pyproject.toml.jinja           a .jinja file is rendered, and loses the ending
    ├── src/{{source}}/__init__.py.jinja   file and folder names are rendered too
    └── fixtures/.gitkeep              any other file is copied as it is
```

```yaml
# template.yaml
name: package                          # the folder's name
input: package-spec                    # the type of the data
pattern:                               # what conform requires: for each file, the lines it must hold
  pyproject.toml:
    - 'name = "{{ dist }}"'
  "src/{{source}}/__init__.py": []     # an empty list only requires the file
```

## Install

Not installed standalone: pulled in as a dependency by `process-framework`
(`uses: [process-kit-template]`). From a checkout of this repository, `uv sync --all-packages --group dev`
builds it into the workspace for its own tests.

## Usage

```python
from process_kit.template import Template

template = Template.load("definitions/process-os/template/package", "process-os")  # a Template, or a list of Error
template.id  # process-os.package
template.input  # process-os.package-spec

files = template.render(data, types)  # {"pyproject.toml": "...", "fixtures/.gitkeep": b""}, or a list of Error. Nothing is written
errors = template.conform(data, folder, types)  # [] when every file still holds what the pattern requires
```

## What it holds

| | |
|---|---|
| `Template` | a frozen dataclass: `id` (in full), `input` (the full name of the type of the data), `pattern` (path text → a tuple of line texts, read-only, in the order written), and `home`, the folder |
| `Template.load(folder, namespace="")` | the template in a folder, or every error in it, never both |
| `Template.references()` | `[(("input",), <the input's full name>)]`, for a caller that loads the type of the data |
| `Template.render(data, types)` | every file the template makes, `{path: text or bytes}` in path order, or every error. It writes nothing. The data is checked against `input` first. A `.jinja` file is rendered and loses the ending, so it is text; any other file is copied, so it is bytes. A name is rendered too |
| `Template.conform(data, source, types)` | the errors in files that should still hold what the pattern requires: `missing` for a file, `missing_line` for a line. The data is checked first |
| `Source` | what `conform` reads through: `read(path)` gives the text, or `None`. A `Folder` of `filesystem` is one without knowing it |
| `register(types)` | adds the schemas of `template.yaml` to a `Types`, as `process_kit.template.template`, `.name`, `.type-name`, `.pattern`, `.path-template` and `.lines` |

## Rules

- **A broken template is found when it loads.** After `template.yaml` is checked against its schema, the `name` must be the folder's
  name, there must be a `files/` folder, every file name under it must be valid Jinja, so must every `.jinja` file, and so must every key
  and line of the `pattern`. Any other file is not read. A folder named `__pycache__` and a file named `.DS_Store` are skipped; a hidden file such as `.gitkeep` is part of the template.
- **The engine is strict.** A name the data lacks is an error, not an empty string, and a file keeps the line feed it ends with.
- **Wrong data is returned; a wrong call raises.** Errors are `Error`s, each path starting with the file. A folder that is not there, that
  has no `template.yaml`, or that is a file raises, and so does a wrong namespace.
- **A render is all or nothing.** Every name and every file is rendered before any is returned, and a problem in any is an error: a name the data
  lacks (`undefined`), a path that is empty, absolute or has a `..` step (`invalid`), two files that make one path (`duplicate`). The caller writes
  only what it is given, so a bad render leaves nothing behind.
- **conform is a line-by-line check.** Each rendered line must be found in a line of its file: the file may hold more, and a line may hold more. So a file
  that others have extended still conforms. An empty list only requires the file.
- **Line ends.** Jinja turns every line end of a `.jinja` file into a line feed, and keeps the one at its end. Other files are copied byte for byte.
- **Nothing is written.** The template never writes a file. It reads its own folder, and `conform` reads only through the `Source` it is handed.
