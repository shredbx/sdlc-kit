from process_kit.schema import Schema
from process_kit.types import Types


def test_a_mapping_with_named_fields_loads_and_validates_through_them(case):
    schema = Schema.loads(case["text"], case["namespace"])
    errors = schema.validate(case["value"], (), Types())
    assert [(error.path, error.code) for error in errors] == [(tuple(one["path"]), one["code"]) for one in case["expect"]]
    assert all(error.message for error in errors)
