import pytest


@pytest.fixture
def types():
    """The `Types` `gather` checks against: `text`, a lower-case word; `style`, a mapping with a required string field `mark`; and,
    for a value given by `--set`, `count` (an integer, `minimum: 0`), `ratio` (a float) and `flag` (a boolean). `tags`, a sequence of
    `text`, is for a value given already typed, where a sequence has something `--set`'s own casting could never produce."""
    from process_kit.schema import Schema
    from process_kit.types import Types

    found = Types()
    for data in (
        {"name": "text", "type": "string", "pattern": "[a-z]+"},
        {"name": "style", "type": "mapping", "fields": {"mark": {"type": "string", "required": True}}},
        {"name": "count", "type": "integer", "minimum": 0},
        {"name": "ratio", "type": "float"},
        {"name": "flag", "type": "boolean"},
        {"name": "tags", "type": "sequence", "items": "text"},
    ):
        schema = Schema.from_data(data)
        assert isinstance(schema, Schema), schema
        found.add(schema.name, schema)
    return found
