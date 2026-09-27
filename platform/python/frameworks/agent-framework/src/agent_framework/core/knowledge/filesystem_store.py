"""Loads KnowledgeEntry objects from one YAML file per entry under a root folder — add a file,
restart, no code change. The default, local-dev adapter; a postgres_store implements the same
interface for real deployments."""

from pathlib import Path

import yaml

from agent_framework.core.knowledge.base import KnowledgeEntry, KnowledgeStore, KnowledgeVariant


class FilesystemKnowledgeStore(KnowledgeStore):
    def __init__(self, root: Path) -> None:
        self._entries = tuple(self._load(path) for path in sorted(root.rglob("*.yaml")))

    def all_entries(self) -> tuple[KnowledgeEntry, ...]:
        return self._entries

    @staticmethod
    def _load(path: Path) -> KnowledgeEntry:
        data = yaml.safe_load(path.read_text(encoding="utf-8"))
        return KnowledgeEntry(
            id=data["id"],
            category=data["category"],
            canonical_question=data["canonical_question"],
            variations=tuple(data.get("variations", [])),
            tags=tuple(data.get("tags", [])),
            variants=tuple(KnowledgeVariant(when=v["when"], answer_template=v["answer_template"], tone_note=v.get("tone_note")) for v in data["variants"]),
        )
