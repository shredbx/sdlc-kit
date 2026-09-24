"""A root folder, and the paths inside it."""

import os
import shutil
from dataclasses import dataclass
from pathlib import Path, PurePath


@dataclass(frozen=True)
class Folder:
    """A root folder that refuses any path outside it. `root` is absolute and need not exist: a write makes it.
    A path is checked as text, `..` folded and no symlink followed, so a link inside the root that leads out is not seen."""

    root: Path

    def __post_init__(self) -> None:
        if not isinstance(self.root, (str, PurePath)):
            raise TypeError(f"root must be a path, got {self.root!r}")
        if not os.path.isabs(self.root):
            raise ValueError(f"root must be an absolute path, got {str(self.root)!r}")
        object.__setattr__(self, "root", Path(os.path.normpath(self.root)))

    def path(self, relative: str | PurePath) -> Path:
        """The absolute path of `relative`. A wrong call raises: `TypeError` for a path that is not text or a
        `PurePath`, `ValueError` for one that is absolute or ends outside the root."""
        if not isinstance(relative, (str, PurePath)):
            raise TypeError(f"a path must be text or a PurePath, got {relative!r}")
        target = Path(os.path.normpath(self.root / relative))
        if os.path.isabs(relative) or not target.is_relative_to(self.root):
            raise ValueError(f"{str(relative)!r} is not inside {self.root}")
        return target

    def inside(self, relative: str | PurePath) -> bool:
        """Whether `path(relative)` would work, for a caller that returns an error instead of raising."""
        try:
            self.path(relative)
        except ValueError:
            return False
        return True

    def exists(self, relative: str | PurePath) -> bool:
        """Whether a file or a folder is there."""
        return self.path(relative).exists()

    def read(self, relative: str | PurePath) -> str | None:
        """The text of the file, read as UTF-8 with no newline translation, or None when there is no file there."""
        file = self.path(relative)
        return file.read_bytes().decode("utf-8") if file.is_file() else None

    def write(self, relative: str | PurePath, content: str | bytes) -> None:
        """Write `content`, text as UTF-8, making every folder above the file and replacing a file that is there."""
        if not isinstance(content, (str, bytes)):
            raise TypeError(f"content must be text or bytes, got {content!r}")
        file = self.path(relative)
        file.parent.mkdir(parents=True, exist_ok=True)
        file.write_bytes(content.encode("utf-8") if isinstance(content, str) else content)

    def remove(self, relative: str | PurePath) -> None:
        """Delete the file or folder at `relative`, and everything under it. Does nothing when there is nothing
        there. A wrong call raises exactly as `path` does."""
        target = self.path(relative)
        if target.is_dir():
            shutil.rmtree(target)
        elif target.exists():
            target.unlink()

    def make(self) -> None:
        """Make the root folder, and every folder above it, if it is not there. A root that is a file raises `FileExistsError`."""
        self.root.mkdir(parents=True, exist_ok=True)

    def folder(self, relative: str | PurePath) -> "Folder":
        """A folder rooted at `relative`, which need not exist. It refuses paths outside itself, as this one does."""
        return Folder(self.path(relative))
