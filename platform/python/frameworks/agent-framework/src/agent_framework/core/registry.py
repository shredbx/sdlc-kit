"""Discovers agents from an importable package: each `<agents_package>.<name>.config` module
becomes one `RegisteredAgent`. No build step — add a folder, restart."""

import importlib
import importlib.util
from dataclasses import replace
from pathlib import Path

from agent_framework.core.types import RegisteredAgent


def discover(agents_package: str) -> list[RegisteredAgent]:
    """Walk `agents_package` (a dotted, importable package name, e.g. `"agents"`), importing each
    subfolder's `config` module and collecting the `RegisteredAgent` it exports as
    `registered_agent`. Raises if a `config.py` exists but doesn't export one."""
    package = importlib.import_module(agents_package)
    if package.__file__ is None:
        raise ValueError(f"{agents_package!r} has no __file__ — is it a namespace package?")
    package_path = Path(package.__file__).parent

    discovered: list[RegisteredAgent] = []
    for agent_dir in sorted(p for p in package_path.iterdir() if p.is_dir() and not p.name.startswith("_")):
        if not (agent_dir / "config.py").is_file():
            continue
        module_name = f"{agents_package}.{agent_dir.name}.config"
        module = importlib.import_module(module_name)
        registered = getattr(module, "registered_agent", None)
        if not isinstance(registered, RegisteredAgent):
            raise ValueError(f"{module_name} must define `registered_agent: RegisteredAgent`")
        discovered.append(replace(registered, name=agent_dir.name))
    return discovered
