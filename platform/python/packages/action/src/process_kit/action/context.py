"""All an action is handed besides its inputs."""

from collections.abc import Mapping
from dataclasses import dataclass, field
from pathlib import Path


@dataclass(frozen=True)
class Context:
    """The folder an action works in, and the variables its caller adds to the environment the scripts see."""

    output: Path
    environment: Mapping[str, str] = field(default_factory=dict)
