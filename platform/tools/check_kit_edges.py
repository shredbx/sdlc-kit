#!/usr/bin/env python3
"""Kit dependency check for platform/ (docs/proposals/bos-system-design.md, section 3.1).

A package's kit is its folder: platform/<lang>/packages/<kit>/... . The edges between kits come from go.mod
(direct requires of in-repo modules) and package.json (workspace: dependencies); the rules are data in
kit-edges.toml next to this file. Standard library only (Python 3.11+).

Exits 1 when
  * a shared kit depends on a feature kit and the edge is not listed under known_violation;
  * a feature kit depends on another feature kit and the edge is not listed under declared;
  * kits depend on each other in a cycle (checked per stack);
  * kit-edges.toml is stale or mis-filed: it lists an edge that no longer exists, files an edge under the
    wrong table, or names a shared kit that has no package.
"""

import json
import re
import sys
import tomllib
from pathlib import Path

PLATFORM = Path(__file__).resolve().parents[1]
CONFIG = Path(__file__).with_name("kit-edges.toml")

GO_MODULE = re.compile(r"^module\s+(\S+)", re.M)
GO_REQUIRE = re.compile(r"^\s*(?:require\s+)?(\S+)\s+v\S+(\s*//\s*indirect)?\s*$", re.M)
SVELTE_DEP_FIELDS = ("dependencies", "peerDependencies", "optionalDependencies")

Edges = dict[tuple[str, str], list[str]]  # (from kit, to kit) -> "from package -> to package" examples


def kit_of(package_dir: Path, root: Path) -> str:
    return package_dir.relative_to(root).parts[0]


def go_edges(root: Path) -> tuple[Edges, set[str]]:
    modules: dict[str, Path] = {}
    for go_mod in root.rglob("go.mod"):
        modules[GO_MODULE.search(go_mod.read_text()).group(1)] = go_mod.parent
    edges: Edges = {}
    for name, directory in modules.items():
        for match in GO_REQUIRE.finditer((directory / "go.mod").read_text()):
            dep, indirect = match.group(1), match.group(2)
            if indirect or dep == name or dep not in modules:
                continue
            a, b = kit_of(directory, root), kit_of(modules[dep], root)
            if a != b:
                example = f"{directory.relative_to(root)} -> {modules[dep].relative_to(root)}"
                edges.setdefault((a, b), []).append(example)
    return edges, {kit_of(d, root) for d in modules.values()}


def svelte_edges(root: Path) -> tuple[Edges, set[str]]:
    packages: dict[str, Path] = {}
    for package_json in root.rglob("package.json"):
        if "node_modules" not in package_json.parts:
            packages[json.loads(package_json.read_text())["name"]] = package_json.parent
    edges: Edges = {}
    for name, directory in packages.items():
        manifest = json.loads((directory / "package.json").read_text())
        for field in SVELTE_DEP_FIELDS:
            for dep, spec in manifest.get(field, {}).items():
                if dep in packages and spec.startswith("workspace:"):
                    a, b = kit_of(directory, root), kit_of(packages[dep], root)
                    if a != b:
                        edges.setdefault((a, b), []).append(f"{name} -> {dep}")
    return edges, {kit_of(d, root) for d in packages.values()}


def find_cycle(edges: Edges) -> list[str] | None:
    graph: dict[str, list[str]] = {}
    for a, b in edges:
        graph.setdefault(a, []).append(b)
    state: dict[str, int] = {}  # 1 = on the current path, 2 = done

    def visit(kit: str, path: list[str]) -> list[str] | None:
        state[kit] = 1
        for nxt in sorted(graph.get(kit, [])):
            if state.get(nxt) == 1:
                return path[path.index(nxt) :] + [nxt]
            if nxt not in state and (cycle := visit(nxt, path + [nxt])):
                return cycle
        state[kit] = 2
        return None

    for kit in sorted(graph):
        if kit not in state and (cycle := visit(kit, [kit])):
            return cycle
    return None


def pairs(entries: list[dict]) -> set[tuple[str, str]]:
    return {(e["from"], e["to"]) for e in entries}


def main() -> int:
    config = tomllib.loads(CONFIG.read_text())
    shared = set(config["shared"])
    declared = pairs(config.get("declared", []))
    known = pairs(config.get("known_violation", []))

    go, go_kits = go_edges(PLATFORM / "go" / "packages")
    svelte, svelte_kits = svelte_edges(PLATFORM / "svelte" / "packages")
    errors: list[str] = []
    seen: set[tuple[str, str]] = set()

    for stack, edges in (("Go", go), ("Svelte", svelte)):
        print(f"{stack}: {len(edges)} kit edges")
        for (a, b), examples in sorted(edges.items()):
            seen.add((a, b))
            if a in shared and b not in shared:
                label = "KNOWN VIOLATION shared -> feature" if (a, b) in known else "shared -> feature"
                if (a, b) not in known:
                    errors.append(f"{stack}: shared kit '{a}' depends on feature kit '{b}' ({examples[0]})")
            elif a not in shared and b not in shared:
                label = "declared feature -> feature" if (a, b) in declared else "feature -> feature"
                if (a, b) not in declared:
                    errors.append(f"{stack}: undeclared feature edge '{a}' -> '{b}' ({examples[0]})")
            else:
                label = "ok"
            print(f"  {a:15} -> {b:15} {label:36} {examples[0]}")
        if cycle := find_cycle(edges):
            errors.append(f"{stack}: kits depend on each other in a cycle: {' -> '.join(cycle)}")

    for a, b in sorted(declared | known):
        if (a, b) not in seen:
            errors.append(f"kit-edges.toml is stale: '{a}' -> '{b}' is listed but no package depends that way")
    for a, b in sorted(declared):
        if a in shared or b in shared:
            errors.append(f"kit-edges.toml: declared edge '{a}' -> '{b}' involves a shared kit; declared is feature -> feature")
    for a, b in sorted(known):
        if not (a in shared and b not in shared):
            errors.append(f"kit-edges.toml: known_violation '{a}' -> '{b}' is not shared -> feature")
    for kit in sorted(shared - go_kits - svelte_kits):
        errors.append(f"kit-edges.toml: shared kit '{kit}' has no package in either stack")

    print(f"declared {len(declared)} | known violations {len(known)} | errors {len(errors)}")
    for message in errors:
        print(f"ERROR {message}")
    return 1 if errors else 0


if __name__ == "__main__":
    sys.exit(main())
