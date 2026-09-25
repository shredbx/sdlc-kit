from pathlib import Path

import pytest

REPO = Path(__file__).parents[7]  # sbx-sdlc-kit's root: upstream had this at [5], one layout shallower


@pytest.fixture
def types():
    """A small, synthetic `Types`: `word` (a lower-case string), `count` (a non-negative integer),
    `ratio` (a float), `flag` (a boolean), `words` (a sequence of `word`), `tag` (a mapping of `word`
    to `word`), `thing` (a mapping with one required field, `a`, and one optional, `b`), and `loop`
    (a mapping whose own field names itself)."""
    from process_kit.schema import Schema
    from process_kit.types import Types

    found = Types()
    for data in (
        {"name": "word", "type": "string", "pattern": "[a-z]+"},
        {"name": "count", "type": "integer", "minimum": 0},
        {"name": "ratio", "type": "float"},
        {"name": "flag", "type": "boolean"},
        {"name": "words", "type": "sequence", "items": "word"},
        {"name": "tag", "type": "mapping", "keys": "word", "values": "word"},
        {"name": "thing", "type": "mapping", "fields": {"a": {"type": "word", "required": True}, "b": {"type": "count"}}},
        {"name": "loop", "type": "mapping", "fields": {"next": {"type": "loop"}}},
    ):
        schema = Schema.from_data(data)
        assert isinstance(schema, Schema), schema
        found.add(schema.name, schema)
    return found


@pytest.fixture
def catalog():
    """This repository's `libraries/` folder (a verbatim copy of process-os's `definitions/`), opened —
    `sdlc.resolve-platform`'s own scope and everything it uses, exactly as `process-cli run sdlc.resolve-platform`
    reads it today."""
    from process_framework import Catalog

    found = Catalog.open(REPO / "processos-workspace" / "libraries")
    assert not isinstance(found, list), found
    return found
