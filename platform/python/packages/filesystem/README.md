# process-kit-filesystem

Filesystem. A root folder that refuses any path outside it, and writes files inside it. It knows nothing about actions, templates or runs: it is a folder and the paths in it. Built on the standard library alone.

## Install

Not installed standalone: pulled in as a dependency by every package that writes files
(`uses: [process-kit-filesystem]`). From a checkout of this repository, `uv sync --all-packages --group dev`
builds it into the workspace for its own tests.

## Usage

```python
from pathlib import Path
from process_kit.filesystem import Folder

output = Folder(Path("/work/output"))  # it need not exist yet
output.write("apps/hello/README.md", "# hello")  # makes apps/hello/, and the root if it is missing
output.read("apps/hello/README.md")  # "# hello"
output.read("nothing.txt")  # None
output.inside("../elsewhere")  # False: ask first, and return an error
output.write("../elsewhere", "x")  # raises ValueError: a wrong call
```

## What it holds

| | |
|---|---|
| `Folder(root)` | a frozen dataclass holding `root`, an absolute path with every `..` folded. It need not exist: a write makes it. A `root` that is not a path raises `TypeError`, and a relative one `ValueError` |
| `path(relative)` | the absolute path of `relative`, or a wrong call: `ValueError` if it is absolute or ends outside the root, `TypeError` if it is not text or a `PurePath` |
| `inside(relative)` | whether `path(relative)` would work. It is how a caller returns an error instead of raising |
| `exists(relative)` | whether a file or a folder is there |
| `read(relative)` | the text of the file, as UTF-8 with no newline translation, or `None` when there is no file there |
| `write(relative, content)` | writes text (UTF-8) or bytes, makes every folder above the file, and replaces a file that is there |
| `make()` | makes the root folder, and every folder above it, if it is not there. A root that is a file raises `FileExistsError` |
| `folder(relative)` | a `Folder` rooted at `relative`, which refuses paths outside itself as this one does |
| `remove(relative)` | deletes the file or folder at `relative`, and everything under it; does nothing when there is nothing there. A wrong call raises exactly as `path` does |

## Rules

- **The root is a wall.** A path is read from the root. An absolute path, and one that folds outside
  the root (`a/../..`), is outside. The root itself is inside, and `a/../b` is `b`. A folder that
  only starts with the root's name (`../work-other`) is outside too.
- **A wrong call raises; nothing looks for anything.** A path outside the root raises `ValueError`,
  and the caller that would rather return an error asks `inside` first. The framework does that
  before it writes anything.
- **Paths are checked as text.** `..` is folded and no symlink is followed, as `Config` does, so a
  link inside the root that leads out is not seen.
- **Still no list, move or copy, and no read of bytes.** `remove` is the one exception — the runtime's
  own run-history pruning uses it to delete a `done` run once it is past `runtime.history`; nothing
  else here needs the rest yet.
